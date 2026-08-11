// SPDX-License-Identifier: AGPL-3.0-or-later

package devops

// Milestone: Cancel resolves orphaned runs to a terminal state.
//
// Fast filesystem-level tests for the orphan-reconciliation additions in
// reconcile.go. Uses t.TempDir() and events written via WriteEvent — no
// HTTP, no Runner goroutines.
//
// Run with: go test ./internal/devops/ -run Reconcile

import (
	"os"
	"strings"
	"testing"
)

// seedOrphanLog writes a run.started event (and, unless terminal is true, no
// completion event) so the resulting .log file has no .meta.json sidecar —
// simulating a run whose process died before writing a terminal record.
func seedOrphanLog(t *testing.T, ls *LogStore, project, runID, slug string) {
	t.Helper()
	ls.WriteEvent(project, runID, EventRunStarted, RunStartedPayload{
		RunID:    runID,
		Pipeline: slug,
		Project:  project,
	})
}

func TestReconcileSlugOrphan_MarksCancelled(t *testing.T) {
	ls := NewLogStore(t.TempDir())
	const project = "testproject"
	const slug = "stuck-pipe"
	const runID = "ee00000000000001"

	seedOrphanLog(t, ls, project, runID, slug)

	rec, ok, err := ls.ReconcileSlugOrphan(project, slug)
	if err != nil {
		t.Fatalf("ReconcileSlugOrphan: %v", err)
	}
	if !ok {
		t.Fatal("expected an orphaned run to be reconciled")
	}
	if rec.RunID != runID {
		t.Errorf("RunID = %q, want %q", rec.RunID, runID)
	}
	if rec.Status != string(StepCancelled) {
		t.Errorf("Status = %q, want %q", rec.Status, StepCancelled)
	}

	// The sidecar must now exist and be readable via Record.
	got, ok := ls.Record(project, runID)
	if !ok {
		t.Fatal("Record not found after reconciliation")
	}
	if got.Status != string(StepCancelled) {
		t.Errorf("persisted Status = %q, want %q", got.Status, StepCancelled)
	}

	// The log must have gained a run.completed entry.
	data, err := ls.ReadLog(project, runID)
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	if !strings.Contains(string(data), EventRunCompleted) {
		t.Errorf("expected log to contain a %q entry after reconciliation", EventRunCompleted)
	}
}

func TestReconcileSlugOrphan_NoOrphanReturnsFalse(t *testing.T) {
	ls := NewLogStore(t.TempDir())

	_, ok, err := ls.ReconcileSlugOrphan("testproject", "never-ran")
	if err != nil {
		t.Fatalf("ReconcileSlugOrphan: %v", err)
	}
	if ok {
		t.Fatal("expected no orphan for a slug with no run logs")
	}
}

func TestReconcileSlugOrphan_IgnoresOtherSlugs(t *testing.T) {
	ls := NewLogStore(t.TempDir())
	const project = "testproject"

	seedOrphanLog(t, ls, project, "ff00000000000001", "other-pipe")

	_, ok, err := ls.ReconcileSlugOrphan(project, "stuck-pipe")
	if err != nil {
		t.Fatalf("ReconcileSlugOrphan: %v", err)
	}
	if ok {
		t.Fatal("expected no orphan for a slug with only another slug's runs")
	}

	// The other slug's run must be untouched (no .meta.json written).
	metaPath := ls.metaPath(project, "ff00000000000001")
	if _, err := os.Stat(metaPath); !os.IsNotExist(err) {
		t.Errorf("unrelated slug's run should not have been reconciled: %s", metaPath)
	}
}

func TestReconcileOrphanedRuns_SkipsActiveAndReconcilesRest(t *testing.T) {
	ls := NewLogStore(t.TempDir())
	const project = "testproject"

	seedOrphanLog(t, ls, project, "aa00000000000001", "pipe-a")
	seedOrphanLog(t, ls, project, "bb00000000000001", "pipe-b")

	isActive := func(runID string) bool { return runID == "aa00000000000001" }

	n, err := ls.ReconcileOrphanedRuns(project, isActive)
	if err != nil {
		t.Fatalf("ReconcileOrphanedRuns: %v", err)
	}
	if n != 1 {
		t.Fatalf("reconciled = %d, want 1", n)
	}

	// The active run must be untouched.
	if _, ok := ls.Record(project, "aa00000000000001"); ok {
		t.Error("active run should not have been reconciled")
	}
	// The orphaned run must now be terminal.
	rec, ok := ls.Record(project, "bb00000000000001")
	if !ok {
		t.Fatal("expected orphaned run to be reconciled")
	}
	if rec.Status != string(StepCancelled) {
		t.Errorf("Status = %q, want %q", rec.Status, StepCancelled)
	}
}

func TestReconcileOrphanedRuns_SkipsAlreadyTerminalRuns(t *testing.T) {
	ls := NewLogStore(t.TempDir())
	const project = "testproject"
	const runID = "cc00000000000001"

	ls.WriteEvent(project, runID, EventRunStarted, RunStartedPayload{
		RunID: runID, Pipeline: "pipe-c", Project: project,
	})
	ls.WriteEvent(project, runID, EventRunCompleted, RunCompletedPayload{
		RunID: runID, Pipeline: "pipe-c", Project: project,
		Status: "passed", DurationSeconds: 1.0,
	})
	if err := ls.WriteRecord(project, RunRecord{
		RunID: runID, Slug: "pipe-c",
		StartedAt: "2024-01-01T00:00:00Z", EndedAt: "2024-01-01T00:00:01Z",
		DurationMs: 1000, Status: "passed", LogRef: runID + ".log",
	}); err != nil {
		t.Fatalf("WriteRecord: %v", err)
	}

	n, err := ls.ReconcileOrphanedRuns(project, nil)
	if err != nil {
		t.Fatalf("ReconcileOrphanedRuns: %v", err)
	}
	if n != 0 {
		t.Errorf("reconciled = %d, want 0 (run already has a terminal record)", n)
	}

	rec, _ := ls.Record(project, runID)
	if rec.Status != "passed" {
		t.Errorf("Status was overwritten: got %q, want %q", rec.Status, "passed")
	}
}
