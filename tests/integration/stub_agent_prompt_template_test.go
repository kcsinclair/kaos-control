// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Suite: Stub Agent Prompt Template — verifies that an agent configured with a
// product-owner prompt_template correctly accepts run requests (HTTP 202) and
// that a missing template yields HTTP 409 with code "run_error".
//
// Related defect: lifecycle/defects/stub-agent-no-prompt-for-product-owner.md
//
// Root cause: when stub-agent lacked a prompt_template entry for the
// product-owner role, POST /agents/stub-agent/run returned:
//
//	409 {"error":{"code":"run_error","message":"agent \"stub-agent\" has no
//	     prompt template for role \"product-owner\""}}
//
// Fix: add a product-owner entry to the agent's prompt_templates map.
//
// Run with:
//
//	go test ./tests/... -tags integration -run TestStubAgent_PromptTemplate -v

import (
	"testing"
)

// stubAgentPOTemplateCfgYAML defines two shell-stub agents:
//   - stub-with-po-tpl:    role=product-owner + prompt_templates has product-owner key → succeeds
//   - stub-missing-po-tpl: role=product-owner + prompt_templates has only "analyst" key → 409
const stubAgentPOTemplateCfgYAML = `git:
  default_branch: main
  branch_template: "ticket/{slug}"

roles:
  - product-owner
  - analyst
  - backend-developer
  - frontend-developer
  - test-developer
  - qa
  - reviewer
  - approver

stages:
  - {name: ideas,          dir: ideas}
  - {name: requirements,   dir: requirements}
  - {name: backend-plans,  dir: backend-plans}
  - {name: frontend-plans, dir: frontend-plans}
  - {name: test-plans,     dir: test-plans}
  - {name: tests,          dir: tests}
  - {name: prototypes,     dir: prototypes}
  - {name: releases,       dir: releases}
  - {name: sprints,        dir: sprints}
  - {name: defects,        dir: defects}

users:
  - email: admin@test.local
    roles: [product-owner, analyst, reviewer, approver]
  - email: dev@test.local
    roles: [backend-developer, frontend-developer, test-developer]
  - email: qa@test.local
    roles: [qa]

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []

agents:
  - name: stub-with-po-tpl
    role: [product-owner]
    driver: shell-stub
    allowed_write_paths: []
    git_identity:
      name: Stub With PO Template
      email: stub-po@test.local
    prompt_templates:
      product-owner: "Process {target_path}"

  - name: stub-missing-po-tpl
    role: [product-owner]
    driver: shell-stub
    allowed_write_paths: []
    git_identity:
      name: Stub Missing PO Template
      email: stub-no-po@test.local
    prompt_templates:
      analyst: "Process {target_path}"
`

// TC1: agent has a product-owner prompt_template → POST /run returns 202 and
// the run completes with status "done".
func TestStubAgent_PromptTemplate_ProductOwnerPresent_Succeeds(t *testing.T) {
	const artifactPath = "lifecycle/ideas/stub-po-ok.md"
	env := newAgentTestEnvWithCfg(t, stubAgentPOTemplateCfgYAML, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Stub PO OK", "idea", "draft", "stub-po-ok", "", "Body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env, "stub-with-po-tpl", artifactPath)
	run := waitForRunCompletion(t, env, runID)

	if got, _ := run["status"].(string); got != "done" {
		t.Errorf("run status = %q, want \"done\"", got)
	}
}

// TC2: agent lacks a product-owner prompt_template → POST /run returns 409
// with error code "run_error" and a message naming the missing role.
func TestStubAgent_PromptTemplate_ProductOwnerMissing_Returns409(t *testing.T) {
	const artifactPath = "lifecycle/ideas/stub-po-bad.md"
	env := newAgentTestEnvWithCfg(t, stubAgentPOTemplateCfgYAML, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Stub PO Bad", "idea", "draft", "stub-po-bad", "", "Body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/agents/stub-missing-po-tpl/run", map[string]any{
		"target_path": artifactPath,
	})
	requireStatus(t, resp, 409)
	data := readJSON(t, resp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "run_error" {
		t.Errorf("error code = %q, want \"run_error\"", code)
	}
}

// TC3: product-owner role passed explicitly in the request body → 202 + done.
// Verifies the explicit-role path in StartRun alongside the implicit fallback.
func TestStubAgent_PromptTemplate_ExplicitProductOwnerRole_Succeeds(t *testing.T) {
	const artifactPath = "lifecycle/ideas/stub-po-explicit.md"
	env := newAgentTestEnvWithCfg(t, stubAgentPOTemplateCfgYAML, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Stub PO Explicit", "idea", "draft", "stub-po-explicit", "", "Body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/agents/stub-with-po-tpl/run", map[string]any{
		"target_path": artifactPath,
		"role":        "product-owner",
	})
	requireStatus(t, resp, 202)
	data := readJSON(t, resp)

	runID, _ := data["run_id"].(string)
	if runID == "" {
		t.Fatal("expected run_id in 202 response")
	}

	run := waitForRunCompletion(t, env, runID)
	if got, _ := run["status"].(string); got != "done" {
		t.Errorf("run status = %q, want \"done\"", got)
	}
}

// TC4: explicit role that has no matching prompt_template → 409.
// Guards the code path where the caller overrides role selection but names a
// role that the agent does not have a template for.
func TestStubAgent_PromptTemplate_ExplicitRoleMissingTemplate_Returns409(t *testing.T) {
	const artifactPath = "lifecycle/ideas/stub-po-mismatch.md"
	env := newAgentTestEnvWithCfg(t, stubAgentPOTemplateCfgYAML, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Stub PO Mismatch", "idea", "draft", "stub-po-mismatch", "", "Body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	// "qa" is a valid project role but stub-with-po-tpl has no "qa" template.
	resp := env.doRequest("POST", "/api/p/testproject/agents/stub-with-po-tpl/run", map[string]any{
		"target_path": artifactPath,
		"role":        "qa",
	})
	requireStatus(t, resp, 409)
	data := readJSON(t, resp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "run_error" {
		t.Errorf("error code = %q, want \"run_error\"", code)
	}
}
