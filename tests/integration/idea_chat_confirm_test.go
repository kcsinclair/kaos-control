//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Milestone 3 – Proposal Accept / Reject Tests
//
// Most tests here require ANTHROPIC_API_KEY because they first drive the
// conversation to a "proposed" state before testing the confirmation flow.
// TestIdeaChatAcceptWithoutProposal is the exception and runs without an API key.

// slugPattern is the regex a valid lineage slug must match.
var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]*[a-z0-9]$|^[a-z0-9]$`)

// TestIdeaChatAcceptWithoutProposal verifies that sending __accept__ when no
// proposal exists returns HTTP 409 with code "no_proposal".
// Does NOT require ANTHROPIC_API_KEY: a new session (status=conversing) is
// created implicitly by the __accept__ message.
func TestIdeaChatAcceptWithoutProposal(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Omit session_id → new session created with status=conversing.
	// __accept__ on a non-proposed session must return 409.
	resp := env.doRequest("POST", "/api/p/testproject/ideas/converse", map[string]any{
		"message": "__accept__",
	})
	requireStatus(t, resp, 409)
	data := readJSON(t, resp)

	errData, _ := data["error"].(map[string]any)
	if code, _ := errData["code"].(string); code != "no_proposal" {
		t.Errorf("expected error code 'no_proposal', got %q", code)
	}
}

// TestIdeaChatAcceptCreatesArtifact verifies that sending __accept__ after
// reaching "proposed" state returns status "created" with a non-null
// artifact_path matching lifecycle/ideas/<slug>.md.
func TestIdeaChatAcceptCreatesArtifact(t *testing.T) {
	skipIfNoAPIKey(t)
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	sessionID, _ := convergeToProposal(t, env, uniqueIdeaMessage("accept-creates"))

	resp := converseAPI(env, sessionID, "__accept__")
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	status, _ := data["status"].(string)
	if status != "created" {
		t.Errorf("expected status 'created', got %q", status)
	}

	artifactPath, _ := data["artifact_path"].(string)
	if artifactPath == "" {
		t.Fatal("expected non-null artifact_path in created response")
	}
	if !strings.HasPrefix(artifactPath, "lifecycle/ideas/") {
		t.Errorf("artifact_path %q should start with 'lifecycle/ideas/'", artifactPath)
	}
	if !strings.HasSuffix(artifactPath, ".md") {
		t.Errorf("artifact_path %q should end with '.md'", artifactPath)
	}
}

// TestIdeaChatArtifactFileExistsOnDisk verifies that after accepting a proposal
// the file at artifact_path exists on disk with correct frontmatter fields.
func TestIdeaChatArtifactFileExistsOnDisk(t *testing.T) {
	skipIfNoAPIKey(t)
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	sessionID, _ := convergeToProposal(t, env, uniqueIdeaMessage("disk-check"))

	resp := converseAPI(env, sessionID, "__accept__")
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	artifactPath, _ := data["artifact_path"].(string)
	if artifactPath == "" {
		t.Fatal("missing artifact_path in response")
	}

	absPath := filepath.Join(env.projectRoot, artifactPath)
	content, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("artifact file does not exist on disk at %s: %v", absPath, err)
	}
	text := string(content)

	// Must have type: idea
	if !strings.Contains(text, "type: idea") {
		t.Errorf("artifact file missing 'type: idea' in frontmatter")
	}
	// Must have status: draft
	if !strings.Contains(text, "status: draft") {
		t.Errorf("artifact file missing 'status: draft' in frontmatter")
	}
	// Lineage must match the slug from the path.
	slug := slugFromPath(artifactPath)
	if !strings.Contains(text, "lineage: "+slug) {
		t.Errorf("artifact file missing 'lineage: %s' in frontmatter", slug)
	}
}

// TestIdeaChatArtifactAppearsInIndex verifies that after accepting, a GET to
// /artifacts?lineage=<slug> returns the newly created artifact.
func TestIdeaChatArtifactAppearsInIndex(t *testing.T) {
	skipIfNoAPIKey(t)
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	sessionID, _ := convergeToProposal(t, env, uniqueIdeaMessage("index-check"))

	resp := converseAPI(env, sessionID, "__accept__")
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	artifactPath, _ := data["artifact_path"].(string)
	if artifactPath == "" {
		t.Fatal("missing artifact_path in response")
	}
	slug := slugFromPath(artifactPath)

	// The watcher may not have indexed yet – poll briefly.
	var found bool
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		listResp := env.doRequest("GET", "/api/p/testproject/artifacts?lineage="+slug, nil)
		requireStatus(t, listResp, 200)
		listData := readJSON(t, listResp)

		items, _ := listData["items"].([]any)
		if len(items) > 0 {
			found = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !found {
		t.Errorf("artifact with lineage %q not found in index after accept", slug)
	}
}

// TestIdeaChatSessionDeletedAfterCreation verifies that after a successful
// __accept__, the session is cleaned up: a subsequent message with the same
// session_id returns HTTP 404.
func TestIdeaChatSessionDeletedAfterCreation(t *testing.T) {
	skipIfNoAPIKey(t)
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	sessionID, _ := convergeToProposal(t, env, uniqueIdeaMessage("session-deleted"))

	// Accept the proposal.
	resp := converseAPI(env, sessionID, "__accept__")
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	if status, _ := data["status"].(string); status != "created" {
		t.Fatalf("accept response status %q, want 'created'", status)
	}

	// Now the session must be gone.
	resp2 := converseAPI(env, sessionID, "any message")
	requireStatus(t, resp2, 404)
	data2 := readJSON(t, resp2)
	errData, _ := data2["error"].(map[string]any)
	if code, _ := errData["code"].(string); code != "session_not_found" {
		t.Errorf("expected code 'session_not_found' after session deletion, got %q", code)
	}
}

// TestIdeaChatRejectDiscardsSession verifies that sending __reject__ after
// reaching "proposed" returns status "conversing" with session_id: null,
// and a subsequent request with the old session_id returns 404.
func TestIdeaChatRejectDiscardsSession(t *testing.T) {
	skipIfNoAPIKey(t)
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	sessionID, _ := convergeToProposal(t, env, uniqueIdeaMessage("reject-test"))

	resp := converseAPI(env, sessionID, "__reject__")
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	status, _ := data["status"].(string)
	if status != "conversing" {
		t.Errorf("reject: expected status 'conversing', got %q", status)
	}
	// The spec says session_id should be null after reject.
	returnedSID := data["session_id"]
	if returnedSID != nil {
		t.Errorf("reject: expected session_id to be null, got %v", returnedSID)
	}

	// Old session must no longer exist.
	resp2 := converseAPI(env, sessionID, "follow-up message")
	requireStatus(t, resp2, 404)
	resp2.Body.Close()
}

// --- helpers specific to confirm tests ---

// uniqueIdeaMessage builds a long, unique, detailed message so convergeToProposal
// is likely to succeed in one turn even with a cold session. The tag parameter
// keeps messages distinct across parallel tests.
func uniqueIdeaMessage(tag string) string {
	return "I want to build a feature called " + tag + ". " +
		"It should allow users to export a complete snapshot of all lifecycle artifacts " +
		"in a project to a structured ZIP archive. The archive would contain one folder " +
		"per stage (ideas, requirements, plans, etc.) with the markdown files preserved " +
		"as-is. Useful for audits, backups, and sharing project state with external reviewers " +
		"who do not have access to the live system. The download should be triggered by a " +
		"button in the project settings page."
}

// slugFromPath extracts the slug (filename without .md) from a lifecycle path
// like lifecycle/ideas/my-slug.md.
func slugFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, ".md")
}
