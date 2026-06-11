// SPDX-License-Identifier: AGPL-3.0-or-later

package triage

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/index"
)

// stubIndex satisfies IndexStore; Get always returns a raw idea row.
type stubIndex struct {
	row *index.ArtifactRow
}

func (s *stubIndex) Get(_ string) (*index.ArtifactRow, error) { return s.row, nil }
func (s *stubIndex) List(_ index.Filter) ([]*index.ArtifactRow, int, error) {
	return nil, 0, nil
}
func (s *stubIndex) Labels() ([]string, error)                { return nil, nil }
func (s *stubIndex) IndexFile(_ string) error                  { return nil }
func (s *stubIndex) InsertAgentRun(_ *index.AgentRunRow) error { return nil }
func (s *stubIndex) UpdateAgentRun(_ *index.AgentRunRow) error { return nil }

// stubLocks satisfies LockManager; Acquire always succeeds.
type stubLocks struct{}

func (s *stubLocks) Acquire(_, _, _ string) (*index.LockRow, error) { return &index.LockRow{}, nil }
func (s *stubLocks) Release(_ string) error                          { return nil }

func rawIdeaRow(relPath string) *index.ArtifactRow {
	return &index.ArtifactRow{
		Path:    relPath,
		Slug:    "test-idea",
		Lineage: "test-idea",
		Type:    "idea",
		Status:  "raw",
	}
}

func newTestManager(maxConcurrent int) *Manager {
	deps := Deps{
		Idx:   &stubIndex{row: rawIdeaRow("lifecycle/ideas/test.md")},
		Locks: &stubLocks{},
	}
	return New(deps, Options{MaxConcurrent: maxConcurrent})
}

func TestTrigger_Dedup(t *testing.T) {
	release := make(chan struct{})
	var invocations atomic.Int32

	mgr := newTestManager(2)
	mgr.opts.executeHook = func(_ context.Context, _, _, _ string, _ TriggerSource) error {
		invocations.Add(1)
		<-release
		return nil
	}

	const relPath = "lifecycle/ideas/test.md"
	ctx := context.Background()

	id1, err := mgr.Trigger(ctx, relPath, TriggerAPI)
	if err != nil {
		t.Fatalf("first Trigger: %v", err)
	}

	// Give the goroutine time to start and register in inFlight.
	time.Sleep(5 * time.Millisecond)

	// Second Trigger for the same path should coalesce.
	id2, err := mgr.Trigger(ctx, relPath, TriggerAPI)
	if err != nil {
		t.Fatalf("second Trigger: %v", err)
	}
	if id1 != id2 {
		t.Errorf("expected run IDs to match; got %q and %q", id1, id2)
	}
	if n := invocations.Load(); n != 1 {
		t.Errorf("expected 1 inner invocation, got %d", n)
	}

	close(release)
	mgr.Stop(context.Background())
}

func TestTrigger_SemaphoreCap(t *testing.T) {
	const maxConcurrent = 2
	var mu sync.Mutex
	var started []string
	started = make([]string, 0, maxConcurrent+1)

	startedCh := make(chan string, maxConcurrent+1)
	release := make(chan struct{})

	// Each distinct path needs its own ArtifactRow.
	paths := []string{
		"lifecycle/ideas/a.md",
		"lifecycle/ideas/b.md",
		"lifecycle/ideas/c.md",
	}

	// Create managers for each path with a per-path stub index.
	deps := Deps{
		Idx: &multiPathIndex{
			rows: map[string]*index.ArtifactRow{
				paths[0]: rawIdeaRowForPath(paths[0]),
				paths[1]: rawIdeaRowForPath(paths[1]),
				paths[2]: rawIdeaRowForPath(paths[2]),
			},
		},
		Locks: &stubLocks{},
	}
	mgr := New(deps, Options{MaxConcurrent: maxConcurrent})
	mgr.opts.executeHook = func(_ context.Context, _, relPath, _ string, _ TriggerSource) error {
		startedCh <- relPath
		<-release
		return nil
	}

	ctx := context.Background()

	// Trigger paths[0] and paths[1] — fill the semaphore.
	if _, err := mgr.Trigger(ctx, paths[0], TriggerAPI); err != nil {
		t.Fatalf("Trigger paths[0]: %v", err)
	}
	if _, err := mgr.Trigger(ctx, paths[1], TriggerAPI); err != nil {
		t.Fatalf("Trigger paths[1]: %v", err)
	}

	// Wait for both to start.
	for i := 0; i < 2; i++ {
		select {
		case p := <-startedCh:
			mu.Lock()
			started = append(started, p)
			mu.Unlock()
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for execute to start")
		}
	}

	// Trigger paths[2] concurrently — it must block until a slot frees.
	done := make(chan struct{})
	var thirdID string
	go func() {
		defer close(done)
		id, err := mgr.Trigger(ctx, paths[2], TriggerAPI)
		if err != nil {
			t.Errorf("Trigger paths[2]: %v", err)
			return
		}
		thirdID = id
	}()

	// Release one slot — paths[2]'s Trigger should proceed.
	release <- struct{}{} // unblock one of the first two

	// Wait for the third run to start.
	select {
	case p := <-startedCh:
		mu.Lock()
		started = append(started, p)
		mu.Unlock()
		_ = p
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for third execute to start")
	}

	close(release) // unblock remaining runs
	<-done

	if thirdID == "" {
		t.Error("expected non-empty run ID for paths[2]")
	}

	mgr.Stop(context.Background())
}

func TestStop_WaitsForInFlight(t *testing.T) {
	release := make(chan struct{})
	finished := make(chan struct{})

	mgr := newTestManager(2)
	mgr.opts.executeHook = func(_ context.Context, _, _, _ string, _ TriggerSource) error {
		<-release
		return nil
	}

	if _, err := mgr.Trigger(context.Background(), "lifecycle/ideas/test.md", TriggerAPI); err != nil {
		t.Fatalf("Trigger: %v", err)
	}

	go func() {
		mgr.Stop(context.Background())
		close(finished)
	}()

	// Stop should not return while execute is blocking.
	select {
	case <-finished:
		t.Fatal("Stop returned before execute finished")
	case <-time.After(20 * time.Millisecond):
	}

	close(release) // let execute finish

	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("Stop did not return after execute finished")
	}
}

func TestTrigger_LockReleasedOnFailure(t *testing.T) {
	released := make(chan struct{}, 1)
	locks := &panicTrackingLocks{released: released}

	idx := &stubIndex{row: rawIdeaRow("lifecycle/ideas/fail.md")}
	mgr := New(Deps{Idx: idx, Locks: locks}, Options{})
	mgr.opts.executeHook = func(_ context.Context, _, _, _ string, _ TriggerSource) error {
		return fmt.Errorf("simulated failure")
	}

	if _, err := mgr.Trigger(context.Background(), "lifecycle/ideas/fail.md", TriggerAPI); err != nil {
		t.Fatalf("Trigger: %v", err)
	}

	mgr.Stop(context.Background())

	select {
	case <-released:
		// lock was released
	default:
		t.Error("lock was not released after execute failure")
	}
}

// multiPathIndex returns a different ArtifactRow per path.
type multiPathIndex struct {
	rows map[string]*index.ArtifactRow
}

func (m *multiPathIndex) Get(relPath string) (*index.ArtifactRow, error) {
	return m.rows[relPath], nil
}
func (m *multiPathIndex) List(_ index.Filter) ([]*index.ArtifactRow, int, error) {
	return nil, 0, nil
}
func (m *multiPathIndex) Labels() ([]string, error)                { return nil, nil }
func (m *multiPathIndex) IndexFile(_ string) error                  { return nil }
func (m *multiPathIndex) InsertAgentRun(_ *index.AgentRunRow) error { return nil }
func (m *multiPathIndex) UpdateAgentRun(_ *index.AgentRunRow) error { return nil }

func rawIdeaRowForPath(relPath string) *index.ArtifactRow {
	return &index.ArtifactRow{
		Path:    relPath,
		Slug:    "test-idea",
		Lineage: relPath, // use path as lineage to avoid lock conflicts
		Type:    "idea",
		Status:  "raw",
	}
}
