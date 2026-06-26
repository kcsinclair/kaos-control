// SPDX-License-Identifier: AGPL-3.0-or-later

package devops

// Milestone 1 — Unit tests: run record, back-fill, prune, corrupt-skip
//
// Fast filesystem-level tests for LogStore additions. Uses t.TempDir() and
// synthesized .log/.meta.json files — no HTTP, no Go build tags.
//
// Run with: go test ./internal/devops/ -run RunRecord

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestWriteRecord_AtomicAndReadable verifies that WriteRecord persists a
// RunRecord atomically: the .meta.json file exists, fields round-trip, and no
// leftover temp files remain.
func TestWriteRecord_AtomicAndReadable(t *testing.T) {
	ls := NewLogStore(t.TempDir())

	rec := RunRecord{
		RunID:      "abcdef0123456789",
		Slug:       "test-pipe",
		StartedAt:  "2024-01-01T00:00:00Z",
		EndedAt:    "2024-01-01T00:01:00Z",
		DurationMs: 60000,
		Status:     "passed",
		LogRef:     "abcdef0123456789.log",
	}
	if err := ls.WriteRecord("testproject", rec); err != nil {
		t.Fatalf("WriteRecord: %v", err)
	}

	metaPath := ls.metaPath("testproject", rec.RunID)

	// File must exist.
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Fatalf("meta.json not found at %s", metaPath)
	}

	// No leftover temp files in the directory.
	matches, _ := filepath.Glob(filepath.Join(filepath.Dir(metaPath), "*.tmp*"))
	if len(matches) > 0 {
		t.Errorf("leftover temp files after write: %v", matches)
	}

	// Round-trip: read back and verify all fields.
	got, ok := ls.Record("testproject", rec.RunID)
	if !ok {
		t.Fatal("Record not found after write")
	}
	if got.RunID != rec.RunID {
		t.Errorf("RunID: got %q, want %q", got.RunID, rec.RunID)
	}
	if got.Slug != rec.Slug {
		t.Errorf("Slug: got %q, want %q", got.Slug, rec.Slug)
	}
	if got.StartedAt != rec.StartedAt {
		t.Errorf("StartedAt: got %q, want %q", got.StartedAt, rec.StartedAt)
	}
	if got.EndedAt != rec.EndedAt {
		t.Errorf("EndedAt: got %q, want %q", got.EndedAt, rec.EndedAt)
	}
	if got.DurationMs != rec.DurationMs {
		t.Errorf("DurationMs: got %d, want %d", got.DurationMs, rec.DurationMs)
	}
	if got.Status != rec.Status {
		t.Errorf("Status: got %q, want %q", got.Status, rec.Status)
	}
}

// TestListPipelineRuns_NewestFirstAndFiltered seeds records for two pipeline
// slugs and asserts that only the requested slug is returned, ordered
// newest-first by StartedAt.
func TestListPipelineRuns_NewestFirstAndFiltered(t *testing.T) {
	ls := NewLogStore(t.TempDir())
	base := time.Now().UTC().Truncate(time.Second)

	// Seed 3 records for pipe-a (newest to oldest).
	for i := 0; i < 3; i++ {
		rec := RunRecord{
			RunID:      fmt.Sprintf("aaaa%012x", i),
			Slug:       "pipe-a",
			StartedAt:  base.Add(-time.Duration(i) * time.Minute).Format(time.RFC3339),
			EndedAt:    base.Add(-time.Duration(i)*time.Minute + 30*time.Second).Format(time.RFC3339),
			DurationMs: 30000,
			Status:     "passed",
			LogRef:     fmt.Sprintf("aaaa%012x.log", i),
		}
		if err := ls.WriteRecord("testproject", rec); err != nil {
			t.Fatalf("WriteRecord pipe-a[%d]: %v", i, err)
		}
	}

	// Seed 2 records for pipe-b — these must not appear in pipe-a results.
	for i := 0; i < 2; i++ {
		rec := RunRecord{
			RunID:      fmt.Sprintf("bbbb%012x", i),
			Slug:       "pipe-b",
			StartedAt:  base.Add(-time.Duration(i) * time.Minute).Format(time.RFC3339),
			EndedAt:    base.Add(-time.Duration(i)*time.Minute + 10*time.Second).Format(time.RFC3339),
			DurationMs: 10000,
			Status:     "failed",
			LogRef:     fmt.Sprintf("bbbb%012x.log", i),
		}
		if err := ls.WriteRecord("testproject", rec); err != nil {
			t.Fatalf("WriteRecord pipe-b[%d]: %v", i, err)
		}
	}

	recs, err := ls.ListPipelineRuns("testproject", "pipe-a", 0)
	if err != nil {
		t.Fatalf("ListPipelineRuns: %v", err)
	}
	if len(recs) != 3 {
		t.Fatalf("expected 3 records for pipe-a, got %d", len(recs))
	}
	// All returned records must belong to pipe-a.
	for _, r := range recs {
		if r.Slug != "pipe-a" {
			t.Errorf("unexpected slug %q in pipe-a results", r.Slug)
		}
	}
	// Records must be ordered newest-first (StartedAt descending).
	for i := 1; i < len(recs); i++ {
		if recs[i].StartedAt > recs[i-1].StartedAt {
			t.Errorf("records not newest-first: recs[%d].StartedAt=%q > recs[%d].StartedAt=%q",
				i, recs[i].StartedAt, i-1, recs[i-1].StartedAt)
		}
	}
}

// TestListPipelineRuns_SkipsCorruptRecord verifies that a truncated/garbage
// .meta.json file does not abort the listing: the valid record is returned
// and no error is propagated (NF3).
func TestListPipelineRuns_SkipsCorruptRecord(t *testing.T) {
	ls := NewLogStore(t.TempDir())

	// Write one valid record.
	valid := RunRecord{
		RunID:      "cccc000000000001",
		Slug:       "corrupt-test",
		StartedAt:  "2024-06-01T10:00:00Z",
		EndedAt:    "2024-06-01T10:01:00Z",
		DurationMs: 60000,
		Status:     "passed",
		LogRef:     "cccc000000000001.log",
	}
	if err := ls.WriteRecord("testproject", valid); err != nil {
		t.Fatalf("WriteRecord valid: %v", err)
	}

	// Manually write a corrupt .meta.json for a second runID.
	dir := ls.projectLogsDir("testproject")
	corruptPath := filepath.Join(dir, "cccc000000000002.meta.json")
	if err := os.WriteFile(corruptPath, []byte("{not valid json!!!"), 0o644); err != nil {
		t.Fatalf("writing corrupt meta.json: %v", err)
	}

	recs, err := ls.ListPipelineRuns("testproject", "corrupt-test", 0)
	if err != nil {
		t.Fatalf("ListPipelineRuns returned error: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("expected 1 valid record, got %d", len(recs))
	}
	if recs[0].RunID != valid.RunID {
		t.Errorf("returned record RunID = %q, want %q", recs[0].RunID, valid.RunID)
	}
}

// TestBackfill_FromLegacyLog verifies that a .log file without a .meta.json
// sidecar is back-filled by ListPipelineRuns: it derives a RunRecord, persists
// the .meta.json, and returns the record. An unparseable log is skipped
// without error (NF3).
func TestBackfill_FromLegacyLog(t *testing.T) {
	ls := NewLogStore(t.TempDir())

	legacyRunID := "dd00000000000001"
	const slug = "backfill-pipe"

	// Write run events via WriteEvent (same path as the runner uses).
	// This creates only the .log file — no sidecar yet.
	ls.WriteEvent("testproject", legacyRunID, EventRunStarted, RunStartedPayload{
		RunID:    legacyRunID,
		Pipeline: slug,
		Project:  "testproject",
	})
	ls.WriteEvent("testproject", legacyRunID, EventRunCompleted, RunCompletedPayload{
		RunID:           legacyRunID,
		Pipeline:        slug,
		Project:         "testproject",
		Status:          "passed",
		DurationSeconds: 3.5,
	})

	// Confirm no sidecar exists yet.
	metaPath := ls.metaPath("testproject", legacyRunID)
	if _, err := os.Stat(metaPath); !os.IsNotExist(err) {
		t.Fatalf("meta.json should not exist before back-fill, but found at %s", metaPath)
	}

	// ListPipelineRuns should back-fill from the log.
	recs, err := ls.ListPipelineRuns("testproject", slug, 0)
	if err != nil {
		t.Fatalf("ListPipelineRuns: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("expected 1 record after back-fill, got %d", len(recs))
	}
	r := recs[0]
	if r.RunID != legacyRunID {
		t.Errorf("RunID: got %q, want %q", r.RunID, legacyRunID)
	}
	if r.Slug != slug {
		t.Errorf("Slug: got %q, want %q", r.Slug, slug)
	}
	if r.Status != "passed" {
		t.Errorf("Status: got %q, want %q", r.Status, "passed")
	}

	// Sidecar must have been persisted.
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Error("back-fill did not persist .meta.json sidecar")
	}

	// Write an unparseable legacy log — ListPipelineRuns must skip it silently.
	badRunID := "dd00000000000002"
	badLogPath := ls.logPath("testproject", badRunID)
	if err := os.WriteFile(badLogPath, []byte("this is not json\nalso not json\n"), 0o644); err != nil {
		t.Fatalf("writing bad log: %v", err)
	}

	// Re-list: still only 1 valid record (bad log skipped).
	recs2, err := ls.ListPipelineRuns("testproject", slug, 0)
	if err != nil {
		t.Fatalf("ListPipelineRuns with bad log: %v", err)
	}
	if len(recs2) != 1 {
		t.Errorf("expected 1 record (bad log skipped), got %d", len(recs2))
	}
}

// TestPruneOldRuns_KeepsFiftyAndProtectsActive seeds 55 records for one slug,
// calls PruneOldRuns with keep=50, and asserts:
//   - Exactly 4 runs are removed (the 5 oldest minus the 1 protected active).
//   - Both .meta.json and .log of removed runs are deleted.
//   - The active run's files are preserved.
func TestPruneOldRuns_KeepsFiftyAndProtectsActive(t *testing.T) {
	ls := NewLogStore(t.TempDir())

	const slug = "prune-test"
	const total = 55
	const keep = 50

	base := time.Now().UTC().Truncate(time.Second)

	// Seed 55 records (index 0 = newest, index 54 = oldest) with .log sidecars.
	runIDs := make([]string, total)
	for i := 0; i < total; i++ {
		runIDs[i] = fmt.Sprintf("%016x", uint64(i))
		startedAt := base.Add(-time.Duration(i) * time.Minute)
		rec := RunRecord{
			RunID:      runIDs[i],
			Slug:       slug,
			StartedAt:  startedAt.Format(time.RFC3339),
			EndedAt:    startedAt.Add(30 * time.Second).Format(time.RFC3339),
			DurationMs: 30000,
			Status:     "passed",
			LogRef:     runIDs[i] + ".log",
		}
		if err := ls.WriteRecord("testproject", rec); err != nil {
			t.Fatalf("WriteRecord[%d]: %v", i, err)
		}
		// Create a corresponding empty .log file so PruneOldRuns can remove it.
		logPath := ls.logPath("testproject", runIDs[i])
		if err := os.WriteFile(logPath, []byte{}, 0o644); err != nil {
			t.Fatalf("creating log file[%d]: %v", i, err)
		}
	}

	// With keep=50, the 5 oldest runs (indices 50..54) are candidates for removal.
	// Mark the run at index 52 (third-oldest candidate) as active — must be spared.
	// After ListPipelineRuns sorts newest-first: result[52] = runIDs[52].
	activeRunID := runIDs[52]
	isActive := func(runID string) bool { return runID == activeRunID }

	removed, err := ls.PruneOldRuns("testproject", slug, keep, isActive)
	if err != nil {
		t.Fatalf("PruneOldRuns: %v", err)
	}
	// 5 candidates - 1 protected = 4 removed.
	if removed != 4 {
		t.Errorf("removed = %d, want 4", removed)
	}

	// The 50 kept runs (indices 0..49) must still have their files.
	for i := 0; i < keep; i++ {
		metaPath := ls.metaPath("testproject", runIDs[i])
		if _, err := os.Stat(metaPath); os.IsNotExist(err) {
			t.Errorf("kept run[%d] meta.json missing: %s", i, metaPath)
		}
		logPath := ls.logPath("testproject", runIDs[i])
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Errorf("kept run[%d] log missing: %s", i, logPath)
		}
	}

	// The 4 pruned runs (indices 50, 51, 53, 54) must be gone.
	pruned := []int{50, 51, 53, 54}
	for _, i := range pruned {
		metaPath := ls.metaPath("testproject", runIDs[i])
		if _, err := os.Stat(metaPath); !os.IsNotExist(err) {
			t.Errorf("pruned run[%d] meta.json still exists: %s", i, metaPath)
		}
		logPath := ls.logPath("testproject", runIDs[i])
		if _, err := os.Stat(logPath); !os.IsNotExist(err) {
			t.Errorf("pruned run[%d] log still exists: %s", i, logPath)
		}
	}

	// The protected active run (index 52) must still have its files.
	activeMeta := ls.metaPath("testproject", activeRunID)
	if _, err := os.Stat(activeMeta); os.IsNotExist(err) {
		t.Errorf("active run meta.json was incorrectly pruned: %s", activeMeta)
	}
	activeLog := ls.logPath("testproject", activeRunID)
	if _, err := os.Stat(activeLog); os.IsNotExist(err) {
		t.Errorf("active run log was incorrectly pruned: %s", activeLog)
	}
}

// seedRunRecord is a helper to write a RunRecord directly for integration
// scenarios within unit tests.
func seedRunRecord(t *testing.T, ls *LogStore, project string, rec RunRecord) {
	t.Helper()
	if err := ls.WriteRecord(project, rec); err != nil {
		t.Fatalf("seedRunRecord %q: %v", rec.RunID, err)
	}
}

// writeMinimalLogFile writes a minimal JSON-lines log to the given path,
// usable in unit tests that need a .log file to exist alongside a .meta.json.
func writeMinimalLogFile(t *testing.T, path, runID, slug string) {
	t.Helper()
	entry := logEntry{
		Time:      time.Now().UTC(),
		EventType: EventRunStarted,
		Payload: map[string]any{
			"run_id":       runID,
			"pipeline_slug": slug,
			"project":      "testproject",
		},
	}
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal log entry: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for log: %v", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write log file: %v", err)
	}
}
