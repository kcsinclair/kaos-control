// SPDX-License-Identifier: AGPL-3.0-or-later

package queue

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "queue.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func job(project, path, agent, by string) Job {
	return Job{
		Project:      project,
		ArtifactPath: path,
		AgentName:    agent,
		EnqueuedBy:   by,
	}
}

func TestEnqueueDequeueOrder(t *testing.T) {
	s := openTestStore(t)

	// Enqueue 3 jobs; they should dequeue in FIFO order.
	jobs := []Job{
		job("proj", "lifecycle/ideas/a.md", "analyst", "alice@example.com"),
		job("proj", "lifecycle/ideas/b.md", "analyst", "alice@example.com"),
		job("proj", "lifecycle/ideas/c.md", "analyst", "alice@example.com"),
	}
	for i := range jobs {
		if err := s.Enqueue(jobs[i]); err != nil {
			t.Fatalf("Enqueue[%d]: %v", i, err)
		}
	}

	for i, want := range []string{"lifecycle/ideas/a.md", "lifecycle/ideas/b.md", "lifecycle/ideas/c.md"} {
		got, err := s.Dequeue()
		if err != nil {
			t.Fatalf("Dequeue[%d]: %v", i, err)
		}
		if got == nil {
			t.Fatalf("Dequeue[%d]: got nil, want %q", i, want)
		}
		if got.ArtifactPath != want {
			t.Errorf("Dequeue[%d]: got path %q, want %q", i, got.ArtifactPath, want)
		}
		if got.State != StateRunning {
			t.Errorf("Dequeue[%d]: state=%q, want running", i, got.State)
		}
	}

	// Queue now empty.
	empty, err := s.Dequeue()
	if err != nil || empty != nil {
		t.Errorf("expected empty queue, got %v err=%v", empty, err)
	}
}

func TestTerminalTransitions(t *testing.T) {
	s := openTestStore(t)
	if err := s.Enqueue(job("proj", "lifecycle/ideas/x.md", "analyst", "alice@example.com")); err != nil {
		t.Fatal(err)
	}
	j, _ := s.Dequeue()
	if j == nil {
		t.Fatal("expected job")
	}
	for _, st := range []JobState{StateCompleted, StateFailed, StateSkipped, StateCancelled} {
		// Reset to running for each iteration.
		s2 := openTestStore(t)
		_ = s2.Enqueue(job("proj", "lifecycle/ideas/y-"+string(st)+".md", "analyst", "alice@example.com"))
		jj, _ := s2.Dequeue()
		if err := s2.MarkTerminal(jj.ID, st, "test"); err != nil {
			t.Errorf("MarkTerminal(%q): %v", st, err)
		}
		got, _ := s2.GetByID(jj.ID)
		if got == nil || got.State != st {
			t.Errorf("MarkTerminal(%q): state=%v", st, got)
		}
	}
	// Non-terminal state should error.
	if err := s.MarkTerminal(j.ID, StateRunning, ""); err == nil {
		t.Error("expected error for non-terminal state")
	}
}

func TestDuplicateDetection(t *testing.T) {
	s := openTestStore(t)
	j := job("proj", "lifecycle/ideas/a.md", "analyst", "alice@example.com")
	if err := s.Enqueue(j); err != nil {
		t.Fatal(err)
	}
	// Second enqueue of same path while still pending → duplicate error.
	if err := s.Enqueue(j); err != ErrDuplicateActive {
		t.Errorf("expected ErrDuplicateActive, got %v", err)
	}

	// Dequeue (becomes running) → still an active job.
	running, _ := s.Dequeue()
	if err := s.Enqueue(j); err != ErrDuplicateActive {
		t.Errorf("expected ErrDuplicateActive while running, got %v", err)
	}

	// Complete → can enqueue again.
	_ = s.MarkTerminal(running.ID, StateCompleted, "")
	if err := s.Enqueue(j); err != nil {
		t.Errorf("expected success after completion, got %v", err)
	}
}

func TestOrphanRecovery(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "queue.db")

	// Open, enqueue, dequeue (marks running), then close without completing.
	s1, _ := Open(path)
	_ = s1.Enqueue(job("proj", "lifecycle/ideas/a.md", "analyst", "alice@example.com"))
	j, _ := s1.Dequeue()
	if j.State != StateRunning {
		t.Fatal("expected running state")
	}
	// Record attempts before crash.
	attemptsBefore := j.Attempts
	_ = s1.Close()

	// Reopen and recover — simulates a server restart.
	s2, _ := Open(path)
	defer func() { _ = s2.Close() }()

	// Remove old file — we're testing in-memory recovery; the DB is persistent.
	if err := s2.RecoverOrphans(); err != nil {
		t.Fatalf("RecoverOrphans: %v", err)
	}

	// The orphaned job should be pending again with incremented attempts.
	recovered, err := s2.Dequeue()
	if err != nil {
		t.Fatalf("Dequeue after recovery: %v", err)
	}
	if recovered == nil {
		t.Fatal("expected recovered job")
	}
	if recovered.Attempts != attemptsBefore+1 {
		t.Errorf("attempts: got %d, want %d", recovered.Attempts, attemptsBefore+1)
	}
	_ = os.Remove(path)
}

func TestPauseStateRoundTrip(t *testing.T) {
	s := openTestStore(t)

	// Initially not paused.
	paused, until, reason, err := s.GetPauseState()
	if err != nil || paused || !until.IsZero() || reason != "" {
		t.Errorf("initial state: paused=%v until=%v reason=%q err=%v", paused, until, reason, err)
	}

	// Set paused with a reset time.
	resetTime := time.Now().Add(30 * time.Minute).UTC().Truncate(time.Second)
	if err := s.SetPauseState(true, resetTime, "rate_limit"); err != nil {
		t.Fatalf("SetPauseState: %v", err)
	}
	paused, until, reason, err = s.GetPauseState()
	if err != nil {
		t.Fatalf("GetPauseState: %v", err)
	}
	if !paused {
		t.Error("expected paused=true")
	}
	if reason != "rate_limit" {
		t.Errorf("reason: got %q, want %q", reason, "rate_limit")
	}
	if !until.Equal(resetTime) {
		t.Errorf("until: got %v, want %v", until, resetTime)
	}

	// Clear pause.
	if err := s.SetPauseState(false, time.Time{}, ""); err != nil {
		t.Fatalf("SetPauseState clear: %v", err)
	}
	paused, until, reason, err = s.GetPauseState()
	if err != nil || paused || !until.IsZero() {
		t.Errorf("after clear: paused=%v until=%v reason=%q err=%v", paused, until, reason, err)
	}
}

func TestCancelPendingOnly(t *testing.T) {
	s := openTestStore(t)
	_ = s.Enqueue(job("proj", "lifecycle/ideas/a.md", "analyst", "alice@example.com"))
	pending, _ := s.ListByState(StatePending)
	if len(pending) != 1 {
		t.Fatal("expected 1 pending job")
	}

	// Cancel while pending.
	if err := s.Cancel(pending[0].ID); err != nil {
		t.Fatalf("Cancel pending: %v", err)
	}
	got, _ := s.GetByID(pending[0].ID)
	if got.State != StateCancelled {
		t.Errorf("expected cancelled, got %q", got.State)
	}

	// Set up a running job and verify cancel is rejected.
	_ = s.Enqueue(job("proj", "lifecycle/ideas/b.md", "analyst", "alice@example.com"))
	running, _ := s.Dequeue()
	if err := s.Cancel(running.ID); err != ErrCannotCancelRunning {
		t.Errorf("expected ErrCannotCancelRunning, got %v", err)
	}
}
