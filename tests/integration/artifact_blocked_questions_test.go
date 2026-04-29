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

// TestBlockedQuestions_ExistingBlockedStatusPreserved verifies that saving an
// artifact that is already "blocked" with a body that has no open questions
// preserves the submitted "blocked" status. The backend must not auto-unblock.
func TestBlockedQuestions_ExistingBlockedStatusPreserved(t *testing.T) {
	const relPath = "lifecycle/ideas/blocked-q-stays-blocked.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			content: makeArtifact("Stays Blocked", "idea", "blocked", "blocked-q-stays-blocked", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// PUT: no open questions, but status is explicitly "blocked" — should stay blocked.
	resp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPath, map[string]any{
		"frontmatter": map[string]any{
			"title":   "Stays Blocked",
			"type":    "idea",
			"status":  "blocked",
			"lineage": "blocked-q-stays-blocked",
		},
		"body": "Body without any open questions.\n",
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	art, _ := data["artifact"].(map[string]any)
	fm, _ := art["frontmatter"].(map[string]any)

	if status, _ := fm["status"].(string); status != "blocked" {
		t.Errorf("expected status %q to be preserved, got %q", "blocked", status)
	}
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
