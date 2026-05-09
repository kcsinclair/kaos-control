// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"testing"
)

// ── Milestone 5 — Startup scan corrects stale status ─────────────────────────

// TestAutoBlock_StartupScanBlocksDraftWithOQ verifies that when the server
// starts, a seeded "draft" artifact with a non-empty "## Open Questions"
// section is automatically transitioned to "blocked" before the server
// becomes ready. The test queries the API immediately after newTestEnv
// returns (which implies the startup Scan has already completed).
//
// Run with: go test ./tests/integration/... -tags=integration -run TestAutoBlock_StartupScanBlocksDraftWithOQ
func TestAutoBlock_StartupScanBlocksDraftWithOQ(t *testing.T) {
	const relPath = "lifecycle/ideas/startup-should-block.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			// Seed a draft artifact with open questions: startup Scan must
			// auto-block it before the HTTP server accepts requests.
			content: makeArtifact("Startup Should Block", "idea", "draft",
				"startup-should-block", "",
				"## Open Questions\n\n- Why does this need answering?\n"),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Query the API immediately — startup Scan is synchronous, so the artifact
	// must already be "blocked" by the time newTestEnv returns.
	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	art, _ := data["artifact"].(map[string]any)
	fm, _ := art["frontmatter"].(map[string]any)

	if status, _ := fm["status"].(string); status != "blocked" {
		t.Errorf("expected startup Scan to auto-block the artifact, got status %q", status)
	}

	assignees, _ := fm["assignees"].([]any)
	found := false
	for _, a := range assignees {
		entry, _ := a.(map[string]any)
		if entry["role"] == "product-owner" && entry["who"] == "agent" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected {role:product-owner, who:agent} assignee after startup auto-block; got: %v", assignees)
	}
}

// TestAutoBlock_StartupScanUnblocksBlockedWithNoOQ verifies that when the
// server starts, a seeded "blocked" artifact with NO open questions is
// automatically transitioned to "draft" by the startup Scan.
//
// Run with: go test ./tests/integration/... -tags=integration -run TestAutoBlock_StartupScanUnblocksBlockedWithNoOQ
func TestAutoBlock_StartupScanUnblocksBlockedWithNoOQ(t *testing.T) {
	const relPath = "lifecycle/ideas/startup-should-unblock.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			// Seed a blocked artifact without open questions: startup Scan must
			// auto-unblock it to draft.
			content: makeArtifact("Startup Should Unblock", "idea", "blocked",
				"startup-should-unblock", "",
				"No open questions — this artifact was left blocked by mistake.\n"),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	art, _ := data["artifact"].(map[string]any)
	fm, _ := art["frontmatter"].(map[string]any)

	if status, _ := fm["status"].(string); status != "draft" {
		t.Errorf("expected startup Scan to auto-unblock the artifact to 'draft', got status %q", status)
	}
}
