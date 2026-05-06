//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// TestAnalystRequirementsActivatesStatus verifies that starting an
// requirements-analyst run synchronously sets the target idea artifact's status
// to "clarifying" and commits the change with the expected message pattern.
// Covers test plan Milestone 1.
func TestAnalystRequirementsActivatesStatus(t *testing.T) {
	setupFakeClaude(t, 0)

	const artifactPath = "lifecycle/ideas/activate-req.md"
	env := newAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Activate Req Test", "idea", "draft", "activate-req", "", "Idea body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	startAgentRun(t, env, "requirements-analyst", artifactPath)

	// Status change is synchronous (happens in StartRun before the driver
	// process is spawned) — check immediately without waiting for completion.
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, artifactPath))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(raw), "status: clarifying") {
		t.Errorf("expected status: clarifying on disk after StartRun; got:\n%s", raw)
	}

	// A git commit must exist with the pattern:
	//   status(activate-req): draft → clarifying [run:<hex>]
	commits, err := env.proj.Git.Log(artifactPath, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) < 2 {
		t.Fatalf("expected at least 2 commits (initial + status change), got %d", len(commits))
	}
	pattern := regexp.MustCompile(`status\(activate-req\): draft → clarifying \[run:[0-9a-f]+\]`)
	if !pattern.MatchString(commits[0].Message) {
		t.Errorf("latest commit message %q does not match pattern %s",
			commits[0].Message, pattern)
	}
}

// TestAnalystPlannerActivatesStatus verifies that starting an planning-analyst
// run synchronously sets the target requirement artifact's status to "planning"
// and commits the change.
// Covers test plan Milestone 2.
func TestAnalystPlannerActivatesStatus(t *testing.T) {
	setupFakeClaude(t, 0)

	const artifactPath = "lifecycle/requirements/activate-plan-2.md"
	env := newAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Activate Plan Test", "ticket", "clarifying", "activate-plan", "", "Requirement body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	startAgentRun(t, env, "planning-analyst", artifactPath)

	raw, err := os.ReadFile(filepath.Join(env.projectRoot, artifactPath))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(raw), "status: planning") {
		t.Errorf("expected status: planning on disk after StartRun; got:\n%s", raw)
	}

	commits, err := env.proj.Git.Log(artifactPath, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) < 2 {
		t.Fatalf("expected at least 2 commits, got %d", len(commits))
	}
	pattern := regexp.MustCompile(`status\(activate-plan\): clarifying → planning \[run:[0-9a-f]+\]`)
	if !pattern.MatchString(commits[0].Message) {
		t.Errorf("latest commit message %q does not match pattern %s",
			commits[0].Message, pattern)
	}
}

// TestAnalystStatusPersistsAfterSuccess verifies that after an analyst agent
// exits successfully (exit 0) with done_on_success unset, the target artifact
// retains its active status (clarifying) — it is NOT reverted to draft.
// Covers test plan Milestone 3, scenario 1.
func TestAnalystStatusPersistsAfterSuccess(t *testing.T) {
	setupFakeClaude(t, 0)

	const artifactPath = "lifecycle/ideas/persist-success.md"
	env := newAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Persist Success Test", "idea", "draft", "persist-success", "", "Idea body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env, "requirements-analyst", artifactPath)
	run := waitForRunCompletion(t, env, runID)

	if got, _ := run["status"].(string); got != "done" {
		t.Errorf("expected run record status 'done', got %q", got)
	}

	raw, err := os.ReadFile(filepath.Join(env.projectRoot, artifactPath))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(raw), "status: clarifying") {
		t.Errorf("expected status: clarifying to persist after successful run; got:\n%s", raw)
	}
}

// TestAnalystStatusSetsDoneAfterSuccess verifies that when done_on_success is
// true and the agent exits successfully, the target artifact's status is set
// to "done".
// Covers test plan Milestone 3, scenario 2.
func TestAnalystStatusSetsDoneAfterSuccess(t *testing.T) {
	setupFakeClaude(t, 0)

	// stub-done-agent has active_status=in-development and done_on_success=true.
	const artifactPath = "lifecycle/ideas/done-on-success.md"
	env := newAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Done On Success Test", "idea", "draft", "done-on-success", "", "Idea body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env, "stub-done-agent", artifactPath)
	run := waitForRunCompletion(t, env, runID)

	if got, _ := run["status"].(string); got != "done" {
		t.Errorf("expected run record status 'done', got %q", got)
	}

	raw, err := os.ReadFile(filepath.Join(env.projectRoot, artifactPath))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(raw), "status: done") {
		t.Errorf("expected status: done after successful run with done_on_success=true; got:\n%s", raw)
	}
}

// TestAnalystStatusPersistsAfterFailure verifies that when an analyst agent
// exits with a non-zero code the target artifact retains its active status
// (clarifying) — it is NOT reverted to the prior status (draft).
// Covers test plan Milestone 3, scenario 3.
func TestAnalystStatusPersistsAfterFailure(t *testing.T) {
	setupFakeClaude(t, 1) // non-zero exit → failure

	const artifactPath = "lifecycle/ideas/persist-failure.md"
	env := newAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Persist Failure Test", "idea", "draft", "persist-failure", "", "Idea body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env, "requirements-analyst", artifactPath)
	run := waitForRunCompletion(t, env, runID)

	if got, _ := run["status"].(string); got != "failed" {
		t.Errorf("expected run record status 'failed', got %q", got)
	}

	raw, err := os.ReadFile(filepath.Join(env.projectRoot, artifactPath))
	if err != nil {
		t.Fatal(err)
	}
	// Status must be clarifying (the active value) — NOT reverted to draft.
	if !containsLine(string(raw), "status: clarifying") {
		t.Errorf("expected status: clarifying to persist after failed run; got:\n%s", raw)
	}
	if containsLine(string(raw), "status: draft") {
		t.Error("status must NOT be reverted to draft after failure")
	}
}
