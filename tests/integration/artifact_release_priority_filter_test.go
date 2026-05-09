// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"net/http"
	"testing"
)

// ── Milestone 1: Additional artifact filter tests (TC3–TC6) ──────────────────
//
// TC1 (exact-match) and TC2 (unassigned) are covered by
// releases_filter_test.go:TestReleaseFilter_ByReleaseName and
// TestReleaseFilter_Unassigned.

// TestReleaseFilter_Composition verifies that combining release and status query
// parameters returns only artifacts that satisfy both conditions.
// Covers Milestone 1, test case 3.
func TestReleaseFilter_Composition(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/cmp-draft-v1.md",
			content: makeArtifactWithRelease("Cmp Draft V1", "idea", "draft", "cmp-draft-v1", "v-cmp-1", "Body."),
		},
		{
			relPath: "lifecycle/ideas/cmp-approved-v1.md",
			content: makeArtifactWithRelease("Cmp Approved V1", "idea", "approved", "cmp-approved-v1", "v-cmp-1", "Body."),
		},
		{
			relPath: "lifecycle/ideas/cmp-draft-v2.md",
			content: makeArtifactWithRelease("Cmp Draft V2", "idea", "draft", "cmp-draft-v2", "v-cmp-2", "Body."),
		},
		{
			relPath: "lifecycle/ideas/cmp-none.md",
			content: makeArtifact("Cmp No Release", "idea", "draft", "cmp-none", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{"name": "v-cmp-1", "status": "planned"})
	createRelease(t, env, map[string]any{"name": "v-cmp-2", "status": "planned"})

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?release=v-cmp-1&status=draft", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)
	items, _ := data["items"].([]any)
	total, _ := data["total"].(float64)

	if len(items) != 1 {
		t.Errorf("?release=v-cmp-1&status=draft: want 1 item, got %d", len(items))
	}
	if int(total) != 1 {
		t.Errorf("?release=v-cmp-1&status=draft: want total=1, got %d", int(total))
	}
	if len(items) >= 1 {
		item, _ := items[0].(map[string]any)
		fm, _ := item["frontmatter"].(map[string]any)
		if rel, _ := fm["release"].(string); rel != "v-cmp-1" {
			t.Errorf("composition: release want %q, got %q", "v-cmp-1", rel)
		}
		if status, _ := item["status"].(string); status != "draft" {
			t.Errorf("composition: status want %q, got %q", "draft", status)
		}
	}
}

// TestReleaseFilter_NoMatch verifies that querying with a release value that
// matches no artifacts returns an empty result set with total: 0.
// Covers Milestone 1, test case 4.
func TestReleaseFilter_NoMatch(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/nm-has-release.md",
			content: makeArtifactWithRelease("NM Has Release", "idea", "draft", "nm-has-release", "v-nm-1", "Body."),
		},
		{
			relPath: "lifecycle/ideas/nm-no-release.md",
			content: makeArtifact("NM No Release", "idea", "draft", "nm-no-release", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{"name": "v-nm-1", "status": "planned"})

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?release=nonexistent-release-xyz", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)
	items, _ := data["items"].([]any)
	total, _ := data["total"].(float64)

	if len(items) != 0 {
		t.Errorf("?release=nonexistent-release-xyz: want 0 items, got %d", len(items))
	}
	if int(total) != 0 {
		t.Errorf("?release=nonexistent-release-xyz: want total=0, got %d", int(total))
	}
}

// TestPriority_InListResponse verifies that the list endpoint includes
// frontmatter.priority with the correct value for each artifact.
// Covers Milestone 1, test case 5.
func TestPriority_InListResponse(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/prl-high.md",
			content: makeArtifactWithPriority("PRL High", "idea", "draft", "prl-high", "high", "Body."),
		},
		{
			relPath: "lifecycle/ideas/prl-critical.md",
			content: makeArtifactWithPriority("PRL Critical", "idea", "draft", "prl-critical", "critical", "Body."),
		},
		{
			relPath: "lifecycle/ideas/prl-low.md",
			content: makeArtifactWithPriority("PRL Low", "idea", "draft", "prl-low", "low", "Body."),
		},
		{
			relPath: "lifecycle/ideas/prl-none.md",
			content: makeArtifact("PRL None", "idea", "draft", "prl-none", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)
	items, _ := data["items"].([]any)

	if len(items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(items))
	}

	// Build lineage → priority map from the response.
	byLineage := map[string]string{}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		lineage, _ := item["lineage"].(string)
		fm, _ := item["frontmatter"].(map[string]any)
		prio, _ := fm["priority"].(string)
		byLineage[lineage] = prio
	}

	cases := []struct{ lineage, want string }{
		{"prl-high", "high"},
		{"prl-critical", "critical"},
		{"prl-low", "low"},
		{"prl-none", ""},
	}
	for _, c := range cases {
		if got := byLineage[c.lineage]; got != c.want {
			t.Errorf("lineage %q: frontmatter.priority want %q, got %q", c.lineage, c.want, got)
		}
	}
}

// TestRelease_InListResponse verifies that the list endpoint includes
// frontmatter.release with the correct value for each artifact.
// Covers Milestone 1, test case 6.
func TestRelease_InListResponse(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rrl-v1.md",
			content: makeArtifactWithRelease("RRL V1", "idea", "draft", "rrl-v1", "v-rrl-1", "Body."),
		},
		{
			relPath: "lifecycle/ideas/rrl-v2.md",
			content: makeArtifactWithRelease("RRL V2", "idea", "draft", "rrl-v2", "v-rrl-2", "Body."),
		},
		{
			relPath: "lifecycle/ideas/rrl-none.md",
			content: makeArtifact("RRL None", "idea", "draft", "rrl-none", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{"name": "v-rrl-1", "status": "planned"})
	createRelease(t, env, map[string]any{"name": "v-rrl-2", "status": "active"})

	resp := env.doRequest("GET", "/api/p/testproject/artifacts", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)
	items, _ := data["items"].([]any)

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	// Build lineage → release map from the response.
	byLineage := map[string]string{}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		lineage, _ := item["lineage"].(string)
		fm, _ := item["frontmatter"].(map[string]any)
		rel, _ := fm["release"].(string)
		byLineage[lineage] = rel
	}

	cases := []struct{ lineage, want string }{
		{"rrl-v1", "v-rrl-1"},
		{"rrl-v2", "v-rrl-2"},
		{"rrl-none", ""},
	}
	for _, c := range cases {
		if got := byLineage[c.lineage]; got != c.want {
			t.Errorf("lineage %q: frontmatter.release want %q, got %q", c.lineage, c.want, got)
		}
	}
}
