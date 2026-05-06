//go:build integration

package integration

import (
	"net/http"
	"testing"
)

// ── Milestone 5: Artifact release-filter query tests ──────────────────────────

// TestReleaseFilter_ByReleaseName verifies that GET /artifacts?release=<name>
// returns only artifacts with that release field set in frontmatter.
func TestReleaseFilter_ByReleaseName(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rf-v1-idea-1.md",
			content: makeArtifactWithRelease("RF V1 Idea 1", "idea", "draft", "rf-v1-idea-1", "v-filter-1", "Body."),
		},
		{
			relPath: "lifecycle/ideas/rf-v1-idea-2.md",
			content: makeArtifactWithRelease("RF V1 Idea 2", "idea", "draft", "rf-v1-idea-2", "v-filter-1", "Body."),
		},
		{
			relPath: "lifecycle/defects/rf-v2-defect.md",
			content: makeArtifactWithRelease("RF V2 Defect", "defect", "draft", "rf-v2-defect", "v-filter-2", "Body."),
		},
		{
			relPath: "lifecycle/ideas/rf-no-release.md",
			content: makeArtifact("RF No Release", "idea", "draft", "rf-no-release", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{"name": "v-filter-1", "status": "planned"})
	createRelease(t, env, map[string]any{"name": "v-filter-2", "status": "planned"})

	// Filter by v-filter-1 → exactly two artifacts.
	resp := env.doRequest("GET", "/api/p/testproject/artifacts?release=v-filter-1", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)
	items, _ := data["items"].([]any)
	if len(items) != 2 {
		t.Errorf("?release=v-filter-1: want 2 items, got %d", len(items))
	}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		fm, _ := item["frontmatter"].(map[string]any)
		rel, _ := fm["release"].(string)
		if rel != "v-filter-1" {
			t.Errorf("filtered artifact has release %q, want %q", rel, "v-filter-1")
		}
	}

	// Filter by v-filter-2 → exactly one artifact.
	resp2 := env.doRequest("GET", "/api/p/testproject/artifacts?release=v-filter-2", nil)
	requireStatus(t, resp2, http.StatusOK)
	data2 := readJSON(t, resp2)
	items2, _ := data2["items"].([]any)
	if len(items2) != 1 {
		t.Errorf("?release=v-filter-2: want 1 item, got %d", len(items2))
	}
}

// TestReleaseFilter_Unassigned verifies that GET /artifacts?release=__unassigned__
// returns only artifacts with no release field.
func TestReleaseFilter_Unassigned(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rfu-assigned.md",
			content: makeArtifactWithRelease("RFU Assigned", "idea", "draft", "rfu-assigned", "v-rfu", "Body."),
		},
		{
			relPath: "lifecycle/ideas/rfu-none-1.md",
			content: makeArtifact("RFU None 1", "idea", "draft", "rfu-none-1", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/rfu-none-2.md",
			content: makeArtifact("RFU None 2", "idea", "draft", "rfu-none-2", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{"name": "v-rfu", "status": "planned"})

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?release=__unassigned__", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)
	items, _ := data["items"].([]any)
	if len(items) != 2 {
		t.Errorf("?release=__unassigned__: want 2 items, got %d", len(items))
	}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		fm, _ := item["frontmatter"].(map[string]any)
		if rel, ok := fm["release"]; ok && rel != "" && rel != nil {
			t.Errorf("unassigned filter returned artifact with release field %q", rel)
		}
	}
}

// TestReleaseFilter_NoFilter verifies that GET /artifacts without a release
// parameter returns all artifacts regardless of release assignment.
func TestReleaseFilter_NoFilter(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rfn-v1.md",
			content: makeArtifactWithRelease("RFN V1", "idea", "draft", "rfn-v1", "v-rfn-1", "Body."),
		},
		{
			relPath: "lifecycle/ideas/rfn-v2.md",
			content: makeArtifactWithRelease("RFN V2", "idea", "draft", "rfn-v2", "v-rfn-2", "Body."),
		},
		{
			relPath: "lifecycle/ideas/rfn-none.md",
			content: makeArtifact("RFN None", "idea", "draft", "rfn-none", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)
	items, _ := data["items"].([]any)
	if len(items) != 3 {
		t.Errorf("no release filter: want 3 items, got %d", len(items))
	}
}

// TestReleaseFilter_GraphEndpoint verifies that GET /graph?release=<name>
// returns only nodes assigned to that release.
func TestReleaseFilter_GraphEndpoint(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rfg-v1-idea.md",
			content: makeArtifactWithRelease("RFG V1 Idea", "idea", "draft", "rfg-v1-idea", "v-graph-1", "Body."),
		},
		{
			relPath: "lifecycle/ideas/rfg-v2-idea.md",
			content: makeArtifactWithRelease("RFG V2 Idea", "idea", "draft", "rfg-v2-idea", "v-graph-2", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{"name": "v-graph-1", "status": "planned"})
	createRelease(t, env, map[string]any{"name": "v-graph-2", "status": "planned"})

	resp := env.doRequest("GET", "/api/p/testproject/graph?release=v-graph-1", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	nodes, _ := data["nodes"].([]any)
	if len(nodes) != 1 {
		t.Errorf("graph?release=v-graph-1: want 1 node, got %d", len(nodes))
	}
	if len(nodes) > 0 {
		node, _ := nodes[0].(map[string]any)
		id, _ := node["id"].(string)
		if id != "lifecycle/ideas/rfg-v1-idea.md" {
			t.Errorf("graph node id: want %q, got %q", "lifecycle/ideas/rfg-v1-idea.md", id)
		}
	}
}

// TestReleaseFilter_RoadmapGraph verifies that GET /releases/graph:
//   - returns releases as nodes of type "release"
//   - includes only idea and defect artifact types as child nodes
//   - connects releases chronologically via timeline edges
func TestReleaseFilter_RoadmapGraph(t *testing.T) {
	seeds := []seedArtifact{
		// idea and defect → should appear as child nodes
		{
			relPath: "lifecycle/ideas/rm-idea.md",
			content: makeArtifactWithRelease("RM Idea", "idea", "draft", "rm-idea", "v-roadmap-1", "Body."),
		},
		{
			relPath: "lifecycle/defects/rm-defect.md",
			content: makeArtifactWithRelease("RM Defect", "defect", "draft", "rm-defect", "v-roadmap-1", "Body."),
		},
		// plan → should NOT appear as a child node
		{
			relPath: "lifecycle/backend-plans/rm-plan-2.md",
			content: makeArtifactWithRelease("RM Plan", "plan-backend", "draft", "rm-plan", "v-roadmap-1", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{
		"name": "v-roadmap-1", "status": "planned",
		"start_date": "2026-01-01", "end_date": "2026-03-31",
	})
	createRelease(t, env, map[string]any{
		"name": "v-roadmap-2", "status": "planned",
		"start_date": "2026-04-01", "end_date": "2026-06-30",
	})

	resp := env.doRequest("GET", "/api/p/testproject/releases/graph", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	nodes, _ := data["nodes"].([]any)
	edges, _ := data["edges"].([]any)

	// Count release nodes and artifact child nodes.
	releaseNodeCount := 0
	for _, raw := range nodes {
		node, _ := raw.(map[string]any)
		typ, _ := node["type"].(string)
		if typ == "release" {
			releaseNodeCount++
		}
		// plan-backend must not appear.
		if typ == "plan-backend" {
			t.Errorf("plan-backend artifact should not appear in roadmap graph")
		}
	}

	if releaseNodeCount != 2 {
		t.Errorf("roadmap graph: want 2 release nodes, got %d", releaseNodeCount)
	}

	// There should be a timeline edge between the two scheduled releases.
	timelineEdgeCount := 0
	for _, raw := range edges {
		edge, _ := raw.(map[string]any)
		if kind, _ := edge["kind"].(string); kind == "timeline" {
			timelineEdgeCount++
		}
	}
	if timelineEdgeCount < 1 {
		t.Errorf("roadmap graph: want at least 1 timeline edge, got %d", timelineEdgeCount)
	}

	// There should be assigned edges for the idea and defect.
	assignedPaths := map[string]bool{}
	for _, raw := range edges {
		edge, _ := raw.(map[string]any)
		if kind, _ := edge["kind"].(string); kind == "assigned" {
			target, _ := edge["target"].(string)
			assignedPaths[target] = true
		}
	}
	for _, wantPath := range []string{
		"lifecycle/ideas/rm-idea.md",
		"lifecycle/defects/rm-defect.md",
	} {
		if !assignedPaths[wantPath] {
			t.Errorf("roadmap graph: expected assigned edge to %q", wantPath)
		}
	}
	// The plan should not be a target of an assigned edge.
	if assignedPaths["lifecycle/backend-plans/rm-plan-2.md"] {
		t.Error("roadmap graph: plan-backend artifact should not have an assigned edge")
	}

}
