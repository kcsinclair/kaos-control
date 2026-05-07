//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestRoadmapGraph_Perf50Releases benchmarks GET /releases/graph with 50
// releases (mix of scheduled and unscheduled) and asserts that the API
// responds in under 100ms.
func TestRoadmapGraph_Perf50Releases(t *testing.T) {
	const numReleases = 50

	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Create 35 scheduled releases with quarterly cadence, then 15 unscheduled.
	// Scheduled: start 2020-01-01, increment by ~10 days each.
	for i := 0; i < 35; i++ {
		// Spread across ~350 days starting 2026-01-01 (10 days apart).
		year := 2026 + i/36
		day := (i % 36) * 10
		// Build date: start 2026-01-01 and add day offsets within the year.
		// Use a fixed base and add days directly as a date string approximation.
		_ = year
		_ = day
		startDate := fmt.Sprintf("2026-%02d-%02d",
			1+((i*10)/30)%12+1,
			1+((i*10)%30),
		)
		// Clamp to valid month/day to avoid date parsing errors.
		// Use a simpler approach: fixed months with incrementing start days.
		month := (i % 12) + 1
		startDate = fmt.Sprintf("2026-%02d-01", month)
		// Each release in the same month uses an incremented name for uniqueness.
		resp := env.doRequest("POST", "/api/p/testproject/releases", map[string]any{
			"name":       fmt.Sprintf("perf-sched-%03d", i),
			"status":     "planned",
			"start_date": startDate,
		})
		requireStatus(t, resp, http.StatusCreated)
		resp.Body.Close()
	}

	for i := 0; i < 15; i++ {
		resp := env.doRequest("POST", "/api/p/testproject/releases", map[string]any{
			"name":   fmt.Sprintf("perf-unsched-%03d", i),
			"status": "planned",
		})
		requireStatus(t, resp, http.StatusCreated)
		resp.Body.Close()
	}

	// Measure the GET /releases/graph response time.
	start := time.Now()
	resp := env.doRequest("GET", "/api/p/testproject/releases/graph", nil)
	elapsed := time.Since(start)

	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	t.Logf("GET /releases/graph for %d releases responded in %v", numReleases, elapsed)

	if elapsed > 100*time.Millisecond {
		t.Errorf("roadmap graph API for %d releases: response time %v exceeds 100ms limit", numReleases, elapsed)
	}

	nodes, _ := data["nodes"].([]any)
	edges, _ := data["edges"].([]any)

	// Sanity: at least numReleases+1 nodes (Backlog + all releases).
	if len(nodes) < numReleases+1 {
		t.Errorf("perf: want at least %d nodes, got %d", numReleases+1, len(nodes))
	}

	// Sanity: at least numReleases timeline edges (one per release from root).
	timelineCount := countEdgesByKind(edges, "timeline")
	if timelineCount < numReleases {
		t.Errorf("perf: want at least %d timeline edges, got %d", numReleases, timelineCount)
	}
}

// TestRoadmapGraph_Perf50ReleasesWithArtifacts benchmarks GET /releases/graph
// with 50 releases and 100 artifacts (2 per release), verifying the response
// still arrives in under 200ms.
func TestRoadmapGraph_Perf50ReleasesWithArtifacts(t *testing.T) {
	const numReleases = 50
	const artifactsPerRelease = 2

	seeds := make([]seedArtifact, 0, numReleases*artifactsPerRelease)
	for i := 0; i < numReleases; i++ {
		relName := fmt.Sprintf("perf-art-rel-%03d", i)
		for j := 0; j < artifactsPerRelease; j++ {
			slug := fmt.Sprintf("perf-art-%03d-%d", i, j)
			seeds = append(seeds, seedArtifact{
				relPath: fmt.Sprintf("lifecycle/ideas/%s.md", slug),
				content: makeArtifactWithRelease(
					fmt.Sprintf("Perf Art %d-%d", i, j),
					"idea", "draft", slug, relName,
					"Performance test body.",
				),
			})
		}
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	for i := 0; i < numReleases; i++ {
		month := (i % 12) + 1
		resp := env.doRequest("POST", "/api/p/testproject/releases", map[string]any{
			"name":       fmt.Sprintf("perf-art-rel-%03d", i),
			"status":     "planned",
			"start_date": fmt.Sprintf("2026-%02d-01", month),
		})
		requireStatus(t, resp, http.StatusCreated)
		resp.Body.Close()
	}

	start := time.Now()
	resp := env.doRequest("GET", "/api/p/testproject/releases/graph", nil)
	elapsed := time.Since(start)

	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	t.Logf("GET /releases/graph for %d releases + %d artifacts responded in %v",
		numReleases, numReleases*artifactsPerRelease, elapsed)

	if elapsed > 200*time.Millisecond {
		t.Errorf("roadmap graph with artifacts: response time %v exceeds 200ms limit", elapsed)
	}

	nodes, _ := data["nodes"].([]any)
	// Expect: 1 Backlog + 50 releases + 100 artifact nodes = 151.
	wantNodes := 1 + numReleases + numReleases*artifactsPerRelease
	if len(nodes) < wantNodes {
		t.Errorf("perf with artifacts: want at least %d nodes, got %d", wantNodes, len(nodes))
	}
}

// TestRoadmapGraph_Perf20ReleasesResponseShape verifies that the graph for 20
// releases has the expected structural properties at scale: every node has an
// id field and every timeline edge has source, target, and kind fields.
func TestRoadmapGraph_Perf20ReleasesResponseShape(t *testing.T) {
	const numReleases = 20

	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	for i := 0; i < numReleases; i++ {
		month := (i % 12) + 1
		resp := env.doRequest("POST", "/api/p/testproject/releases", map[string]any{
			"name":       fmt.Sprintf("perf-shape-%03d", i),
			"status":     "planned",
			"start_date": fmt.Sprintf("2026-%02d-01", month),
		})
		requireStatus(t, resp, http.StatusCreated)
		resp.Body.Close()
	}

	resp := env.doRequest("GET", "/api/p/testproject/releases/graph", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	nodes, _ := data["nodes"].([]any)
	edges, _ := data["edges"].([]any)

	// Every node must have a non-empty id.
	for idx, raw := range nodes {
		node, _ := raw.(map[string]any)
		id, _ := node["id"].(string)
		if id == "" {
			t.Errorf("node[%d] has missing or empty id", idx)
		}
	}

	// Every timeline edge must have source, target, and kind.
	for idx, raw := range edges {
		edge, _ := raw.(map[string]any)
		kind, _ := edge["kind"].(string)
		if kind != "timeline" {
			continue
		}
		src, _ := edge["source"].(string)
		tgt, _ := edge["target"].(string)
		if src == "" {
			t.Errorf("timeline edge[%d] has missing source", idx)
		}
		if tgt == "" {
			t.Errorf("timeline edge[%d] has missing target", idx)
		}
	}

	t.Logf("20-release graph: %d nodes, %d edges", len(nodes), len(edges))
}
