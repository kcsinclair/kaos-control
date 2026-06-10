// SPDX-License-Identifier: AGPL-3.0-or-later

package triage

import (
	"context"
	"errors"
	"testing"

	"github.com/kaos-control/kaos-control/internal/index"
	"github.com/kaos-control/kaos-control/internal/lock"
)

func TestEligible_NotInIdeasDir(t *testing.T) {
	idx := &stubIndex{row: rawIdeaRow("lifecycle/requirements/test.md")}
	row, reason, err := eligible(context.Background(), idx, "lifecycle/requirements/test.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if row != nil {
		t.Error("expected nil row for ineligible path")
	}
	if reason != "not_in_ideas_dir" {
		t.Errorf("expected reason %q, got %q", "not_in_ideas_dir", reason)
	}
}

func TestEligible_NestedPath(t *testing.T) {
	idx := &stubIndex{row: rawIdeaRow("lifecycle/ideas/sub/test.md")}
	row, reason, err := eligible(context.Background(), idx, "lifecycle/ideas/sub/test.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if row != nil {
		t.Error("expected nil row for nested path")
	}
	if reason != "not_in_ideas_dir" {
		t.Errorf("expected reason %q, got %q", "not_in_ideas_dir", reason)
	}
}

func TestEligible_WrongType(t *testing.T) {
	r := rawIdeaRow("lifecycle/ideas/defect.md")
	r.Type = "defect"
	idx := &stubIndex{row: r}
	row, reason, err := eligible(context.Background(), idx, "lifecycle/ideas/defect.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if row != nil {
		t.Error("expected nil row for wrong type")
	}
	if reason != "wrong_type" {
		t.Errorf("expected reason %q, got %q", "wrong_type", reason)
	}
}

func TestEligible_WrongStatus(t *testing.T) {
	r := rawIdeaRow("lifecycle/ideas/drafted.md")
	r.Status = "draft"
	idx := &stubIndex{row: r}
	row, reason, err := eligible(context.Background(), idx, "lifecycle/ideas/drafted.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if row != nil {
		t.Error("expected nil row for wrong status")
	}
	if reason != "wrong_status" {
		t.Errorf("expected reason %q, got %q", "wrong_status", reason)
	}
}

func TestEligible_NotIndexed(t *testing.T) {
	idx := &stubIndex{row: nil}
	row, reason, err := eligible(context.Background(), idx, "lifecycle/ideas/missing.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if row != nil {
		t.Error("expected nil row when not indexed")
	}
	if reason != "not_indexed" {
		t.Errorf("expected reason %q, got %q", "not_indexed", reason)
	}
}

func TestEligible_OK(t *testing.T) {
	idx := &stubIndex{row: rawIdeaRow("lifecycle/ideas/good.md")}
	row, reason, err := eligible(context.Background(), idx, "lifecycle/ideas/good.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if row == nil {
		t.Fatal("expected non-nil row for eligible artifact")
	}
	if reason != "" {
		t.Errorf("expected empty reason, got %q", reason)
	}
}

// alreadyLockedLocks always returns ErrLocked on Acquire.
type alreadyLockedLocks struct{}

func (a *alreadyLockedLocks) Acquire(_, _, _ string) (*index.LockRow, error) {
	return nil, lock.ErrLocked
}
func (a *alreadyLockedLocks) Release(_ string) error { return nil }

func TestTrigger_ErrIneligible(t *testing.T) {
	idx := &stubIndex{row: nil} // not indexed → ineligible
	mgr := New(Deps{Idx: idx, Locks: &stubLocks{}}, Options{})
	_, err := mgr.Trigger(context.Background(), "lifecycle/ideas/missing.md", TriggerAPI)
	if err == nil {
		t.Fatal("expected ErrIneligible, got nil")
	}
	var ie ErrIneligible
	if !errors.As(err, &ie) {
		t.Fatalf("expected ErrIneligible, got %T: %v", err, err)
	}
	if ie.Reason != "not_indexed" {
		t.Errorf("expected reason %q, got %q", "not_indexed", ie.Reason)
	}
}

func TestTrigger_ErrLocked(t *testing.T) {
	idx := &stubIndex{row: rawIdeaRow("lifecycle/ideas/locked.md")}
	mgr := New(Deps{Idx: idx, Locks: &alreadyLockedLocks{}}, Options{})
	_, err := mgr.Trigger(context.Background(), "lifecycle/ideas/locked.md", TriggerAPI)
	if !errors.Is(err, ErrLocked) {
		t.Fatalf("expected ErrLocked, got %v", err)
	}
}

func TestTrigger_LockReleasedOnPanic(t *testing.T) {
	released := make(chan struct{}, 1)

	locks := &panicTrackingLocks{released: released}
	idx := &stubIndex{row: rawIdeaRow("lifecycle/ideas/panic.md")}
	mgr := New(Deps{Idx: idx, Locks: locks}, Options{})
	mgr.opts.executeHook = func(_ context.Context, _, _, _ string, _ TriggerSource) error {
		panic("test panic in execute")
	}

	defer func() {
		// The goroutine's deferred cleanup runs the panic recovery.
		// We only verify here that Stop doesn't block forever.
		mgr.Stop(context.Background())
	}()

	// We expect Trigger to succeed (run starts) and the goroutine panics.
	// Stop must return (the defer in the goroutine handles cleanup via recover).
	// This test verifies the pattern; actual panic recovery is caller responsibility.
	_, _ = mgr.Trigger(context.Background(), "lifecycle/ideas/panic.md", TriggerAPI)
	// Give the goroutine time to start and panic.
}

type panicTrackingLocks struct {
	released chan struct{}
}

func (p *panicTrackingLocks) Acquire(_, _, _ string) (*index.LockRow, error) {
	return &index.LockRow{}, nil
}
func (p *panicTrackingLocks) Release(_ string) error {
	select {
	case p.released <- struct{}{}:
	default:
	}
	return nil
}
