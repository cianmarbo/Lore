package docgen

import (
	"fmt"
	"strings"
)

const generateSystem = `You are a technical documentation writer. Given a conversation between a user and an AI assistant, create a concise, well-structured document that captures the key knowledge from the conversation.

Rules:
- Start with a clear title on the first line prefixed with "# "
- Focus on decisions made, solutions found, and knowledge gained
- Use markdown formatting with headers, code blocks, and lists as appropriate
- Omit conversational filler — extract only useful technical content
- Keep it concise but complete — aim for 200-800 words
- Write in the present tense, as reference documentation`

const updateSystem = `You are a technical documentation writer. You are given an existing document and a new conversation that relates to the same topic. Update the document to incorporate new knowledge from the conversation.

Rules:
- Keep the "# Title" on the first line (update if the scope has changed)
- Preserve accurate existing content — only modify what the new conversation changes or extends
- Add new sections or details as needed
- Remove information that the new conversation shows to be outdated or incorrect
- Maintain the same concise, reference-documentation style
- Do not add a changelog or "updated" annotations — the document should read as a single coherent piece`

func generatePrompt(conversationText string) string {
	return fmt.Sprintf("Here is the conversation to document:\n\n%s", conversationText)
}

func updatePrompt(existingDoc, conversationText string) string {
	return fmt.Sprintf("Here is the existing document:\n\n%s\n\n---\n\nHere is the new conversation to incorporate:\n\n%s", existingDoc, conversationText)
}

const segmentSystem = `You are a conversation analyst. Given a conversation between a user and an AI assistant, identify distinct topic segments. Each segment is a contiguous run of messages about the same subject.

Rules:
- Each segment must have a short descriptive label (under 80 characters)
- Segments are defined by message number ranges (inclusive)
- Messages are numbered sequentially starting from the numbers shown in the input
- Every message must belong to exactly one segment (no gaps, no overlaps)
- Merge very short topics (1-2 messages) into adjacent segments
- Return ONLY valid JSON, no markdown code fences or other text

Return JSON in this exact format:
{"segments": [{"label": "...", "start": <first_msg_number>, "end": <last_msg_number>}]}`

const matchSystem = `You are a document matching assistant. Given a topic summary and a list of existing document titles, determine if the topic matches an existing document.

Rules:
- A match means the topic covers substantially the same subject as the document
- Respond with ONLY the document ID number if there is a match
- Respond with ONLY the word "none" if there is no match
- Do not explain your reasoning`

type documentTitle struct {
	ID    int64
	Title string
}

func segmentPrompt(numberedConversation string) string {
	return fmt.Sprintf("Here is the conversation with numbered messages:\n\n%s", numberedConversation)
}

func matchPrompt(topicText string, documents []documentTitle) string {
	var sb strings.Builder
	sb.WriteString("Topic text:\n\n")
	sb.WriteString(topicText)
	sb.WriteString("\n\n---\n\nExisting documents:\n\n")
	for _, d := range documents {
		sb.WriteString(fmt.Sprintf("[%d] %s\n", d.ID, d.Title))
	}
	if len(documents) == 0 {
		sb.WriteString("(none)\n")
	}
	return sb.String()
}
