// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"net/http"
	"strings"
	"testing"
)

// ── Milestone 4: Delete and reassign ─────────────────────────────────────────

// TestReleaseDelete_WithoutReassignment verifies that deleting a release with
// assigned artifacts leaves the artifact files unchanged on disk (release field
// is NOT cleared) and returns the correct orphaned_artifact_count.
func TestReleaseDelete_WithoutReassignment(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rd-orphan-1.md",
			content: makeArtifactWithRelease("RD Orphan 1", "idea", "draft", "rd-orphan-1", "v-del-no-reassign", "Body."),
		},
		{
			relPath: "lifecycle/ideas/rd-orphan-2.md",
			content: makeArtifactWithRelease("RD Orphan 2", "idea", "draft", "rd-orphan-2", "v-del-no-reassign", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{"name": "v-del-no-reassign", "status": "planned"})
	id := releaseID(t, data)

	resp := env.doRequest("DELETE", releasePath(id), nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	count, _ := body["orphaned_artifact_count"].(float64)
	if int(count) != 2 {
		t.Errorf("orphaned_artifact_count: want 2, got %d", int(count))
	}

	// Artifact files must still have the original release field on disk.
	for _, relPath := range []string{
		"lifecycle/ideas/rd-orphan-1.md",
		"lifecycle/ideas/rd-orphan-2.md",
	} {
		got := readArtifactRelease(t, env.projectRoot, relPath)
		if got != "v-del-no-reassign" {
			t.Errorf("%s: release field should be unchanged after delete without reassignment, got %q", relPath, got)
		}
	}

	// Release must not appear in the list.
	listResp := env.doRequest("GET", "/api/p/testproject/releases", nil)
	requireStatus(t, listResp, http.StatusOK)
	listData := readJSON(t, listResp)
	releases, _ := listData["releases"].([]any)
	for _, raw := range releases {
		rel, _ := raw.(map[string]any)
		if name, _ := rel["name"].(string); name == "v-del-no-reassign" {
			t.Error("deleted release should not appear in the list")
		}
	}
}

// TestReleaseDelete_WithReassignment verifies that deleting a release with
// reassign_to=<id> rewrites artifact files to point to the target release and
// they appear in the target release's artifact list.
func TestReleaseDelete_WithReassignment(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rd-reassign-1.md",
			content: makeArtifactWithRelease("RD Reassign 1", "idea", "draft", "rd-reassign-1", "v-del-src", "Body."),
		},
		{
			relPath: "lifecycle/ideas/rd-reassign-2.md",
			content: makeArtifactWithRelease("RD Reassign 2", "idea", "draft", "rd-reassign-2", "v-del-src", "Body."),
		},
		{
			relPath: "lifecycle/ideas/rd-reassign-3.md",
			content: makeArtifactWithRelease("RD Reassign 3", "idea", "draft", "rd-reassign-3", "v-del-src", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	srcData := createRelease(t, env, map[string]any{"name": "v-del-src", "status": "planned"})
	srcID := releaseID(t, srcData)

	tgtData := createRelease(t, env, map[string]any{"name": "v-del-tgt", "status": "planned"})
	tgtID := releaseID(t, tgtData)

	// Delete source release, reassigning artifacts to target.
	resp := env.doRequest("DELETE", releasePath(srcID)+"?reassign_to="+releasePath2(tgtID), nil)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// All three artifact files must now have release: v-del-tgt on disk.
	for _, relPath := range []string{
		"lifecycle/ideas/rd-reassign-1.md",
		"lifecycle/ideas/rd-reassign-2.md",
		"lifecycle/ideas/rd-reassign-3.md",
	} {
		got := readArtifactRelease(t, env.projectRoot, relPath)
		if got != "v-del-tgt" {
			t.Errorf("%s: release field: want %q, got %q", relPath, "v-del-tgt", got)
		}
	}

	// Target release artifact list must contain the three reassigned artifacts.
	listResp := env.doRequest("GET", releasePath(tgtID)+"/artifacts", nil)
	requireStatus(t, listResp, http.StatusOK)
	listData := readJSON(t, listResp)
	items, _ := listData["items"].([]any)
	if len(items) != 3 {
		t.Errorf("target release artifact list: want 3, got %d", len(items))
	}
}

// TestReleaseDelete_EmptyRelease verifies that deleting a release with no
// assigned artifacts returns 200 with orphaned_artifact_count=0.
func TestReleaseDelete_EmptyRelease(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{"name": "v-del-empty2", "status": "planned"})
	id := releaseID(t, data)

	resp := env.doRequest("DELETE", releasePath(id), nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	count, _ := body["orphaned_artifact_count"].(float64)
	if int(count) != 0 {
		t.Errorf("orphaned_artifact_count: want 0, got %d", int(count))
	}
}

// TestReleaseDelete_ReassignToNonExistent verifies that specifying a non-existent
// reassign_to target returns 400 or 404.
func TestReleaseDelete_ReassignToNonExistent(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{"name": "v-del-reassign-bad", "status": "planned"})
	id := releaseID(t, data)

	resp := env.doRequest("DELETE", releasePath(id)+"?reassign_to=99999", nil)
	// Spec allows 400 or 404 for a non-existent reassign target.
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
		b, _ := readBodyBytes(resp)
		t.Errorf("expected 400 or 404, got %d: %s", resp.StatusCode, b)
	} else {
		resp.Body.Close()
	}
}

// releasePath2 returns just the numeric ID portion for use in query strings.
func releasePath2(id int64) string {
	return strings.TrimPrefix(releasePath(id), "/api/p/testproject/releases/")
}

// readBodyBytes reads and returns the raw body bytes from a response, closing
// the body afterwards.
func readBodyBytes(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	buf := make([]byte, 0, 512)
	tmp := make([]byte, 512)
	for {
		n, err := resp.Body.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			break
		}
	}
	return buf, nil
}
