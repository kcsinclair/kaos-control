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
// dispatcher's project-wide failover path (Milestone 4) is exercised
// deterministically.
type failoverTestHarness struct {
	store                *Store
	appHub               *hub.Hub
	projHub              *hub.Hub
	failoverWideCalls    atomic.Int32
	lastFailoverProvider atomic.Value // string — the "from" provider passed to FailoverProviderWide
	startRunCalls        atomic.Int32
	failoverPolicy       FailoverPolicy
	activeProvider       string
	fallbackProvider     string
	fallbackModel        string
	agentHasFallback     bool
	agentAlreadyFailed   bool // simulates NFR-6's one-level cap via IsAgentFailedOver
	providerHealthy      bool
	failoverWideErr      error
	switchedAgents       []string
	noSecondaryAgents    []string

	// disconnectCountLastHour, when set, wires
	// ProviderDisconnectCountLastHour and simulateAgentFailed (below)
	// exercises the provider_disconnected retry_in_place/pause_queue path
	// instead of the default queue.rate_limit simulation.
	disconnectCountLastHour int
	simulateAgentFailed     bool
	simulateFailureReason   string
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
		activeProvider:   "anthropic-cloud",
		fallbackProvider: "gemini-cloud",
		fallbackModel:    "gemini-2.5-flash",
		agentHasFallback: true,
		providerHealthy:  true,
		switchedAgents:   []string{"analyst"},
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
				switch {
				case first && h.simulateAgentFailed:
					h.projHub.Broadcast(hub.Event{
						Type: "agent.failed",
						Payload: map[string]any{
							"run_id":         "test-run",
							"status":         "failed",
							"failure_reason": h.simulateFailureReason,
						},
					})
				case first:
					h.projHub.Broadcast(hub.Event{
						Type: "queue.rate_limit",
						Payload: map[string]any{
							"run_id":   "test-run",
							"raw_text": "HTTP 529 Overloaded",
							"kind":     "overloaded",
						},
					})
				default:
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
		AgentActiveProvider: func(agentName string) (string, bool) {
			return h.activeProvider, true
		},
		IsAgentFailedOver: func(agentName string) bool {
			return h.agentAlreadyFailed
		},
		FailoverProviderWide: func(provider, reason string, resetsAtUnix int64, bucket string) ([]string, []string, error) {
			h.failoverWideCalls.Add(1)
			h.lastFailoverProvider.Store(provider)
			if h.failoverWideErr != nil {
				return nil, nil, h.failoverWideErr
			}
			return h.switchedAgents, h.noSecondaryAgents, nil
		},
		ProviderDisconnectCountLastHour: func(providerName string) int {
			return h.disconnectCountLastHour
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
// failure with the "overloaded" reason resolved to "failover" and a healthy
// fallback provider triggers a project-wide FailoverProviderWide call,
// re-enqueues the job at the head without pausing the queue, and does not
// emit queue.paused.
func TestDispatcher_AutoSwitchWithHealthyFallback(t *testing.T) {
	h := newFailoverTestHarness(t)
	h.failoverPolicy = FailoverPolicy{
		Actions: map[string]string{"overloaded": "failover"},
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

	if h.failoverWideCalls.Load() != 1 {
		t.Errorf("expected FailoverProviderWide called once, got %d", h.failoverWideCalls.Load())
	}
	if v, _ := h.lastFailoverProvider.Load().(string); v != "anthropic-cloud" {
		t.Errorf("expected project-wide failover triggered from anthropic-cloud, got %q", v)
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

// TestDispatcher_ActionPauseQueue_FallsBackToPause verifies that when the
// resolved action for the failure reason is "pause_queue" (e.g. automated
// switchover disabled), the standard rate-limit pause path runs unchanged
// and FailoverProviderWide is never called.
func TestDispatcher_ActionPauseQueue_FallsBackToPause(t *testing.T) {
	h := newFailoverTestHarness(t)
	h.failoverPolicy = FailoverPolicy{
		Actions: map[string]string{"overloaded": "pause_queue"},
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

	if h.failoverWideCalls.Load() != 0 {
		t.Errorf("expected FailoverProviderWide never called, got %d", h.failoverWideCalls.Load())
	}

	paused, _, _, err := h.store.GetPauseState()
	if err != nil {
		t.Fatal(err)
	}
	if !paused {
		t.Error("expected queue to be paused when the resolved action is pause_queue")
	}
	stopCollect()
	if !containsEvent(events, "queue.paused") {
		t.Errorf("expected queue.paused broadcast; got %v", events())
	}
}

// TestDispatcher_UnhealthyFallback_FallsBackToPause verifies that a failed
// pre-switch health probe on the triggering agent's own fallback provider
// prevents project-wide failover from engaging and falls back to the
// standard pause, without calling FailoverProviderWide.
func TestDispatcher_UnhealthyFallback_FallsBackToPause(t *testing.T) {
	h := newFailoverTestHarness(t)
	h.failoverPolicy = FailoverPolicy{
		Actions: map[string]string{"overloaded": "failover"},
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
	if h.failoverWideCalls.Load() != 0 {
		t.Errorf("expected FailoverProviderWide never called when fallback is unhealthy, got %d", h.failoverWideCalls.Load())
	}

	stopCollect()
	if !containsEvent(events, "queue.paused") {
		t.Errorf("expected queue.paused broadcast; got %v", events())
	}
}

// TestDispatcher_AgentAlreadyFailedOver_CapsAtOneLevel verifies the NFR-6
// one-level failover cap: when the triggering agent is already in a
// failover state (its secondary just failed too), a further transient
// failure does not trigger another failover and instead falls back to the
// standard pause — no third target, no cyclic switching.
func TestDispatcher_AgentAlreadyFailedOver_CapsAtOneLevel(t *testing.T) {
	h := newFailoverTestHarness(t)
	h.failoverPolicy = FailoverPolicy{
		Actions: map[string]string{"overloaded": "failover"},
	}
	h.agentAlreadyFailed = true
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
		t.Errorf("failed job reason: got %q, want %q (one-level cap should not trigger a second failover)", failed[0].Reason, "rate_limit")
	}
	if h.failoverWideCalls.Load() != 0 {
		t.Errorf("expected FailoverProviderWide never called once the agent is already failed over, got %d", h.failoverWideCalls.Load())
	}

	stopCollect()
	if !containsEvent(events, "queue.paused") {
		t.Errorf("expected queue.paused broadcast; got %v", events())
	}
}

// TestDispatcher_ProviderDisconnected_RetriesInPlaceBelowThreshold verifies
// that a provider_disconnected failure resolved to retry_in_place is
// re-enqueued immediately without pausing the queue when the rolling-hour
// disconnect count is at or below the FR-6.3 threshold.
func TestDispatcher_ProviderDisconnected_RetriesInPlaceBelowThreshold(t *testing.T) {
	h := newFailoverTestHarness(t)
	h.failoverPolicy = FailoverPolicy{
		Actions: map[string]string{"provider_disconnected": "retry_in_place"},
	}
	h.simulateAgentFailed = true
	h.simulateFailureReason = "provider_disconnected"
	h.disconnectCountLastHour = 2 // at the threshold (>3 triggers pause), not over it
	d := h.dispatcher(time.Now())

	events, stopCollect := collectAppEvents(t, h.appHub)
	defer stopCollect()

	h.enqueue(t, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	d.Start(ctx)

	// The re-enqueued job succeeds immediately (the harness's non-first
	// StartRun call always broadcasts agent.finished), so it may already
	// have left the pending state by the time we observe it — poll across
	// pending/running/completed rather than pending alone.
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
	if failed[0].Reason != "provider_disconnected" {
		t.Errorf("failed job reason: got %q, want %q", failed[0].Reason, "provider_disconnected")
	}

	paused, _, _, err := h.store.GetPauseState()
	if err != nil {
		t.Fatal(err)
	}
	if paused {
		t.Error("expected queue NOT to be paused below the disconnect threshold")
	}
	stopCollect()
	if containsEvent(events, "queue.paused") {
		t.Errorf("expected no queue.paused broadcast below threshold; got %v", events())
	}
}

// TestDispatcher_ProviderDisconnected_PausesOnceThresholdExceeded verifies
// FR-6.3: once the rolling-hour disconnect count for a provider exceeds 3,
// a further provider_disconnected failure pauses the queue instead of
// retrying in place again.
func TestDispatcher_ProviderDisconnected_PausesOnceThresholdExceeded(t *testing.T) {
	h := newFailoverTestHarness(t)
	h.failoverPolicy = FailoverPolicy{
		Actions: map[string]string{"provider_disconnected": "retry_in_place"},
	}
	h.simulateAgentFailed = true
	h.simulateFailureReason = "provider_disconnected"
	h.disconnectCountLastHour = 4 // over the threshold
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

	paused, _, _, err := h.store.GetPauseState()
	if err != nil {
		t.Fatal(err)
	}
	if !paused {
		t.Error("expected queue to be paused once the disconnect threshold is exceeded")
	}
	stopCollect()
	if !containsEvent(events, "queue.paused") {
		t.Errorf("expected queue.paused broadcast; got %v", events())
	}
}
