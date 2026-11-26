// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
)

// TestOpenAI_Warmup verifies that when a local model takes more than 5s to
// produce its first token, the driver emits a "warming_up" progress event,
// and that TTFT is recorded once the token finally arrives (Milestone 3).
func TestOpenAI_Warmup(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)

		if stream, _ := req["stream"].(bool); !stream {
			promptTokens := 10
			if _, hasTools := req["tools"]; hasTools {
				promptTokens = 50
			}
			_ = json.NewEncoder(w).Encode(openAIChatCompletionResponse{
				Usage: openAIUsage{PromptTokens: promptTokens},
			})
			return
		}

		// Simulate a lazy-loading local model: pause well past the 5s
		// warmup threshold before the first token.
		time.Sleep(5300 * time.Millisecond)

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		fmt.Fprint(w, "data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hi\"},\"finish_reason\":null}]}\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		fmt.Fprint(w, "data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer ts.Close()

	driver := &OpenAICompatibleDriver{
		Providers: []config.Provider{
			{Name: "local-test", BaseURL: ts.URL, Driver: "openai-compatible"},
		},
		HTTPClient: ts.Client(),
	}

	var ttftMs int64
	run := Run{
		RunID:        "run-warmup",
		AgentName:    "warmup-agent",
		Driver:       "openai-compatible",
		ProviderName: "local-test",
		Model:        "gemma-4-26b",
		PromptText:   "System prompt\n\n---\n\nUser prompt",
		OnTTFT:       func(ms int64) { ttftMs = ms },
	}

	proc, err := driver.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	var sawWarmingUp, sawGenerating bool
	for ev := range proc.Progress() {
		if ev.Event == nil {
			continue
		}
		switch ev.Event["stage"] {
		case "warming_up":
			sawWarmingUp = true
		case "generating":
			sawGenerating = true
		}
	}

	if err := proc.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if !sawWarmingUp {
		t.Error("expected a warming_up progress event")
	}
	if !sawGenerating {
		t.Error("expected a generating progress event once the first token arrived")
	}
	if ttftMs < 5000 {
		t.Errorf("expected TTFT >= 5000ms (server slept 5.3s), got %dms", ttftMs)
	}
}

// TestOpenAI_LoadTimeout verifies that a model which never produces a first
// token is aborted at the dedicated loading timeout (not the much longer
// overall run timeout), with an error wrapping ErrModelLoadTimeout.
func TestOpenAI_LoadTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)

		if stream, _ := req["stream"].(bool); !stream {
			promptTokens := 10
			if _, hasTools := req["tools"]; hasTools {
				promptTokens = 50
			}
			_ = json.NewEncoder(w).Encode(openAIChatCompletionResponse{
				Usage: openAIUsage{PromptTokens: promptTokens},
			})
			return
		}

		// Never respond with a token; hang until the client gives up.
		<-r.Context().Done()
	}))
	defer ts.Close()

	driver := &OpenAICompatibleDriver{
		Providers: []config.Provider{
			{Name: "local-test", BaseURL: ts.URL, Driver: "openai-compatible"},
		},
		HTTPClient: ts.Client(),
	}

	run := Run{
		RunID:               "run-load-timeout",
		AgentName:           "load-timeout-agent",
		Driver:              "openai-compatible",
		ProviderName:        "local-test",
		Model:               "gemma-4-26b",
		PromptText:          "System prompt\n\n---\n\nUser prompt",
		TimeoutMinutes:      10, // overall run timeout — must not be what fires here
		ModelLoadingTimeout: 200 * time.Millisecond,
	}

	start := time.Now()
	proc, err := driver.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	for range proc.Progress() {
	}
	waitErr := proc.Wait()
	elapsed := time.Since(start)

	if waitErr == nil {
		t.Fatal("expected a load-timeout error, got nil")
	}
	if !errors.Is(waitErr, ErrModelLoadTimeout) {
		t.Fatalf("expected error to wrap ErrModelLoadTimeout, got: %v", waitErr)
	}
	if elapsed > 5*time.Second {
		t.Errorf("expected the dedicated 200ms load timeout to fire quickly, took %s", elapsed)
	}
	if !strings.Contains(waitErr.Error(), "loading timeout") {
		t.Errorf("expected error message to mention the loading timeout, got: %v", waitErr)
	}
}

// TestOpenAI_ReasoningContentPreventsLoadTimeout verifies that a model
// streaming only delta.reasoning_content — never delta.content — past the
// load-timeout deadline is not killed by the watchdog (regression test for
// openai-driver-ttft-ignores-reasoning-content, fix guidance item 5): any
// sign of generation, not just delta.content, must disarm it.
func TestOpenAI_ReasoningContentPreventsLoadTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)

		if stream, _ := req["stream"].(bool); !stream {
			promptTokens := 10
			if _, hasTools := req["tools"]; hasTools {
				promptTokens = 50
			}
			_ = json.NewEncoder(w).Encode(openAIChatCompletionResponse{
				Usage: openAIUsage{PromptTokens: promptTokens},
			})
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		// Stream only reasoning_content, spanning well past the dedicated
		// load timeout below, before ever emitting delta.content.
		fmt.Fprint(w, "data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{\"reasoning_content\":\"let me think...\"},\"finish_reason\":null}]}\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		time.Sleep(400 * time.Millisecond)
		fmt.Fprint(w, "data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{\"reasoning_content\":\" ...done thinking\"},\"finish_reason\":null}]}\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		fmt.Fprint(w, "data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"the answer\"},\"finish_reason\":null}]}\n\n")
		fmt.Fprint(w, "data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer ts.Close()

	driver := &OpenAICompatibleDriver{
		Providers: []config.Provider{
			{Name: "local-test", BaseURL: ts.URL, Driver: "openai-compatible"},
		},
		HTTPClient: ts.Client(),
	}

	run := Run{
		RunID:               "run-reasoning",
		AgentName:           "reasoning-agent",
		Driver:              "openai-compatible",
		ProviderName:        "local-test",
		Model:               "gemma-4-26b",
		PromptText:          "System prompt\n\n---\n\nUser prompt",
		TimeoutMinutes:      10,
		ModelLoadingTimeout: 200 * time.Millisecond, // shorter than the 400ms reasoning gap above
	}

	proc, err := driver.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	var sawReasoning, sawGenerating bool
	for ev := range proc.Progress() {
		if ev.Event == nil {
			continue
		}
		switch ev.Event["stage"] {
		case "reasoning":
			sawReasoning = true
		case "generating":
			sawGenerating = true
		}
	}

	if err := proc.Wait(); err != nil {
		t.Fatalf("expected the run to complete without being killed by the load-timeout watchdog, got: %v", err)
	}
	if !sawReasoning {
		t.Error("expected a reasoning progress event")
	}
	if !sawGenerating {
		t.Error("expected a generating progress event once delta.content arrived")
	}
}
