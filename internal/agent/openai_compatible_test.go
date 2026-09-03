// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
)

func TestOpenAICompatibleDriver_SingleTurnToolCall(t *testing.T) {
	root := t.TempDir()
	testFilePath := filepath.Join(root, "lifecycle", "ideas", "idea-1.md")
	if err := os.MkdirAll(filepath.Dir(testFilePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testFilePath, []byte("# Originating Idea Content"), 0o644); err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(root, "run.log")

	var turnCount int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)

		// Preflight requests (stream: false)
		if stream, _ := req["stream"].(bool); !stream {
			promptTokens := 10
			if _, hasTools := req["tools"]; hasTools {
				promptTokens = 50
			}
			resp := openAIChatCompletionResponse{
				Usage: openAIUsage{PromptTokens: promptTokens},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Streaming completions
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		turn := atomic.AddInt32(&turnCount, 1)
		if turn == 1 {
			// Model calls read_file
			chunk1 := `{"choices":[{"index":0,"delta":{"role":"assistant","tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"lifecycle/ideas/idea-1.md\"}"}}]},"finish_reason":null}]}`
			fmt.Fprintf(w, "data: %s\n\n", chunk1)
			flusher.Flush()

			chunk2 := `{"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`
			fmt.Fprintf(w, "data: %s\n\n", chunk2)
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
		} else {
			// Model returns final answer
			chunk1 := `{"choices":[{"index":0,"delta":{"role":"assistant","content":"I read the idea successfully."},"finish_reason":null}]}`
			fmt.Fprintf(w, "data: %s\n\n", chunk1)
			flusher.Flush()

			chunk2 := `{"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`
			fmt.Fprintf(w, "data: %s\n\n", chunk2)
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
		}
	}))
	defer ts.Close()

	driver := &OpenAICompatibleDriver{
		Providers: []config.Provider{
			{
				Name:    "local-test",
				BaseURL: ts.URL,
				Driver:  "openai-compatible",
				APIKey:  "secret-test-token-12345",
			},
		},
		HTTPClient: ts.Client(),
	}

	var ttftMs int64
	run := Run{
		RunID:        "run-101",
		AgentName:    "test-analyst",
		Role:         "analyst",
		Driver:       "openai-compatible",
		ProviderName: "local-test",
		Model:        "gemma-4-26b",
		PromptText:   "---SYSTEM---\nYou are an analyst.\n---USER---\nRead the idea.",
		ProjectRoot:  root,
		LogPath:      logPath,
		OnTTFT: func(ms int64) {
			ttftMs = ms
		},
	}

	proc, err := driver.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("driver.Start error: %v", err)
	}

	for ev := range proc.Progress() {
		_ = ev
	}

	if err := proc.Wait(); err != nil {
		t.Fatalf("proc.Wait unexpected error: %v", err)
	}

	if ttftMs < 0 {
		t.Errorf("expected non-negative TTFT, got: %d", ttftMs)
	}

	// Verify log file contents and secret masking
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}
	logStr := string(logData)
	if strings.Contains(logStr, "secret-test-token-12345") {
		t.Error("secret APIKey found in log file")
	}
	if !strings.Contains(logStr, "# tool result (call_1): # Originating Idea Content") {
		t.Errorf("tool result missing in log: %s", logStr)
	}
	if !strings.Contains(logStr, "I read the idea successfully.") {
		t.Errorf("final response missing in log: %s", logStr)
	}
}

func TestOpenAICompatibleDriver_MultiTurnAndScopeRecovery(t *testing.T) {
	root := t.TempDir()
	execDir := filepath.Join(root, "lifecycle", "requirements")
	if err := os.MkdirAll(execDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var turnCount int32

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

		turn := atomic.AddInt32(&turnCount, 1)
		switch turn {
		case 1:
			// Model attempts an out-of-scope write to /cmd/main.go
			chunk := `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"write_file","arguments":"{\"path\":\"cmd/main.go\",\"content\":\"package main\"}"}}]},"finish_reason":"tool_calls"}]}`
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
		case 2:
			// Model sees error feedback and writes to allowed path instead
			chunk := `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_2","type":"function","function":{"name":"write_file","arguments":"{\"path\":\"lifecycle/requirements/req-1.md\",\"content\":\"# Requirements\"}"}}]},"finish_reason":"tool_calls"}]}`
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
		default:
			// Model finishes
			chunk := `{"choices":[{"index":0,"delta":{"content":"Requirements written."},"finish_reason":"stop"}]}`
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
		}
	}))
	defer ts.Close()

	driver := &OpenAICompatibleDriver{
		Providers: []config.Provider{
			{
				Name:    "local-test",
				BaseURL: ts.URL,
				Driver:  "openai-compatible",
			},
		},
		HTTPClient: ts.Client(),
	}

	run := Run{
		RunID:        "run-102",
		AgentName:    "test-agent",
		Driver:       "openai-compatible",
		ProviderName: "local-test",
		Model:        "gemma-4-26b",
		PromptText:   "Write requirements.",
		ProjectRoot:  root,
		AllowedPaths: []string{"lifecycle/requirements"},
	}

	proc, err := driver.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("driver.Start error: %v", err)
	}

	for range proc.Progress() {
	}

	if err := proc.Wait(); err != nil {
		t.Fatalf("proc.Wait unexpected error: %v", err)
	}

	// Verify cmd/main.go was NOT created
	if _, err := os.Stat(filepath.Join(root, "cmd", "main.go")); !os.IsNotExist(err) {
		t.Error("cmd/main.go should not have been created")
	}

	// Verify allowed file was written
	data, err := os.ReadFile(filepath.Join(root, "lifecycle", "requirements", "req-1.md"))
	if err != nil {
		t.Fatalf("reading allowed target file: %v", err)
	}
	if string(data) != "# Requirements" {
		t.Errorf("target file content mismatch: %q", string(data))
	}
}

func TestOpenAICompatibleDriver_MaxIterationsCap(t *testing.T) {
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

		// Model loops forever calling list_dir
		chunk := `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_loop","type":"function","function":{"name":"list_dir","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}`
		fmt.Fprintf(w, "data: %s\n\n", chunk)
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer ts.Close()

	driver := &OpenAICompatibleDriver{
		Providers: []config.Provider{
			{
				Name:    "local-test",
				BaseURL: ts.URL,
				Driver:  "openai-compatible",
			},
		},
		HTTPClient: ts.Client(),
	}

	run := Run{
		RunID:             "run-103",
		AgentName:         "loop-agent",
		Driver:            "openai-compatible",
		ProviderName:      "local-test",
		Model:             "gemma-4-26b",
		PromptText:        "Loop forever.",
		ProjectRoot:       t.TempDir(),
		MaxToolIterations: 3, // low cap for testing
	}

	proc, err := driver.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("driver.Start error: %v", err)
	}

	for range proc.Progress() {
	}

	err = proc.Wait()
	if err == nil {
		t.Fatal("expected max iteration cap error, got nil")
	}
	if !strings.Contains(err.Error(), "max tool iterations cap (3) reached") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestOpenAICompatibleDriver_KillCancellation(t *testing.T) {
	hangCh := make(chan struct{})
	defer close(hangCh)

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

		// Hang until request is canceled
		<-r.Context().Done()
	}))
	defer ts.Close()

	driver := &OpenAICompatibleDriver{
		Providers: []config.Provider{
			{
				Name:    "local-test",
				BaseURL: ts.URL,
				Driver:  "openai-compatible",
			},
		},
		HTTPClient: ts.Client(),
	}

	run := Run{
		RunID:        "run-104",
		AgentName:    "hang-agent",
		Driver:       "openai-compatible",
		ProviderName: "local-test",
		Model:        "gemma-4-26b",
		PromptText:   "Hang forever.",
		ProjectRoot:  t.TempDir(),
	}

	proc, err := driver.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("driver.Start error: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	if err := proc.Kill(); err != nil {
		t.Fatalf("proc.Kill error: %v", err)
	}

	err = proc.Wait()
	if err == nil {
		t.Fatal("expected canceled error from Kill, got nil")
	}
}

// withFastProviderDisconnectBackoff temporarily shrinks the retry backoff
// schedule (agent-switchover-and-failover Milestone 5) so tests exercising
// the in-loop retry path don't wait through the real 2s/8s/30s schedule.
func withFastProviderDisconnectBackoff(t *testing.T) {
	t.Helper()
	orig := providerDisconnectBackoff
	providerDisconnectBackoff = []time.Duration{5 * time.Millisecond, 5 * time.Millisecond}
	t.Cleanup(func() { providerDisconnectBackoff = orig })
}

// midStreamDisconnectHandler streams one SSE chunk, flushes it, then panics
// to sever the connection mid-response — net/http's per-request panic
// recovery closes the underlying connection without a valid chunked
// terminator, so the client reliably observes a genuine stream read error
// (not a clean EOF) on failCount calls, then completes normally.
func midStreamDisconnectHandler(t *testing.T, failCount int) http.HandlerFunc {
	t.Helper()
	var calls atomic.Int32
	return openAIPreflightOKHandler(t, func(w http.ResponseWriter, r *http.Request) {
		fl, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected http.Flusher")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n")
		fl.Flush()

		if calls.Add(1) <= int32(failCount) {
			panic(http.ErrAbortHandler) // severs the connection; logged, recovered by net/http
		}

		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\" done\"},\"finish_reason\":\"stop\"}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
		fl.Flush()
	})
}

// TestOpenAICompatibleDriver_MidStreamDisconnect_RetriesAndSucceeds verifies
// FR-6.1: a mid-stream disconnect retries the identical request in place,
// inside the turn loop, and a subsequent successful attempt completes the
// run normally — no error is ever reported to the caller.
func TestOpenAICompatibleDriver_MidStreamDisconnect_RetriesAndSucceeds(t *testing.T) {
	withFastProviderDisconnectBackoff(t)

	ts := httptest.NewServer(midStreamDisconnectHandler(t, 1))
	defer ts.Close()

	var disconnects atomic.Int32
	var lastProvider string
	driver := &OpenAICompatibleDriver{
		Providers:  []config.Provider{{Name: "local-test", BaseURL: ts.URL, Driver: "openai-compatible"}},
		HTTPClient: ts.Client(),
	}
	run := Run{
		RunID:        "run-disc-1",
		Driver:       "openai-compatible",
		ProviderName: "local-test",
		Model:        "gemma-4-26b",
		PromptText:   "Do the thing.",
		ProjectRoot:  t.TempDir(),
		OnProviderDisconnect: func(providerName string, at time.Time) {
			disconnects.Add(1)
			lastProvider = providerName
		},
	}

	proc, err := driver.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("driver.Start error: %v", err)
	}
	for range proc.Progress() {
	}
	if err := proc.Wait(); err != nil {
		t.Fatalf("expected the run to succeed after retrying, got: %v", err)
	}
	if disconnects.Load() != 1 {
		t.Errorf("expected exactly 1 recorded disconnect, got %d", disconnects.Load())
	}
	if lastProvider != "local-test" {
		t.Errorf("expected disconnect recorded for local-test, got %q", lastProvider)
	}
}

// TestOpenAICompatibleDriver_MidStreamDisconnect_ExhaustsRetries verifies
// that once the backoff schedule is exhausted, the run fails with the
// stream error (classified as provider_disconnected upstream) rather than
// retrying forever, and every attempt — including the final failed one —
// was recorded via OnProviderDisconnect (FR-6.3: a run's disconnects all
// count toward the rolling-hour threshold, whether or not it eventually
// succeeds).
func TestOpenAICompatibleDriver_MidStreamDisconnect_ExhaustsRetries(t *testing.T) {
	withFastProviderDisconnectBackoff(t)

	ts := httptest.NewServer(midStreamDisconnectHandler(t, 1000)) // always disconnects
	defer ts.Close()

	var disconnects atomic.Int32
	driver := &OpenAICompatibleDriver{
		Providers:  []config.Provider{{Name: "local-test", BaseURL: ts.URL, Driver: "openai-compatible"}},
		HTTPClient: ts.Client(),
	}
	run := Run{
		RunID:        "run-disc-2",
		Driver:       "openai-compatible",
		ProviderName: "local-test",
		Model:        "gemma-4-26b",
		PromptText:   "Do the thing.",
		ProjectRoot:  t.TempDir(),
		OnProviderDisconnect: func(providerName string, at time.Time) {
			disconnects.Add(1)
		},
	}

	proc, err := driver.Start(context.Background(), run)
	if err != nil {
		t.Fatalf("driver.Start error: %v", err)
	}
	for range proc.Progress() {
	}
	if err := proc.Wait(); err == nil {
		t.Fatal("expected the run to fail once the retry budget is exhausted")
	}

	wantAttempts := int32(len(providerDisconnectBackoff) + 1) // initial attempt + one per backoff step
	if disconnects.Load() != wantAttempts {
		t.Errorf("expected %d recorded disconnects (retry budget exhausted), got %d", wantAttempts, disconnects.Load())
	}
}
