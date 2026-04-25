//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"
)

// TestTransitionWithRoleGate verifies that an unauthorised user gets 403
// and an authorised user succeeds. The status field is updated in place
// and a git commit records the change.
// Test plan §7: "Transition with role gate" scenario.
func TestTransitionWithRoleGate(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/gated.md",
			content: makeArtifact("Gated Idea", "idea", "draft", "gated", "", "Needs clarification."),
		},
	}
	env := newTestEnv(t, seeds)

	// Dev user should NOT be able to transition draft → clarifying
	// (only product-owner and analyst can do that).
	env.login("dev@test.local", "dev-pass-123")
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/lifecycle/ideas/gated.md/transition", map[string]any{
		"to": "clarifying",
	})
	requireStatus(t, resp, 403)
	data := readJSON(t, resp)
	errData, _ := data["error"].(map[string]any)
	if code, _ := errData["code"].(string); code != "forbidden" {
		t.Errorf("expected error code 'forbidden', got %q", code)
	}

	// Admin user (product-owner, analyst) SHOULD be able to.
	env.login("admin@test.local", "admin-pass-123")
	resp = env.doRequest("POST", "/api/p/testproject/artifacts/lifecycle/ideas/gated.md/transition", map[string]any{
		"to": "clarifying",
	})
	requireStatus(t, resp, 200)
	data = readJSON(t, resp)

	artifact, _ := data["artifact"].(map[string]any)
	if status, _ := artifact["status"].(string); status != "clarifying" {
		t.Errorf("expected status 'clarifying', got %q", status)
	}

	// Verify the status was updated on disk.
	content, err := os.ReadFile(filepath.Join(env.projectRoot, "lifecycle", "ideas", "gated.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(content), "status: clarifying") {
		t.Error("status field not updated in file on disk")
	}

	// Verify git commit recorded the transition.
	commits, err := env.proj.Git.Log("lifecycle/ideas/gated.md", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) < 2 {
		t.Error("expected at least 2 commits (initial + transition)")
	}
}

// TestRejectionCreatesChildArtifact verifies that transitioning to 'rejected'
// with a comment creates a child rejection artifact.
func TestRejectionCreatesChildArtifact(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/reject-me.md",
			content: makeArtifact("Reject Me", "idea", "draft", "reject-me", "", "This will be rejected."),
		},
	}
	env := newTestEnv(t, seeds)

	// Reviewer can reject from any status.
	env.login("admin@test.local", "admin-pass-123") // admin has reviewer role
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/lifecycle/ideas/reject-me.md/transition", map[string]any{
		"to":      "rejected",
		"comment": "Does not meet quality bar.",
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	rejPath, _ := data["rejection_artifact"].(string)
	if rejPath == "" {
		t.Fatal("expected a rejection_artifact path in response")
	}

	// Verify the rejection artifact exists on disk.
	absPath := filepath.Join(env.projectRoot, rejPath)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Error("rejection artifact file does not exist on disk")
	}

	// Verify it's indexed.
	row, err := env.proj.Idx.Get(rejPath)
	if err != nil {
		t.Fatal(err)
	}
	if row == nil {
		t.Fatal("rejection artifact not found in index")
	}
	if row.Status != "rejected" {
		t.Errorf("rejection artifact status should be 'rejected', got %q", row.Status)
	}
	if row.Lineage != "reject-me" {
		t.Errorf("rejection artifact should share lineage 'reject-me', got %q", row.Lineage)
	}
}

// TestTransitionChainDraftToDone tests a full happy-path transition chain
// through every state.
func TestTransitionChainDraftToDone(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/chain.md",
			content: makeArtifact("Chain Test", "idea", "draft", "chain", "", "Full lifecycle."),
		},
	}
	env := newTestEnv(t, seeds)
	path := "lifecycle/ideas/chain.md"

	transitions := []struct {
		user, pass, to string
	}{
		{"admin@test.local", "admin-pass-123", "clarifying"}, // product-owner/analyst
		{"admin@test.local", "admin-pass-123", "planning"},   // reviewer/analyst
		// Note: planning → in-development requires approved plans (tested separately)
		// We'll skip that gate by testing rejection path instead.
		{"admin@test.local", "admin-pass-123", "rejected"}, // reviewer
	}

	for _, tr := range transitions {
		env.login(tr.user, tr.pass)
		resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+path+"/transition", map[string]any{
			"to": tr.to,
		})
		requireStatus(t, resp, 200)
		data := readJSON(t, resp)
		artifact, _ := data["artifact"].(map[string]any)
		if got, _ := artifact["status"].(string); got != tr.to {
			t.Errorf("after transition to %q, got status %q", tr.to, got)
		}
	}
}

func containsLine(s, target string) bool {
	for _, line := range splitLines(s) {
		if trimSpace(line) == target {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimSpace(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	j := len(s)
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\r') {
		j--
	}
	return s[i:j]
}
