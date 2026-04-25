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

// TestAllSpecTypesInGraph creates one artifact of each of the 12 spec-defined
// types, calls GET /graph, and verifies all 12 nodes appear with the correct
// type field.
func TestAllSpecTypesInGraph(t *testing.T) {
	seeds := make([]seedArtifact, 0, len(specTypes))
	for _, typ := range specTypes {
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

	if len(nodes) != len(specTypes) {
		t.Errorf("graph node count: want %d, got %d", len(specTypes), len(nodes))
	}

	for _, typ := range specTypes {
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
	seeds := make([]seedArtifact, 0, len(specTypes))
	for _, typ := range specTypes {
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

	for _, typ := range specTypes {
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
