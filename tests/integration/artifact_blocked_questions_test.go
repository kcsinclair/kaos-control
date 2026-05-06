//go:build integration

package integration

import (
	"testing"
)

// TestBlockedQuestions_WithOpenQuestionsTriggersBlocked verifies that saving an
// artifact whose body contains a non-empty "## Open Questions" section causes
// the backend to override the submitted status to "blocked" and inject a
// product-owner assignee.
//
// Run with: go test ./tests/integration/... -tags=integration -run TestBlockedQuestions
func TestBlockedQuestions_WithOpenQuestionsTriggersBlocked(t *testing.T) {
	const relPath = "lifecycle/ideas/blocked-q-trigger.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			content: makeArtifact("Blocked Trigger", "idea", "draft", "blocked-q-trigger", "", "Initial body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPath, map[string]any{
		"frontmatter": map[string]any{
			"title":   "Blocked Trigger",
			"type":    "idea",
			"status":  "draft",
			"lineage": "blocked-q-trigger",
		},
		"body": "## Open Questions\n\n- Why is X?\n",
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	art, _ := data["artifact"].(map[string]any)
	fm, _ := art["frontmatter"].(map[string]any)

	if status, _ := fm["status"].(string); status != "blocked" {
		t.Errorf("expected status %q, got %q", "blocked", status)
	}

	assignees, _ := fm["assignees"].([]any)
	if len(assignees) == 0 {
		t.Fatal("expected at least one assignee, got none")
	}
	found := false
	for _, a := range assignees {
		entry, _ := a.(map[string]any)
		if entry["role"] == "product-owner" && entry["who"] == "agent" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected assignee {role: product-owner, who: agent} in response, got: %v", assignees)
	}
}

// TestBlockedQuestions_WithoutOpenQuestionsPreservesStatus verifies that saving
// an artifact whose body has no "## Open Questions" section preserves the
// submitted status unchanged.
func TestBlockedQuestions_WithoutOpenQuestionsPreservesStatus(t *testing.T) {
	const relPath = "lifecycle/ideas/blocked-q-no-oq.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			content: makeArtifact("No OQ", "idea", "draft", "blocked-q-no-oq", "", "Just a body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPath, map[string]any{
		"frontmatter": map[string]any{
			"title":   "No OQ",
			"type":    "idea",
			"status":  "draft",
			"lineage": "blocked-q-no-oq",
		},
		"body": "Just a body with no open questions heading.\n",
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	art, _ := data["artifact"].(map[string]any)
	fm, _ := art["frontmatter"].(map[string]any)

	if status, _ := fm["status"].(string); status != "draft" {
		t.Errorf("expected status %q (unchanged), got %q", "draft", status)
	}
}

// TestBlockedQuestions_EmptySectionDoesNotBlock verifies that a "## Open
// Questions" heading with only whitespace/blank lines below it (no actual
// content) does NOT trigger the auto-block logic.
func TestBlockedQuestions_EmptySectionDoesNotBlock(t *testing.T) {
	const relPath = "lifecycle/ideas/blocked-q-empty-section.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			content: makeArtifact("Empty OQ Section", "idea", "draft", "blocked-q-empty-section", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPath, map[string]any{
		"frontmatter": map[string]any{
			"title":   "Empty OQ Section",
			"type":    "idea",
			"status":  "draft",
			"lineage": "blocked-q-empty-section",
		},
		// Heading present but section body is blank.
		"body": "## Open Questions\n\n   \n\n",
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	art, _ := data["artifact"].(map[string]any)
	fm, _ := art["frontmatter"].(map[string]any)

	if status, _ := fm["status"].(string); status != "draft" {
		t.Errorf("empty OQ section must NOT trigger block: expected %q, got %q", "draft", status)
	}
}

// TestBlockedQuestions_BlockedWithNoOQAutoUnblocks verifies the new
// indexer-level auto-unblock behaviour: a "blocked" artifact whose body has
// NO open questions is automatically transitioned back to "draft" by the
// indexer whenever IndexFile fires (startup Scan, watcher, or PUT handler).
//
// Previously (handler-level auto-block) a blocked artifact without OQ would
// stay blocked. With the indexer-level approach the auto-unblock fires
// synchronously inside IndexFile, so both the startup Scan and the PUT
// handler return "draft" for such an artifact.
func TestBlockedQuestions_BlockedWithNoOQAutoUnblocks(t *testing.T) {
	const relPath = "lifecycle/ideas/blocked-q-auto-unblocks.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			// Seeded as "blocked" with no OQ: startup Scan auto-unblocks it to "draft".
			content: makeArtifact("Auto Unblocks", "idea", "blocked", "blocked-q-auto-unblocks", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// The startup Scan fires applyOpenQuestionTransition on the seed (blocked + no OQ)
	// and auto-unblocks it to draft before the server accepts requests.
	getResp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
	requireStatus(t, getResp, 200)
	getData := readJSON(t, getResp)
	getArt, _ := getData["artifact"].(map[string]any)
	getFM, _ := getArt["frontmatter"].(map[string]any)
	if status, _ := getFM["status"].(string); status != "draft" {
		t.Errorf("startup Scan: expected auto-unblock to 'draft', got %q", status)
	}

	// PUT with status=blocked and no OQ: IndexFile fires again → auto-unblock → draft.
	resp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPath, map[string]any{
		"frontmatter": map[string]any{
			"title":   "Auto Unblocks",
			"type":    "idea",
			"status":  "blocked",
			"lineage": "blocked-q-auto-unblocks",
		},
		"body": "Body without any open questions.\n",
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	art, _ := data["artifact"].(map[string]any)
	fm, _ := art["frontmatter"].(map[string]any)

	if status, _ := fm["status"].(string); status != "draft" {
		t.Errorf("PUT handler: expected auto-unblock to 'draft', got %q", status)
	}
}

// ── Milestone 6 additions ─────────────────────────────────────────────────────

// TestBlockedQuestions_PutOQThenRemoveOQ verifies the full auto-block →
// auto-unblock cycle via sequential PUT requests:
//  1. PUT with "## Open Questions" body → response shows "blocked".
//  2. PUT the same artifact (now blocked on disk) with the questions removed
//     → response shows "draft" (auto-unblock fires via indexer).
//
// Covers test plan Milestone 6, cases 1 & 2.
func TestBlockedQuestions_PutOQThenRemoveOQ(t *testing.T) {
	const relPath = "lifecycle/ideas/blocked-q-cycle.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			content: makeArtifact("Cycle Test", "idea", "draft", "blocked-q-cycle", "", "Initial body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// ── PUT 1: add open questions → should auto-block ─────────────────────
	resp1 := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPath, map[string]any{
		"frontmatter": map[string]any{
			"title":   "Cycle Test",
			"type":    "idea",
			"status":  "draft",
			"lineage": "blocked-q-cycle",
		},
		"body": "## Open Questions\n\n- What should we do?\n",
	})
	requireStatus(t, resp1, 200)
	data1 := readJSON(t, resp1)

	art1, _ := data1["artifact"].(map[string]any)
	fm1, _ := art1["frontmatter"].(map[string]any)
	if status, _ := fm1["status"].(string); status != "blocked" {
		t.Fatalf("PUT 1: expected 'blocked', got %q", status)
	}

	// ── PUT 2: remove open questions → should auto-unblock ────────────────
	resp2 := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPath, map[string]any{
		"frontmatter": map[string]any{
			"title":   "Cycle Test",
			"type":    "idea",
			"status":  "blocked",
			"lineage": "blocked-q-cycle",
		},
		"body": "Questions have been resolved.\n",
	})
	requireStatus(t, resp2, 200)
	data2 := readJSON(t, resp2)

	art2, _ := data2["artifact"].(map[string]any)
	fm2, _ := art2["frontmatter"].(map[string]any)
	if status, _ := fm2["status"].(string); status != "draft" {
		t.Errorf("PUT 2: expected auto-unblock to 'draft', got %q", status)
	}
}

// TestBlockedQuestions_NoDuplicateEventsOnPut verifies that a single PUT
// that triggers auto-block inserts exactly one "status_changed" event into
// the feed for the artifact path. No duplicate events should be produced by
// a combination of handler logic and indexer logic.
//
// Covers test plan Milestone 6, case 3.
func TestBlockedQuestions_NoDuplicateEventsOnPut(t *testing.T) {
	const relPath = "lifecycle/ideas/blocked-q-dedup-events.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			content: makeArtifact("Dedup Events", "idea", "draft", "blocked-q-dedup-events", "", "Initial body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Capture feed event count before PUT.
	feedBefore := env.doRequest("GET", "/api/p/testproject/feed?limit=200", nil)
	requireStatus(t, feedBefore, 200)
	beforeData := readJSON(t, feedBefore)
	eventsBefore, _ := beforeData["events"].([]any)
	countBefore := countStatusChangedForPath(t, eventsBefore, relPath)

	// PUT with open questions → auto-block.
	resp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPath, map[string]any{
		"frontmatter": map[string]any{
			"title":   "Dedup Events",
			"type":    "idea",
			"status":  "draft",
			"lineage": "blocked-q-dedup-events",
		},
		"body": "## Open Questions\n\n- Is there a duplicate event?\n",
	})
	requireStatus(t, resp, 200)
	resp.Body.Close()

	// Capture feed event count after PUT.
	feedAfter := env.doRequest("GET", "/api/p/testproject/feed?limit=200", nil)
	requireStatus(t, feedAfter, 200)
	afterData := readJSON(t, feedAfter)
	eventsAfter, _ := afterData["events"].([]any)
	countAfter := countStatusChangedForPath(t, eventsAfter, relPath)

	newEvents := countAfter - countBefore
	if newEvents != 1 {
		t.Errorf("expected exactly 1 new status_changed event for a single auto-block PUT, got %d", newEvents)
	}
}

// countStatusChangedForPath counts "status_changed" events in events whose
// artifact_path matches relPath.
func countStatusChangedForPath(t *testing.T, events []any, relPath string) int {
	t.Helper()
	n := 0
	for _, ev := range events {
		entry, _ := ev.(map[string]any)
		if entry["event_type"] == "status_changed" {
			if p, _ := entry["artifact_path"].(string); p == relPath {
				n++
			}
		}
	}
	return n
}

// TestBlockedQuestions_ProductOwnerAssigneeNotDuplicated verifies that when a
// product-owner/agent assignee already exists in the submitted frontmatter and
// the body contains open questions, the auto-block logic does not add a second
// product-owner entry.
func TestBlockedQuestions_ProductOwnerAssigneeNotDuplicated(t *testing.T) {
	const relPath = "lifecycle/ideas/blocked-q-no-dup.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			content: makeArtifactWithAssignees(
				"No Dup PO", "idea", "draft", "blocked-q-no-dup",
				[]map[string]string{{"role": "product-owner", "who": "agent"}},
				"Initial body.",
			),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// PUT: body has open questions AND front matter already carries product-owner/agent.
	resp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPath, map[string]any{
		"frontmatter": map[string]any{
			"title":   "No Dup PO",
			"type":    "idea",
			"status":  "draft",
			"lineage": "blocked-q-no-dup",
			"assignees": []map[string]string{
				{"role": "product-owner", "who": "agent"},
			},
		},
		"body": "## Open Questions\n\n- Should we do X?\n",
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	art, _ := data["artifact"].(map[string]any)
	fm, _ := art["frontmatter"].(map[string]any)
	assignees, _ := fm["assignees"].([]any)

	poCount := 0
	for _, a := range assignees {
		entry, _ := a.(map[string]any)
		if entry["role"] == "product-owner" && entry["who"] == "agent" {
			poCount++
		}
	}
	if poCount != 1 {
		t.Errorf("expected exactly 1 product-owner/agent assignee, got %d (full list: %v)", poCount, assignees)
	}
}
