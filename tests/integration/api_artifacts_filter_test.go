// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"testing"
)

// ── Milestone 1: multi-value type filter ──────────────────────────────────────

// typeFilterSeeds returns a mix of ideas, defects, and requirements for type-filter tests.
func typeFilterSeeds() []seedArtifact {
	return []seedArtifact{
		{
			relPath: "lifecycle/ideas/typefilter-idea-a.md",
			content: makeArtifact("Idea Alpha", "idea", "draft", "typefilter-idea-a", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/typefilter-idea-b.md",
			content: makeArtifact("Idea Beta", "idea", "draft", "typefilter-idea-b", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/typefilter-idea-c.md",
			content: makeArtifact("Idea Gamma", "idea", "draft", "typefilter-idea-c", "", "Body."),
		},
		{
			relPath: "lifecycle/defects/typefilter-defect-a.md",
			content: makeArtifact("Defect Alpha", "defect", "draft", "typefilter-defect-a", "", "Body."),
		},
		{
			relPath: "lifecycle/defects/typefilter-defect-b.md",
			content: makeArtifact("Defect Beta", "defect", "draft", "typefilter-defect-b", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/typefilter-req-a-2.md",
			content: makeArtifact("Requirement Alpha", "ticket", "draft", "typefilter-req-a", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/typefilter-req-b-2.md",
			content: makeArtifact("Requirement Beta", "ticket", "draft", "typefilter-req-b", "", "Body."),
		},
	}
}

// TestArtifactTypeFilter_MultiValue verifies that ?type=idea,defect returns only
// artifacts of type "idea" or "defect", excluding other types.
// Covers test plan Milestone 1, scenario 1.
func TestArtifactTypeFilter_MultiValue(t *testing.T) {
	env := newTestEnv(t, typeFilterSeeds())
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?type=idea,defect", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	total, _ := data["total"].(float64)

	if len(items) != 5 {
		t.Errorf("expected 5 items (3 ideas + 2 defects), got %d", len(items))
	}
	if int(total) != 5 {
		t.Errorf("expected total=5, got %d", int(total))
	}

	for _, raw := range items {
		item, _ := raw.(map[string]any)
		typ, _ := item["type"].(string)
		if typ != "idea" && typ != "defect" {
			t.Errorf("multi-value filter returned artifact with type %q; want idea or defect", typ)
		}
	}
}

// TestArtifactTypeFilter_SingleValue verifies that ?type=idea returns only ideas
// (single-value backward compatibility).
// Covers test plan Milestone 1, scenario 2.
func TestArtifactTypeFilter_SingleValue(t *testing.T) {
	env := newTestEnv(t, typeFilterSeeds())
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?type=idea", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	total, _ := data["total"].(float64)

	if len(items) != 3 {
		t.Errorf("expected 3 ideas, got %d", len(items))
	}
	if int(total) != 3 {
		t.Errorf("expected total=3, got %d", int(total))
	}

	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if typ, _ := item["type"].(string); typ != "idea" {
			t.Errorf("single-value filter returned artifact with type %q; want idea", typ)
		}
	}
}

// TestArtifactTypeFilter_ThreeTypes verifies that ?type=idea,defect,ticket
// returns artifacts of all three specified types.
// Covers test plan Milestone 1, scenario 3.
func TestArtifactTypeFilter_ThreeTypes(t *testing.T) {
	env := newTestEnv(t, typeFilterSeeds())
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?type=idea,defect,ticket", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	total, _ := data["total"].(float64)

	// 3 ideas + 2 defects + 2 tickets = 7
	if len(items) != 7 {
		t.Errorf("expected 7 items (3 ideas + 2 defects + 2 tickets), got %d", len(items))
	}
	if int(total) != 7 {
		t.Errorf("expected total=7, got %d", int(total))
	}

	allowed := map[string]bool{"idea": true, "defect": true, "ticket": true}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		typ, _ := item["type"].(string)
		if !allowed[typ] {
			t.Errorf("three-type filter returned artifact with type %q; want idea, defect, or ticket", typ)
		}
	}
}

// TestArtifactTypeFilter_Nonexistent verifies that ?type=nonexistent returns an
// empty list and no error.
// Covers test plan Milestone 1, scenario 4.
func TestArtifactTypeFilter_Nonexistent(t *testing.T) {
	env := newTestEnv(t, typeFilterSeeds())
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?type=nonexistent", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	total, _ := data["total"].(float64)

	if len(items) != 0 {
		t.Errorf("expected empty items for nonexistent type, got %d", len(items))
	}
	if int(total) != 0 {
		t.Errorf("expected total=0 for nonexistent type, got %d", int(total))
	}
}

// TestArtifactTypeFilter_Omitted verifies that omitting the type parameter
// returns all artifact types (existing behaviour preserved).
// Covers test plan Milestone 1, scenario 5.
func TestArtifactTypeFilter_Omitted(t *testing.T) {
	env := newTestEnv(t, typeFilterSeeds())
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	total, _ := data["total"].(float64)

	// 7 seeded artifacts total (3 ideas + 2 defects + 2 tickets)
	if int(total) != 7 {
		t.Errorf("expected total=7 when type omitted, got %d", int(total))
	}

	// Verify all three types are represented.
	items, _ := data["items"].([]any)
	types := map[string]int{}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		typ, _ := item["type"].(string)
		types[typ]++
	}
	if types["idea"] != 3 {
		t.Errorf("expected 3 ideas in unfiltered results, got %d", types["idea"])
	}
	if types["defect"] != 2 {
		t.Errorf("expected 2 defects in unfiltered results, got %d", types["defect"])
	}
	if types["ticket"] != 2 {
		t.Errorf("expected 2 tickets in unfiltered results, got %d", types["ticket"])
	}
}
