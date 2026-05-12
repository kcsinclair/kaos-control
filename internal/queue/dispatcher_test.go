// SPDX-License-Identifier: AGPL-3.0-or-later

package queue

import (
	"context"
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
