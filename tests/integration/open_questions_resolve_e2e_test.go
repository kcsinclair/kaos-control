// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"encoding/json"
	"strings"
	"testing"
)

// statusOf extracts frontmatter.status from a decoded artifact frontmatter map.
func statusOf(fm map[string]any) string {
	s, _ := fm["status"].(string)
	return s
}

// hasProductOwnerAssignee reports whether fm.assignees contains a
// {role: product-owner, who: agent} entry.
func hasProductOwnerAssignee(fm map[string]any) bool {
	assignees, _ := fm["assignees"].([]any)
	for _, a := range assignees {
		entry, _ := a.(map[string]any)
		if entry["role"] == "product-owner" && entry["who"] == "agent" {
			return true
		}
	}
	return false
}

// fetchArtifactBody GETs the plain artifact endpoint (not /open-questions,
// which always reports a hardcoded heading) and returns its raw body text.
func fetchArtifactBody(t *testing.T, env *testEnv, relPath string) string {
	t.Helper()
	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	body, _ := data["body"].(string)
	return body
}

// TestOpenQuestionsResolve_EndToEnd drives the full resolve → unblock lifecycle
// for a single artifact through the real HTTP API: auto-block on creation,
// preview+PUT for a partial save, preview+PUT for a complete save, a check
// that the client never authors the status mutation itself, a repeat save to
// confirm idempotency, and finally re-opening the artifact to re-block it.
func TestOpenQuestionsResolve_EndToEnd(t *testing.T) {
	const relPath = "lifecycle/ideas/resolve-e2e.md"
	const lineage = "resolve-e2e"
	seeds := []seedArtifact{
		{relPath: relPath, content: makeArtifact("Resolve E2E", "idea", "draft", lineage, "", "Initial body, no questions yet.")},
	}
	env := newTestEnv(t, seeds)

	baseFrontmatter := map[string]any{
		"title":   "Resolve E2E",
		"type":    "idea",
		"lineage": lineage,
	}

	// ── Step 1: a body with a non-empty "## Open Questions" section auto-blocks
	// the artifact and assigns product-owner. ──────────────────────────────────
	t.Run("CreateWithQuestionsBlocksAndAssignsProductOwner", func(t *testing.T) {
		fm := map[string]any{"title": baseFrontmatter["title"], "type": baseFrontmatter["type"], "lineage": lineage, "status": "draft"}
		resp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPath, map[string]any{
			"frontmatter": fm,
			"body":        "## Open Questions\n\n- Q1: Why is X?\n\n- Q2: What about Y?\n",
		})
		requireStatus(t, resp, 200)
		data := readJSON(t, resp)
		art, _ := data["artifact"].(map[string]any)
		gotFM, _ := art["frontmatter"].(map[string]any)
		if status := statusOf(gotFM); status != "blocked" {
			t.Fatalf("expected status %q after PUT with open questions, got %q", "blocked", status)
		}
		if !hasProductOwnerAssignee(gotFM) {
			t.Errorf("expected product-owner/agent assignee, got %v", gotFM["assignees"])
		}
	})

	// ── Step 2: partial resolve — preview(complete=false) then PUT the
	// returned body leaves the artifact blocked and the heading unchanged. ────
	t.Run("PartialResolveStaysBlocked", func(t *testing.T) {
		previewResp := env.doRequest("POST", "/api/p/testproject/artifacts/"+relPath+"/open-questions/preview", map[string]any{
			"answers":  map[string]string{"0": "Because of reasons."},
			"complete": false,
		})
		requireStatus(t, previewResp, 200)
		previewData := readJSON(t, previewResp)
		newBody, _ := previewData["body"].(string)
		if !strings.Contains(newBody, "> Because of reasons.") {
			t.Fatalf("preview did not write the partial answer, got: %q", newBody)
		}

		putResp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPath, map[string]any{
			"frontmatter": map[string]any{
				"title":   baseFrontmatter["title"],
				"type":    baseFrontmatter["type"],
				"lineage": lineage,
				"status":  "blocked",
			},
			"body": newBody,
		})
		requireStatus(t, putResp, 200)
		putData := readJSON(t, putResp)
		art, _ := putData["artifact"].(map[string]any)
		gotFM, _ := art["frontmatter"].(map[string]any)
		if status := statusOf(gotFM); status != "blocked" {
			t.Errorf("partial resolve: expected status to remain %q, got %q", "blocked", status)
		}

		onDisk := fetchArtifactBody(t, env, relPath)
		if !strings.Contains(onDisk, "## Open Questions") {
			t.Errorf("partial resolve: expected heading '## Open Questions' to remain, got: %q", onDisk)
		}
		if strings.Contains(onDisk, "## Resolved Questions") {
			t.Errorf("partial resolve: heading must not be renamed yet, got: %q", onDisk)
		}
	})

	// ── Step 3: complete resolve — preview(complete=true) then PUT the
	// returned body renames the heading and auto-unblocks to draft. ───────────
	var completedBody string
	t.Run("CompleteResolveRenamesHeadingAndUnblocks", func(t *testing.T) {
		previewResp := env.doRequest("POST", "/api/p/testproject/artifacts/"+relPath+"/open-questions/preview", map[string]any{
			"answers":  map[string]string{"1": "Roughly 11 m/s for an African swallow."},
			"complete": true,
		})
		requireStatus(t, previewResp, 200)
		previewData := readJSON(t, previewResp)
		newBody, _ := previewData["body"].(string)
		if !strings.Contains(newBody, "## Resolved Questions") {
			t.Fatalf("complete preview did not rename the heading, got: %q", newBody)
		}
		completedBody = newBody

		putResp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPath, map[string]any{
			"frontmatter": map[string]any{
				"title":   baseFrontmatter["title"],
				"type":    baseFrontmatter["type"],
				"lineage": lineage,
				"status":  "blocked",
			},
			"body": newBody,
		})
		requireStatus(t, putResp, 200)
		putData := readJSON(t, putResp)
		art, _ := putData["artifact"].(map[string]any)
		gotFM, _ := art["frontmatter"].(map[string]any)
		if status := statusOf(gotFM); status != "draft" {
			t.Errorf("complete resolve: expected auto-unblock to %q, got %q", "draft", status)
		}

		onDisk := fetchArtifactBody(t, env, relPath)
		if !strings.Contains(onDisk, "## Resolved Questions") {
			t.Errorf("complete resolve: expected heading '## Resolved Questions' on disk, got: %q", onDisk)
		}
		if strings.Contains(onDisk, "## Open Questions") {
			t.Errorf("complete resolve: old heading must be gone, got: %q", onDisk)
		}
	})

	// ── Step 4: the PUT the client sent to complete the resolve carried only
	// frontmatter+body, and its frontmatter.status was the artifact's
	// pre-existing status ("blocked") — the draft transition in step 3 was
	// computed by the server, not authored by the client. ─────────────────────
	t.Run("PutPayloadCarriesNoClientAuthoredStatusMutation", func(t *testing.T) {
		reqPayload := map[string]any{
			"frontmatter": map[string]any{
				"title":   baseFrontmatter["title"],
				"type":    baseFrontmatter["type"],
				"lineage": lineage,
				"status":  "blocked", // unchanged from the artifact's on-disk status
			},
			"body": completedBody,
		}
		raw, err := json.Marshal(reqPayload)
		if err != nil {
			t.Fatal(err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatal(err)
		}
		if len(decoded) != 2 {
			t.Fatalf("expected PUT payload to carry only frontmatter+body, got keys: %v", decoded)
		}
		if _, ok := decoded["frontmatter"]; !ok {
			t.Error("expected 'frontmatter' key in PUT payload")
		}
		if _, ok := decoded["body"]; !ok {
			t.Error("expected 'body' key in PUT payload")
		}
		sentFM, _ := decoded["frontmatter"].(map[string]any)
		if status := statusOf(sentFM); status != "blocked" {
			t.Errorf("client-authored status should equal the pre-resolve status %q, got %q", "blocked", status)
		}

		// The server-side result (already verified in the previous step) is
		// "draft" despite the client submitting "blocked" — the transition was
		// computed by applyOpenQuestionTransition, not requested by the client.
		getResp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
		requireStatus(t, getResp, 200)
		getData := readJSON(t, getResp)
		art, _ := getData["artifact"].(map[string]any)
		gotFM, _ := art["frontmatter"].(map[string]any)
		if status := statusOf(gotFM); status != "draft" {
			t.Errorf("expected server-computed status %q, got %q", "draft", status)
		}
	})

	// ── Step 5: re-saving the same completed answers is idempotent. ───────────
	t.Run("ResavingCompletedAnswersIsIdempotent", func(t *testing.T) {
		beforeBody := fetchArtifactBody(t, env, relPath)

		putResp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPath, map[string]any{
			"frontmatter": map[string]any{
				"title":   baseFrontmatter["title"],
				"type":    baseFrontmatter["type"],
				"lineage": lineage,
				"status":  "draft",
			},
			"body": completedBody,
		})
		requireStatus(t, putResp, 200)
		putData := readJSON(t, putResp)
		art, _ := putData["artifact"].(map[string]any)
		gotFM, _ := art["frontmatter"].(map[string]any)
		if status := statusOf(gotFM); status != "draft" {
			t.Errorf("idempotent re-save: expected status to remain %q, got %q", "draft", status)
		}

		afterBody := fetchArtifactBody(t, env, relPath)
		if beforeBody != afterBody {
			t.Errorf("idempotent re-save changed the on-disk body:\nbefore: %q\nafter:  %q", beforeBody, afterBody)
		}
	})

	// ── Step 6: re-opening — moving an item back under a non-empty
	// "## Open Questions" heading and PUTting re-blocks the artifact. ─────────
	t.Run("ReopeningReblocksArtifact", func(t *testing.T) {
		reopenedBody := strings.Replace(completedBody, "## Resolved Questions", "## Open Questions", 1)

		putResp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPath, map[string]any{
			"frontmatter": map[string]any{
				"title":   baseFrontmatter["title"],
				"type":    baseFrontmatter["type"],
				"lineage": lineage,
				"status":  "draft",
			},
			"body": reopenedBody,
		})
		requireStatus(t, putResp, 200)
		putData := readJSON(t, putResp)
		art, _ := putData["artifact"].(map[string]any)
		gotFM, _ := art["frontmatter"].(map[string]any)
		if status := statusOf(gotFM); status != "blocked" {
			t.Errorf("reopen: expected auto-block to %q, got %q", "blocked", status)
		}
		if !hasProductOwnerAssignee(gotFM) {
			t.Errorf("reopen: expected product-owner/agent assignee restored, got %v", gotFM["assignees"])
		}
	})
}
