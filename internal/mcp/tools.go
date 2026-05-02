package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"lore/internal/parser"
)

func (s *Server) toolsList() toolsListResult {
	return toolsListResult{
		Tools: []toolDef{
			{
				Name:        "upload_conversation",
				Description: "Upload a Claude Code conversation from a .jsonl session file. Provide either a session_id (UUID) to auto-locate the file, or a full path to the .jsonl file.",
				InputSchema: inputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"session_id": map[string]interface{}{
							"type":        "string",
							"description": "Session UUID (e.g. a77a49ed-5860-46fe-a842-be3b161ae6a1). The tool will search ~/.claude/ for the matching .jsonl file.",
						},
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Full path to a .jsonl session file.",
						},
					},
				},
			},
			{
				Name:        "list_conversations",
				Description: "List all uploaded conversations with their titles, dates, message counts, and topic counts.",
				InputSchema: inputSchema{
					Type:       "object",
					Properties: map[string]interface{}{},
				},
			},
			{
				Name:        "delete_conversation",
				Description: "Delete an uploaded conversation by its numeric ID.",
				InputSchema: inputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "number",
							"description": "The conversation ID to delete.",
						},
					},
				},
			},
			{
				Name:        "list_documents",
				Description: "List all generated knowledge base documents with their titles and linked conversation IDs.",
				InputSchema: inputSchema{
					Type:       "object",
					Properties: map[string]interface{}{},
				},
			},
			{
				Name:        "get_document",
				Description: "Get the full content of a knowledge base document by its numeric ID.",
				InputSchema: inputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "number",
							"description": "The document ID.",
						},
					},
				},
			},
			{
				Name:        "search_documents",
				Description: "Search knowledge base documents by keyword. Returns documents matching the query in title or content.",
				InputSchema: inputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "The search query.",
						},
						"limit": map[string]interface{}{
							"type":        "number",
							"description": "Maximum number of results (default 5).",
						},
					},
				},
			},
			{
				Name:        "delete_document",
				Description: "Delete a knowledge base document by its numeric ID.",
				InputSchema: inputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "number",
							"description": "The document ID to delete.",
						},
					},
				},
			},
		},
	}
}

func (s *Server) callTool(params toolCallParams) (*toolCallResult, error) {
	switch params.Name {
	case "upload_conversation":
		return s.toolUpload(params.Arguments)
	case "list_conversations":
		return s.toolList()
	case "delete_conversation":
		return s.toolDelete(params.Arguments)
	case "list_documents":
		return s.toolListDocs()
	case "get_document":
		return s.toolGetDoc(params.Arguments)
	case "search_documents":
		return s.toolSearchDocs(params.Arguments)
	case "delete_document":
		return s.toolDeleteDoc(params.Arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", params.Name)
	}
}

func (s *Server) toolUpload(args json.RawMessage) (*toolCallResult, error) {
	var input struct {
		SessionID string `json:"session_id"`
		Path      string `json:"path"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	path := input.Path
	if path == "" && input.SessionID != "" {
		path = findSessionFile(input.SessionID)
		if path == "" {
			return nil, fmt.Errorf("could not find JSONL file for session %s", input.SessionID)
		}
	}
	if path == "" {
		return nil, fmt.Errorf("provide either session_id or path")
	}

	parsed, err := parser.ParseConversation(path)
	if err != nil {
		return nil, fmt.Errorf("parsing: %w", err)
	}

	convo, err := s.store.SaveConversation(parsed, path)
	if err != nil {
		return nil, fmt.Errorf("saving: %w", err)
	}

	// Trigger doc generation (non-fatal)
	if s.pipeline != nil {
		if err := s.pipeline.ProcessConversation(context.Background(), convo); err != nil {
			log.Printf("docgen: %v", err)
		}
	}

	text := fmt.Sprintf("Uploaded conversation successfully!\n\n"+
		"  ID:       %d\n"+
		"  Session:  %s\n"+
		"  Title:    %s\n"+
		"  Messages: %d\n"+
		"  Topics:   %d\n"+
		"  Period:   %s to %s",
		convo.ID, convo.SessionID, convo.Title,
		convo.MessageCount, convo.TopicCount,
		convo.CreatedAt.Format("2006-01-02 15:04"),
		convo.UpdatedAt.Format("2006-01-02 15:04"),
	)

	return &toolCallResult{
		Content: []contentBlock{{Type: "text", Text: text}},
	}, nil
}

func (s *Server) toolList() (*toolCallResult, error) {
	convos, err := s.store.ListConversations()
	if err != nil {
		return nil, err
	}

	if len(convos) == 0 {
		return &toolCallResult{
			Content: []contentBlock{{Type: "text", Text: "No conversations uploaded yet."}},
		}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d conversation(s):\n\n", len(convos)))

	for _, c := range convos {
		sb.WriteString(fmt.Sprintf("  [%d] %s\n", c.ID, c.Title))
		sb.WriteString(fmt.Sprintf("      Session: %s\n", c.SessionID))
		sb.WriteString(fmt.Sprintf("      %d messages, %d topics  |  %s\n\n",
			c.MessageCount, c.TopicCount, c.CreatedAt.Format("2006-01-02")))
	}

	return &toolCallResult{
		Content: []contentBlock{{Type: "text", Text: sb.String()}},
	}, nil
}

func (s *Server) toolDelete(args json.RawMessage) (*toolCallResult, error) {
	var input struct {
		ID float64 `json:"id"` // JSON numbers are float64
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if err := s.store.DeleteConversation(int64(input.ID)); err != nil {
		return nil, err
	}

	return &toolCallResult{
		Content: []contentBlock{{Type: "text", Text: fmt.Sprintf("Deleted conversation %d.", int64(input.ID))}},
	}, nil
}

func (s *Server) toolListDocs() (*toolCallResult, error) {
	docs, err := s.store.ListDocuments()
	if err != nil {
		return nil, err
	}

	if len(docs) == 0 {
		return &toolCallResult{
			Content: []contentBlock{{Type: "text", Text: "No documents generated yet."}},
		}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d document(s):\n\n", len(docs)))
	for _, d := range docs {
		sb.WriteString(fmt.Sprintf("  [%d] %s\n", d.ID, d.Title))
		sb.WriteString(fmt.Sprintf("      Conversations: %v\n", d.ConversationIDs))
		sb.WriteString(fmt.Sprintf("      Updated: %s\n\n", d.UpdatedAt.Format("2006-01-02 15:04")))
	}

	return &toolCallResult{
		Content: []contentBlock{{Type: "text", Text: sb.String()}},
	}, nil
}

func (s *Server) toolGetDoc(args json.RawMessage) (*toolCallResult, error) {
	var input struct {
		ID float64 `json:"id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	doc, err := s.store.GetDocument(int64(input.ID))
	if err != nil {
		return nil, fmt.Errorf("document not found: %w", err)
	}

	text := fmt.Sprintf("# %s\n\n%s\n\n---\nDocument ID: %d | Conversations: %v | Updated: %s",
		doc.Title, doc.Content, doc.ID, doc.ConversationIDs,
		doc.UpdatedAt.Format("2006-01-02 15:04"))

	return &toolCallResult{
		Content: []contentBlock{{Type: "text", Text: text}},
	}, nil
}

func (s *Server) toolSearchDocs(args json.RawMessage) (*toolCallResult, error) {
	var input struct {
		Query string  `json:"query"`
		Limit float64 `json:"limit"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if input.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	limit := int(input.Limit)
	if limit <= 0 {
		limit = 5
	}

	docs, err := s.store.SearchDocuments(input.Query, limit)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	if len(docs) == 0 {
		return &toolCallResult{
			Content: []contentBlock{{Type: "text", Text: "No matching documents found."}},
		}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d matching document(s):\n\n", len(docs)))
	for _, doc := range docs {
		sb.WriteString(fmt.Sprintf("  [%d] %s\n", doc.ID, doc.Title))
		sb.WriteString(fmt.Sprintf("      Updated: %s\n\n", doc.UpdatedAt.Format("2006-01-02 15:04")))
	}

	return &toolCallResult{
		Content: []contentBlock{{Type: "text", Text: sb.String()}},
	}, nil
}

func (s *Server) toolDeleteDoc(args json.RawMessage) (*toolCallResult, error) {
	var input struct {
		ID float64 `json:"id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if err := s.store.DeleteDocument(int64(input.ID)); err != nil {
		return nil, err
	}

	return &toolCallResult{
		Content: []contentBlock{{Type: "text", Text: fmt.Sprintf("Deleted document %d.", int64(input.ID))}},
	}, nil
}

// findSessionFile searches common Claude Code session file locations.
func findSessionFile(sessionID string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	filename := sessionID + ".jsonl"

	// Search in Claude projects directories
	projectsDir := filepath.Join(home, ".claude", "projects")
	var found string

	filepath.Walk(projectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.Name() == filename {
			found = path
			return filepath.SkipAll
		}
		return nil
	})

	if found != "" {
		return found
	}

	// Check ~/.claude/sessions/
	sessionsPath := filepath.Join(home, ".claude", "sessions", filename)
	if _, err := os.Stat(sessionsPath); err == nil {
		return sessionsPath
	}

	return ""
}
