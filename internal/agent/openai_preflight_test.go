// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/config"
)

func TestOpenAIPreflight(t *testing.T) {
	tools := DefaultOpenAITools()

	t.Run("Mode C - valid token delta succeeds", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			promptTokens := 20
			if _, hasTools := req["tools"]; hasTools {
				promptTokens = 120 // tools increase tokens
			}
			resp := openAIChatCompletionResponse{
				ID: "chat-123",
				Usage: openAIUsage{
					PromptTokens:     promptTokens,
					CompletionTokens: 1,
					TotalTokens:      promptTokens + 1,
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer ts.Close()

		provider := config.Provider{
			Name:    "test-prov",
			BaseURL: ts.URL,
			Driver:  "openai-compatible",
		}

		err := VerifyToolCapability(context.Background(), ts.Client(), provider, "good-model", tools)
		if err != nil {
			t.Fatalf("expected nil error on valid delta, got: %v", err)
		}
	})

	t.Run("Mode A - silent tools drop triggers ErrToolsSilentlyDropped", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Always return 20 prompt tokens regardless of tools
			resp := openAIChatCompletionResponse{
				ID: "chat-123",
				Usage: openAIUsage{
					PromptTokens:     20,
					CompletionTokens: 1,
					TotalTokens:      21,
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer ts.Close()

		provider := config.Provider{
			Name:    "test-prov",
			BaseURL: ts.URL,
			Driver:  "openai-compatible",
		}

		err := VerifyToolCapability(context.Background(), ts.Client(), provider, "dolphin-llama", tools)
		if err == nil {
			t.Fatal("expected ErrToolsSilentlyDropped, got nil")
		}
		if !errors.Is(err, ErrToolsSilentlyDropped) {
			t.Errorf("expected ErrToolsSilentlyDropped, got: %v", err)
		}
	})

	t.Run("Mode B - HTTP 400 explicit rejection triggers ErrToolsUnsupported", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			_ = json.NewDecoder(r.Body).Decode(&req)
			if _, hasTools := req["tools"]; hasTools {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error": {"message": "gemma3:12b does not support tools"}}`))
				return
			}
			_ = json.NewEncoder(w).Encode(openAIChatCompletionResponse{
				ID: "chat-123",
				Usage: openAIUsage{
					PromptTokens: 20,
				},
			})
		}))
		defer ts.Close()

		provider := config.Provider{
			Name:    "ollama-prov",
			BaseURL: ts.URL,
			Driver:  "openai-compatible",
		}

		err := VerifyToolCapability(context.Background(), ts.Client(), provider, "gemma3:12b", tools)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrToolsUnsupported) {
			t.Errorf("expected ErrToolsUnsupported, got: %v", err)
		}
		if !strings.Contains(err.Error(), "gemma3:12b does not support tools") {
			t.Errorf("expected server error message, got: %v", err)
		}
	})

	t.Run("OpenRouter metadata rejection", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/models" {
				resp := openRouterModelsResponse{
					Data: []struct {
						ID                  string   `json:"id"`
						Name                string   `json:"name"`
						SupportedParameters []string `json:"supported_parameters"`
					}{
						{
							ID:                  "vendor/notool-model",
							Name:                "No Tool Model",
							SupportedParameters: []string{"temperature", "top_p"},
						},
					},
				}
				_ = json.NewEncoder(w).Encode(resp)
				return
			}
			_ = json.NewEncoder(w).Encode(openAIChatCompletionResponse{
				Usage: openAIUsage{PromptTokens: 50},
			})
		}))
		defer ts.Close()

		provider := config.Provider{
			Name:    "openrouter",
			BaseURL: ts.URL,
			Driver:  "openai-compatible",
		}

		err := VerifyToolCapability(context.Background(), ts.Client(), provider, "vendor/notool-model", tools)
		if err == nil {
			t.Fatal("expected ErrToolsUnsupported, got nil")
		}
		if !errors.Is(err, ErrToolsUnsupported) {
			t.Errorf("expected ErrToolsUnsupported, got: %v", err)
		}
	})
}
