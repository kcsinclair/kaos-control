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

// ── Milestone 3: combined widget query end-to-end ─────────────────────────────

// widgetQuerySeeds generates seed artifacts matching the test plan's requirement:
// 10 ideas and 5 defects with known creation dates, plus 3 requirements.
// Timestamps are spaced 1 hour apart so ordering is deterministic.
func widgetQuerySeeds(t *testing.T, env *testEnv) {
	t.Helper()

	base := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	// 10 ideas — created at base, base+1h, ..., base+9h
	for i := 0; i < 10; i++ {
		created := base.Add(time.Duration(i) * time.Hour).UTC().Format(time.RFC3339)
		lineage := fmt.Sprintf("wq-idea-%02d", i+1)
		title := fmt.Sprintf("Widget Idea %02d", i+1)
		content := makeArtifactDated(title, "idea", "draft", lineage, created)
		relPath := fmt.Sprintf("lifecycle/ideas/%s.md", lineage)
		absPath := filepath.Join(env.projectRoot, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", relPath, err)
		}
		if err := env.proj.Idx.IndexFile(absPath); err != nil {
			t.Fatalf("IndexFile(%s): %v", relPath, err)
		}
	}

	// 5 defects — created at base+10h, base+11h, ..., base+14h
	for i := 0; i < 5; i++ {
		created := base.Add(time.Duration(10+i) * time.Hour).UTC().Format(time.RFC3339)
		lineage := fmt.Sprintf("wq-defect-%02d", i+1)
		title := fmt.Sprintf("Widget Defect %02d", i+1)
		content := makeArtifactDated(title, "defect", "draft", lineage, created)
		relPath := fmt.Sprintf("lifecycle/defects/%s.md", lineage)
		absPath := filepath.Join(env.projectRoot, relPath)
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", relPath, err)
		}
		if err := env.proj.Idx.IndexFile(absPath); err != nil {
			t.Fatalf("IndexFile(%s): %v", relPath, err)
		}
	}

	// 3 requirements — should be excluded by the widget query
	for i := 0; i < 3; i++ {
		created := base.Add(time.Duration(20+i) * time.Hour).UTC().Format(time.RFC3339)
		lineage := fmt.Sprintf("wq-req-%02d", i+1)
		title := fmt.Sprintf("Widget Req %02d", i+1)
		content := makeArtifactDated(title, "ticket", "draft", lineage, created)
		relPath := fmt.Sprintf("lifecycle/requirements/%s-2.md", lineage)
		absPath := filepath.Join(env.projectRoot, relPath)
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", relPath, err)
		}
		if err := env.proj.Idx.IndexFile(absPath); err != nil {
			t.Fatalf("IndexFile(%s): %v", relPath, err)
		}
	}
}

// TestWidgetQuery_LimitApplied verifies that ?type=idea,defect&sort=created:desc&limit=6
// returns at most 6 items.
// Covers test plan Milestone 3, scenario 1.
func TestWidgetQuery_LimitApplied(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")
	widgetQuerySeeds(t, env)

	resp := env.doRequest("GET",
		"/api/p/testproject/artifacts?type=idea,defect&sort=created:desc&limit=6", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	if len(items) > 6 {
		t.Errorf("expected at most 6 items with limit=6, got %d", len(items))
	}
	if len(items) != 6 {
		t.Errorf("expected exactly 6 items (15 matching, limit=6), got %d", len(items))
	}
}

// TestWidgetQuery_OnlyIdeasAndDefects verifies that all returned items are type
// "idea" or "defect" — no requirements.
// Covers test plan Milestone 3, scenario 2.
func TestWidgetQuery_OnlyIdeasAndDefects(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")
	widgetQuerySeeds(t, env)

	resp := env.doRequest("GET",
		"/api/p/testproject/artifacts?type=idea,defect&sort=created:desc&limit=6", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	for i, raw := range items {
		item, _ := raw.(map[string]any)
		typ, _ := item["type"].(string)
		if typ != "idea" && typ != "defect" {
			t.Errorf("item[%d] has type %q; want idea or defect", i, typ)
		}
	}
}

// TestWidgetQuery_SortedByCreatedDesc verifies that the returned items are sorted
// by created date descending (most recent first).
// Covers test plan Milestone 3, scenario 3.
func TestWidgetQuery_SortedByCreatedDesc(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")
	widgetQuerySeeds(t, env)

	resp := env.doRequest("GET",
		"/api/p/testproject/artifacts?type=idea,defect&sort=created:desc&limit=6", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	if len(items) < 2 {
		t.Fatalf("expected at least 2 items to verify ordering, got %d", len(items))
	}

	for i := 0; i < len(items)-1; i++ {
		a, _ := items[i].(map[string]any)
		b, _ := items[i+1].(map[string]any)
		aCreated, _ := a["created"].(string)
		bCreated, _ := b["created"].(string)
		if aCreated < bCreated {
			t.Errorf("created:desc order violated at index %d: %q < %q", i, aCreated, bCreated)
		}
	}
}

// TestWidgetQuery_TotalIsFullMatchCount verifies that the total in the response
// equals the full count of ideas + defects (15), not capped at limit=6.
// Covers test plan Milestone 3, scenario 4.
func TestWidgetQuery_TotalIsFullMatchCount(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")
	widgetQuerySeeds(t, env)

	resp := env.doRequest("GET",
		"/api/p/testproject/artifacts?type=idea,defect&sort=created:desc&limit=6", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	total, _ := data["total"].(float64)
	// 10 ideas + 5 defects = 15 matching, regardless of limit=6
	if int(total) != 15 {
		t.Errorf("expected total=15 (full match count), got %d", int(total))
	}
}

// TestWidgetQuery_FewerThanLimit verifies that when fewer matching artifacts exist
// than the limit, only the available items are returned with the correct total.
// Covers test plan Milestone 3, scenario 5.
func TestWidgetQuery_FewerThanLimit(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Seed only 2 ideas (fewer than the limit of 6).
	base := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 2; i++ {
		created := base.Add(time.Duration(i) * time.Hour).UTC().Format(time.RFC3339)
		lineage := fmt.Sprintf("few-idea-%02d", i+1)
		title := fmt.Sprintf("Few Idea %02d", i+1)
		content := makeArtifactDated(title, "idea", "draft", lineage, created)
		relPath := fmt.Sprintf("lifecycle/ideas/%s.md", lineage)
		absPath := filepath.Join(env.projectRoot, relPath)
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		if err := env.proj.Idx.IndexFile(absPath); err != nil {
			t.Fatalf("IndexFile: %v", err)
		}
	}

	resp := env.doRequest("GET",
		"/api/p/testproject/artifacts?type=idea,defect&sort=created:desc&limit=6", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	total, _ := data["total"].(float64)

	if len(items) != 2 {
		t.Errorf("expected 2 items (fewer than limit), got %d", len(items))
	}
	if int(total) != 2 {
		t.Errorf("expected total=2, got %d", int(total))
	}
}

// TestWidgetQuery_ZeroResults verifies that when no ideas or defects exist,
// the response contains an empty items array and total=0.
// Covers test plan Milestone 3, scenario 6.
func TestWidgetQuery_ZeroResults(t *testing.T) {
	// Seed only requirements — no ideas or defects.
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/requirements/zero-req-2.md",
			content: makeArtifact("Zero Req", "ticket", "draft", "zero-req", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET",
		"/api/p/testproject/artifacts?type=idea,defect&sort=created:desc&limit=6", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	total, _ := data["total"].(float64)

	if len(items) != 0 {
		t.Errorf("expected empty items when no ideas/defects exist, got %d", len(items))
	}
	if int(total) != 0 {
		t.Errorf("expected total=0 when no ideas/defects exist, got %d", int(total))
	}
}
