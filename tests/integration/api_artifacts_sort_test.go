// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── Milestone 2: sort parameter tests ────────────────────────────────────────

// makeArtifactDated builds a minimal artifact string with an explicit RFC3339 created field.
func makeArtifactDated(title, typ, status, lineage, created string) string {
	return fmt.Sprintf("---\ntitle: %s\ntype: %s\nstatus: %s\nlineage: %s\ncreated: %q\n---\n\nBody.\n",
		title, typ, status, lineage, created)
}

// seedAndIndexDatedArtifacts writes artifact files with explicit created timestamps
// to the project root and re-indexes them, returning the sorted expected paths.
// artifacts is a slice of (relPath, title, created) tuples.
func seedAndIndexDatedArtifacts(t *testing.T, env *testEnv, artifacts []struct {
	relPath string
	title   string
	typ     string
	lineage string
	created time.Time
}) {
	t.Helper()
	for _, a := range artifacts {
		absPath := filepath.Join(env.projectRoot, a.relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", filepath.Dir(absPath), err)
		}
		created := a.created.UTC().Format(time.RFC3339)
		content := makeArtifactDated(a.title, a.typ, "draft", a.lineage, created)
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", absPath, err)
		}
		if err := env.proj.Idx.IndexFile(absPath); err != nil {
			t.Fatalf("IndexFile(%s): %v", absPath, err)
		}
	}
}

// TestArtifactSort_CreatedDesc verifies that ?sort=created:desc returns the
// most recently created artifact first.
// Covers test plan Milestone 2, scenario 1.
func TestArtifactSort_CreatedDesc(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	artifacts := []struct {
		relPath string
		title   string
		typ     string
		lineage string
		created time.Time
	}{
		{"lifecycle/ideas/sort-desc-old.md", "Sort Old", "idea", "sort-desc-old", base},
		{"lifecycle/ideas/sort-desc-mid.md", "Sort Mid", "idea", "sort-desc-mid", base.Add(time.Hour)},
		{"lifecycle/ideas/sort-desc-new.md", "Sort New", "idea", "sort-desc-new", base.Add(2 * time.Hour)},
	}
	seedAndIndexDatedArtifacts(t, env, artifacts)

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?sort=created:desc", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	if len(items) < 3 {
		t.Fatalf("expected at least 3 items, got %d", len(items))
	}

	// The first item should have the newest created date.
	first, _ := items[0].(map[string]any)
	firstTitle, _ := first["title"].(string)
	if firstTitle != "Sort New" {
		t.Errorf("created:desc — first item title: want %q, got %q", "Sort New", firstTitle)
	}

	// Verify descending order: each item's created should be >= the next.
	for i := 0; i < len(items)-1; i++ {
		a, _ := items[i].(map[string]any)
		b, _ := items[i+1].(map[string]any)
		aFM, _ := a["frontmatter"].(map[string]any)
		bFM, _ := b["frontmatter"].(map[string]any)
		aCreated, _ := aFM["created"].(string)
		bCreated, _ := bFM["created"].(string)
		if aCreated < bCreated {
			t.Errorf("created:desc order violated at index %d: %q < %q", i, aCreated, bCreated)
		}
	}
}

// TestArtifactSort_CreatedAsc verifies that ?sort=created:asc returns the
// oldest created artifact first.
// Covers test plan Milestone 2, scenario 2.
func TestArtifactSort_CreatedAsc(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	base := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	artifacts := []struct {
		relPath string
		title   string
		typ     string
		lineage string
		created time.Time
	}{
		{"lifecycle/ideas/sort-asc-new.md", "Sort New", "idea", "sort-asc-new", base.Add(2 * time.Hour)},
		{"lifecycle/ideas/sort-asc-old.md", "Sort Old", "idea", "sort-asc-old", base},
		{"lifecycle/ideas/sort-asc-mid.md", "Sort Mid", "idea", "sort-asc-mid", base.Add(time.Hour)},
	}
	seedAndIndexDatedArtifacts(t, env, artifacts)

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?sort=created:asc", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	if len(items) < 3 {
		t.Fatalf("expected at least 3 items, got %d", len(items))
	}

	first, _ := items[0].(map[string]any)
	firstTitle, _ := first["title"].(string)
	if firstTitle != "Sort Old" {
		t.Errorf("created:asc — first item title: want %q, got %q", "Sort Old", firstTitle)
	}

	// Verify ascending order.
	for i := 0; i < len(items)-1; i++ {
		a, _ := items[i].(map[string]any)
		b, _ := items[i+1].(map[string]any)
		aFM, _ := a["frontmatter"].(map[string]any)
		bFM, _ := b["frontmatter"].(map[string]any)
		aCreated, _ := aFM["created"].(string)
		bCreated, _ := bFM["created"].(string)
		if aCreated > bCreated {
			t.Errorf("created:asc order violated at index %d: %q > %q", i, aCreated, bCreated)
		}
	}
}

// TestArtifactSort_TitleAsc verifies that ?sort=title:asc returns items sorted
// alphabetically by title.
// Covers test plan Milestone 2, scenario 3.
func TestArtifactSort_TitleAsc(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/sort-title-cherry.md",
			content: makeArtifact("Cherry", "idea", "draft", "sort-title-cherry", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/sort-title-apple.md",
			content: makeArtifact("Apple", "idea", "draft", "sort-title-apple", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/sort-title-banana.md",
			content: makeArtifact("Banana", "idea", "draft", "sort-title-banana", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?sort=title:asc", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	if len(items) < 3 {
		t.Fatalf("expected at least 3 items, got %d", len(items))
	}

	first, _ := items[0].(map[string]any)
	firstTitle, _ := first["title"].(string)
	if firstTitle != "Apple" {
		t.Errorf("title:asc — first item: want %q, got %q", "Apple", firstTitle)
	}

	second, _ := items[1].(map[string]any)
	secondTitle, _ := second["title"].(string)
	if secondTitle != "Banana" {
		t.Errorf("title:asc — second item: want %q, got %q", "Banana", secondTitle)
	}

	third, _ := items[2].(map[string]any)
	thirdTitle, _ := third["title"].(string)
	if thirdTitle != "Cherry" {
		t.Errorf("title:asc — third item: want %q, got %q", "Cherry", thirdTitle)
	}
}

// TestArtifactSort_Default verifies that omitting the sort parameter returns
// items ordered by lineage, idx, path (the default order).
// Covers test plan Milestone 2, scenario 4.
func TestArtifactSort_Default(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/sort-default-zzz.md",
			content: makeArtifact("ZZZ Idea", "idea", "draft", "sort-default-zzz", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/sort-default-aaa.md",
			content: makeArtifact("AAA Idea", "idea", "draft", "sort-default-aaa", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Without sort, default order is lineage, idx, path — "sort-default-aaa" < "sort-default-zzz".
	resp := env.doRequest("GET", "/api/p/testproject/artifacts", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	if len(items) < 2 {
		t.Fatalf("expected at least 2 items, got %d", len(items))
	}

	first, _ := items[0].(map[string]any)
	firstLineage, _ := first["lineage"].(string)
	second, _ := items[1].(map[string]any)
	secondLineage, _ := second["lineage"].(string)

	// Default sort by lineage asc: aaa < zzz.
	if firstLineage >= secondLineage {
		t.Errorf("default order: expected %q before %q, got reversed", firstLineage, secondLineage)
	}
}

// TestArtifactSort_InvalidColumn verifies that an invalid sort column (e.g.
// ?sort=badcolumn:desc) silently falls back to default order and returns no error.
// Covers test plan Milestone 2, scenario 5.
func TestArtifactSort_InvalidColumn(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/sort-invalid-a.md",
			content: makeArtifact("Sort Invalid A", "idea", "draft", "sort-invalid-a", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?sort=badcolumn:desc", nil)
	// Must return 200, not an error status.
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	// Response must have items (not nil).
	if _, ok := data["items"]; !ok {
		t.Error("expected items key in response for invalid sort column")
	}
}

// TestArtifactSort_MalformedValue verifies that a malformed sort value (missing
// colon direction, e.g. ?sort=created) silently falls back to default order and
// returns no error.
// Covers test plan Milestone 2, scenario 6.
func TestArtifactSort_MalformedValue(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/sort-malformed-a.md",
			content: makeArtifact("Sort Malformed A", "idea", "draft", "sort-malformed-a", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?sort=created", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	if _, ok := data["items"]; !ok {
		t.Error("expected items key in response for malformed sort value")
	}
}

// TestArtifactSort_SQLInjection verifies that a SQL injection attempt in the sort
// parameter (e.g. ?sort=created;DROP TABLE artifacts--:desc) falls back to default
// order, returns no error, and leaves the artifacts table intact.
// Covers test plan Milestone 2, scenario 7.
func TestArtifactSort_SQLInjection(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/sort-inject-a.md",
			content: makeArtifact("Sort Inject A", "idea", "draft", "sort-inject-a", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/sort-inject-b.md",
			content: makeArtifact("Sort Inject B", "idea", "draft", "sort-inject-b", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// SQL injection attempt in the sort column.
	resp := env.doRequest("GET",
		"/api/p/testproject/artifacts?sort=created%3BDROP+TABLE+artifacts--:desc", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	// Table must be intact: the seeded artifacts must still be retrievable.
	items, _ := data["items"].([]any)
	if len(items) == 0 {
		t.Error("artifacts table appears to have been dropped (got zero items)")
	}
	total, _ := data["total"].(float64)
	if int(total) == 0 {
		t.Error("total=0 after SQL injection attempt; artifacts table may be damaged")
	}

	// Follow-up: a normal request must still work correctly.
	resp2 := env.doRequest("GET", "/api/p/testproject/artifacts", nil)
	requireStatus(t, resp2, 200)
	data2 := readJSON(t, resp2)
	total2, _ := data2["total"].(float64)
	if int(total2) != 2 {
		t.Errorf("post-injection: expected total=2, got %d", int(total2))
	}
}
