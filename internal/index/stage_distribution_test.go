// SPDX-License-Identifier: AGPL-3.0-or-later

package index

// Milestone 2 — Unit tests for the StageDistribution index method.
//
// These tests open a bare in-memory SQLite index and insert rows directly
// via Upsert (using synthetic Artifact values) to exercise StageDistribution
// in isolation from the HTTP layer.

import (
	"os"
	"path/filepath"
	"testing"
)

// openStageDistIndex opens a minimal temp-dir SQLite index suitable for
// StageDistribution unit tests. No hub or workflow wiring is needed.
func openStageDistIndex(t *testing.T) (*Index, string) {
	t.Helper()
	dir := t.TempDir()
	// IndexFile only processes files under lifecycle/.
	for _, sub := range []string{"ideas", "requirements", "backend-plans", "frontend-plans"} {
		if err := os.MkdirAll(filepath.Join(dir, "lifecycle", sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	idx, err := Open(dir+"/test.db", dir, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { idx.Close() })
	return idx, dir
}

// upsertTestArtifact writes a minimal markdown file to disk and calls
// IndexFile so the artifact ends up in the index with the correct stage
// (derived from its directory path).
func upsertTestArtifact(t *testing.T, idx *Index, dir, relPath, typ, status string) {
	t.Helper()
	content := "---\ntitle: Test\ntype: " + typ + "\nstatus: " + status + "\nlineage: test\n---\n\nBody.\n"
	absPath := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := idx.IndexFile(absPath); err != nil {
		t.Fatalf("IndexFile(%s): %v", relPath, err)
	}
}

// TestStageDistribution_CorrectGrouping inserts artifacts into multiple stages
// and verifies StageDistribution returns the correct counts per stage.
func TestStageDistribution_CorrectGrouping(t *testing.T) {
	idx, dir := openStageDistIndex(t)

	upsertTestArtifact(t, idx, dir, "lifecycle/ideas/a1.md", "ticket", "draft")
	upsertTestArtifact(t, idx, dir, "lifecycle/ideas/a2.md", "ticket", "draft")
	upsertTestArtifact(t, idx, dir, "lifecycle/requirements/b1.md", "ticket", "planning")
	upsertTestArtifact(t, idx, dir, "lifecycle/backend-plans/c1.md", "ticket", "in-development")
	upsertTestArtifact(t, idx, dir, "lifecycle/backend-plans/c2.md", "ticket", "in-development")
	upsertTestArtifact(t, idx, dir, "lifecycle/backend-plans/c3.md", "ticket", "in-development")

	counts, err := idx.StageDistribution([]string{"ticket"})
	if err != nil {
		t.Fatalf("StageDistribution: %v", err)
	}

	byStage := make(map[string]int, len(counts))
	for _, sc := range counts {
		byStage[sc.Stage] = sc.Count
	}

	if byStage["ideas"] != 2 {
		t.Errorf("ideas: want 2, got %d", byStage["ideas"])
	}
	if byStage["requirements"] != 1 {
		t.Errorf("requirements: want 1, got %d", byStage["requirements"])
	}
	if byStage["backend-plans"] != 3 {
		t.Errorf("backend-plans: want 3, got %d", byStage["backend-plans"])
	}
}

// TestStageDistribution_EmptyDatabase verifies that StageDistribution returns
// a non-nil empty slice (not nil) when no matching artifacts exist.
func TestStageDistribution_EmptyDatabase(t *testing.T) {
	idx, _ := openStageDistIndex(t)

	counts, err := idx.StageDistribution([]string{"ticket"})
	if err != nil {
		t.Fatalf("StageDistribution: %v", err)
	}
	if counts == nil {
		t.Fatal("expected non-nil slice, got nil")
	}
	if len(counts) != 0 {
		t.Errorf("expected empty slice, got %d entries", len(counts))
	}
}

// TestStageDistribution_TrackedTypesDefault verifies that nil/empty trackedTypes
// falls back to ["ticket"] — only ticket-type artifacts are counted.
func TestStageDistribution_TrackedTypesDefault(t *testing.T) {
	idx, dir := openStageDistIndex(t)

	// 2 tickets (should be counted with the default fallback)
	upsertTestArtifact(t, idx, dir, "lifecycle/ideas/def-ticket-1.md", "ticket", "draft")
	upsertTestArtifact(t, idx, dir, "lifecycle/ideas/def-ticket-2.md", "ticket", "planning")
	// 1 plan-backend (should NOT be counted with nil tracked types)
	upsertTestArtifact(t, idx, dir, "lifecycle/backend-plans/def-plan-1.md", "plan-backend", "draft")

	countsNil, err := idx.StageDistribution(nil)
	if err != nil {
		t.Fatalf("StageDistribution(nil): %v", err)
	}

	countsEmpty, err := idx.StageDistribution([]string{})
	if err != nil {
		t.Fatalf("StageDistribution([]): %v", err)
	}

	for _, counts := range [][]StageCount{countsNil, countsEmpty} {
		byStage := make(map[string]int, len(counts))
		for _, sc := range counts {
			byStage[sc.Stage] = sc.Count
		}
		if byStage["ideas"] != 2 {
			t.Errorf("ideas: want 2, got %d (nil/empty tracked types should default to ticket)", byStage["ideas"])
		}
		if byStage["backend-plans"] != 0 {
			t.Errorf("backend-plans: want 0, got %d (plan-backend should be excluded by default)", byStage["backend-plans"])
		}
	}
}

// TestStageDistribution_StatusExclusion verifies that artifacts with status
// "done" or "abandoned" are excluded from the stage counts.
func TestStageDistribution_StatusExclusion(t *testing.T) {
	idx, dir := openStageDistIndex(t)

	// Excluded statuses
	upsertTestArtifact(t, idx, dir, "lifecycle/requirements/excl-done-1.md", "ticket", "done")
	upsertTestArtifact(t, idx, dir, "lifecycle/requirements/excl-done-2.md", "ticket", "done")
	upsertTestArtifact(t, idx, dir, "lifecycle/ideas/excl-aband-1.md", "ticket", "abandoned")

	// One visible ticket
	upsertTestArtifact(t, idx, dir, "lifecycle/requirements/excl-visible-1.md", "ticket", "planning")

	counts, err := idx.StageDistribution([]string{"ticket"})
	if err != nil {
		t.Fatalf("StageDistribution: %v", err)
	}

	byStage := make(map[string]int, len(counts))
	for _, sc := range counts {
		byStage[sc.Stage] = sc.Count
	}

	// requirements should count only the planning ticket.
	if byStage["requirements"] != 1 {
		t.Errorf("requirements: want 1, got %d (done tickets must be excluded)", byStage["requirements"])
	}
	// ideas should not appear (only abandoned artifact).
	if byStage["ideas"] != 0 {
		t.Errorf("ideas: want 0, got %d (abandoned ticket must be excluded)", byStage["ideas"])
	}
	// Exactly one entry in the result.
	if len(counts) != 1 {
		t.Errorf("expected 1 result entry, got %d", len(counts))
	}
}

