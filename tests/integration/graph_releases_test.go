//go:build integration

package integration

import (
	"net/http"
	"testing"
)

// ── Graph releases test helpers ───────────────────────────────────────────────

// graphWithReleases calls GET /graph?include_releases=true (no auth required)
// and returns decoded nodes and edges.
func graphWithReleases(t *testing.T, env *testEnv) (nodes []any, edges []any) {
	t.Helper()
	resp, err := http.Get(env.baseURL + "/api/p/testproject/graph?include_releases=true")
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	nodes, _ = data["nodes"].([]any)
	edges, _ = data["edges"].([]any)
	return
}

// nodesOfType returns all nodes whose "type" field equals typ.
func nodesOfType(nodes []any, typ string) []map[string]any {
	var result []map[string]any
	for _, raw := range nodes {
		node, _ := raw.(map[string]any)
		if t, _ := node["type"].(string); t == typ {
			result = append(result, node)
		}
	}
	return result
}

// occurrencesOfNode counts how many nodes have the given id.
func occurrencesOfNode(nodes []any, id string) int {
	n := 0
	for _, raw := range nodes {
		node, _ := raw.(map[string]any)
		if nodeID, _ := node["id"].(string); nodeID == id {
			n++
		}
	}
	return n
}

// ── Milestone 1: include_releases parameter ───────────────────────────────────

// TestGraphReleases_BaselineNoParam verifies that GET /graph without
// include_releases returns no nodes with type "release" and no timeline or
// assigned edges.
func TestGraphReleases_BaselineNoParam(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/gr-base-idea.md",
			content: makeArtifact("Base Idea", "idea", "draft", "gr-base-idea", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")
	createRelease(t, env, map[string]any{
		"name": "gr-base-v1", "status": "planned", "start_date": "2026-01-01",
	})

	// Use the plain /graph endpoint (no include_releases param).
	data := graphResponseForProject(t, env)
	nodes := decodeGraphNodes(t, data)
	edges, _ := data["edges"].([]any)

	releaseNodes := nodesOfType(nodes, "release")
	if len(releaseNodes) != 0 {
		t.Errorf("baseline /graph: want 0 release nodes, got %d", len(releaseNodes))
	}

	for _, raw := range edges {
		edge, _ := raw.(map[string]any)
		kind, _ := edge["kind"].(string)
		if kind == "timeline" || kind == "assigned" {
			t.Errorf("baseline /graph: unexpected edge with kind %q (should only appear with include_releases=true)", kind)
		}
	}
}

// TestGraphReleases_WithParam verifies that GET /graph?include_releases=true
// includes release nodes and timeline edges in the response.
func TestGraphReleases_WithParam(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/gr-wp-idea.md",
			content: makeArtifact("WP Idea", "idea", "draft", "gr-wp-idea", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")
	createRelease(t, env, map[string]any{
		"name": "gr-wp-v1", "status": "planned", "start_date": "2026-01-01",
	})

	nodes, edges := graphWithReleases(t, env)

	releaseNodes := nodesOfType(nodes, "release")
	if len(releaseNodes) == 0 {
		t.Error("GET /graph?include_releases=true: want at least one release node, got 0")
	}

	// Backlog synthetic node must be present.
	if findNodeByID(nodes, backlogNodeID) == nil {
		t.Errorf("GET /graph?include_releases=true: Backlog node %q not found", backlogNodeID)
	}

	// At least one timeline edge must be present (Backlog → v1).
	timelineCount := countEdgesByKind(edges, "timeline")
	if timelineCount == 0 {
		t.Error("GET /graph?include_releases=true: want at least one timeline edge, got 0")
	}
}

// TestGraphReleases_NoDuplicateNodes verifies that an artifact that appears in
// both the standard artifact graph and the release overlay is not duplicated.
func TestGraphReleases_NoDuplicateNodes(t *testing.T) {
	const artifactPath = "lifecycle/ideas/gr-dup-idea.md"
	seeds := []seedArtifact{
		{
			relPath: artifactPath,
			content: makeArtifactWithRelease("Dup Idea", "idea", "draft", "gr-dup-idea", "gr-dup-v1", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")
	createRelease(t, env, map[string]any{
		"name": "gr-dup-v1", "status": "planned", "start_date": "2026-01-01",
	})

	nodes, _ := graphWithReleases(t, env)

	// The artifact must appear exactly once despite appearing in both graphs.
	count := occurrencesOfNode(nodes, artifactPath)
	if count != 1 {
		t.Errorf("node %q: want exactly 1 occurrence, got %d", artifactPath, count)
	}
}

// TestGraphReleases_FilterIndependence verifies that with include_releases=true
// and type=idea, only idea-type artifact nodes plus all release nodes are
// returned — release nodes are not filtered out by the type parameter.
func TestGraphReleases_FilterIndependence(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/gr-fi-idea.md",
			content: makeArtifact("FI Idea", "idea", "draft", "gr-fi-idea", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/gr-fi-ticket.md",
			content: makeArtifact("FI Ticket", "ticket", "draft", "gr-fi-ticket", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")
	createRelease(t, env, map[string]any{
		"name": "gr-fi-v1", "status": "planned", "start_date": "2026-01-01",
	})

	resp, err := http.Get(env.baseURL + "/api/p/testproject/graph?include_releases=true&type=idea")
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	nodes := decodeGraphNodes(t, data)

	// Every non-release node must have type "idea".
	for _, raw := range nodes {
		node, _ := raw.(map[string]any)
		typ, _ := node["type"].(string)
		id, _ := node["id"].(string)
		if typ != "idea" && typ != "release" {
			t.Errorf("unexpected node %q with type %q (want only 'idea' or 'release' with type=idea filter)", id, typ)
		}
	}

	// Release nodes must not be filtered out.
	releaseNodes := nodesOfType(nodes, "release")
	if len(releaseNodes) == 0 {
		t.Error("type=idea filter with include_releases=true: release nodes should not be filtered out")
	}
}

// TestGraphReleases_EmptyReleases verifies that with include_releases=true but
// no releases in the project, the response contains only the Backlog synthetic
// node and no timeline edges.
func TestGraphReleases_EmptyReleases(t *testing.T) {
	// No seeds, no releases.
	env := newTestEnv(t, nil)

	nodes, edges := graphWithReleases(t, env)

	releaseNodes := nodesOfType(nodes, "release")
	if len(releaseNodes) != 1 {
		t.Errorf("empty releases: want 1 release node (Backlog), got %d", len(releaseNodes))
	}
	if len(releaseNodes) == 1 {
		id, _ := releaseNodes[0]["id"].(string)
		if id != backlogNodeID {
			t.Errorf("empty releases: sole release node id: want %q, got %q", backlogNodeID, id)
		}
	}

	timelineCount := countEdgesByKind(edges, "timeline")
	if timelineCount != 0 {
		t.Errorf("empty releases: want 0 timeline edges, got %d", timelineCount)
	}
}

// ── Milestone 2: Backlog node semantics ──────────────────────────────────────

// TestGraphReleases_BacklogPresent verifies that when unassigned ideas/defects
// exist, the roadmap graph contains a Backlog node with the expected fields.
func TestGraphReleases_BacklogPresent(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/gr-bp-idea.md",
			content: makeArtifact("BP Idea", "idea", "draft", "gr-bp-idea", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	data := roadmapGraph(t, env)
	nodes, _ := data["nodes"].([]any)

	backlog := findNodeByID(nodes, backlogNodeID)
	if backlog == nil {
		t.Fatalf("Backlog node %q not found in roadmap graph", backlogNodeID)
	}
	if title, _ := backlog["title"].(string); title != "Backlog" {
		t.Errorf("Backlog title: want %q, got %q", "Backlog", title)
	}
	if typ, _ := backlog["type"].(string); typ != "release" {
		t.Errorf("Backlog type: want %q, got %q", "release", typ)
	}
}

// TestGraphReleases_BacklogEdges verifies that each unassigned idea and defect
// has an "assigned" edge from the Backlog node.
func TestGraphReleases_BacklogEdges(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/gr-be-idea1.md",
			content: makeArtifact("BE Idea 1", "idea", "draft", "gr-be-idea1", "", "Body."),
		},
		{
			relPath: "lifecycle/defects/gr-be-defect1.md",
			content: makeArtifact("BE Defect 1", "defect", "draft", "gr-be-defect1", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	data := roadmapGraph(t, env)
	edges, _ := data["edges"].([]any)

	if findEdge(edges, backlogNodeID, "lifecycle/ideas/gr-be-idea1.md", "assigned") == nil {
		t.Error("missing assigned edge: Backlog → lifecycle/ideas/gr-be-idea1.md")
	}
	if findEdge(edges, backlogNodeID, "lifecycle/defects/gr-be-defect1.md", "assigned") == nil {
		t.Error("missing assigned edge: Backlog → lifecycle/defects/gr-be-defect1.md")
	}
}

// TestGraphReleases_BacklogTimelinePosition verifies that a timeline edge
// connects release:backlog directly to the earliest dated release.
func TestGraphReleases_BacklogTimelinePosition(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	rEarly := createRelease(t, env, map[string]any{
		"name": "gr-bt-early", "status": "planned", "start_date": "2026-03-01",
	})
	rLater := createRelease(t, env, map[string]any{
		"name": "gr-bt-later", "status": "planned", "start_date": "2026-06-01",
	})
	idEarly := releaseNodeID(releaseID(t, rEarly))
	idLater := releaseNodeID(releaseID(t, rLater))

	data := roadmapGraph(t, env)
	edges, _ := data["edges"].([]any)

	// Backlog must connect directly to the earliest release.
	if findEdge(edges, backlogNodeID, idEarly, "timeline") == nil {
		t.Errorf("expected timeline edge: Backlog → earliest release %q", idEarly)
	}
	// Backlog must NOT connect directly to the later release.
	if findEdge(edges, backlogNodeID, idLater, "timeline") != nil {
		t.Errorf("unexpected direct timeline edge: Backlog → later release %q", idLater)
	}
}

// TestGraphReleases_AllAssigned verifies that when every idea/defect has a
// release, the Backlog node is still present but has no assigned edges.
func TestGraphReleases_AllAssigned(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/gr-aa-idea1.md",
			content: makeArtifactWithRelease("AA Idea 1", "idea", "draft", "gr-aa-idea1", "gr-aa-v1", "Body."),
		},
		{
			relPath: "lifecycle/ideas/gr-aa-idea2.md",
			content: makeArtifactWithRelease("AA Idea 2", "idea", "draft", "gr-aa-idea2", "gr-aa-v1", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")
	createRelease(t, env, map[string]any{
		"name": "gr-aa-v1", "status": "planned", "start_date": "2026-01-01",
	})

	data := roadmapGraph(t, env)
	nodes, _ := data["nodes"].([]any)
	edges, _ := data["edges"].([]any)

	// Backlog node must still be present.
	if findNodeByID(nodes, backlogNodeID) == nil {
		t.Error("Backlog node missing when all artifacts are assigned to a release")
	}

	// No assigned edges should originate from the Backlog.
	for _, raw := range edges {
		edge, _ := raw.(map[string]any)
		src, _ := edge["source"].(string)
		kind, _ := edge["kind"].(string)
		if src == backlogNodeID && kind == "assigned" {
			tgt, _ := edge["target"].(string)
			t.Errorf("unexpected assigned edge from Backlog → %q when all artifacts are assigned", tgt)
		}
	}
}

// TestGraphReleases_NoArtifactsBacklog verifies that when no artifacts exist at
// all, the Backlog node is present and there are no edges.
func TestGraphReleases_NoArtifactsBacklog(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	data := roadmapGraph(t, env)
	nodes, _ := data["nodes"].([]any)
	edges, _ := data["edges"].([]any)

	if findNodeByID(nodes, backlogNodeID) == nil {
		t.Error("Backlog node missing when no artifacts exist")
	}

	if len(edges) != 0 {
		t.Errorf("expected 0 edges when no releases and no artifacts, got %d", len(edges))
	}
}

// ── Milestone 3: Unscheduled node semantics ───────────────────────────────────

// TestGraphReleases_UnscheduledPresent verifies that when at least one release
// has no start_date, a synthetic "release:unscheduled" node exists.
func TestGraphReleases_UnscheduledPresent(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")
	createRelease(t, env, map[string]any{"name": "gr-up-v1", "status": "planned"}) // no start_date

	data := roadmapGraph(t, env)
	nodes, _ := data["nodes"].([]any)

	const unschedID = "release:unscheduled"
	unschedNode := findNodeByID(nodes, unschedID)
	if unschedNode == nil {
		t.Fatalf("Unscheduled node %q not found when an undated release exists", unschedID)
	}
	if title, _ := unschedNode["title"].(string); title != "Unscheduled" {
		t.Errorf("Unscheduled node title: want %q, got %q", "Unscheduled", title)
	}
	if typ, _ := unschedNode["type"].(string); typ != "release" {
		t.Errorf("Unscheduled node type: want %q, got %q", "release", typ)
	}
}

// TestGraphReleases_UnscheduledAbsent verifies that when all releases have a
// start_date, no "release:unscheduled" node exists.
func TestGraphReleases_UnscheduledAbsent(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")
	createRelease(t, env, map[string]any{
		"name": "gr-ua-v1", "status": "planned", "start_date": "2026-01-01",
	})

	data := roadmapGraph(t, env)
	nodes, _ := data["nodes"].([]any)

	const unschedID = "release:unscheduled"
	if findNodeByID(nodes, unschedID) != nil {
		t.Errorf("Unscheduled node %q should not exist when all releases have start_date", unschedID)
	}
}

// TestGraphReleases_UnscheduledTerminus verifies that "release:unscheduled" has
// at least one incoming timeline edge but no outgoing timeline edges — it is a
// terminal node in the timeline chain.
func TestGraphReleases_UnscheduledTerminus(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")
	createRelease(t, env, map[string]any{
		"name": "gr-ut-dated", "status": "planned", "start_date": "2026-01-01",
	})
	createRelease(t, env, map[string]any{"name": "gr-ut-unsched", "status": "planned"})

	data := roadmapGraph(t, env)
	edges, _ := data["edges"].([]any)

	const unschedID = "release:unscheduled"
	incomingTimeline := 0
	for _, raw := range edges {
		edge, _ := raw.(map[string]any)
		kind, _ := edge["kind"].(string)
		if kind != "timeline" {
			continue
		}
		src, _ := edge["source"].(string)
		tgt, _ := edge["target"].(string)
		if tgt == unschedID {
			incomingTimeline++
		}
		if src == unschedID {
			t.Errorf("%q should have no outgoing timeline edges, but found edge → %q", unschedID, tgt)
		}
	}
	if incomingTimeline == 0 {
		t.Errorf("%q has no incoming timeline edges; expected at least one", unschedID)
	}
}

// TestGraphReleases_UndatedReleaseNodes verifies that individual undated
// releases appear as separate nodes with their own IDs and titles, and each
// has a timeline edge targeting "release:unscheduled".
func TestGraphReleases_UndatedReleaseNodes(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	rAlpha := createRelease(t, env, map[string]any{"name": "gr-urn-alpha", "status": "planned"})
	rBeta := createRelease(t, env, map[string]any{"name": "gr-urn-beta", "status": "planned"})
	nidAlpha := releaseNodeID(releaseID(t, rAlpha))
	nidBeta := releaseNodeID(releaseID(t, rBeta))

	const unschedID = "release:unscheduled"

	data := roadmapGraph(t, env)
	nodes, _ := data["nodes"].([]any)
	edges, _ := data["edges"].([]any)

	// Both undated release nodes must appear as distinct nodes.
	if findNodeByID(nodes, nidAlpha) == nil {
		t.Errorf("undated release node %q not found", nidAlpha)
	}
	if findNodeByID(nodes, nidBeta) == nil {
		t.Errorf("undated release node %q not found", nidBeta)
	}

	// Verify titles match release names.
	if n := findNodeByID(nodes, nidAlpha); n != nil {
		if title, _ := n["title"].(string); title != "gr-urn-alpha" {
			t.Errorf("undated release %q title: want %q, got %q", nidAlpha, "gr-urn-alpha", title)
		}
	}

	// Each undated release must have a timeline edge TO the unscheduled terminus.
	if findEdge(edges, nidAlpha, unschedID, "timeline") == nil {
		t.Errorf("missing timeline edge: %s → %s", nidAlpha, unschedID)
	}
	if findEdge(edges, nidBeta, unschedID, "timeline") == nil {
		t.Errorf("missing timeline edge: %s → %s", nidBeta, unschedID)
	}
}

// ── Milestone 4: Timeline ordering ───────────────────────────────────────────

// TestGraphReleases_ChronologicalOrder verifies that three releases created in
// non-chronological order are chained by start_date ascending:
// Backlog → Jan → Feb → Mar.
func TestGraphReleases_ChronologicalOrder(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Create in non-chronological order.
	rMar := createRelease(t, env, map[string]any{
		"name": "gr-co-mar", "status": "planned", "start_date": "2026-03-01",
	})
	rJan := createRelease(t, env, map[string]any{
		"name": "gr-co-jan", "status": "planned", "start_date": "2026-01-01",
	})
	rFeb := createRelease(t, env, map[string]any{
		"name": "gr-co-feb", "status": "planned", "start_date": "2026-02-01",
	})
	nidMar := releaseNodeID(releaseID(t, rMar))
	nidJan := releaseNodeID(releaseID(t, rJan))
	nidFeb := releaseNodeID(releaseID(t, rFeb))

	data := roadmapGraph(t, env)
	edges, _ := data["edges"].([]any)
	chain := timelineChain(edges)

	want := []string{backlogNodeID, nidJan, nidFeb, nidMar}
	if len(chain) != len(want) {
		t.Fatalf("chronological chain length: want %d, got %d: %v", len(want), len(chain), chain)
	}
	for i, w := range want {
		if chain[i] != w {
			t.Errorf("chain[%d]: want %q, got %q", i, w, chain[i])
		}
	}
}

// TestGraphReleases_SameDateStability verifies that releases with identical
// start_date are sorted alphabetically by name (secondary sort).
func TestGraphReleases_SameDateStability(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Create in reverse alphabetical order; expect alphabetical in the chain.
	rZeta := createRelease(t, env, map[string]any{
		"name": "gr-sd-zeta", "status": "planned", "start_date": "2026-04-01",
	})
	rAlpha := createRelease(t, env, map[string]any{
		"name": "gr-sd-alpha", "status": "planned", "start_date": "2026-04-01",
	})
	nidZeta := releaseNodeID(releaseID(t, rZeta))
	nidAlpha := releaseNodeID(releaseID(t, rAlpha))

	data := roadmapGraph(t, env)
	edges, _ := data["edges"].([]any)
	chain := timelineChain(edges)

	// Alphabetically: gr-sd-alpha before gr-sd-zeta.
	want := []string{backlogNodeID, nidAlpha, nidZeta}
	if len(chain) != len(want) {
		t.Fatalf("same-date chain length: want %d, got %d: %v", len(want), len(chain), chain)
	}
	for i, w := range want {
		if chain[i] != w {
			t.Errorf("same-date chain[%d]: want %q, got %q", i, w, chain[i])
		}
	}
}

// TestGraphReleases_SingleReleaseNoUnscheduled verifies that a single dated
// release produces exactly Backlog → Release with no Unscheduled node.
func TestGraphReleases_SingleReleaseNoUnscheduled(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	r := createRelease(t, env, map[string]any{
		"name": "gr-sr-v1", "status": "planned", "start_date": "2026-01-01",
	})
	nid := releaseNodeID(releaseID(t, r))

	data := roadmapGraph(t, env)
	nodes, _ := data["nodes"].([]any)
	edges, _ := data["edges"].([]any)
	chain := timelineChain(edges)

	// Chain: Backlog → release.
	want := []string{backlogNodeID, nid}
	if len(chain) != len(want) {
		t.Fatalf("single-release chain length: want %d, got %d: %v", len(want), len(chain), chain)
	}
	if chain[1] != nid {
		t.Errorf("chain[1]: want %q, got %q", nid, chain[1])
	}

	// No Unscheduled node.
	if findNodeByID(nodes, "release:unscheduled") != nil {
		t.Error("single dated release: Unscheduled node should not exist")
	}
}

// TestGraphReleases_IncludeReleasesEdgeCountGrowth verifies that switching from
// GET /graph to GET /graph?include_releases=true results in more nodes and edges
// when releases exist, confirming the overlay is actually merged.
func TestGraphReleases_IncludeReleasesEdgeCountGrowth(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/gr-ecg-idea.md",
			content: makeArtifact("ECG Idea", "idea", "draft", "gr-ecg-idea", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")
	createRelease(t, env, map[string]any{
		"name": "gr-ecg-v1", "status": "planned", "start_date": "2026-01-01",
	})

	// Count nodes/edges without overlay.
	baseData := graphResponseForProject(t, env)
	baseNodes := decodeGraphNodes(t, baseData)
	baseEdges, _ := baseData["edges"].([]any)

	// Count with overlay.
	overlayNodes, overlayEdges := graphWithReleases(t, env)

	// Overlay must have more nodes (added release nodes).
	if len(overlayNodes) <= len(baseNodes) {
		t.Errorf("include_releases=true: want more nodes than baseline (%d), got %d", len(baseNodes), len(overlayNodes))
	}

	// Overlay must have more edges (added timeline/assigned edges).
	if len(overlayEdges) <= len(baseEdges) {
		t.Errorf("include_releases=true: want more edges than baseline (%d), got %d", len(baseEdges), len(overlayEdges))
	}
}

// TestGraphReleases_ReleaseNodesNotFilteredByType confirms that a release node
// has type "release" in the overlay, matching the spec vocabulary.
func TestGraphReleases_ReleaseNodeType(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")
	r := createRelease(t, env, map[string]any{
		"name": "gr-rnt-v1", "status": "planned", "start_date": "2026-01-01",
	})
	nid := releaseNodeID(releaseID(t, r))

	nodes, _ := graphWithReleases(t, env)

	node := findNodeByID(nodes, nid)
	if node == nil {
		t.Fatalf("release node %q not found in /graph?include_releases=true response", nid)
	}
	if typ, _ := node["type"].(string); typ != "release" {
		t.Errorf("release node %q type: want %q, got %q", nid, "release", typ)
	}

	// The synthetic Backlog node must also carry type "release".
	backlog := findNodeByID(nodes, backlogNodeID)
	if backlog == nil {
		t.Fatal("Backlog node not found")
	}
	if typ, _ := backlog["type"].(string); typ != "release" {
		t.Errorf("Backlog node type: want %q, got %q", "release", typ)
	}
}

// TestGraphReleases_OverlayAssignedEdge verifies that when include_releases=true,
// an artifact assigned to a release has an "assigned" edge from the release node.
func TestGraphReleases_OverlayAssignedEdge(t *testing.T) {
	const artifactPath = "lifecycle/ideas/gr-oae-idea.md"
	seeds := []seedArtifact{
		{
			relPath: artifactPath,
			content: makeArtifactWithRelease("OAE Idea", "idea", "draft", "gr-oae-idea", "gr-oae-v1", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")
	r := createRelease(t, env, map[string]any{
		"name": "gr-oae-v1", "status": "planned", "start_date": "2026-01-01",
	})
	nid := releaseNodeID(releaseID(t, r))

	_, edges := graphWithReleases(t, env)

	// Assigned edge: release node → artifact.
	if findEdge(edges, nid, artifactPath, "assigned") == nil {
		t.Errorf("missing assigned edge: %s → %s in /graph?include_releases=true", nid, artifactPath)
	}
}

// TestGraphReleases_BacklogAssignedEdgeInOverlay verifies that unassigned
// ideas have an "assigned" edge from Backlog in the overlay response.
func TestGraphReleases_BacklogAssignedEdgeInOverlay(t *testing.T) {
	const artifactPath = "lifecycle/ideas/gr-bae-idea.md"
	seeds := []seedArtifact{
		{
			relPath: artifactPath,
			content: makeArtifact("BAE Idea", "idea", "draft", "gr-bae-idea", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	_, edges := graphWithReleases(t, env)

	if findEdge(edges, backlogNodeID, artifactPath, "assigned") == nil {
		t.Errorf("missing assigned edge: Backlog → %s in /graph?include_releases=true", artifactPath)
	}
}

// TestGraphReleases_MultipleReleasesChainInOverlay verifies that the timeline
// chain is correctly reflected in the /graph?include_releases=true response.
func TestGraphReleases_MultipleReleasesChainInOverlay(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	r1 := createRelease(t, env, map[string]any{
		"name": "gr-mrc-v1", "status": "planned", "start_date": "2026-01-01",
	})
	r2 := createRelease(t, env, map[string]any{
		"name": "gr-mrc-v2", "status": "planned", "start_date": "2026-04-01",
	})
	nid1 := releaseNodeID(releaseID(t, r1))
	nid2 := releaseNodeID(releaseID(t, r2))

	_, edges := graphWithReleases(t, env)

	// Backlog → v1 → v2.
	if findEdge(edges, backlogNodeID, nid1, "timeline") == nil {
		t.Errorf("missing timeline edge: Backlog → %s", nid1)
	}
	if findEdge(edges, nid1, nid2, "timeline") == nil {
		t.Errorf("missing timeline edge: %s → %s", nid1, nid2)
	}
}

