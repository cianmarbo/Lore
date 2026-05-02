package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"lore/internal/models"
)

// jsonlLine represents a single line from the Claude Code JSONL session log.
type jsonlLine struct {
	Type      string          `json:"type"`
	Subtype   string          `json:"subtype"`
	Timestamp string          `json:"timestamp"`
	Content   json.RawMessage `json:"content"`
	Message   json.RawMessage `json:"message"`
	IsMeta    bool            `json:"isMeta"`
}

// messageEnvelope is the "message" field within a JSONL line.
type messageEnvelope struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// contentBlock is one element of a content array.
type contentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
}

// rawMessage is the intermediate parse result before merging.
type rawMessage struct {
	Role        string
	Text        string
	Tools       []models.ToolSummary
	ToolResults []string
	Timestamp   time.Time
}

var (
	xmlTagRe     = regexp.MustCompile(`<[a-zA-Z][a-zA-Z0-9_-]*>.*?</[a-zA-Z][a-zA-Z0-9_-]*>`)
	orphanTagRe  = regexp.MustCompile(`</?[a-zA-Z][a-zA-Z0-9_-]*>`)
	bracketPrefixRe = regexp.MustCompile(`^\[.*?\]\s*`)
	cmdNameRe    = regexp.MustCompile(`<command-name>(.*?)</command-name>`)
	caveatRe     = regexp.MustCompile(`(?i)^Caveat:`)
	farewellRe   = regexp.MustCompile(`(?i)^(Bye!?|See ya!?|Goodbye!?)$`)
	interruptRe  = regexp.MustCompile(`(?i)^(Request interrupted|No response requested)`)
)

func parseTimestamp(ts string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.000Z", ts)
		if err != nil {
			return time.Time{}
		}
	}
	return t
}

// ParseFile reads a Claude Code JSONL session log and returns raw messages.
func ParseFile(path string) (string, []rawMessage, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", nil, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	var messages []rawMessage
	seenTexts := make(map[string]bool)
	var sessionID string

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024) // 10MB max line

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var jl jsonlLine
		if err := json.Unmarshal([]byte(line), &jl); err != nil {
			continue
		}

		// Extract session ID from first line that has it
		if sessionID == "" {
			var raw map[string]json.RawMessage
			json.Unmarshal([]byte(line), &raw)
			if sid, ok := raw["sessionId"]; ok {
				json.Unmarshal(sid, &sessionID)
			}
		}

		// Handle system slash commands
		if jl.Type == "system" {
			if jl.Subtype == "local_command" {
				var content string
				if err := json.Unmarshal(jl.Content, &content); err == nil {
					if m := cmdNameRe.FindStringSubmatch(content); m != nil {
						messages = append(messages, rawMessage{
							Role:      "system",
							Text:      "Slash command: " + m[1],
							Timestamp: parseTimestamp(jl.Timestamp),
						})
					}
				}
			}
			continue
		}

		if jl.Type != "user" && jl.Type != "assistant" {
			continue
		}

		if len(jl.Message) == 0 {
			continue
		}

		var env messageEnvelope
		if err := json.Unmarshal(jl.Message, &env); err != nil {
			continue
		}

		ts := parseTimestamp(jl.Timestamp)

		if jl.Type == "user" && env.Role == "user" {
			text := extractText(env.Content)
			toolResults := extractToolResults(env.Content)

			// Skip /init meta prompts
			if jl.IsMeta && strings.HasPrefix(text, "Please analyze this codebase") {
				messages = append(messages, rawMessage{
					Role:      "system",
					Text:      "/init — auto-generated CLAUDE.md prompt",
					Timestamp: ts,
				})
				continue
			}

			// Clean up XML tags
			text = xmlTagRe.ReplaceAllString(text, "")
			text = orphanTagRe.ReplaceAllString(text, "")
			text = bracketPrefixRe.ReplaceAllString(text, "")
			text = strings.TrimSpace(text)

			// Skip noise
			if caveatRe.MatchString(text) || farewellRe.MatchString(text) || interruptRe.MatchString(text) {
				continue
			}

			if text != "" || len(toolResults) > 0 {
				messages = append(messages, rawMessage{
					Role:        "user",
					Text:        text,
					ToolResults: toolResults,
					Timestamp:   ts,
				})
			}

		} else if jl.Type == "assistant" && env.Role == "assistant" {
			text := extractText(env.Content)
			tools := extractToolUses(env.Content)

			// Deduplicate streaming chunks
			dedupKey := text
			if len(dedupKey) > 200 {
				dedupKey = dedupKey[:200]
			}
			if dedupKey != "" && seenTexts[dedupKey] && len(tools) == 0 {
				continue
			}
			if dedupKey != "" {
				seenTexts[dedupKey] = true
			}

			if text != "" || len(tools) > 0 {
				messages = append(messages, rawMessage{
					Role:      "assistant",
					Text:      text,
					Tools:     tools,
					Timestamp: ts,
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return sessionID, nil, fmt.Errorf("scanning file: %w", err)
	}

	return sessionID, messages, nil
}

func extractText(content json.RawMessage) string {
	// Try as string first
	var s string
	if err := json.Unmarshal(content, &s); err == nil {
		return strings.TrimSpace(s)
	}

	// Try as array of content blocks
	var blocks []contentBlock
	if err := json.Unmarshal(content, &blocks); err != nil {
		return ""
	}

	var parts []string
	for _, b := range blocks {
		if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
			parts = append(parts, strings.TrimSpace(b.Text))
		}
	}
	return strings.Join(parts, "\n\n")
}

func extractToolUses(content json.RawMessage) []models.ToolSummary {
	var blocks []contentBlock
	if err := json.Unmarshal(content, &blocks); err != nil {
		return nil
	}

	var tools []models.ToolSummary
	for _, b := range blocks {
		if b.Type != "tool_use" {
			continue
		}
		summary := formatToolSummary(b.Name, b.Input)
		tools = append(tools, models.ToolSummary{
			Name:    b.Name,
			Summary: summary,
		})
	}
	return tools
}

func formatToolSummary(name string, input json.RawMessage) string {
	var inp map[string]interface{}
	json.Unmarshal(input, &inp)

	getStr := func(key string) string {
		if v, ok := inp[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}

	switch name {
	case "Read":
		return "Read: " + getStr("file_path")
	case "Glob":
		return "Glob: " + getStr("pattern")
	case "Grep":
		return fmt.Sprintf("Grep: /%s/ in %s", getStr("pattern"), getStr("path"))
	case "Edit":
		return "Edit: " + getStr("file_path")
	case "Write":
		return "Write: " + getStr("file_path")
	case "Bash":
		cmd := getStr("command")
		if len(cmd) > 120 {
			cmd = cmd[:120] + "..."
		}
		return "Bash: " + cmd
	case "Agent":
		return "Agent: " + getStr("description")
	case "ToolSearch":
		return "ToolSearch: " + getStr("query")
	default:
		return name
	}
}

func extractToolResults(content json.RawMessage) []string {
	var blocks []contentBlock
	if err := json.Unmarshal(content, &blocks); err != nil {
		return nil
	}

	var results []string
	for _, b := range blocks {
		if b.Type != "tool_result" {
			continue
		}
		text := extractToolResultText(b.Content)
		if text != "" {
			results = append(results, text)
		}
	}
	return results
}

func extractToolResultText(content json.RawMessage) string {
	// Try as string
	var s string
	if err := json.Unmarshal(content, &s); err == nil {
		return strings.TrimSpace(s)
	}

	// Try as array
	var blocks []contentBlock
	if err := json.Unmarshal(content, &blocks); err != nil {
		return ""
	}

	var parts []string
	for _, b := range blocks {
		if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
			parts = append(parts, strings.TrimSpace(b.Text))
		}
	}
	return strings.Join(parts, "\n")
}
