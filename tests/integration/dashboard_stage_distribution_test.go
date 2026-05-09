// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"testing"
)

// ── Milestone 1 ───────────────────────────────────────────────────────────────
//
// Integration tests for GET /api/p/:project/dashboard/stage-distribution.
// Each test spins up a full HTTP server with a seeded project and verifies
// the response shape, filtering, and edge-case behaviour.

// TestStageDistribution_Empty verifies that a project with no artifacts
// returns {"distribution": []} — an empty array, not null.
func TestStageDistribution_Empty(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/stage-distribution", nil)
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

// TestStageDistribution_HappyPath creates artifacts in multiple lifecycle
// stages and verifies the response contains the correct stage names and counts.
func TestStageDistribution_HappyPath(t *testing.T) {
	seeds := []seedArtifact{
		// 2 tickets in ideas
		{relPath: "lifecycle/ideas/stage-idea-1.md",
			content: makeArtifact("Stage Idea 1", "ticket", "draft", "stage-idea-1", "", "Body.")},
		{relPath: "lifecycle/ideas/stage-idea-2.md",
			content: makeArtifact("Stage Idea 2", "ticket", "draft", "stage-idea-2", "", "Body.")},
		// 3 tickets in requirements
		{relPath: "lifecycle/requirements/stage-req-1.md",
			content: makeArtifact("Stage Req 1", "ticket", "planning", "stage-req-1", "", "Body.")},
		{relPath: "lifecycle/requirements/stage-req-2.md",
			content: makeArtifact("Stage Req 2", "ticket", "in-development", "stage-req-2", "", "Body.")},
		{relPath: "lifecycle/requirements/stage-req-3.md",
			content: makeArtifact("Stage Req 3", "ticket", "in-qa", "stage-req-3", "", "Body.")},
		// 1 ticket in backend-plans
		{relPath: "lifecycle/backend-plans/stage-be-1.md",
			content: makeArtifact("Stage BE 1", "ticket", "draft", "stage-be-1", "", "Body.")},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/stage-distribution", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	counts := extractStageDistributionCounts(t, data)

	if counts["ideas"] != 2 {
		t.Errorf("ideas count: want 2, got %d", counts["ideas"])
	}
	if counts["requirements"] != 3 {
		t.Errorf("requirements count: want 3, got %d", counts["requirements"])
	}
	if counts["backend-plans"] != 1 {
		t.Errorf("backend-plans count: want 1, got %d", counts["backend-plans"])
	}
}

// TestStageDistribution_TrackedTypesFiltering configures Dashboard.TrackedTypes
// to a subset (plan-backend only). Creates artifacts of both tracked and
// untracked types. Verifies only tracked-type artifacts are counted.
func TestStageDistribution_TrackedTypesFiltering(t *testing.T) {
	// Custom config: only plan-backend is a tracked type.
	customCfg := defaultCfgYAML + `
dashboard:
  tracked_types: [plan-backend]
`
	seeds := []seedArtifact{
		// 2 plan-backend artifacts (tracked)
		{relPath: "lifecycle/backend-plans/tf-be-1.md",
			content: makeArtifact("TF BE 1", "plan-backend", "draft", "tf-be-1", "", "Body.")},
		{relPath: "lifecycle/backend-plans/tf-be-2.md",
			content: makeArtifact("TF BE 2", "plan-backend", "in-development", "tf-be-2", "", "Body.")},
		// 3 ticket artifacts (not tracked)
		{relPath: "lifecycle/requirements/tf-ticket-1.md",
			content: makeArtifact("TF Ticket 1", "ticket", "planning", "tf-ticket-1", "", "Body.")},
		{relPath: "lifecycle/requirements/tf-ticket-2.md",
			content: makeArtifact("TF Ticket 2", "ticket", "planning", "tf-ticket-2", "", "Body.")},
		{relPath: "lifecycle/requirements/tf-ticket-3.md",
			content: makeArtifact("TF Ticket 3", "ticket", "in-development", "tf-ticket-3", "", "Body.")},
	}

	env := newTestEnvWithCfgYAML(t, seeds, customCfg)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/stage-distribution", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	counts := extractStageDistributionCounts(t, data)

	// Only the plan-backend stage should appear; requirements should be absent.
	if counts["backend-plans"] != 2 {
		t.Errorf("backend-plans count: want 2, got %d", counts["backend-plans"])
	}
	if _, hasReq := counts["requirements"]; hasReq {
		t.Errorf("'requirements' stage should not appear when tickets are not tracked, but found %d entries", counts["requirements"])
	}
}

// TestStageDistribution_ExcludesDoneAndAbandoned verifies that artifacts with
// status "done" or "abandoned" are excluded from the distribution.
func TestStageDistribution_ExcludesDoneAndAbandoned(t *testing.T) {
	seeds := []seedArtifact{
		// done and abandoned — must not appear
		{relPath: "lifecycle/requirements/excl-done-1.md",
			content: makeArtifact("Excl Done 1", "ticket", "done", "excl-done-1", "", "Body.")},
		{relPath: "lifecycle/requirements/excl-done-2.md",
			content: makeArtifact("Excl Done 2", "ticket", "done", "excl-done-2", "", "Body.")},
		{relPath: "lifecycle/ideas/excl-aband-1.md",
			content: makeArtifact("Excl Aband 1", "ticket", "abandoned", "excl-aband-1", "", "Body.")},
		// One visible ticket so the array is non-empty.
		{relPath: "lifecycle/requirements/excl-visible-1.md",
			content: makeArtifact("Excl Visible 1", "ticket", "planning", "excl-visible-1", "", "Body.")},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/stage-distribution", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	raw, ok := data["distribution"]
	if !ok {
		t.Fatal("response missing 'distribution' key")
	}
	items := raw.([]any)

	// Verify done/abandoned stages are absent and only the planning ticket is counted.
	counts := extractStageDistributionCounts(t, data)

	if counts["requirements"] != 1 {
		t.Errorf("requirements count: want 1 (only planning), got %d", counts["requirements"])
	}

	// ideas stage should not appear (only abandoned tickets were there).
	if _, hasIdeas := counts["ideas"]; hasIdeas {
		t.Errorf("'ideas' stage should be absent (only abandoned artifact), got count %d", counts["ideas"])
	}

	// Sanity check: distribution has exactly one entry.
	if len(items) != 1 {
		t.Errorf("expected exactly 1 stage entry, got %d", len(items))
	}
}

// TestStageDistribution_MixedStatuses creates artifacts with draft,
// in-development, and done statuses in the same stage. Verifies that only
// non-done/non-abandoned artifacts are counted.
func TestStageDistribution_MixedStatuses(t *testing.T) {
	seeds := []seedArtifact{
		{relPath: "lifecycle/requirements/mix-draft-1.md",
			content: makeArtifact("Mix Draft 1", "ticket", "draft", "mix-draft-1", "", "Body.")},
		{relPath: "lifecycle/requirements/mix-dev-1.md",
			content: makeArtifact("Mix Dev 1", "ticket", "in-development", "mix-dev-1", "", "Body.")},
		// done — excluded
		{relPath: "lifecycle/requirements/mix-done-1.md",
			content: makeArtifact("Mix Done 1", "ticket", "done", "mix-done-1", "", "Body.")},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/stage-distribution", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	counts := extractStageDistributionCounts(t, data)

	// 2 non-done tickets; the done one must be excluded.
	if counts["requirements"] != 2 {
		t.Errorf("requirements count: want 2, got %d", counts["requirements"])
	}
}

// TestStageDistribution_SingleStage creates artifacts in one stage only and
// verifies the response contains exactly one entry.
func TestStageDistribution_SingleStage(t *testing.T) {
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/single-1.md",
			content: makeArtifact("Single 1", "ticket", "draft", "single-1", "", "Body.")},
		{relPath: "lifecycle/ideas/single-2.md",
			content: makeArtifact("Single 2", "ticket", "planning", "single-2", "", "Body.")},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/stage-distribution", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	raw, _ := data["distribution"]
	items := raw.([]any)
	if len(items) != 1 {
		t.Errorf("expected exactly 1 stage entry, got %d", len(items))
	}

	counts := extractStageDistributionCounts(t, data)
	if counts["ideas"] != 2 {
		t.Errorf("ideas count: want 2, got %d", counts["ideas"])
	}
}

// TestStageDistribution_AlphabeticalOrdering verifies the distribution array
// is sorted by stage name (alphabetical ascending).
func TestStageDistribution_AlphabeticalOrdering(t *testing.T) {
	seeds := []seedArtifact{
		// Insert in reverse alphabetical order to confirm ordering is not insertion-order.
		{relPath: "lifecycle/requirements/ord-req-1.md",
			content: makeArtifact("Ord Req 1", "ticket", "draft", "ord-req-1", "", "Body.")},
		{relPath: "lifecycle/ideas/ord-idea-1.md",
			content: makeArtifact("Ord Idea 1", "ticket", "draft", "ord-idea-1", "", "Body.")},
		{relPath: "lifecycle/backend-plans/ord-be-1.md",
			content: makeArtifact("Ord BE 1", "ticket", "draft", "ord-be-1", "", "Body.")},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/stage-distribution", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	raw, _ := data["distribution"]
	items := raw.([]any)

	if len(items) != 3 {
		t.Fatalf("expected 3 stage entries, got %d", len(items))
	}

	// Expected alphabetical order: backend-plans < ideas < requirements
	wantOrder := []string{"backend-plans", "ideas", "requirements"}
	for i, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("items[%d] is not an object", i)
		}
		gotStage, _ := entry["stage"].(string)
		if gotStage != wantOrder[i] {
			t.Errorf("items[%d].stage: want %q, got %q", i, wantOrder[i], gotStage)
		}
	}
}

// ── helper ────────────────────────────────────────────────────────────────────

// extractStageDistributionCounts parses the distribution array from a
// stage-distribution API response into a map of stage → count for assertions.
func extractStageDistributionCounts(t *testing.T, data map[string]any) map[string]int {
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
		stage, _ := entry["stage"].(string)
		count, _ := entry["count"].(float64)
		out[stage] = int(count)
	}
	return out
}
