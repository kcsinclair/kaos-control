// SPDX-License-Identifier: AGPL-3.0-or-later

package ideachat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/config"
)

// TestCallLLM_DispatchEndToEnd_OpenAICompatible is a Milestone 6 regression
// test: it drives the real (unstubbed) CallLLM dispatcher — not a test
// double — through the OpenAI-compatible path via Converse, confirming the
// dispatcher-to-completer wiring produces the same proposed-artifact shape
// as the CLI path and that the provider's api_key never appears in the
// session history or the response (NFR-1, NFR-2).
func TestCallLLM_DispatchEndToEnd_OpenAICompatible(t *testing.T) {
	const apiKey = "sk-super-secret-do-not-leak"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":` +
			`"{\"action\":\"propose\",\"reply\":\"Here you go.\",\"slug\":\"dark-mode\",\"title\":\"Dark Mode\",\"labels\":[],\"body\":\"# Dark Mode\"}"` +
			`}}]}`))
	}))
	defer srv.Close()

	cfg := ModelConfig{
		Model:        "test-model",
		SystemPrompt: "You are an idea-capture assistant.",
		Provider: &config.ProviderConfig{
			Name:    "local-llama",
			BaseURL: srv.URL,
			Driver:  "openai-compatible",
			APIKey:  apiKey,
		},
	}

	sess := &Session{}
	resp, err := Converse(context.Background(), sess, "add a dark mode toggle please", nil, nil, cfg)
	if err != nil {
		t.Fatalf("Converse: %v", err)
	}
	if resp.Status != StatusProposed {
		t.Fatalf("Status = %q, want %q", resp.Status, StatusProposed)
	}
	if resp.ProposedSlug != "dark-mode" {
		t.Errorf("ProposedSlug = %q, want dark-mode", resp.ProposedSlug)
	}
	if resp.ProposedFM == nil || resp.ProposedFM.Title != "Dark Mode" {
		t.Errorf("ProposedFM = %+v, want title Dark Mode", resp.ProposedFM)
	}

	for _, m := range sess.Messages {
		if strings.Contains(m.Content, apiKey) {
			t.Fatalf("session message leaks api_key: %q", m.Content)
		}
	}
	if strings.Contains(resp.Reply, apiKey) {
		t.Fatalf("reply leaks api_key: %q", resp.Reply)
	}
}

// TestCallLLM_DispatchEndToEnd_UnknownDriver verifies the dispatcher's
// defensive branch (M4) surfaces a named error rather than reaching a
// completer for a provider driver config validation should have already
// rejected.
func TestCallLLM_DispatchEndToEnd_UnknownDriver(t *testing.T) {
	cfg := ModelConfig{
		Model: "test-model",
		Provider: &config.ProviderConfig{
			Name:   "mystery",
			Driver: "carrier-pigeon",
		},
	}
	_, err := CallLLM(context.Background(), cfg, []LLMMessage{{Role: "user", Content: "hi"}})
	if err == nil {
		t.Fatal("expected error for unsupported provider driver")
	}
	if !strings.Contains(err.Error(), "mystery") || !strings.Contains(err.Error(), "carrier-pigeon") {
		t.Errorf("error = %v, want it to name provider and driver", err)
	}
}
