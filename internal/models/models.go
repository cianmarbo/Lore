package models

import "time"

type Conversation struct {
	ID           int64     `json:"id"`
	SessionID    string    `json:"session_id"`
	Title        string    `json:"title"`
	SourcePath   string    `json:"source_path"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	MessageCount int       `json:"message_count"`
	TopicCount   int       `json:"topic_count"`
	Topics       []Topic   `json:"topics,omitempty"`
}

type Topic struct {
	ID             int64     `json:"id"`
	ConversationID int64     `json:"conversation_id"`
	Seq            int       `json:"seq"`
	Label          string    `json:"label"`
	StartedAt      time.Time `json:"started_at"`
	MessageCount   int       `json:"message_count"`
}

type Message struct {
	ID             int64     `json:"id"`
	ConversationID int64     `json:"conversation_id"`
	TopicID        *int64    `json:"topic_id"`
	Seq            int       `json:"seq"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	ToolsJSON      *string   `json:"tools_json,omitempty"`
	ResultsJSON    *string   `json:"tool_results_json,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// ParsedTurn is the intermediate result from parsing before DB storage.
type ParsedTurn struct {
	Role        string
	Content     string
	Tools       []ToolSummary
	ToolResults []string
	Timestamp   time.Time
	TopicSeq    int
}

type ToolSummary struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
}

type ParsedConversation struct {
	SessionID string
	Turns     []ParsedTurn
	Topics    []ParsedTopic
}

type ParsedTopic struct {
	Seq       int
	Label     string
	StartedAt time.Time
	MsgCount  int
}

type Document struct {
	ID              int64     `json:"id"`
	Title           string    `json:"title"`
	Content         string    `json:"content"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	ConversationIDs []int64   `json:"conversation_ids,omitempty"`
}

type DocumentConversation struct {
	DocumentID     int64     `json:"document_id"`
	ConversationID int64     `json:"conversation_id"`
	ContributedAt  time.Time `json:"contributed_at"`
}

type TopicSegment struct {
	Label     string
	StartSeq  int // message seq (inclusive)
	EndSeq    int // message seq (inclusive)
	StartedAt time.Time
}
