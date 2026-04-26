package ideachat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	anthropicAPIURL = "https://api.anthropic.com/v1/messages"
	llmTimeout      = 30 * time.Second
	anthropicVersion = "2023-06-01"
)

// ModelConfig holds the LLM model configuration for a conversation.
type ModelConfig struct {
	Model          string
	SystemPrompt   string
	MaxTokens      int
}

// LLMMessage is a single message in an Anthropic API request.
type LLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CallLLM calls the Anthropic Messages API and returns the assistant text reply.
// It uses ANTHROPIC_API_KEY from the environment and enforces a 30-second timeout.
func CallLLM(ctx context.Context, cfg ModelConfig, messages []LLMMessage) (string, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
	}

	model := cfg.Model
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 2048
	}

	type requestBody struct {
		Model     string       `json:"model"`
		MaxTokens int          `json:"max_tokens"`
		System    string       `json:"system,omitempty"`
		Messages  []LLMMessage `json:"messages"`
	}

	body := requestBody{
		Model:     model,
		MaxTokens: maxTokens,
		System:    cfg.SystemPrompt,
		Messages:  messages,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("CallLLM: marshalling request: %w", err)
	}

	callCtx, cancel := context.WithTimeout(ctx, llmTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(callCtx, http.MethodPost, anthropicAPIURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("CallLLM: creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("CallLLM: HTTP request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("CallLLM: reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("CallLLM: API returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Error *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return "", fmt.Errorf("CallLLM: parsing response: %w", err)
	}
	if apiResp.Error != nil {
		return "", fmt.Errorf("CallLLM: API error %s: %s", apiResp.Error.Type, apiResp.Error.Message)
	}

	for _, block := range apiResp.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}
	return "", fmt.Errorf("CallLLM: no text content in response")
}
