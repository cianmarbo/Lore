package store

import (
	"database/sql"
	"fmt"
	"time"

	"lore/internal/models"
)

func (s *Store) CreateDocument(title, content string, conversationID int64) (*models.Document, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()

	res, err := tx.Exec(
		`INSERT INTO documents (title, content, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		title, content, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert document: %w", err)
	}

	docID, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get document id: %w", err)
	}

	if _, err := tx.Exec(
		`INSERT INTO document_conversations (document_id, conversation_id, contributed_at) VALUES (?, ?, ?)`,
		docID, conversationID, now,
	); err != nil {
		return nil, fmt.Errorf("insert document_conversation: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &models.Document{
		ID:              docID,
		Title:           title,
		Content:         content,
		CreatedAt:       now,
		UpdatedAt:       now,
		ConversationIDs: []int64{conversationID},
	}, nil
}

func (s *Store) UpdateDocument(docID int64, title, content string, conversationID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()

	if _, err := tx.Exec(
		`UPDATE documents SET title = ?, content = ?, updated_at = ? WHERE id = ?`,
		title, content, now, docID,
	); err != nil {
		return fmt.Errorf("update document: %w", err)
	}

	if _, err := tx.Exec(
		`INSERT OR IGNORE INTO document_conversations (document_id, conversation_id, contributed_at) VALUES (?, ?, ?)`,
		docID, conversationID, now,
	); err != nil {
		return fmt.Errorf("insert document_conversation: %w", err)
	}

	return tx.Commit()
}

func (s *Store) UpdateDocumentContent(docID int64, title, content string) error {
	now := time.Now()
	_, err := s.db.Exec(
		`UPDATE documents SET title = ?, content = ?, updated_at = ? WHERE id = ?`,
		title, content, now, docID,
	)
	if err != nil {
		return fmt.Errorf("update document: %w", err)
	}
	return nil
}

func (s *Store) GetDocument(id int64) (*models.Document, error) {
	d := &models.Document{}
	err := s.db.QueryRow(
		`SELECT id, title, content, created_at, updated_at FROM documents WHERE id = ?`, id,
	).Scan(&d.ID, &d.Title, &d.Content, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(
		`SELECT conversation_id FROM document_conversations WHERE document_id = ? ORDER BY contributed_at`, id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int64
		if err := rows.Scan(&cid); err != nil {
			return nil, err
		}
		d.ConversationIDs = append(d.ConversationIDs, cid)
	}

	return d, rows.Err()
}

func (s *Store) ListDocuments() ([]models.Document, error) {
	rows, err := s.db.Query(
		`SELECT id, title, '', created_at, updated_at FROM documents ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []models.Document
	for rows.Next() {
		var d models.Document
		if err := rows.Scan(&d.ID, &d.Title, &d.Content, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range docs {
		crows, err := s.db.Query(
			`SELECT conversation_id FROM document_conversations WHERE document_id = ? ORDER BY contributed_at`,
			docs[i].ID,
		)
		if err != nil {
			return nil, err
		}
		for crows.Next() {
			var cid int64
			if err := crows.Scan(&cid); err != nil {
				crows.Close()
				return nil, err
			}
			docs[i].ConversationIDs = append(docs[i].ConversationIDs, cid)
		}
		crows.Close()
	}

	return docs, nil
}

func (s *Store) DeleteDocument(id int64) error {
	res, err := s.db.Exec(`DELETE FROM documents WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("document %d not found", id)
	}
	return nil
}

func (s *Store) SearchDocuments(query string, limit int) ([]models.Document, error) {
	if limit <= 0 {
		limit = 10
	}
	pattern := "%" + query + "%"
	rows, err := s.db.Query(
		`SELECT id, title, content, created_at, updated_at
		 FROM documents
		 WHERE title LIKE ? OR content LIKE ?
		 ORDER BY updated_at DESC
		 LIMIT ?`,
		pattern, pattern, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []models.Document
	for rows.Next() {
		var d models.Document
		if err := rows.Scan(&d.ID, &d.Title, &d.Content, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, rows.Err()
}

func (s *Store) GetDocumentForConversation(conversationID int64) (*models.Document, error) {
	var docID int64
	err := s.db.QueryRow(
		`SELECT document_id FROM document_conversations WHERE conversation_id = ? LIMIT 1`,
		conversationID,
	).Scan(&docID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.GetDocument(docID)
}
