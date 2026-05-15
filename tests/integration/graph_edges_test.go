// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestone 5 — Backend constant consistency check
//
// Verifies that:
//   TC12: The graph API returns edges whose "kind" values match the canonical
//         edge-kind constants defined in internal/artifact/artifact.go.
//   TC13: Each edge in the graph response has exactly the expected fields
//         (source, target, kind, and optionally label) — no new fields added.
//
// Run with:
//
//	go test ./tests/integration/ -tags integration -run TestGraphEdges

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// canonicalEdgeKinds is the set of valid edge kind values as defined in
// internal/artifact/artifact.go (EdgeKind* constants).
var canonicalEdgeKinds = map[string]bool{
	"parent":     true,
	"depends_on": true,
	"blocks":     true,
	"related_to": true,
	"members":    true,
	"wiki":       true,
	"assigned":   true,
	"timeline":   true,
}

// makeArtifactWithRelationships builds a markdown artifact string whose
// frontmatter includes all the standard relationship fields and whose body
// contains a wiki-style [[slug]] link.
func makeArtifactWithRelationships(
	title, typ, status, lineage, parent string,
	dependsOn, blocks, relatedTo, members []string,
	body string,
) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("title: " + title + "\n")
	sb.WriteString("type: " + typ + "\n")
	sb.WriteString("status: " + status + "\n")
	sb.WriteString("lineage: " + lineage + "\n")
	if parent != "" {
		sb.WriteString("parent: " + parent + "\n")
	}
	for _, field := range []struct {
		name   string
		values []string
	}{
		{"depends_on", dependsOn},
		{"blocks", blocks},
		{"related_to", relatedTo},
		{"members", members},
	} {
		if len(field.values) > 0 {
			sb.WriteString(field.name + ":\n")
			for _, v := range field.values {
				sb.WriteString("  - " + v + "\n")
			}
		}
	}
	sb.WriteString("---\n\n")
	sb.WriteString(body + "\n")
	return sb.String()
}

// decodeGraphEdges extracts the "edges" slice from a raw graph JSON map.
func decodeGraphEdges(t *testing.T, data map[string]any) []any {
	t.Helper()
	edges, ok := data["edges"].([]any)
	if !ok {
		// edges may be null/missing when there are no relationships — return empty
		return nil
	}
	return edges
}

// graphEdgeKind returns the "kind" field of a raw edge map.
func graphEdgeKind(edge map[string]any) string {
	kind, _ := edge["kind"].(string)
	return kind
}

// TestGraphEdges_KindValues (TC12): seeds one artifact of each relationship
// kind, calls GET /graph, and asserts that every edge's "kind" field matches
// one of the canonical constants.
func TestGraphEdges_KindValues(t *testing.T) {
	// Base artifact — will be referenced by parent / depends_on / blocks /
	// related_to / members links from the child artifact.
	baseIdeaPath := "lifecycle/ideas/ge-base.md"
	depPath := "lifecycle/backend-plans/ge-dep-3-be.md"
	blocksPath := "lifecycle/backend-plans/ge-blocks-4-be.md"
	relatedPath := "lifecycle/backend-plans/ge-related-5-be.md"
	memberPath := "lifecycle/backend-plans/ge-member-6-be.md"
	wikiPath := "lifecycle/ideas/ge-wiki.md" // referenced via [[ge-wiki]] body link

	seeds := []seedArtifact{
		// Base idea
		{
			relPath: baseIdeaPath,
			content: makeArtifact("GE Base", "idea", "draft", "ge-base", "", "Base artifact body."),
		},
		// Dep target
		{
			relPath: depPath,
			content: makeArtifact("GE Dep", "plan-backend", "draft", "ge-dep", "", "Dep artifact body."),
		},
		// Blocks target
		{
			relPath: blocksPath,
			content: makeArtifact("GE Blocks", "plan-backend", "draft", "ge-blocks", "", "Blocks artifact body."),
		},
		// Related target
		{
			relPath: relatedPath,
			content: makeArtifact("GE Related", "plan-backend", "draft", "ge-related", "", "Related artifact body."),
		},
		// Member target
		{
			relPath: memberPath,
			content: makeArtifact("GE Member", "plan-backend", "draft", "ge-member", "", "Member artifact body."),
		},
		// Wiki link target (same stage as child so [[slug]] resolves correctly)
		{
			relPath: wikiPath,
			content: makeArtifact("GE Wiki", "idea", "draft", "ge-wiki", "", "Wiki target artifact body."),
		},
		// Child artifact with all relationship types
		{
			relPath: "lifecycle/requirements/ge-child-2.md",
			content: makeArtifactWithRelationships(
				"GE Child", "ticket", "draft", "ge-child", baseIdeaPath,
				[]string{depPath},
				[]string{blocksPath},
				[]string{relatedPath},
				[]string{memberPath},
				// Body contains a wiki link — resolves relative to this file's stage dir
				// (lifecycle/requirements/), so [[ge-wiki]] → lifecycle/requirements/ge-wiki.md,
				// but ge-wiki.md lives in ideas/. Use full path format instead so the
				// parser resolves it correctly via the lifecycle/ prefix logic.
				fmt.Sprintf("Depends on, blocks, relates, includes, and links to [[%s]].", "ideas/ge-wiki"),
			),
		},
	}

	env := newTestEnv(t, seeds)
	// newTestEnv auto-logs in as admin — no extra login needed.

	resp := env.doRequest("GET", "/api/p/testproject/graph", nil)
	requireStatus(t, resp, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}

	edges := decodeGraphEdges(t, data)

	// Collect the kinds that actually appear in the response.
	seenKinds := map[string]bool{}
	for _, raw := range edges {
		edge, _ := raw.(map[string]any)
		kind := graphEdgeKind(edge)
		seenKinds[kind] = true

		if !canonicalEdgeKinds[kind] {
			t.Errorf("edge has non-canonical kind %q (source=%v, target=%v)",
				kind, edge["source"], edge["target"])
		}
	}

	// Assert that at minimum the relationship-specific kinds seeded above appear.
	expectedSeeded := []string{"parent", "depends_on", "blocks", "related_to", "members", "wiki"}
	for _, want := range expectedSeeded {
		if !seenKinds[want] {
			t.Errorf("expected edge kind %q not found in graph response", want)
		}
	}
}

// TestGraphEdges_JSONShape (TC13): verifies that each edge object in the graph
// response has exactly the allowed fields: source, target, kind, and
// optionally label. No unexpected fields must appear.
func TestGraphEdges_JSONShape(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/shape-parent.md",
			content: makeArtifact("Shape Parent", "idea", "draft", "shape-parent", "", "Parent body."),
		},
		{
			relPath: "lifecycle/requirements/shape-child-2.md",
			content: makeArtifact("Shape Child", "ticket", "draft", "shape-child",
				"lifecycle/ideas/shape-parent.md", "Child body."),
		},
	}

	env := newTestEnv(t, seeds)

	resp := env.doRequest("GET", "/api/p/testproject/graph", nil)
	requireStatus(t, resp, http.StatusOK)

	raw, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}

	// Parse into a structure that preserves all JSON keys so we can check for
	// unexpected fields.
	var payload struct {
		Edges []map[string]json.RawMessage `json:"edges"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}

	// Allowed top-level fields on a GraphEdge.
	allowedFields := map[string]bool{
		"source": true,
		"target": true,
		"kind":   true,
		"label":  true, // optional — only present on timeline edges
	}

	// Required fields that must always be present.
	requiredFields := []string{"source", "target", "kind"}

	for i, edge := range payload.Edges {
		// Required fields
		for _, field := range requiredFields {
			if _, ok := edge[field]; !ok {
				t.Errorf("edge[%d]: missing required field %q", i, field)
			}
		}

		// No unexpected fields
		for key := range edge {
			if !allowedFields[key] {
				t.Errorf("edge[%d]: unexpected field %q in GraphEdge JSON", i, key)
			}
		}

		// source and target must be non-empty strings
		var src, dst, kind string
		if err := json.Unmarshal(edge["source"], &src); err != nil || src == "" {
			t.Errorf("edge[%d]: source must be a non-empty string", i)
		}
		if err := json.Unmarshal(edge["target"], &dst); err != nil || dst == "" {
			t.Errorf("edge[%d]: target must be a non-empty string", i)
		}
		if err := json.Unmarshal(edge["kind"], &kind); err != nil || kind == "" {
			t.Errorf("edge[%d]: kind must be a non-empty string", i)
		}
	}
}

// TestGraphEdges_ParentEdgePresent is a focused sanity check that a seeded
// parent relationship appears as a "parent" kind edge in the graph response.
func TestGraphEdges_ParentEdgePresent(t *testing.T) {
	parentPath := "lifecycle/ideas/ep-parent.md"
	childPath := "lifecycle/requirements/ep-child-2.md"

	seeds := []seedArtifact{
		{
			relPath: parentPath,
			content: makeArtifact("EP Parent", "idea", "draft", "ep-parent", "", "Parent body."),
		},
		{
			relPath: childPath,
			content: makeArtifact("EP Child", "ticket", "draft", "ep-child", parentPath, "Child body."),
		},
	}

	env := newTestEnv(t, seeds)

	data := graphResponseForProject(t, env)
	edges := decodeGraphEdges(t, data)

	found := false
	for _, raw := range edges {
		edge, _ := raw.(map[string]any)
		src, _ := edge["source"].(string)
		dst, _ := edge["target"].(string)
		kind := graphEdgeKind(edge)

		if src == childPath && dst == parentPath && kind == "parent" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected a parent-kind edge from %q to %q, not found in graph response", childPath, parentPath)
	}
}
