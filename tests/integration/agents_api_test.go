//go:build integration

package integration

import (
	"bytes"
	"net/http"
	"testing"
)

// agentPanelCfgYAML is the lifecycle/config.yaml for agent-launcher-panels tests.
// It includes:
//   - agent-with-model:       model + active_status set (claude-code-cli)
//   - agent-no-model:         active_status set, no model (claude-code-cli)
//   - agent-no-active-status: model set, no active_status (claude-code-cli)
//   - idea-capture:           driver=inline, no model, no active_status
const agentPanelCfgYAML = `git:
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
  - name: agent-with-model
    role: [analyst]
    driver: claude-code-cli
    model: claude-opus-4-6
    active_status: clarifying
    allowed_write_paths:
      - lifecycle/requirements
    git_identity:
      name: Agent With Model
      email: agent-with-model@test.local
    prompt_templates:
      analyst: "Test prompt for {target_path}"

  - name: agent-no-model
    role: [analyst]
    driver: claude-code-cli
    active_status: planning
    allowed_write_paths:
      - lifecycle/requirements
    git_identity:
      name: Agent No Model
      email: agent-no-model@test.local
    prompt_templates:
      analyst: "Test prompt for {target_path}"

  - name: agent-no-active-status
    role: [backend-developer]
    driver: claude-code-cli
    model: claude-sonnet-4-6
    allowed_write_paths:
      - lifecycle/requirements
    git_identity:
      name: Agent No Active Status
      email: agent-no-active-status@test.local
    prompt_templates:
      backend-developer: "Test prompt for {target_path}"

  - name: idea-capture
    role: [product-owner]
    driver: inline
    allowed_write_paths:
      - lifecycle/ideas
    git_identity:
      name: Idea Capture
      email: idea-capture@test.local
    prompt_templates:
      idea-capture: "Test inline prompt"
`

// ── Milestone 1 ───────────────────────────────────────────────────────────

// TestListAgents_ModelAndActiveStatus verifies that GET /agents exposes the
// model and active_status fields for agents that define them, and omits those
// fields (or returns empty string) for agents that do not.
// Covers test plan Milestone 1, scenario 1.
func TestListAgents_ModelAndActiveStatus(t *testing.T) {
	env := newAgentTestEnvWithCfg(t, agentPanelCfgYAML, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/agents", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	agentsRaw, _ := data["agents"].([]any)
	if len(agentsRaw) == 0 {
		t.Fatal("expected non-empty agents list")
	}

	// Build lookup map by name.
	byName := make(map[string]map[string]any, len(agentsRaw))
	for _, raw := range agentsRaw {
		ag, _ := raw.(map[string]any)
		name, _ := ag["name"].(string)
		byName[name] = ag
	}

	// Every agent must expose name, roles, driver.
	for _, raw := range agentsRaw {
		ag, _ := raw.(map[string]any)
		if _, ok := ag["name"]; !ok {
			t.Errorf("agent missing 'name' field: %v", ag)
		}
		if _, ok := ag["roles"]; !ok {
			t.Errorf("agent %v missing 'roles' field", ag["name"])
		}
		if _, ok := ag["driver"]; !ok {
			t.Errorf("agent %v missing 'driver' field", ag["name"])
		}
	}

	// agent-with-model: model and active_status must be present and correct.
	if ag, ok := byName["agent-with-model"]; !ok {
		t.Error("agent-with-model not found in response")
	} else {
		if model, _ := ag["model"].(string); model != "claude-opus-4-6" {
			t.Errorf("agent-with-model: want model %q, got %q", "claude-opus-4-6", model)
		}
		if as, _ := ag["active_status"].(string); as != "clarifying" {
			t.Errorf("agent-with-model: want active_status %q, got %q", "clarifying", as)
		}
	}

	// agent-no-model: active_status present, model must be absent/empty (omitempty).
	if ag, ok := byName["agent-no-model"]; !ok {
		t.Error("agent-no-model not found in response")
	} else {
		if model, _ := ag["model"].(string); model != "" {
			t.Errorf("agent-no-model: expected model absent/empty (omitempty), got %q", model)
		}
		if as, _ := ag["active_status"].(string); as != "planning" {
			t.Errorf("agent-no-model: want active_status %q, got %q", "planning", as)
		}
	}

	// agent-no-active-status: model present, active_status must be absent/empty.
	if ag, ok := byName["agent-no-active-status"]; !ok {
		t.Error("agent-no-active-status not found in response")
	} else {
		if model, _ := ag["model"].(string); model != "claude-sonnet-4-6" {
			t.Errorf("agent-no-active-status: want model %q, got %q", "claude-sonnet-4-6", model)
		}
		if as, _ := ag["active_status"].(string); as != "" {
			t.Errorf("agent-no-active-status: expected active_status absent/empty (omitempty), got %q", as)
		}
	}
}

// TestListAgents_InlineDriver verifies that an inline-driver agent appears in
// the agent list with driver="inline", allowing the frontend to identify
// non-launchable agents.
// Covers test plan Milestone 1, scenario 2.
func TestListAgents_InlineDriver(t *testing.T) {
	env := newAgentTestEnvWithCfg(t, agentPanelCfgYAML, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/agents", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	agentsRaw, _ := data["agents"].([]any)
	for _, raw := range agentsRaw {
		ag, _ := raw.(map[string]any)
		if ag["name"] == "idea-capture" {
			if driver, _ := ag["driver"].(string); driver != "inline" {
				t.Errorf("idea-capture: want driver %q, got %q", "inline", driver)
			}
			return // found and verified
		}
	}
	t.Error("idea-capture agent with driver=inline not found in GET /agents response")
}

// ── Milestone 3 ───────────────────────────────────────────────────────────

// TestStartAgentRun_Success verifies that POSTing a valid target artifact path
// to an existing agent returns HTTP 202 with a non-empty run_id.
// Covers test plan Milestone 3, scenario 1.
func TestStartAgentRun_Success(t *testing.T) {
	setupFakeClaude(t, 0)

	const artifactPath = "lifecycle/ideas/launch-target.md"
	env := newAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Launch Target", "idea", "draft", "launch-target", "", "Idea body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/agents/analyst-requirements/run", map[string]any{
		"target_path": artifactPath,
	})
	requireStatus(t, resp, 202)
	data := readJSON(t, resp)

	runID, _ := data["run_id"].(string)
	if runID == "" {
		t.Error("expected non-empty run_id in 202 response")
	}
}

// TestStartAgentRun_AgentNotFound verifies that POSTing to a non-existent agent
// name returns HTTP 404 with error code "not_found".
// Covers test plan Milestone 3, scenario 2.
func TestStartAgentRun_AgentNotFound(t *testing.T) {
	const artifactPath = "lifecycle/ideas/notfound-target.md"
	env := newAgentTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Not Found Target", "idea", "draft", "notfound-target", "", "Idea body."),
	}})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/agents/does-not-exist/run", map[string]any{
		"target_path": artifactPath,
	})
	requireStatus(t, resp, 404)
	data := readJSON(t, resp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "not_found" {
		t.Errorf("expected error code %q, got %q", "not_found", code)
	}
}

// TestStartAgentRun_BadRequest verifies that POSTing malformed JSON returns
// HTTP 400 with error code "bad_request".
// Covers test plan Milestone 3, scenario 3.
func TestStartAgentRun_BadRequest(t *testing.T) {
	env := newAgentTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Send raw invalid JSON directly, bypassing doRequest which always marshals valid JSON.
	req, err := http.NewRequest("POST",
		env.baseURL+"/api/p/testproject/agents/analyst-requirements/run",
		bytes.NewReader([]byte(`{not valid json`)),
	)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", env.csrfToken)
	for _, c := range env.cookies {
		req.AddCookie(c)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, resp, 400)
	data := readJSON(t, resp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "bad_request" {
		t.Errorf("expected error code %q, got %q", "bad_request", code)
	}
}
