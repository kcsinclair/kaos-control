// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTransitionExecuteDraftToClarifying verifies that transitioning a draft
// artifact to clarifying succeeds for an analyst-role user (admin@test.local).
// The response must return an ArtifactRow with status "clarifying".
func TestTransitionExecuteDraftToClarifying(t *testing.T) {
	const artifactPath = "lifecycle/requirements/te-draft-clarifying.md"
	env := newTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("TE Draft Clarifying", "ticket", "draft", "te-draft-clarifying", "", "Body."),
	}})

	env.login("admin@test.local", "admin-pass-123")
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]any{"to": "clarifying"})
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	artifact, ok := data["artifact"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'artifact' object in response, got: %v", data)
	}
	if got, _ := artifact["status"].(string); got != "clarifying" {
		t.Errorf("expected status 'clarifying', got %q", got)
	}
}

// TestTransitionExecuteForbiddenRole verifies that a user whose roles do not
// permit the requested transition receives 403 with error.code == "forbidden"
// and a non-empty allowed_targets hint. dev@test.local cannot do draft →
// clarifying (only analyst/product-owner may).
func TestTransitionExecuteForbiddenRole(t *testing.T) {
	const artifactPath = "lifecycle/requirements/te-forbidden-role.md"
	env := newTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("TE Forbidden Role", "ticket", "draft", "te-forbidden-role", "", "Body."),
	}})

	env.login("dev@test.local", "dev-pass-123")
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]any{"to": "clarifying"})
	requireStatus(t, resp, http.StatusForbidden)
	data := readJSON(t, resp)

	errData, _ := data["error"].(map[string]any)
	if code, _ := errData["code"].(string); code != "forbidden" {
		t.Errorf("expected error code 'forbidden', got %q", code)
	}
	if msg, _ := errData["message"].(string); msg == "" {
		t.Error("expected non-empty error message in 403 response")
	}

	// allowed_targets hint must be present and non-empty.
	allowed, ok := data["allowed_targets"].([]any)
	if !ok || len(allowed) == 0 {
		t.Errorf("expected non-empty allowed_targets hint in 403 response, got: %v", data["allowed_targets"])
	}
}

// TestTransitionExecuteInvalidTarget verifies that requesting a transition to a
// status that does not exist in the workflow graph returns 403. The workflow's
// CanTransition returns false for unknown target statuses.
func TestTransitionExecuteInvalidTarget(t *testing.T) {
	const artifactPath = "lifecycle/requirements/te-invalid-target.md"
	env := newTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("TE Invalid Target", "ticket", "draft", "te-invalid-target", "", "Body."),
	}})

	env.login("admin@test.local", "admin-pass-123")
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]any{"to": "nonexistent-status"})
	requireStatus(t, resp, http.StatusForbidden)
	resp.Body.Close()
}

// TestTransitionExecuteNotFound verifies that trying to transition a
// non-existent artifact path returns 404.
func TestTransitionExecuteNotFound(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/artifacts/lifecycle/requirements/does-not-exist.md/transition",
		map[string]any{"to": "clarifying"})
	requireStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

// TestTransitionExecuteDiskUpdate verifies that after a successful transition
// the artifact file on disk has its status frontmatter field updated to the
// new status value.
func TestTransitionExecuteDiskUpdate(t *testing.T) {
	const artifactPath = "lifecycle/requirements/te-disk-update.md"
	env := newTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("TE Disk Update", "ticket", "draft", "te-disk-update", "", "Body."),
	}})

	env.login("admin@test.local", "admin-pass-123")
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]any{"to": "clarifying"})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	raw, err := os.ReadFile(filepath.Join(env.projectRoot, artifactPath))
	if err != nil {
		t.Fatalf("reading artifact from disk: %v", err)
	}
	if !containsLine(string(raw), "status: clarifying") {
		t.Errorf("expected 'status: clarifying' in frontmatter on disk; file contents:\n%s", raw)
	}
}

// TestTransitionExecuteGitCommit verifies that a successful transition records
// a git commit whose first-line message matches the format
// "transition(<lineage>): <from> → <to>".
func TestTransitionExecuteGitCommit(t *testing.T) {
	const artifactPath = "lifecycle/requirements/te-git-commit.md"
	const lineage = "te-git-commit"
	env := newTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("TE Git Commit", "ticket", "draft", lineage, "", "Body."),
	}})

	env.login("admin@test.local", "admin-pass-123")
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]any{"to": "clarifying"})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	commits, err := env.proj.Git.Log(artifactPath, 10)
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	if len(commits) == 0 {
		t.Fatal("expected at least one commit after transition, got none")
	}

	// The most recent commit (index 0) must be the transition commit.
	want := "transition(" + lineage + "): draft → clarifying"
	if !strings.HasPrefix(commits[0].Message, want) {
		t.Errorf("expected commit message prefix %q, got %q", want, commits[0].Message)
	}
}
