// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// readRawBody reads and closes the response body, returning it as a string.
// Unlike readJSON, it does not attempt to decode JSON, so it is safe to use
// for substring assertions (e.g. verifying a secret literal is absent).
func readRawBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// agentEditorFullConfigCfgYAML backs Milestone 1: one agent ("full-config-agent")
// exercising every non-secret field the -3-be plan's Milestone 1 table lists,
// and a second claude-env agent ("claude-env-agent") carrying an auth_token
// that must never appear in the GET /agents response (NFR-3).
const agentEditorFullConfigCfgYAML = `git:
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
  - name: full-config-agent
    role: [backend-developer]
    driver: claude-mediated
    model: claude-opus-4-6
    active_status: in-development
    done_on_success: true
    source_types: [ticket]
    allowed_write_paths:
      - internal
      - cmd
    timeout_minutes: 45
    git_identity:
      name: Full Config Agent
      email: full-config-agent@test.local
    prompt_templates:
      backend-developer: "BE prompt for {target_path}"
      reviewer: "Review prompt for {target_path}"
      qa: "QA prompt for {target_path}"
    bash_allowlist:
      - "go test *"
      - "go build *"
    bash_denylist:
      - "rm *"
    on_denial: abort
    observe_only: true

  - name: claude-env-agent
    role: [analyst]
    driver: claude-env
    model: claude-opus-4-6
    base_url: "http://localhost:9999"
    auth_token: "s3cr3t-token-must-not-appear"
    prompt_templates:
      analyst: "Analyst prompt for {target_path}"
`

// TestAgentEditorConfig_ListReturnsFullNonSecretConfig covers test plan
// Milestone 1: GET /api/p/{project}/agents must return every non-secret
// field of a richly-configured agent, with all prompt_templates keys intact,
// and must never leak the auth_token of a claude-env agent.
func TestAgentEditorConfig_ListReturnsFullNonSecretConfig(t *testing.T) {
	env := newAgentTestEnvWithCfg(t, agentEditorFullConfigCfgYAML, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/agents", nil)
	requireStatus(t, resp, 200)

	// Capture the raw body before JSON-decoding so we can also assert on the
	// literal absence of "auth_token" and its value (NFR-3, byte-level check).
	data := readJSON(t, resp)

	agentsRaw, _ := data["agents"].([]any)
	byName := make(map[string]map[string]any, len(agentsRaw))
	for _, raw := range agentsRaw {
		ag, _ := raw.(map[string]any)
		name, _ := ag["name"].(string)
		byName[name] = ag
	}

	rich, ok := byName["full-config-agent"]
	if !ok {
		t.Fatal("full-config-agent not found in GET /agents response")
	}

	if got, _ := rich["timeout_minutes"].(float64); got != 45 {
		t.Errorf("timeout_minutes: got %v, want 45", rich["timeout_minutes"])
	}
	gitIdentity, _ := rich["git_identity"].(map[string]any)
	if gitIdentity == nil {
		t.Fatal("git_identity: got nil, want non-nil")
	}
	if name, _ := gitIdentity["name"].(string); name != "Full Config Agent" {
		t.Errorf("git_identity.name: got %q, want %q", name, "Full Config Agent")
	}
	if email, _ := gitIdentity["email"].(string); email != "full-config-agent@test.local" {
		t.Errorf("git_identity.email: got %q, want %q", email, "full-config-agent@test.local")
	}
	templates, _ := rich["prompt_templates"].(map[string]any)
	wantTemplates := map[string]string{
		"backend-developer": "BE prompt for {target_path}",
		"reviewer":          "Review prompt for {target_path}",
		"qa":                "QA prompt for {target_path}",
	}
	if len(templates) != len(wantTemplates) {
		t.Fatalf("prompt_templates: got %d keys (%v), want %d keys (%v)",
			len(templates), templates, len(wantTemplates), wantTemplates)
	}
	for k, want := range wantTemplates {
		if got, _ := templates[k].(string); got != want {
			t.Errorf("prompt_templates[%q]: got %q, want %q", k, got, want)
		}
	}
	if sourceTypes, _ := rich["source_types"].([]any); len(sourceTypes) != 1 || sourceTypes[0] != "ticket" {
		t.Errorf("source_types: got %v, want [ticket]", rich["source_types"])
	}
	if done, _ := rich["done_on_success"].(bool); !done {
		t.Errorf("done_on_success: got %v, want true", rich["done_on_success"])
	}
	if onDenial, _ := rich["on_denial"].(string); onDenial != "abort" {
		t.Errorf("on_denial: got %q, want %q", onDenial, "abort")
	}
	if observeOnly, _ := rich["observe_only"].(bool); !observeOnly {
		t.Errorf("observe_only: got %v, want true", rich["observe_only"])
	}
	if allowlist, _ := rich["bash_allowlist"].([]any); len(allowlist) != 2 {
		t.Errorf("bash_allowlist: got %v, want 2 entries", rich["bash_allowlist"])
	}
	if denylist, _ := rich["bash_denylist"].([]any); len(denylist) != 1 || denylist[0] != "rm *" {
		t.Errorf("bash_denylist: got %v, want [\"rm *\"]", rich["bash_denylist"])
	}

	// Secret hygiene: the claude-env agent's auth_token must never appear.
	claudeEnv, ok := byName["claude-env-agent"]
	if !ok {
		t.Fatal("claude-env-agent not found in GET /agents response")
	}
	if _, present := claudeEnv["auth_token"]; present {
		t.Errorf("claude-env-agent response contains auth_token field: %v", claudeEnv)
	}
	if baseURL, _ := claudeEnv["base_url"].(string); baseURL != "http://localhost:9999" {
		t.Errorf("claude-env-agent base_url: got %q, want %q", baseURL, "http://localhost:9999")
	}

	resp2 := env.doRequest("GET", "/api/p/testproject/agents", nil)
	requireStatus(t, resp2, 200)
	rawBody := readRawBody(t, resp2)
	if strings.Contains(rawBody, "s3cr3t-token-must-not-appear") {
		t.Error("auth_token value literal found in raw GET /agents response body")
	}
	if strings.Contains(rawBody, "auth_token") {
		t.Error(`"auth_token" key found in raw GET /agents response body`)
	}
}

// agentEditorRoundTripCfgYAML backs Milestone 4: a single, fully-populated
// agent ("round-trip-agent") including auth_token, so the test can drive a
// GET /agents -> edit -> PUT /config -> GET /config round trip and assert
// nothing not managed by the editor form is lost, and the secret never moves
// through the client-visible API.
const agentEditorRoundTripCfgYAML = `git:
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
  - name: round-trip-agent
    role: [backend-developer]
    driver: claude-env
    model: claude-opus-4-6
    active_status: in-development
    done_on_success: true
    source_types: [ticket]
    allowed_write_paths:
      - internal
    timeout_minutes: 30
    git_identity:
      name: Round Trip Agent
      email: round-trip-agent@test.local
    prompt_templates:
      backend-developer: "BE prompt for {target_path}"
    bash_allowlist:
      - "go test *"
    bash_denylist:
      - "rm *"
    on_denial: continue
    observe_only: false
    endpoint: "https://legacy.example.com/endpoint"
    base_url: "http://localhost:9999"
    auth_token: "s3cr3t-token-must-survive-untouched"
`

// editorFormManagedKeys mirrors the set of config.yaml keys AgentConfigForm.vue
// exposes and AgentsRunsView.vue's handleAgentFormSubmit assigns onto the
// existing entry on save (see -4-fe Milestone 4). Everything else on the
// existing entry must survive untouched.
var editorFormManagedKeys = []string{
	"name", "role", "driver", "model", "allowed_write_paths",
	"timeout_minutes", "git_identity", "prompt_templates",
	"ollama_instance", "ollama_endpoint",
}

// mergeAgentEntryLikeEditor reproduces handleAgentFormSubmit's merge-on-save
// behaviour (web/src/views/project/AgentsRunsView.vue): copy the existing
// entry, then overwrite only the form-managed keys, deleting ones the form
// cleared. newModel simulates the user editing just the model field.
func mergeAgentEntryLikeEditor(existing map[string]any, newModel string) map[string]any {
	entry := make(map[string]any, len(existing))
	for k, v := range existing {
		entry[k] = v
	}

	entry["name"] = existing["name"]
	entry["role"] = existing["role"]
	entry["driver"] = existing["driver"]

	if newModel != "" {
		entry["model"] = newModel
	} else {
		delete(entry, "model")
	}

	if tm, ok := existing["timeout_minutes"]; ok {
		entry["timeout_minutes"] = tm
	}

	if paths, ok := existing["allowed_write_paths"]; ok {
		entry["allowed_write_paths"] = paths
	} else {
		delete(entry, "allowed_write_paths")
	}

	if gi, ok := existing["git_identity"]; ok {
		entry["git_identity"] = gi
	} else {
		delete(entry, "git_identity")
	}

	if pt, ok := existing["prompt_templates"]; ok {
		entry["prompt_templates"] = pt
	} else {
		delete(entry, "prompt_templates")
	}

	return entry
}

// TestAgentEditorConfig_RoundTripPreservesNonExposedFields covers test plan
// Milestone 4 (NFR-2's cross-layer guard): read the agent-read API, apply a
// single-field edit through the same merge rules the frontend fix uses,
// PUT the resulting config.yaml, reload, and re-read. Every field of the
// original agent entry that the editor does not expose must be byte-identical
// on disk, and auth_token must never appear in any GET /agents response.
func TestAgentEditorConfig_RoundTripPreservesNonExposedFields(t *testing.T) {
	env := newAgentTestEnvWithCfg(t, agentEditorRoundTripCfgYAML, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Sanity: the read API never leaks the secret before we even touch it.
	preResp := env.doRequest("GET", "/api/p/testproject/agents", nil)
	requireStatus(t, preResp, 200)
	preBody := readRawBody(t, preResp)
	if strings.Contains(preBody, "s3cr3t-token-must-survive-untouched") || strings.Contains(preBody, "auth_token") {
		t.Fatal("auth_token leaked in GET /agents before any edit")
	}

	// "Open": read the raw config.yaml, as the editor's save path does.
	getResp := env.doRequest("GET", "/api/p/testproject/config", nil)
	requireStatus(t, getResp, 200)
	getData := readJSON(t, getResp)
	rawBefore, _ := getData["raw"].(string)

	var cfgBefore map[string]any
	if err := yaml.Unmarshal([]byte(rawBefore), &cfgBefore); err != nil {
		t.Fatalf("parsing config.yaml: %v", err)
	}
	agentsBefore, _ := cfgBefore["agents"].([]any)
	var existingEntry map[string]any
	idx := -1
	for i, raw := range agentsBefore {
		ag, _ := raw.(map[string]any)
		if ag["name"] == "round-trip-agent" {
			existingEntry = ag
			idx = i
			break
		}
	}
	if existingEntry == nil {
		t.Fatal("round-trip-agent not found in parsed config.yaml")
	}
	originalAuthToken := existingEntry["auth_token"]
	originalActiveStatus := existingEntry["active_status"]
	originalDoneOnSuccess := existingEntry["done_on_success"]
	originalSourceTypes := existingEntry["source_types"]
	originalOnDenial := existingEntry["on_denial"]
	originalObserveOnly := existingEntry["observe_only"]
	originalBashAllowlist := existingEntry["bash_allowlist"]
	originalBashDenylist := existingEntry["bash_denylist"]
	originalEndpoint := existingEntry["endpoint"]
	originalBaseURL := existingEntry["base_url"]

	// "Edit": change only the model field, via the same merge rules the
	// frontend fix applies (merge onto existing, not replace).
	mergedEntry := mergeAgentEntryLikeEditor(existingEntry, "claude-opus-5-1")
	agentsBefore[idx] = mergedEntry
	cfgBefore["agents"] = agentsBefore

	rawAfterBytes, err := yaml.Marshal(cfgBefore)
	if err != nil {
		t.Fatalf("marshalling config.yaml: %v", err)
	}

	// "Save": PUT the merged config.yaml.
	putResp := env.doRequest("PUT", "/api/p/testproject/config", map[string]any{
		"raw": string(rawAfterBytes),
	})
	requireStatus(t, putResp, 200)

	// "Reload": re-read config.yaml from disk via the API.
	reReadResp := env.doRequest("GET", "/api/p/testproject/config", nil)
	requireStatus(t, reReadResp, 200)
	reReadData := readJSON(t, reReadResp)
	rawAfter, _ := reReadData["raw"].(string)

	var cfgAfter map[string]any
	if err := yaml.Unmarshal([]byte(rawAfter), &cfgAfter); err != nil {
		t.Fatalf("parsing post-save config.yaml: %v", err)
	}
	agentsAfter, _ := cfgAfter["agents"].([]any)
	var reReadEntry map[string]any
	for _, raw := range agentsAfter {
		ag, _ := raw.(map[string]any)
		if ag["name"] == "round-trip-agent" {
			reReadEntry = ag
			break
		}
	}
	if reReadEntry == nil {
		t.Fatal("round-trip-agent not found in post-save config.yaml")
	}

	// The edited field changed...
	if got, _ := reReadEntry["model"].(string); got != "claude-opus-5-1" {
		t.Errorf("model after save: got %q, want %q", got, "claude-opus-5-1")
	}

	// ...and every non-exposed field is byte-identical to before the edit.
	if got := reReadEntry["auth_token"]; got != originalAuthToken {
		t.Errorf("auth_token: got %v, want %v (must survive untouched)", got, originalAuthToken)
	}
	if got := reReadEntry["active_status"]; got != originalActiveStatus {
		t.Errorf("active_status: got %v, want %v", got, originalActiveStatus)
	}
	if got := reReadEntry["done_on_success"]; got != originalDoneOnSuccess {
		t.Errorf("done_on_success: got %v, want %v", got, originalDoneOnSuccess)
	}
	if fmtSlice(reReadEntry["source_types"]) != fmtSlice(originalSourceTypes) {
		t.Errorf("source_types: got %v, want %v", reReadEntry["source_types"], originalSourceTypes)
	}
	if got := reReadEntry["on_denial"]; got != originalOnDenial {
		t.Errorf("on_denial: got %v, want %v", got, originalOnDenial)
	}
	if got := reReadEntry["observe_only"]; got != originalObserveOnly {
		t.Errorf("observe_only: got %v, want %v", got, originalObserveOnly)
	}
	if fmtSlice(reReadEntry["bash_allowlist"]) != fmtSlice(originalBashAllowlist) {
		t.Errorf("bash_allowlist: got %v, want %v", reReadEntry["bash_allowlist"], originalBashAllowlist)
	}
	if fmtSlice(reReadEntry["bash_denylist"]) != fmtSlice(originalBashDenylist) {
		t.Errorf("bash_denylist: got %v, want %v", reReadEntry["bash_denylist"], originalBashDenylist)
	}
	if got := reReadEntry["endpoint"]; got != originalEndpoint {
		t.Errorf("endpoint: got %v, want %v", got, originalEndpoint)
	}
	if got := reReadEntry["base_url"]; got != originalBaseURL {
		t.Errorf("base_url: got %v, want %v", got, originalBaseURL)
	}

	// The secret must never have appeared in any GET /agents response,
	// before or after the save.
	postResp := env.doRequest("GET", "/api/p/testproject/agents", nil)
	requireStatus(t, postResp, 200)
	postBody := readRawBody(t, postResp)
	if strings.Contains(postBody, "s3cr3t-token-must-survive-untouched") {
		t.Error("auth_token value literal found in GET /agents response body after save")
	}
	if strings.Contains(postBody, "auth_token") {
		t.Error(`"auth_token" key found in GET /agents response body after save`)
	}
}

// fmtSlice renders a []any (or nil) as a comparable string, for equality
// checks on YAML-decoded list fields where reflect.DeepEqual would be
// brittle across []any vs []string interface boxing.
func fmtSlice(v any) string {
	s, _ := v.([]any)
	parts := make([]string, 0, len(s))
	for _, e := range s {
		parts = append(parts, toStr(e))
	}
	return strings.Join(parts, ",")
}

func toStr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
