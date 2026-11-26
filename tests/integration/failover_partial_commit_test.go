// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Suite — Milestone 7: restart semantics & the partial-commit race (FR-7).
//
// Before any automatic re-run of an interrupted job, the dispatcher checks
// whether the repository shows a commit at or after the job's StartedAt —
// evidence the run reached its commit step before failing. No such
// evidence -> clean restart (FR-7.2). Evidence found -> the job is held for
// an operator decision instead of being auto-rerun or auto-rolled-back
// (FR-7.3).

import (
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
)

// TestFailover_RestartSemantics_CleanJobRestarts (FR-7.2): a job that fails
// without the fake `claude` ever touching git is restarted automatically —
// requeued at the head of the paused queue — since no partial commit is in
// evidence.
func TestFailover_RestartSemantics_CleanJobRestarts(t *testing.T) {
	markerPath := t.TempDir() + "/clean-restart-invoked"
	errorJSON := `{"error":{"type":"overloaded_error","message":"Overloaded"}}`
	setupFakeClaudeWithScript(t, failoverThenSucceedScript(markerPath, errorJSON))

	providers := []config.Provider{
		{Name: "anthropic-cloud", BaseURL: "http://127.0.0.1:1", Driver: "openai-compatible"},
		{Name: "gemini-cloud", BaseURL: "http://127.0.0.1:1", Driver: "openai-compatible"},
	}

	env := newFailoverTestEnv(t, failoverDisabledCfgYAML, providers, []seedArtifact{
		{relPath: "lifecycle/ideas/clean-idea.md", content: makeApprovedArtifact("Clean Idea", "idea", "clean-idea")},
	})
	beforeSHA := env.headSHA()

	env.enqueue("lifecycle/ideas/clean-idea.md", "requirements-analyst")

	env.waitFor(15*time.Second, "queue to pause after the transient failure", func() bool {
		snap := env.queueSnapshot()
		paused, _ := snap["paused"].(bool)
		return paused
	})

	if got := env.headSHA(); got != beforeSHA {
		t.Fatalf("test setup invariant broken: expected no commit from the fake claude script, HEAD moved from %s to %s", beforeSHA, got)
	}

	snap := env.queueSnapshot()
	j := findJobByPath(snap, "lifecycle/ideas/clean-idea.md")
	if j == nil {
		t.Fatal("expected the failed job to be requeued (clean restart), found none")
	}
	if attempts, _ := j["attempts"].(float64); attempts != 2 {
		t.Errorf("expected the requeued job to have attempts=2, got %v", attempts)
	}
	if j["state"] != "pending" {
		t.Errorf("expected the requeued job to be pending, got state=%v", j["state"])
	}

	statusResp := env.doRequest("GET", "/api/p/testproject/provider-switch/status", nil)
	requireStatus(t, statusResp, 200)
	statusData := readJSON(t, statusResp)
	agents, _ := statusData["agents"].([]any)
	for _, raw := range agents {
		a, _ := raw.(map[string]any)
		if a["agent"] == "requirements-analyst" {
			if awaiting, _ := a["awaiting_decision"].(bool); awaiting {
				t.Error("expected requirements-analyst not to be awaiting an operator decision after a clean restart")
			}
		}
	}
}

// TestFailover_RestartSemantics_PartialCommitSurfaced (FR-7.3): a job whose
// fake `claude` invocation lands a real git commit (simulating work
// committed moments before the process died) before failing must NOT be
// auto-restarted or auto-rolled-back — it is surfaced as
// awaiting-operator-decision instead, and the queue still pauses.
func TestFailover_RestartSemantics_PartialCommitSurfaced(t *testing.T) {
	markerPath := t.TempDir() + "/partial-commit-invoked"
	errorJSON := `{"error":{"type":"overloaded_error","message":"Overloaded"}}`
	script := `if [ -f ` + markerPath + ` ]; then
` + fakeClaudeSuccessEvents + `exit 0
else
touch ` + markerPath + `
git commit --allow-empty -m "partial work landed moments before the process died"
printf '%s\n' '` + errorJSON + `'
exit 0
fi
`
	setupFakeClaudeWithScript(t, script)

	providers := []config.Provider{
		{Name: "anthropic-cloud", BaseURL: "http://127.0.0.1:1", Driver: "openai-compatible"},
		{Name: "gemini-cloud", BaseURL: "http://127.0.0.1:1", Driver: "openai-compatible"},
	}

	env := newFailoverTestEnv(t, failoverDisabledCfgYAML, providers, []seedArtifact{
		{relPath: "lifecycle/ideas/partial-idea.md", content: makeApprovedArtifact("Partial Idea", "idea", "partial-idea")},
	})
	beforeSHA := env.headSHA()

	env.enqueue("lifecycle/ideas/partial-idea.md", "requirements-analyst")

	// queue.* events are broadcast on the app-level hub, not the
	// per-project hub /api/p/{project}/ws subscribes to, so poll the
	// status API (operations.yaml-backed) rather than waiting on a WS event.
	env.waitFor(15*time.Second, "requirements-analyst to be marked awaiting an operator decision", func() bool {
		resp := env.doRequest("GET", "/api/p/testproject/provider-switch/status", nil)
		requireStatus(t, resp, 200)
		data := readJSON(t, resp)
		agents, _ := data["agents"].([]any)
		for _, raw := range agents {
			a, _ := raw.(map[string]any)
			if a["agent"] != "requirements-analyst" {
				continue
			}
			awaiting, _ := a["awaiting_decision"].(bool)
			return awaiting
		}
		return false
	})

	// The script's own commit must have actually landed (test setup
	// invariant), proving the scenario is real, not vacuous.
	if got := env.headSHA(); got == beforeSHA {
		t.Fatal("test setup invariant broken: expected the fake claude script's own commit to land")
	}

	// The queue still pauses (FR-7.3 gates only the automatic restart, not
	// the pause).
	env.waitFor(5*time.Second, "queue to pause", func() bool {
		snap := env.queueSnapshot()
		paused, _ := snap["paused"].(bool)
		return paused
	})

	// No auto-requeue: no second (attempts=2) job for this artifact, in
	// pending or anywhere else — only the original (failed, attempts=1)
	// job exists.
	snap := env.queueSnapshot()
	pending, _ := snap["pending"].([]any)
	for _, raw := range pending {
		j, _ := raw.(map[string]any)
		if j["artifact_path"] == "lifecycle/ideas/partial-idea.md" {
			t.Errorf("expected no auto-requeued pending job for the suspected-partial-commit artifact, found one: %v", j)
		}
	}
	recent, _ := snap["recent"].([]any)
	for _, raw := range recent {
		j, _ := raw.(map[string]any)
		if j["artifact_path"] != "lifecycle/ideas/partial-idea.md" {
			continue
		}
		if attempts, _ := j["attempts"].(float64); attempts != 1 {
			t.Errorf("expected only the original attempt (attempts=1) to be recorded, found attempts=%v", attempts)
		}
	}

	statusResp := env.doRequest("GET", "/api/p/testproject/provider-switch/status", nil)
	requireStatus(t, statusResp, 200)
	statusData := readJSON(t, statusResp)
	agents, _ := statusData["agents"].([]any)
	found := false
	for _, raw := range agents {
		a, _ := raw.(map[string]any)
		if a["agent"] != "requirements-analyst" {
			continue
		}
		found = true
		awaiting, _ := a["awaiting_decision"].(bool)
		if !awaiting {
			t.Error("expected requirements-analyst to be awaiting an operator decision")
		}
		if jobID, _ := a["awaiting_decision_job_id"].(string); jobID == "" {
			t.Error("expected a non-empty awaiting_decision_job_id")
		}
	}
	if !found {
		t.Fatal("expected requirements-analyst in provider-switch/status response")
	}
}
