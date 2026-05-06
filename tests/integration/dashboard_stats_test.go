//go:build integration

package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/index"
)

// ptr returns a pointer to s, used to satisfy *string fields in EventRow.
func strPtr(s string) *string { return &s }

// isoWeekStartForTest returns midnight on the Monday that begins the current
// ISO week, matching the logic in internal/http/dashboard.go.
func isoWeekStartForTest() time.Time {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday - 1))
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, now.Location())
}

// ── Milestone 1 ───────────────────────────────────────────────────────────────

// TestDashboardStats_Empty verifies that an empty project returns all zero counts.
func TestDashboardStats_Empty(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/stats", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	assertIntField(t, data, "total_tickets", 0)
	assertIntField(t, data, "in_progress", 0)
	assertIntField(t, data, "blocked", 0)
	assertIntField(t, data, "completed_this_week", 0)
}

// TestDashboardStats_MixedStatuses verifies counts with a realistic mix of ticket statuses.
// Checks total_tickets, in_progress, and blocked counts.
func TestDashboardStats_MixedStatuses(t *testing.T) {
	seeds := []seedArtifact{
		// in-development (counted in total + in_progress)
		{relPath: "lifecycle/requirements/stats-dev-1.md",
			content: makeArtifact("Stats Dev 1", "ticket", "in-development", "stats-dev-1", "", "Body.")},
		{relPath: "lifecycle/requirements/stats-dev-2.md",
			content: makeArtifact("Stats Dev 2", "ticket", "in-development", "stats-dev-2", "", "Body.")},
		// blocked (counted in total + blocked)
		{relPath: "lifecycle/requirements/stats-blocked-1.md",
			content: makeArtifact("Stats Blocked 1", "ticket", "blocked", "stats-blocked-1", "", "Body.")},
		// clarifying (also counted as blocked per implementation)
		{relPath: "lifecycle/requirements/stats-clarifying-1.md",
			content: makeArtifact("Stats Clarifying 1", "ticket", "clarifying", "stats-clarifying-1", "", "Body.")},
		// planning (counted in total, not in_progress or blocked)
		{relPath: "lifecycle/requirements/stats-planning-1.md",
			content: makeArtifact("Stats Planning 1", "ticket", "planning", "stats-planning-1", "", "Body.")},
		// done (counted in total)
		{relPath: "lifecycle/requirements/stats-done-1.md",
			content: makeArtifact("Stats Done 1", "ticket", "done", "stats-done-1", "", "Body.")},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/stats", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	// 5 non-abandoned tickets (all except abandoned; none abandoned here — all 6 count)
	assertIntField(t, data, "total_tickets", 6)
	assertIntField(t, data, "in_progress", 2)
	// blocked + clarifying = 2
	assertIntField(t, data, "blocked", 2)
}

// TestDashboardStats_CompletedThisWeek verifies that completed_this_week counts
// only done-transition events whose timestamp falls within the current ISO week.
// Events from previous weeks must not be included.
func TestDashboardStats_CompletedThisWeek(t *testing.T) {
	seeds := []seedArtifact{
		{relPath: "lifecycle/requirements/stats-ctw-1.md",
			content: makeArtifact("Stats CTW 1", "ticket", "done", "stats-ctw-1", "", "Body.")},
		{relPath: "lifecycle/requirements/stats-ctw-2.md",
			content: makeArtifact("Stats CTW 2", "ticket", "done", "stats-ctw-2", "", "Body.")},
		{relPath: "lifecycle/requirements/stats-ctw-old.md",
			content: makeArtifact("Stats CTW Old", "ticket", "done", "stats-ctw-old", "", "Body.")},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	weekStart := isoWeekStartForTest()

	// Insert two events this week.
	path1 := "lifecycle/requirements/stats-ctw-1.md"
	path2 := "lifecycle/requirements/stats-ctw-2.md"
	pathOld := "lifecycle/requirements/stats-ctw-old.md"

	thisWeekTS := weekStart.Add(1 * time.Hour).Unix()
	lastWeekTS := weekStart.AddDate(0, 0, -7).Unix()

	for _, e := range []*index.EventRow{
		{EventType: "status_transition", Timestamp: thisWeekTS, Actor: "test",
			ArtifactPath: &path1, Summary: `"Stats CTW 1" transitioned from approved → done`},
		{EventType: "status_transition", Timestamp: thisWeekTS, Actor: "test",
			ArtifactPath: &path2, Summary: `"Stats CTW 2" transitioned from approved → done`},
		// Old event — should NOT be counted.
		{EventType: "status_transition", Timestamp: lastWeekTS, Actor: "test",
			ArtifactPath: &pathOld, Summary: `"Stats CTW Old" transitioned from approved → done`},
	} {
		if err := env.proj.Idx.InsertEvent(e); err != nil {
			t.Fatalf("InsertEvent: %v", err)
		}
	}

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/stats", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	assertIntField(t, data, "completed_this_week", 2)
}

// TestDashboardStats_NonTicketsExcluded verifies that ideas and plan artifacts
// do not appear in any ticket count.
func TestDashboardStats_NonTicketsExcluded(t *testing.T) {
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/stats-idea-1.md",
			content: makeArtifact("Stats Idea 1", "idea", "draft", "stats-idea-1", "", "Body.")},
		{relPath: "lifecycle/backend-plans/stats-plan-1.md",
			content: makeArtifact("Stats Plan 1", "plan-backend", "approved", "stats-plan-1", "", "Body.")},
		{relPath: "lifecycle/frontend-plans/stats-plan-2.md",
			content: makeArtifact("Stats Plan 2", "plan-frontend", "draft", "stats-plan-2", "", "Body.")},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/stats", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	assertIntField(t, data, "total_tickets", 0)
	assertIntField(t, data, "in_progress", 0)
	assertIntField(t, data, "blocked", 0)
}

// TestDashboardStats_AbandonedExcluded verifies that abandoned tickets are not
// counted in total_tickets.
func TestDashboardStats_AbandonedExcluded(t *testing.T) {
	seeds := []seedArtifact{
		{relPath: "lifecycle/requirements/stats-abandoned-1.md",
			content: makeArtifact("Abandoned 1", "ticket", "abandoned", "stats-abandoned-1", "", "Body.")},
		{relPath: "lifecycle/requirements/stats-abandoned-2.md",
			content: makeArtifact("Abandoned 2", "ticket", "abandoned", "stats-abandoned-2", "", "Body.")},
		// One live ticket for contrast.
		{relPath: "lifecycle/requirements/stats-live-1.md",
			content: makeArtifact("Live 1", "ticket", "planning", "stats-live-1", "", "Body.")},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/stats", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	// Only the live ticket counts.
	assertIntField(t, data, "total_tickets", 1)
}

// TestDashboardStats_Performance verifies that the stats endpoint responds in
// under 100 ms when the index contains 500 seeded artifacts.
func TestDashboardStats_Performance(t *testing.T) {
	const n = 500
	seeds := make([]seedArtifact, 0, n)
	statuses := []string{"draft", "planning", "in-development", "blocked", "done", "abandoned"}
	types := []string{"ticket", "ticket", "ticket", "idea", "plan-backend"}

	for i := 0; i < n; i++ {
		typ := types[i%len(types)]
		status := statuses[i%len(statuses)]
		dir := "lifecycle/requirements"
		if typ == "idea" {
			dir = "lifecycle/ideas"
		} else if typ == "plan-backend" {
			dir = "lifecycle/backend-plans"
		}
		slug := fmt.Sprintf("stats-perf-%04d", i)
		seeds = append(seeds, seedArtifact{
			relPath: fmt.Sprintf("%s/%s.md", dir, slug),
			content: makeArtifact(fmt.Sprintf("Perf %d", i), typ, status, slug, "", "Body."),
		})
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	start := time.Now()
	resp := env.doRequest("GET", "/api/p/testproject/dashboard/stats", nil)
	elapsed := time.Since(start)

	requireStatus(t, resp, 200)
	if elapsed > 100*time.Millisecond {
		t.Errorf("stats response time %v exceeds 100ms limit with %d artifacts", elapsed, n)
	}
	t.Logf("stats endpoint with %d artifacts responded in %v", n, elapsed)
}

// ── shared dashboard helper ───────────────────────────────────────────────────

// assertIntField checks that a JSON response map contains the expected integer
// value for the named key. JSON numbers arrive as float64 from json.Unmarshal.
func assertIntField(t *testing.T, data map[string]any, key string, want int) {
	t.Helper()
	raw, ok := data[key]
	if !ok {
		t.Errorf("response missing field %q", key)
		return
	}
	got, ok := raw.(float64)
	if !ok {
		t.Errorf("field %q: expected number, got %T (%v)", key, raw, raw)
		return
	}
	if int(got) != want {
		t.Errorf("field %q: want %d, got %d", key, want, int(got))
	}
}
