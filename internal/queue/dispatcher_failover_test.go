// SPDX-License-Identifier: AGPL-3.0-or-later

package queue

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/hub"
)

// failoverTestHarness wires a Dispatcher whose lookup broadcasts a
// queue.rate_limit event (kind "overloaded") shortly after StartRun, so the
// dispatcher's failover path is exercised deterministically.
type failoverTestHarness struct {
	store              *Store
	appHub             *hub.Hub
	projHub            *hub.Hub
	switchCalls        atomic.Int32
	lastSwitchProvider atomic.Value // string
	startRunCalls      atomic.Int32
	failoverPolicy     FailoverPolicy
	fallbackProvider   string
	fallbackModel      string
	agentHasFallback   bool
	providerHealthy    bool
}

func newFailoverTestHarness(t *testing.T) *failoverTestHarness {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	h := &failoverTestHarness{
		store:            s,
		appHub:           hub.New(),
		projHub:          hub.New(),
		fallbackProvider: "gemini-cloud",
		fallbackModel:    "gemini-2.5-flash",
		agentHasFallback: true,
		providerHealthy:  true,
	}
	return h
}

func (h *failoverTestHarness) lookup(name string) (ProjectAccess, bool) {
	return ProjectAccess{
		StartRun: func(ctx context.Context, agentName, targetPath string) (string, error) {
			// Only the first run (per test process) fails with a transient
			// error; a re-enqueued job (after a successful failover) succeeds
			// instead, so tests observe exactly one failover/pause decision
			// deterministically rather than racing the dispatcher's next tick.
			first := h.startRunCalls.Add(1) == 1
			go func() {
				time.Sleep(10 * time.Millisecond)
				if first {
					h.projHub.Broadcast(hub.Event{
						Type: "queue.rate_limit",
						Payload: map[string]any{
							"run_id":   "test-run",
							"raw_text": "HTTP 529 Overloaded",
							"kind":     "overloaded",
						},
					})
				} else {
					h.projHub.Broadcast(hub.Event{
						Type:    "agent.finished",
						Payload: map[string]any{"run_id": "test-run", "status": "done"},
					})
				}
			}()
			return "test-run", nil
		},
		ArtifactStatus: func(relPath string) string { return "approved" },
		Hub:            h.projHub,
		FailoverPolicy: func() FailoverPolicy { return h.failoverPolicy },
		AgentFailoverInfo: func(agentName string) (AgentFailoverInfo, bool) {
			if !h.agentHasFallback {
				return AgentFailoverInfo{}, false
			}
			return AgentFailoverInfo{FallbackProvider: h.fallbackProvider, FallbackModel: h.fallbackModel}, true
		},
		ProbeProviderHealth: func(ctx context.Context, providerName string) bool {
			return h.providerHealthy
		},
		SwitchAgentProvider: func(agentName, provider, model, reason string, isFailover bool) error {
			h.switchCalls.Add(1)
			h.lastSwitchProvider.Store(provider)
			return nil
		},
	}, true
}

func (h *failoverTestHarness) dispatcher(now time.Time) *Dispatcher {
	cfg := Config{
		TickInterval:  20 * time.Millisecond,
		ClockFn:       func() time.Time { return now },
		MaxAttempts:   5,
		ResumeGrace:   time.Minute,
		FallbackPause: 30 * time.Minute,
		OverloadPause: 5 * time.Minute,
	}
	return New(h.store, h.lookup, h.appHub, cfg)
}

func (h *failoverTestHarness) enqueue(t *testing.T, attempts int) {
	t.Helper()
	if err := h.store.Enqueue(Job{
		Project:      "proj",
		ArtifactPath: "lifecycle/ideas/a.md",
		AgentName:    "analyst",
		EnqueuedBy:   "alice@example.com",
		Attempts:     attempts,
	}); err != nil {
		t.Fatal(err)
	}
}

// waitForFailedAndPending polls until at least one failed job and one
// pending job exist, or the deadline passes.
func waitForFailedAndPending(t *testing.T, s *Store) (failed, pending []*Job) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		failed, _ = s.ListByState(StateFailed)
		pending, _ = s.ListByState(StatePending)
		if len(failed) > 0 && len(pending) > 0 {
			return failed, pending
		}
		time.Sleep(20 * time.Millisecond)
	}
	return failed, pending
}

// TestDispatcher_AutoSwitchWithHealthyFallback verifies that a 529/overloaded
// failure with auto_switch enabled and a healthy fallback provider triggers
// SwitchAgentProvider, re-enqueues the job at the head without pausing the
// queue, and does not emit queue.paused.
func TestDispatcher_AutoSwitchWithHealthyFallback(t *testing.T) {
	h := newFailoverTestHarness(t)
	h.failoverPolicy = FailoverPolicy{
		Enabled:            true,
		AutoSwitch:         true,
		SwitchOnKinds:      []string{"overloaded", "rate_limit", "unreachable"},
		MaxFailoversPerRun: 1,
	}
	d := h.dispatcher(time.Now())

	events, stopCollect := collectAppEvents(t, h.appHub)
	defer stopCollect()

	h.enqueue(t, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	d.Start(ctx)

	// The re-enqueued job succeeds immediately (see lookup's "first" branch),
	// so it may already have left the pending state by the time we observe
	// it — poll across pending/running/completed rather than pending alone.
	var failed, requeued []*Job
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		failed, _ = h.store.ListByState(StateFailed)
		inFlight, _ := h.store.ListByState(StatePending, StateRunning)
		done, _ := h.store.ListByState(StateCompleted)
		requeued = append(inFlight, done...)
		if len(failed) > 0 && len(requeued) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(failed) == 0 || len(requeued) == 0 {
		t.Fatalf("expected a failed job and a re-queued job; failed=%d requeued=%d", len(failed), len(requeued))
	}
	if failed[0].Reason != "failover_triggered" {
		t.Errorf("failed job reason: got %q, want %q", failed[0].Reason, "failover_triggered")
	}
	if requeued[0].Attempts != 2 {
		t.Errorf("re-queued attempts: got %d, want 2", requeued[0].Attempts)
	}
	if requeued[0].Position >= failed[0].Position {
		t.Errorf("re-queued position %d should be less than failed position %d", requeued[0].Position, failed[0].Position)
	}

	if h.switchCalls.Load() != 1 {
		t.Errorf("expected SwitchAgentProvider called once, got %d", h.switchCalls.Load())
	}
	if v, _ := h.lastSwitchProvider.Load().(string); v != "gemini-cloud" {
		t.Errorf("expected switch to gemini-cloud, got %q", v)
	}

	paused, _, _, err := h.store.GetPauseState()
	if err != nil {
		t.Fatal(err)
	}
	if paused {
		t.Error("expected queue NOT to be paused after a successful failover")
	}
	stopCollect()
	if containsEvent(events, "queue.paused") {
		t.Errorf("expected no queue.paused broadcast after failover; got %v", events())
	}
}

// TestDispatcher_AutoSwitchDisabled_FallsBackToPause verifies that with
// auto_switch: false the standard rate-limit pause path runs unchanged and
// SwitchAgentProvider is never called.
func TestDispatcher_AutoSwitchDisabled_FallsBackToPause(t *testing.T) {
	h := newFailoverTestHarness(t)
	h.failoverPolicy = FailoverPolicy{
		Enabled:            true,
		AutoSwitch:         false,
		SwitchOnKinds:      []string{"overloaded", "rate_limit", "unreachable"},
		MaxFailoversPerRun: 1,
	}
	d := h.dispatcher(time.Now())

	events, stopCollect := collectAppEvents(t, h.appHub)
	defer stopCollect()

	h.enqueue(t, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	d.Start(ctx)

	failed, pending := waitForFailedAndPending(t, h.store)
	if len(failed) == 0 || len(pending) == 0 {
		t.Fatalf("expected a failed job and a re-queued job; failed=%d pending=%d", len(failed), len(pending))
	}
	if failed[0].Reason != "rate_limit" {
		t.Errorf("failed job reason: got %q, want %q", failed[0].Reason, "rate_limit")
	}

	if h.switchCalls.Load() != 0 {
		t.Errorf("expected SwitchAgentProvider never called, got %d", h.switchCalls.Load())
	}

	paused, _, _, err := h.store.GetPauseState()
	if err != nil {
		t.Fatal(err)
	}
	if !paused {
		t.Error("expected queue to be paused when auto_switch is disabled")
	}
	stopCollect()
	if !containsEvent(events, "queue.paused") {
		t.Errorf("expected queue.paused broadcast; got %v", events())
	}
}

// TestDispatcher_UnhealthyFallback_FallsBackToPause verifies that a failed
// pre-switch health probe on the fallback provider prevents failover and
// falls back to the standard pause, without calling SwitchAgentProvider.
func TestDispatcher_UnhealthyFallback_FallsBackToPause(t *testing.T) {
	h := newFailoverTestHarness(t)
	h.failoverPolicy = FailoverPolicy{
		Enabled:            true,
		AutoSwitch:         true,
		SwitchOnKinds:      []string{"overloaded", "rate_limit", "unreachable"},
		MaxFailoversPerRun: 1,
	}
	h.providerHealthy = false
	d := h.dispatcher(time.Now())

	events, stopCollect := collectAppEvents(t, h.appHub)
	defer stopCollect()

	h.enqueue(t, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	d.Start(ctx)

	failed, _ := waitForFailedAndPending(t, h.store)
	if len(failed) == 0 {
		t.Fatal("expected a failed job")
	}
	if failed[0].Reason != "rate_limit" {
		t.Errorf("failed job reason: got %q, want %q (unhealthy fallback should not trigger failover)", failed[0].Reason, "rate_limit")
	}
	if h.switchCalls.Load() != 0 {
		t.Errorf("expected SwitchAgentProvider never called when fallback is unhealthy, got %d", h.switchCalls.Load())
	}

	stopCollect()
	if !containsEvent(events, "queue.paused") {
		t.Errorf("expected queue.paused broadcast; got %v", events())
	}
}

// TestDispatcher_MaxFailoversPerRunExceeded verifies that once a job's
// Attempts already exceeds max_failovers_per_run, a further rate-limit
// failure does not trigger another failover and instead falls back to the
// standard pause — cascading failover is bounded.
func TestDispatcher_MaxFailoversPerRunExceeded(t *testing.T) {
	h := newFailoverTestHarness(t)
	h.failoverPolicy = FailoverPolicy{
		Enabled:            true,
		AutoSwitch:         true,
		SwitchOnKinds:      []string{"overloaded", "rate_limit", "unreachable"},
		MaxFailoversPerRun: 1,
	}
	d := h.dispatcher(time.Now())

	events, stopCollect := collectAppEvents(t, h.appHub)
	defer stopCollect()

	// Attempts=2 already exceeds MaxFailoversPerRun=1 (job.Attempts > policy.MaxFailoversPerRun).
	h.enqueue(t, 2)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	d.Start(ctx)

	failed, _ := waitForFailedAndPending(t, h.store)
	if len(failed) == 0 {
		t.Fatal("expected a failed job")
	}
	if failed[0].Reason != "rate_limit" {
		t.Errorf("failed job reason: got %q, want %q (max_failovers_per_run exceeded should not trigger failover)", failed[0].Reason, "rate_limit")
	}
	if h.switchCalls.Load() != 0 {
		t.Errorf("expected SwitchAgentProvider never called once max_failovers_per_run is exceeded, got %d", h.switchCalls.Load())
	}

	stopCollect()
	if !containsEvent(events, "queue.paused") {
		t.Errorf("expected queue.paused broadcast; got %v", events())
	}
}
