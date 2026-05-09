// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRenameWithLinkRewrite verifies that renaming an artifact rewrites
// inbound links in other files and commits all changes atomically.
// Test plan §7: "Rename with link rewrite" scenario.
func TestRenameWithLinkRewrite(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/old-name.md",
			content: makeArtifact("Old Name", "idea", "draft", "old-name", "",
				"The original idea."),
		},
		{
			relPath: "lifecycle/requirements/old-name-2.md",
			content: makeArtifact("Old Name Requirements", "ticket", "draft", "old-name",
				"lifecycle/ideas/old-name.md",
				"Requirements referencing [[ideas/old-name]]."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Rename the idea from old-name to new-name.
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/lifecycle/ideas/old-name.md/rename", map[string]any{
		"new_slug": "new-name",
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	newPath, _ := data["path"].(string)
	if newPath != "lifecycle/ideas/new-name.md" {
		t.Errorf("expected new path lifecycle/ideas/new-name.md, got %s", newPath)
	}

	// Verify old file no longer exists on disk.
	oldAbsPath := filepath.Join(env.projectRoot, "lifecycle", "ideas", "old-name.md")
	if _, err := os.Stat(oldAbsPath); !os.IsNotExist(err) {
		t.Error("old file should no longer exist on disk")
	}

	// Verify new file exists on disk.
	newAbsPath := filepath.Join(env.projectRoot, "lifecycle", "ideas", "new-name.md")
	if _, err := os.Stat(newAbsPath); os.IsNotExist(err) {
		t.Error("new file should exist on disk")
	}

	// Verify old path is removed from index.
	row, _ := env.proj.Idx.Get("lifecycle/ideas/old-name.md")
	if row != nil {
		t.Error("old path should be removed from index")
	}

	// Verify new path is in index.
	row, _ = env.proj.Idx.Get("lifecycle/ideas/new-name.md")
	if row == nil {
		t.Fatal("new path should be in index")
	}
	if row.Title != "Old Name" {
		t.Errorf("title should be preserved, got %q", row.Title)
	}

	// Verify inbound links in the requirement file were rewritten.
	reqContent, err := os.ReadFile(filepath.Join(env.projectRoot, "lifecycle", "requirements", "old-name-2.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(reqContent), "old-name") && !strings.Contains(string(reqContent), "new-name") {
		t.Error("inbound links in requirement should be rewritten to new-name")
	}

	// Verify git commit was created covering the rename.
	commits, err := env.proj.Git.Log("lifecycle/ideas/new-name.md", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) == 0 {
		t.Error("expected at least one commit for the renamed artifact")
	}
}

// TestRenameToExistingSlugFails verifies that renaming to a slug that already
// has a file at the target path returns 409.
func TestRenameToExistingSlugFails(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/source.md",
			content: makeArtifact("Source", "idea", "draft", "source", "", "Source idea."),
		},
		{
			relPath: "lifecycle/ideas/target.md",
			content: makeArtifact("Target", "idea", "draft", "target", "", "Target idea."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/artifacts/lifecycle/ideas/source.md/rename", map[string]any{
		"new_slug": "target",
	})
	requireStatus(t, resp, 409)
	resp.Body.Close()
}
