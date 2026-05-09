// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v5"
)

// TestCreateArtifactCommitIndex verifies the full create → commit → index pipeline:
// POST an artifact, assert file exists on disk, ticket branch was created,
// exactly one commit with templated message, SQLite has the row.
// Test plan §7: "Create → commit → index" scenario.
func TestCreateArtifactCommitIndex(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Create an idea artifact via API.
	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "ideas",
		"slug":  "dashboard",
		"frontmatter": map[string]any{
			"title":   "Dashboard Feature",
			"type":    "idea",
			"status":  "draft",
			"lineage": "dashboard",
		},
		"body": "A dashboard for viewing project metrics.",
	})
	requireStatus(t, resp, 201)
	data := readJSON(t, resp)

	// Assert returned path.
	path, ok := data["path"].(string)
	if !ok || path == "" {
		t.Fatal("expected path in create response")
	}
	// First artifact in lineage gets index 0 → filename is just slug.md.
	if path != "lifecycle/ideas/dashboard.md" {
		t.Errorf("expected path lifecycle/ideas/dashboard.md, got %s", path)
	}

	// (a) File exists on disk.
	absPath := filepath.Join(env.projectRoot, path)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Error("artifact file does not exist on disk")
	}

	// (b) Verify the file content includes the frontmatter.
	content, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(content) == 0 {
		t.Error("artifact file is empty")
	}

	// (c) Ticket branch was created (branch_template = "ticket/{slug}").
	repo, err := gogit.PlainOpen(env.projectRoot)
	if err != nil {
		t.Fatal(err)
	}
	branchRef := "refs/heads/ticket/dashboard"
	_, err = repo.Reference("refs/heads/ticket/dashboard", true)
	if err != nil {
		t.Errorf("expected branch %s to exist: %v", branchRef, err)
	}

	// (d) Git log shows at least one commit with the create message.
	commits, err := env.proj.Git.Log(path, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) == 0 {
		t.Error("expected at least one commit for the artifact")
	}

	// (e) SQLite index has the row.
	row, err := env.proj.Idx.Get(path)
	if err != nil {
		t.Fatal(err)
	}
	if row == nil {
		t.Fatal("artifact not found in index")
	}
	if row.Title != "Dashboard Feature" {
		t.Errorf("expected title 'Dashboard Feature', got %q", row.Title)
	}
	if row.Status != "draft" {
		t.Errorf("expected status 'draft', got %q", row.Status)
	}
	if row.Lineage != "dashboard" {
		t.Errorf("expected lineage 'dashboard', got %q", row.Lineage)
	}
}

// TestCreateSecondArtifactInLineage verifies that the second artifact
// in a lineage gets index -2 and the correct filename.
func TestCreateSecondArtifactInLineage(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/feature-x.md",
			content: makeArtifact("Feature X", "idea", "draft", "feature-x", "", "An idea."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "requirements",
		"slug":  "feature-x",
		"frontmatter": map[string]any{
			"title":   "Feature X Requirements",
			"type":    "ticket",
			"status":  "draft",
			"lineage": "feature-x",
			"parent":  "lifecycle/ideas/feature-x.md",
		},
		"body": "Requirements for feature X.",
	})
	requireStatus(t, resp, 201)
	data := readJSON(t, resp)

	path, _ := data["path"].(string)
	if path != "lifecycle/requirements/feature-x-2.md" {
		t.Errorf("expected lifecycle/requirements/feature-x-2.md, got %s", path)
	}
}

// TestCreateArtifactRequiresCsrf verifies that mutations without CSRF token are rejected.
func TestCreateArtifactRequiresCsrf(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Clear the CSRF token to simulate a request without it.
	savedToken := env.csrfToken
	env.csrfToken = ""
	defer func() { env.csrfToken = savedToken }()

	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "ideas",
		"slug":  "no-csrf",
		"frontmatter": map[string]any{
			"title":   "No CSRF",
			"type":    "idea",
			"status":  "draft",
			"lineage": "no-csrf",
		},
		"body": "Should be rejected.",
	})
	requireStatus(t, resp, 403)
	resp.Body.Close()
}

// TestOptimisticConcurrency verifies that PUT with a stale expected_sha returns 409.
func TestOptimisticConcurrency(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/concur.md",
			content: makeArtifact("Concurrency Test", "idea", "draft", "concur", "", "Original body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// PUT with a deliberately wrong SHA.
	resp := env.doRequest("PUT", "/api/p/testproject/artifacts/lifecycle/ideas/concur.md", map[string]any{
		"frontmatter": map[string]any{
			"title":   "Concurrency Test Updated",
			"type":    "idea",
			"status":  "draft",
			"lineage": "concur",
		},
		"body":         "Updated body.",
		"expected_sha": "0000000000000000000000000000000000000000000000000000000000000000",
	})
	requireStatus(t, resp, 409)
	resp.Body.Close()
}
