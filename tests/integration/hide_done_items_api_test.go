// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Tests for the "Hide Done Items by Default" feature — backend API layer.
//
// The "Show completed" toggle is entirely client-side: the backend always
// returns all artifacts regardless of status.  These tests verify that:
//
//  1. The API returns terminal-status artifacts (done/rejected/abandoned) in
//     the normal list response — without server-side suppression.
//  2. Each artifact row carries a `status` field that the frontend can use to
//     filter client-side.
//  3. The graph API includes a `status` field on every node so the graph
//     store can apply the hideTerminal filter.
//  4. Terminal-status artifacts can be retrieved individually and as a group
//     via the ?status= query parameter.
//  5. A single API call with no filters returns all statuses (active + terminal).
//
// These tests correspond to Milestone 1 (seed data helpers / data availability)
// of the test plan for lifecycle/test-plans/hide-done-items-by-default-5-test.md.

import (
	"fmt"
	"testing"
)

// terminalStatuses are the three statuses the frontend hides by default.
var terminalStatuses = []string{"done", "rejected", "abandoned"}

// activeStatuses are the non-terminal workflow statuses.
var activeStatuses = []string{"draft", "clarifying", "planning", "in-development", "in-qa"}

// allWorkflowStatuses is the union of active and terminal statuses.
var allWorkflowStatuses = append(activeStatuses, terminalStatuses...)

// hideDoneSeeds returns a slice of seedArtifact with one artifact per status.
// Titles deliberately avoid colons to keep the YAML valid (colons followed
// by a space in unquoted block scalars are interpreted as mapping keys).
func hideDoneSeeds() []seedArtifact {
	seeds := make([]seedArtifact, 0, len(allWorkflowStatuses))
	for i, status := range allWorkflowStatuses {
		slug := fmt.Sprintf("hd-%s", status)
		// Hyphen avoids the YAML colon issue while remaining human-readable.
		title := fmt.Sprintf("HideDone-%s", status)
		seeds = append(seeds, seedArtifact{
			relPath: fmt.Sprintf("lifecycle/ideas/%s-%d.md", slug, i+1),
			content: makeArtifact(
				title,
				"idea",
				status,
				slug,
				"",
				"Test artifact for hide-done-items feature.",
			),
		})
	}
	return seeds
}

// ── Milestone 1: Seed Data Helpers / Data Availability ──────────────────────

// TestHideDoneItems_APIReturnsAllStatuses verifies that GET /artifacts with no
// filters returns artifacts for all statuses, including all three terminal
// statuses, in a single response.
func TestHideDoneItems_APIReturnsAllStatuses(t *testing.T) {
	env := newTestEnv(t, hideDoneSeeds())
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?limit=100", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	total, _ := data["total"].(float64)

	if int(total) != len(allWorkflowStatuses) {
		t.Fatalf("expected total=%d (one per status), got %d", len(allWorkflowStatuses), int(total))
	}
	if len(items) != len(allWorkflowStatuses) {
		t.Fatalf("expected %d items, got %d", len(allWorkflowStatuses), len(items))
	}

	// Build a set of returned statuses.
	returned := map[string]bool{}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if s, _ := item["status"].(string); s != "" {
			returned[s] = true
		}
	}

	for _, want := range allWorkflowStatuses {
		if !returned[want] {
			t.Errorf("status %q not found in API response — backend may be filtering it out", want)
		}
	}
}

// TestHideDoneItems_APIReturnsTerminalArtifactsUnfiltered verifies that a call
// to GET /artifacts (no status filter) includes all three terminal-status
// artifacts.  This confirms the backend does not hide completed items.
func TestHideDoneItems_APIReturnsTerminalArtifactsUnfiltered(t *testing.T) {
	env := newTestEnv(t, hideDoneSeeds())
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?limit=100", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)

	terminalFound := map[string]bool{}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		status, _ := item["status"].(string)
		for _, t := range terminalStatuses {
			if status == t {
				terminalFound[t] = true
			}
		}
	}

	for _, ts := range terminalStatuses {
		if !terminalFound[ts] {
			t.Errorf("terminal status %q not returned by unfiltered /artifacts — backend must not suppress it", ts)
		}
	}
}

// TestHideDoneItems_EachItemHasStatusField verifies that every artifact row
// returned by GET /artifacts has a non-empty `status` field.  The frontend
// relies on this field to decide whether to show or hide the item.
func TestHideDoneItems_EachItemHasStatusField(t *testing.T) {
	env := newTestEnv(t, hideDoneSeeds())
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?limit=100", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	if len(items) == 0 {
		t.Fatal("expected artifacts in list response")
	}

	for _, raw := range items {
		item, _ := raw.(map[string]any)
		path, _ := item["path"].(string)
		status, _ := item["status"].(string)
		if status == "" {
			t.Errorf("artifact %q has empty or missing status field", path)
		}
	}
}

// TestHideDoneItems_FilterByTerminalStatus verifies that the backend supports
// querying artifacts by each terminal status individually.
func TestHideDoneItems_FilterByTerminalStatus(t *testing.T) {
	env := newTestEnv(t, hideDoneSeeds())
	env.login("admin@test.local", "admin-pass-123")

	for _, ts := range terminalStatuses {
		t.Run(ts, func(t *testing.T) {
			resp := env.doRequest("GET", "/api/p/testproject/artifacts?status="+ts, nil)
			requireStatus(t, resp, 200)
			data := readJSON(t, resp)

			items, _ := data["items"].([]any)
			total, _ := data["total"].(float64)

			if int(total) != 1 {
				t.Errorf("status=%s: expected total=1, got %d", ts, int(total))
			}
			if len(items) != 1 {
				t.Errorf("status=%s: expected 1 item, got %d", ts, len(items))
			}
			if len(items) == 1 {
				item, _ := items[0].(map[string]any)
				got, _ := item["status"].(string)
				if got != ts {
					t.Errorf("status=%s: returned item has status %q", ts, got)
				}
			}
		})
	}
}

// TestHideDoneItems_FilterByActiveStatus verifies the backend supports querying
// by each active (non-terminal) status.
func TestHideDoneItems_FilterByActiveStatus(t *testing.T) {
	env := newTestEnv(t, hideDoneSeeds())
	env.login("admin@test.local", "admin-pass-123")

	for _, as_ := range activeStatuses {
		t.Run(as_, func(t *testing.T) {
			resp := env.doRequest("GET", "/api/p/testproject/artifacts?status="+as_, nil)
			requireStatus(t, resp, 200)
			data := readJSON(t, resp)

			items, _ := data["items"].([]any)
			total, _ := data["total"].(float64)

			if int(total) != 1 {
				t.Errorf("status=%s: expected total=1, got %d", as_, int(total))
			}
			if len(items) > 0 {
				item, _ := items[0].(map[string]any)
				got, _ := item["status"].(string)
				if got != as_ {
					t.Errorf("status=%s: returned item has status %q", as_, got)
				}
			}
		})
	}
}

// TestHideDoneItems_GraphNodesHaveStatusField verifies that every node in the
// graph API response includes a non-empty `status` field.  The graph store's
// hideTerminal computed uses this to filter nodes client-side.
func TestHideDoneItems_GraphNodesHaveStatusField(t *testing.T) {
	env := newTestEnv(t, hideDoneSeeds())

	data := graphResponseForProject(t, env)
	nodes := decodeGraphNodes(t, data)

	if len(nodes) == 0 {
		t.Fatal("expected nodes in graph response")
	}

	for _, raw := range nodes {
		node, _ := raw.(map[string]any)
		id, _ := node["id"].(string)
		status, _ := node["status"].(string)
		if status == "" {
			t.Errorf("graph node %q has empty or missing status field", id)
		}
	}
}

// TestHideDoneItems_GraphIncludesTerminalStatusNodes verifies that the graph
// API returns terminal-status nodes — no server-side suppression.
func TestHideDoneItems_GraphIncludesTerminalStatusNodes(t *testing.T) {
	env := newTestEnv(t, hideDoneSeeds())

	data := graphResponseForProject(t, env)
	nodes := decodeGraphNodes(t, data)

	terminalFound := map[string]bool{}
	for _, raw := range nodes {
		node, _ := raw.(map[string]any)
		status, _ := node["status"].(string)
		for _, ts := range terminalStatuses {
			if status == ts {
				terminalFound[ts] = true
			}
		}
	}

	for _, ts := range terminalStatuses {
		if !terminalFound[ts] {
			t.Errorf("graph API missing node with terminal status %q — backend must not suppress it", ts)
		}
	}
}

// TestHideDoneItems_GraphNodeCount verifies that the graph returns exactly one
// node per seeded artifact (all statuses, no server-side filtering).
func TestHideDoneItems_GraphNodeCount(t *testing.T) {
	env := newTestEnv(t, hideDoneSeeds())

	data := graphResponseForProject(t, env)
	nodes := decodeGraphNodes(t, data)

	if len(nodes) != len(allWorkflowStatuses) {
		t.Errorf("graph node count: want %d (one per status), got %d", len(allWorkflowStatuses), len(nodes))
	}
}

// TestHideDoneItems_TerminalArtifactsRetrievableIndividually verifies that each
// terminal-status artifact can be fetched via GET /artifacts/<path>.
func TestHideDoneItems_TerminalArtifactsRetrievableIndividually(t *testing.T) {
	seeds := hideDoneSeeds()
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Collect the paths of seeded terminal-status artifacts.
	terminalPaths := []string{}
	for _, s := range seeds {
		for _, ts := range terminalStatuses {
			if contains(s.content, "status: "+ts) {
				terminalPaths = append(terminalPaths, s.relPath)
			}
		}
	}

	if len(terminalPaths) != len(terminalStatuses) {
		t.Fatalf("expected %d terminal seed artifacts, found %d", len(terminalStatuses), len(terminalPaths))
	}

	for _, path := range terminalPaths {
		resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+path, nil)
		requireStatus(t, resp, 200)
		data := readJSON(t, resp)
		artifact, ok := data["artifact"].(map[string]any)
		if !ok {
			t.Fatalf("GET /artifacts/%s: no 'artifact' field in response", path)
		}
		status, _ := artifact["status"].(string)
		if status == "" {
			t.Errorf("GET /artifacts/%s: artifact has empty status", path)
		}
	}
}

// TestHideDoneItems_ActiveArtifactCountIsConsistent verifies that filtering by
// active statuses cumulatively yields exactly the number of active artifacts.
func TestHideDoneItems_ActiveArtifactCountIsConsistent(t *testing.T) {
	env := newTestEnv(t, hideDoneSeeds())
	env.login("admin@test.local", "admin-pass-123")

	total := 0
	for _, as_ := range activeStatuses {
		resp := env.doRequest("GET", "/api/p/testproject/artifacts?status="+as_, nil)
		requireStatus(t, resp, 200)
		data := readJSON(t, resp)
		n, _ := data["total"].(float64)
		total += int(n)
	}

	if total != len(activeStatuses) {
		t.Errorf("sum of per-active-status counts: want %d, got %d", len(activeStatuses), total)
	}
}

// contains is a simple string-substring check used by tests above.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}()
}
