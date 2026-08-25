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
// exercising RecoveryProber.probe, which only reads p.Config(), p.Providers,
// p.Hub, and p.Idx.
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
		Entry:     &config.ProjectEntry{Name: "recovery-test", Path: dir},
		Cfg:       cfg,
		Idx:       idx,
		Hub:       h,
		Providers: providers,
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

func TestRecoveryProber_IdleWhenNoAgentInFailover(t *testing.T) {
	cfg := &config.Project{
		Agents: []config.AgentConfig{
			{Name: "a", Provider: "anthropic-cloud", Model: "m"}, // PrimaryProvider empty — not in failover
		},
	}
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

func TestRecoveryProber_TwoConsecutiveHealthyProbesTriggerRecovery(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := &config.Project{
		Agents: []config.AgentConfig{
			{Name: "a", Provider: "gemini-cloud", Model: "m", PrimaryProvider: "anthropic-cloud", PrimaryModel: "claude"},
		},
	}
	providers := []config.Provider{{Name: "anthropic-cloud", BaseURL: ts.URL, Driver: "openai-compatible"}}
	p := newTestProjectForRecoveryProber(t, cfg, providers)

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
			{Name: "a", Provider: "gemini-cloud", Model: "m", PrimaryProvider: "anthropic-cloud", PrimaryModel: "claude"},
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

func TestRecoveryProber_StopsTrackingProviderNoLongerInFailover(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := &config.Project{
		Agents: []config.AgentConfig{
			{Name: "a", Provider: "gemini-cloud", Model: "m", PrimaryProvider: "anthropic-cloud", PrimaryModel: "claude"},
		},
	}
	providers := []config.Provider{{Name: "anthropic-cloud", BaseURL: ts.URL, Driver: "openai-compatible"}}
	p := newTestProjectForRecoveryProber(t, cfg, providers)

	ctx := context.Background()
	p.RecoveryProber.probe(ctx)
	if _, tracked := p.RecoveryProber.consecutiveSuccess["anthropic-cloud"]; !tracked {
		t.Fatal("expected anthropic-cloud to be tracked while agent is in failover")
	}

	// Simulate the agent having been restored: no more agents point to it as primary.
	p.SetConfig(&config.Project{Agents: []config.AgentConfig{{Name: "a", Provider: "anthropic-cloud", Model: "m"}}})
	p.RecoveryProber.probe(ctx)
	if _, tracked := p.RecoveryProber.consecutiveSuccess["anthropic-cloud"]; tracked {
		t.Error("expected anthropic-cloud tracking to be cleared once no agent is in failover on it")
	}
}
