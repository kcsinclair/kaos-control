// SPDX-License-Identifier: AGPL-3.0-or-later

package ideachat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/config"
)

func TestOpenAICompleter_RequestShape(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotHeader string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotHeader = r.Header.Get("X-Custom")
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"hello there"}}]}`))
	}))
	defer srv.Close()

	c := &openAICompleter{provider: config.ProviderConfig{
		Name:         "local",
		BaseURL:      srv.URL + "/",
		Driver:       "openai-compatible",
		APIKey:       "secret-key",
		ExtraHeaders: map[string]string{"X-Custom": "yes"},
	}}

	out, err := c.Complete(context.Background(), ModelConfig{Model: "test-model", SystemPrompt: "be helpful"}, []LLMMessage{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "hello"},
		{Role: "user", Content: "how are you"},
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if out != "hello there" {
		t.Errorf("out = %q, want %q", out, "hello there")
	}
	if gotPath != "/v1/chat/completions" {
		t.Errorf("path = %q, want /v1/chat/completions", gotPath)
	}
	if gotAuth != "Bearer secret-key" {
		t.Errorf("Authorization = %q, want Bearer secret-key", gotAuth)
	}
	if gotHeader != "yes" {
		t.Errorf("X-Custom = %q, want yes", gotHeader)
	}
	if gotBody["model"] != "test-model" {
		t.Errorf("model = %v, want test-model", gotBody["model"])
	}
	if _, hasTools := gotBody["tools"]; hasTools {
		t.Error("request body must not contain a tools key")
	}
	if _, hasMaxTokens := gotBody["max_tokens"]; hasMaxTokens {
		t.Error("max_tokens must be omitted when ModelConfig.MaxTokens is 0")
	}
	msgs, ok := gotBody["messages"].([]any)
	if !ok || len(msgs) != 4 {
		t.Fatalf("messages = %v, want 4 entries (system + 3)", gotBody["messages"])
	}
	wantRoles := []string{"system", "user", "assistant", "user"}
	for i, want := range wantRoles {
		m := msgs[i].(map[string]any)
		if m["role"] != want {
			t.Errorf("messages[%d].role = %v, want %q", i, m["role"], want)
		}
	}
	if msgs[0].(map[string]any)["content"] != "be helpful" {
		t.Errorf("system message content = %v, want %q", msgs[0], "be helpful")
	}
}

func TestOpenAICompleter_NoAuthHeaderWhenNoAPIKey(t *testing.T) {
	var authSet bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, authSet = r.Header["Authorization"]
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	c := &openAICompleter{provider: config.ProviderConfig{Name: "local", BaseURL: srv.URL}}
	if _, err := c.Complete(context.Background(), ModelConfig{Model: "m"}, nil); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if authSet {
		t.Error("Authorization header set, want unset when api_key is empty")
	}
}

func TestOpenAICompleter_MaxTokensOmittedUnlessPositive(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	c := &openAICompleter{provider: config.ProviderConfig{Name: "local", BaseURL: srv.URL}}
	if _, err := c.Complete(context.Background(), ModelConfig{Model: "m", MaxTokens: 128}, nil); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if gotBody["max_tokens"] != float64(128) {
		t.Errorf("max_tokens = %v, want 128", gotBody["max_tokens"])
	}
}

func TestOpenAICompleter_NonTextContentParts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":[{"type":"text","text":"part one "},{"type":"text","text":"part two"}]}}]}`))
	}))
	defer srv.Close()

	c := &openAICompleter{provider: config.ProviderConfig{Name: "local", BaseURL: srv.URL}}
	out, err := c.Complete(context.Background(), ModelConfig{Model: "m"}, nil)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if out != "part one part two" {
		t.Errorf("out = %q, want %q", out, "part one part two")
	}
}

func TestOpenAICompleter_NonSuccessStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid api key AbCdEf123"}`))
	}))
	defer srv.Close()

	c := &openAICompleter{provider: config.ProviderConfig{Name: "local", BaseURL: srv.URL, APIKey: "AbCdEf123"}}
	_, err := c.Complete(context.Background(), ModelConfig{Model: "m"}, nil)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if strings.Contains(err.Error(), "AbCdEf123") {
		t.Errorf("error leaks api_key: %v", err)
	}
}

func TestOpenAICompleter_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c := &openAICompleter{provider: config.ProviderConfig{Name: "local", BaseURL: srv.URL}}
	if _, err := c.Complete(context.Background(), ModelConfig{Model: "m"}, nil); err == nil {
		t.Fatal("expected error for malformed JSON response")
	}
}

func TestOpenAICompleter_EmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv.Close()

	c := &openAICompleter{provider: config.ProviderConfig{Name: "local", BaseURL: srv.URL}}
	if _, err := c.Complete(context.Background(), ModelConfig{Model: "m"}, nil); err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestOpenAICompleter_ContextCanceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := &openAICompleter{provider: config.ProviderConfig{Name: "local", BaseURL: srv.URL}}
	if _, err := c.Complete(ctx, ModelConfig{Model: "m"}, nil); err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestOpenAICompleter_UnreachableEndpoint(t *testing.T) {
	c := &openAICompleter{provider: config.ProviderConfig{Name: "local", BaseURL: "http://127.0.0.1:1"}}
	if _, err := c.Complete(context.Background(), ModelConfig{Model: "m"}, nil); err == nil {
		t.Fatal("expected error for unreachable endpoint")
	}
}
