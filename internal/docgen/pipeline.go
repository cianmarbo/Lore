package docgen

import (
	"context"
	"fmt"
	"log"
	"strings"

	"lore/internal/llm"
	"lore/internal/models"
	"lore/internal/store"
)

// Pipeline orchestrates document generation and updates when conversations are uploaded.
type Pipeline struct {
	store *store.Store
	llm   llm.Provider // nil if unavailable
}

// New creates a Pipeline. If llm is nil, ProcessConversation is a no-op.
func New(st *store.Store, llmProvider llm.Provider) *Pipeline {
	return &Pipeline{
		store: st,
		llm:   llmProvider,
	}
}

// ProcessConversation segments the conversation into topics, then processes
// each topic independently through the match→generate/update pipeline.
func (p *Pipeline) ProcessConversation(ctx context.Context, convo *models.Conversation) error {
	if p.llm == nil {
		return nil
	}

	messages, err := p.store.GetMessages(convo.ID, nil)
	if err != nil {
		return fmt.Errorf("get messages: %w", err)
	}

	if len(messages) == 0 {
		return nil
	}

	segments := p.segmentConversation(ctx, messages)
	if len(segments) == 0 {
		return nil
	}

	if err := p.store.ReSegmentTopics(convo.ID, segments); err != nil {
		log.Printf("docgen: re-segment failed: %v — using existing topics", err)
	}

	for i, seg := range segments {
		topicMessages := filterMessagesBySeq(messages, seg.StartSeq, seg.EndSeq)
		text := ExtractText(topicMessages)
		if len(strings.TrimSpace(text)) < 100 {
			log.Printf("docgen: skipping segment %d/%d — too short", i+1, len(segments))
			continue
		}

		if err := p.processTopicSegment(ctx, convo.ID, text); err != nil {
			log.Printf("docgen: segment %d/%d error: %v", i+1, len(segments), err)
		}
	}

	return nil
}

func (p *Pipeline) segmentConversation(ctx context.Context, messages []models.Message) []models.TopicSegment {
	if p.llm != nil {
		segments, err := LLMSegmentTopics(ctx, p.llm, messages)
		if err != nil {
			log.Printf("docgen: LLM segmentation failed: %v — falling back to time-gap topics", err)
		} else {
			return segments
		}
	}
	return segmentsFromTopicIDs(messages)
}

func filterMessagesBySeq(messages []models.Message, startSeq, endSeq int) []models.Message {
	var filtered []models.Message
	for _, m := range messages {
		if m.Seq >= startSeq && m.Seq <= endSeq {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func (p *Pipeline) processTopicSegment(ctx context.Context, conversationID int64, text string) error {
	matchedDoc, err := p.findMatchingDocument(ctx, text)
	if err != nil {
		log.Printf("docgen: document matching failed: %v — generating new doc", err)
	}

	if matchedDoc != nil {
		log.Printf("docgen: matched to document %d (%q)", matchedDoc.ID, matchedDoc.Title)
		return p.updateDocument(ctx, matchedDoc, text, conversationID)
	}

	return p.generateDocument(ctx, text, conversationID)
}

func (p *Pipeline) findMatchingDocument(ctx context.Context, topicText string) (*models.Document, error) {
	docs, err := p.store.ListDocuments()
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	if len(docs) == 0 {
		return nil, nil
	}

	titles := make([]documentTitle, len(docs))
	for i, d := range docs {
		titles[i] = documentTitle{ID: d.ID, Title: d.Title}
	}

	resp, err := p.llm.Generate(ctx, llm.Request{
		SystemPrompt: matchSystem,
		UserContent:  matchPrompt(topicText, titles),
		MaxTokens:    64,
	})
	if err != nil {
		return nil, fmt.Errorf("llm match: %w", err)
	}

	resp = strings.TrimSpace(resp)
	if strings.Contains(strings.ToLower(resp), "none") {
		return nil, nil
	}

	var docID int64
	if _, err := fmt.Sscanf(resp, "%d", &docID); err != nil {
		return nil, fmt.Errorf("parse match response %q: %w", resp, err)
	}

	doc, err := p.store.GetDocument(docID)
	if err != nil {
		return nil, fmt.Errorf("get matched doc %d: %w", docID, err)
	}

	return doc, nil
}

func (p *Pipeline) generateDocument(ctx context.Context, conversationText string, conversationID int64) error {
	resp, err := p.llm.Generate(ctx, llm.Request{
		SystemPrompt: generateSystem,
		UserContent:  generatePrompt(conversationText),
		MaxTokens:    4096,
	})
	if err != nil {
		return fmt.Errorf("llm generate: %w", err)
	}

	title, content := splitTitleContent(resp)

	doc, err := p.store.CreateDocument(title, content, conversationID)
	if err != nil {
		return fmt.Errorf("save document: %w", err)
	}

	log.Printf("docgen: created document %d (%q) from conversation %d", doc.ID, title, conversationID)
	return nil
}

func (p *Pipeline) updateDocument(ctx context.Context, doc *models.Document, conversationText string, conversationID int64) error {
	existingContent := doc.Content
	if doc.Title != "" {
		existingContent = "# " + doc.Title + "\n\n" + existingContent
	}

	resp, err := p.llm.Generate(ctx, llm.Request{
		SystemPrompt: updateSystem,
		UserContent:  updatePrompt(existingContent, conversationText),
		MaxTokens:    4096,
	})
	if err != nil {
		return fmt.Errorf("llm update: %w", err)
	}

	title, content := splitTitleContent(resp)

	if err := p.store.UpdateDocument(doc.ID, title, content, conversationID); err != nil {
		return fmt.Errorf("update document: %w", err)
	}

	log.Printf("docgen: updated document %d (%q) with conversation %d", doc.ID, title, conversationID)
	return nil
}

// RegenerateDocument re-generates a document from all its linked conversations.
func (p *Pipeline) RegenerateDocument(ctx context.Context, doc *models.Document) error {
	if p.llm == nil {
		return fmt.Errorf("no LLM configured")
	}

	var allText strings.Builder
	for _, cid := range doc.ConversationIDs {
		messages, err := p.store.GetMessages(cid, nil)
		if err != nil {
			log.Printf("docgen: skip conversation %d: %v", cid, err)
			continue
		}
		text := ExtractText(messages)
		if text != "" {
			allText.WriteString(text)
			allText.WriteString("\n---\n")
		}
	}

	combined := allText.String()
	if len(strings.TrimSpace(combined)) < 100 {
		return fmt.Errorf("linked conversations too short to regenerate")
	}

	resp, err := p.llm.Generate(ctx, llm.Request{
		SystemPrompt: generateSystem,
		UserContent:  generatePrompt(combined),
		MaxTokens:    4096,
	})
	if err != nil {
		return fmt.Errorf("llm generate: %w", err)
	}

	title, content := splitTitleContent(resp)

	if err := p.store.UpdateDocumentContent(doc.ID, title, content); err != nil {
		return fmt.Errorf("save regenerated doc: %w", err)
	}

	log.Printf("docgen: regenerated document %d (%q)", doc.ID, title)
	return nil
}

func splitTitleContent(text string) (title, content string) {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "# ") {
		if idx := strings.Index(text, "\n"); idx != -1 {
			title = strings.TrimSpace(text[2:idx])
			content = strings.TrimSpace(text[idx+1:])
			return title, content
		}
		return strings.TrimSpace(text[2:]), ""
	}
	return "", text
}
