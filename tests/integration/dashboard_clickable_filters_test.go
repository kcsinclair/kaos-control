//go:build integration

package integration

import (
	"io"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// dashboardClickableFiltersSeeds returns a deterministic set of lifecycle artifacts
// covering multiple statuses for dashboard-clickable-filters tests.
//
// Fixture inventory:
//   - 2× status=draft  (ideas)
//   - 1× status=blocked ticket (includes "## Open Questions" to prevent auto-transition)
//   - 1× status=in-development ticket
//   - 1× status=done ticket
//   - 1× status=planning ticket
func dashboardClickableFiltersSeeds() []seedArtifact {
	return []seedArtifact{
		// 2 draft ideas
		{relPath: "lifecycle/ideas/dcf-draft-1.md",
			content: makeArtifact("DCF Draft Idea 1", "idea", "draft", "dcf-draft-1", "", "Body.")},
		{relPath: "lifecycle/ideas/dcf-draft-2.md",
			content: makeArtifact("DCF Draft Idea 2", "idea", "draft", "dcf-draft-2", "", "Body.")},
		// 1 blocked ticket — must include "## Open Questions" so the indexer
		// autoblock rule does not transition it back to "draft" on startup.
		{relPath: "lifecycle/requirements/dcf-blocked-2.md",
			content: makeBlockedArtifact("DCF Blocked Ticket", "ticket", "dcf-blocked", "")},
		// 1 in-development ticket
		{relPath: "lifecycle/requirements/dcf-indev-2.md",
			content: makeArtifact("DCF In-Dev Ticket", "ticket", "in-development", "dcf-indev", "", "Body.")},
		// 1 done ticket
		{relPath: "lifecycle/requirements/dcf-done-2.md",
			content: makeArtifact("DCF Done Ticket", "ticket", "done", "dcf-done", "", "Body.")},
		// 1 planning ticket
		{relPath: "lifecycle/requirements/dcf-planning-2.md",
			content: makeArtifact("DCF Planning Ticket", "ticket", "planning", "dcf-planning", "", "Body.")},
	}
}

// ── Milestone 2: ArtifactListView Query Parameter Filter Tests ────────────────

// TestDCF_DirectNavigationWithStatusFilter verifies that
// GET /api/p/:project/artifacts?status=blocked returns only artifacts with
// status=blocked, and the total matches the count of seeded blocked artifacts.
//
// This is the server-side contract for M2-TC1 (direct navigation with status filter).
func TestDCF_DirectNavigationWithStatusFilter(t *testing.T) {
	env := newTestEnv(t, dashboardClickableFiltersSeeds())
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?status=blocked", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	total, _ := data["total"].(float64)

	if int(total) != 1 {
		t.Errorf("blocked filter: expected total=1, got %d", int(total))
	}
	if len(items) != 1 {
		t.Errorf("blocked filter: expected 1 item, got %d", len(items))
	}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if status, _ := item["status"].(string); status != "blocked" {
			t.Errorf("blocked filter returned artifact with status=%q, expected blocked", status)
		}
	}
}

// TestDCF_DirectNavigationNoFilter verifies that GET /api/p/:project/artifacts
// with no filter returns all artifacts (the "Lifecycle Total" card target).
//
// This is the server-side contract for M2-TC2.
func TestDCF_DirectNavigationNoFilter(t *testing.T) {
	env := newTestEnv(t, dashboardClickableFiltersSeeds())
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	total, _ := data["total"].(float64)

	if int(total) == 0 {
		t.Error("no-filter request: expected at least 1 artifact, got 0")
	}
	if len(items) != int(total) {
		t.Errorf("no-filter request: total=%d but items length=%d", int(total), len(items))
	}
}

// TestDCF_DeepLinkBookmarkFidelity verifies that querying with status=draft
// returns only draft artifacts. This covers M2-TC3 (deep-link bookmark fidelity)
// at the API level — the filter is applied correctly regardless of how the user
// arrived at the URL.
func TestDCF_DeepLinkBookmarkFidelity(t *testing.T) {
	env := newTestEnv(t, dashboardClickableFiltersSeeds())
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?status=draft", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	total, _ := data["total"].(float64)

	if int(total) != 2 {
		t.Errorf("draft filter: expected total=2 (2 draft ideas), got %d", int(total))
	}
	if len(items) != 2 {
		t.Errorf("draft filter: expected 2 items, got %d", len(items))
	}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if status, _ := item["status"].(string); status != "draft" {
			t.Errorf("deep-link draft filter returned artifact with status=%q", status)
		}
	}
}

// TestDCF_UnknownStatusGracefulDegradation verifies that querying with an
// unrecognised status value returns HTTP 200 with an empty items array, not an
// error response. Covers M2-TC4.
func TestDCF_UnknownStatusGracefulDegradation(t *testing.T) {
	env := newTestEnv(t, dashboardClickableFiltersSeeds())
	env.login("admin@test.local", "admin-pass-123")

	q := url.QueryEscape("nonexistent")
	resp := env.doRequest("GET", "/api/p/testproject/artifacts?status="+q, nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	total, _ := data["total"].(float64)
	items, _ := data["items"].([]any)

	if int(total) != 0 {
		t.Errorf("unknown status: expected total=0, got %d", int(total))
	}
	if len(items) != 0 {
		t.Errorf("unknown status: expected 0 items, got %d", len(items))
	}
}

// ── Key invariant: distribution counts match filtered list counts ─────────────

// TestDCF_PieSegmentClickNavigationContract verifies the core invariant for
// StatusDistributionWidget click-through: for each status returned by the
// status-distribution endpoint, the count matches the number of artifacts
// returned by the artifacts API filtered to that status.
//
// If this breaks, clicking a pie segment would navigate to a filtered list with
// a different count than the dashboard displayed — a regression.
func TestDCF_PieSegmentClickNavigationContract(t *testing.T) {
	env := newTestEnv(t, dashboardClickableFiltersSeeds())
	env.login("admin@test.local", "admin-pass-123")

	distResp := env.doRequest("GET", "/api/p/testproject/dashboard/status-distribution", nil)
	requireStatus(t, distResp, 200)
	distData := readJSON(t, distResp)
	distCounts := extractDistributionCounts(t, distData)

	if len(distCounts) == 0 {
		t.Skip("no distribution entries to check (empty project)")
	}

	for status, distCount := range distCounts {
		listResp := env.doRequest("GET", "/api/p/testproject/artifacts?status="+url.QueryEscape(status), nil)
		requireStatus(t, listResp, 200)
		listData := readJSON(t, listResp)
		listTotal, _ := listData["total"].(float64)
		listResp.Body.Close()

		if int(listTotal) != distCount {
			t.Errorf("status %q: distribution count=%d, artifacts filter count=%d — "+
				"clicking the pie segment would show a different count than displayed",
				status, distCount, int(listTotal))
		}
	}
}

// TestDCF_BlockedCardClickContract verifies that the blocked count in dashboard
// stats is consistent with a status=blocked artifact query.
// The stats "blocked" count includes clarifying artifacts (which is intentional),
// so we verify that the exact-match filter is a subset of the stat count.
func TestDCF_BlockedCardClickContract(t *testing.T) {
	env := newTestEnv(t, dashboardClickableFiltersSeeds())
	env.login("admin@test.local", "admin-pass-123")

	statsResp := env.doRequest("GET", "/api/p/testproject/dashboard/stats", nil)
	requireStatus(t, statsResp, 200)
	statsData := readJSON(t, statsResp)
	blockedStat, _ := statsData["blocked"].(float64)

	listResp := env.doRequest("GET", "/api/p/testproject/artifacts?status=blocked", nil)
	requireStatus(t, listResp, 200)
	listData := readJSON(t, listResp)
	listTotal, _ := listData["total"].(float64)

	if int(listTotal) == 0 {
		t.Error("expected at least 1 blocked artifact in filtered list")
	}
	// The stat count includes clarifying; filtered list is exact. Stat >= filtered.
	if int(blockedStat) < int(listTotal) {
		t.Errorf("dashboard blocked stat (%d) < filtered list total (%d) — invariant broken",
			int(blockedStat), int(listTotal))
	}
}

// ── Milestone 6: Regression Tests ────────────────────────────────────────────

// TestDCF_Regression_DashboardEndpointsLoad verifies that the dashboard stats
// and status-distribution endpoints return 200 with valid JSON when artifacts
// are present. This is M6-TC1 at the API level.
func TestDCF_Regression_DashboardEndpointsLoad(t *testing.T) {
	env := newTestEnv(t, dashboardClickableFiltersSeeds())
	env.login("admin@test.local", "admin-pass-123")

	for _, path := range []string{
		"/api/p/testproject/dashboard/stats",
		"/api/p/testproject/dashboard/status-distribution",
	} {
		resp := env.doRequest("GET", path, nil)
		if resp.StatusCode != 200 {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			t.Errorf("GET %s: expected 200, got %d: %s", path, resp.StatusCode, b)
			continue
		}
		_ = readJSON(t, resp) // validates parseable JSON
	}
}

// TestDCF_Regression_FeedEndpointReachable verifies that the project feed API
// returns HTTP 200, confirming that the "View all" link target (/p/:project/feed)
// is a valid route. This is M6-TC3 at the API level.
func TestDCF_Regression_FeedEndpointReachable(t *testing.T) {
	env := newTestEnv(t, dashboardClickableFiltersSeeds())
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/feed", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	// Feed must return a valid JSON object with an "events" array.
	if _, ok := data["events"]; !ok {
		t.Error("feed response missing 'events' key")
	}
}

// TestDCF_Regression_FeedActivityLinksHaveValidPaths verifies that feed events
// that reference an artifact path point to paths that exist in the index.
// This is the server-side contract for M6-TC2 (Activity Feed links still work).
func TestDCF_Regression_FeedActivityLinksHaveValidPaths(t *testing.T) {
	env := newTestEnv(t, dashboardClickableFiltersSeeds())
	env.login("admin@test.local", "admin-pass-123")

	// Fetch all artifacts to get known paths.
	listResp := env.doRequest("GET", "/api/p/testproject/artifacts", nil)
	requireStatus(t, listResp, 200)
	listData := readJSON(t, listResp)

	knownPaths := map[string]bool{}
	if items, ok := listData["items"].([]any); ok {
		for _, raw := range items {
			item, _ := raw.(map[string]any)
			if path, _ := item["path"].(string); path != "" {
				knownPaths[path] = true
			}
		}
	}

	// Fetch feed events.
	feedResp := env.doRequest("GET", "/api/p/testproject/feed?limit=50", nil)
	requireStatus(t, feedResp, 200)
	feedData := readJSON(t, feedResp)

	events, _ := feedData["events"].([]any)
	for _, raw := range events {
		ev, _ := raw.(map[string]any)
		artifactPath, _ := ev["artifact_path"].(string)
		if artifactPath == "" {
			continue // some events are not artifact-specific
		}
		if !knownPaths[artifactPath] {
			t.Errorf("feed event references artifact_path=%q which is not in the index", artifactPath)
		}
	}
}

// TestDCF_Regression_StatsUpdateAfterReindex verifies that after a new artifact
// is added and re-indexed, the dashboard stats endpoint reflects the updated
// counts. This is the API-level contract for M6-TC5.
func TestDCF_Regression_StatsUpdateAfterReindex(t *testing.T) {
	env := newTestEnv(t, dashboardClickableFiltersSeeds())
	env.login("admin@test.local", "admin-pass-123")

	// Record initial counts.
	initResp := env.doRequest("GET", "/api/p/testproject/dashboard/stats", nil)
	requireStatus(t, initResp, 200)
	initData := readJSON(t, initResp)
	initTotal, _ := initData["total_tickets"].(float64)

	// Write a new ticket directly to disk and wait for the watcher debounce.
	newPath := filepath.Join(env.projectRoot, "lifecycle/requirements/dcf-reindex-2.md")
	newContent := makeArtifact("DCF Reindex Ticket", "ticket", "planning", "dcf-reindex", "", "Body.")
	if err := os.WriteFile(newPath, []byte(newContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for the watcher debounce (150 ms) plus processing margin.
	time.Sleep(400 * time.Millisecond)

	// Stats must now show one more ticket.
	afterResp := env.doRequest("GET", "/api/p/testproject/dashboard/stats", nil)
	requireStatus(t, afterResp, 200)
	afterData := readJSON(t, afterResp)
	afterTotal, _ := afterData["total_tickets"].(float64)

	if int(afterTotal) <= int(initTotal) {
		t.Errorf("stats did not update after reindex: initial total=%d, after total=%d",
			int(initTotal), int(afterTotal))
	}
}
