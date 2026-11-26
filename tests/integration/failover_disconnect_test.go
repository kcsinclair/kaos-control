// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Suite — Milestone 5: provider_disconnected retry-in-place, backoff, and
// the rolling-hour pause threshold (FR-6).
//
// The driver's own in-loop backoff retry (2s/8s/30s,
// internal/agent's providerDisconnectBackoff) is exercised with an
// injectable clock at the unit level (internal/agent/openai_compatible_test.go);
// this integration test drives one real, full-speed retry-exhaustion cycle
// end to end to prove the dispatcher's rolling-hour pause threshold
// (FR-6.3) and disconnect-counter durability (FR-6.4) work against the real
// operations.yaml store and the real driver — it accepts the ~40s
// wall-clock cost of the real backoff schedule as the price of that
// fidelity.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/project"
)

const failoverDisconnectCfgYAML = `git:
  default_branch: main
  branch_template: "ticket/{slug}"

roles:
  - product-owner
  - analyst

stages:
  - {name: ideas,          dir: ideas}
  - {name: requirements,   dir: requirements}
  - {name: backend-plans,  dir: backend-plans}
  - {name: frontend-plans, dir: frontend-plans}
  - {name: test-plans,     dir: test-plans}
  - {name: tests,          dir: tests}
  - {name: prototypes,     dir: prototypes}
  - {name: releases,       dir: releases}
  - {name: sprints,        dir: sprints}
  - {name: defects,        dir: defects}

users:
  - email: admin@test.local
    roles: [product-owner]

required_plans:
  ticket: []
  epic: []

agents:
  - name: disc-agent
    role: [analyst]
    driver: openai-compatible
    provider: flaky-provider
    model: test-model
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: Disconnect Agent
      email: disc-agent@test.local
    prompt_templates:
      analyst: "x"
`

// alwaysDisconnectServer answers GET /v1/models and a non-streaming
// completions preflight normally, but severs the connection (panics with
// http.ErrAbortHandler, recovered by net/http, so the client observes a
// genuine read error rather than a clean EOF) on every streaming
// completions request — the same fault openai_compatible_test.go's
// midStreamDisconnectHandler injects at the unit level, reimplemented here
// since that helper is unexported in package agent.
func alwaysDisconnectServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"test-model","object":"model","supported_parameters":["tools"]}]}`))
	})
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqMap map[string]any
		_ = json.Unmarshal(body, &reqMap)
		isStream, _ := reqMap["stream"].(bool)
		if !isStream {
			// Mirror testutil.MockOpenAIServer's mode-c preflight signal: a
			// token-count delta between a request with "tools" and one
			// without proves the server actually honours the tools
			// parameter (rather than silently dropping it).
			hasTools := false
			if tools, ok := reqMap["tools"].([]any); ok && len(tools) > 0 {
				hasTools = true
			}
			promptTokens := 5
			if hasTools {
				promptTokens = 25
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "chatcmpl-preflight",
				"object":  "chat.completion",
				"created": time.Now().Unix(),
				"choices": []any{
					map[string]any{"index": 0, "message": map[string]any{"role": "assistant", "content": "preflight-ok"}, "finish_reason": "stop"},
				},
				"usage": map[string]any{"prompt_tokens": promptTokens, "completion_tokens": 1, "total_tokens": promptTokens + 1},
			})
			return
		}
		// Send headers and a partial SSE chunk before severing the
		// connection — a disconnect after some bytes have already arrived
		// classifies as provider_disconnected (FR-6); aborting before any
		// response at all classifies as endpoint_unreachable instead, which
		// is a different (non-retrying) path.
		fl, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected http.Flusher")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n")
		fl.Flush()
		panic(http.ErrAbortHandler)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// TestFailover_ProviderDisconnect_ThresholdPausesQueue (Milestone 5, FR-6.3/
// FR-6.4): with three prior disconnect occurrences already recorded for a
// provider within the rolling hour, a fourth real disconnect — driven end
// to end through the openai-compatible driver's own backoff-and-exhaust
// cycle — must pause the queue rather than retrying in place again. The
// disconnect counter is durable: reloading operations.yaml from disk (a
// simulated restart) shows the same count.
func TestFailover_ProviderDisconnect_ThresholdPausesQueue(t *testing.T) {
	flaky := alwaysDisconnectServer(t)
	providers := []config.Provider{
		{Name: "flaky-provider", BaseURL: flaky.URL, Driver: "openai-compatible"},
	}

	env := newFailoverTestEnv(t, failoverDisconnectCfgYAML, providers, []seedArtifact{
		{relPath: "lifecycle/ideas/disc-idea.md", content: makeApprovedArtifact("Disconnect Idea", "idea", "disc-idea")},
	})

	// Seed 3 prior, safely-distinct disconnect occurrences for the provider
	// within the last hour (spaced well over a minute apart — comfortably
	// wider than the driver's own backoff-window collapse, Resolved
	// Question 1 — so each counts on its own).
	now := time.Now()
	for i, ago := range []time.Duration{6 * time.Minute, 4*time.Minute + 30*time.Second, 3 * time.Minute} {
		if _, err := env.proj.Operations().RecordDisconnect("flaky-provider", now.Add(-ago), 1*time.Second); err != nil {
			t.Fatalf("seeding disconnect %d: %v", i, err)
		}
	}
	if got := env.proj.Operations().DisconnectCountSince("flaky-provider", now.Add(-1*time.Hour)); got != 3 {
		t.Fatalf("expected 3 seeded disconnects before the live run, got %d", got)
	}

	env.enqueue("lifecycle/ideas/disc-idea.md", "disc-agent")

	// The driver retries in place through the full 2s/8s/30s backoff
	// schedule before giving up — this genuinely takes about 40s.
	env.waitFor(75*time.Second, "queue to pause after the 4th distinct disconnect", func() bool {
		snap := env.queueSnapshot()
		paused, _ := snap["paused"].(bool)
		return paused
	})

	count := env.proj.Operations().DisconnectCountSince("flaky-provider", now.Add(-1*time.Hour))
	if count < 4 {
		t.Errorf("expected at least 4 recorded disconnects after the live run, got %d", count)
	}

	// Durability (FR-6.4): reload operations.yaml from disk (a simulated
	// restart) and confirm the counter survives intact.
	reloaded, err := project.LoadOperations(env.projectRoot)
	if err != nil {
		t.Fatalf("reloading operations.yaml: %v", err)
	}
	reloadedCount := reloaded.DisconnectCountSince("flaky-provider", now.Add(-1*time.Hour))
	if reloadedCount != count {
		t.Errorf("disconnect count after reload: got %d, want %d (durability across restart)", reloadedCount, count)
	}
}
