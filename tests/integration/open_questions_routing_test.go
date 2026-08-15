// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"strings"
	"testing"
)

// routingAgentsCfgYAML extends defaultCfgYAML with a single backend-developer
// agent so TestOpenQuestionsRouting_DeveloperArtifactNoAutoRequeue can drive
// an explicit requeue call (POST .../agents/backend-developer/run) against a
// resolved plan-backend artifact.
const routingAgentsCfgYAML = defaultCfgYAML + `
agents:
  - name: backend-developer
    role: [backend-developer]
    driver: claude-code-cli
    source_types: [plan-backend]
    allowed_write_paths:
      - lifecycle/backend-plans
    git_identity:
      name: Backend Developer Agent
      email: backend-developer@test.local
    prompt_templates:
      backend-developer: "Test backend developer prompt for {target_path}"
`

// resolveOpenQuestions drives the standard preview(complete=true) + PUT flow
// used across the open-questions test suite (see the CompleteResolve step in
// open_questions_resolve_e2e_test.go): it previews a complete resolution of
// the single question written by makeBlockedArtifact, persists it via PUT,
// and returns the resulting status. fm supplies the frontmatter fields other
// than status (title, type, lineage, ...) for the PUT payload.
func resolveOpenQuestions(t *testing.T, env *testEnv, relPath string, fm map[string]any) string {
	t.Helper()

	previewResp := env.doRequest("POST", "/api/p/testproject/artifacts/"+relPath+"/open-questions/preview", map[string]any{
		"answers":  map[string]string{"0": "Rayleigh scattering."},
		"complete": true,
	})
	requireStatus(t, previewResp, 200)
	previewData := readJSON(t, previewResp)
	newBody, _ := previewData["body"].(string)
	if !strings.Contains(newBody, "## Resolved Questions") {
		t.Fatalf("complete preview did not rename the heading, got: %q", newBody)
	}

	putFM := map[string]any{"status": "blocked"}
	for k, v := range fm {
		putFM[k] = v
	}
	putResp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPath, map[string]any{
		"frontmatter": putFM,
		"body":        newBody,
	})
	requireStatus(t, putResp, 200)
	putData := readJSON(t, putResp)
	art, _ := putData["artifact"].(map[string]any)
	gotFM, _ := art["frontmatter"].(map[string]any)
	return statusOf(gotFM)
}

// TestOpenQuestionsRouting_RequirementNoAutoApprove verifies that resolving
// the open questions on a "requirement" artifact routes it to "draft"
// (awaiting approval) — never straight to "approved" — and that "approved"
// is only reached via a separate, deliberate transition call (FR/NFR from
// lifecycle/tests/open-questions-gui-5-test.md Milestone 4).
func TestOpenQuestionsRouting_RequirementNoAutoApprove(t *testing.T) {
	const relPath = "lifecycle/requirements/routing-requirement.md"
	const lineage = "routing-requirement"
	seeds := []seedArtifact{
		{relPath: relPath, content: makeBlockedArtifact("Routing Requirement", "requirement", lineage, "")},
	}
	env := newTestEnv(t, seeds)
	fm := map[string]any{"title": "Routing Requirement", "type": "requirement", "lineage": lineage}

	t.Run("ResolveRoutesToDraftNotApproved", func(t *testing.T) {
		status := resolveOpenQuestions(t, env, relPath, fm)
		if status != "draft" {
			t.Fatalf("expected resolve to route requirement artifact to %q, got %q", "draft", status)
		}
	})

	t.Run("ApprovalOnlyHappensViaExplicitTransition", func(t *testing.T) {
		// Precondition: the resolve above left the artifact at "draft"; no
		// approval has occurred yet.
		getResp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
		requireStatus(t, getResp, 200)
		getData := readJSON(t, getResp)
		art, _ := getData["artifact"].(map[string]any)
		preFM, _ := art["frontmatter"].(map[string]any)
		if status := statusOf(preFM); status != "draft" {
			t.Fatalf("precondition failed: expected status %q before explicit approve, got %q", "draft", status)
		}

		// The explicit, deliberate approve action: a workflow transition call.
		transResp := env.doRequest("POST", "/api/p/testproject/artifacts/"+relPath+"/transition", map[string]any{
			"to": "approved",
		})
		requireStatus(t, transResp, 200)
		transData := readJSON(t, transResp)
		postArt, _ := transData["artifact"].(map[string]any)
		if status, _ := postArt["status"].(string); status != "approved" {
			t.Fatalf("expected explicit transition to set status %q, got %q", "approved", status)
		}
	})
}

// TestOpenQuestionsRouting_DeveloperArtifactNoAutoRequeue verifies that
// resolving the open questions on a developer-raised artifact (a plan-*
// artifact carrying an "## Open Questions" section) routes it to "draft" and
// starts no agent run automatically. An explicit requeue call — POST
// .../agents/{name}/run — targets the originating developer role
// (backend-developer for a plan-backend artifact) and is the only thing that
// starts a run (FR/NFR from lifecycle/tests/open-questions-gui-5-test.md
// Milestone 4).
func TestOpenQuestionsRouting_DeveloperArtifactNoAutoRequeue(t *testing.T) {
	setupFakeClaude(t, 0)

	const relPath = "lifecycle/backend-plans/routing-plan-2-be.md"
	const lineage = "routing-plan"
	seeds := []seedArtifact{
		{relPath: relPath, content: makeBlockedArtifact("Routing Plan", "plan-backend", lineage, "")},
	}
	env := newTestEnvWithCfgYAML(t, seeds, routingAgentsCfgYAML)
	fm := map[string]any{"title": "Routing Plan", "type": "plan-backend", "lineage": lineage}

	t.Run("ResolveRoutesToDraftWithNoRunStarted", func(t *testing.T) {
		status := resolveOpenQuestions(t, env, relPath, fm)
		if status != "draft" {
			t.Fatalf("expected resolve to route plan-backend artifact to %q, got %q", "draft", status)
		}

		runsResp := env.doRequest("GET", "/api/p/testproject/agents/runs?target_path="+relPath, nil)
		requireStatus(t, runsResp, 200)
		runsData := readJSON(t, runsResp)
		runs, _ := runsData["runs"].([]any)
		if len(runs) != 0 {
			t.Fatalf("expected no agent run started automatically after resolve, got %d: %v", len(runs), runs)
		}
	})

	t.Run("ExplicitRequeueTargetsOriginatingRoleAndStartsRun", func(t *testing.T) {
		runID := startAgentRun(t, env, "backend-developer", relPath)
		run := waitForRunCompletion(t, env, runID)

		if role, _ := run["role"].(string); role != "backend-developer" {
			t.Errorf("expected requeued run role %q (originating developer role for plan-backend), got %q", "backend-developer", role)
		}
		if agentName, _ := run["agent_name"].(string); agentName != "backend-developer" {
			t.Errorf("expected requeued run agent_name %q, got %q", "backend-developer", agentName)
		}

		runsResp := env.doRequest("GET", "/api/p/testproject/agents/runs?target_path="+relPath, nil)
		requireStatus(t, runsResp, 200)
		runsData := readJSON(t, runsResp)
		runs, _ := runsData["runs"].([]any)
		if len(runs) != 1 {
			t.Fatalf("expected exactly 1 run after the explicit requeue call, got %d", len(runs))
		}
	})
}
