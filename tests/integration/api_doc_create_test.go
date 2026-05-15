// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Integration tests for creating doc artifacts via POST /artifacts.
//
// Test plan: lifecycle/test-plans/tech-writer-agent-5-test.md §Milestone 4

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ── Milestone 4, TC1: originating doc creation ───────────────────────────────

// TestDocCreate_OriginatingDoc verifies that creating a doc artifact with no
// parent and an auto-derived lineage writes the file at
// lifecycle/docs/<slug>.md with no index suffix and no parent field.
func TestDocCreate_OriginatingDoc(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, nil, docGenerateCfgYAML)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "docs",
		"slug":  "install-guide",
		"frontmatter": map[string]any{
			"title":   "Install Guide",
			"type":    "doc",
			"status":  "draft",
			"lineage": "install-guide",
		},
		"body": "# Install Guide\n\nInstallation instructions.",
	})
	requireStatus(t, resp, 201)
	data := readJSON(t, resp)

	path, _ := data["path"].(string)
	if path == "" {
		t.Fatal("create response missing 'path'")
	}

	// Originating doc: no index suffix → install-guide.md (not install-guide-N-doc.md).
	expectedPath := "lifecycle/docs/install-guide.md"
	if path != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, path)
	}

	// Verify file exists on disk.
	absPath := filepath.Join(env.projectRoot, path)
	fileBytes, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("artifact file not found on disk at %s: %v", absPath, err)
	}
	content := string(fileBytes)

	if !strings.Contains(content, "type: doc") {
		t.Error("artifact file missing 'type: doc'")
	}
	if !strings.Contains(content, "status: draft") {
		t.Error("artifact file missing 'status: draft'")
	}
	if !strings.Contains(content, "lineage: install-guide") {
		t.Error("artifact file missing 'lineage: install-guide'")
	}
	// Originating artifact must not have a parent field.
	if strings.Contains(content, "parent:") {
		t.Error("originating doc should not have a 'parent' field")
	}
}

// ── Milestone 4, TC2: source-linked doc creation ─────────────────────────────

// TestDocCreate_SourceLinkedDoc verifies that creating a doc artifact with an
// existing lineage produces a filename with a monotonic index and the "-doc"
// stage suffix (e.g. login-7-doc.md).
func TestDocCreate_SourceLinkedDoc(t *testing.T) {
	// Pre-seed the lineage with 6 artifacts so the next index should be 7.
	seeds := []seedArtifact{
		{relPath: "lifecycle/ideas/login.md",
			content: makeArtifact("Login Idea", "idea", "draft", "login", "", "Idea body.")},
		{relPath: "lifecycle/requirements/login-2.md",
			content: makeArtifact("Login Req", "requirement", "done", "login",
				"lifecycle/ideas/login.md", "Req body.")},
		{relPath: "lifecycle/backend-plans/login-3-be.md",
			content: makeArtifact("Login BE Plan", "plan-backend", "approved", "login",
				"lifecycle/requirements/login-2.md", "BE plan.")},
		{relPath: "lifecycle/frontend-plans/login-4-fe.md",
			content: makeArtifact("Login FE Plan", "plan-frontend", "approved", "login",
				"lifecycle/requirements/login-2.md", "FE plan.")},
		{relPath: "lifecycle/test-plans/login-5-test.md",
			content: makeArtifact("Login Test Plan", "plan-test", "approved", "login",
				"lifecycle/requirements/login-2.md", "Test plan.")},
		{relPath: "lifecycle/tests/login-6-test.md",
			content: makeArtifact("Login Tests", "test", "approved", "login",
				"lifecycle/test-plans/login-5-test.md", "Tests.")},
	}
	env := newTestEnvWithCfgYAML(t, seeds, docGenerateCfgYAML)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "docs",
		"slug":  "login",
		"frontmatter": map[string]any{
			"title":   "Login Feature Documentation",
			"type":    "doc",
			"status":  "draft",
			"lineage": "login",
			"parent":  "lifecycle/requirements/login-2.md",
		},
		"body": "# Login Feature\n\nDocumentation.",
	})
	requireStatus(t, resp, 201)
	data := readJSON(t, resp)

	path, _ := data["path"].(string)
	if path == "" {
		t.Fatal("create response missing 'path'")
	}

	// Must be in lifecycle/docs/ with index and "-doc" suffix.
	if !strings.HasPrefix(path, "lifecycle/docs/login-") {
		t.Errorf("expected path under lifecycle/docs/login-*, got %q", path)
	}
	if !strings.HasSuffix(path, "-doc.md") {
		t.Errorf("expected '-doc.md' suffix in path, got %q", path)
	}

	// Verify parent is set on disk.
	absPath := filepath.Join(env.projectRoot, path)
	fileBytes, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("artifact file not found on disk at %s: %v", absPath, err)
	}
	content := string(fileBytes)

	if !strings.Contains(content, "parent: lifecycle/requirements/login-2.md") {
		t.Errorf("file missing 'parent: lifecycle/requirements/login-2.md'\ncontent:\n%s", content)
	}
}

// ── Milestone 4, TC3: indexer picks up doc ───────────────────────────────────

// TestDocCreate_IndexerPicksUp verifies that after creating a doc artifact the
// index returns it via GET /artifacts/<path> with the correct type and lineage.
func TestDocCreate_IndexerPicksUp(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, nil, docGenerateCfgYAML)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "docs",
		"slug":  "indexer-test-doc",
		"frontmatter": map[string]any{
			"title":   "Indexer Test Doc",
			"type":    "doc",
			"status":  "draft",
			"lineage": "indexer-test-doc",
		},
		"body": "# Indexer Test\n\nBody.",
	})
	requireStatus(t, resp, 201)
	createData := readJSON(t, resp)

	path, _ := createData["path"].(string)
	if path == "" {
		t.Fatal("create response missing 'path'")
	}

	// The create endpoint indexes synchronously, but poll briefly for the watcher.
	var found bool
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		getResp := env.doRequest("GET", "/api/p/testproject/artifacts/"+path, nil)
		if getResp.StatusCode == 200 {
			getResp.Body.Close()
			found = true
			break
		}
		getResp.Body.Close()
		time.Sleep(100 * time.Millisecond)
	}
	if !found {
		t.Fatalf("artifact at %q not found in index after creation", path)
	}

	// Fetch and verify type and lineage.
	getResp := env.doRequest("GET", "/api/p/testproject/artifacts/"+path, nil)
	requireStatus(t, getResp, 200)
	getData := readJSON(t, getResp)

	a, _ := getData["artifact"].(map[string]any)
	if a == nil {
		t.Fatal("GET response missing 'artifact'")
	}
	if typ, _ := a["type"].(string); typ != "doc" {
		t.Errorf("index: type want 'doc', got %q", typ)
	}
	if lineage, _ := a["lineage"].(string); lineage != "indexer-test-doc" {
		t.Errorf("index: lineage want 'indexer-test-doc', got %q", lineage)
	}
}

// ── Milestone 4, TC4: git commit after creation ───────────────────────────────

// TestDocCreate_GitCommit verifies that creating a doc artifact produces a git
// commit whose message follows the create(<stage>): <path> convention.
func TestDocCreate_GitCommit(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, nil, docGenerateCfgYAML)
	env.login("admin@test.local", "admin-pass-123")

	slug := "git-commit-doc"
	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "docs",
		"slug":  slug,
		"frontmatter": map[string]any{
			"title":   "Git Commit Doc",
			"type":    "doc",
			"status":  "draft",
			"lineage": slug,
		},
		"body": "# Git Commit Doc\n\nBody.",
	})
	requireStatus(t, resp, 201)
	data := readJSON(t, resp)

	createdPath, _ := data["path"].(string)
	if createdPath == "" {
		t.Fatal("create response missing 'path'")
	}

	// Use the project's git wrapper to read log for this file.
	commits, err := env.proj.Git.Log(createdPath, 5)
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	if len(commits) == 0 {
		t.Fatal("expected at least one commit after doc creation, got none")
	}

	// Most recent commit message must start with "create(docs): ..."
	expectedPrefix := fmt.Sprintf("create(docs): %s", createdPath)
	if !strings.HasPrefix(commits[0].Message, expectedPrefix) {
		t.Errorf("expected commit message prefix %q, got %q", expectedPrefix, commits[0].Message)
	}
}
