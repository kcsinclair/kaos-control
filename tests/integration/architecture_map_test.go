// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Test plan: lifecycle/test-plans/architecture-relationship-map-5-test.md
//
// Milestone 3 (FR-3, FR-8, FR-10): the /architecture-map endpoint's HTTP
// contract through a live server — response shape, stack_for handling, auth
// parity with GET /graph, and the read-only guarantee (no mutating verb).
//
// Milestone 4 (FR-12): freshness — the map reflects on-disk catalog changes
// via the real fsnotify watcher, with no process restart, and the main
// artifact/lineage graph (GET /graph) is untouched by this feature.
//
// internal/http/graph_test.go already covers the handler in isolation via
// httptest + a directly-injected project/user context (bypassing real
// session/CSRF auth and the live watcher). These tests exercise the same
// contract end to end through newTestEnv's running server, which is what the
// plan's Milestone 3/4 files call for.
//
// Fixtures under lifecycle/architecture/ are written AFTER newTestEnv starts
// and awaited via the live watcher rather than passed as newTestEnv seeds:
// lifecycle/architecture/ is not a configured stage (see
// lifecycle/tests/architectural-artefacts-6-test.md, gap #1), so pre-boot
// seeds placed there are silently skipped by the startup scan — confirmed
// while writing this file (indexed=0 despite seeded architecture fixtures).
// The live fsnotify watcher does cover the whole lifecycle/ tree, so writing
// post-boot and polling is the reliable way to get architecture fixtures
// indexed, and happens to be exactly what Milestone 4 needs to test anyway.

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// makeArchFixture builds a minimal type: architecture fixture.
func makeArchFixture(title, body string) string {
	return makeCleanSlugArtifact(title, "architecture", "approved", body)
}

// writeArchFile writes a fixture to disk under the project root and waits
// for the live watcher to index it (see the package-level doc comment for
// why pre-boot seeds cannot be used for lifecycle/architecture/ fixtures).
func writeArchFile(t *testing.T, env *testEnv, relPath, content string) {
	t.Helper()
	absPath := filepath.Join(env.projectRoot, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	awaitIndexed(t, env, relPath)
}

// awaitIndexed polls GET /artifacts until relPath is indexed (live watcher).
func awaitIndexed(t *testing.T, env *testEnv, relPath string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp := env.doRequest("GET", "/api/p/testproject/artifacts?limit=0", nil)
		if resp.StatusCode == http.StatusOK {
			data := readJSON(t, resp)
			if findArtifactRow(t, data, relPath) != nil {
				return
			}
		} else {
			resp.Body.Close()
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatalf("expected %q to be indexed by the live watcher within 2s", relPath)
}

// TestArchitectureMap_BaseMapShape verifies FR-3: GET returns 200 with a
// {nodes, edges} payload; nodes carry labels, and a typed relationship field
// produces an edge with both kind and label set.
func TestArchitectureMap_BaseMapShape(t *testing.T) {
	fooPath := "lifecycle/architecture/architectures/am-foo.md"
	barPath := "lifecycle/architecture/architectures/am-bar.md"

	env := newTestEnv(t, nil)
	writeArchFile(t, env, barPath, makeArchFixture("AM Bar", "Body."))
	writeArchFile(t, env, fooPath, "---\ntitle: AM Foo\ntype: architecture\nstatus: approved\n"+
		"evolves_into:\n  - architecture/architectures/am-bar.md\n---\n\nBody.\n")

	resp := env.doRequest("GET", "/api/p/testproject/architecture-map", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	nodes, ok := data["nodes"].([]any)
	if !ok || len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got: %v", data["nodes"])
	}
	for _, raw := range nodes {
		node, _ := raw.(map[string]any)
		if _, ok := node["labels"]; !ok {
			t.Errorf("node %v missing labels field", node["id"])
		}
	}

	edges, ok := data["edges"].([]any)
	if !ok || len(edges) != 1 {
		t.Fatalf("expected 1 edge, got: %v", data["edges"])
	}
	edge, _ := edges[0].(map[string]any)
	if edge["kind"] != "evolves_into" {
		t.Errorf("edge.kind = %v, want %q", edge["kind"], "evolves_into")
	}
	if label, _ := edge["label"].(string); label == "" {
		t.Errorf("expected a non-empty label on the typed edge, got %v", edge["label"])
	}
}

// TestArchitectureMap_StackForParam verifies FR-8: ?stack_for=<archId>
// includes that architecture's related_to tech-stack ring, and omitting the
// param returns the architecture-only base map (default-off).
func TestArchitectureMap_StackForParam(t *testing.T) {
	fooPath := "lifecycle/architecture/architectures/am-stack-foo.md"
	stackPath := "lifecycle/architecture/tech-stacks/am-stack.md"

	env := newTestEnv(t, nil)
	writeArchFile(t, env, stackPath, makeCleanSlugArtifact("AM Stack", "tech-stack", "approved", "Body."))
	writeArchFile(t, env, fooPath, "---\ntitle: AM Stack Foo\ntype: architecture\nstatus: approved\n"+
		"related_to:\n  - architecture/tech-stacks/am-stack.md\n---\n\nBody.\n")

	// Omitting stack_for: base map only, no tech-stack node.
	baseResp := env.doRequest("GET", "/api/p/testproject/architecture-map", nil)
	requireStatus(t, baseResp, http.StatusOK)
	baseData := readJSON(t, baseResp)
	baseNodes, _ := baseData["nodes"].([]any)
	if len(baseNodes) != 1 {
		t.Fatalf("expected 1 node in base map (stack ring off by default), got %d: %v", len(baseNodes), baseNodes)
	}

	// With stack_for: tech-stack node + connecting edge included.
	stackResp := env.doRequest("GET", "/api/p/testproject/architecture-map?stack_for="+fooPath, nil)
	requireStatus(t, stackResp, http.StatusOK)
	stackData := readJSON(t, stackResp)
	stackNodes, _ := stackData["nodes"].([]any)
	if len(stackNodes) != 2 {
		t.Fatalf("expected 2 nodes with stack_for set, got %d: %v", len(stackNodes), stackNodes)
	}
	found := false
	for _, raw := range stackNodes {
		node, _ := raw.(map[string]any)
		if node["id"] == stackPath {
			found = true
			if node["type"] != "tech-stack" {
				t.Errorf("stack node type = %v, want tech-stack", node["type"])
			}
		}
	}
	if !found {
		t.Errorf("expected stack node %q in payload, got: %v", stackPath, stackNodes)
	}
}

// TestArchitectureMap_RequiresAuth verifies FR-10/auth parity with GET
// /graph: an unauthenticated request is rejected with 401, not served.
func TestArchitectureMap_RequiresAuth(t *testing.T) {
	env := newTestEnv(t, nil)
	env.logout() // newTestEnv auto-logs in; clear the session for this test.

	resp := env.doRequest("GET", "/api/p/testproject/architecture-map", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated request to /architecture-map, got %d", resp.StatusCode)
	}
}

// TestArchitectureMap_NoMutatingVariant verifies FR-10: the endpoint is
// read-only — POST/PUT/DELETE/PATCH on the same path are rejected, and never
// perform a write (no create/edit/delete/persist-layout endpoint exists).
func TestArchitectureMap_NoMutatingVariant(t *testing.T) {
	env := newTestEnv(t, nil)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		resp := env.doRequest(method, "/api/p/testproject/architecture-map", nil)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			t.Errorf("%s /architecture-map: expected rejection, got 200", method)
		}
	}
}

// TestArchitectureMap_UnknownProject verifies the endpoint shares the same
// error contract as handleGraph for an unknown project — no panic/500 leak
// with an unexpected shape.
func TestArchitectureMap_UnknownProject(t *testing.T) {
	env := newTestEnv(t, nil)

	graphResp := env.doRequest("GET", "/api/p/does-not-exist/graph", nil)
	graphStatus := graphResp.StatusCode
	graphResp.Body.Close()

	mapResp := env.doRequest("GET", "/api/p/does-not-exist/architecture-map", nil)
	defer mapResp.Body.Close()

	if mapResp.StatusCode != graphStatus {
		t.Errorf("architecture-map unknown-project status = %d, want parity with /graph's %d", mapResp.StatusCode, graphStatus)
	}
	if mapResp.StatusCode == http.StatusOK {
		t.Errorf("expected an error status for an unknown project, got 200")
	}
}

// TestArchitectureMap_LiveWatchReflectsNewFile verifies FR-12: writing a new
// type: architecture file into the indexed tree after boot, and letting the
// real fsnotify watcher re-scan it, makes the node appear on the next
// endpoint call — with no process restart.
func TestArchitectureMap_LiveWatchReflectsNewFile(t *testing.T) {
	env := newTestEnv(t, nil)

	path := "lifecycle/architecture/architectures/am-live-new.md"
	absPath := filepath.Join(env.projectRoot, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absPath, []byte(makeArchFixture("AM Live New", "Body.")), 0o644); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	var found bool
	for time.Now().Before(deadline) {
		resp := env.doRequest("GET", "/api/p/testproject/architecture-map", nil)
		if resp.StatusCode == http.StatusOK {
			data := readJSON(t, resp)
			nodes, _ := data["nodes"].([]any)
			for _, raw := range nodes {
				node, _ := raw.(map[string]any)
				if node["id"] == path {
					found = true
				}
			}
		} else {
			resp.Body.Close()
		}
		if found {
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	if !found {
		t.Fatalf("expected %q to appear in /architecture-map within 2s of the live watcher indexing it", path)
	}
}

// TestArchitectureMap_LiveWatchReflectsRemoval verifies FR-12: removing an
// architecture file removes its node (and its now-dangling edges) on the
// next call, with no process restart.
func TestArchitectureMap_LiveWatchReflectsRemoval(t *testing.T) {
	fooPath := "lifecycle/architecture/architectures/am-live-foo.md"
	barPath := "lifecycle/architecture/architectures/am-live-bar.md"

	env := newTestEnv(t, nil)
	writeArchFile(t, env, barPath, makeArchFixture("AM Live Bar", "Body."))
	writeArchFile(t, env, fooPath, "---\ntitle: AM Live Foo\ntype: architecture\nstatus: approved\n---\n\nSee [[am-live-bar]].\n")

	// Sanity: both nodes and the connecting edge are present before removal.
	before := readJSON(t, env.doRequest("GET", "/api/p/testproject/architecture-map", nil))
	beforeNodes, _ := before["nodes"].([]any)
	if len(beforeNodes) != 2 {
		t.Fatalf("expected 2 nodes before removal, got %d: %v", len(beforeNodes), beforeNodes)
	}

	absPath := filepath.Join(env.projectRoot, filepath.FromSlash(barPath))
	if err := os.Remove(absPath); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	var gone bool
	for time.Now().Before(deadline) {
		resp := env.doRequest("GET", "/api/p/testproject/architecture-map", nil)
		if resp.StatusCode == http.StatusOK {
			data := readJSON(t, resp)
			nodes, _ := data["nodes"].([]any)
			edges, _ := data["edges"].([]any)
			if len(nodes) == 1 {
				gone = true
				if len(edges) != 0 {
					t.Errorf("expected no dangling edges after removing %q, got: %v", barPath, edges)
				}
			}
		} else {
			resp.Body.Close()
		}
		if gone {
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	if !gone {
		t.Fatalf("expected %q's node to disappear from /architecture-map within 2s of removal", barPath)
	}
}

// TestArchitectureMap_GraphUnaffected verifies the non-goal called out in the
// plan: the main artifact/lineage graph (/graph) is unchanged by this
// feature — querying /architecture-map does not perturb the /graph response
// for the same project, before or after an architecture fixture is indexed.
func TestArchitectureMap_GraphUnaffected(t *testing.T) {
	ideaPath := "lifecycle/ideas/am-graph-idea.md"
	archPath := "lifecycle/architecture/architectures/am-graph-arch.md"

	seeds := []seedArtifact{
		{relPath: ideaPath, content: makeArtifact("AM Graph Idea", "idea", "draft", "am-graph-idea", "", "Body.")},
	}
	env := newTestEnv(t, seeds)
	writeArchFile(t, env, archPath, makeArchFixture("AM Graph Arch", "Body."))

	before := graphResponseForProject(t, env)
	beforeNodes, _ := before["nodes"].([]any)
	if findNodeByID(beforeNodes, archPath) == nil {
		t.Fatalf("expected the indexed architecture artifact %q to appear in /graph (it's a regular artifact too), got nodes: %v", archPath, beforeNodes)
	}

	// Exercise the architecture-map endpoint in between.
	mapResp := env.doRequest("GET", "/api/p/testproject/architecture-map", nil)
	requireStatus(t, mapResp, http.StatusOK)

	after := graphResponseForProject(t, env)
	afterNodes, _ := after["nodes"].([]any)
	if len(beforeNodes) != len(afterNodes) {
		t.Fatalf("/graph node count changed after querying /architecture-map: %d -> %d", len(beforeNodes), len(afterNodes))
	}
}
