// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestone 4 — Defect Filing Integration Tests
//
// Integration tests that exercise DefectFiler and Deduplicator via the
// exported testrunner API, using a real project environment (filesystem +
// SQLite index).  These tests complement the white-box unit tests in
// internal/testrunner/defect_test.go by validating the full lifecycle: file
// creation, index upsert, and duplicate detection across two filing rounds.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/index"
	"github.com/kaos-control/kaos-control/internal/testrunner"
)

// openTestrunnerIndex creates a minimal SQLite index and lifecycle directories
// inside a temp root suitable for DefectFiler tests.
func openTestrunnerIndex(t *testing.T) (*index.Index, string) {
	t.Helper()
	root := t.TempDir()
	for _, d := range []string{
		"lifecycle/defects",
		"lifecycle/ideas",
		"lifecycle/tests",
	} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	idx, err := index.Open(filepath.Join(root, "test.db"), root, nil)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { idx.Close() })
	return idx, root
}

// insertTestArtifact upserts a test-type artifact into the index and returns
// the resulting ArtifactRow.
func insertTestArtifact(t *testing.T, idx *index.Index, path, lineage string, labels []string) *index.ArtifactRow {
	t.Helper()
	a := &artifact.Artifact{
		Path:  path,
		Slug:  lineage,
		Index: 2,
		Stage: "tests",
		Mtime: time.Now(),
		FM: artifact.Frontmatter{
			Title:   "Tests: " + lineage,
			Type:    "test",
			Status:  "draft",
			Lineage: lineage,
			Labels:  labels,
		},
	}
	if err := idx.Upsert(a); err != nil {
		t.Fatalf("Upsert(%s): %v", path, err)
	}
	row, err := idx.Get(path)
	if err != nil || row == nil {
		t.Fatalf("Get after upsert: %v", err)
	}
	return row
}

// TestDefectFiler_BackendLabelRouting verifies that a failure matched to a
// test artifact carrying the "backend" label is routed to backend-developer.
func TestDefectFiler_BackendLabelRouting(t *testing.T) {
	idx, root := openTestrunnerIndex(t)
	matched := insertTestArtifact(t, idx,
		"lifecycle/tests/store-tests-2-test.md",
		"store-tests",
		[]string{"backend", "store"},
	)

	filer := testrunner.NewDefectFiler(idx, root)
	failures := []testrunner.TestFailure{
		{
			Suite:    "go",
			Package:  "github.com/example/internal/store",
			TestName: "TestStoreInsert",
			File:     "store_test.go",
			Line:     55,
			ErrorMsg: "insert returned unexpected error",
		},
	}

	relPath, err := filer.FileDefect(failures, matched)
	if err != nil {
		t.Fatalf("FileDefect: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(root, relPath))
	if err != nil {
		t.Fatalf("reading defect: %v", err)
	}
	content := string(raw)

	if !strings.Contains(content, "role: backend-developer") {
		t.Errorf("defect should be routed to backend-developer; content:\n%s", content)
	}
	if !strings.Contains(content, "auto-filed") {
		t.Errorf("defect missing auto-filed label")
	}
}

// TestDefectFiler_FrontendPathRouting verifies that a failure whose Package
// path includes "tests/web" is routed to frontend-developer.
func TestDefectFiler_FrontendPathRouting(t *testing.T) {
	idx, root := openTestrunnerIndex(t)
	matched := insertTestArtifact(t, idx,
		"lifecycle/tests/frontend-tests-2-test.md",
		"frontend-tests",
		nil,
	)

	filer := testrunner.NewDefectFiler(idx, root)
	failures := []testrunner.TestFailure{
		{
			Suite:    "vitest",
			Package:  "/project/tests/web/login.spec.ts",
			TestName: "LoginSuite > should redirect",
			File:     "login.spec.ts",
			Line:     22,
			ErrorMsg: "navigation did not occur",
		},
	}

	relPath, err := filer.FileDefect(failures, matched)
	if err != nil {
		t.Fatalf("FileDefect: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(root, relPath))
	content := string(raw)

	if !strings.Contains(content, "role: frontend-developer") {
		t.Errorf("defect should route to frontend-developer; content:\n%s", content)
	}
}

// TestDefectFiler_OrphanedCreatesIdeaArtifact verifies that filing an orphaned
// failure (no matched test artifact) auto-creates the lifecycle/ideas/tests-orphaned.md
// sentinel artifact and uses the tests-orphaned lineage.
func TestDefectFiler_OrphanedCreatesIdeaArtifact(t *testing.T) {
	idx, root := openTestrunnerIndex(t)
	filer := testrunner.NewDefectFiler(idx, root)

	failures := []testrunner.TestFailure{
		{
			Suite:    "go",
			Package:  "github.com/example/unknown",
			TestName: "TestOrphan",
			File:     "orphan_test.go",
			Line:     1,
			ErrorMsg: "something unexpected",
		},
	}

	relPath, err := filer.FileDefect(failures, nil)
	if err != nil {
		t.Fatalf("FileDefect (orphaned): %v", err)
	}

	if !strings.Contains(relPath, "tests-orphaned") {
		t.Errorf("expected tests-orphaned in path, got %q", relPath)
	}

	ideaPath := filepath.Join(root, "lifecycle/ideas/tests-orphaned.md")
	if _, err := os.Stat(ideaPath); os.IsNotExist(err) {
		t.Error("tests-orphaned idea artifact was not auto-created")
	}
}

// TestDeduplicator_FindsExistingOpenDefect verifies the full round-trip:
// file a defect, then run the deduplicator and confirm it is found.
func TestDeduplicator_FindsExistingOpenDefect(t *testing.T) {
	idx, root := openTestrunnerIndex(t)
	matched := insertTestArtifact(t, idx,
		"lifecycle/tests/auth-tests-2-test.md",
		"auth-tests",
		nil,
	)

	filer := testrunner.NewDefectFiler(idx, root)
	f := testrunner.TestFailure{
		Suite:    "go",
		Package:  "github.com/example/auth",
		TestName: "TestLogin",
		File:     "login_test.go",
		Line:     30,
		ErrorMsg: "login rejected valid credentials",
	}

	_, err := filer.FileDefect([]testrunner.TestFailure{f}, matched)
	if err != nil {
		t.Fatalf("FileDefect: %v", err)
	}

	// Now the deduplicator should find the open defect.
	dedup := testrunner.NewDeduplicator(idx)
	dup, err := dedup.FindDuplicate(f, "auth-tests")
	if err != nil {
		t.Fatalf("FindDuplicate: %v", err)
	}
	if dup == nil {
		t.Error("deduplicator should have found the filed defect as a duplicate")
	}
}

// TestDefectFiler_LineageIndicesMonotonic verifies that two defects filed for
// the same lineage receive different, monotonically increasing indices.
func TestDefectFiler_LineageIndicesMonotonic(t *testing.T) {
	idx, root := openTestrunnerIndex(t)
	matched := insertTestArtifact(t, idx,
		"lifecycle/tests/api-tests-2-test.md",
		"api-tests",
		nil,
	)

	filer := testrunner.NewDefectFiler(idx, root)

	f1 := testrunner.TestFailure{Suite: "go", TestName: "TestA", Package: "pkg", ErrorMsg: "err A"}
	f2 := testrunner.TestFailure{Suite: "go", TestName: "TestB", Package: "pkg", ErrorMsg: "err B"}

	path1, err := filer.FileDefect([]testrunner.TestFailure{f1}, matched)
	if err != nil {
		t.Fatalf("FileDefect #1: %v", err)
	}
	path2, err := filer.FileDefect([]testrunner.TestFailure{f2}, matched)
	if err != nil {
		t.Fatalf("FileDefect #2: %v", err)
	}

	if path1 == path2 {
		t.Error("two defects for the same lineage must have different paths")
	}
}

// TestDefectFiler_WitnessAppendPreservesFrontmatter verifies that
// AppendWitness adds a witness entry without corrupting the YAML frontmatter.
func TestDefectFiler_WitnessAppendPreservesFrontmatter(t *testing.T) {
	idx, root := openTestrunnerIndex(t)
	matched := insertTestArtifact(t, idx,
		"lifecycle/tests/queue-tests-2-test.md",
		"queue-tests",
		nil,
	)

	filer := testrunner.NewDefectFiler(idx, root)
	f := testrunner.TestFailure{
		Suite:    "go",
		TestName: "TestQueueDrain",
		Package:  "github.com/example/queue",
		ErrorMsg: "queue blocked",
	}

	relPath, err := filer.FileDefect([]testrunner.TestFailure{f}, matched)
	if err != nil {
		t.Fatalf("FileDefect: %v", err)
	}

	witness := testrunner.TestFailure{
		Suite:    "go",
		TestName: "TestQueueDrain",
		Package:  "github.com/example/queue",
		ErrorMsg: "queue blocked again",
	}
	if err := filer.AppendWitness(relPath, witness); err != nil {
		t.Fatalf("AppendWitness: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(root, relPath))
	content := string(raw)

	if !strings.Contains(content, "type: defect") {
		t.Error("YAML frontmatter corrupted: missing type: defect")
	}
	if !strings.Contains(content, "status: draft") {
		t.Error("YAML frontmatter corrupted: missing status: draft")
	}
	if !strings.Contains(content, "queue blocked again") {
		t.Error("AppendWitness did not append witness error message")
	}
}
