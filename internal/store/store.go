package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"lore/internal/models"
)

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// SaveConversation stores a fully parsed conversation (with topics and messages) in a single transaction.
func (s *Store) SaveConversation(parsed *models.ParsedConversation, sourcePath string) (*models.Conversation, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Determine title from first topic label or first user message
	title := "Untitled Conversation"
	if len(parsed.Topics) > 0 {
		title = parsed.Topics[0].Label
	}

	// Determine time range
	var earliest, latest time.Time
	for _, t := range parsed.Turns {
		if !t.Timestamp.IsZero() {
			if earliest.IsZero() || t.Timestamp.Before(earliest) {
				earliest = t.Timestamp
			}
			if latest.IsZero() || t.Timestamp.After(latest) {
				latest = t.Timestamp
			}
		}
	}
	if earliest.IsZero() {
		earliest = time.Now()
	}
	if latest.IsZero() {
		latest = earliest
	}

	// Count non-system messages
	msgCount := 0
	for _, t := range parsed.Turns {
		if t.Role != "system" {
			msgCount++
		}
	}

	// Upsert conversation
	if _, err := tx.Exec(
		`INSERT INTO conversations (session_id, title, source_path, created_at, updated_at, message_count, topic_count)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(session_id) DO UPDATE SET
		   title = excluded.title,
		   source_path = excluded.source_path,
		   updated_at = excluded.updated_at,
		   message_count = excluded.message_count,
		   topic_count = excluded.topic_count`,
		parsed.SessionID, title, sourcePath, earliest, latest, msgCount, len(parsed.Topics),
	); err != nil {
		return nil, fmt.Errorf("insert conversation: %w", err)
	}

	// LastInsertId is unreliable when ON CONFLICT triggers UPDATE — look up by session_id.
	var convoID int64
	if err := tx.QueryRow("SELECT id FROM conversations WHERE session_id = ?", parsed.SessionID).Scan(&convoID); err != nil {
		return nil, fmt.Errorf("get conversation id: %w", err)
	}

	// Clear any prior topics/messages so re-upload replaces rather than appends.
	// Messages first: they reference topics via topic_id.
	if _, err := tx.Exec("DELETE FROM messages WHERE conversation_id = ?", convoID); err != nil {
		return nil, fmt.Errorf("clear messages: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM topics WHERE conversation_id = ?", convoID); err != nil {
		return nil, fmt.Errorf("clear topics: %w", err)
	}

	// Insert topics
	topicIDs := make(map[int]int64) // seq -> db ID
	for _, topic := range parsed.Topics {
		res, err := tx.Exec(
			`INSERT INTO topics (conversation_id, seq, label, started_at, message_count)
			 VALUES (?, ?, ?, ?, ?)`,
			convoID, topic.Seq, topic.Label, topic.StartedAt, topic.MsgCount,
		)
		if err != nil {
			return nil, fmt.Errorf("insert topic %d: %w", topic.Seq, err)
		}
		topicID, _ := res.LastInsertId()
		topicIDs[topic.Seq] = topicID
	}

	// Insert messages
	for i, turn := range parsed.Turns {
		var toolsJSON, resultsJSON *string

		if len(turn.Tools) > 0 {
			b, _ := json.Marshal(turn.Tools)
			s := string(b)
			toolsJSON = &s
		}
		if len(turn.ToolResults) > 0 {
			b, _ := json.Marshal(turn.ToolResults)
			s := string(b)
			resultsJSON = &s
		}

		var topicID *int64
		if tid, ok := topicIDs[turn.TopicSeq]; ok {
			topicID = &tid
		}

		_, err := tx.Exec(
			`INSERT INTO messages (conversation_id, topic_id, seq, role, content, tools_json, tool_results_json, timestamp)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			convoID, topicID, i, turn.Role, turn.Content, toolsJSON, resultsJSON, turn.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("insert message %d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &models.Conversation{
		ID:           convoID,
		SessionID:    parsed.SessionID,
		Title:        title,
		SourcePath:   sourcePath,
		CreatedAt:    earliest,
		UpdatedAt:    latest,
		MessageCount: msgCount,
		TopicCount:   len(parsed.Topics),
	}, nil
}

func (s *Store) ListConversations() ([]models.Conversation, error) {
	rows, err := s.db.Query(
		`SELECT id, session_id, title, source_path, created_at, updated_at, message_count, topic_count
		 FROM conversations ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convos []models.Conversation
	for rows.Next() {
		var c models.Conversation
		if err := rows.Scan(&c.ID, &c.SessionID, &c.Title, &c.SourcePath, &c.CreatedAt, &c.UpdatedAt, &c.MessageCount, &c.TopicCount); err != nil {
			return nil, err
		}
		convos = append(convos, c)
	}
	return convos, rows.Err()
}

func (s *Store) GetConversation(id int64) (*models.Conversation, error) {
	c := &models.Conversation{}
	err := s.db.QueryRow(
		`SELECT id, session_id, title, source_path, created_at, updated_at, message_count, topic_count
		 FROM conversations WHERE id = ?`, id,
	).Scan(&c.ID, &c.SessionID, &c.Title, &c.SourcePath, &c.CreatedAt, &c.UpdatedAt, &c.MessageCount, &c.TopicCount)
	if err != nil {
		return nil, err
	}

	// Load topics
	rows, err := s.db.Query(
		`SELECT id, conversation_id, seq, label, started_at, message_count
		 FROM topics WHERE conversation_id = ? ORDER BY seq`, id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t models.Topic
		if err := rows.Scan(&t.ID, &t.ConversationID, &t.Seq, &t.Label, &t.StartedAt, &t.MessageCount); err != nil {
			return nil, err
		}
		c.Topics = append(c.Topics, t)
	}

	return c, rows.Err()
}

func (s *Store) GetMessages(conversationID int64, topicID *int64) ([]models.Message, error) {
	var rows *sql.Rows
	var err error

	if topicID != nil {
		rows, err = s.db.Query(
			`SELECT id, conversation_id, topic_id, seq, role, content, tools_json, tool_results_json, timestamp
			 FROM messages WHERE conversation_id = ? AND topic_id = ? ORDER BY seq`,
			conversationID, *topicID,
		)
	} else {
		rows, err = s.db.Query(
			`SELECT id, conversation_id, topic_id, seq, role, content, tools_json, tool_results_json, timestamp
			 FROM messages WHERE conversation_id = ? ORDER BY seq`,
			conversationID,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var m models.Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.TopicID, &m.Seq, &m.Role, &m.Content, &m.ToolsJSON, &m.ResultsJSON, &m.Timestamp); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func (s *Store) DeleteConversation(id int64) error {
	res, err := s.db.Exec("DELETE FROM conversations WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("conversation %d not found", id)
	}
	return nil
}

// ReSegmentTopics replaces the topics for a conversation with new semantic segments.
// Must null out topic_id on messages first since topic_id ON DELETE CASCADE would delete messages.
func (s *Store) ReSegmentTopics(conversationID int64, segments []models.TopicSegment) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Detach messages from old topics (must come before deleting topics)
	if _, err := tx.Exec(`UPDATE messages SET topic_id = NULL WHERE conversation_id = ?`, conversationID); err != nil {
		return fmt.Errorf("null topic_ids: %w", err)
	}

	// Delete old topics
	if _, err := tx.Exec(`DELETE FROM topics WHERE conversation_id = ?`, conversationID); err != nil {
		return fmt.Errorf("delete old topics: %w", err)
	}

	// Insert new topics and reassign messages
	for i, seg := range segments {
		msgCount := seg.EndSeq - seg.StartSeq + 1

		res, err := tx.Exec(
			`INSERT INTO topics (conversation_id, seq, label, started_at, message_count) VALUES (?, ?, ?, ?, ?)`,
			conversationID, i, seg.Label, seg.StartedAt, msgCount,
		)
		if err != nil {
			return fmt.Errorf("insert topic %d: %w", i, err)
		}

		topicID, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("get topic id: %w", err)
		}

		if _, err := tx.Exec(
			`UPDATE messages SET topic_id = ? WHERE conversation_id = ? AND seq >= ? AND seq <= ?`,
			topicID, conversationID, seg.StartSeq, seg.EndSeq,
		); err != nil {
			return fmt.Errorf("assign messages to topic %d: %w", i, err)
		}
	}

	// Update topic count
	if _, err := tx.Exec(
		`UPDATE conversations SET topic_count = ? WHERE id = ?`,
		len(segments), conversationID,
	); err != nil {
		return fmt.Errorf("update topic count: %w", err)
	}

	return tx.Commit()
}

func (s *Store) SearchTopics(query string) ([]models.Topic, error) {
	rows, err := s.db.Query(
		`SELECT id, conversation_id, seq, label, started_at, message_count
		 FROM topics WHERE label LIKE ? ORDER BY started_at DESC`,
		"%"+query+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topics []models.Topic
	for rows.Next() {
		var t models.Topic
		if err := rows.Scan(&t.ID, &t.ConversationID, &t.Seq, &t.Label, &t.StartedAt, &t.MessageCount); err != nil {
			return nil, err
		}
		topics = append(topics, t)
	}
	return topics, rows.Err()
}
