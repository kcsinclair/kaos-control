// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"testing"
)

// ── Milestone 2 ───────────────────────────────────────────────────────────

// TestListArtifacts_FilterByStatus verifies that GET /artifacts?status=<s>
// returns only artifacts whose status matches the query parameter.
// Covers test plan Milestone 2, scenario 1.
func TestListArtifacts_FilterByStatus(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/filter-draft-1.md",
			content: makeArtifact("Draft Idea 1", "idea", "draft", "filter-draft-1", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/filter-draft-2.md",
			content: makeArtifact("Draft Idea 2", "idea", "draft", "filter-draft-2", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/filter-clarifying-2.md",
			content: makeArtifact("Clarifying Req", "ticket", "clarifying", "filter-clarifying", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/filter-planning-2.md",
			content: makeArtifact("Planning Req", "ticket", "planning", "filter-planning", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?status=draft", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	total, _ := data["total"].(float64)

	if int(total) != len(items) {
		t.Errorf("total %d does not match items length %d", int(total), len(items))
	}
	if len(items) != 2 {
		t.Errorf("expected 2 draft artifacts, got %d", len(items))
	}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if status, _ := item["status"].(string); status != "draft" {
			t.Errorf("filter returned artifact with status %q, expected %q", status, "draft")
		}
	}
}

// TestListArtifacts_FilterByStatusAndType verifies that
// GET /artifacts?status=<s>&type=<t> returns only artifacts matching both.
// Covers test plan Milestone 2, scenario 2.
func TestListArtifacts_FilterByStatusAndType(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/combo-draft-idea-1.md",
			content: makeArtifact("Draft Idea A", "idea", "draft", "combo-draft-idea-1", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/combo-draft-idea-2.md",
			content: makeArtifact("Draft Idea B", "idea", "draft", "combo-draft-idea-2", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/combo-draft-ticket-2.md",
			content: makeArtifact("Draft Ticket", "ticket", "draft", "combo-draft-ticket", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/combo-clarifying-idea-1.md",
			content: makeArtifact("Clarifying Idea", "idea", "clarifying", "combo-clarifying-idea", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?status=draft&type=idea", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	total, _ := data["total"].(float64)

	if int(total) != len(items) {
		t.Errorf("total %d does not match items length %d", int(total), len(items))
	}
	if len(items) != 2 {
		t.Errorf("expected 2 draft ideas, got %d", len(items))
	}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if status, _ := item["status"].(string); status != "draft" {
			t.Errorf("filtered artifact has status %q, want %q", status, "draft")
		}
		if typ, _ := item["type"].(string); typ != "idea" {
			t.Errorf("filtered artifact has type %q, want %q", typ, "idea")
		}
	}
}

// TestListArtifacts_FilterNoMatch verifies that querying with a status that has
// no matching artifacts returns an empty items array and total=0.
// Covers test plan Milestone 2, scenario 3.
func TestListArtifacts_FilterNoMatch(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/nomatch-idea.md",
			content: makeArtifact("No Match Idea", "idea", "draft", "nomatch-idea", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// "approved" has no seeded artifacts.
	resp := env.doRequest("GET", "/api/p/testproject/artifacts?status=approved", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	total, _ := data["total"].(float64)

	if int(total) != 0 {
		t.Errorf("expected total=0 for no-match query, got %d", int(total))
	}
	if len(items) != 0 {
		t.Errorf("expected empty items for no-match query, got %d items", len(items))
	}
}
