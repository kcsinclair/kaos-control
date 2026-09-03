// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Suite 2.2 — Provider switching REST API integration tests (SA1-SA6), plus
// Milestone 4 (policy inspection) and Milestone 8 (in-flight-run guard)
// coverage.
//
// providerSwitchAPICfgYAML wires four agents against three app-level
// providers (anthropic-cloud, gemini-cloud, local-ollama):
//   - agent-a: on its normal (primary) provider, not in failover.
//   - agent-b, agent-c, agent-d: declared on anthropic-cloud too; tests that
//     need them already in a failover state seed that directly into
//     operations.yaml via newProviderSwitchAPIEnv, so restore-all has three
//     agents to act on atomically (SA4).
//
// A "local-ai" provider_templates entry maps all four agents to
// local-ollama/llama3 for the template-apply test (SA5).

import (
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
)

const providerSwitchAPICfgYAML = `git:
  default_branch: main
  branch_template: "ticket/{slug}"

roles:
  - product-owner
  - analyst
  - devops

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
    roles: [product-owner]
  - email: dev@test.local
    roles: [devops]
  - email: qa@test.local
    roles: [analyst]

required_plans:
  ticket: []
  epic: []

provider_templates:
  - name: local-ai
    description: "Switch every agent to local Ollama targets"
    agents:
      agent-a: {provider: local-ollama, model: llama3}
      agent-b: {provider: local-ollama, model: llama3}
      agent-c: {provider: local-ollama, model: llama3}
      agent-d: {provider: local-ollama, model: llama3}

agents:
  - name: agent-a
    role: [analyst]
    driver: claude-code-cli
    provider: anthropic-cloud
    model: claude-3-7-sonnet
    fallback_provider: gemini-cloud
    fallback_model: gemini-2.5-flash
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: Agent A
      email: agent-a@test.local
    prompt_templates:
      analyst: "x"
  - name: agent-b
    role: [analyst]
    driver: claude-code-cli
    provider: anthropic-cloud
    model: claude-3-7-sonnet
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: Agent B
      email: agent-b@test.local
    prompt_templates:
      analyst: "x"
  - name: agent-c
    role: [analyst]
    driver: claude-code-cli
    provider: anthropic-cloud
    model: claude-3-7-sonnet
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: Agent C
      email: agent-c@test.local
    prompt_templates:
      analyst: "x"
  - name: agent-d
    role: [analyst]
    driver: claude-code-cli
    provider: anthropic-cloud
    model: claude-3-7-sonnet
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: Agent D
      email: agent-d@test.local
    prompt_templates:
      analyst: "x"
`

// providerSwitchAPIProviders builds the app-level provider catalog for the
// SA1-SA6 tests: three healthy mock upstreams named to match
// providerSwitchAPICfgYAML.
func providerSwitchAPIProviders(t *testing.T) []config.Provider {
	t.Helper()
	anthropic := newMockProvider(t, true)
	gemini := newMockProvider(t, true)
	ollama := newMockProvider(t, true)
	return []config.Provider{
		{Name: "anthropic-cloud", BaseURL: anthropic.URL, Driver: "openai-compatible"},
		{Name: "gemini-cloud", BaseURL: gemini.URL, Driver: "openai-compatible"},
		{Name: "local-ollama", BaseURL: ollama.URL, Driver: "openai-compatible"},
	}
}

// newProviderSwitchAPIEnv builds the SA1-SA6 test environment and, unless
// noSeed is set, seeds agent-b/c/d as already failed over to gemini-cloud
// (operations.yaml override, Milestone 2/8) so restore-all has three agents
// to act on atomically (SA4) and GetStatus has failover state to report
// (SA1).
func newProviderSwitchAPIEnv(t *testing.T, seeds ...seedArtifact) *failoverTestEnv {
	t.Helper()
	env := newFailoverTestEnv(t, providerSwitchAPICfgYAML, providerSwitchAPIProviders(t), seeds)
	for _, name := range []string{"agent-b", "agent-c", "agent-d"} {
		env.seedFailoverState(name, "gemini-cloud", "gemini-2.5-flash", "seed")
	}
	return env
}

// TestProviderSwitchAPI_GetStatus (SA1): GET .../provider-switch/status
// reports failover_active, per-agent is_failover flags, and active vs
// primary targets for an agent currently in failover.
func TestProviderSwitchAPI_GetStatus(t *testing.T) {
	env := newProviderSwitchAPIEnv(t)

	resp := env.doRequest("GET", "/api/p/testproject/provider-switch/status", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	if active, _ := data["failover_active"].(bool); !active {
		t.Error("expected failover_active=true (agent-b/c/d are in failover)")
	}

	agents, _ := data["agents"].([]any)
	byName := map[string]map[string]any{}
	for _, raw := range agents {
		a, _ := raw.(map[string]any)
		byName[a["agent"].(string)] = a
	}

	a, ok := byName["agent-a"]
	if !ok {
		t.Fatal("expected agent-a in status response")
	}
	if isFailover, _ := a["is_failover"].(bool); isFailover {
		t.Error("agent-a should not be in failover")
	}
	if a["active_provider"] != "anthropic-cloud" {
		t.Errorf("agent-a active_provider: got %v, want anthropic-cloud", a["active_provider"])
	}

	b, ok := byName["agent-b"]
	if !ok {
		t.Fatal("expected agent-b in status response")
	}
	if isFailover, _ := b["is_failover"].(bool); !isFailover {
		t.Error("agent-b should be in failover")
	}
	if b["active_provider"] != "gemini-cloud" {
		t.Errorf("agent-b active_provider: got %v, want gemini-cloud", b["active_provider"])
	}
	if b["primary_provider"] != "anthropic-cloud" {
		t.Errorf("agent-b primary_provider: got %v, want anthropic-cloud", b["primary_provider"])
	}
}

// TestProviderSwitchAPI_ManualSwitch (SA2): POST
// .../agents/{name}/switch-provider manually switches agent-a (not
// currently in failover). Asserts the switch lands in operations.yaml only
// — lifecycle/config.yaml and git are untouched — and the provider.switched
// WS event is broadcast with is_failover=false.
func TestProviderSwitchAPI_ManualSwitch(t *testing.T) {
	env := newProviderSwitchAPIEnv(t)
	beforeSHA := env.headSHA()
	beforeCfg := env.readConfigYAML()
	ws := env.connectProjectWS()

	// Switch to local-ollama (not agent-a's own fallback_provider, which
	// must remain distinct from the active provider per config validation).
	resp := env.doRequest("POST", "/api/p/testproject/agents/agent-a/switch-provider", map[string]any{
		"provider": "local-ollama",
		"model":    "llama3",
		"reason":   "manual test switch",
	})
	requireStatus(t, resp, 200)
	body := readJSON(t, resp)
	if body["provider"] != "local-ollama" {
		t.Errorf("response provider: got %v, want local-ollama", body["provider"])
	}

	assertAgentFailedOverTo(t, env, "agent-a", "anthropic-cloud", "local-ollama")

	if got := env.readConfigYAML(); got != beforeCfg {
		t.Error("expected lifecycle/config.yaml to be byte-for-byte unchanged after a manual switch")
	}
	if got := env.headSHA(); got != beforeSHA {
		t.Errorf("expected no new git commit after a manual switch, HEAD moved from %s to %s", beforeSHA, got)
	}

	payload := ws.waitForEventType(t, 5*time.Second, "provider.switched")
	if payload["agent"] != "agent-a" {
		t.Errorf("provider.switched agent: got %v, want agent-a", payload["agent"])
	}
	if isFailover, _ := payload["is_failover"].(bool); isFailover {
		t.Error("manual switch should broadcast is_failover=false")
	}
}

// TestProviderSwitchAPI_RestoreAgent (SA3): POST
// .../agents/{name}/restore-provider restores agent-b (already in
// failover) to its stashed primary, clearing its operations.yaml override.
// lifecycle/config.yaml and git are untouched throughout.
func TestProviderSwitchAPI_RestoreAgent(t *testing.T) {
	env := newProviderSwitchAPIEnv(t)
	beforeSHA := env.headSHA()

	resp := env.doRequest("POST", "/api/p/testproject/agents/agent-b/restore-provider", nil)
	requireStatus(t, resp, 200)
	body := readJSON(t, resp)
	if body["provider"] != "anthropic-cloud" {
		t.Errorf("response provider: got %v, want anthropic-cloud", body["provider"])
	}

	if _, hasState := env.proj.Operations().AgentState("agent-b"); hasState {
		t.Error("expected agent-b's operations.yaml override cleared after restore")
	}

	ag, ok := findFailoverAgentConfig(env.loadConfig(), "agent-b")
	if !ok {
		t.Fatal("agent-b missing from disk config")
	}
	if ag.Provider != "anthropic-cloud" || ag.Model != "claude-3-7-sonnet" {
		t.Errorf("expected agent-b's declared config to remain anthropic-cloud/claude-3-7-sonnet, got %s/%s", ag.Provider, ag.Model)
	}
	if got := env.headSHA(); got != beforeSHA {
		t.Errorf("expected no new git commit after restore, HEAD moved from %s to %s", beforeSHA, got)
	}
}

// TestProviderSwitchAPI_RestoreAll (SA4): POST
// .../provider-switch/restore-all restores every agent currently in
// failover (agent-b, agent-c, agent-d) atomically in one call, leaving
// agent-a untouched.
func TestProviderSwitchAPI_RestoreAll(t *testing.T) {
	env := newProviderSwitchAPIEnv(t)

	resp := env.doRequest("POST", "/api/p/testproject/provider-switch/restore-all", nil)
	requireStatus(t, resp, 200)
	body := readJSON(t, resp)
	restored, _ := body["restored_agents"].(float64)
	if restored != 3 {
		t.Errorf("restored_agents: got %v, want 3", restored)
	}

	statusResp := env.doRequest("GET", "/api/p/testproject/provider-switch/status", nil)
	requireStatus(t, statusResp, 200)
	statusData := readJSON(t, statusResp)
	if active, _ := statusData["failover_active"].(bool); active {
		t.Error("expected failover_active=false after restore-all")
	}
}

// TestProviderSwitchAPI_ApplyTemplate (SA5): POST
// .../provider-templates/apply switches every mapped agent to the named
// template's provider/model in one atomic write.
func TestProviderSwitchAPI_ApplyTemplate(t *testing.T) {
	env := newProviderSwitchAPIEnv(t)
	beforeCfg := env.readConfigYAML()

	resp := env.doRequest("POST", "/api/p/testproject/provider-templates/apply", map[string]any{
		"template": "local-ai",
	})
	requireStatus(t, resp, 200)
	body := readJSON(t, resp)
	updated, _ := body["updated_agents"].(float64)
	if updated != 4 {
		t.Errorf("updated_agents: got %v, want 4", updated)
	}

	for _, name := range []string{"agent-a", "agent-b", "agent-c", "agent-d"} {
		assertAgentFailedOverTo(t, env, name, "anthropic-cloud", "local-ollama")
	}
	if got := env.readConfigYAML(); got != beforeCfg {
		t.Error("expected lifecycle/config.yaml to be byte-for-byte unchanged after a template apply")
	}
}

// TestProviderSwitchAPI_RoleAuth (SA6): a user without product-owner/devops
// roles (qa@test.local, role=analyst only) receives 403 Forbidden on
// provider-switch mutation endpoints.
func TestProviderSwitchAPI_RoleAuth(t *testing.T) {
	env := newProviderSwitchAPIEnv(t)
	env.login("qa@test.local", "qa-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/agents/agent-a/switch-provider", map[string]any{
		"provider": "gemini-cloud",
		"model":    "gemini-2.5-flash",
	})
	requireStatus(t, resp, 403)
	resp.Body.Close()

	resp2 := env.doRequest("POST", "/api/p/testproject/provider-switch/restore-all", nil)
	requireStatus(t, resp2, 403)
	resp2.Body.Close()
}

// TestProviderSwitchAPI_GetPolicy (Milestone 4/FR-2.4): GET
// .../provider-switch/policy returns automated_switchover plus an explicit
// action for every classified reason the system knows about — no reason is
// allowed to fall through to an implicit default.
func TestProviderSwitchAPI_GetPolicy(t *testing.T) {
	env := newProviderSwitchAPIEnv(t)

	resp := env.doRequest("GET", "/api/p/testproject/provider-switch/policy", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	if _, ok := data["automated_switchover"].(bool); !ok {
		t.Fatal("expected automated_switchover bool in policy response")
	}
	actions, ok := data["actions"].(map[string]any)
	if !ok {
		t.Fatal("expected actions map in policy response")
	}
	if len(actions) != len(config.SwitchoverReasons) {
		t.Errorf("actions map has %d entries, want %d (one per SwitchoverReasons entry)", len(actions), len(config.SwitchoverReasons))
	}
	for _, reason := range config.SwitchoverReasons {
		action, ok := actions[reason].(string)
		if !ok || action == "" {
			t.Errorf("expected a non-empty action for reason %q, got %v", reason, actions[reason])
		}
	}
}

// TestProviderSwitchAPI_ManualSwitchRejectedWhileRunning (Milestone 8/FR-8.2):
// a manual switch attempted while a run is in flight for the project is
// rejected with 409 and the running jobs named, rather than racing the
// live run's own provider.
func TestProviderSwitchAPI_ManualSwitchRejectedWhileRunning(t *testing.T) {
	setupFakeClaudeWithScript(t, "sleep 5\n"+fakeClaudeSuccessEvents+"exit 0\n")

	env := newProviderSwitchAPIEnv(t, seedArtifact{
		relPath: "lifecycle/ideas/running-idea.md",
		content: makeApprovedArtifact("Running Idea", "idea", "running-idea"),
	})

	env.enqueue("lifecycle/ideas/running-idea.md", "agent-a")
	env.waitFor(5*time.Second, "job to start running", func() bool {
		snap := env.queueSnapshot()
		running, _ := snap["running"].(map[string]any)
		return running != nil && running["artifact_path"] == "lifecycle/ideas/running-idea.md"
	})

	resp := env.doRequest("POST", "/api/p/testproject/agents/agent-a/switch-provider", map[string]any{
		"provider": "gemini-cloud",
		"model":    "gemini-2.5-flash",
	})
	requireStatus(t, resp, 409)
	body := readJSON(t, resp)
	apiErr, _ := body["error"].(map[string]any)
	if apiErr["code"] != "runs_in_progress" {
		t.Errorf("error code: got %v, want runs_in_progress", apiErr["code"])
	}
	runningJobs, _ := body["running_jobs"].([]any)
	if len(runningJobs) != 1 {
		t.Fatalf("running_jobs: got %d entries, want 1", len(runningJobs))
	}
	job, _ := runningJobs[0].(map[string]any)
	if job["agent"] != "agent-a" || job["artifact_path"] != "lifecycle/ideas/running-idea.md" {
		t.Errorf("running_jobs[0]: got %v", job)
	}

	// The rejected switch must not have engaged.
	if _, hasState := env.proj.Operations().AgentState("agent-a"); hasState {
		t.Error("expected no operations.yaml override for agent-a after a rejected switch")
	}
}
