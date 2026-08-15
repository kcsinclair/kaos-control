// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"testing"
)

// Milestone 3 of requirements-analyst-suppress-empty-open-questions-5-test.md:
// end-to-end index-outcome coverage for applyOpenQuestionTransition
// (internal/index/autoblock.go), exercised via the startup scan that indexes
// seeded artifacts rather than via the PUT/preview endpoints already covered
// by TestOpenQuestionsResolve_EndToEnd.

// TestOpenQuestionsIndexOutcome_NoSectionStaysNonBlocking verifies that
// indexing a requirement body with no "## Open Questions" heading at all
// leaves its authored status untouched and does not auto-block it
// (requirement Acceptance bullet 2).
func TestOpenQuestionsIndexOutcome_NoSectionStaysNonBlocking(t *testing.T) {
	const relPath = "lifecycle/requirements/index-outcome-no-section.md"
	seeds := []seedArtifact{
		{relPath: relPath, content: makeArtifact(
			"Index Outcome No Section", "requirement", "draft",
			"index-outcome-no-section", "",
			"## Problem\n\nSome prose with no Open Questions heading anywhere.",
		)},
	}
	env := newTestEnv(t, seeds)

	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	art, _ := data["artifact"].(map[string]any)
	fm, _ := art["frontmatter"].(map[string]any)

	if status := statusOf(fm); status != "draft" {
		t.Errorf("expected status to remain %q with no Open Questions section, got %q", "draft", status)
	}
	if hasProductOwnerAssignee(fm) {
		t.Errorf("expected no product-owner assignee for a non-blocking artifact, got %v", fm["assignees"])
	}
}

// TestOpenQuestionsIndexOutcome_SentinelOnlyStaysNonBlocking verifies that
// indexing a requirement whose only "## Open Questions" content is a
// recognised sentinel does not auto-block it — covering both a freshly
// authored draft (stays draft) and a previously-blocked artifact that now
// only carries a sentinel (auto-unblocks to draft), per Non-functional §2
// backward compatibility.
func TestOpenQuestionsIndexOutcome_SentinelOnlyStaysNonBlocking(t *testing.T) {
	sentinels := []string{"None", "N/A", "na", "nil", "TBD", "no open questions", "no questions"}

	for _, sentinel := range sentinels {
		t.Run(sentinel, func(t *testing.T) {
			slug := "index-outcome-sentinel-" + sanitizeSlug(sentinel)
			relPath := "lifecycle/requirements/" + slug + ".md"
			body := "## Problem\n\nSome prose.\n\n## Open Questions\n\n" + sentinel + "\n"

			seeds := []seedArtifact{
				{relPath: relPath, content: makeArtifact(
					"Index Outcome Sentinel "+sentinel, "requirement", "draft", slug, "", body,
				)},
			}
			env := newTestEnv(t, seeds)

			resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
			requireStatus(t, resp, 200)
			data := readJSON(t, resp)
			art, _ := data["artifact"].(map[string]any)
			fm, _ := art["frontmatter"].(map[string]any)

			if status := statusOf(fm); status != "draft" {
				t.Errorf("sentinel %q: expected status to remain %q, got %q", sentinel, "draft", status)
			}
		})
	}

	// A previously-blocked artifact whose Open Questions section has been
	// reduced to a bare sentinel must auto-unblock back to draft.
	t.Run("PreviouslyBlockedAutoUnblocksOnSentinelOnly", func(t *testing.T) {
		const relPath = "lifecycle/requirements/index-outcome-sentinel-reblocked.md"
		body := "## Problem\n\nSome prose.\n\n## Open Questions\n\nNone.\n"
		seeds := []seedArtifact{
			{relPath: relPath, content: makeArtifact(
				"Index Outcome Sentinel Reblocked", "requirement", "blocked",
				"index-outcome-sentinel-reblocked", "", body,
			)},
		}
		env := newTestEnv(t, seeds)

		resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
		requireStatus(t, resp, 200)
		data := readJSON(t, resp)
		art, _ := data["artifact"].(map[string]any)
		fm, _ := art["frontmatter"].(map[string]any)

		if status := statusOf(fm); status != "draft" {
			t.Errorf("expected auto-unblock to %q for a sentinel-only section, got %q", "draft", status)
		}
	})
}

// TestOpenQuestionsIndexOutcome_GenuineQuestionAutoBlocksWithAssignee
// verifies that indexing a requirement with a real, unresolved question under
// "## Open Questions" auto-blocks it and adds a
// {role: product-owner, who: agent} assignee — the escalation path preserved
// alongside the Milestone 1 prompt change (requirement Functional §3).
func TestOpenQuestionsIndexOutcome_GenuineQuestionAutoBlocksWithAssignee(t *testing.T) {
	const relPath = "lifecycle/requirements/index-outcome-real-question.md"
	body := "## Problem\n\nSome prose.\n\n## Open Questions\n\n- What is the retry budget?\n"
	seeds := []seedArtifact{
		{relPath: relPath, content: makeArtifact(
			"Index Outcome Real Question", "requirement", "draft",
			"index-outcome-real-question", "", body,
		)},
	}
	env := newTestEnv(t, seeds)

	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	art, _ := data["artifact"].(map[string]any)
	fm, _ := art["frontmatter"].(map[string]any)

	if status := statusOf(fm); status != "blocked" {
		t.Errorf("expected auto-block to %q for a genuine open question, got %q", "blocked", status)
	}
	if !hasProductOwnerAssignee(fm) {
		t.Errorf("expected product-owner/agent assignee on auto-block, got %v", fm["assignees"])
	}
}

// sanitizeSlug lowercases and strips characters that are not safe in a
// lineage slug / filename, so each sentinel variant gets a distinct seed path.
func sanitizeSlug(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
		case r >= 'A' && r <= 'Z':
			out = append(out, r-'A'+'a')
		case r == ' ' || r == '/':
			out = append(out, '-')
		}
	}
	return string(out)
}
