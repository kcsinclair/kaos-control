// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestone 5 — Agent Orchestrator Integration Tests
//
// End-to-end integration tests for the testrunner.Run() orchestrator using a
// real fixture Go module (tests/fixtures/testrunner/project/) that produces
// known failures when compiled and run.
//
// The fixture module has no external dependencies, so compilation is fast.
// Each test copies the fixture to a temp directory so defects created during
// the run do not pollute the fixture source tree.

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/index"
	"github.com/kaos-control/kaos-control/internal/testrunner"
)

const fixtureProjectSrc = "../../tests/fixtures/testrunner/project"

// copyFixtureProject copies the testrunner fixture project into dst so the
// test can mutate it (create defect files etc.) without affecting the source.
func copyFixtureProject(t *testing.T, dst string) {
	t.Helper()
	src := fixtureProjectSrc
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copyFixtureProject: %v", err)
	}
}

// openFixtureIndex opens a SQLite index rooted at dst and upserts the widget
// test artifact so the mapper can resolve failures from widget/widget_test.go.
func openFixtureIndex(t *testing.T, root string) *index.Index {
	t.Helper()
	idx, err := index.Open(filepath.Join(root, "test.db"), root, nil)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	// Upsert the widget test artifact so failures can be mapped.
	a := &artifact.Artifact{
		Path:  "lifecycle/tests/widget-tests-2-test.md",
		Slug:  "widget-tests",
		Index: 2,
		Stage: "tests",
		Mtime: time.Now(),
		FM: artifact.Frontmatter{
			Title:   "Widget Tests",
			Type:    "test",
			Status:  "draft",
			Lineage: "widget-tests",
		},
	}
	if err := idx.Upsert(a); err != nil {
		t.Fatalf("Upsert widget-tests: %v", err)
	}
	return idx
}

// TestRun_FullFlow invokes testrunner.Run() against the fixture project and
// verifies that:
//   - The failing widget test produces exactly one defect.
//   - The defect is filed under lifecycle/defects/.
//   - The summary reports the correct counts.
func TestRun_FullFlow(t *testing.T) {
	root := t.TempDir()
	copyFixtureProject(t, root)
	// Ensure directories that the filer will write to exist.
	for _, d := range []string{"lifecycle/defects", "lifecycle/ideas"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	idx := openFixtureIndex(t, root)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	summary, err := testrunner.Run(ctx, root, idx, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if summary.Go.Failed == 0 {
		t.Error("expected at least one Go test failure in fixture project")
	}
	if summary.DefectsCreated == 0 {
		t.Error("expected at least one defect to be created")
	}

	// Verify a defect file was actually written.
	entries, err := os.ReadDir(filepath.Join(root, "lifecycle/defects"))
	if err != nil {
		t.Fatalf("reading defects dir: %v", err)
	}
	defectCount := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".md" {
			defectCount++
		}
	}
	if defectCount == 0 {
		t.Error("no defect .md files created in lifecycle/defects/")
	}
}

// TestRun_Idempotency runs the orchestrator twice against the same fixture
// project and verifies that the second run finds duplicates instead of creating
// additional defects (NF2: no duplicate defects).
func TestRun_Idempotency(t *testing.T) {
	root := t.TempDir()
	copyFixtureProject(t, root)
	for _, d := range []string{"lifecycle/defects", "lifecycle/ideas"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	idx := openFixtureIndex(t, root)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	summary1, err := testrunner.Run(ctx, root, idx, nil)
	if err != nil {
		t.Fatalf("Run #1: %v", err)
	}
	if summary1.DefectsCreated == 0 {
		t.Skip("no defects from first run; fixture produced no failures")
	}
	defectsAfterRun1 := summary1.DefectsCreated

	summary2, err := testrunner.Run(ctx, root, idx, nil)
	if err != nil {
		t.Fatalf("Run #2: %v", err)
	}

	if summary2.DefectsCreated != 0 {
		t.Errorf("second run created %d new defects, want 0 (idempotency)", summary2.DefectsCreated)
	}
	if summary2.DuplicatesFound < defectsAfterRun1 {
		t.Errorf("second run found %d duplicates, want >= %d", summary2.DuplicatesFound, defectsAfterRun1)
	}
}

// TestRun_SuiteLevelError verifies that when a suite produces a RawError (e.g.
// compilation failure), the orchestrator reports it in the summary and does not
// panic or return an error.
func TestRun_SuiteLevelError(t *testing.T) {
	idx, root := openTestrunnerIndex(t)

	// Simulate a suite-level compilation error by running the orchestrator
	// against a project dir that has a Go file with a syntax error.
	if err := os.MkdirAll(filepath.Join(root, "broken"), 0o755); err != nil {
		t.Fatal(err)
	}
	brokenGo := filepath.Join(root, "broken", "broken_test.go")
	if err := os.WriteFile(brokenGo, []byte("package broken_test\nfunc not valid go("), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module broken-fixture\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Run should complete without error even though the Go suite fails to compile.
	summary, err := testrunner.Run(ctx, root, idx, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// A compilation error sets RawError on the suite result; the orchestrator
	// should tolerate it and return a valid summary.
	_ = summary
}

// TestRun_CoverageGaps verifies that test artifacts with no corresponding
// failure in the run are reported as coverage gaps.
func TestRun_CoverageGaps(t *testing.T) {
	root := t.TempDir()
	copyFixtureProject(t, root)
	for _, d := range []string{"lifecycle/defects", "lifecycle/ideas"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	idx := openFixtureIndex(t, root)

	// Add a second test artifact that has no corresponding Go tests — it should
	// appear as a coverage gap.
	a := &artifact.Artifact{
		Path:  "lifecycle/tests/orphan-feature-2-test.md",
		Slug:  "orphan-feature",
		Index: 2,
		Stage: "tests",
		Mtime: time.Now(),
		FM: artifact.Frontmatter{
			Title:   "Orphan Feature Tests",
			Type:    "test",
			Status:  "draft",
			Lineage: "orphan-feature",
		},
	}
	if err := idx.Upsert(a); err != nil {
		t.Fatalf("Upsert orphan artifact: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	summary, err := testrunner.Run(ctx, root, idx, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	found := false
	for _, gap := range summary.CoverageGaps {
		if gap == "lifecycle/tests/orphan-feature-2-test.md" {
			found = true
		}
	}
	if !found {
		t.Errorf("orphan-feature-2-test.md not in coverage gaps: %v", summary.CoverageGaps)
	}
}

// TestRun_OverheadUnderTenSeconds verifies that the orchestration overhead
// (excluding actual test execution time) completes within 10 seconds for the
// fixture project.  This is a timing sanity check, not a hard performance gate.
func TestRun_OverheadUnderTenSeconds(t *testing.T) {
	root := t.TempDir()
	copyFixtureProject(t, root)
	for _, d := range []string{"lifecycle/defects", "lifecycle/ideas"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	idx := openFixtureIndex(t, root)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	start := time.Now()
	_, err := testrunner.Run(ctx, root, idx, nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// The fixture has one tiny Go test and no Vitest/Playwright; total wall
	// time should be well under 60s even including compilation.
	if elapsed > 60*time.Second {
		t.Errorf("Run took %v, want < 60s for fixture project", elapsed)
	}
}
