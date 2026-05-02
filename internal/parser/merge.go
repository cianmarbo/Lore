package parser

import (
	"regexp"
	"strings"
	"time"
	"unicode"

	"lore/internal/models"
)

var (
	mdFormatRe    = regexp.MustCompile(`[*#` + "`" + `|]`)
	leadingSymRe  = regexp.MustCompile(`^[^\w"'(]+`)
	sentenceEndRe = regexp.MustCompile(`^(.+?[.?!])\s`)
)

// MergeTurns collapses consecutive assistant messages and folds tool-result-only
// user messages into the previous assistant turn.
func MergeTurns(messages []rawMessage) []models.ParsedTurn {
	var turns []models.ParsedTurn

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			text := strings.TrimSpace(msg.Text)
			// Tool-result-only user messages fold into previous assistant turn
			if text == "" && len(msg.ToolResults) > 0 && len(turns) > 0 && turns[len(turns)-1].Role == "assistant" {
				turns[len(turns)-1].ToolResults = append(turns[len(turns)-1].ToolResults, msg.ToolResults...)
				continue
			}
			turns = append(turns, models.ParsedTurn{
				Role:        "user",
				Content:     text,
				ToolResults: msg.ToolResults,
				Timestamp:   msg.Timestamp,
			})

		case "assistant":
			text := strings.TrimSpace(msg.Text)
			// Merge into existing assistant turn if consecutive
			if len(turns) > 0 && turns[len(turns)-1].Role == "assistant" {
				turn := &turns[len(turns)-1]
				if text != "" {
					if turn.Content != "" {
						turn.Content += "\n\n" + text
					} else {
						turn.Content = text
					}
				}
				turn.Tools = append(turn.Tools, msg.Tools...)
				continue
			}
			turns = append(turns, models.ParsedTurn{
				Role:      "assistant",
				Content:   text,
				Tools:     append([]models.ToolSummary(nil), msg.Tools...),
				Timestamp: msg.Timestamp,
			})

		case "system":
			turns = append(turns, models.ParsedTurn{
				Role:      "system",
				Content:   msg.Text,
				Timestamp: msg.Timestamp,
			})
		}
	}

	return turns
}

// DetectTopics groups turns into topics based on time gaps between user messages.
// A new topic starts when the gap exceeds gapMinutes.
func DetectTopics(turns []models.ParsedTurn, gapMinutes float64) ([]models.ParsedTurn, []models.ParsedTopic) {
	var topics []models.ParsedTopic
	currentTopic := -1
	var lastUserTime time.Time

	for i := range turns {
		turn := &turns[i]
		userText := strings.TrimSpace(turn.Content)

		if turn.Role == "user" && len(userText) > 5 {
			startNew := false
			if lastUserTime.IsZero() {
				startNew = true
			} else if !turn.Timestamp.IsZero() {
				gap := turn.Timestamp.Sub(lastUserTime).Minutes()
				if gap >= gapMinutes {
					startNew = true
				}
			}

			if startNew {
				currentTopic++
				label := makeTopicLabel(userText)
				topics = append(topics, models.ParsedTopic{
					Seq:       currentTopic,
					Label:     label,
					StartedAt: turn.Timestamp,
					MsgCount:  0,
				})
			}

			if !turn.Timestamp.IsZero() {
				lastUserTime = turn.Timestamp
			}
		}

		topicIdx := currentTopic
		if topicIdx < 0 {
			topicIdx = 0
		}
		turn.TopicSeq = topicIdx
		if topicIdx < len(topics) {
			topics[topicIdx].MsgCount++
		}
	}

	return turns, topics
}

func makeTopicLabel(text string) string {
	// Strip XML/HTML tags and markdown formatting
	clean := xmlTagRe.ReplaceAllString(text, "")
	clean = orphanTagRe.ReplaceAllString(clean, "")
	clean = mdFormatRe.ReplaceAllString(clean, "")
	clean = strings.TrimSpace(clean)

	// Strip leading symbols
	clean = leadingSymRe.ReplaceAllString(clean, "")
	clean = strings.TrimSpace(clean)

	// Collapse whitespace
	clean = collapseSpaces(clean)

	if len(clean) < 5 {
		return "Untitled"
	}

	// Take first sentence if short enough
	if m := sentenceEndRe.FindStringSubmatch(clean); m != nil && len(m[1]) <= 100 {
		return m[1]
	}

	if len(clean) > 80 {
		// Break at word boundary
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

// ParseConversation is the main entry point: parse file, merge turns, detect topics.
func ParseConversation(path string) (*models.ParsedConversation, error) {
	sessionID, messages, err := ParseFile(path)
	if err != nil {
		return nil, err
	}

	turns := MergeTurns(messages)
	turns, topics := DetectTopics(turns, 30)

	return &models.ParsedConversation{
		SessionID: sessionID,
		Turns:     turns,
		Topics:    topics,
	}, nil
}
