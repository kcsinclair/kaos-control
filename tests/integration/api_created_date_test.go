// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// TestCreateArtifact_SetsCreatedDate verifies that POST /api/p/:project/artifacts
// automatically stamps the created field in frontmatter and returns it as a valid
// ISO 8601 timestamp within 5 seconds of the request.
func TestCreateArtifact_SetsCreatedDate(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	before := time.Now().Add(-time.Second)

	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "ideas",
		"slug":  "created-stamp",
		"frontmatter": map[string]any{
			"title":   "Created Stamp Test",
			"type":    "idea",
			"status":  "draft",
			"lineage": "created-stamp",
		},
		"body": "Testing that server stamps created date.",
	})
	requireStatus(t, resp, 201)

	after := time.Now().Add(time.Second)

	data := readJSON(t, resp)
	artifact, _ := data["artifact"].(map[string]any)
	if artifact == nil {
		t.Fatal("expected artifact in response")
	}

	// Check created in the index row (time.Time, JSON as RFC3339Nano).
	createdRaw, ok := artifact["created"].(string)
	if !ok || createdRaw == "" {
		t.Fatalf("expected artifact.created string in response, got %T %v", artifact["created"], artifact["created"])
	}
	createdTime, err := time.Parse(time.RFC3339Nano, createdRaw)
	if err != nil {
		// Try plain RFC3339
		createdTime, err = time.Parse(time.RFC3339, createdRaw)
		if err != nil {
			t.Fatalf("artifact.created %q is not a valid ISO 8601 timestamp: %v", createdRaw, err)
		}
	}
	if createdTime.Before(before) || createdTime.After(after) {
		t.Errorf("artifact.created %v is not within 5s window [%v, %v]", createdTime, before, after)
	}

	// Also verify the frontmatter.created field is set and parseable.
	fm, _ := artifact["frontmatter"].(map[string]any)
	fmCreated, _ := fm["created"].(string)
	if fmCreated == "" {
		t.Error("expected frontmatter.created to be set")
	} else {
		if _, err := time.Parse(time.RFC3339, fmCreated); err != nil {
			t.Errorf("frontmatter.created %q is not valid RFC3339: %v", fmCreated, err)
		}
	}
}

// TestUpdateArtifact_PreservesCreatedDate verifies that PUT does not change the
// `created` field that was set when the artifact was first created.
func TestUpdateArtifact_PreservesCreatedDate(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Create the artifact.
	createResp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "ideas",
		"slug":  "preserve-created",
		"frontmatter": map[string]any{
			"title":   "Preserve Created",
			"type":    "idea",
			"status":  "draft",
			"lineage": "preserve-created",
		},
		"body": "Original body.",
	})
	requireStatus(t, createResp, 201)
	createData := readJSON(t, createResp)

	createArtifact, _ := createData["artifact"].(map[string]any)
	origCreated, _ := createArtifact["created"].(string)
	if origCreated == "" {
		t.Fatal("expected created in create response")
	}
	createPath, _ := createData["path"].(string)

	// Update the artifact body via PUT.
	putResp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+createPath, map[string]any{
		"frontmatter": map[string]any{
			"title":   "Preserve Created",
			"type":    "idea",
			"status":  "draft",
			"lineage": "preserve-created",
		},
		"body": "Updated body.",
	})
	requireStatus(t, putResp, 200)
	putData := readJSON(t, putResp)

	putArtifact, _ := putData["artifact"].(map[string]any)
	newCreated, _ := putArtifact["created"].(string)
	if newCreated == "" {
		t.Fatal("expected created in PUT response")
	}
	if newCreated != origCreated {
		t.Errorf("created changed after PUT: was %q, now %q", origCreated, newCreated)
	}
}

// TestUpdateArtifact_CannotOverwriteCreated verifies that PUT with a different
// `created` value in the request frontmatter is silently ignored: the server
// always preserves the original on-disk created value.
func TestUpdateArtifact_CannotOverwriteCreated(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Create the artifact.
	createResp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "ideas",
		"slug":  "no-overwrite-created",
		"frontmatter": map[string]any{
			"title":   "No Overwrite Created",
			"type":    "idea",
			"status":  "draft",
			"lineage": "no-overwrite-created",
		},
		"body": "Original.",
	})
	requireStatus(t, createResp, 201)
	createData := readJSON(t, createResp)

	createArtifact, _ := createData["artifact"].(map[string]any)
	origCreated, _ := createArtifact["created"].(string)
	if origCreated == "" {
		t.Fatal("expected created in create response")
	}
	createPath, _ := createData["path"].(string)

	// PUT with an explicitly different created value in the frontmatter.
	fakeCreated := "2000-01-01T00:00:00Z"
	putResp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+createPath, map[string]any{
		"frontmatter": map[string]any{
			"title":   "No Overwrite Created",
			"type":    "idea",
			"status":  "draft",
			"lineage": "no-overwrite-created",
			"created": fakeCreated,
		},
		"body": "Updated.",
	})
	requireStatus(t, putResp, 200)
	putData := readJSON(t, putResp)

	putArtifact, _ := putData["artifact"].(map[string]any)
	resultCreated, _ := putArtifact["created"].(string)
	if resultCreated == fakeCreated {
		t.Errorf("server accepted overwritten created value %q; expected original %q", fakeCreated, origCreated)
	}
	if resultCreated != origCreated {
		t.Errorf("created changed unexpectedly: was %q, now %q", origCreated, resultCreated)
	}
}

// TestGetArtifact_ReturnsCreatedAndMtime verifies that the GET detail endpoint
// includes both `created` and `mtime` fields in the artifact JSON.
func TestGetArtifact_ReturnsCreatedAndMtime(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Create an artifact.
	createResp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "ideas",
		"slug":  "get-dates",
		"frontmatter": map[string]any{
			"title":   "Get Dates Test",
			"type":    "idea",
			"status":  "draft",
			"lineage": "get-dates",
		},
		"body": "Testing date fields in GET response.",
	})
	requireStatus(t, createResp, 201)
	createData := readJSON(t, createResp)
	createPath, _ := createData["path"].(string)

	// GET the artifact detail.
	getResp := env.doRequest("GET", "/api/p/testproject/artifacts/"+createPath, nil)
	requireStatus(t, getResp, 200)
	getData := readJSON(t, getResp)

	artifact, _ := getData["artifact"].(map[string]any)
	if artifact == nil {
		t.Fatal("expected artifact in GET response")
	}

	// Both created and mtime must be present and non-empty.
	created, _ := artifact["created"].(string)
	if created == "" {
		t.Error("artifact.created missing or empty in GET response")
	}
	mtime, _ := artifact["mtime"].(string)
	if mtime == "" {
		t.Error("artifact.mtime missing or empty in GET response")
	}

	// Both must parse as valid timestamps.
	for _, tc := range []struct{ name, val string }{
		{"created", created},
		{"mtime", mtime},
	} {
		if tc.val == "" {
			continue
		}
		if _, err := time.Parse(time.RFC3339Nano, tc.val); err != nil {
			if _, err2 := time.Parse(time.RFC3339, tc.val); err2 != nil {
				t.Errorf("artifact.%s %q is not a valid timestamp: %v", tc.name, tc.val, err)
			}
		}
	}
}

// TestIndexScan_BackfillsCreatedFromGit verifies that when an artifact file
// has no `created:` frontmatter field but is committed in git, IndexFile
// derives the created date from the first git commit's author date.
func TestIndexScan_BackfillsCreatedFromGit(t *testing.T) {
	env := newTestEnv(t, nil)

	const relPath = "lifecycle/ideas/backfill-git.md"
	absPath := filepath.Join(env.projectRoot, relPath)

	// Write a file without `created:` in frontmatter.
	content := makeArtifact("Backfill Git", "idea", "draft", "backfill-git", "", "No created field.")
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Commit via go-git so git history exists for this file.
	before := time.Now().Add(-time.Second)
	repo, err := gogit.PlainOpen(env.projectRoot)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Add(relPath); err != nil {
		t.Fatal(err)
	}
	commitWhen := time.Now()
	_, err = wt.Commit("add backfill-git artifact", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@test.local",
			When:  commitWhen,
		},
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	after := time.Now().Add(time.Second)

	// Trigger index update using the project's index (which has git support).
	if err := env.proj.Idx.IndexFile(absPath); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	row, err := env.proj.Idx.Get(relPath)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if row == nil {
		t.Fatal("artifact not found in index after IndexFile")
	}
	if row.Created.IsZero() {
		t.Fatal("expected non-zero Created from git backfill")
	}

	// Backfill should use the git commit's author time, which is within our window.
	if row.Created.Before(before) || row.Created.After(after) {
		t.Errorf("Created from git backfill %v outside expected window [%v, %v]",
			row.Created, before, after)
	}

	// On-disk file must NOT have been modified (backfill is index-only).
	raw, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatal(err)
	}
	if findSubstring(string(raw), "created:") {
		t.Error("IndexFile backfill must not write `created:` to the on-disk file")
	}
}

// TestIndexScan_BackfillFallsBackToMtime verifies that when an artifact has no
// `created:` frontmatter and no git history (untracked file), IndexFile falls
// back to the filesystem mtime as the created date.
func TestIndexScan_BackfillFallsBackToMtime(t *testing.T) {
	env := newTestEnv(t, nil)

	const relPath = "lifecycle/ideas/backfill-mtime.md"
	absPath := filepath.Join(env.projectRoot, relPath)

	// Write the file — do NOT commit it so there is no git history.
	content := makeArtifact("Backfill Mtime", "idea", "draft", "backfill-mtime", "", "No git history.")
	before := time.Now().Add(-time.Second)
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	after := time.Now().Add(time.Second)

	// Index the file (git lookup will fail → mtime fallback).
	if err := env.proj.Idx.IndexFile(absPath); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	row, err := env.proj.Idx.Get(relPath)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if row == nil {
		t.Fatal("artifact not found in index after IndexFile")
	}
	if row.Created.IsZero() {
		t.Fatal("expected non-zero Created from mtime fallback")
	}
	if row.Created.Before(before) || row.Created.After(after) {
		t.Errorf("Created from mtime fallback %v outside expected window [%v, %v]",
			row.Created, before, after)
	}

	// On-disk file must NOT have been modified.
	raw, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatal(err)
	}
	if findSubstring(string(raw), "created:") {
		t.Error("IndexFile backfill must not write `created:` to the on-disk file")
	}
}
