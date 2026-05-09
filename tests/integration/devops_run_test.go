// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestone 3 — Pipeline Execution & Concurrency Tests
//
// Tests for POST /api/p/testproject/devops/pipelines/{slug}/run covering:
//   - Role-based access (product-owner, devops, wrong role, unauthenticated)
//   - Pipeline not found (404)
//   - Already running (409 conflict)
//   - Re-triggering after completion (no stale lock)
//   - Multiple different pipelines can run concurrently

import (
	"net/http"
	"sync"
	"testing"
	"time"
)

// TestDevopsRun_ProductOwnerSucceeds verifies that a product-owner can start a
// pipeline run and receives a valid run_id in return.
func TestDevopsRun_ProductOwnerSucceeds(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/quick-pass/run", nil)
	requireStatus(t, resp, http.StatusAccepted)
	data := readJSON(t, resp)

	runID, _ := data["run_id"].(string)
	if runID == "" {
		t.Error("expected non-empty run_id in response")
	}
	if len(runID) != 16 {
		// run IDs are 8-byte hex strings (16 chars)
		t.Errorf("run_id %q has unexpected length %d, want 16", runID, len(runID))
	}

	// Wait for run to complete so cleanup is clean.
	waitForRunComplete(t, env, "quick-pass", 10*time.Second)
}

// TestDevopsRun_DevopsRoleSucceeds verifies that a devops-role user can also
// trigger pipeline runs.
func TestDevopsRun_DevopsRoleSucceeds(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("dev@test.local", "dev-pass-123")

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/quick-pass/run", nil)
	requireStatus(t, resp, http.StatusAccepted)
	data := readJSON(t, resp)

	if runID, _ := data["run_id"].(string); runID == "" {
		t.Error("expected non-empty run_id")
	}

	waitForRunComplete(t, env, "quick-pass", 10*time.Second)
}

// TestDevopsRun_ForbiddenRole verifies that a user without the devops or
// product-owner role receives a 403 when attempting to run a pipeline.
func TestDevopsRun_ForbiddenRole(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("qa@test.local", "qa-pass-123")

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/quick-pass/run", nil)
	requireStatus(t, resp, http.StatusForbidden)
	resp.Body.Close()
}

// TestDevopsRun_NotFound verifies that running a non-existent pipeline returns 404.
func TestDevopsRun_NotFound(t *testing.T) {
	env := newDevopsTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/nonexistent/run", nil)
	requireStatus(t, resp, http.StatusNotFound)
	data := readJSON(t, resp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "not_found" {
		t.Errorf("expected error code 'not_found', got %q", code)
	}
}

// TestDevopsRun_AlreadyRunning verifies that triggering a run on an
// already-active pipeline returns 409 Conflict.
func TestDevopsRun_AlreadyRunning(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"slow-step.yaml": pipelineSlowStep,
	})
	env.login("admin@test.local", "admin-pass-123")

	// Start the slow pipeline.
	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/slow-step/run", nil)
	requireStatus(t, resp, http.StatusAccepted)
	resp.Body.Close()

	// Immediately try to start it again — should conflict.
	resp2 := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/slow-step/run", nil)
	requireStatus(t, resp2, http.StatusConflict)
	data := readJSON(t, resp2)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "conflict" {
		t.Errorf("expected error code 'conflict', got %q", code)
	}

	// Cancel to clean up.
	cancelResp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/slow-step/cancel", nil)
	cancelResp.Body.Close()
	waitForRunComplete(t, env, "slow-step", 10*time.Second)
}

// TestDevopsRun_ReTriggerAfterCompletion verifies that after a run finishes,
// the same pipeline can be triggered again (no stale lock remains).
func TestDevopsRun_ReTriggerAfterCompletion(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	// First run.
	resp1 := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/quick-pass/run", nil)
	requireStatus(t, resp1, http.StatusAccepted)
	data1 := readJSON(t, resp1)
	runID1, _ := data1["run_id"].(string)

	waitForRunComplete(t, env, "quick-pass", 10*time.Second)

	// Second run — must succeed, not conflict.
	resp2 := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/quick-pass/run", nil)
	requireStatus(t, resp2, http.StatusAccepted)
	data2 := readJSON(t, resp2)
	runID2, _ := data2["run_id"].(string)

	if runID2 == "" {
		t.Error("expected non-empty run_id for second run")
	}
	if runID1 == runID2 {
		t.Errorf("expected different run IDs; both are %q", runID1)
	}

	waitForRunComplete(t, env, "quick-pass", 10*time.Second)
}

// TestDevopsRun_MultiplePipelinesConcurrently verifies that different pipeline
// slugs can run simultaneously without blocking each other.
func TestDevopsRun_MultiplePipelinesConcurrently(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"slow-step.yaml":  pipelineSlowStep,
		"slow-step2.yaml": pipelineSlowStep2,
	})
	env.login("admin@test.local", "admin-pass-123")

	// Start both pipelines concurrently.
	var wg sync.WaitGroup
	errs := make([]string, 2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/slow-step/run", nil)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusAccepted {
			errs[0] = "slow-step: expected 202, got " + http.StatusText(resp.StatusCode)
		}
	}()
	go func() {
		defer wg.Done()
		resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/slow-step2/run", nil)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusAccepted {
			errs[1] = "slow-step2: expected 202, got " + http.StatusText(resp.StatusCode)
		}
	}()
	wg.Wait()

	for _, e := range errs {
		if e != "" {
			t.Error(e)
		}
	}

	// Both should be running simultaneously.
	if !env.proj.DevopsRunner.IsRunning("slow-step") {
		t.Error("slow-step should still be running")
	}
	if !env.proj.DevopsRunner.IsRunning("slow-step2") {
		t.Error("slow-step2 should still be running")
	}

	// Cancel both to clean up.
	c1 := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/slow-step/cancel", nil)
	c1.Body.Close()
	c2 := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/slow-step2/cancel", nil)
	c2.Body.Close()

	waitForRunComplete(t, env, "slow-step", 10*time.Second)
	waitForRunComplete(t, env, "slow-step2", 10*time.Second)
}
