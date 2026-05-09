// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Universal Text Filter — Milestone 1: Backend API integration tests.
//
// These tests verify that GET /api/p/:project/artifacts?q=<term> correctly
// filters artifacts by a free-text substring matched across title, slug,
// lineage, type, and status fields.  They complement the unit tests in
// internal/index/filter_test.go which verify the SQL-level buildWhere logic.
//
// Run with:
//   go test ./tests/integration/ -tags integration -run TestUniversalTextFilterAPI

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
)

// ── helpers ────────────────────────────────────────────────────────────────

// utf1Seeds returns a slice of seed artifacts with controlled titles, slugs,
// statuses, and types designed to exercise every Milestone 1 test case.
func utf1Seeds() []seedArtifact {
	return []seedArtifact{
		// Kanban-related — used by cases 1, 2, 3, 7
		{
			relPath: "lifecycle/ideas/kanban-view.md",
			content: makeArtifact("Kanban View", "idea", "draft", "kanban-view", "", "A kanban view idea."),
		},
		{
			relPath: "lifecycle/requirements/kanban-view-2.md",
			content: makeArtifact("Kanban View Spec", "ticket", "planning", "kanban-view", "lifecycle/ideas/kanban-view.md", "Spec body."),
		},
		// Non-kanban artifacts for contrast
		{
			relPath: "lifecycle/ideas/login-flow.md",
			content: makeArtifact("Login Flow", "idea", "draft", "login-flow", "", "Login idea."),
		},
		{
			relPath: "lifecycle/requirements/login-flow-2.md",
			content: makeArtifact("Login Flow Spec", "ticket", "in-development", "login-flow", "lifecycle/ideas/login-flow.md", "Spec body."),
		},
		// A ticket-type artifact for type-match test (case 5)
		{
			relPath: "lifecycle/requirements/search-filter-2.md",
			content: makeArtifact("Search Filter Spec", "ticket", "draft", "search-filter", "", "Spec body."),
		},
	}
}

// utf1ListAll returns all items from the artifacts endpoint for the test project.
func utf1ListAll(t *testing.T, env *testEnv, query string) (items []any, total int) {
	t.Helper()
	path := "/api/p/testproject/artifacts"
	if query != "" {
		path += "?" + query
	}
	resp := env.doRequest("GET", path, nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	items, _ = data["items"].([]any)
	f, _ := data["total"].(float64)
	total = int(f)
	return
}

// utf1ItemTitles extracts title strings from an items slice.
func utf1ItemTitles(items []any) []string {
	out := make([]string, 0, len(items))
	for _, raw := range items {
		m, _ := raw.(map[string]any)
		if t, _ := m["title"].(string); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// utf1HasField returns true if any item in items has the given field equal to value.
func utf1HasField(items []any, field, value string) bool {
	for _, raw := range items {
		m, _ := raw.(map[string]any)
		if v, _ := m[field].(string); v == value {
			return true
		}
	}
	return false
}

// ── Test cases ────────────────────────────────────────────────────────────

// TestUniversalTextFilterAPI_BasicSubstringMatch seeds artifacts with known
// titles and verifies that GET /artifacts?q=<substring> returns only artifacts
// whose title (or other indexed field) contains the substring.
// Test plan Milestone 1, case 1.
func TestUniversalTextFilterAPI_BasicSubstringMatch(t *testing.T) {
	env := newTestEnv(t, utf1Seeds())
	env.login("admin@test.local", "admin-pass-123")

	items, total := utf1ListAll(t, env, "q=kanban")

	if total == 0 {
		t.Fatal("expected at least one match for q=kanban, got 0")
	}
	if len(items) != total {
		t.Errorf("total=%d does not match len(items)=%d", total, len(items))
	}
	for _, raw := range items {
		m, _ := raw.(map[string]any)
		title, _ := m["title"].(string)
		slug, _ := m["slug"].(string)
		lineage, _ := m["lineage"].(string)
		typ, _ := m["type"].(string)
		status, _ := m["status"].(string)
		combined := strings.ToLower(title + " " + slug + " " + lineage + " " + typ + " " + status)
		if !strings.Contains(combined, "kanban") {
			t.Errorf("result %q does not match q=kanban in any indexed field", title)
		}
	}
	// The two non-kanban artifacts (login-flow) must not appear.
	for _, title := range utf1ItemTitles(items) {
		if strings.Contains(strings.ToLower(title), "login") {
			t.Errorf("non-matching artifact %q appeared in results for q=kanban", title)
		}
	}
}

// TestUniversalTextFilterAPI_CaseInsensitivity verifies that q matching is
// case-insensitive (q=kanban matches "Kanban View").
// Test plan Milestone 1, case 2.
func TestUniversalTextFilterAPI_CaseInsensitivity(t *testing.T) {
	env := newTestEnv(t, utf1Seeds())
	env.login("admin@test.local", "admin-pass-123")

	items, total := utf1ListAll(t, env, "q=KANBAN")
	if total == 0 {
		t.Fatal("expected at least one match for q=KANBAN (case-insensitive), got 0")
	}
	if !utf1HasField(items, "title", "Kanban View") {
		t.Errorf("'Kanban View' not found in results for q=KANBAN")
	}
}

// TestUniversalTextFilterAPI_MatchesOnSlug verifies that q matches against the
// slug field (e.g. q=kanban-view returns artifacts with slug "kanban-view").
// Test plan Milestone 1, case 3.
func TestUniversalTextFilterAPI_MatchesOnSlug(t *testing.T) {
	env := newTestEnv(t, utf1Seeds())
	env.login("admin@test.local", "admin-pass-123")

	items, total := utf1ListAll(t, env, "q=kanban-view")
	if total == 0 {
		t.Fatal("expected at least one match for q=kanban-view, got 0")
	}
	found := false
	for _, raw := range items {
		m, _ := raw.(map[string]any)
		if slug, _ := m["slug"].(string); slug == "kanban-view" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("artifact with slug=kanban-view not found in results for q=kanban-view")
	}
}

// TestUniversalTextFilterAPI_MatchesOnLineage verifies that q matches the
// lineage field and returns all artifacts in that lineage.
// Test plan Milestone 1, case 4.
func TestUniversalTextFilterAPI_MatchesOnLineage(t *testing.T) {
	env := newTestEnv(t, utf1Seeds())
	env.login("admin@test.local", "admin-pass-123")

	items, total := utf1ListAll(t, env, "q=login-flow")
	if total == 0 {
		t.Fatal("expected at least one match for q=login-flow, got 0")
	}
	// Both the idea and its spec share lineage "login-flow".
	for _, raw := range items {
		m, _ := raw.(map[string]any)
		lineage, _ := m["lineage"].(string)
		if lineage != "login-flow" {
			t.Errorf("result lineage=%q does not match q=login-flow", lineage)
		}
	}
}

// TestUniversalTextFilterAPI_MatchesOnType verifies that q matches the type
// field (e.g. q=ticket returns ticket-type artifacts).
// Test plan Milestone 1, case 5.
func TestUniversalTextFilterAPI_MatchesOnType(t *testing.T) {
	// Seed one idea and two tickets so the distinction is clear.
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/alpha.md",
			content: makeArtifact("Alpha Idea", "idea", "draft", "alpha", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/beta-2.md",
			content: makeArtifact("Beta Spec", "ticket", "draft", "beta", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/gamma-2.md",
			content: makeArtifact("Gamma Spec", "ticket", "planning", "gamma", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	items, total := utf1ListAll(t, env, "q=ticket")
	if total == 0 {
		t.Fatal("expected at least one match for q=ticket, got 0")
	}
	for _, raw := range items {
		m, _ := raw.(map[string]any)
		typ, _ := m["type"].(string)
		if typ != "ticket" {
			t.Errorf("result type=%q does not match q=ticket filter", typ)
		}
	}
}

// TestUniversalTextFilterAPI_MatchesOnStatus verifies that q matches the
// status field (e.g. q=draft returns only draft artifacts).
// Test plan Milestone 1, case 6.
func TestUniversalTextFilterAPI_MatchesOnStatus(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/art-draft.md",
			content: makeArtifact("Draft Artifact", "idea", "draft", "art-draft", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/art-planning.md",
			content: makeArtifact("Planning Artifact", "idea", "planning", "art-planning", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	items, total := utf1ListAll(t, env, "q=draft")
	if total == 0 {
		t.Fatal("expected at least one match for q=draft, got 0")
	}
	for _, raw := range items {
		m, _ := raw.(map[string]any)
		// The query matches against all fields; only verify none of the returned
		// items have status=planning (which contains no "draft" substring).
		status, _ := m["status"].(string)
		title, _ := m["title"].(string)
		slug, _ := m["slug"].(string)
		lineage, _ := m["lineage"].(string)
		combined := strings.ToLower(title + " " + slug + " " + lineage + " " + status)
		if !strings.Contains(combined, "draft") {
			t.Errorf("result %q does not contain 'draft' in any indexed field", title)
		}
	}
}

// TestUniversalTextFilterAPI_CompositionWithDropdownFilters verifies that q
// composes with the status filter using AND logic.
// Test plan Milestone 1, case 7.
func TestUniversalTextFilterAPI_CompositionWithDropdownFilters(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/kanban-draft.md",
			content: makeArtifact("Kanban Draft Idea", "idea", "draft", "kanban-draft", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/kanban-planning-2.md",
			content: makeArtifact("Kanban Planning Spec", "ticket", "planning", "kanban-planning", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/other-draft.md",
			content: makeArtifact("Other Draft Idea", "idea", "draft", "other-draft", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	items, total := utf1ListAll(t, env, "q=kanban&status=draft")
	if total == 0 {
		t.Fatal("expected at least one match for q=kanban&status=draft, got 0")
	}
	for _, raw := range items {
		m, _ := raw.(map[string]any)
		title, _ := m["title"].(string)
		status, _ := m["status"].(string)
		slug, _ := m["slug"].(string)
		lineage, _ := m["lineage"].(string)
		typ, _ := m["type"].(string)
		combined := strings.ToLower(title + " " + slug + " " + lineage + " " + typ + " " + status)

		if status != "draft" {
			t.Errorf("result %q has status=%q but status=draft filter was applied", title, status)
		}
		if !strings.Contains(combined, "kanban") {
			t.Errorf("result %q does not contain 'kanban' in any indexed field", title)
		}
	}
	// "Kanban Planning Spec" must NOT appear (status != draft).
	for _, title := range utf1ItemTitles(items) {
		if strings.Contains(title, "Planning") {
			t.Errorf("'%s' should not appear (wrong status) in q=kanban&status=draft results", title)
		}
	}
}

// TestUniversalTextFilterAPI_NoMatches verifies that a query matching no
// artifacts returns an empty items array and total=0.
// Test plan Milestone 1, case 8.
func TestUniversalTextFilterAPI_NoMatches(t *testing.T) {
	env := newTestEnv(t, utf1Seeds())
	env.login("admin@test.local", "admin-pass-123")

	items, total := utf1ListAll(t, env, "q=zzz_nonexistent_zzz")
	if total != 0 {
		t.Errorf("expected total=0 for non-matching query, got %d", total)
	}
	if len(items) != 0 {
		t.Errorf("expected empty items for non-matching query, got %d items", len(items))
	}
}

// TestUniversalTextFilterAPI_EmptyQ verifies that an empty q parameter returns
// all artifacts (same behaviour as omitting q entirely).
// Test plan Milestone 1, case 9.
func TestUniversalTextFilterAPI_EmptyQ(t *testing.T) {
	env := newTestEnv(t, utf1Seeds())
	env.login("admin@test.local", "admin-pass-123")

	_, totalNoQ := utf1ListAll(t, env, "")
	_, totalEmptyQ := utf1ListAll(t, env, "q=")

	if totalEmptyQ != totalNoQ {
		t.Errorf("empty q returned total=%d but no-q returned total=%d; they must be equal",
			totalEmptyQ, totalNoQ)
	}
}

// TestUniversalTextFilterAPI_SpecialCharacters verifies that LIKE-wildcard
// characters (% and _) in q are escaped and treated as literals, preventing
// SQL injection and false-positive matches.
// Test plan Milestone 1, case 10.
func TestUniversalTextFilterAPI_SpecialCharacters(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/pct-idea.md",
			content: makeArtifact("100% Complete", "idea", "draft", "pct-idea", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/plain-idea.md",
			content: makeArtifact("Plain Idea", "idea", "draft", "plain-idea", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// q=100% URL-encoded as 100%25.
	encodedQ := url.QueryEscape("100%")
	items, total := utf1ListAll(t, env, "q="+encodedQ)

	// "100%" only appears literally in the title "100% Complete".
	if total > 1 {
		t.Errorf("q=100%% matched %d items; expected exactly 1 (literal match only)", total)
	}
	if total == 1 {
		m, _ := items[0].(map[string]any)
		title, _ := m["title"].(string)
		if title != "100% Complete" {
			t.Errorf("q=100%% returned %q, want %q", title, "100% Complete")
		}
	}

	// q=_idea should match only artifacts whose indexed fields contain the literal
	// underscore, not act as a wildcard.
	encodedQ2 := url.QueryEscape("_idea")
	items2, total2 := utf1ListAll(t, env, "q="+encodedQ2)
	// Neither seeded artifact has "_idea" literally in any indexed field.
	if total2 != 0 {
		titles := make([]string, 0, len(items2))
		for _, raw := range items2 {
			m, _ := raw.(map[string]any)
			titles = append(titles, fmt.Sprintf("%v", m["title"]))
		}
		t.Errorf("q=_idea (literal underscore) returned %d items; expected 0; got: %v", total2, titles)
	}
}

// TestUniversalTextFilterAPI_PaginationReset seeds more than one page of
// kanban artifacts and verifies that q=kanban&offset=0 starts from the first
// result regardless of prior navigation.
// Test plan Milestone 1, case 11.
func TestUniversalTextFilterAPI_PaginationReset(t *testing.T) {
	// Seed 6 kanban artifacts — enough to span two pages at default limit.
	seeds := make([]seedArtifact, 0, 8)
	for i := 1; i <= 6; i++ {
		slug := fmt.Sprintf("kanban-pg-%d", i)
		seeds = append(seeds, seedArtifact{
			relPath: fmt.Sprintf("lifecycle/ideas/%s.md", slug),
			content: makeArtifact(
				fmt.Sprintf("Kanban Page Item %d", i),
				"idea", "draft", slug, "", "Body.",
			),
		})
	}
	// Add a non-kanban artifact to confirm filtering still works.
	seeds = append(seeds, seedArtifact{
		relPath: "lifecycle/ideas/other-item.md",
		content: makeArtifact("Other Item", "idea", "draft", "other-item", "", "Body."),
	})

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Fetch page 1 with offset=0, limit=3.
	resp1 := env.doRequest("GET", "/api/p/testproject/artifacts?q=kanban&limit=3&offset=0", nil)
	requireStatus(t, resp1, 200)
	data1 := readJSON(t, resp1)
	items1, _ := data1["items"].([]any)

	if len(items1) != 3 {
		t.Errorf("page 1 (limit=3, offset=0) returned %d items, want 3", len(items1))
	}

	// Re-send offset=0 — must start from the beginning, not a later page.
	resp2 := env.doRequest("GET", "/api/p/testproject/artifacts?q=kanban&limit=3&offset=0", nil)
	requireStatus(t, resp2, 200)
	data2 := readJSON(t, resp2)
	items2, _ := data2["items"].([]any)

	if len(items2) != len(items1) {
		t.Errorf("re-requesting offset=0 returned %d items, want %d", len(items2), len(items1))
	}
	// First item should be the same in both responses.
	if len(items1) > 0 && len(items2) > 0 {
		path1, _ := (items1[0].(map[string]any))["path"].(string)
		path2, _ := (items2[0].(map[string]any))["path"].(string)
		if path1 != path2 {
			t.Errorf("offset=0 reset did not return same first item: %q vs %q", path1, path2)
		}
	}

	// Fetch page 2 and confirm it does not overlap with page 1.
	resp3 := env.doRequest("GET", "/api/p/testproject/artifacts?q=kanban&limit=3&offset=3", nil)
	requireStatus(t, resp3, 200)
	data3 := readJSON(t, resp3)
	items3, _ := data3["items"].([]any)

	page1Paths := map[string]bool{}
	for _, raw := range items1 {
		m, _ := raw.(map[string]any)
		page1Paths[m["path"].(string)] = true
	}
	for _, raw := range items3 {
		m, _ := raw.(map[string]any)
		p, _ := m["path"].(string)
		if page1Paths[p] {
			t.Errorf("page 2 item %q also appeared on page 1 — pagination overlap", p)
		}
	}
}
