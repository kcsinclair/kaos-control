// SPDX-License-Identifier: AGPL-3.0-or-later

package ideachat

import (
	"strings"

	"github.com/kaos-control/kaos-control/internal/config"
)

// ModelConfig holds the LLM model configuration for a conversation. It is
// internal-only and must never be marshalled to JSON/YAML or otherwise
// exposed on a serialised surface — carrying api_key/extra_headers via
// Provider relies on that.
type ModelConfig struct {
	Model        string
	SystemPrompt string
	MaxTokens    int

	// Provider identifies the app-level provider + driver to route this call
	// through. Nil means "use the Claude CLI default" (today's behaviour).
	Provider *config.ProviderConfig
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
