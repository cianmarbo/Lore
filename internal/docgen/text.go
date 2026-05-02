package docgen

import (
	"strings"

	"lore/internal/models"
)

const maxExtractChars = 32000 // ~8K tokens

// ExtractText builds a plain-text summary of a conversation's messages
// suitable for passing to an LLM. Filters out noise like
// tool orchestration, acknowledgments, and debugging chatter, keeping
// only knowledge-bearing content.
func ExtractText(messages []models.Message) string {
	var b strings.Builder
	for _, m := range messages {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		text := strings.TrimSpace(m.Content)
		if text == "" {
			continue
		}
		if isNoise(m.Role, text) {
			continue
		}
		b.WriteString(m.Role)
		b.WriteString(": ")
		b.WriteString(text)
		b.WriteByte('\n')

		if b.Len() > maxExtractChars {
			break
		}
	}
	s := b.String()
	if len(s) > maxExtractChars {
		s = s[:maxExtractChars]
	}
	return s
}

// isNoise returns true if a message is filler rather than knowledge-bearing content.
func isNoise(role, text string) bool {
	lower := strings.ToLower(text)

	// Short messages are almost always acknowledgments or filler
	if len(text) < 20 {
		return isShortNoise(lower)
	}

	if role == "assistant" {
		return isAssistantNoise(lower)
	}

	if role == "user" {
		return isUserNoise(lower)
	}

	return false
}

// isShortNoise catches brief acknowledgments and filler from either role.
func isShortNoise(lower string) bool {
	noisy := []string{
		"yes", "yeah", "yep", "no", "nope",
		"ok", "okay", "k",
		"sure", "thanks", "thank you", "ty",
		"perfect", "great", "awesome", "nice",
		"got it", "sounds good", "go ahead",
		"please", "please do", "yes please",
		"correct", "exactly", "right",
		"done", "done.", "fixed", "fixed.",
		"lgtm",
		"continue", "proceed", "go on",
	}
	stripped := strings.TrimRight(lower, ".!? ")
	for _, n := range noisy {
		if stripped == n {
			return true
		}
	}
	return false
}

// isAssistantNoise detects Claude's typical filler patterns.
func isAssistantNoise(lower string) bool {
	// Tool orchestration — "Let me read/check/search/look at..."
	orchestration := []string{
		"let me read", "let me check", "let me search",
		"let me look", "let me find", "let me see",
		"let me verify", "let me examine", "let me explore",
		"let me run", "let me try", "let me fix",
		"i'll read", "i'll check", "i'll search",
		"i'll look", "i'll find", "i'll verify",
		"i'll examine", "i'll explore", "i'll run",
		"i'll try", "i'll fix",
		"now let me", "first, let me", "now i'll",
	}
	for _, prefix := range orchestration {
		if strings.HasPrefix(lower, prefix) {
			// Only noise if the message is short (just the orchestration line)
			if len(lower) < 120 {
				return true
			}
		}
	}

	// Brief status updates with no substance
	status := []string{
		"clean build",
		"the build succeeded",
		"that compiled",
		"no errors",
		"everything builds",
		"builds clean",
		"all tests pass",
		"tests pass",
		"the file has been updated",
		"the file was updated",
		"i've updated the file",
		"i've made the change",
		"changes have been made",
		"here's what i changed",
		"done. ",
		"fixed. ",
	}
	for _, s := range status {
		if strings.HasPrefix(lower, s) && len(lower) < 150 {
			return true
		}
	}

	return false
}

// isUserNoise detects common non-substantive user messages.
func isUserNoise(lower string) bool {
	// Slash commands and interrupts
	if strings.HasPrefix(lower, "/") {
		return true
	}

	// Generic prompts to continue
	continuePatterns := []string{
		"can you continue",
		"please continue",
		"keep going",
		"go ahead",
		"carry on",
		"what's next",
		"next step",
	}
	for _, p := range continuePatterns {
		if strings.HasPrefix(lower, p) && len(lower) < 80 {
			return true
		}
	}

	return false
}
