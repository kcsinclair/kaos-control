// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"net/http"
	"testing"
)

// ── Milestone 4: Backlog Panel API Tests ──────────────────────────────────────
//
// These tests verify the server-side behaviour that backs the Backlog panel:
//   - GET /artifacts?release=__unassigned__ returns only artifacts that have no
//     release assignment in their frontmatter.
//   - release and sprint type artifacts are excluded from the backlog query
//     (the frontend applies this filter client-side using the full artifact list,
//     but the API must at least return the type field so the client can filter).
//   - The count returned matches the expected number of backlog items.
//   - Card-level fields (title, type, status, lineage) are present in each item.
//   - When all artifacts have release assignments the unassigned query returns
//     an empty list (empty-state scenario).

// TestBacklogPanel_UnassignedQueryExcludesReleaseAndSprint verifies that
// artifacts of type "release" and "sprint" do NOT surface as backlog items in
// GET /artifacts?release=__unassigned__ (FR3.2).
//
// Since release-artefacts-9 (DR-4) releases are single-cached in the dedicated
// `releases` table and are NOT indexed into the artifacts table, so they never
// appear in this query at all. Sprints are still artifacts, so they are still
// returned by the API's __unassigned__ filter (no release field) and the client
// filters them out via `a.type !== 'sprint'` in RoadmapView.vue — the type field
// must therefore be present on each item.
func TestBacklogPanel_UnassignedQueryExcludesReleaseAndSprint(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/bp-idea-no-release.md",
			content: makeArtifact("BP Idea No Release", "idea", "draft", "bp-idea-no-release", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/bp-req-no-release.md",
			content: makeArtifact("BP Req No Release", "ticket", "draft", "bp-req-no-release", "", "Body."),
		},
		{
			relPath: "lifecycle/releases/bp-release-artifact.md",
			content: makeArtifact("BP Release Type", "release", "draft", "bp-release-artifact", "", "Body."),
		},
		{
			relPath: "lifecycle/sprints/bp-sprint-artifact.md",
			content: makeArtifact("BP Sprint Type", "sprint", "draft", "bp-sprint-artifact", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/bp-idea-with-release.md",
			content: makeArtifactWithRelease("BP Idea With Release", "idea", "draft", "bp-idea-with-release", "some-release", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?release=__unassigned__", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	items, _ := body["items"].([]any)

	// Expected: bp-idea-no-release, bp-req-no-release, bp-sprint-artifact.
	// Release-type artifacts are NOT in the artifacts table (DR-4), so
	// bp-release-artifact must be absent. Sprint IS still an artifact and is
	// returned (client filters it), so its type field must be present.
	foundTypes := map[string]int{}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		typ, _ := item["type"].(string)
		foundTypes[typ]++
	}

	// Releases are single-cached in the releases table and must not appear here.
	if foundTypes["release"] != 0 {
		t.Errorf("release-type artifacts must not appear in the artifacts list (DR-4); found %d", foundTypes["release"])
	}
	// Sprint is still an artifact, returned by the API for the client to filter.
	if foundTypes["sprint"] == 0 {
		t.Error("expected at least one item of type 'sprint' in unassigned list (type field must be present for client filtering)")
	}
	if foundTypes["idea"] == 0 {
		t.Error("expected at least one item of type 'idea' in unassigned list")
	}

	// The artifact with a release assignment must not appear.
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		fm, _ := item["frontmatter"].(map[string]any)
		if rel, ok := fm["release"]; ok && rel != "" && rel != nil {
			t.Errorf("unassigned filter returned artifact with non-empty release field: %v", rel)
		}
	}
}

// TestBacklogPanel_CardFields verifies that each artifact returned by the
// unassigned query carries the fields required to render a Backlog card (FR3.3):
// title, type, status, and lineage.
func TestBacklogPanel_CardFields(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/bp-card-1.md",
			content: makeArtifact("BP Card One", "idea", "draft", "bp-card-1", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/bp-card-2.md",
			content: makeArtifact("BP Card Two", "ticket", "planning", "bp-card-2", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?release=__unassigned__", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)
	items, _ := body["items"].([]any)

	if len(items) < 2 {
		t.Fatalf("expected at least 2 backlog items, got %d", len(items))
	}

	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if title, _ := item["title"].(string); title == "" {
			t.Errorf("backlog item missing title field: %v", item)
		}
		if typ, _ := item["type"].(string); typ == "" {
			t.Errorf("backlog item missing type field: %v", item)
		}
		if status, _ := item["status"].(string); status == "" {
			t.Errorf("backlog item missing status field: %v", item)
		}
		if lineage, _ := item["lineage"].(string); lineage == "" {
			t.Errorf("backlog item missing lineage field: %v", item)
		}
	}
}

// TestBacklogPanel_CountMatchesFixtures verifies that the total returned by
// the unassigned query equals the number of unassigned fixture artifacts (FR3.5).
func TestBacklogPanel_CountMatchesFixtures(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/bp-count-1.md",
			content: makeArtifact("BP Count 1", "idea", "draft", "bp-count-1", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/bp-count-2.md",
			content: makeArtifact("BP Count 2", "idea", "draft", "bp-count-2", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/bp-count-assigned.md",
			content: makeArtifactWithRelease("BP Count Assigned", "idea", "draft", "bp-count-assigned", "v-count-rel", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?release=__unassigned__", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	items, _ := body["items"].([]any)
	total, _ := body["total"].(float64)

	// Only bp-count-1 and bp-count-2 should be unassigned.
	if len(items) != 2 {
		t.Errorf("backlog item count: want 2, got %d", len(items))
	}
	if int(total) != 2 {
		t.Errorf("backlog total field: want 2, got %d", int(total))
	}
}

// TestBacklogPanel_EmptyStateWhenAllAssigned verifies that GET /artifacts?release=__unassigned__
// returns an empty list when every artifact has a release assignment (FR3.6).
func TestBacklogPanel_EmptyStateWhenAllAssigned(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/bp-empty-1.md",
			content: makeArtifactWithRelease("BP Empty 1", "idea", "draft", "bp-empty-1", "v-empty-rel", "Body."),
		},
		{
			relPath: "lifecycle/ideas/bp-empty-2.md",
			content: makeArtifactWithRelease("BP Empty 2", "idea", "draft", "bp-empty-2", "v-empty-rel", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?release=__unassigned__", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	items, _ := body["items"].([]any)
	if len(items) != 0 {
		t.Errorf("empty backlog: want 0 unassigned items, got %d", len(items))
	}
}

// TestBacklogPanel_FilterByType verifies that GET /artifacts?release=__unassigned__&type=<t>
// returns only items of the requested type (OQ1 — type filter).
func TestBacklogPanel_FilterByType(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/bp-ft-idea-1.md",
			content: makeArtifact("BP FT Idea 1", "idea", "draft", "bp-ft-idea-1", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/bp-ft-idea-2.md",
			content: makeArtifact("BP FT Idea 2", "idea", "draft", "bp-ft-idea-2", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/bp-ft-ticket-1.md",
			content: makeArtifact("BP FT Ticket 1", "ticket", "draft", "bp-ft-ticket-1", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?release=__unassigned__&type=idea", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	items, _ := body["items"].([]any)
	if len(items) != 2 {
		t.Errorf("type=idea filter: want 2 items, got %d", len(items))
	}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if typ, _ := item["type"].(string); typ != "idea" {
			t.Errorf("type filter returned item with type %q, want %q", typ, "idea")
		}
	}
}

// TestBacklogPanel_FilterByStatus verifies that GET /artifacts?release=__unassigned__&status=<s>
// returns only items with the requested status (OQ1 — status filter).
func TestBacklogPanel_FilterByStatus(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/bp-fs-draft-1.md",
			content: makeArtifact("BP FS Draft 1", "idea", "draft", "bp-fs-draft-1", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/bp-fs-draft-2.md",
			content: makeArtifact("BP FS Draft 2", "idea", "draft", "bp-fs-draft-2", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/bp-fs-planning.md",
			content: makeArtifact("BP FS Planning", "idea", "planning", "bp-fs-planning", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?release=__unassigned__&status=draft", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	items, _ := body["items"].([]any)
	if len(items) != 2 {
		t.Errorf("status=draft filter: want 2 items, got %d", len(items))
	}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if status, _ := item["status"].(string); status != "draft" {
			t.Errorf("status filter returned item with status %q, want %q", status, "draft")
		}
	}
}

// TestBacklogPanel_RoadmapGraphBacklogNode verifies that the roadmap graph
// always includes a synthetic Backlog node (the root of the release chain),
// and that artifacts with no release field are attached to it via "assigned"
// edges (FR3.1 — Backlog is always present).
func TestBacklogPanel_RoadmapGraphBacklogNode(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/bp-gn-unassigned.md",
			content: makeArtifact("BP GN Unassigned", "idea", "draft", "bp-gn-unassigned", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/releases/graph", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	nodes, _ := body["nodes"].([]any)
	edges, _ := body["edges"].([]any)

	// Backlog node must always exist.
	var backlogFound bool
	for _, raw := range nodes {
		node, _ := raw.(map[string]any)
		if id, _ := node["id"].(string); id == "release:backlog" {
			backlogFound = true
			if title, _ := node["title"].(string); title != "Backlog" {
				t.Errorf("backlog node title: want %q, got %q", "Backlog", title)
			}
		}
	}
	if !backlogFound {
		t.Error("roadmap graph: synthetic 'release:backlog' node not found")
	}

	// Unassigned artifact must attach to Backlog via an "assigned" edge.
	var backlogEdgeFound bool
	for _, raw := range edges {
		edge, _ := raw.(map[string]any)
		src, _ := edge["source"].(string)
		kind, _ := edge["kind"].(string)
		if src == "release:backlog" && kind == "assigned" {
			backlogEdgeFound = true
			break
		}
	}
	if !backlogEdgeFound {
		t.Error("roadmap graph: no 'assigned' edge from 'release:backlog' to unassigned artifact")
	}
}
