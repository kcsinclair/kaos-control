// SPDX-License-Identifier: AGPL-3.0-or-later

package ideachat

import (
	"strings"
)

// ModelConfig holds the LLM model configuration for a conversation.
type ModelConfig struct {
	Model        string
	SystemPrompt string
	MaxTokens    int
}

// LLMMessage is a single message in a conversation turn.
type LLMMessage struct {
	Role    string
	Content string
}

// buildPrompt combines the system prompt and message history into a single
// prompt string for the claude CLI -p flag.
func buildPrompt(systemPrompt string, messages []LLMMessage) string {
	var sb strings.Builder
	if systemPrompt != "" {
		sb.WriteString(systemPrompt)
		sb.WriteString("\n\n---\n\n")
	}
	for i, m := range messages {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		switch m.Role {
		case "user":
			sb.WriteString("Human: ")
		case "assistant":
			sb.WriteString("Assistant: ")
		}
		sb.WriteString(m.Content)
	}
	return sb.String()
}
