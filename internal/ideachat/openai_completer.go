// SPDX-License-Identifier: AGPL-3.0-or-later

package ideachat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/kaos-control/kaos-control/internal/config"
)

// openAICompleter is a plain, tool-free, non-streaming Completer against an
// OpenAI-compatible /v1/chat/completions endpoint. It is deliberately
// standalone rather than reusing the async agent-loop's openai-compatible
// driver, which is welded to streaming, tool-calling, preflight and
// recovery — a blocking, tool-free client is smaller and clearer.
type openAICompleter struct {
	provider config.ProviderConfig
	client   *http.Client
}

type openAICompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAICompletionRequest struct {
	Model     string                    `json:"model"`
	Messages  []openAICompletionMessage `json:"messages"`
	MaxTokens int                       `json:"max_tokens,omitempty"`
}

type openAICompletionResponse struct {
	Choices []struct {
		Message struct {
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (c *openAICompleter) Complete(ctx context.Context, cfg ModelConfig, messages []LLMMessage) (string, error) {
	reqBody := openAICompletionRequest{
		Model:    cfg.Model,
		Messages: make([]openAICompletionMessage, 0, len(messages)+1),
	}
	if cfg.SystemPrompt != "" {
		reqBody.Messages = append(reqBody.Messages, openAICompletionMessage{Role: "system", Content: cfg.SystemPrompt})
	}
	for _, m := range messages {
		reqBody.Messages = append(reqBody.Messages, openAICompletionMessage{Role: m.Role, Content: m.Content})
	}
	if cfg.MaxTokens > 0 {
		reqBody.MaxTokens = cfg.MaxTokens
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("ideachat: marshaling openai-compatible request: %w", err)
	}

	url := strings.TrimRight(c.provider.BaseURL, "/") + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("ideachat: building openai-compatible request: %w", c.scrub(err))
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.provider.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.provider.APIKey)
	}
	for k, v := range c.provider.ExtraHeaders {
		httpReq.Header.Set(k, v)
	}

	client := c.client
	if client == nil {
		client = &http.Client{}
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("ideachat: calling %s: %w", c.provider.Name, c.scrub(err))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ideachat: reading %s response: %w", c.provider.Name, c.scrub(err))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("ideachat: %s returned status %d: %s", c.provider.Name, resp.StatusCode, c.scrubString(errorSnippet(respBody)))
	}

	var parsed openAICompletionResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("ideachat: decoding %s response: %w", c.provider.Name, c.scrub(err))
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("ideachat: %s response had no choices", c.provider.Name)
	}

	content, err := decodeMessageContent(parsed.Choices[0].Message.Content)
	if err != nil {
		return "", fmt.Errorf("ideachat: decoding %s message content: %w", c.provider.Name, c.scrub(err))
	}

	return strings.TrimSpace(content), nil
}

// scrub removes the provider's api_key value from an error's text so it is
// never echoed back to logs or callers.
func (c *openAICompleter) scrub(err error) error {
	if err == nil || c.provider.APIKey == "" {
		return err
	}
	return fmt.Errorf("%s", c.scrubString(err.Error()))
}

// scrubString removes the provider's api_key value from a plain string.
func (c *openAICompleter) scrubString(s string) string {
	if c.provider.APIKey == "" {
		return s
	}
	return strings.ReplaceAll(s, c.provider.APIKey, "***")
}

// errorSnippet bounds how much of a non-2xx response body is echoed into an
// error message.
func errorSnippet(body []byte) string {
	const maxLen = 500
	s := strings.TrimSpace(string(body))
	if len(s) > maxLen {
		s = s[:maxLen] + "..."
	}
	return s
}

// decodeMessageContent decodes an OpenAI-compatible message content field,
// which is a plain string for most providers but may be an array of
// {"type":"text","text":"..."} parts for others. Non-text parts are ignored.
func decodeMessageContent(raw json.RawMessage) (string, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}

	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return "", fmt.Errorf("unrecognized content shape: %w", err)
	}
	var sb strings.Builder
	for _, p := range parts {
		if p.Type == "" || p.Type == "text" {
			sb.WriteString(p.Text)
		}
	}
	return sb.String(), nil
}
