// SPDX-License-Identifier: AGPL-3.0-or-later

package project

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
)

// newTestProjectForRecoveryProber builds a minimal in-memory Project
// (real SQLite index + hub, no git, no on-disk config.yaml) sufficient for
// exercising RecoveryProber.probe, which reads p.Config(), p.Providers,
// p.Hub, p.Idx, and p.Operations().
func newTestProjectForRecoveryProber(t *testing.T, cfg *config.Project, providers []config.Provider) *Project {
	t.Helper()
	dir := t.TempDir()
	h := hub.New()
	idx, err := index.Open(filepath.Join(dir, "recovery-prober-test.db"), dir, nil, index.WithHub(h))
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { _ = idx.Close() })

	p := &Project{
		Entry:      &config.ProjectEntry{Name: "recovery-test", Path: dir},
		Cfg:        cfg,
		Idx:        idx,
		Hub:        h,
		Providers:  providers,
		operations: &Operations{path: operationsPath(dir)},
	}
	p.RecoveryProber = newRecoveryProber(p)
	return p
}

// collectHubEvents subscribes to h and returns an accessor for decoded
// {type, payload} events observed so far, plus a stop func.
func collectHubEvents(t *testing.T, h *hub.Hub) (events func() []map[string]any, stop func()) {
	t.Helper()
	ch := make(chan []byte, 64)
	h.Register(ch)
	var collected []map[string]any
	done := make(chan struct{})
	var closeOnce bool
	go func() {
		for data := range ch {
			var evt map[string]any
			if err := json.Unmarshal(data, &evt); err == nil {
				collected = append(collected, evt)
			}
		}
		close(done)
	}()
	return func() []map[string]any {
			out := make([]map[string]any, len(collected))
			copy(out, collected)
			return out
		}, func() {
			if !closeOnce {
				closeOnce = true
				h.Unregister(ch)
				close(ch)
				<-done
			}
		}
}

func hasHubEventType(events []map[string]any, typ string) bool {
	for _, e := range events {
		if t, _ := e["type"].(string); t == typ {
			return true
		}
	}
	return false
}

// TestRecoveryProber_IdleWithNoProviderBoundAgents verifies the prober does
// nothing (no tracked providers) only when no agent references any
// provider at all — not merely when no agent is in failover (FR-5.1: a
// single-provider project must still be probed).
func TestRecoveryProber_IdleWithNoProviderBoundAgents(t *testing.T) {
	cfg := &config.Project{Agents: []config.AgentConfig{{Name: "a", Driver: "claude-code-cli"}}}
	p := newTestProjectForRecoveryProber(t, cfg, nil)

	events, stop := collectHubEvents(t, p.Hub)
	defer stop()

	p.RecoveryProber.probe(context.Background())

	stop()
	if len(events()) != 0 {
		t.Errorf("expected no events while idle, got %v", events())
	}
	if len(p.RecoveryProber.consecutiveSuccess) != 0 {
		t.Errorf("expected no tracked providers while idle, got %v", p.RecoveryProber.consecutiveSuccess)
	}
}

// TestRecoveryProber_SingleProviderMode_ProbesAndRecordsReachability
// verifies FR-5.1: an agent with no secondary configured (single-provider
// mode) is still probed, and its reachability is written to
// operations.yaml — previously the prober only ever built its target list
// from agents already in failover and was permanently idle in this mode.
func TestRecoveryProber_SingleProviderMode_ProbesAndRecordsReachability(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := &config.Project{
		Agents: []config.AgentConfig{
			{Name: "a", Provider: "anthropic-cloud", Model: "m"}, // no fallback_provider — single-provider mode
		},
	}
	providers := []config.Provider{{Name: "anthropic-cloud", BaseURL: ts.URL, Driver: "openai-compatible"}}
	p := newTestProjectForRecoveryProber(t, cfg, providers)

	events, stop := collectHubEvents(t, p.Hub)
	defer stop()

	p.RecoveryProber.probe(context.Background())
	p.RecoveryProber.probe(context.Background())

	reach, ok := p.Operations().GetReachability("anthropic-cloud")
	if !ok || !reach.Healthy {
		t.Fatalf("expected anthropic-cloud reachability recorded as healthy, got %+v ok=%v", reach, ok)
	}

	// No agent is in failover, so there is nothing to "recover" — the event
	// must not fire even though the provider is healthy.
	stop()
	if hasHubEventType(events(), "provider.primary_recovered") {
		t.Errorf("expected no provider.primary_recovered when no agent is in failover, got %v", events())
	}
}

func TestRecoveryProber_TwoConsecutiveHealthyProbesTriggerRecovery(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := &config.Project{
		Agents: []config.AgentConfig{
			{Name: "a", Provider: "anthropic-cloud", Model: "claude", FallbackProvider: "gemini-cloud", FallbackModel: "m"},
		},
	}
	providers := []config.Provider{{Name: "anthropic-cloud", BaseURL: ts.URL, Driver: "openai-compatible"}}
	p := newTestProjectForRecoveryProber(t, cfg, providers)

	// Seed operations state as if the agent already failed over to gemini-cloud.
	if err := p.Operations().SetAgentState(AgentOperationalState{
		Agent:   "a",
		Primary: ProviderModel{Provider: "anthropic-cloud", Model: "claude"},
		Active:  ProviderModel{Provider: "gemini-cloud", Model: "m"},
	}); err != nil {
		t.Fatal(err)
	}

	events, stop := collectHubEvents(t, p.Hub)
	defer stop()

	ctx := context.Background()
	p.RecoveryProber.probe(ctx) // 1st healthy probe: count=1, no event yet
	if hasHubEventType(events(), "provider.primary_recovered") {
		t.Fatal("did not expect provider.primary_recovered after only 1 healthy probe")
	}
	p.RecoveryProber.probe(ctx) // 2nd healthy probe: count=2, triggers recovery

	// Give the async hub subscriber goroutine a moment to receive the broadcast.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !hasHubEventType(events(), "provider.primary_recovered") {
		time.Sleep(10 * time.Millisecond)
	}
	stop()

	got := events()
	if !hasHubEventType(got, "provider.primary_recovered") {
		t.Fatalf("expected provider.primary_recovered after 2 consecutive healthy probes, got %v", got)
	}

	// A feed record was also inserted.
	feedEvents, err := p.Idx.ListEvents(10, 0, []string{"primary_recovered"})
	if err != nil {
		t.Fatal(err)
	}
	if len(feedEvents) != 1 {
		t.Fatalf("expected 1 primary_recovered feed event, got %d", len(feedEvents))
	}
}

// TestRecoveryProber_QuotaGated_SuppressesRecoveryBeforeResetTime verifies
// FR-9.3: a rate-limit failover's recovery announcement is suppressed until
// the recorded resets_at_unix has passed, even with healthy probes.
func TestRecoveryProber_QuotaGated_SuppressesRecoveryBeforeResetTime(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := &config.Project{
		Agents: []config.AgentConfig{
			{Name: "a", Provider: "anthropic-cloud", Model: "claude", FallbackProvider: "gemini-cloud", FallbackModel: "m"},
		},
	}
	providers := []config.Provider{{Name: "anthropic-cloud", BaseURL: ts.URL, Driver: "openai-compatible"}}
	p := newTestProjectForRecoveryProber(t, cfg, providers)

	if err := p.Operations().SetAgentState(AgentOperationalState{
		Agent:        "a",
		Primary:      ProviderModel{Provider: "anthropic-cloud", Model: "claude"},
		Active:       ProviderModel{Provider: "gemini-cloud", Model: "m"},
		ResetsAtUnix: time.Now().Add(1 * time.Hour).Unix(), // reset is still an hour away
	}); err != nil {
		t.Fatal(err)
	}

	events, stop := collectHubEvents(t, p.Hub)
	defer stop()

	ctx := context.Background()
	p.RecoveryProber.probe(ctx)
	p.RecoveryProber.probe(ctx) // 2 consecutive healthy probes — would fire without the quota gate

	time.Sleep(100 * time.Millisecond)
	stop()
	if hasHubEventType(events(), "provider.primary_recovered") {
		t.Errorf("expected recovery suppressed before the recorded reset time, got %v", events())
	}
}

// TestRecoveryProber_QuotaGate_ClearsAfterResetTime verifies that once the
// recorded reset time has passed, the (already-healthy) provider announces
// recovery on its next probe cycle.
func TestRecoveryProber_QuotaGate_ClearsAfterResetTime(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := &config.Project{
		Agents: []config.AgentConfig{
			{Name: "a", Provider: "anthropic-cloud", Model: "claude", FallbackProvider: "gemini-cloud", FallbackModel: "m"},
		},
	}
	providers := []config.Provider{{Name: "anthropic-cloud", BaseURL: ts.URL, Driver: "openai-compatible"}}
	p := newTestProjectForRecoveryProber(t, cfg, providers)

	if err := p.Operations().SetAgentState(AgentOperationalState{
		Agent:        "a",
		Primary:      ProviderModel{Provider: "anthropic-cloud", Model: "claude"},
		Active:       ProviderModel{Provider: "gemini-cloud", Model: "m"},
		ResetsAtUnix: time.Now().Add(-1 * time.Minute).Unix(), // reset already passed
	}); err != nil {
		t.Fatal(err)
	}

	events, stop := collectHubEvents(t, p.Hub)
	defer stop()

	ctx := context.Background()
	p.RecoveryProber.probe(ctx)
	p.RecoveryProber.probe(ctx)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !hasHubEventType(events(), "provider.primary_recovered") {
		time.Sleep(10 * time.Millisecond)
	}
	stop()
	if !hasHubEventType(events(), "provider.primary_recovered") {
		t.Errorf("expected recovery announced once the reset time has passed, got %v", events())
	}
}

func TestRecoveryProber_TransientFailureResetsCounter(t *testing.T) {
	healthy := true
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if healthy {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer ts.Close()

	cfg := &config.Project{
		Agents: []config.AgentConfig{
			{Name: "a", Provider: "anthropic-cloud", Model: "claude", FallbackProvider: "gemini-cloud", FallbackModel: "m"},
		},
	}
	providers := []config.Provider{{Name: "anthropic-cloud", BaseURL: ts.URL, Driver: "openai-compatible"}}
	p := newTestProjectForRecoveryProber(t, cfg, providers)

	ctx := context.Background()
	p.RecoveryProber.probe(ctx) // healthy: count=1
	if p.RecoveryProber.consecutiveSuccess["anthropic-cloud"] != 1 {
		t.Fatalf("expected count=1, got %d", p.RecoveryProber.consecutiveSuccess["anthropic-cloud"])
	}

	healthy = false
	p.RecoveryProber.probe(ctx) // unhealthy: resets to 0
	if got := p.RecoveryProber.consecutiveSuccess["anthropic-cloud"]; got != 0 {
		t.Fatalf("expected count reset to 0 after failure, got %d", got)
	}

	healthy = true
	p.RecoveryProber.probe(ctx) // healthy again: count=1, not yet recovered
	if got := p.RecoveryProber.consecutiveSuccess["anthropic-cloud"]; got != 1 {
		t.Fatalf("expected count=1 after one healthy probe post-reset, got %d", got)
	}

	reach, ok := p.Operations().GetReachability("anthropic-cloud")
	if !ok || !reach.Healthy {
		t.Errorf("expected the latest (healthy) probe result recorded, got %+v ok=%v", reach, ok)
	}
}

func TestRecoveryProber_ExitsCleanlyOnContextCancellation(t *testing.T) {
	cfg := &config.Project{}
	p := newTestProjectForRecoveryProber(t, cfg, nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := p.RecoveryProber.Start(ctx)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("expected prober goroutine to exit promptly after context cancellation")
	}
}

func TestRecoveryProber_StopsTrackingProviderNoLongerBound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := &config.Project{
		Agents: []config.AgentConfig{
			{Name: "a", Provider: "anthropic-cloud", Model: "claude", FallbackProvider: "gemini-cloud", FallbackModel: "m"},
		},
	}
	providers := []config.Provider{{Name: "anthropic-cloud", BaseURL: ts.URL, Driver: "openai-compatible"}}
	p := newTestProjectForRecoveryProber(t, cfg, providers)

	ctx := context.Background()
	p.RecoveryProber.probe(ctx)
	if _, tracked := p.RecoveryProber.consecutiveSuccess["anthropic-cloud"]; !tracked {
		t.Fatal("expected anthropic-cloud to be tracked while an agent references it")
	}

	// Simulate the agent being reconfigured onto a different provider entirely.
	p.SetConfig(&config.Project{Agents: []config.AgentConfig{{Name: "a", Provider: "gemini-cloud", Model: "m"}}})
	p.RecoveryProber.probe(ctx)
	if _, tracked := p.RecoveryProber.consecutiveSuccess["anthropic-cloud"]; tracked {
		t.Error("expected anthropic-cloud tracking to be cleared once no agent references it")
	}
}
