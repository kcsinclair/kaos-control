// SPDX-License-Identifier: AGPL-3.0-or-later

package index

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
)

// mustUpsertParsed parses raw markdown at relPath and upserts it, failing the
// test on any parse error or upsert error.
func mustUpsertParsed(t *testing.T, idx *Index, raw, relPath string) {
	t.Helper()
	a := artifact.Parse([]byte(raw), relPath, time.Now())
	if len(a.ParseErrs) != 0 {
		t.Fatalf("unexpected parse errors for %s: %v", relPath, a.ParseErrs)
	}
	if err := idx.Upsert(a); err != nil {
		t.Fatalf("Upsert(%s): %v", relPath, err)
	}
}

// archRaw builds a minimal type: architecture fixture with the given title.
func archRaw(title string) string {
	return "---\ntitle: " + title + "\ntype: architecture\nstatus: approved\n---\n\nBody.\n"
}

// TestArchitectureMap_NodesScopedToArchitectureType verifies FR-2: the base
// map contains exactly one node per type: architecture artifact and no nodes
// of any other type.
func TestArchitectureMap_NodesScopedToArchitectureType(t *testing.T) {
	idx := openTestIndex(t)
	mustUpsertParsed(t, idx, archRaw("Foo"), "lifecycle/architecture/architectures/foo.md")
	mustUpsertParsed(t, idx, archRaw("Bar"), "lifecycle/architecture/architectures/bar.md")
	mustUpsertParsed(t, idx,
		"---\ntitle: An Idea\ntype: idea\nstatus: draft\nlineage: an-idea\n---\n\nBody.\n",
		"lifecycle/ideas/an-idea.md")

	data, err := idx.ArchitectureMap("")
	if err != nil {
		t.Fatalf("ArchitectureMap: %v", err)
	}
	if len(data.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d: %+v", len(data.Nodes), data.Nodes)
	}
	for _, n := range data.Nodes {
		if n.Type != "architecture" {
			t.Errorf("unexpected node type %q in map", n.Type)
		}
	}
}

// TestGraph_IncludesSummaryFromFrontmatter verifies the graph tooltip data
// carries the summary: frontmatter field (extracted from the stored blob).
func TestGraph_IncludesSummaryFromFrontmatter(t *testing.T) {
	idx := openTestIndex(t)
	raw := "---\ntitle: Modular Monolith\ntype: architecture\nstatus: approved\n" +
		"lineage: arch-modular-monolith\n" +
		"summary: A single deployable organised into well-bounded modules.\n" +
		"---\n\nBody.\n"
	mustUpsertParsed(t, idx, raw, "lifecycle/architecture/architectures/modular-monolith.md")

	data, err := idx.Graph(Filter{})
	if err != nil {
		t.Fatalf("Graph: %v", err)
	}
	var found *GraphNode
	for _, n := range data.Nodes {
		if n.Title == "Modular Monolith" {
			found = n
			break
		}
	}
	if found == nil {
		t.Fatal("node not found in graph")
	}
	if found.Summary != "A single deployable organised into well-bounded modules." {
		t.Errorf("Summary = %q, want the frontmatter summary", found.Summary)
	}
}

// TestArchitectureMap_WikiLinkCollapsesToRelated verifies FR-3: a pair linked
// only by a body wiki-link appears as a single generic "related" edge.
func TestArchitectureMap_WikiLinkCollapsesToRelated(t *testing.T) {
	idx := openTestIndex(t)
	mustUpsertParsed(t, idx,
		"---\ntitle: Foo\ntype: architecture\nstatus: approved\n---\n\nSee [[bar]].\n",
		"lifecycle/architecture/architectures/foo.md")
	mustUpsertParsed(t, idx, archRaw("Bar"), "lifecycle/architecture/architectures/bar.md")

	data, err := idx.ArchitectureMap("")
	if err != nil {
		t.Fatalf("ArchitectureMap: %v", err)
	}
	if len(data.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d: %+v", len(data.Edges), data.Edges)
	}
	if e := data.Edges[0]; e.Kind != "related" || e.Label != "" {
		t.Errorf("expected a generic related edge, got %+v", e)
	}
}

// TestArchitectureMap_TypedFieldDegradesToGenericOnRemoval verifies FR-4: a
// typed evolves_into field classifies the edge with a non-empty label, and
// removing it degrades the same pair (still linked by a wiki-link) to a
// generic "related" edge with no error.
func TestArchitectureMap_TypedFieldDegradesToGenericOnRemoval(t *testing.T) {
	idx := openTestIndex(t)
	withTyped := "---\ntitle: Foo\ntype: architecture\nstatus: approved\n" +
		"evolves_into:\n  - architecture/architectures/bar.md\n---\n\nSee [[bar]].\n"
	mustUpsertParsed(t, idx, withTyped, "lifecycle/architecture/architectures/foo.md")
	mustUpsertParsed(t, idx, archRaw("Bar"), "lifecycle/architecture/architectures/bar.md")

	data, err := idx.ArchitectureMap("")
	if err != nil {
		t.Fatalf("ArchitectureMap: %v", err)
	}
	if len(data.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d: %+v", len(data.Edges), data.Edges)
	}
	if e := data.Edges[0]; e.Kind != artifact.EdgeKindEvolvesInto || e.Label == "" {
		t.Errorf("expected a typed evolves_into edge with a non-empty label, got %+v", e)
	}

	withoutTyped := "---\ntitle: Foo\ntype: architecture\nstatus: approved\n---\n\nSee [[bar]].\n"
	mustUpsertParsed(t, idx, withoutTyped, "lifecycle/architecture/architectures/foo.md")

	data2, err := idx.ArchitectureMap("")
	if err != nil {
		t.Fatalf("ArchitectureMap after removing evolves_into: %v", err)
	}
	if len(data2.Edges) != 1 {
		t.Fatalf("expected 1 edge after degrade, got %d: %+v", len(data2.Edges), data2.Edges)
	}
	if e := data2.Edges[0]; e.Kind != "related" {
		t.Errorf("expected degrade to a generic related edge, got %+v", e)
	}
}

// TestArchitectureMap_StackForAddsTechStackRing verifies FR-8/NFR-2:
// stack_for adds only the named architecture's related_to tech-stack nodes
// and connecting edges, and nothing for any other architecture.
func TestArchitectureMap_StackForAddsTechStackRing(t *testing.T) {
	idx := openTestIndex(t)
	fooID := "lifecycle/architecture/architectures/foo.md"
	barID := "lifecycle/architecture/architectures/bar.md"
	stackID := "lifecycle/architecture/tech-stacks/go-postgres.md"

	mustUpsertParsed(t, idx,
		"---\ntitle: Foo\ntype: architecture\nstatus: approved\n"+
			"related_to:\n  - architecture/tech-stacks/go-postgres.md\n---\n\nBody.\n",
		fooID)
	mustUpsertParsed(t, idx, archRaw("Bar"), barID)
	mustUpsertParsed(t, idx,
		"---\ntitle: Go + Postgres\ntype: tech-stack\nstatus: approved\n---\n\nBody.\n",
		stackID)

	data, err := idx.ArchitectureMap(fooID)
	if err != nil {
		t.Fatalf("ArchitectureMap: %v", err)
	}
	if len(data.Nodes) != 3 {
		t.Fatalf("expected 3 nodes (2 arch + 1 stack), got %d: %+v", len(data.Nodes), data.Nodes)
	}
	var foundStackNode, foundStackEdge bool
	for _, n := range data.Nodes {
		if n.ID == stackID {
			foundStackNode = true
			if n.Type != "tech-stack" {
				t.Errorf("stack node type: want tech-stack, got %q", n.Type)
			}
		}
	}
	for _, e := range data.Edges {
		if e.Kind == artifact.EdgeKindRelatedTo && e.Source == fooID && e.Target == stackID {
			foundStackEdge = true
		}
	}
	if !foundStackNode {
		t.Errorf("expected stack node in payload, got nodes: %+v", data.Nodes)
	}
	if !foundStackEdge {
		t.Errorf("expected related_to edge foo->stack, got edges: %+v", data.Edges)
	}

	// Nothing added for a different architecture.
	dataBar, err := idx.ArchitectureMap(barID)
	if err != nil {
		t.Fatalf("ArchitectureMap(bar): %v", err)
	}
	if len(dataBar.Nodes) != 2 {
		t.Errorf("expected no stack ring for bar, got %d nodes: %+v", len(dataBar.Nodes), dataBar.Nodes)
	}
}

// TestArchitectureMap_UnknownStackForReturnsBaseMapNoError verifies that a
// stack_for value that is not an architecture node yields the base map, not
// an error (NFR-5).
func TestArchitectureMap_UnknownStackForReturnsBaseMapNoError(t *testing.T) {
	idx := openTestIndex(t)
	mustUpsertParsed(t, idx, archRaw("Foo"), "lifecycle/architecture/architectures/foo.md")

	data, err := idx.ArchitectureMap("lifecycle/does/not/exist.md")
	if err != nil {
		t.Fatalf("ArchitectureMap: %v", err)
	}
	if len(data.Nodes) != 1 {
		t.Errorf("expected base map (1 node), got %d: %+v", len(data.Nodes), data.Nodes)
	}
}

// TestArchitectureMap_DanglingLinkDropped verifies that a wiki-link target
// that does not resolve to an indexed architecture node is dropped from the
// map rather than rendered as a dangling node (NFR-5).
func TestArchitectureMap_DanglingLinkDropped(t *testing.T) {
	idx := openTestIndex(t)
	mustUpsertParsed(t, idx,
		"---\ntitle: Foo\ntype: architecture\nstatus: approved\n---\n\nSee [[missing]].\n",
		"lifecycle/architecture/architectures/foo.md")

	data, err := idx.ArchitectureMap("")
	if err != nil {
		t.Fatalf("ArchitectureMap: %v", err)
	}
	if len(data.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d: %+v", len(data.Nodes), data.Nodes)
	}
	if len(data.Edges) != 0 {
		t.Errorf("expected no edges for a dangling link, got %+v", data.Edges)
	}
}

// TestArchitectureMap_ReflectsReindexWithoutRestart verifies FR-12: a new
// type: architecture fixture written to disk and indexed via IndexFile — the
// same mechanism the fsnotify watcher uses on a change event — is visible in
// the very next ArchitectureMap("") call, with no cached snapshot in the way.
func TestArchitectureMap_ReflectsReindexWithoutRestart(t *testing.T) {
	idx := openTestIndex(t)

	data, err := idx.ArchitectureMap("")
	if err != nil {
		t.Fatalf("ArchitectureMap (before): %v", err)
	}
	if len(data.Nodes) != 0 {
		t.Fatalf("expected 0 nodes before any fixture is written, got %d", len(data.Nodes))
	}

	relPath := "lifecycle/architecture/architectures/foo.md"
	absPath := filepath.Join(idx.projectRoot, relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absPath, []byte(archRaw("Foo")), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := idx.IndexFile(absPath); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	data2, err := idx.ArchitectureMap("")
	if err != nil {
		t.Fatalf("ArchitectureMap (after): %v", err)
	}
	if len(data2.Nodes) != 1 || data2.Nodes[0].ID != relPath {
		t.Fatalf("expected the new node visible with no restart, got nodes: %+v", data2.Nodes)
	}
}
