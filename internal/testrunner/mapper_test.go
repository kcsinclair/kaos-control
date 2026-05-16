// SPDX-License-Identifier: AGPL-3.0-or-later

package testrunner

import (
	"os"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/index"
)

// openMapperIndex opens a minimal SQLite index in a temp directory.
func openMapperIndex(t *testing.T) (*index.Index, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(dir+"/lifecycle/tests", 0o755); err != nil {
		t.Fatal(err)
	}
	idx, err := index.Open(dir+"/test.db", dir, nil)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { idx.Close() })
	return idx, dir
}

// upsertTestArtifact inserts a test-type artifact into the index.
func upsertTestArtifact(t *testing.T, idx *index.Index, path, slug, lineage string, labels []string) {
	t.Helper()
	a := &artifact.Artifact{
		Path:  path,
		Slug:  slug,
		Index: 2,
		Stage: "tests",
		Mtime: time.Now(),
		FM: artifact.Frontmatter{
			Title:   "Test: " + slug,
			Type:    "test",
			Status:  "draft",
			Lineage: lineage,
			Labels:  labels,
		},
	}
	if err := idx.Upsert(a); err != nil {
		t.Fatalf("Upsert(%s): %v", path, err)
	}
}

func TestArtifactMapper_SlugMatch(t *testing.T) {
	idx, _ := openMapperIndex(t)
	// Insert a test artifact whose lineage matches the slug derived from the test file.
	upsertTestArtifact(t, idx,
		"lifecycle/tests/artifact-store-2-test.md",
		"artifact-store", "artifact-store", nil)

	m := NewArtifactMapper(idx)
	f := TestFailure{
		Suite:   "go",
		Package: "github.com/foo/bar/internal/store",
		File:    "artifact_store_test.go",
	}
	row, err := m.MapFailure(f)
	if err != nil {
		t.Fatalf("MapFailure: %v", err)
	}
	if row == nil {
		t.Fatal("expected non-nil ArtifactRow for slug match")
	}
	if row.Lineage != "artifact-store" {
		t.Errorf("Lineage = %q, want artifact-store", row.Lineage)
	}
}

func TestArtifactMapper_LabelMatch(t *testing.T) {
	idx, _ := openMapperIndex(t)
	// Insert a test artifact labelled with the package name.
	upsertTestArtifact(t, idx,
		"lifecycle/tests/index-tests-2-test.md",
		"index-tests", "index-tests", []string{"index"})

	m := NewArtifactMapper(idx)
	f := TestFailure{
		Suite:   "go",
		Package: "github.com/kaos-control/kaos-control/internal/index",
		File:    "query_test.go",
	}
	// The slug of "query_test.go" → "query" which won't match "index-tests".
	// But the label "index" should match.
	row, err := m.MapFailure(f)
	if err != nil {
		t.Fatalf("MapFailure: %v", err)
	}
	if row == nil {
		t.Fatal("expected non-nil ArtifactRow for label match")
	}
	if row.Lineage != "index-tests" {
		t.Errorf("Lineage = %q, want index-tests", row.Lineage)
	}
}

func TestArtifactMapper_LineageMatch(t *testing.T) {
	idx, _ := openMapperIndex(t)
	// Insert a test artifact whose lineage matches the path-derived slug.
	upsertTestArtifact(t, idx,
		"lifecycle/tests/internal-store-2-test.md",
		"internal-store", "internal-store", nil)

	m := NewArtifactMapper(idx)
	f := TestFailure{
		Suite:   "go",
		Package: "github.com/foo/bar/internal/store",
		File:    "completely_different_test.go", // slug "completely-different" won't match
	}
	// pathToSlug("github.com/foo/bar/internal/store") → "internal-store"
	row, err := m.MapFailure(f)
	if err != nil {
		t.Fatalf("MapFailure: %v", err)
	}
	if row == nil {
		t.Fatal("expected non-nil ArtifactRow for lineage match")
	}
	if row.Lineage != "internal-store" {
		t.Errorf("Lineage = %q, want internal-store", row.Lineage)
	}
}

func TestArtifactMapper_NoMatch(t *testing.T) {
	idx, _ := openMapperIndex(t)
	// No test artifacts in the index.

	m := NewArtifactMapper(idx)
	f := TestFailure{
		Suite:   "go",
		Package: "example.com/foo",
		File:    "foo_test.go",
	}
	row, err := m.MapFailure(f)
	if err != nil {
		t.Fatalf("MapFailure: %v", err)
	}
	if row != nil {
		t.Errorf("expected nil ArtifactRow when no match, got %+v", row)
	}
}

func TestDetectOrphans(t *testing.T) {
	idx, _ := openMapperIndex(t)
	upsertTestArtifact(t, idx,
		"lifecycle/tests/login-2-test.md",
		"login", "login", nil)
	upsertTestArtifact(t, idx,
		"lifecycle/tests/signup-2-test.md",
		"signup", "signup", nil)

	artifacts, _, err := idx.List(index.Filter{Stage: "tests", Unlimited: true})
	if err != nil {
		t.Fatal(err)
	}

	// "login" has a failure, "signup" does not → signup is an orphan.
	results := []SuiteResult{
		{
			Suite: "go",
			Failures: []TestFailure{
				{File: "login_test.go", Package: "example.com/login"},
			},
		},
	}

	orphans := DetectOrphans(results, artifacts)
	if len(orphans) == 0 {
		t.Fatal("expected signup to be detected as orphan")
	}
	found := false
	for _, o := range orphans {
		if o == "lifecycle/tests/signup-2-test.md" {
			found = true
		}
	}
	if !found {
		t.Errorf("orphans = %v, expected signup-2-test.md", orphans)
	}
}

func TestFileToSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"artifact_store_test.go", "artifact-store"},
		{"login.spec.ts", "login"},
		{"user_profile_test.go", "user-profile"},
		{"index.test.ts", "index"},
		{"foo_test.go", "foo"},
		{"", ""},
	}
	for _, tc := range tests {
		got := fileToSlug(tc.input)
		if got != tc.want {
			t.Errorf("fileToSlug(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestPathToSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"github.com/foo/bar/internal/store", "internal-store"},
		{"github.com/foo/bar/internal/index", "internal-index"},
		{"example.com/foo", "example-foo"},
		{"", ""},
	}
	for _, tc := range tests {
		got := pathToSlug(tc.input)
		if got != tc.want {
			t.Errorf("pathToSlug(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
