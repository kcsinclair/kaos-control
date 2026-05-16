// SPDX-License-Identifier: AGPL-3.0-or-later

package testrunner

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/index"
)

// openDefectIndex opens a minimal index and creates the lifecycle/defects dir.
func openDefectIndex(t *testing.T) (*index.Index, string) {
	t.Helper()
	dir := t.TempDir()
	for _, d := range []string{"lifecycle/defects", "lifecycle/ideas", "lifecycle/tests"} {
		if err := os.MkdirAll(dir+"/"+d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	idx, err := index.Open(dir+"/test.db", dir, nil)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { idx.Close() })
	return idx, dir
}

// upsertTestArtifactForDefect inserts a test artifact into the index.
func upsertTestArtifactForDefect(t *testing.T, idx *index.Index, path, lineage string) *index.ArtifactRow {
	t.Helper()
	a := &artifact.Artifact{
		Path:  path,
		Slug:  "my-feature",
		Index: 6,
		Stage: "tests",
		Mtime: time.Now(),
		FM: artifact.Frontmatter{
			Title:   "Tests for " + lineage,
			Type:    "test",
			Status:  "draft",
			Lineage: lineage,
		},
	}
	if err := idx.Upsert(a); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	row, err := idx.Get(path)
	if err != nil || row == nil {
		t.Fatalf("Get after upsert: %v", err)
	}
	return row
}

func TestDefectFiler_FileDefect_Basic(t *testing.T) {
	idx, dir := openDefectIndex(t)
	df := NewDefectFiler(idx, dir)

	matched := upsertTestArtifactForDefect(t, idx, "lifecycle/tests/my-feature-6-test.md", "my-feature")

	failures := []TestFailure{
		{
			Suite:    "go",
			Package:  "github.com/foo/bar/internal/myfeature",
			TestName: "TestFoo",
			File:     "myfeature_test.go",
			Line:     42,
			ErrorMsg: "expected 1, got 2",
			Output:   "    myfeature_test.go:42: expected 1, got 2\n",
			Elapsed:  0.01,
		},
	}

	relPath, err := df.FileDefect(failures, matched)
	if err != nil {
		t.Fatalf("FileDefect: %v", err)
	}

	// Verify file was created.
	if !strings.HasPrefix(relPath, "lifecycle/defects/") {
		t.Errorf("relPath = %q, expected lifecycle/defects/ prefix", relPath)
	}
	absPath := dir + "/" + relPath
	raw, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading created defect: %v", err)
	}
	content := string(raw)

	// Check required frontmatter fields.
	if !strings.Contains(content, "type: defect") {
		t.Error("defect missing 'type: defect'")
	}
	if !strings.Contains(content, "status: draft") {
		t.Error("defect missing 'status: draft'")
	}
	if !strings.Contains(content, "lineage: my-feature") {
		t.Error("defect missing lineage")
	}
	if !strings.Contains(content, "parent: lifecycle/tests/my-feature-6-test.md") {
		t.Error("defect missing parent")
	}
	if !strings.Contains(content, "auto-filed") {
		t.Error("defect missing auto-filed label")
	}
	if !strings.Contains(content, "role: backend-developer") {
		t.Error("defect missing role: backend-developer")
	}
	if !strings.Contains(content, "who: agent") {
		t.Error("defect missing who: agent")
	}

	// Check body content.
	if !strings.Contains(content, "TestFoo") {
		t.Error("defect body missing test name")
	}
	if !strings.Contains(content, "expected 1, got 2") {
		t.Error("defect body missing error message")
	}
}

func TestDefectFiler_FileDefect_Orphaned(t *testing.T) {
	idx, dir := openDefectIndex(t)
	df := NewDefectFiler(idx, dir)

	failures := []TestFailure{
		{
			Suite:    "go",
			Package:  "github.com/foo/bar/orphaned",
			TestName: "TestOrphan",
			File:     "orphaned_test.go",
			Line:     1,
			ErrorMsg: "orphan error",
		},
	}

	relPath, err := df.FileDefect(failures, nil)
	if err != nil {
		t.Fatalf("FileDefect (orphaned): %v", err)
	}

	if !strings.Contains(relPath, "tests-orphaned") {
		t.Errorf("expected tests-orphaned lineage in path, got %q", relPath)
	}

	// The idea artifact should also have been auto-created.
	if _, err := os.Stat(dir + "/lifecycle/ideas/tests-orphaned.md"); err != nil {
		t.Error("tests-orphaned idea artifact was not created")
	}
}

func TestDefectFiler_FileDefect_GroupedFailures(t *testing.T) {
	idx, dir := openDefectIndex(t)
	df := NewDefectFiler(idx, dir)

	matched := upsertTestArtifactForDefect(t, idx, "lifecycle/tests/my-feature-6-test.md", "my-feature")

	failures := []TestFailure{
		{Suite: "go", TestName: "TestA", File: "f_test.go", Line: 5, ErrorMsg: "err"},
		{Suite: "go", TestName: "TestB", File: "f_test.go", Line: 5, ErrorMsg: "err"},
		{Suite: "go", TestName: "TestC", File: "f_test.go", Line: 5, ErrorMsg: "err"},
	}

	relPath, err := df.FileDefect(failures, matched)
	if err != nil {
		t.Fatalf("FileDefect (grouped): %v", err)
	}

	raw, _ := os.ReadFile(dir + "/" + relPath)
	content := string(raw)

	// All three test names should appear in the witnesses section.
	for _, name := range []string{"TestA", "TestB", "TestC"} {
		if !strings.Contains(content, name) {
			t.Errorf("grouped defect missing witness %q", name)
		}
	}
}

func TestDefectFiler_AppendWitness(t *testing.T) {
	idx, dir := openDefectIndex(t)
	df := NewDefectFiler(idx, dir)

	matched := upsertTestArtifactForDefect(t, idx, "lifecycle/tests/my-feature-6-test.md", "my-feature")

	failures := []TestFailure{
		{Suite: "go", TestName: "TestFoo", File: "foo_test.go", Line: 10, ErrorMsg: "error"},
	}

	relPath, err := df.FileDefect(failures, matched)
	if err != nil {
		t.Fatalf("FileDefect: %v", err)
	}

	// Append a witness entry.
	witness := TestFailure{
		Suite:    "go",
		TestName: "TestFoo",
		File:     "foo_test.go",
		Line:     10,
		ErrorMsg: "same error again",
	}
	if err := df.AppendWitness(relPath, witness); err != nil {
		t.Fatalf("AppendWitness: %v", err)
	}

	raw, _ := os.ReadFile(dir + "/" + relPath)
	content := string(raw)

	if !strings.Contains(content, "Witness") {
		t.Error("AppendWitness did not add witness section")
	}
	if !strings.Contains(content, "same error again") {
		t.Error("AppendWitness did not include error message")
	}

	// Verify frontmatter was not corrupted.
	if !strings.Contains(content, "type: defect") {
		t.Error("frontmatter corrupted after AppendWitness")
	}
}

func TestRouteRole_BackendPath(t *testing.T) {
	df := &DefectFiler{}
	f := TestFailure{Package: "github.com/foo/bar/internal/store"}
	role := df.routeRole(f, nil)
	if role != "backend-developer" {
		t.Errorf("role = %q, want backend-developer", role)
	}
}

func TestRouteRole_FrontendPath(t *testing.T) {
	df := &DefectFiler{}
	f := TestFailure{Package: "/project/tests/web/login.spec.ts"}
	role := df.routeRole(f, nil)
	if role != "frontend-developer" {
		t.Errorf("role = %q, want frontend-developer", role)
	}
}

func TestBuildTitle(t *testing.T) {
	f := TestFailure{
		TestName: "TestFoo",
		ErrorMsg: "expected 1, got 2",
	}
	title := buildTitle(f)
	if !strings.Contains(title, "TestFoo") {
		t.Error("title missing test name")
	}
	if !strings.Contains(title, "expected 1, got 2") {
		t.Error("title missing error message")
	}
}

func TestBuildReproduction(t *testing.T) {
	tests := []struct {
		f    TestFailure
		want string
	}{
		{
			f:    TestFailure{Suite: "go", TestName: "TestFoo", Package: "github.com/foo/bar"},
			want: "go test -run TestFoo -count=1 github.com/foo/bar",
		},
		{
			f:    TestFailure{Suite: "vitest", Package: "/project/tests/web/login.spec.ts"},
			want: "pnpm exec vitest run",
		},
		{
			f:    TestFailure{Suite: "playwright", TestName: "Login flow > should redirect"},
			want: "pnpm exec playwright test --grep",
		},
	}

	for _, tc := range tests {
		got := buildReproduction(tc.f)
		if !strings.Contains(got, tc.want) {
			t.Errorf("buildReproduction(%q) = %q, want to contain %q", tc.f.Suite, got, tc.want)
		}
	}
}

func TestDefectFiler_LineageIndexIncremented(t *testing.T) {
	idx, dir := openDefectIndex(t)
	df := NewDefectFiler(idx, dir)

	matched := upsertTestArtifactForDefect(t, idx, "lifecycle/tests/my-feature-6-test.md", "my-feature")

	f := TestFailure{Suite: "go", TestName: "Test1", Package: "pkg", ErrorMsg: "err1"}
	path1, err := df.FileDefect([]TestFailure{f}, matched)
	if err != nil {
		t.Fatalf("FileDefect #1: %v", err)
	}

	f2 := TestFailure{Suite: "go", TestName: "Test2", Package: "pkg", ErrorMsg: "err2"}
	path2, err := df.FileDefect([]TestFailure{f2}, matched)
	if err != nil {
		t.Fatalf("FileDefect #2: %v", err)
	}

	if path1 == path2 {
		t.Error("two defects must have different paths (index not incremented)")
	}
}
