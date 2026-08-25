// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Suite 2.2 — Provider switching REST API integration tests (SA1-SA6).
//
// providerSwitchAPICfgYAML wires four agents against three app-level
// providers (anthropic-cloud, gemini-cloud, local-ollama):
//   - agent-a: on its normal (primary) provider, not in failover.
//   - agent-b, agent-c, agent-d: already in a failover state (provider =
//     gemini-cloud, primary_provider = anthropic-cloud stashed), so
//     restore-all has three agents to act on atomically (SA4).
//
// A "local-ai" provider_templates entry maps all four agents to
// local-ollama/llama3 for the template-apply test (SA5).

import (
	"strings"
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
    provider: gemini-cloud
    model: gemini-2.5-flash
    primary_provider: anthropic-cloud
    primary_model: claude-3-7-sonnet
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: Agent B
      email: agent-b@test.local
    prompt_templates:
      analyst: "x"
  - name: agent-c
    role: [analyst]
    driver: claude-code-cli
    provider: gemini-cloud
    model: gemini-2.5-flash
    primary_provider: anthropic-cloud
    primary_model: claude-3-7-sonnet
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: Agent C
      email: agent-c@test.local
    prompt_templates:
      analyst: "x"
  - name: agent-d
    role: [analyst]
    driver: claude-code-cli
    provider: gemini-cloud
    model: gemini-2.5-flash
    primary_provider: anthropic-cloud
    primary_model: claude-3-7-sonnet
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

// TestProviderSwitchAPI_GetStatus (SA1): GET .../provider-switch/status
// reports failover_active, per-agent is_failover flags, and active vs
// primary targets for an agent currently in failover.
func TestProviderSwitchAPI_GetStatus(t *testing.T) {
	env := newFailoverTestEnv(t, providerSwitchAPICfgYAML, providerSwitchAPIProviders(t), nil)

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
// currently in failover). Asserts disk updated, git commit created, and the
// provider.switched WS event broadcast.
func TestProviderSwitchAPI_ManualSwitch(t *testing.T) {
	env := newFailoverTestEnv(t, providerSwitchAPICfgYAML, providerSwitchAPIProviders(t), nil)
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

	ag, ok := findFailoverAgentConfig(env.loadConfig(), "agent-a")
	if !ok {
		t.Fatal("agent-a missing from disk config after switch")
	}
	if ag.Provider != "local-ollama" || ag.Model != "llama3" {
		t.Errorf("agent-a provider/model: got %s/%s, want local-ollama/llama3", ag.Provider, ag.Model)
	}

	found := false
	for _, msg := range env.gitLogMessages(10) {
		if strings.Contains(msg, "switch(agent):") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected a switch(agent): commit in lifecycle/config.yaml history")
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
// failover) to its stashed primary, clearing primary_provider on disk and
// committing the change.
func TestProviderSwitchAPI_RestoreAgent(t *testing.T) {
	env := newFailoverTestEnv(t, providerSwitchAPICfgYAML, providerSwitchAPIProviders(t), nil)

	resp := env.doRequest("POST", "/api/p/testproject/agents/agent-b/restore-provider", nil)
	requireStatus(t, resp, 200)
	body := readJSON(t, resp)
	if body["provider"] != "anthropic-cloud" {
		t.Errorf("response provider: got %v, want anthropic-cloud", body["provider"])
	}

	ag, ok := findFailoverAgentConfig(env.loadConfig(), "agent-b")
	if !ok {
		t.Fatal("agent-b missing from disk config after restore")
	}
	if ag.PrimaryProvider != "" || ag.PrimaryModel != "" {
		t.Errorf("expected primary_provider/model cleared for agent-b after restore, got %q/%q", ag.PrimaryProvider, ag.PrimaryModel)
	}
	if ag.Provider != "anthropic-cloud" || ag.Model != "claude-3-7-sonnet" {
		t.Errorf("expected agent-b restored to anthropic-cloud/claude-3-7-sonnet, got %s/%s", ag.Provider, ag.Model)
	}

	found := false
	for _, msg := range env.gitLogMessages(10) {
		if strings.Contains(msg, "restore(agent):") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected a restore(agent): commit in lifecycle/config.yaml history")
	}
}

// TestProviderSwitchAPI_RestoreAll (SA4): POST
// .../provider-switch/restore-all restores every agent currently in
// failover (agent-b, agent-c, agent-d) atomically in one call, leaving
// agent-a untouched.
func TestProviderSwitchAPI_RestoreAll(t *testing.T) {
	env := newFailoverTestEnv(t, providerSwitchAPICfgYAML, providerSwitchAPIProviders(t), nil)

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
	env := newFailoverTestEnv(t, providerSwitchAPICfgYAML, providerSwitchAPIProviders(t), nil)

	resp := env.doRequest("POST", "/api/p/testproject/provider-templates/apply", map[string]any{
		"template": "local-ai",
	})
	requireStatus(t, resp, 200)
	body := readJSON(t, resp)
	updated, _ := body["updated_agents"].(float64)
	if updated != 4 {
		t.Errorf("updated_agents: got %v, want 4", updated)
	}

	cfg := env.loadConfig()
	for _, name := range []string{"agent-a", "agent-b", "agent-c", "agent-d"} {
		ag, ok := findFailoverAgentConfig(cfg, name)
		if !ok {
			t.Fatalf("%s missing from disk config after template apply", name)
		}
		if ag.Provider != "local-ollama" || ag.Model != "llama3" {
			t.Errorf("%s provider/model: got %s/%s, want local-ollama/llama3", name, ag.Provider, ag.Model)
		}
	}
}

// TestProviderSwitchAPI_RoleAuth (SA6): a user without product-owner/devops
// roles (qa@test.local, role=analyst only) receives 403 Forbidden on
// provider-switch mutation endpoints.
func TestProviderSwitchAPI_RoleAuth(t *testing.T) {
	env := newFailoverTestEnv(t, providerSwitchAPICfgYAML, providerSwitchAPIProviders(t), nil)
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
