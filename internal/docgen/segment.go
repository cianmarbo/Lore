package docgen

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"lore/internal/llm"
	"lore/internal/models"
)

const maxSegmentPromptChars = 32000

var (
	segXMLTagRe    = regexp.MustCompile(`<[a-zA-Z][a-zA-Z0-9_-]*>.*?</[a-zA-Z][a-zA-Z0-9_-]*>`)
	segOrphanTagRe = regexp.MustCompile(`</?[a-zA-Z][a-zA-Z0-9_-]*>`)
	segMdFormatRe  = regexp.MustCompile(`[*#` + "`" + `|]`)
	segLeadingRe   = regexp.MustCompile(`^[^\w"'(]+`)
	segSentenceRe  = regexp.MustCompile(`^(.+?[.?!])\s`)
)

// LLMSegmentTopics asks the LLM to identify topic boundaries in the conversation.
func LLMSegmentTopics(ctx context.Context, provider llm.Provider, messages []models.Message) ([]models.TopicSegment, error) {
	var sb strings.Builder
	for _, m := range messages {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		text := strings.TrimSpace(m.Content)
		if text == "" {
			continue
		}
		if len(text) > 500 {
			text = text[:500] + "..."
		}
		sb.WriteString(fmt.Sprintf("[%d] %s: %s\n", m.Seq, m.Role, text))
		if sb.Len() > maxSegmentPromptChars {
			break
		}
	}

	resp, err := provider.Generate(ctx, llm.Request{
		SystemPrompt: segmentSystem,
		UserContent:  segmentPrompt(sb.String()),
		MaxTokens:    2048,
	})
	if err != nil {
		return nil, fmt.Errorf("llm segment: %w", err)
	}

	return parseSegmentResponse(resp, messages)
}

type segmentResponse struct {
	Segments []struct {
		Label string `json:"label"`
		Start int    `json:"start"`
		End   int    `json:"end"`
	} `json:"segments"`
}

func parseSegmentResponse(resp string, messages []models.Message) ([]models.TopicSegment, error) {
	resp = strings.TrimSpace(resp)

	// Extract JSON if wrapped in markdown code fences or surrounding text
	if start := strings.Index(resp, "{"); start != -1 {
		if end := strings.LastIndex(resp, "}"); end != -1 && end > start {
			resp = resp[start : end+1]
		}
	}

	var parsed segmentResponse
	if err := json.Unmarshal([]byte(resp), &parsed); err != nil {
		return nil, fmt.Errorf("parse segment JSON: %w", err)
	}

	if len(parsed.Segments) == 0 {
		return nil, fmt.Errorf("LLM returned zero segments")
	}

	seqToTime := make(map[int]time.Time, len(messages))
	for _, m := range messages {
		seqToTime[m.Seq] = m.Timestamp
	}

	var segments []models.TopicSegment
	for _, s := range parsed.Segments {
		segments = append(segments, models.TopicSegment{
			Label:     s.Label,
			StartSeq:  s.Start,
			EndSeq:    s.End,
			StartedAt: seqToTime[s.Start],
		})
	}

	return segments, nil
}

// segmentsFromTopicIDs builds TopicSegments from existing topic_id assignments on messages.
func segmentsFromTopicIDs(messages []models.Message) []models.TopicSegment {
	if len(messages) == 0 {
		return nil
	}

	var segments []models.TopicSegment
	var currentTopicID *int64
	segStart := 0

	for i, m := range messages {
		sameGroup := (currentTopicID == nil && m.TopicID == nil) ||
			(currentTopicID != nil && m.TopicID != nil && *currentTopicID == *m.TopicID)

		if i > 0 && !sameGroup {
			segments = append(segments, models.TopicSegment{
				Label:     makeSegmentLabel(messages, messages[segStart].Seq, messages[i-1].Seq),
				StartSeq:  messages[segStart].Seq,
				EndSeq:    messages[i-1].Seq,
				StartedAt: messages[segStart].Timestamp,
			})
			segStart = i
		}
		currentTopicID = m.TopicID
	}

	// Final segment
	segments = append(segments, models.TopicSegment{
		Label:     makeSegmentLabel(messages, messages[segStart].Seq, messages[len(messages)-1].Seq),
		StartSeq:  messages[segStart].Seq,
		EndSeq:    messages[len(messages)-1].Seq,
		StartedAt: messages[segStart].Timestamp,
	})

	return segments
}

func makeSegmentLabel(messages []models.Message, startSeq, endSeq int) string {
	for _, m := range messages {
		if m.Seq < startSeq {
			continue
		}
		if m.Seq > endSeq {
			break
		}
		if m.Role == "user" && strings.TrimSpace(m.Content) != "" {
			return cleanLabel(m.Content)
		}
	}
	return "Untitled"
}

func cleanLabel(text string) string {
	clean := segXMLTagRe.ReplaceAllString(text, "")
	clean = segOrphanTagRe.ReplaceAllString(clean, "")
	clean = segMdFormatRe.ReplaceAllString(clean, "")
	clean = strings.TrimSpace(clean)
	clean = segLeadingRe.ReplaceAllString(clean, "")
	clean = strings.TrimSpace(clean)
	clean = collapseSpaces(clean)

	if len(clean) < 5 {
		return "Untitled"
	}

	if m := segSentenceRe.FindStringSubmatch(clean); m != nil && len(m[1]) <= 100 {
		return m[1]
	}

	if len(clean) > 80 {
		truncated := clean[:80]
		if idx := strings.LastIndex(truncated, " "); idx > 0 {
			truncated = truncated[:idx]
		}
		return truncated + "..."
	}

	return clean
}

func collapseSpaces(s string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteRune(' ')
			}
			prevSpace = true
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return b.String()
}
