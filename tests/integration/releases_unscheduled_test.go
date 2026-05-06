//go:build integration

package integration

import (
	"net/http"
	"strings"
	"testing"
)

// ── Milestone 7: Unscheduled release tests ────────────────────────────────────

// TestReleaseUnscheduled_CreateSucceeds verifies that a release with no
// start_date and no end_date is created successfully with 201.
func TestReleaseUnscheduled_CreateSucceeds(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/releases", map[string]any{
		"name":   "v-unsched",
		"status": "planned",
	})
	requireStatus(t, resp, http.StatusCreated)
	body := readJSON(t, resp)

	rel, _ := body["release"].(map[string]any)
	if rel["start_date"] != nil {
		t.Errorf("start_date should be nil, got %v", rel["start_date"])
	}
	if rel["end_date"] != nil {
		t.Errorf("end_date should be nil, got %v", rel["end_date"])
	}
}

// TestReleaseUnscheduled_SortsAfterScheduled verifies that unscheduled releases
// appear after all scheduled releases in the GET /releases list.
func TestReleaseUnscheduled_SortsAfterScheduled(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{"name": "v-sched-early", "status": "planned", "start_date": "2026-01-01", "end_date": "2026-03-31"})
	createRelease(t, env, map[string]any{"name": "v-unsched-a", "status": "planned"})
	createRelease(t, env, map[string]any{"name": "v-sched-late", "status": "planned", "start_date": "2026-07-01", "end_date": "2026-09-30"})

	resp := env.doRequest("GET", "/api/p/testproject/releases", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	releases, _ := body["releases"].([]any)
	if len(releases) != 3 {
		t.Fatalf("want 3 releases, got %d", len(releases))
	}

	// Last release must be unscheduled.
	last, _ := releases[2].(map[string]any)
	if name, _ := last["name"].(string); name != "v-unsched-a" {
		t.Errorf("last release should be unscheduled, got %q", name)
	}
	if last["start_date"] != nil {
		t.Errorf("last release start_date should be nil, got %v", last["start_date"])
	}
}

// TestReleaseUnscheduled_TwoUnscheduledOrderByName verifies that when there are
// multiple unscheduled releases they are sorted by name.
func TestReleaseUnscheduled_TwoUnscheduledOrderByName(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Create in reverse alphabetical order.
	createRelease(t, env, map[string]any{"name": "v-unsched-zzz", "status": "planned"})
	createRelease(t, env, map[string]any{"name": "v-unsched-aaa", "status": "planned"})

	resp := env.doRequest("GET", "/api/p/testproject/releases", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	releases, _ := body["releases"].([]any)
	if len(releases) != 2 {
		t.Fatalf("want 2 releases, got %d", len(releases))
	}

	first, _ := releases[0].(map[string]any)
	second, _ := releases[1].(map[string]any)
	if name, _ := first["name"].(string); name != "v-unsched-aaa" {
		t.Errorf("first unscheduled release: want %q, got %q", "v-unsched-aaa", name)
	}
	if name, _ := second["name"].(string); name != "v-unsched-zzz" {
		t.Errorf("second unscheduled release: want %q, got %q", "v-unsched-zzz", name)
	}
}

// TestReleaseUnscheduled_ArtifactAssignment verifies that artifacts can be
// assigned to an unscheduled release and appear in its artifact list.
func TestReleaseUnscheduled_ArtifactAssignment(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/ru-idea-1.md",
			content: makeArtifactWithRelease("RU Idea 1", "idea", "draft", "ru-idea-1", "v-ru-assign", "Body."),
		},
		{
			relPath: "lifecycle/ideas/ru-idea-2.md",
			content: makeArtifactWithRelease("RU Idea 2", "idea", "draft", "ru-idea-2", "v-ru-assign", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Unscheduled release.
	data := createRelease(t, env, map[string]any{"name": "v-ru-assign", "status": "planned"})
	id := releaseID(t, data)

	resp := env.doRequest("GET", releasePath(id)+"/artifacts", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	items, _ := body["items"].([]any)
	if len(items) != 2 {
		t.Errorf("unscheduled release artifact list: want 2, got %d", len(items))
	}
}

// TestReleaseUnscheduled_RoadmapGraphDisconnected verifies that unscheduled
// releases appear as nodes in the roadmap graph but have no timeline edges.
func TestReleaseUnscheduled_RoadmapGraphDisconnected(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// One scheduled, one unscheduled.
	createRelease(t, env, map[string]any{
		"name": "v-sched-rg", "status": "planned",
		"start_date": "2026-01-01", "end_date": "2026-03-31",
	})
	unschedData := createRelease(t, env, map[string]any{
		"name": "v-unsched-rg", "status": "planned",
	})
	unschedID := releaseID(t, unschedData)
	unschedNodeID := releasePath(unschedID)[len("/api/p/testproject"):]
	// The node ID for a release in the roadmap graph is "release:<id>".
	unschedGraphID := strings.TrimPrefix(unschedNodeID, "/releases/")
	// Build the graph node id as the handler does: "release:<id>"
	_ = unschedGraphID // used below via unschedID

	resp := env.doRequest("GET", "/api/p/testproject/releases/graph", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	nodes, _ := data["nodes"].([]any)
	edges, _ := data["edges"].([]any)

	// Both releases must appear as nodes.
	releaseNodeIDs := map[string]bool{}
	for _, raw := range nodes {
		node, _ := raw.(map[string]any)
		if typ, _ := node["type"].(string); typ == "release" {
			id, _ := node["id"].(string)
			releaseNodeIDs[id] = true
		}
	}
	if len(releaseNodeIDs) != 2 {
		t.Errorf("roadmap graph: want 2 release nodes, got %d", len(releaseNodeIDs))
	}

	// The unscheduled release node must exist.
	unschedExpectedID := strings.Join([]string{"release", releasePath2(unschedID)}, ":")
	if !releaseNodeIDs[unschedExpectedID] {
		t.Errorf("roadmap graph: unscheduled release node %q not found; nodes: %v", unschedExpectedID, releaseNodeIDs)
	}

	// No timeline edge should involve the unscheduled release.
	for _, raw := range edges {
		edge, _ := raw.(map[string]any)
		if kind, _ := edge["kind"].(string); kind == "timeline" {
			src, _ := edge["source"].(string)
			tgt, _ := edge["target"].(string)
			if src == unschedExpectedID || tgt == unschedExpectedID {
				t.Errorf("unscheduled release should not participate in timeline edges; got edge %q→%q", src, tgt)
			}
		}
	}
}

// TestReleaseUnscheduled_UpdateToAddDates verifies that adding dates to a
// previously unscheduled release moves it to the correct chronological position
// in the list.
func TestReleaseUnscheduled_UpdateToAddDates(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Scheduled release for Q2.
	createRelease(t, env, map[string]any{
		"name": "v-q2-sched", "status": "planned",
		"start_date": "2026-04-01", "end_date": "2026-06-30",
	})

	// Initially unscheduled — should appear last.
	data := createRelease(t, env, map[string]any{"name": "v-q1-unsched", "status": "planned"})
	unschedID := releaseID(t, data)

	listBefore := func() []string {
		resp := env.doRequest("GET", "/api/p/testproject/releases", nil)
		requireStatus(t, resp, http.StatusOK)
		body := readJSON(t, resp)
		releases, _ := body["releases"].([]any)
		names := make([]string, len(releases))
		for i, raw := range releases {
			rel, _ := raw.(map[string]any)
			names[i], _ = rel["name"].(string)
		}
		return names
	}

	beforeNames := listBefore()
	if len(beforeNames) < 2 || beforeNames[len(beforeNames)-1] != "v-q1-unsched" {
		t.Errorf("before scheduling: expected v-q1-unsched last, got %v", beforeNames)
	}

	// Schedule the release in Q1 (before Q2).
	resp := env.doRequest("PUT", releasePath(unschedID), map[string]any{
		"name":       "v-q1-unsched",
		"status":     "planned",
		"start_date": "2026-01-01",
		"end_date":   "2026-03-31",
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	afterNames := listBefore()
	if len(afterNames) < 2 {
		t.Fatalf("expected at least 2 releases after update, got %v", afterNames)
	}
	// v-q1-unsched (now Q1) should now appear before v-q2-sched.
	if afterNames[0] != "v-q1-unsched" {
		t.Errorf("after scheduling: expected v-q1-unsched first, got %v", afterNames)
	}
	if afterNames[1] != "v-q2-sched" {
		t.Errorf("after scheduling: expected v-q2-sched second, got %v", afterNames)
	}
}
