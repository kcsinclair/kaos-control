// SPDX-License-Identifier: AGPL-3.0-or-later

package ideachat

import (
	"context"
	"fmt"
	"os/exec"
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

// CallLLM sends messages to the claude CLI and returns the assistant text reply.
// The system prompt and conversation history are combined into a single -p prompt.
func CallLLM(ctx context.Context, cfg ModelConfig, messages []LLMMessage) (string, error) {
	prompt := buildPrompt(cfg.SystemPrompt, messages)

	args := []string{"--dangerously-skip-permissions", "-p", prompt}
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}

	out, err := exec.CommandContext(ctx, "claude", args...).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("claude exited %d: %s", ee.ExitCode(), strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("calling claude: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
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
