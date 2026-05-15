// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestone 3 — Permission Endpoint Integration Tests
//
// HTTP-level tests for POST /api/agent/{run_id}/permission. A real Manager is
// wired via the test environment; run secrets and policies are injected directly
// into the Manager without spawning an agent process. This isolates the HTTP
// handler, authentication, policy evaluation, WS broadcast, and denial recording
// from subprocess concerns.
//
// Run with:
//   go test ./tests/... -tags integration -run TestPermission -v

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/agent"
)

// hookPermCfgYAML is a lifecycle/config.yaml with a single claude-mediated agent
// so that the Manager is non-nil. Permission tests never actually start the driver;
// they call StoreRunSecret / StoreRunPolicy directly.
const hookPermCfgYAML = `git:
  default_branch: main
  branch_template: "ticket/{slug}"

roles:
  - product-owner
  - backend-developer

stages:
  - {name: ideas,         dir: ideas}
  - {name: requirements,  dir: requirements}
  - {name: backend-plans, dir: backend-plans}
  - {name: frontend-plans,dir: frontend-plans}
  - {name: test-plans,    dir: test-plans}
  - {name: tests,         dir: tests}
  - {name: prototypes,    dir: prototypes}
  - {name: releases,      dir: releases}
  - {name: sprints,       dir: sprints}
  - {name: defects,       dir: defects}

users:
  - email: admin@test.local
    roles: [product-owner]
  - email: dev@test.local
    roles: [backend-developer]

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []

agents:
  - name: perm-test-agent
    role: [backend-developer]
    driver: claude-mediated
    active_status: in-development
    allowed_write_paths:
      - lifecycle/requirements
    git_identity:
      name: Perm Test Agent
      email: perm-test-agent@test.local
    prompt_templates:
      backend-developer: "Test prompt for {target_path}"
`

// newPermTestEnv returns a test environment wired with hookPermCfgYAML so the
// Manager is non-nil. No agents are actually started.
func newPermTestEnv(t *testing.T) *testEnv {
	t.Helper()
	return newAgentTestEnvWithCfg(t, hookPermCfgYAML, nil)
}

// seedPermRun injects a per-run secret and policy directly into the Manager so
// the permission endpoint can authenticate and evaluate requests without an
// active agent process.
func seedPermRun(t *testing.T, env *testEnv, runID, secret string, policy *agent.PolicyConfig) {
	t.Helper()
	if env.proj.Agents == nil {
		t.Fatal("env.proj.Agents is nil: check that lifecycle/config.yaml contains at least one agent")
	}
	env.proj.Agents.StoreRunSecret(runID, secret)
	env.proj.Agents.StoreRunPolicy(runID, policy)
}

// doPermissionRequest POSTs a permission request to the endpoint and returns the response.
// secret may be empty to simulate a missing Authorization header.
func doPermissionRequest(t *testing.T, env *testEnv, runID, secret string, body map[string]any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost,
		env.baseURL+"/api/agent/"+runID+"/permission",
		bytes.NewReader(b))
	if err != nil {
		t.Fatalf("building permission request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("executing permission request: %v", err)
	}
	return resp
}

// permDecision reads the response body as JSON and returns the "decision" field.
func permDecision(t *testing.T, resp *http.Response) string {
	t.Helper()
	data := readJSON(t, resp)
	dec, _ := data["decision"].(string)
	return dec
}

// ── Authentication ────────────────────────────────────────────────────────────

// TestPermission_CorrectSecret_Returns200 verifies that a request with the
// correct per-run secret is accepted (FR8, AC12).
func TestPermission_CorrectSecret_Returns200(t *testing.T) {
	env := newPermTestEnv(t)
	seedPermRun(t, env, "run-auth-ok", "secret-ok-abc", &agent.PolicyConfig{})

	resp := doPermissionRequest(t, env, "run-auth-ok", "secret-ok-abc", map[string]any{
		"tool_name":  "Read",
		"tool_input": map[string]any{},
	})
	defer resp.Body.Close()
	requireStatus(t, resp, http.StatusOK)
}

// TestPermission_WrongSecret_Returns403 verifies that a request with an
// incorrect secret is rejected (FR8, AC12).
func TestPermission_WrongSecret_Returns403(t *testing.T) {
	env := newPermTestEnv(t)
	seedPermRun(t, env, "run-auth-wrong", "correct-secret", &agent.PolicyConfig{})

	resp := doPermissionRequest(t, env, "run-auth-wrong", "wrong-secret", map[string]any{
		"tool_name":  "Read",
		"tool_input": map[string]any{},
	})
	defer resp.Body.Close()
	requireStatus(t, resp, http.StatusForbidden)
}

// TestPermission_MissingSecret_Returns403 verifies that a request without any
// Authorization header is rejected (FR8, AC12).
func TestPermission_MissingSecret_Returns403(t *testing.T) {
	env := newPermTestEnv(t)
	seedPermRun(t, env, "run-auth-nosecret", "some-secret", &agent.PolicyConfig{})

	resp := doPermissionRequest(t, env, "run-auth-nosecret", "" /* no secret */, map[string]any{
		"tool_name":  "Read",
		"tool_input": map[string]any{},
	})
	defer resp.Body.Close()
	requireStatus(t, resp, http.StatusForbidden)
}

// TestPermission_UnknownRunID_Returns400 verifies that an unknown run_id
// results in a 400 (the Manager has no policy for it).
func TestPermission_UnknownRunID_Returns400(t *testing.T) {
	env := newPermTestEnv(t)
	// Intentionally do NOT call seedPermRun.

	resp := doPermissionRequest(t, env, "run-does-not-exist-xyz", "any-secret", map[string]any{
		"tool_name":  "Read",
		"tool_input": map[string]any{},
	})
	defer resp.Body.Close()
	requireStatus(t, resp, http.StatusBadRequest)
}

// ── Request validation ────────────────────────────────────────────────────────

// TestPermission_EmptyBody_Returns400 verifies that an empty request body
// produces a 400 (FR8).
func TestPermission_EmptyBody_Returns400(t *testing.T) {
	env := newPermTestEnv(t)
	seedPermRun(t, env, "run-empty-body", "secret-eb", &agent.PolicyConfig{})

	req, _ := http.NewRequest(http.MethodPost,
		env.baseURL+"/api/agent/run-empty-body/permission",
		bytes.NewReader([]byte("")))
	req.Header.Set("Authorization", "Bearer secret-eb")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	requireStatus(t, resp, http.StatusBadRequest)
}

// TestPermission_ValidBody_Returns200 verifies that a well-formed request body
// is accepted (FR8).
func TestPermission_ValidBody_Returns200(t *testing.T) {
	env := newPermTestEnv(t)
	seedPermRun(t, env, "run-valid-body", "secret-vb", &agent.PolicyConfig{})

	resp := doPermissionRequest(t, env, "run-valid-body", "secret-vb", map[string]any{
		"tool_name":  "Read",
		"tool_input": map[string]any{"file_path": "lifecycle/requirements/foo.md"},
	})
	defer resp.Body.Close()
	requireStatus(t, resp, http.StatusOK)
}

// ── Policy decisions ──────────────────────────────────────────────────────────

// TestPermission_AllowedWrite_ReturnsAllow verifies that a Write to an allowed
// path produces {"decision":"allow"} (FR7, AC3).
func TestPermission_AllowedWrite_ReturnsAllow(t *testing.T) {
	env := newPermTestEnv(t)
	seedPermRun(t, env, "run-write-allow", "secret-wa", &agent.PolicyConfig{
		AllowedPaths: []string{"lifecycle/requirements"},
	})

	resp := doPermissionRequest(t, env, "run-write-allow", "secret-wa", map[string]any{
		"tool_name":  "Write",
		"tool_input": map[string]any{"file_path": "lifecycle/requirements/foo.md"},
	})
	defer resp.Body.Close()
	requireStatus(t, resp, http.StatusOK)
	if dec := permDecision(t, resp); dec != "allow" {
		t.Errorf("decision = %q, want allow", dec)
	}
}

// TestPermission_DisallowedWrite_ReturnsDeny verifies that a Write outside the
// allowed paths produces {"decision":"deny","reason":"..."} (FR7, AC3, AC4).
func TestPermission_DisallowedWrite_ReturnsDeny(t *testing.T) {
	env := newPermTestEnv(t)
	seedPermRun(t, env, "run-write-deny", "secret-wd", &agent.PolicyConfig{
		AllowedPaths: []string{"lifecycle/requirements"},
	})

	resp := doPermissionRequest(t, env, "run-write-deny", "secret-wd", map[string]any{
		"tool_name":  "Write",
		"tool_input": map[string]any{"file_path": "web/src/App.vue"},
	})
	defer resp.Body.Close()
	requireStatus(t, resp, http.StatusOK)

	data := readJSON(t, resp)
	if dec, _ := data["decision"].(string); dec != "deny" {
		t.Errorf("decision = %q, want deny", dec)
	}
	if reason, _ := data["reason"].(string); reason == "" {
		t.Error("expected non-empty reason on deny")
	}
}

// TestPermission_BashDenylist_ReturnsDeny verifies that a Bash command matching
// the default denylist is denied (FR11, AC5).
func TestPermission_BashDenylist_ReturnsDeny(t *testing.T) {
	env := newPermTestEnv(t)
	seedPermRun(t, env, "run-bash-deny", "secret-bd", &agent.PolicyConfig{
		BashDenylist: agent.DefaultBashDenylist,
	})

	resp := doPermissionRequest(t, env, "run-bash-deny", "secret-bd", map[string]any{
		"tool_name":  "Bash",
		"tool_input": map[string]any{"command": "sudo rm -rf /"},
	})
	defer resp.Body.Close()
	requireStatus(t, resp, http.StatusOK)
	if dec := permDecision(t, resp); dec != "deny" {
		t.Errorf("decision = %q, want deny for bash denylist match", dec)
	}
}

// TestPermission_ReadTools_ReturnAllow verifies that read-only tools are always
// allowed regardless of path restrictions (FR13).
func TestPermission_ReadTools_ReturnAllow(t *testing.T) {
	env := newPermTestEnv(t)
	// Tight policy: no allowed paths, denylist active — read tools should still pass.
	seedPermRun(t, env, "run-read-allow", "secret-ra", &agent.PolicyConfig{
		AllowedPaths: []string{"lifecycle/requirements"},
		BashDenylist: agent.DefaultBashDenylist,
	})

	for _, tool := range []string{"Read", "Glob", "Grep", "WebFetch"} {
		t.Run(tool, func(t *testing.T) {
			resp := doPermissionRequest(t, env, "run-read-allow", "secret-ra", map[string]any{
				"tool_name":  tool,
				"tool_input": map[string]any{},
			})
			defer resp.Body.Close()
			requireStatus(t, resp, http.StatusOK)
			if dec := permDecision(t, resp); dec != "allow" {
				t.Errorf("tool=%s: decision = %q, want allow", tool, dec)
			}
		})
	}
}

// ── Observe-only mode ─────────────────────────────────────────────────────────

// TestPermission_ObserveOnly_AlwaysAllow verifies that observe-only mode returns
// allow even when the policy would normally deny (FR17, AC7).
func TestPermission_ObserveOnly_AlwaysAllow(t *testing.T) {
	env := newPermTestEnv(t)
	seedPermRun(t, env, "run-observe-only", "secret-obs", &agent.PolicyConfig{
		AllowedPaths: []string{"lifecycle/requirements"},
		ObserveOnly:  true,
	})

	// Write to a path that is NOT in AllowedPaths — should still return allow.
	resp := doPermissionRequest(t, env, "run-observe-only", "secret-obs", map[string]any{
		"tool_name":  "Write",
		"tool_input": map[string]any{"file_path": "web/src/App.vue"},
	})
	defer resp.Body.Close()
	requireStatus(t, resp, http.StatusOK)
	if dec := permDecision(t, resp); dec != "allow" {
		t.Errorf("observe-only: decision = %q, want allow even for disallowed path", dec)
	}
}

// ── Denial recording ──────────────────────────────────────────────────────────

// TestPermission_Denial_RecordedInManager verifies that a denied tool call is
// added to Manager.DeniedCalls (FR14, FR15).
func TestPermission_Denial_RecordedInManager(t *testing.T) {
	env := newPermTestEnv(t)
	const runID = "run-denial-record"
	seedPermRun(t, env, runID, "secret-dr", &agent.PolicyConfig{
		AllowedPaths: []string{"lifecycle/requirements"},
	})

	resp := doPermissionRequest(t, env, runID, "secret-dr", map[string]any{
		"tool_name":  "Write",
		"tool_input": map[string]any{"file_path": "web/src/App.vue"},
	})
	defer resp.Body.Close()
	requireStatus(t, resp, http.StatusOK)

	// Verify the denial was recorded in the Manager.
	denials := env.proj.Agents.DeniedCalls(runID)
	if len(denials) == 0 {
		t.Fatal("expected at least one denial record after a denied write, got none")
	}
	if denials[0].ToolName != "Write" {
		t.Errorf("denial[0].ToolName = %q, want Write", denials[0].ToolName)
	}
	if denials[0].Path == "" {
		t.Error("denial[0].Path is empty; expected the denied file path to be recorded")
	}
}

// ── WebSocket event ───────────────────────────────────────────────────────────

// TestPermission_BroadcastsWSEvent verifies that every permission decision
// broadcasts an "agent.permission" event on the project hub (FR20).
func TestPermission_BroadcastsWSEvent(t *testing.T) {
	env := newPermTestEnv(t)
	const runID = "run-ws-event"
	seedPermRun(t, env, runID, "secret-ws", &agent.PolicyConfig{
		AllowedPaths: []string{"lifecycle/requirements"},
	})

	ch := make(chan []byte, 32)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	resp := doPermissionRequest(t, env, runID, "secret-ws", map[string]any{
		"tool_name":  "Write",
		"tool_input": map[string]any{"file_path": "lifecycle/requirements/foo.md"},
	})
	defer resp.Body.Close()
	requireStatus(t, resp, http.StatusOK)

	// Wait for the agent.permission event on the hub.
	deadline := time.After(3 * time.Second)
	for {
		select {
		case raw := <-ch:
			var evt struct {
				Type    string         `json:"type"`
				Payload map[string]any `json:"payload"`
			}
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Type != "agent.permission" {
				continue
			}
			rid, _ := evt.Payload["run_id"].(string)
			if rid != runID {
				continue
			}
			// Verify the event has the expected fields.
			if dec, _ := evt.Payload["decision"].(string); dec != "allow" {
				t.Errorf("ws event decision = %q, want allow", dec)
			}
			if tn, _ := evt.Payload["tool_name"].(string); tn != "Write" {
				t.Errorf("ws event tool_name = %q, want Write", tn)
			}
			return // success
		case <-deadline:
			t.Fatal("timed out waiting for agent.permission WS event")
		}
	}
}

// TestPermission_DenyEvent_BroadcastedOnHub verifies that a denied tool call
// also produces an agent.permission WS event (with decision=deny).
func TestPermission_DenyEvent_BroadcastedOnHub(t *testing.T) {
	env := newPermTestEnv(t)
	const runID = "run-ws-deny-event"
	seedPermRun(t, env, runID, "secret-wsd", &agent.PolicyConfig{
		AllowedPaths: []string{"lifecycle/requirements"},
	})

	ch := make(chan []byte, 32)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	// Write to a disallowed path.
	resp := doPermissionRequest(t, env, runID, "secret-wsd", map[string]any{
		"tool_name":  "Edit",
		"tool_input": map[string]any{"file_path": "web/src/App.vue"},
	})
	defer resp.Body.Close()
	requireStatus(t, resp, http.StatusOK)

	deadline := time.After(3 * time.Second)
	for {
		select {
		case raw := <-ch:
			var evt struct {
				Type    string         `json:"type"`
				Payload map[string]any `json:"payload"`
			}
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Type != "agent.permission" {
				continue
			}
			if rid, _ := evt.Payload["run_id"].(string); rid != runID {
				continue
			}
			if dec, _ := evt.Payload["decision"].(string); dec != "deny" {
				t.Errorf("ws event decision = %q, want deny", dec)
			}
			return
		case <-deadline:
			t.Fatal("timed out waiting for agent.permission deny WS event")
		}
	}
}
