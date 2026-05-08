//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
)

// ─── Roadmap graph test helpers ───────────────────────────────────────────────

const backlogNodeID = "release:backlog"

// releaseNodeID returns the graph node ID for a release with the given DB id.
func releaseNodeID(id int64) string {
	return fmt.Sprintf("release:%d", id)
}

// roadmapGraph calls GET /api/p/testproject/releases/graph and returns the decoded JSON.
func roadmapGraph(t *testing.T, env *testEnv) map[string]any {
	t.Helper()
	resp := env.doRequest("GET", "/api/p/testproject/releases/graph", nil)
	requireStatus(t, resp, http.StatusOK)
	return readJSON(t, resp)
}

// timelineChain follows directed timeline edges starting from the Backlog node
// and returns the ordered slice of node IDs in the chain.
// Terminates after 200 steps as a cycle guard.
func timelineChain(edges []any) []string {
	next := make(map[string]string)
	for _, raw := range edges {
		edge, _ := raw.(map[string]any)
		if kind, _ := edge["kind"].(string); kind != "timeline" {
			continue
		}
		src, _ := edge["source"].(string)
		tgt, _ := edge["target"].(string)
		next[src] = tgt
	}
	chain := []string{backlogNodeID}
	cur := backlogNodeID
	for i := 0; i < 200; i++ {
		tgt, ok := next[cur]
		if !ok {
			break
		}
		chain = append(chain, tgt)
		cur = tgt
	}
	return chain
}

// findEdge returns the first edge matching source, target, and kind.
// Use kind="" to match any kind.
func findEdge(edges []any, src, tgt, kind string) map[string]any {
	for _, raw := range edges {
		edge, _ := raw.(map[string]any)
		s, _ := edge["source"].(string)
		tt, _ := edge["target"].(string)
		if s != src || tt != tgt {
			continue
		}
		if kind != "" {
			if k, _ := edge["kind"].(string); k != kind {
				continue
			}
		}
		return edge
	}
	return nil
}

// countEdgesByKind returns the number of edges with the given kind.
func countEdgesByKind(edges []any, kind string) int {
	n := 0
	for _, raw := range edges {
		edge, _ := raw.(map[string]any)
		if k, _ := edge["kind"].(string); k == kind {
			n++
		}
	}
	return n
}

// makeArtifactWithDependsOn builds a markdown artifact that includes a
// depends_on frontmatter field pointing to a single artifact path.
func makeArtifactWithDependsOn(title, typ, status, lineage, release, dependsOnPath string) string {
	s := "---\ntitle: " + title + "\ntype: " + typ + "\nstatus: " + status + "\nlineage: " + lineage + "\n"
	if release != "" {
		s += "release: " + release + "\n"
	}
	if dependsOnPath != "" {
		s += "depends_on:\n  - " + dependsOnPath + "\n"
	}
	s += "---\n\nBody.\n"
	return s
}

// ─── Milestone 1: Chain construction logic ────────────────────────────────────

// TestRoadmapGraph_EmptyState verifies that with no releases, the response
// contains exactly the synthetic Backlog node and no edges.
func TestRoadmapGraph_EmptyState(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	data := roadmapGraph(t, env)
	nodes, _ := data["nodes"].([]any)
	edges, _ := data["edges"].([]any)

	// Must contain exactly one node: the Backlog.
	if len(nodes) != 1 {
		t.Errorf("empty state: want 1 node (Backlog), got %d", len(nodes))
	}
	if len(nodes) > 0 {
		node, _ := nodes[0].(map[string]any)
		if id, _ := node["id"].(string); id != backlogNodeID {
			t.Errorf("empty state: sole node id: want %q, got %q", backlogNodeID, id)
		}
		if synthetic, _ := node["synthetic"].(bool); !synthetic {
			t.Error("empty state: Backlog node must have synthetic=true")
		}
	}

	// No edges.
	if len(edges) != 0 {
		t.Errorf("empty state: want 0 edges, got %d", len(edges))
	}
}

// TestRoadmapGraph_BacklogNodeAlwaysPresent verifies that the Backlog node is
// always present with the correct fields, even when releases exist.
func TestRoadmapGraph_BacklogNodeAlwaysPresent(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{
		"name": "rg-bap-v1", "status": "planned",
		"start_date": "2026-01-01", "end_date": "2026-03-31",
	})

	data := roadmapGraph(t, env)
	nodes, _ := data["nodes"].([]any)

	backlog := findNodeByID(nodes, backlogNodeID)
	if backlog == nil {
		t.Fatal("Backlog node not found in graph response")
	}
	if title, _ := backlog["title"].(string); title != "Backlog" {
		t.Errorf("Backlog node title: want %q, got %q", "Backlog", title)
	}
	if typ, _ := backlog["type"].(string); typ != "release" {
		t.Errorf("Backlog node type: want %q, got %q", "release", typ)
	}
	if synthetic, _ := backlog["synthetic"].(bool); !synthetic {
		t.Error("Backlog node must have synthetic=true")
	}
}

// TestRoadmapGraph_SingleScheduled verifies that a single scheduled release
// produces exactly one timeline edge: Backlog → release.
func TestRoadmapGraph_SingleScheduled(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{
		"name": "rg-ss-v1", "status": "planned",
		"start_date": "2026-01-01", "end_date": "2026-03-31",
	})
	id := releaseID(t, data)
	nodeID := releaseNodeID(id)

	graph := roadmapGraph(t, env)
	edges, _ := graph["edges"].([]any)

	chain := timelineChain(edges)
	// Expected chain: [backlog, nodeID]
	if len(chain) != 2 {
		t.Fatalf("single scheduled: want chain length 2, got %d: %v", len(chain), chain)
	}
	if chain[0] != backlogNodeID {
		t.Errorf("chain[0]: want %q, got %q", backlogNodeID, chain[0])
	}
	if chain[1] != nodeID {
		t.Errorf("chain[1]: want %q, got %q", nodeID, chain[1])
	}
}

// TestRoadmapGraph_MultipleScheduledChronological verifies that three scheduled
// releases are connected in ascending start_date order:
// Backlog → R1(Jan) → R2(Apr) → R3(Jul).
func TestRoadmapGraph_MultipleScheduledChronological(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Create in reverse chronological order to exercise sorting.
	d3 := createRelease(t, env, map[string]any{
		"name": "rg-msc-v3", "status": "planned",
		"start_date": "2026-07-01", "end_date": "2026-09-30",
	})
	d1 := createRelease(t, env, map[string]any{
		"name": "rg-msc-v1", "status": "planned",
		"start_date": "2026-01-01", "end_date": "2026-03-31",
	})
	d2 := createRelease(t, env, map[string]any{
		"name": "rg-msc-v2", "status": "planned",
		"start_date": "2026-04-01", "end_date": "2026-06-30",
	})

	id1 := releaseNodeID(releaseID(t, d1))
	id2 := releaseNodeID(releaseID(t, d2))
	id3 := releaseNodeID(releaseID(t, d3))

	graph := roadmapGraph(t, env)
	edges, _ := graph["edges"].([]any)

	chain := timelineChain(edges)
	want := []string{backlogNodeID, id1, id2, id3}

	if len(chain) != len(want) {
		t.Fatalf("chronological chain length: want %d, got %d: %v", len(want), len(chain), chain)
	}
	for i, w := range want {
		if chain[i] != w {
			t.Errorf("chain[%d]: want %q, got %q", i, w, chain[i])
		}
	}
}

// TestRoadmapGraph_TieBreakingAlphabetical verifies that two scheduled releases
// sharing the same start_date are ordered alphabetically by name.
func TestRoadmapGraph_TieBreakingAlphabetical(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Both have the same start_date; "rg-tie-aaa" < "rg-tie-zzz".
	dZ := createRelease(t, env, map[string]any{
		"name": "rg-tie-zzz", "status": "planned", "start_date": "2026-06-01",
	})
	dA := createRelease(t, env, map[string]any{
		"name": "rg-tie-aaa", "status": "planned", "start_date": "2026-06-01",
	})

	idA := releaseNodeID(releaseID(t, dA))
	idZ := releaseNodeID(releaseID(t, dZ))

	graph := roadmapGraph(t, env)
	edges, _ := graph["edges"].([]any)

	chain := timelineChain(edges)
	want := []string{backlogNodeID, idA, idZ}

	if len(chain) != len(want) {
		t.Fatalf("tie-breaking chain length: want %d, got %d: %v", len(want), len(chain), chain)
	}
	for i, w := range want {
		if chain[i] != w {
			t.Errorf("chain[%d]: want %q (alphabetically first), got %q", i, w, chain[i])
		}
	}
}

// TestRoadmapGraph_SingleUnscheduled verifies that a single unscheduled release
// appears as a terminal leaf connected from the Backlog via a timeline edge.
func TestRoadmapGraph_SingleUnscheduled(t *testing.T) {
	t.Skip("documents the older chained-undated roadmap spec; superseded by the unscheduled-terminus pattern in TestGraphReleases_*. Resolve which spec is canonical and unskip.")
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	d := createRelease(t, env, map[string]any{"name": "rg-su-v1", "status": "planned"})
	nodeID := releaseNodeID(releaseID(t, d))

	graph := roadmapGraph(t, env)
	edges, _ := graph["edges"].([]any)

	chain := timelineChain(edges)
	want := []string{backlogNodeID, nodeID}

	if len(chain) != len(want) {
		t.Fatalf("single unscheduled chain length: want %d, got %d: %v", len(want), len(chain), chain)
	}
	if chain[1] != nodeID {
		t.Errorf("single unscheduled: chain[1]: want %q, got %q", nodeID, chain[1])
	}
}

// TestRoadmapGraph_MultipleUnscheduledAlphabetical verifies that multiple
// unscheduled releases are sorted alphabetically and connected in that order.
func TestRoadmapGraph_MultipleUnscheduledAlphabetical(t *testing.T) {
	t.Skip("documents the older chained-undated roadmap spec; superseded by the unscheduled-terminus pattern in TestGraphReleases_*. Resolve which spec is canonical and unskip.")
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Create in reverse alphabetical order.
	dZ := createRelease(t, env, map[string]any{"name": "rg-mua-zzz", "status": "planned"})
	dA := createRelease(t, env, map[string]any{"name": "rg-mua-aaa", "status": "planned"})

	idA := releaseNodeID(releaseID(t, dA))
	idZ := releaseNodeID(releaseID(t, dZ))

	graph := roadmapGraph(t, env)
	edges, _ := graph["edges"].([]any)

	chain := timelineChain(edges)
	want := []string{backlogNodeID, idA, idZ}

	if len(chain) != len(want) {
		t.Fatalf("multiple unscheduled chain length: want %d, got %d: %v", len(want), len(chain), chain)
	}
	for i, w := range want {
		if chain[i] != w {
			t.Errorf("chain[%d]: want %q, got %q", i, w, chain[i])
		}
	}
}

// TestRoadmapGraph_NoScheduledDirectToUnscheduled verifies that when there are
// no scheduled releases, the Backlog connects directly to the first unscheduled
// release (alphabetically), then to subsequent unscheduled ones.
func TestRoadmapGraph_NoScheduledDirectToUnscheduled(t *testing.T) {
	t.Skip("documents the older chained-undated roadmap spec; superseded by the unscheduled-terminus pattern in TestGraphReleases_*. Resolve which spec is canonical and unskip.")
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	dC := createRelease(t, env, map[string]any{"name": "rg-nsd-ccc", "status": "planned"})
	dA := createRelease(t, env, map[string]any{"name": "rg-nsd-aaa", "status": "planned"})
	dB := createRelease(t, env, map[string]any{"name": "rg-nsd-bbb", "status": "planned"})

	idA := releaseNodeID(releaseID(t, dA))
	idB := releaseNodeID(releaseID(t, dB))
	idC := releaseNodeID(releaseID(t, dC))

	graph := roadmapGraph(t, env)
	edges, _ := graph["edges"].([]any)

	chain := timelineChain(edges)
	want := []string{backlogNodeID, idA, idB, idC}

	if len(chain) != len(want) {
		t.Fatalf("no-scheduled chain length: want %d, got %d: %v", len(want), len(chain), chain)
	}
	for i, w := range want {
		if chain[i] != w {
			t.Errorf("chain[%d]: want %q, got %q", i, w, chain[i])
		}
	}
}

// TestRoadmapGraph_MixedScheduledAndUnscheduled verifies the complete chain:
// Backlog → scheduled (chronological) → unscheduled (alphabetical).
func TestRoadmapGraph_MixedScheduledAndUnscheduled(t *testing.T) {
	t.Skip("documents the older chained-undated roadmap spec; superseded by the unscheduled-terminus pattern in TestGraphReleases_*. Resolve which spec is canonical and unskip.")
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	dS2 := createRelease(t, env, map[string]any{
		"name": "rg-mix-s2", "status": "planned",
		"start_date": "2026-04-01", "end_date": "2026-06-30",
	})
	dS1 := createRelease(t, env, map[string]any{
		"name": "rg-mix-s1", "status": "planned",
		"start_date": "2026-01-01", "end_date": "2026-03-31",
	})
	dUZ := createRelease(t, env, map[string]any{"name": "rg-mix-uz", "status": "planned"})
	dUA := createRelease(t, env, map[string]any{"name": "rg-mix-ua", "status": "planned"})

	idS1 := releaseNodeID(releaseID(t, dS1))
	idS2 := releaseNodeID(releaseID(t, dS2))
	idUA := releaseNodeID(releaseID(t, dUA))
	idUZ := releaseNodeID(releaseID(t, dUZ))

	graph := roadmapGraph(t, env)
	edges, _ := graph["edges"].([]any)

	chain := timelineChain(edges)
	want := []string{backlogNodeID, idS1, idS2, idUA, idUZ}

	if len(chain) != len(want) {
		t.Fatalf("mixed chain length: want %d, got %d: %v", len(want), len(chain), chain)
	}
	for i, w := range want {
		if chain[i] != w {
			t.Errorf("chain[%d]: want %q, got %q", i, w, chain[i])
		}
	}
}

// ─── Milestone 2: Edge metadata — duration labels ─────────────────────────────

// TestRoadmapGraph_BacklogEdgeNoLabel verifies that the Backlog→first-scheduled
// timeline edge has no duration label (empty string).
func TestRoadmapGraph_BacklogEdgeNoLabel(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	d := createRelease(t, env, map[string]any{
		"name": "rg-bl-v1", "status": "planned", "start_date": "2026-01-01",
	})
	nodeID := releaseNodeID(releaseID(t, d))

	graph := roadmapGraph(t, env)
	edges, _ := graph["edges"].([]any)

	edge := findEdge(edges, backlogNodeID, nodeID, "timeline")
	if edge == nil {
		t.Fatalf("no timeline edge from Backlog to %q", nodeID)
	}
	label, _ := edge["label"].(string)
	if label != "" {
		t.Errorf("Backlog→first scheduled edge label: want empty, got %q", label)
	}
}

// TestRoadmapGraph_UnscheduledEdgesNoLabel verifies that timeline edges
// involving unscheduled releases carry no duration label.
func TestRoadmapGraph_UnscheduledEdgesNoLabel(t *testing.T) {
	t.Skip("documents the older chained-undated roadmap spec; superseded by the unscheduled-terminus pattern in TestGraphReleases_*. Resolve which spec is canonical and unskip.")
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// One scheduled, two unscheduled — edges for unscheduled must have empty labels.
	dSched := createRelease(t, env, map[string]any{
		"name": "rg-uenl-s1", "status": "planned", "start_date": "2026-01-01",
	})
	dUA := createRelease(t, env, map[string]any{"name": "rg-uenl-ua", "status": "planned"})
	dUB := createRelease(t, env, map[string]any{"name": "rg-uenl-ub", "status": "planned"})

	idSched := releaseNodeID(releaseID(t, dSched))
	idUA := releaseNodeID(releaseID(t, dUA))
	idUB := releaseNodeID(releaseID(t, dUB))

	graph := roadmapGraph(t, env)
	edges, _ := graph["edges"].([]any)

	// Edge from last scheduled → first unscheduled: no label.
	edgeSU := findEdge(edges, idSched, idUA, "timeline")
	if edgeSU == nil {
		t.Fatalf("no timeline edge %q→%q", idSched, idUA)
	}
	if label, _ := edgeSU["label"].(string); label != "" {
		t.Errorf("scheduled→unscheduled edge label: want empty, got %q", label)
	}

	// Edge between consecutive unscheduled releases: no label.
	edgeUU := findEdge(edges, idUA, idUB, "timeline")
	if edgeUU == nil {
		t.Fatalf("no timeline edge %q→%q", idUA, idUB)
	}
	if label, _ := edgeUU["label"].(string); label != "" {
		t.Errorf("unscheduled→unscheduled edge label: want empty, got %q", label)
	}
}

// TestRoadmapGraph_EdgeLabel1Day verifies that releases 1 day apart produce
// a timeline edge label of "1 day".
func TestRoadmapGraph_EdgeLabel1Day(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	d1 := createRelease(t, env, map[string]any{
		"name": "rg-el1d-v1", "status": "planned", "start_date": "2026-01-01",
	})
	d2 := createRelease(t, env, map[string]any{
		"name": "rg-el1d-v2", "status": "planned", "start_date": "2026-01-02",
	})
	id1 := releaseNodeID(releaseID(t, d1))
	id2 := releaseNodeID(releaseID(t, d2))

	graph := roadmapGraph(t, env)
	edges, _ := graph["edges"].([]any)

	edge := findEdge(edges, id1, id2, "timeline")
	if edge == nil {
		t.Fatalf("no timeline edge %q→%q", id1, id2)
	}
	label, _ := edge["label"].(string)
	if label != "1 day" {
		t.Errorf("1-day gap edge label: want %q, got %q", "1 day", label)
	}
}

// TestRoadmapGraph_EdgeLabel7Days verifies that releases 7 days apart produce
// a timeline edge label of "1 week".
func TestRoadmapGraph_EdgeLabel7Days(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	d1 := createRelease(t, env, map[string]any{
		"name": "rg-el7d-v1", "status": "planned", "start_date": "2026-01-01",
	})
	d2 := createRelease(t, env, map[string]any{
		"name": "rg-el7d-v2", "status": "planned", "start_date": "2026-01-08",
	})
	id1 := releaseNodeID(releaseID(t, d1))
	id2 := releaseNodeID(releaseID(t, d2))

	graph := roadmapGraph(t, env)
	edges, _ := graph["edges"].([]any)

	edge := findEdge(edges, id1, id2, "timeline")
	if edge == nil {
		t.Fatalf("no timeline edge %q→%q", id1, id2)
	}
	label, _ := edge["label"].(string)
	if label != "1 week" {
		t.Errorf("7-day gap edge label: want %q, got %q", "1 week", label)
	}
}

// TestRoadmapGraph_EdgeLabel14Days verifies that releases 14 days apart produce
// a timeline edge label of "2 weeks".
func TestRoadmapGraph_EdgeLabel14Days(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	d1 := createRelease(t, env, map[string]any{
		"name": "rg-el14d-v1", "status": "planned", "start_date": "2026-01-01",
	})
	d2 := createRelease(t, env, map[string]any{
		"name": "rg-el14d-v2", "status": "planned", "start_date": "2026-01-15",
	})
	id1 := releaseNodeID(releaseID(t, d1))
	id2 := releaseNodeID(releaseID(t, d2))

	graph := roadmapGraph(t, env)
	edges, _ := graph["edges"].([]any)

	edge := findEdge(edges, id1, id2, "timeline")
	if edge == nil {
		t.Fatalf("no timeline edge %q→%q", id1, id2)
	}
	label, _ := edge["label"].(string)
	if label != "2 weeks" {
		t.Errorf("14-day gap edge label: want %q, got %q", "2 weeks", label)
	}
}

// TestRoadmapGraph_EdgeLabel30Days verifies that releases 30 days apart produce
// a timeline edge label of "4 weeks" (30 days / 7 = 4 weeks, threshold < 5 weeks).
func TestRoadmapGraph_EdgeLabel30Days(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	d1 := createRelease(t, env, map[string]any{
		"name": "rg-el30d-v1", "status": "planned", "start_date": "2026-01-01",
	})
	d2 := createRelease(t, env, map[string]any{
		"name": "rg-el30d-v2", "status": "planned", "start_date": "2026-01-31",
	})
	id1 := releaseNodeID(releaseID(t, d1))
	id2 := releaseNodeID(releaseID(t, d2))

	graph := roadmapGraph(t, env)
	edges, _ := graph["edges"].([]any)

	edge := findEdge(edges, id1, id2, "timeline")
	if edge == nil {
		t.Fatalf("no timeline edge %q→%q", id1, id2)
	}
	label, _ := edge["label"].(string)
	if label != "4 weeks" {
		t.Errorf("30-day gap edge label: want %q, got %q", "4 weeks", label)
	}
}

// TestRoadmapGraph_EdgeLabelMonths verifies that a gap of 35+ days produces a
// months label, and a gap of 390+ days produces a years label.
func TestRoadmapGraph_EdgeLabelMonths(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// 35-day gap: weeks=5 (>=5), months=1 → "1 month".
	d1 := createRelease(t, env, map[string]any{
		"name": "rg-elm-v1", "status": "planned", "start_date": "2026-01-01",
	})
	// 2026-01-01 + 35 days = 2026-02-05
	d2 := createRelease(t, env, map[string]any{
		"name": "rg-elm-v2", "status": "planned", "start_date": "2026-02-05",
	})
	// 2026-02-05 + 390 days ≈ 2027-03-02
	d3 := createRelease(t, env, map[string]any{
		"name": "rg-elm-v3", "status": "planned", "start_date": "2027-03-02",
	})

	id1 := releaseNodeID(releaseID(t, d1))
	id2 := releaseNodeID(releaseID(t, d2))
	id3 := releaseNodeID(releaseID(t, d3))

	graph := roadmapGraph(t, env)
	edges, _ := graph["edges"].([]any)

	// 35-day gap → "1 month"
	edge12 := findEdge(edges, id1, id2, "timeline")
	if edge12 == nil {
		t.Fatalf("no timeline edge %q→%q", id1, id2)
	}
	label12, _ := edge12["label"].(string)
	if label12 != "1 month" {
		t.Errorf("35-day gap edge label: want %q, got %q", "1 month", label12)
	}

	// 390-day gap → "1 year"
	edge23 := findEdge(edges, id2, id3, "timeline")
	if edge23 == nil {
		t.Fatalf("no timeline edge %q→%q", id2, id3)
	}
	label23, _ := edge23["label"].(string)
	if label23 != "1 year" {
		t.Errorf("~390-day gap edge label: want %q, got %q", "1 year", label23)
	}
}

// ─── Milestone 3: Artifact assignment ─────────────────────────────────────────

// TestRoadmapGraph_ArtifactAssignedToRelease verifies that an artifact assigned
// to a release appears as a node with an "assigned" edge from the release node.
func TestRoadmapGraph_ArtifactAssignedToRelease(t *testing.T) {
	const artifactPath = "lifecycle/ideas/rg-assigned-idea.md"

	seeds := []seedArtifact{
		{
			relPath: artifactPath,
			content: makeArtifactWithRelease("RG Assigned Idea", "idea", "draft", "rg-assigned-idea", "rg-rel-assign", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	d := createRelease(t, env, map[string]any{"name": "rg-rel-assign", "status": "planned"})
	releaseNID := releaseNodeID(releaseID(t, d))

	graph := roadmapGraph(t, env)
	nodes, _ := graph["nodes"].([]any)
	edges, _ := graph["edges"].([]any)

	// Artifact node must be present.
	artifactNode := findNodeByID(nodes, artifactPath)
	if artifactNode == nil {
		t.Fatalf("artifact node %q not found in roadmap graph", artifactPath)
	}

	// Assigned edge from release node to artifact.
	edge := findEdge(edges, releaseNID, artifactPath, "assigned")
	if edge == nil {
		t.Errorf("no 'assigned' edge from %q to %q", releaseNID, artifactPath)
	}
}

// TestRoadmapGraph_ArtifactUnassignedFromBacklog verifies that an artifact with
// no release field has an "assigned" edge from the Backlog node.
func TestRoadmapGraph_ArtifactUnassignedFromBacklog(t *testing.T) {
	const artifactPath = "lifecycle/ideas/rg-unassigned-idea.md"

	seeds := []seedArtifact{
		{
			relPath: artifactPath,
			content: makeArtifact("RG Unassigned Idea", "idea", "draft", "rg-unassigned-idea", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	graph := roadmapGraph(t, env)
	nodes, _ := graph["nodes"].([]any)
	edges, _ := graph["edges"].([]any)

	// Artifact node must be present.
	artifactNode := findNodeByID(nodes, artifactPath)
	if artifactNode == nil {
		t.Fatalf("unassigned artifact node %q not found in roadmap graph", artifactPath)
	}

	// Assigned edge from Backlog to artifact.
	edge := findEdge(edges, backlogNodeID, artifactPath, "assigned")
	if edge == nil {
		t.Errorf("no 'assigned' edge from Backlog to unassigned artifact %q", artifactPath)
	}
}

// TestRoadmapGraph_ArtifactNodeFields verifies that artifact nodes include
// the required fields: id, title, type, and status.
func TestRoadmapGraph_ArtifactNodeFields(t *testing.T) {
	const artifactPath = "lifecycle/ideas/rg-fields-idea.md"

	seeds := []seedArtifact{
		{
			relPath: artifactPath,
			content: makeArtifactWithRelease("RG Fields Idea", "idea", "in-development", "rg-fields-idea", "rg-rel-fields", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{"name": "rg-rel-fields", "status": "planned"})

	graph := roadmapGraph(t, env)
	nodes, _ := graph["nodes"].([]any)

	node := findNodeByID(nodes, artifactPath)
	if node == nil {
		t.Fatalf("artifact node %q not found", artifactPath)
	}

	if id, _ := node["id"].(string); id != artifactPath {
		t.Errorf("artifact node id: want %q, got %q", artifactPath, id)
	}
	if title, _ := node["title"].(string); title != "RG Fields Idea" {
		t.Errorf("artifact node title: want %q, got %q", "RG Fields Idea", title)
	}
	if typ, _ := node["type"].(string); typ != "idea" {
		t.Errorf("artifact node type: want %q, got %q", "idea", typ)
	}
	if status, _ := node["status"].(string); status != "in-development" {
		t.Errorf("artifact node status: want %q, got %q", "in-development", status)
	}
}

// TestRoadmapGraph_PlansExcluded verifies that plan-type artifacts (plan-backend,
// plan-frontend, plan-test) are not included as nodes in the roadmap graph.
func TestRoadmapGraph_PlansExcluded(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rg-pe-idea.md",
			content: makeArtifactWithRelease("RG PE Idea", "idea", "draft", "rg-pe-idea", "rg-pe-rel", "Body."),
		},
		{
			relPath: "lifecycle/backend-plans/rg-pe-plan-2.md",
			content: makeArtifactWithRelease("RG PE Plan", "plan-backend", "draft", "rg-pe-plan", "rg-pe-rel", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{"name": "rg-pe-rel", "status": "planned"})

	graph := roadmapGraph(t, env)
	nodes, _ := graph["nodes"].([]any)

	for _, raw := range nodes {
		node, _ := raw.(map[string]any)
		typ, _ := node["type"].(string)
		id, _ := node["id"].(string)
		switch typ {
		case "plan-backend", "plan-frontend", "plan-test", "plan-dev":
			t.Errorf("plan artifact %q (type %q) must not appear in roadmap graph", id, typ)
		}
	}
}

// TestRoadmapGraph_DependsOnEdgesPreserved verifies that existing depends_on
// relationships between artifacts in the graph are included as edges.
func TestRoadmapGraph_DependsOnEdgesPreserved(t *testing.T) {
	const pathA = "lifecycle/ideas/rg-dep-a.md"
	const pathB = "lifecycle/ideas/rg-dep-b.md"

	seeds := []seedArtifact{
		{
			relPath: pathB,
			content: makeArtifactWithRelease("RG Dep B", "idea", "draft", "rg-dep-b", "rg-dep-rel", "Body."),
		},
		{
			// A depends_on B.
			relPath: pathA,
			content: makeArtifactWithDependsOn("RG Dep A", "idea", "draft", "rg-dep-a", "rg-dep-rel", pathB),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{"name": "rg-dep-rel", "status": "planned"})

	graph := roadmapGraph(t, env)
	edges, _ := graph["edges"].([]any)

	// There should be a depends_on edge from A to B.
	depEdge := findEdge(edges, pathA, pathB, "depends_on")
	if depEdge == nil {
		t.Errorf("no depends_on edge from %q to %q in roadmap graph", pathA, pathB)
	}
}

// ─── Milestone 7: Edge cases and regression ────────────────────────────────────

// TestRoadmapGraph_DeleteOnlyScheduledUpdatesChain verifies that after deleting
// the only scheduled release, the chain becomes Backlog → unscheduled.
func TestRoadmapGraph_DeleteOnlyScheduledUpdatesChain(t *testing.T) {
	t.Skip("documents the older chained-undated roadmap spec; superseded by the unscheduled-terminus pattern in TestGraphReleases_*. Resolve which spec is canonical and unskip.")
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	dSched := createRelease(t, env, map[string]any{
		"name": "rg-dos-sched", "status": "planned", "start_date": "2026-01-01",
	})
	schedID := releaseID(t, dSched)

	dUnsched := createRelease(t, env, map[string]any{"name": "rg-dos-unsched", "status": "planned"})
	unschedNodeID := releaseNodeID(releaseID(t, dUnsched))

	// Delete the only scheduled release.
	resp := env.doRequest("DELETE", releasePath(schedID), nil)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// After deletion, chain must be: Backlog → unscheduled.
	graph := roadmapGraph(t, env)
	edges, _ := graph["edges"].([]any)

	chain := timelineChain(edges)
	want := []string{backlogNodeID, unschedNodeID}

	if len(chain) != len(want) {
		t.Fatalf("post-delete chain length: want %d, got %d: %v", len(want), len(chain), chain)
	}
	if chain[1] != unschedNodeID {
		t.Errorf("post-delete chain[1]: want %q, got %q", unschedNodeID, chain[1])
	}
}

// TestRoadmapGraph_InsertReleaseInMiddleChain verifies that adding a release
// with a start_date between two existing releases inserts it at the correct
// position on the next graph fetch.
func TestRoadmapGraph_InsertReleaseInMiddleChain(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// R1 (Jan) and R3 (Jul) already exist.
	d1 := createRelease(t, env, map[string]any{
		"name": "rg-ins-v1", "status": "planned", "start_date": "2026-01-01",
	})
	d3 := createRelease(t, env, map[string]any{
		"name": "rg-ins-v3", "status": "planned", "start_date": "2026-07-01",
	})

	// Insert R2 (Apr) — should slot between R1 and R3.
	d2 := createRelease(t, env, map[string]any{
		"name": "rg-ins-v2", "status": "planned", "start_date": "2026-04-01",
	})

	id1 := releaseNodeID(releaseID(t, d1))
	id2 := releaseNodeID(releaseID(t, d2))
	id3 := releaseNodeID(releaseID(t, d3))

	graph := roadmapGraph(t, env)
	edges, _ := graph["edges"].([]any)

	chain := timelineChain(edges)
	want := []string{backlogNodeID, id1, id2, id3}

	if len(chain) != len(want) {
		t.Fatalf("insert-middle chain length: want %d, got %d: %v", len(want), len(chain), chain)
	}
	for i, w := range want {
		if chain[i] != w {
			t.Errorf("chain[%d]: want %q, got %q", i, w, chain[i])
		}
	}
}

// TestRoadmapGraph_RenameUpdatesNodeLabel verifies that renaming a release
// updates the node title on the next graph fetch.
func TestRoadmapGraph_RenameUpdatesNodeLabel(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	d := createRelease(t, env, map[string]any{
		"name": "rg-ren-old", "status": "planned", "start_date": "2026-01-01",
	})
	id := releaseID(t, d)
	nodeID := releaseNodeID(id)

	// Rename the release.
	resp := env.doRequest("PUT", releasePath(id), map[string]any{
		"name": "rg-ren-new", "status": "planned", "start_date": "2026-01-01",
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Fetch graph and check updated title.
	graph := roadmapGraph(t, env)
	nodes, _ := graph["nodes"].([]any)

	node := findNodeByID(nodes, nodeID)
	if node == nil {
		t.Fatalf("release node %q not found after rename", nodeID)
	}
	title, _ := node["title"].(string)
	if title != "rg-ren-new" {
		t.Errorf("release node title after rename: want %q, got %q", "rg-ren-new", title)
	}
}

// TestRoadmapGraph_MainGraphUnaffected verifies that the main artifact graph
// endpoint (GET /graph) is unaffected by the presence of releases and the
// roadmap graph endpoint.
func TestRoadmapGraph_MainGraphUnaffected(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rg-main-idea.md",
			content: makeArtifactWithRelease("RG Main Idea", "idea", "draft", "rg-main-idea", "rg-main-rel", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{
		"name": "rg-main-rel", "status": "planned", "start_date": "2026-01-01",
	})

	// GET /graph must work and return the idea node.
	resp := env.doRequest("GET", "/api/p/testproject/graph", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	nodes := decodeGraphNodes(t, data)
	found := false
	for _, raw := range nodes {
		node, _ := raw.(map[string]any)
		if id, _ := node["id"].(string); id == "lifecycle/ideas/rg-main-idea.md" {
			found = true
			break
		}
	}
	if !found {
		t.Error("main graph: artifact node not found; roadmap graph changes may have broken /graph")
	}
}

// TestRoadmapGraph_TimelineEdgeCount verifies that the number of timeline edges
// equals the total number of releases (Backlog + scheduled + unscheduled − 1 = n releases).
func TestRoadmapGraph_TimelineEdgeCount(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{
		"name": "rg-tec-s1", "status": "planned", "start_date": "2026-01-01",
	})
	createRelease(t, env, map[string]any{
		"name": "rg-tec-s2", "status": "planned", "start_date": "2026-04-01",
	})
	createRelease(t, env, map[string]any{"name": "rg-tec-u1", "status": "planned"})

	graph := roadmapGraph(t, env)
	edges, _ := graph["edges"].([]any)

	// 3 releases → 3 timeline edges (Backlog→s1, s1→s2, s2→u1).
	timelineCount := countEdgesByKind(edges, "timeline")
	if timelineCount != 3 {
		t.Errorf("timeline edge count: want 3, got %d", timelineCount)
	}
}
