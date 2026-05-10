// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestGraphReleasesPerf_BackendResponseTime verifies that
// GET /graph?include_releases=true with 500 artifacts and 20 releases responds
// in under 500ms (Milestone 7, test case 1 of the releases overlay test plan).
func TestGraphReleasesPerf_BackendResponseTime(t *testing.T) {
	const numArtifacts = 500
	const numReleases = 20

	// Build seed artifacts evenly distributed across ideas and defects.
	// Each artifact is assigned to one of the 20 releases by index.
	seeds := make([]seedArtifact, 0, numArtifacts)
	for i := 0; i < numArtifacts; i++ {
		relName := fmt.Sprintf("gr-pt-v%02d", i%numReleases)
		if i%5 == 0 {
			slug := fmt.Sprintf("gr-pt-defect-%04d", i)
			seeds = append(seeds, seedArtifact{
				relPath: fmt.Sprintf("lifecycle/defects/%s.md", slug),
				content: makeArtifactWithRelease(
					"Perf Defect "+slug, "defect", "draft", slug, relName, "Body.",
				),
			})
		} else {
			slug := fmt.Sprintf("gr-pt-idea-%04d", i)
			seeds = append(seeds, seedArtifact{
				relPath: fmt.Sprintf("lifecycle/ideas/%s.md", slug),
				content: makeArtifactWithRelease(
					"Perf Idea "+slug, "idea", "draft", slug, relName, "Body.",
				),
			})
		}
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Create the 20 releases so assignment look-ups resolve properly.
	// Distribute across months 01–12 (some months used twice; names are unique).
	for i := 0; i < numReleases; i++ {
		month := (i % 12) + 1
		resp := env.doRequest("POST", "/api/p/testproject/releases", map[string]any{
			"name":       fmt.Sprintf("gr-pt-v%02d", i),
			"status":     "planned",
			"start_date": fmt.Sprintf("2026-%02d-01", month),
		})
		requireStatus(t, resp, http.StatusCreated)
		resp.Body.Close()
	}

	start := time.Now()
	resp := env.doRequest("GET", "/api/p/testproject/graph?include_releases=true", nil)
	elapsed := time.Since(start)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	t.Logf("GET /graph?include_releases=true: %d artifacts, %d releases → %v", numArtifacts, numReleases, elapsed)

	if elapsed > 500*time.Millisecond {
		t.Errorf("response time %v exceeds 500ms threshold for %d artifacts + %d releases",
			elapsed, numArtifacts, numReleases)
	}

	// Sanity: the response must contain nodes (artifacts + at least release nodes).
	nodes, _ := data["nodes"].([]any)
	if len(nodes) < numArtifacts+numReleases+1 { // +1 for Backlog
		t.Errorf("expected at least %d nodes (artifacts + releases + Backlog), got %d",
			numArtifacts+numReleases+1, len(nodes))
	}
}

// TestGraphReleasesPerf_BaselineComparison verifies that the baseline
// GET /graph (without overlay) is not regressed by the release overlay feature:
// both endpoints must respond in under 500ms for the same dataset.
func TestGraphReleasesPerf_BaselineComparison(t *testing.T) {
	const numArtifacts = 200
	const numReleases = 10

	seeds := make([]seedArtifact, 0, numArtifacts)
	for i := 0; i < numArtifacts; i++ {
		relName := fmt.Sprintf("gr-bc-v%02d", i%numReleases)
		slug := fmt.Sprintf("gr-bc-idea-%04d", i)
		seeds = append(seeds, seedArtifact{
			relPath: fmt.Sprintf("lifecycle/ideas/%s.md", slug),
			content: makeArtifactWithRelease(
				"BC Idea "+slug, "idea", "draft", slug, relName, "Body.",
			),
		})
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	for i := 0; i < numReleases; i++ {
		month := (i % 12) + 1
		resp := env.doRequest("POST", "/api/p/testproject/releases", map[string]any{
			"name":       fmt.Sprintf("gr-bc-v%02d", i),
			"status":     "planned",
			"start_date": fmt.Sprintf("2026-%02d-01", month),
		})
		requireStatus(t, resp, http.StatusCreated)
		resp.Body.Close()
	}

	// Baseline: no overlay.
	t0 := time.Now()
	resp0 := env.doRequest("GET", "/api/p/testproject/graph", nil)
	baselineElapsed := time.Since(t0)
	requireStatus(t, resp0, http.StatusOK)
	resp0.Body.Close()

	// Overlay: with include_releases=true.
	t1 := time.Now()
	resp1 := env.doRequest("GET", "/api/p/testproject/graph?include_releases=true", nil)
	overlayElapsed := time.Since(t1)
	requireStatus(t, resp1, http.StatusOK)
	resp1.Body.Close()

	t.Logf("baseline /graph: %v; overlay /graph?include_releases=true: %v", baselineElapsed, overlayElapsed)

	if baselineElapsed > 500*time.Millisecond {
		t.Errorf("baseline /graph response time %v exceeds 500ms threshold", baselineElapsed)
	}
	if overlayElapsed > 500*time.Millisecond {
		t.Errorf("overlay /graph?include_releases=true response time %v exceeds 500ms threshold", overlayElapsed)
	}
}
