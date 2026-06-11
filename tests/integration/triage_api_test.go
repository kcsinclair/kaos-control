// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/ideachat"
)

// triageURL returns the POST path for the triage endpoint for a given slug.
func triageURL(slug string) string {
	return "/api/p/testproject/ideas/" + slug + "/triage"
}

// TestTriageAPI_Unauthenticated verifies that the triage endpoint returns
// 401 or 403 when called without a session cookie.
func TestTriageAPI_Unauthenticated(t *testing.T) {
	env := newTriageTestEnv(t)
	// Deliberately no login — no session cookies.

	resp, err := http.Post(env.baseURL+triageURL("anything"), "application/json", nil)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	// CSRF middleware may return 403 before auth check; either is acceptable.
	if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 401 or 403, got %d", resp.StatusCode)
	}
}

// TestTriageAPI_WrongRole verifies that a user without product-owner, analyst,
// or reviewer role receives 403.
func TestTriageAPI_WrongRole(t *testing.T) {
	seeds := []seedArtifact{{
		relPath: "lifecycle/ideas/role-test.md",
		content: makeArtifact("Role Test", "idea", "raw", "role-test", "",
			"This idea tests role enforcement with enough words to qualify."),
	}}
	env := newTriageTestEnvWithSeeds(t, seeds)
	time.Sleep(300 * time.Millisecond)

	// Login as dev@test.local (backend-developer only, no product-owner/analyst/reviewer).
	env.login("dev@test.local", "dev-pass-123")

	resp := env.doRequest("POST", triageURL("role-test"), nil)
	requireStatus(t, resp, http.StatusForbidden)
}

// TestTriageAPI_UnknownSlug verifies that a 404 is returned when the slug
// does not correspond to any idea artifact.
func TestTriageAPI_UnknownSlug(t *testing.T) {
	env := newTriageTestEnv(t)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", triageURL("nope"), nil)
	requireStatus(t, resp, http.StatusNotFound)
	data := readJSON(t, resp)
	if _, ok := data["error"]; !ok {
		t.Error("404 response missing 'error' field")
	}
}

// TestTriageAPI_AlreadyDraft verifies that triggering triage on a draft idea
// returns 409 with reason=wrong_status.
func TestTriageAPI_AlreadyDraft(t *testing.T) {
	seeds := []seedArtifact{{
		relPath: "lifecycle/ideas/already-draft.md",
		content: makeArtifact("Already Draft", "idea", "draft", "already-draft", "",
			"Draft idea body with enough words to qualify."),
	}}
	env := newTriageTestEnvWithSeeds(t, seeds)
	env.login("admin@test.local", "admin-pass-123")
	time.Sleep(300 * time.Millisecond)

	resp := env.doRequest("POST", triageURL("already-draft"), nil)
	requireStatus(t, resp, http.StatusConflict)
	data := readJSON(t, resp)
	if reason, _ := data["reason"].(string); reason != "wrong_status" {
		t.Errorf("expected reason 'wrong_status', got %q; full body: %v", reason, data)
	}
}

// TestTriageAPI_WrongType verifies that triggering triage on a raw defect
// under lifecycle/ideas/ returns 409 with reason=wrong_type.
func TestTriageAPI_WrongType(t *testing.T) {
	seeds := []seedArtifact{{
		relPath: "lifecycle/ideas/raw-defect.md",
		content: makeArtifact("Raw Defect", "defect", "raw", "raw-defect", "",
			"Defect body with enough words to qualify if it were an idea."),
	}}
	env := newTestEnvWithCfgYAML(t, seeds, triageCfgYAML)
	env.login("admin@test.local", "admin-pass-123")
	time.Sleep(300 * time.Millisecond)

	resp := env.doRequest("POST", triageURL("raw-defect"), nil)
	requireStatus(t, resp, http.StatusConflict)
	data := readJSON(t, resp)
	if reason, _ := data["reason"].(string); reason != "wrong_type" {
		t.Errorf("expected reason 'wrong_type', got %q; full body: %v", reason, data)
	}
}

// TestTriageAPI_Success verifies that triggering triage on a raw idea returns
// 202 with a run_id, and the run completes with status=done within 5 s.
func TestTriageAPI_Success(t *testing.T) {
	installLLMFake(t, []string{defaultProposeJSON("api-success", "API Success", nil)})

	seeds := []seedArtifact{{
		relPath: "lifecycle/ideas/api-success.md",
		content: makeArtifact("API Success", "idea", "raw", "api-success", "",
			"Raw idea for success path test with enough words to qualify."),
	}}
	env := newTestEnvWithCfgYAML(t, seeds, triageCfgYAML)
	env.login("admin@test.local", "admin-pass-123")
	time.Sleep(300 * time.Millisecond)

	resp := env.doRequest("POST", triageURL("api-success"), nil)
	requireStatus(t, resp, http.StatusAccepted)
	data := readJSON(t, resp)

	runID, _ := data["run_id"].(string)
	if runID == "" {
		t.Fatal("202 response missing non-empty run_id")
	}

	// Poll for the run to complete.
	run := pollForRunStatus(t, env, "lifecycle/ideas/api-success.md", "done", 5*time.Second)
	if run == nil {
		// Also accept the run completing synchronously.
		run = pollForRunStatus(t, env, "lifecycle/ideas/api-success.md", "failed", 1*time.Second)
		if run != nil {
			t.Errorf("run failed instead of succeeding; run: %v", run)
		} else {
			t.Error("run did not complete within 5s")
		}
	}
}

// TestTriageAPI_InFlightCoalesce verifies that two rapid calls to the triage
// endpoint for the same path coalesce onto a single run with the same run_id.
func TestTriageAPI_InFlightCoalesce(t *testing.T) {
	block := make(chan struct{})
	var unblockOnce sync.Once
	unblock := func() { unblockOnce.Do(func() { close(block) }) }
	defer unblock()

	orig := ideachat.CallLLM
	t.Cleanup(func() {
		unblock()
		time.Sleep(100 * time.Millisecond)
		ideachat.CallLLM = orig
	})
	ideachat.CallLLM = func(ctx context.Context, cfg ideachat.ModelConfig, msgs []ideachat.LLMMessage) (string, error) {
		select {
		case <-block:
			return defaultProposeJSON("coalesce", "Coalesce Idea", nil), nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	seeds := []seedArtifact{{
		relPath: "lifecycle/ideas/coalesce.md",
		content: makeArtifact("Coalesce Idea", "idea", "raw", "coalesce", "",
			"Raw idea for coalesce test with enough words to qualify."),
	}}
	env := newTestEnvWithCfgYAML(t, seeds, triageCfgYAML)
	env.login("admin@test.local", "admin-pass-123")
	time.Sleep(300 * time.Millisecond)

	// First call starts the run; LLM blocks so the run stays in-flight.
	resp1 := env.doRequest("POST", triageURL("coalesce"), nil)
	requireStatus(t, resp1, http.StatusAccepted)
	data1 := readJSON(t, resp1)
	runID1, _ := data1["run_id"].(string)
	if runID1 == "" {
		t.Fatal("first POST missing run_id")
	}

	// Brief pause to ensure the goroutine is in-flight.
	time.Sleep(20 * time.Millisecond)

	// Second call should coalesce.
	resp2 := env.doRequest("POST", triageURL("coalesce"), nil)
	requireStatus(t, resp2, http.StatusAccepted)
	data2 := readJSON(t, resp2)
	runID2, _ := data2["run_id"].(string)

	if runID1 != runID2 {
		t.Errorf("expected coalesced run IDs to match; got %q and %q", runID1, runID2)
	}

	// Verify only one agent_runs row for this path.
	unblock()
	time.Sleep(500 * time.Millisecond) // wait for run to finish

	runs, err := env.proj.Idx.ListAgentRunsByTargetPath("lifecycle/ideas/coalesce.md")
	if err != nil {
		t.Fatalf("ListAgentRunsByTargetPath: %v", err)
	}
	if len(runs) != 1 {
		t.Errorf("expected exactly 1 agent_runs row, got %d", len(runs))
	}
}

// TestTriageAPI_LockedLineage verifies that triggering triage on a locked
// lineage returns 409 with error=locked.
func TestTriageAPI_LockedLineage(t *testing.T) {
	seeds := []seedArtifact{{
		relPath: "lifecycle/ideas/locked-idea.md",
		content: makeArtifact("Locked Idea", "idea", "raw", "locked-idea", "",
			"Raw idea for lock test with enough words to qualify."),
	}}
	env := newTestEnvWithCfgYAML(t, seeds, triageCfgYAML)
	env.login("admin@test.local", "admin-pass-123")
	time.Sleep(300 * time.Millisecond)

	// Pre-acquire the lineage lock.
	if _, err := env.proj.Locks.Acquire("locked-idea", "test-holder", "agent"); err != nil {
		t.Fatalf("Acquire lock: %v", err)
	}
	t.Cleanup(func() { _ = env.proj.Locks.Release("locked-idea") })

	resp := env.doRequest("POST", triageURL("locked-idea"), nil)
	requireStatus(t, resp, http.StatusConflict)
	data := readJSON(t, resp)
	if errStr, _ := data["error"].(string); errStr != "locked" {
		t.Errorf("expected error 'locked', got %q; full body: %v", errStr, data)
	}
}
