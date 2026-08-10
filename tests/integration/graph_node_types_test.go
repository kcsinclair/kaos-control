// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"fmt"
	"testing"
)

// specTypes lists all 12 artifact type values defined in the spec vocabulary.
var specTypes = []string{
	"idea",
	"ticket",
	"epic",
	"plan-backend",
	"plan-frontend",
	"plan-dev",
	"plan-test",
	"test",
	"prototype",
	"release",
	"sprint",
	"defect",
}

// graphNodeTypes is specTypes minus "release". Since release-artefacts-9 (DR-4)
// releases are single-cached in the dedicated `releases` table and are NOT
// indexed into the artifacts table, so they never appear as nodes in the main
// artifact graph (they live on the roadmap graph instead — see
// graph_releases_test.go). The remaining 11 types are still artifact graph nodes.
var graphNodeTypes = func() []string {
	out := make([]string, 0, len(specTypes))
	for _, typ := range specTypes {
		if typ == "release" {
			continue
		}
		out = append(out, typ)
	}
	return out
}()

// specTypeStage maps each spec type to a lifecycle stage directory that
// will hold it in the test project. Types without a dedicated stage share
// a stage with a closely related type (the type field in frontmatter is
// what the graph uses, not the directory name).
var specTypeStage = map[string]string{
	"idea":          "lifecycle/ideas",
	"ticket":        "lifecycle/requirements",
	"epic":          "lifecycle/requirements",
	"plan-backend":  "lifecycle/backend-plans",
	"plan-frontend": "lifecycle/frontend-plans",
	"plan-dev":      "lifecycle/backend-plans",
	"plan-test":     "lifecycle/test-plans",
	"test":          "lifecycle/tests",
	"prototype":     "lifecycle/prototypes",
	"release":       "lifecycle/releases",
	"sprint":        "lifecycle/sprints",
	"defect":        "lifecycle/defects",
}

// TestAllSpecTypesInGraph creates one artifact of each graph-visible spec type
// (all spec types except "release", which is not an artifact graph node since
// DR-4), calls GET /graph, and verifies each node appears with the correct type
// field.
func TestAllSpecTypesInGraph(t *testing.T) {
	seeds := make([]seedArtifact, 0, len(graphNodeTypes))
	for _, typ := range graphNodeTypes {
		slug := "nt-" + typ
		stage := specTypeStage[typ]
		seeds = append(seeds, seedArtifact{
			relPath: fmt.Sprintf("%s/%s.md", stage, slug),
			content: makeArtifact(
				"Node Type "+typ, typ, "draft", slug, "", "Body for "+typ+" artifact.",
			),
		})
	}

	env := newTestEnv(t, seeds)

	data := graphResponseForProject(t, env)
	nodes := decodeGraphNodes(t, data)

	// Build id → node map for O(1) lookup.
	nodeByID := map[string]map[string]any{}
	for _, n := range nodes {
		node, _ := n.(map[string]any)
		if id, _ := node["id"].(string); id != "" {
			nodeByID[id] = node
		}
	}

	if len(nodes) != len(graphNodeTypes) {
		t.Errorf("graph node count: want %d, got %d", len(graphNodeTypes), len(nodes))
	}

	for _, typ := range graphNodeTypes {
		slug := "nt-" + typ
		stage := specTypeStage[typ]
		path := fmt.Sprintf("%s/%s.md", stage, slug)

		node, ok := nodeByID[path]
		if !ok {
			t.Errorf("missing graph node for type %q (expected path %q)", typ, path)
			continue
		}

		gotType, _ := node["type"].(string)
		if gotType == "" {
			t.Errorf("node %q has empty type field", path)
		} else if gotType != typ {
			t.Errorf("node %q type: want %q, got %q", path, typ, gotType)
		}
	}
}

// TestTypeFieldAccuracy verifies that for a seeded set of all spec types, each
// graph node's type field matches exactly the frontmatter type that was written.
func TestTypeFieldAccuracy(t *testing.T) {
	seeds := make([]seedArtifact, 0, len(graphNodeTypes))
	for _, typ := range graphNodeTypes {
		slug := "ta-" + typ
		stage := specTypeStage[typ]
		seeds = append(seeds, seedArtifact{
			relPath: fmt.Sprintf("%s/%s.md", stage, slug),
			content: makeArtifact(
				"Type Accuracy "+typ, typ, "draft", slug, "", "Body.",
			),
		})
	}

	env := newTestEnv(t, seeds)

	data := graphResponseForProject(t, env)
	nodes := decodeGraphNodes(t, data)

	nodeByID := map[string]map[string]any{}
	for _, n := range nodes {
		node, _ := n.(map[string]any)
		if id, _ := node["id"].(string); id != "" {
			nodeByID[id] = node
		}
	}

	for _, typ := range graphNodeTypes {
		slug := "ta-" + typ
		stage := specTypeStage[typ]
		path := fmt.Sprintf("%s/%s.md", stage, slug)

		node, ok := nodeByID[path]
		if !ok {
			t.Errorf("missing graph node for type %q", typ)
			continue
		}

		gotType, _ := node["type"].(string)
		if gotType != typ {
			t.Errorf("node %q: frontmatter type %q does not match graph node type %q", path, typ, gotType)
		}
	}
}
