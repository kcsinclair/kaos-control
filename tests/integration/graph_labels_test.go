// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"testing"
)

// TestGraphLabelsNormalisedWhenAbsent verifies that an artifact with no labels
// field in its frontmatter still has labels: [] (not null or absent) in the
// graph API response.
func TestGraphLabelsNormalisedWhenAbsent(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/nolabels.md",
			content: makeArtifact("No Labels", "idea", "draft", "nolabels", "", "An artifact with no labels."),
		},
	}
	env := newTestEnv(t, seeds)

	data := graphResponseForProject(t, env)
	nodes := decodeGraphNodes(t, data)

	node := findNodeByID(nodes, "lifecycle/ideas/nolabels.md")
	if node == nil {
		t.Fatal("expected node for lifecycle/ideas/nolabels.md in graph response")
	}

	// The labels field must be present and be an empty array, not null/absent.
	raw, exists := node["labels"]
	if !exists {
		t.Fatal("graph node missing 'labels' field entirely")
	}
	if raw == nil {
		t.Fatal("graph node 'labels' is null; want an empty array []")
	}
	labels, ok := raw.([]any)
	if !ok {
		t.Fatalf("graph node 'labels' is not an array, got %T", raw)
	}
	if len(labels) != 0 {
		t.Errorf("expected empty labels array, got %v", labels)
	}
}

// TestGraphLabelsPresent verifies that an artifact with labels in its
// frontmatter returns those labels correctly in the graph node.
func TestGraphLabelsPresent(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/withlabels.md",
			content: makeArtifact("With Labels", "idea", "draft", "withlabels", "", "Has labels.", "auth", "backend"),
		},
	}
	env := newTestEnv(t, seeds)

	data := graphResponseForProject(t, env)
	nodes := decodeGraphNodes(t, data)

	node := findNodeByID(nodes, "lifecycle/ideas/withlabels.md")
	if node == nil {
		t.Fatal("expected node for lifecycle/ideas/withlabels.md in graph response")
	}

	labels := graphNodeLabels(node)
	if labels == nil {
		t.Fatal("graph node 'labels' is missing or null")
	}
	wantLabels := map[string]bool{"auth": true, "backend": true}
	if len(labels) != len(wantLabels) {
		t.Errorf("expected %d labels, got %d: %v", len(wantLabels), len(labels), labels)
	}
	for _, l := range labels {
		label, _ := l.(string)
		if !wantLabels[label] {
			t.Errorf("unexpected label %q in graph node", label)
		}
	}
}

// TestGraphLabelsMixedSet verifies that in a graph response containing both
// labelled and unlabelled artifacts, every node has a non-null labels array.
func TestGraphLabelsMixedSet(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/mixed-a.md",
			content: makeArtifact("Mixed A", "idea", "draft", "mixed-a", "", "No labels."),
		},
		{
			relPath: "lifecycle/ideas/mixed-b.md",
			content: makeArtifact("Mixed B", "idea", "draft", "mixed-b", "", "Has labels.", "frontend"),
		},
		{
			relPath: "lifecycle/requirements/mixed-c.md",
			content: makeArtifact("Mixed C", "ticket", "draft", "mixed-c", "", "Also no labels."),
		},
		{
			relPath: "lifecycle/requirements/mixed-d.md",
			content: makeArtifact("Mixed D", "ticket", "draft", "mixed-d", "", "Two labels.", "auth", "backend"),
		},
	}
	env := newTestEnv(t, seeds)

	resp := env.doRequest("GET", "/api/p/testproject/graph", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	nodes := decodeGraphNodes(t, data)

	if len(nodes) != 4 {
		t.Errorf("expected 4 nodes, got %d", len(nodes))
	}

	for _, n := range nodes {
		node, _ := n.(map[string]any)
		id, _ := node["id"].(string)

		raw, exists := node["labels"]
		if !exists {
			t.Errorf("node %q missing 'labels' field", id)
			continue
		}
		if raw == nil {
			t.Errorf("node %q has labels: null", id)
			continue
		}
		if _, ok := raw.([]any); !ok {
			t.Errorf("node %q labels is not an array, got %T", id, raw)
		}
	}
}
