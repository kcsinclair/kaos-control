// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"net/http"
	"os"
	"strings"
	"testing"
)

// ── Milestone 3: Rename propagation ───────────────────────────────────────────

// TestReleaseRename_PropagatesReleaseField verifies that renaming a release via
// PUT updates the release frontmatter field in all assigned artifact files on
// disk and re-indexes them so queries by the new name work correctly.
func TestReleaseRename_PropagatesReleaseField(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rr-idea-1.md",
			content: makeArtifactWithRelease("RR Idea 1", "idea", "draft", "rr-idea-1", "v1.0", "Body."),
		},
		{
			relPath: "lifecycle/ideas/rr-idea-2.md",
			content: makeArtifactWithRelease("RR Idea 2", "idea", "draft", "rr-idea-2", "v1.0", "Body."),
		},
		{
			relPath: "lifecycle/defects/rr-defect-1.md",
			content: makeArtifactWithRelease("RR Defect 1", "defect", "draft", "rr-defect-1", "v1.0", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Create the release record so PropagateRename can find it.
	data := createRelease(t, env, map[string]any{"name": "v1.0", "status": "planned"})
	id := releaseID(t, data)

	// Rename v1.0 → v1.1.
	resp := env.doRequest("PUT", releasePath(id), map[string]any{
		"name":   "v1.1",
		"status": "planned",
	})
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	renamed, _ := body["artifacts_renamed"].(float64)
	if int(renamed) != 3 {
		t.Errorf("artifacts_renamed: want 3, got %d", int(renamed))
	}

	// All three artifact files must have release: v1.1 on disk.
	for _, relPath := range []string{
		"lifecycle/ideas/rr-idea-1.md",
		"lifecycle/ideas/rr-idea-2.md",
		"lifecycle/defects/rr-defect-1.md",
	} {
		got := readArtifactRelease(t, env.projectRoot, relPath)
		if got != "v1.1" {
			t.Errorf("%s: release field on disk: want %q, got %q", relPath, "v1.1", got)
		}
	}

	// Index must reflect new name: query by v1.1 returns 3 artifacts.
	filterResp := env.doRequest("GET", "/api/p/testproject/artifacts?release=v1.1", nil)
	requireStatus(t, filterResp, http.StatusOK)
	filterData := readJSON(t, filterResp)
	items, _ := filterData["items"].([]any)
	if len(items) != 3 {
		t.Errorf("GET /artifacts?release=v1.1: want 3, got %d", len(items))
	}

	// Query by old name must return empty.
	oldResp := env.doRequest("GET", "/api/p/testproject/artifacts?release=v1.0", nil)
	requireStatus(t, oldResp, http.StatusOK)
	oldData := readJSON(t, oldResp)
	oldItems, _ := oldData["items"].([]any)
	if len(oldItems) != 0 {
		t.Errorf("GET /artifacts?release=v1.0 after rename: want 0, got %d", len(oldItems))
	}
}

// TestReleaseRename_GitCommitCreated verifies that renaming a release with at
// least one assigned artifact produces a git commit whose message contains
// both the old and new release names.
func TestReleaseRename_GitCommitCreated(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rr-git-idea.md",
			content: makeArtifactWithRelease("RR Git Idea", "idea", "draft", "rr-git-idea", "v-git-old", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{"name": "v-git-old", "status": "planned"})
	id := releaseID(t, data)

	resp := env.doRequest("PUT", releasePath(id), map[string]any{
		"name":   "v-git-new",
		"status": "planned",
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Verify a git commit was created for the artifact file.
	commits, err := env.proj.Git.Log("lifecycle/ideas/rr-git-idea.md", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) == 0 {
		t.Fatal("expected at least one commit after rename propagation")
	}

	// The most recent commit message should reference both old and new names.
	msg := commits[0].Message
	if !strings.Contains(msg, "v-git-old") || !strings.Contains(msg, "v-git-new") {
		t.Errorf("commit message %q should contain both old and new release names", msg)
	}
}

// TestReleaseRename_NoCollateralDamage verifies that artifacts assigned to a
// different release are not modified when another release is renamed.
func TestReleaseRename_NoCollateralDamage(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rr-v1-idea.md",
			content: makeArtifactWithRelease("RR V1 Idea", "idea", "draft", "rr-v1-idea", "v-ncd-1", "Body."),
		},
		{
			relPath: "lifecycle/ideas/rr-v2-idea.md",
			content: makeArtifactWithRelease("RR V2 Idea", "idea", "draft", "rr-v2-idea", "v-ncd-2", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	data1 := createRelease(t, env, map[string]any{"name": "v-ncd-1", "status": "planned"})
	id1 := releaseID(t, data1)
	createRelease(t, env, map[string]any{"name": "v-ncd-2", "status": "planned"})

	// Rename v-ncd-1 → v-ncd-1-renamed.
	resp := env.doRequest("PUT", releasePath(id1), map[string]any{
		"name":   "v-ncd-1-renamed",
		"status": "planned",
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// v-ncd-2 artifact must be unchanged.
	got := readArtifactRelease(t, env.projectRoot, "lifecycle/ideas/rr-v2-idea.md")
	if got != "v-ncd-2" {
		t.Errorf("collateral artifact release field: want %q, got %q", "v-ncd-2", got)
	}
}

// TestReleaseRename_UnassignedUnaffected verifies that an artifact with no
// release field is not modified by a rename propagation.
func TestReleaseRename_UnassignedUnaffected(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rr-assigned.md",
			content: makeArtifactWithRelease("RR Assigned", "idea", "draft", "rr-assigned", "v-unaffected", "Body."),
		},
		{
			relPath: "lifecycle/ideas/rr-unassigned.md",
			// No release field.
			content: makeArtifact("RR Unassigned", "idea", "draft", "rr-unassigned", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{"name": "v-unaffected", "status": "planned"})
	id := releaseID(t, data)

	resp := env.doRequest("PUT", releasePath(id), map[string]any{
		"name":   "v-unaffected-new",
		"status": "planned",
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Unassigned artifact must still have no release field.
	content, err := os.ReadFile(env.projectRoot + "/lifecycle/ideas/rr-unassigned.md")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "release:") {
		t.Error("unassigned artifact should not have a release field after rename")
	}
}
