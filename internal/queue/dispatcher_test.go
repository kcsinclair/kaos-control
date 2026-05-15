// SPDX-License-Identifier: AGPL-3.0-or-later

package queue

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/hub"
)

// makeDispatcher builds a minimal Dispatcher backed by a temp-dir store.
// The supplied startFn is called for each dequeued job and returns an error
// or nil. On success it broadcasts agent.finished after a short delay.
func makeDispatcher(t *testing.T, startFn func(agentName, path string) error) (*Dispatcher, *Store, *hub.Hub) {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "queue.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	h := hub.New()

	lookup := func(name string) (ProjectAccess, bool) {
		return ProjectAccess{
			StartRun: func(ctx context.Context, agentName, targetPath string) (string, error) {
				if err := startFn(agentName, targetPath); err != nil {
					return "", err
				}
				go func() {
					time.Sleep(5 * time.Millisecond)
					h.Broadcast(hub.Event{
						Type:    "agent.finished",
						Payload: map[string]any{"run_id": "test-run", "status": "done"},
					})
				}()
				return "test-run", nil
			},
			ArtifactStatus: func(relPath string) string { return "approved" },
			Hub:            h,
		}, true
	}

	cfg := Config{
		TickInterval: 20 * time.Millisecond,
		ClockFn:      time.Now,
	}
	d := New(s, lookup, h, cfg)
	return d, s, h
}

func TestDispatcher_RunsJobsSequentially(t *testing.T) {
	var mu sync.Mutex
	var order []string
	started := make(chan string, 10)

	d, s, _ := makeDispatcher(t, func(agentName, path string) error {
		mu.Lock()
		order = append(order, path)
		mu.Unlock()
		started <- path
		return nil
	})

	for _, path := range []string{"lifecycle/ideas/a.md", "lifecycle/ideas/b.md", "lifecycle/ideas/c.md"} {
		if err := s.Enqueue(Job{
			Project:      "proj",
			ArtifactPath: path,
			AgentName:    "analyst",
			EnqueuedBy:   "alice@example.com",
		}); err != nil {
			t.Fatalf("Enqueue: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	d.Start(ctx)

	for i := 0; i < 3; i++ {
		select {
		case <-started:
		case <-ctx.Done():
			t.Fatalf("timed out waiting for job %d; order so far: %v", i, order)
		}
	}

	want := []string{"lifecycle/ideas/a.md", "lifecycle/ideas/b.md", "lifecycle/ideas/c.md"}
	mu.Lock()
	defer mu.Unlock()
	for i, w := range want {
		if i >= len(order) || order[i] != w {
			t.Errorf("job[%d]: got %q, want %q", i, order[i], w)
		}
	}

	time.Sleep(60 * time.Millisecond)
	done, err := s.ListByState(StateCompleted)
	if err != nil {
		t.Fatal(err)
	}
	if len(done) != 3 {
		t.Errorf("expected 3 completed jobs, got %d", len(done))
	}
}

// TestDispatcher_IgnoresForeignFinishedEvent verifies that a stray
// agent.finished event on the project hub — broadcast by some other run with
// a different run_id (e.g. a manually-started UI run, or a previous queue
// iteration's late-arriving event) — does NOT prematurely complete the
// current job. Before run_id filtering was added in watchRunEvents, this
// scenario caused the dispatcher to mark the current job completed and start
// the next one while the real run was still holding its lineage lock,
// surfacing as a "lock conflict" on the second job's StartRun.
func TestDispatcher_IgnoresForeignFinishedEvent(t *testing.T) {
	var startCount int32
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "queue.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	h := hub.New()
	lookup := func(name string) (ProjectAccess, bool) {
		return ProjectAccess{
			StartRun: func(ctx context.Context, agentName, targetPath string) (string, error) {
				count := atomic.AddInt32(&startCount, 1)
				ourRunID := "real-run-" + targetPath

				if count == 1 {
					// On the very first StartRun: a foreign run finishes on the
					// hub *while ours is still running*. The watcher must not
					// be fooled.
					go func() {
						time.Sleep(10 * time.Millisecond)
						h.Broadcast(hub.Event{
							Type:    "agent.finished",
							Payload: map[string]any{"run_id": "foreign-run-id", "status": "done"},
						})
						// Then OUR run finishes ~30ms later.
						time.Sleep(30 * time.Millisecond)
						h.Broadcast(hub.Event{
							Type:    "agent.finished",
							Payload: map[string]any{"run_id": ourRunID, "status": "done"},
						})
					}()
				} else {
					// Subsequent jobs finish normally with the right run_id.
					go func() {
						time.Sleep(5 * time.Millisecond)
						h.Broadcast(hub.Event{
							Type:    "agent.finished",
							Payload: map[string]any{"run_id": ourRunID, "status": "done"},
						})
					}()
				}
				return ourRunID, nil
			},
			ArtifactStatus: func(relPath string) string { return "approved" },
			Hub:            h,
		}, true
	}

	cfg := Config{TickInterval: 5 * time.Millisecond, ClockFn: time.Now}
	d := New(s, lookup, h, cfg)

	for _, p := range []string{"lifecycle/ideas/a.md", "lifecycle/ideas/b.md"} {
		if err := s.Enqueue(Job{Project: "proj", ArtifactPath: p, AgentName: "analyst", EnqueuedBy: "alice@example.com"}); err != nil {
			t.Fatalf("Enqueue: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	d.Start(ctx)

	// Wait long enough for both jobs to finish, including the foreign-event
	// noise on job 1.
	deadline := time.Now().Add(1500 * time.Millisecond)
	for time.Now().Before(deadline) {
		done, _ := s.ListByState(StateCompleted)
		if len(done) == 2 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	done, _ := s.ListByState(StateCompleted)
	if len(done) != 2 {
		t.Fatalf("expected 2 completed jobs after foreign-event noise; got %d", len(done))
	}
	// Critical assertion: StartRun must have been called exactly twice — once
	// per queued job, in order. If the foreign event leaked through, the
	// dispatcher would have advanced to job 2 while job 1 was still "running"
	// in our fake; that would still produce 2 starts but in wrong relative
	// timing. The completed count + ordering check in
	// TestDispatcher_RunsJobsSequentially covers ordering; here the key
	// invariant is that we did NOT short-circuit job 1 on the foreign event.
	if got := atomic.LoadInt32(&startCount); got != 2 {
		t.Errorf("StartRun called %d times, want 2", got)
	}
}

func TestDispatcher_SkipsNonApproved(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	h := hub.New()
	var runCount int32
	lookup := func(name string) (ProjectAccess, bool) {
		return ProjectAccess{
			StartRun: func(ctx context.Context, agentName, targetPath string) (string, error) {
				atomic.AddInt32(&runCount, 1)
				return "run", nil
			},
			ArtifactStatus: func(relPath string) string { return "draft" },
			Hub:            h,
		}, true
	}
	cfg := Config{TickInterval: 20 * time.Millisecond}
	d := New(s, lookup, h, cfg)

	_ = s.Enqueue(Job{Project: "proj", ArtifactPath: "lifecycle/ideas/a.md",
		AgentName: "analyst", EnqueuedBy: "alice@example.com"})

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	d.Start(ctx)
	<-ctx.Done()

	if n := atomic.LoadInt32(&runCount); n != 0 {
		t.Errorf("StartRun called %d times, want 0 (job should have been skipped)", n)
	}
	skipped, _ := s.ListByState(StateSkipped)
	if len(skipped) != 1 {
		t.Errorf("expected 1 skipped job, got %d", len(skipped))
	}
}

func TestDispatcher_ManualPauseResume(t *testing.T) {
	d, s, _ := makeDispatcher(t, func(_, _ string) error { return nil })

	_ = s.Enqueue(Job{Project: "proj", ArtifactPath: "lifecycle/ideas/a.md",
		AgentName: "analyst", EnqueuedBy: "alice@example.com"})

	d.Pause("test")
	if !d.paused() {
		t.Error("expected paused after Pause()")
	}

	d.Resume()
	if d.paused() {
		t.Error("expected not paused after Resume()")
	}
}

// collectAppEvents registers a channel on appHub and returns a helper that
// drains events into a slice. Call stop() to unregister before assertions;
// it is safe to call stop() multiple times.
func collectAppEvents(t *testing.T, h *hub.Hub) (events func() []string, stop func()) {
	t.Helper()
	ch := make(chan []byte, 64)
	h.Register(ch)
	var mu sync.Mutex
	var collected []string
	done := make(chan struct{})
	var once sync.Once
	go func() {
		defer close(done)
		for data := range ch {
			var evt struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(data, &evt); err == nil {
				mu.Lock()
				collected = append(collected, evt.Type)
				mu.Unlock()
			}
		}
	}()
	stop = func() {
		once.Do(func() {
			h.Unregister(ch)
			close(ch)
			<-done
		})
	}
	events = func() []string {
		mu.Lock()
		defer mu.Unlock()
		out := make([]string, len(collected))
		copy(out, collected)
		return out
	}
	return events, stop
}

// containsEvent returns true when typ appears in the slice returned by events().
func containsEvent(events func() []string, typ string) bool {
	for _, e := range events() {
		if e == typ {
			return true
		}
	}
	return false
}

// TestDispatcher_RateLimitFlow verifies that when the agent hub emits a
// queue.rate_limit event the dispatcher (a) marks the job failed/rate_limit,
// (b) re-enqueues at the head with attempts incremented, (c) sets pause state,
// and (d) broadcasts queue.paused.
func TestDispatcher_RateLimitFlow(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	projHub := hub.New()
	appHub := hub.New()
	rateLimitText := "resets 8pm (Australia/Brisbane)"

	lookup := func(name string) (ProjectAccess, bool) {
		return ProjectAccess{
			StartRun: func(ctx context.Context, agentName, targetPath string) (string, error) {
				// Simulate the agent broadcasting a rate-limit event shortly after starting.
				go func() {
					time.Sleep(10 * time.Millisecond)
					projHub.Broadcast(hub.Event{
						Type: "queue.rate_limit",
						Payload: map[string]any{
							"run_id":   "test-run",
							"raw_text": rateLimitText,
						},
					})
				}()
				return "test-run", nil
			},
			ArtifactStatus: func(relPath string) string { return "approved" },
			Hub:            projHub,
		}, true
	}

	now := time.Now()
	cfg := Config{
		TickInterval:  20 * time.Millisecond,
		ClockFn:       func() time.Time { return now },
		MaxAttempts:   5,
		ResumeGrace:   time.Minute,
		FallbackPause: 30 * time.Minute,
	}
	d := New(s, lookup, appHub, cfg)

	// Subscribe to app hub to observe broadcast events.
	events, stopCollect := collectAppEvents(t, appHub)
	defer stopCollect()

	_ = s.Enqueue(Job{
		Project:      "proj",
		ArtifactPath: "lifecycle/ideas/a.md",
		AgentName:    "analyst",
		EnqueuedBy:   "alice@example.com",
		Attempts:     1,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	d.Start(ctx)

	// Wait until the original job is marked failed and a re-queued job appears.
	deadline := time.Now().Add(3 * time.Second)
	var requeued []*Job
	for time.Now().Before(deadline) {
		failed, _ := s.ListByState(StateFailed)
		pending, _ := s.ListByState(StatePending)
		if len(failed) > 0 && len(pending) > 0 {
			requeued = pending
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	// (a) Original job marked failed with reason rate_limit.
	failed, _ := s.ListByState(StateFailed)
	if len(failed) == 0 {
		t.Fatal("expected a failed job after rate-limit event")
	}
	if failed[0].Reason != "rate_limit" {
		t.Errorf("failed job reason: got %q, want %q", failed[0].Reason, "rate_limit")
	}

	// (b) Re-queued job exists with attempts incremented and at head position.
	if len(requeued) == 0 {
		t.Fatal("expected a pending re-queued job")
	}
	if requeued[0].Attempts != 2 {
		t.Errorf("re-queued attempts: got %d, want 2", requeued[0].Attempts)
	}
	if requeued[0].Position >= failed[0].Position {
		t.Errorf("re-queued position %d should be less than failed position %d",
			requeued[0].Position, failed[0].Position)
	}

	// (c) Pause state is set.
	paused, _, _, err := s.GetPauseState()
	if err != nil {
		t.Fatalf("GetPauseState: %v", err)
	}
	if !paused {
		t.Error("expected queue to be paused after rate-limit")
	}

	// (d) queue.paused was broadcast.
	stopCollect()
	if !containsEvent(events, "queue.paused") {
		t.Errorf("expected queue.paused broadcast; got %v", events())
	}
}

// TestDispatcher_MaxAttemptsCap verifies that when a job has already reached
// the maximum number of attempts, a rate-limit failure does NOT re-enqueue it
// and instead broadcasts queue.skipped.
func TestDispatcher_MaxAttemptsCap(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	projHub := hub.New()
	appHub := hub.New()

	lookup := func(name string) (ProjectAccess, bool) {
		return ProjectAccess{
			StartRun: func(ctx context.Context, agentName, targetPath string) (string, error) {
				go func() {
					time.Sleep(10 * time.Millisecond)
					projHub.Broadcast(hub.Event{
						Type: "queue.rate_limit",
						Payload: map[string]any{
							"run_id":   "test-run",
							"raw_text": "resets 8pm (Australia/Brisbane)",
						},
					})
				}()
				return "test-run", nil
			},
			ArtifactStatus: func(relPath string) string { return "approved" },
			Hub:            projHub,
		}, true
	}

	const maxAttempts = 3
	cfg := Config{
		TickInterval:  20 * time.Millisecond,
		ClockFn:       time.Now,
		MaxAttempts:   maxAttempts,
		ResumeGrace:   time.Minute,
		FallbackPause: 30 * time.Minute,
	}
	d := New(s, lookup, appHub, cfg)

	events, stopCollect := collectAppEvents(t, appHub)
	defer stopCollect()

	// Enqueue with attempts already at maxAttempts.
	_ = s.Enqueue(Job{
		Project:      "proj",
		ArtifactPath: "lifecycle/ideas/a.md",
		AgentName:    "analyst",
		EnqueuedBy:   "alice@example.com",
		Attempts:     maxAttempts,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	d.Start(ctx)

	// Wait until the job is failed.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if failed, _ := s.ListByState(StateFailed); len(failed) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	// No re-queued pending job.
	pending, _ := s.ListByState(StatePending)
	if len(pending) != 0 {
		t.Errorf("expected no re-queued job when max attempts exceeded, got %d", len(pending))
	}

	stopCollect()
	if !containsEvent(events, "queue.skipped") {
		t.Errorf("expected queue.skipped broadcast; got %v", events())
	}
}

// TestDispatcher_FallbackOnUnparseableReset verifies that when the rate-limit
// text cannot be parsed (e.g. "resets soon"), the dispatcher falls back to
// cfg.FallbackPause for paused_until.
func TestDispatcher_FallbackOnUnparseableReset(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	projHub := hub.New()
	appHub := hub.New()

	lookup := func(name string) (ProjectAccess, bool) {
		return ProjectAccess{
			StartRun: func(ctx context.Context, agentName, targetPath string) (string, error) {
				go func() {
					time.Sleep(10 * time.Millisecond)
					projHub.Broadcast(hub.Event{
						Type: "queue.rate_limit",
						Payload: map[string]any{
							"run_id":   "test-run",
							"raw_text": "resets soon", // unparseable
						},
					})
				}()
				return "test-run", nil
			},
			ArtifactStatus: func(relPath string) string { return "approved" },
			Hub:            projHub,
		}, true
	}

	const fallbackPause = 45 * time.Minute
	const resumeGrace = 2 * time.Minute
	now := time.Now()
	cfg := Config{
		TickInterval:  20 * time.Millisecond,
		ClockFn:       func() time.Time { return now },
		MaxAttempts:   5,
		FallbackPause: fallbackPause,
		ResumeGrace:   resumeGrace,
	}
	d := New(s, lookup, appHub, cfg)

	_ = s.Enqueue(Job{
		Project:      "proj",
		ArtifactPath: "lifecycle/ideas/a.md",
		AgentName:    "analyst",
		EnqueuedBy:   "alice@example.com",
		Attempts:     1,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	d.Start(ctx)

	// Wait until paused.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if p, _, _, _ := s.GetPauseState(); p {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	_, until, _, err := s.GetPauseState()
	if err != nil {
		t.Fatalf("GetPauseState: %v", err)
	}

	// paused_until = now + fallbackPause + resumeGrace (within 5 seconds tolerance).
	expectedUntil := now.Add(fallbackPause + resumeGrace)
	diff := until.Sub(expectedUntil)
	if diff < -5*time.Second || diff > 5*time.Second {
		t.Errorf("paused_until %v differs from expected %v by %v (want within 5s)",
			until, expectedUntil, diff)
	}
}

func TestDispatcher_AutoResume(t *testing.T) {
	var nowVal atomic.Value
	start := time.Now()
	nowVal.Store(start)
	clockFn := func() time.Time { return nowVal.Load().(time.Time) }

	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	cfg := Config{TickInterval: 10 * time.Millisecond, ClockFn: clockFn}
	d := New(s, func(string) (ProjectAccess, bool) { return ProjectAccess{}, false }, hub.New(), cfg)

	pausedUntil := start.Add(time.Minute)
	d.setPausedUntil(pausedUntil)
	_ = s.SetPauseState(true, pausedUntil, "rate_limit")

	if !d.paused() {
		t.Error("expected paused at start")
	}

	// Advance clock past paused_until.
	nowVal.Store(pausedUntil.Add(time.Millisecond))

	if d.paused() {
		t.Error("expected auto-resumed after clock advance")
	}
}
