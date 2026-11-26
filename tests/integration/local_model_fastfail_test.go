// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestone 2 — Manager Fast-Fail & Lineage Lock Release Integration Tests
// (local-model-operability: lifecycle/test-plans/local-model-operability-5-test.md)
//
// FR-2, NFR-1, NFR-3: when the preflight model-availability check fails
// (model not registered on the provider, or the provider endpoint is
// unreachable), Manager.StartRun must abort before acquiring any lineage
// lock, must not mutate the target artifact or produce a git commit, and
// must return within 3 seconds.
//
// internal/agent/openai_preflight_availability_test.go (TestStartRun_Preflight)
// already exercises the same scenarios directly against Manager. These tests
// drive them through the full HTTP API + git-backed project harness used by
// TestOpenAIAgentRun_* in openai_agent_run_test.go, adding the request-timing
// (NFR-1) and zero-git-mutation (NFR-3) assertions the plan calls out that
// aren't observable from inside the agent package alone.

import (
	"net/http"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/kaos-control/kaos-control/tests/integration/testutil"
)

// assertFastFail drives a run against env's mock provider and asserts the
// common local-model-operability fast-fail contract: sub-3s response, 409
// with the given failure reason in the error message, no lineage lock left
// behind, no new git commits, unchanged artifact status, and a classified
// failed run record in the index.
func assertFastFail(t *testing.T, env *openAIAgentTestEnv, wantFailureReason string) {
	t.Helper()

	const targetPath = "lifecycle/ideas/test-idea.md"

	repo, err := gogit.PlainOpen(env.projectRoot)
	if err != nil {
		t.Fatal(err)
	}
	headBefore, err := repo.Head()
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	resp := env.doRequest(http.MethodPost, "/api/p/testproject/agents/openai-analyst/run", map[string]any{
		"target_path": targetPath,
	})
	elapsed := time.Since(start)

	if elapsed > 3*time.Second {
		t.Errorf("fast-fail took %v, want < 3s (NFR-1)", elapsed)
	}
	requireStatus(t, resp, http.StatusConflict)
	body := readJSON(t, resp)
	errObj, _ := body["error"].(map[string]any)
	if msg, _ := errObj["message"].(string); !strings.Contains(msg, wantFailureReason) {
		t.Fatalf("expected error message to mention %q, got: %v", wantFailureReason, errObj)
	}

	// Lineage lock must be free (no lock ever acquired).
	lockRow, err := env.proj.Locks.Get(targetPath)
	if err != nil {
		t.Fatalf("checking lock: %v", err)
	}
	if lockRow != nil {
		t.Fatalf("expected no lineage lock after fast-fail, got %+v", lockRow)
	}

	// Zero artifact corruption or uncommitted git changes (NFR-3).
	headAfter, err := repo.Head()
	if err != nil {
		t.Fatal(err)
	}
	if headBefore.Hash() != headAfter.Hash() {
		t.Errorf("expected no new git commits, HEAD moved from %s to %s", headBefore.Hash(), headAfter.Hash())
	}

	row, err := env.proj.Idx.Get(targetPath)
	if err != nil {
		t.Fatalf("fetching artifact from index: %v", err)
	}
	if row == nil || row.Status != "draft" {
		t.Fatalf("expected target artifact status to remain 'draft', got %+v", row)
	}

	// The database records the failure with the classified reason.
	runs, err := env.proj.Idx.ListAgentRuns("failed", 0)
	if err != nil {
		t.Fatalf("ListAgentRuns: %v", err)
	}
	found := false
	for _, r := range runs {
		if r.TargetPath == targetPath && r.FailureReason != nil && *r.FailureReason == wantFailureReason {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected a failed run record with failure_reason=%s, got %+v", wantFailureReason, runs)
	}
}

// TestOpenAIAgentRun_ModelNotFound_FastFail verifies that requesting a model
// absent from the provider's /v1/models listing aborts the run fast, with no
// lock held and no artifact/git mutation.
func TestOpenAIAgentRun_ModelNotFound_FastFail(t *testing.T) {
	env := newOpenAIAgentTestEnv(t, nil, 4)
	env.mock.Models = []testutil.OpenAIModel{{ID: "some-other-model"}}
	env.login("admin@test.local", "admin-pass-123")

	assertFastFail(t, env, "model_not_found")
}

// TestOpenAIAgentRun_EndpointUnreachable_FastFail verifies that a provider
// endpoint refusing connections aborts the run fast, with no lock held and no
// artifact/git mutation.
func TestOpenAIAgentRun_EndpointUnreachable_FastFail(t *testing.T) {
	env := newOpenAIAgentTestEnv(t, nil, 4)
	env.login("admin@test.local", "admin-pass-123")
	env.mock.Close() // provider is now unreachable (connection refused)

	assertFastFail(t, env, "endpoint_unreachable")
}
