// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"net/http"
	"testing"
)

// TestGraphPriorityPresent verifies that a node for an artifact with
// priority: high carries that value in the graph response.
func TestGraphPriorityPresent(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/gprio-high.md",
			content: makeArtifactWithPriority("Graph Priority High", "idea", "draft", "gprio-high", "high", "Body."),
		},
	}
	env := newTestEnv(t, seeds)

	resp, err := http.Get(env.baseURL + "/api/p/testproject/graph")
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	nodes := decodeGraphNodes(t, data)

	node := findNodeByID(nodes, "lifecycle/ideas/gprio-high.md")
	if node == nil {
		t.Fatal("expected node for lifecycle/ideas/gprio-high.md")
	}
	priority, _ := node["priority"].(string)
	if priority != "high" {
		t.Errorf("graph node priority: want %q, got %q", "high", priority)
	}
}

// TestGraphPriorityAbsent verifies that a node for an artifact with no
// priority field has priority "" or the field omitted — and does not error.
func TestGraphPriorityAbsent(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/gprio-absent.md",
			content: makeArtifact("Graph Priority Absent", "idea", "draft", "gprio-absent", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)

	resp, err := http.Get(env.baseURL + "/api/p/testproject/graph")
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	nodes := decodeGraphNodes(t, data)

	node := findNodeByID(nodes, "lifecycle/ideas/gprio-absent.md")
	if node == nil {
		t.Fatal("expected node for lifecycle/ideas/gprio-absent.md")
	}

	// priority is omitempty on GraphNode — either absent or "" is acceptable.
	if raw, exists := node["priority"]; exists && raw != "" && raw != nil {
		t.Errorf("graph node priority should be absent or empty, got %v", raw)
	}
}

// TestGraphPriorityAfterPatch creates an artifact with priority: low, PATCHes
// it to medium, then verifies the graph node reflects the update.
func TestGraphPriorityAfterPatch(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/gprio-patch.md",
			content: makeArtifactWithPriority("Graph Priority After Patch", "idea", "draft", "gprio-patch", "low", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const path = "lifecycle/ideas/gprio-patch.md"

	resp := env.doRequest("PATCH", "/api/p/testproject/artifacts/"+path+"/priority", map[string]any{
		"priority": "medium",
	})
	requireStatus(t, resp, 200)
	resp.Body.Close()

	// The graph should now reflect medium priority.
	graphData := graphResponseForProject(t, env)
	nodes := decodeGraphNodes(t, graphData)

	node := findNodeByID(nodes, path)
	if node == nil {
		t.Fatal("expected node in graph after PATCH")
	}
	priority, _ := node["priority"].(string)
	if priority != "medium" {
		t.Errorf("graph node priority after PATCH: want %q, got %q", "medium", priority)
	}
}
