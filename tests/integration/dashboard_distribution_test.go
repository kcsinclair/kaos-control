// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── Milestone 2 ───────────────────────────────────────────────────────────────

// TestStatusDistribution_Empty verifies that an empty project returns an empty
// (non-null) distribution array.
func TestStatusDistribution_Empty(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/status-distribution", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	dist, ok := data["distribution"]
	if !ok {
		t.Fatal("response missing 'distribution' key")
	}
	items, ok := dist.([]any)
	if !ok {
		t.Fatalf("'distribution' is not an array, got %T", dist)
	}
	if len(items) != 0 {
		t.Errorf("expected empty distribution for empty project, got %d entries", len(items))
	}
}

// TestStatusDistribution_CorrectCounts verifies that tickets are correctly
// grouped and counted by status.
func TestStatusDistribution_CorrectCounts(t *testing.T) {
	seeds := []seedArtifact{
		// 3 planning tickets
		{relPath: "lifecycle/requirements/dist-planning-1.md",
			content: makeArtifact("Dist Planning 1", "ticket", "planning", "dist-planning-1", "", "Body.")},
		{relPath: "lifecycle/requirements/dist-planning-2.md",
			content: makeArtifact("Dist Planning 2", "ticket", "planning", "dist-planning-2", "", "Body.")},
		{relPath: "lifecycle/requirements/dist-planning-3.md",
			content: makeArtifact("Dist Planning 3", "ticket", "planning", "dist-planning-3", "", "Body.")},
		// 2 in-development tickets
		{relPath: "lifecycle/requirements/dist-dev-1.md",
			content: makeArtifact("Dist Dev 1", "ticket", "in-development", "dist-dev-1", "", "Body.")},
		{relPath: "lifecycle/requirements/dist-dev-2.md",
			content: makeArtifact("Dist Dev 2", "ticket", "in-development", "dist-dev-2", "", "Body.")},
		// 1 blocked ticket; must include ## Open Questions so autoblock does not
		// auto-transition the artifact back to draft on index.
		{relPath: "lifecycle/requirements/dist-blocked-1.md",
			content: makeBlockedArtifact("Dist Blocked 1", "ticket", "dist-blocked-1", "")},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/status-distribution", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	counts := extractDistributionCounts(t, data)

	if counts["planning"] != 3 {
		t.Errorf("planning count: want 3, got %d", counts["planning"])
	}
	if counts["in-development"] != 2 {
		t.Errorf("in-development count: want 2, got %d", counts["in-development"])
	}
	if counts["blocked"] != 1 {
		t.Errorf("blocked count: want 1, got %d", counts["blocked"])
	}
}

// TestStatusDistribution_ExcludesDoneAndAbandoned verifies that tickets with
// status "done" or "abandoned" are not included in the distribution.
func TestStatusDistribution_ExcludesDoneAndAbandoned(t *testing.T) {
	seeds := []seedArtifact{
		{relPath: "lifecycle/requirements/dist-done-1.md",
			content: makeArtifact("Dist Done 1", "ticket", "done", "dist-done-1", "", "Body.")},
		{relPath: "lifecycle/requirements/dist-done-2.md",
			content: makeArtifact("Dist Done 2", "ticket", "done", "dist-done-2", "", "Body.")},
		{relPath: "lifecycle/requirements/dist-abandoned-1.md",
			content: makeArtifact("Dist Abandoned 1", "ticket", "abandoned", "dist-abandoned-1", "", "Body.")},
		// One visible ticket so the array is not empty.
		{relPath: "lifecycle/requirements/dist-visible-1.md",
			content: makeArtifact("Dist Visible 1", "ticket", "planning", "dist-visible-1", "", "Body.")},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/status-distribution", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	counts := extractDistributionCounts(t, data)

	if _, hasDone := counts["done"]; hasDone {
		t.Errorf("'done' status should be excluded from distribution")
	}
	if _, hasAbandoned := counts["abandoned"]; hasAbandoned {
		t.Errorf("'abandoned' status should be excluded from distribution")
	}
	// Only planning should appear.
	if counts["planning"] != 1 {
		t.Errorf("planning count: want 1, got %d", counts["planning"])
	}
}

// TestStatusDistribution_UpdatesAfterReindex verifies that adding a new ticket
// on disk and re-indexing causes the distribution to reflect the change.
// The re-index is triggered via the PUT /artifacts/* API endpoint.
func TestStatusDistribution_UpdatesAfterReindex(t *testing.T) {
	seeds := []seedArtifact{
		{relPath: "lifecycle/requirements/dist-reindex-1.md",
			content: makeArtifact("Dist Reindex 1", "ticket", "in-qa", "dist-reindex-1", "", "Body.")},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Verify initial distribution.
	resp := env.doRequest("GET", "/api/p/testproject/dashboard/status-distribution", nil)
	requireStatus(t, resp, 200)
	before := extractDistributionCounts(t, readJSON(t, resp))
	if before["in-qa"] != 1 {
		t.Fatalf("initial in-qa count: want 1, got %d", before["in-qa"])
	}

	// Write a new ticket directly to disk and wait for the watcher debounce.
	newPath := filepath.Join(env.projectRoot, "lifecycle/requirements/dist-reindex-2.md")
	content := makeArtifact("Dist Reindex 2", "ticket", "in-qa", "dist-reindex-2", "", "Body.")
	if err := os.WriteFile(newPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for the watcher (150 ms debounce + processing time).
	time.Sleep(400 * time.Millisecond)

	// Check that the distribution updated.
	resp = env.doRequest("GET", "/api/p/testproject/dashboard/status-distribution", nil)
	requireStatus(t, resp, 200)
	after := extractDistributionCounts(t, readJSON(t, resp))
	if after["in-qa"] != 2 {
		t.Errorf("after reindex in-qa count: want 2, got %d", after["in-qa"])
	}
}

// ── helper ────────────────────────────────────────────────────────────────────

// extractDistributionCounts parses the distribution array from a stats-distribution
// API response into a map of status → count for easy assertion.
func extractDistributionCounts(t *testing.T, data map[string]any) map[string]int {
	t.Helper()
	raw, ok := data["distribution"]
	if !ok {
		t.Fatal("response missing 'distribution' key")
	}
	items, ok := raw.([]any)
	if !ok {
		t.Fatalf("'distribution' is not an array, got %T", raw)
	}
	out := make(map[string]int, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		status, _ := entry["status"].(string)
		count, _ := entry["count"].(float64)
		out[status] = int(count)
	}
	return out
}
