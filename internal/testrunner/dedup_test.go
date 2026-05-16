// SPDX-License-Identifier: AGPL-3.0-or-later

package testrunner

import (
	"os"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/index"
)

// openDedupIndex creates a minimal SQLite index in a temp dir.
func openDedupIndex(t *testing.T) (*index.Index, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(dir+"/lifecycle/defects", 0o755); err != nil {
		t.Fatal(err)
	}
	idx, err := index.Open(dir+"/test.db", dir, nil)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { idx.Close() })
	return idx, dir
}

// upsertDefect inserts a defect artifact into the index with given status and labels.
func upsertDefect(t *testing.T, idx *index.Index, path, lineage, status string, labels []string) {
	t.Helper()
	a := &artifact.Artifact{
		Path:  path,
		Slug:  "defect",
		Index: 7,
		Stage: "defects",
		Mtime: time.Now(),
		FM: artifact.Frontmatter{
			Title:   "Some defect",
			Type:    "defect",
			Status:  status,
			Lineage: lineage,
			Labels:  labels,
		},
	}
	if err := idx.Upsert(a); err != nil {
		t.Fatalf("Upsert(%s): %v", path, err)
	}
}

func TestFindDuplicate_ByTestLabel(t *testing.T) {
	idx, _ := openDedupIndex(t)

	f := TestFailure{
		Suite:    "go",
		Package:  "github.com/foo/bar",
		TestName: "TestFoo",
		File:     "foo_test.go",
		Line:     42,
		ErrorMsg: "expected 1, got 2",
	}

	label := autoTestLabel(f)
	upsertDefect(t, idx,
		"lifecycle/defects/mylineage-7-defect.md",
		"mylineage", "draft",
		[]string{"defect", "auto-filed", label})

	d := NewDeduplicator(idx)
	dup, err := d.FindDuplicate(f, "mylineage")
	if err != nil {
		t.Fatalf("FindDuplicate: %v", err)
	}
	if dup == nil {
		t.Fatal("expected duplicate to be found by test label")
	}
}

func TestFindDuplicate_ByLocLabel(t *testing.T) {
	idx, _ := openDedupIndex(t)

	f := TestFailure{
		Suite:    "go",
		Package:  "github.com/foo/bar",
		TestName: "TestBar",
		File:     "bar_test.go",
		Line:     99,
		ErrorMsg: "some error",
	}

	locLabel := autoLocLabel(f)
	upsertDefect(t, idx,
		"lifecycle/defects/mylineage-8-defect.md",
		"mylineage", "in-development",
		[]string{"defect", "auto-filed", locLabel})

	d := NewDeduplicator(idx)
	// Use a different test name so the label-based lookup misses.
	fDifferentName := f
	fDifferentName.TestName = "TestCompletlyDifferent"
	dup, err := d.FindDuplicate(fDifferentName, "mylineage")
	if err != nil {
		t.Fatalf("FindDuplicate: %v", err)
	}
	if dup == nil {
		t.Fatal("expected duplicate to be found by location label")
	}
}

func TestFindDuplicate_ClosedNotDuplicate(t *testing.T) {
	idx, _ := openDedupIndex(t)

	f := TestFailure{
		Suite:    "go",
		Package:  "github.com/foo/bar",
		TestName: "TestFoo",
		File:     "foo_test.go",
		Line:     42,
		ErrorMsg: "expected 1, got 2",
	}

	label := autoTestLabel(f)
	// Insert as "done" — should not count as a duplicate.
	upsertDefect(t, idx,
		"lifecycle/defects/mylineage-7-defect.md",
		"mylineage", "done",
		[]string{"defect", "auto-filed", label})

	d := NewDeduplicator(idx)
	dup, err := d.FindDuplicate(f, "mylineage")
	if err != nil {
		t.Fatalf("FindDuplicate: %v", err)
	}
	if dup != nil {
		t.Error("closed (done) defect should not be reported as a duplicate")
	}
}

func TestFindDuplicate_AbandonedNotDuplicate(t *testing.T) {
	idx, _ := openDedupIndex(t)

	f := TestFailure{
		Suite:    "go",
		Package:  "github.com/foo/bar",
		TestName: "TestFoo",
		File:     "foo_test.go",
		Line:     42,
	}

	label := autoTestLabel(f)
	upsertDefect(t, idx,
		"lifecycle/defects/mylineage-7-defect.md",
		"mylineage", "abandoned",
		[]string{"defect", "auto-filed", label})

	d := NewDeduplicator(idx)
	dup, err := d.FindDuplicate(f, "mylineage")
	if err != nil {
		t.Fatalf("FindDuplicate: %v", err)
	}
	if dup != nil {
		t.Error("abandoned defect should not be reported as a duplicate")
	}
}

func TestGroupByAssertion_SameLocation(t *testing.T) {
	d := NewDeduplicator(nil)

	failures := []TestFailure{
		{TestName: "TestA", File: "foo_test.go", Line: 42},
		{TestName: "TestB", File: "foo_test.go", Line: 42},
		{TestName: "TestC", File: "foo_test.go", Line: 42},
		{TestName: "TestD", File: "foo_test.go", Line: 42},
		{TestName: "TestE", File: "foo_test.go", Line: 42},
	}

	groups := d.GroupByAssertion(failures)
	if len(groups) != 1 {
		t.Errorf("expected 1 group for same file:line, got %d", len(groups))
	}
	if len(groups[0]) != 5 {
		t.Errorf("expected 5 failures in group, got %d", len(groups[0]))
	}
}

func TestGroupByAssertion_DifferentLocations(t *testing.T) {
	d := NewDeduplicator(nil)

	failures := []TestFailure{
		{TestName: "TestA", File: "foo_test.go", Line: 10},
		{TestName: "TestB", File: "foo_test.go", Line: 20},
		{TestName: "TestC", File: "bar_test.go", Line: 10},
	}

	groups := d.GroupByAssertion(failures)
	if len(groups) != 3 {
		t.Errorf("expected 3 groups, got %d", len(groups))
	}
}

func TestGroupByAssertion_UnknownLineInOwnGroup(t *testing.T) {
	d := NewDeduplicator(nil)

	failures := []TestFailure{
		{TestName: "TestA", File: "", Line: 0},
		{TestName: "TestB", File: "", Line: 0},
	}

	groups := d.GroupByAssertion(failures)
	// Failures with Line==0 each get their own group.
	if len(groups) != 2 {
		t.Errorf("expected 2 singleton groups for Line==0 failures, got %d", len(groups))
	}
}

func TestNormaliseError(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: "error at 2024-01-15 in object 0xdeadbeef",
			want:  "error at <date> in object <ptr>",
		},
		{
			input: "uuid was 550e8400-e29b-41d4-a716-446655440000 again",
			want:  "uuid was <uuid> again",
		},
		{
			input: "short msg",
			want:  "short msg",
		},
		{
			// Verify truncation to 100 chars.
			input: "a very long error message that exceeds one hundred characters and should be truncated to exactly 100 chars here EXTRA",
			want:  "a very long error message that exceeds one hundred characters and should be truncated to exactly 100",
		},
	}

	for _, tc := range tests {
		got := NormaliseError(tc.input)
		if got != tc.want {
			t.Errorf("NormaliseError(%q)\n got:  %q\n want: %q", tc.input, got, tc.want)
		}
	}
}

func TestNormaliseError_SimilarMessages(t *testing.T) {
	// Two messages differing only in timestamp should normalise to the same string.
	m1 := NormaliseError("timeout waiting for response at 2024-01-15 in handler foo")
	m2 := NormaliseError("timeout waiting for response at 2025-03-22 in handler foo")
	if m1 != m2 {
		t.Errorf("expected identical normalised strings:\n  %q\n  %q", m1, m2)
	}
}
