// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Integration tests for the 'raw' artefact status — API round-trip and
// WebSocket event verification.
//
// Test plan: lifecycle/test-plans/raw-artefact-status-5-test.md §Milestone 2

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRawStatus_CreateAndGet verifies that a 'raw' artefact can be created
// via POST /artifacts and retrieved with status == "raw".
//
// Run with: go test ./tests/integration/... -tags=integration -run TestRawStatus_CreateAndGet
func TestRawStatus_CreateAndGet(t *testing.T) {
	env := newTestEnv(t, nil)

	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "ideas",
		"slug":  "raw-capture",
		"frontmatter": map[string]any{
			"title":   "Raw Capture Idea",
			"type":    "idea",
			"status":  "raw",
			"lineage": "raw-capture",
		},
		"body": "Quick brain dump — not yet processed.",
	})
	requireStatus(t, resp, http.StatusCreated)
	data := readJSON(t, resp)

	path, _ := data["path"].(string)
	if path == "" {
		t.Fatal("create response missing 'path'")
	}

	// Retrieve and verify status is "raw".
	getResp := env.doRequest("GET", "/api/p/testproject/artifacts/"+path, nil)
	requireStatus(t, getResp, http.StatusOK)
	getData := readJSON(t, getResp)

	art, _ := getData["artifact"].(map[string]any)
	if art == nil {
		t.Fatal("GET response missing 'artifact'")
	}
	if status, _ := art["status"].(string); status != "raw" {
		t.Errorf("expected status 'raw', got %q", status)
	}
	if typ, _ := art["type"].(string); typ != "idea" {
		t.Errorf("expected type 'idea', got %q", typ)
	}
}

// analystOnlyCfgYAML is a variant of the default config where dev@test.local
// is assigned only the analyst role (no product-owner bypass), so we can
// verify the analyst-specific allowed-targets set from 'raw' status.
const analystOnlyCfgYAML = `git:
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
  - {name: ideas, dir: ideas}
  - {name: requirements, dir: requirements}
  - {name: backend-plans, dir: backend-plans}
  - {name: frontend-plans, dir: frontend-plans}
  - {name: test-plans, dir: test-plans}
  - {name: tests, dir: tests}
  - {name: prototypes, dir: prototypes}
  - {name: releases, dir: releases}
  - {name: sprints, dir: sprints}
  - {name: defects, dir: defects}

users:
  - email: admin@test.local
    roles: [product-owner, reviewer, approver]
  - email: dev@test.local
    roles: [analyst]
  - email: qa@test.local
    roles: [qa]

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []
`

// TestRawStatus_AllowedTargetsForAnalyst verifies that the allowed-targets
// endpoint for a 'raw' artefact returns 'draft' and 'blocked' for a user
// with the analyst role, and excludes 'raw' itself (no self-transition).
// Uses a custom config where dev@test.local has only the analyst role so the
// product-owner superuser bypass does not expand the target set.
//
// Run with: go test ./tests/integration/... -tags=integration -run TestRawStatus_AllowedTargetsForAnalyst
func TestRawStatus_AllowedTargetsForAnalyst(t *testing.T) {
	const relPath = "lifecycle/ideas/raw-targets.md"
	seeds := []seedArtifact{{
		relPath: relPath,
		content: makeArtifact("Raw Targets", "idea", "raw", "raw-targets", "", "Quick capture."),
	}}
	env := newTestEnvWithCfgYAML(t, seeds, analystOnlyCfgYAML)
	env.login("dev@test.local", "dev-pass-123") // analyst role only

	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath+"/allowed-targets", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	rawTargets, ok := data["targets"].([]any)
	if !ok {
		t.Fatalf("expected 'targets' array, got: %v", data)
	}
	targets := make(map[string]bool, len(rawTargets))
	for _, v := range rawTargets {
		if s, ok := v.(string); ok {
			targets[s] = true
		}
	}

	if !targets["draft"] {
		t.Errorf("analyst allowed-targets from raw must include 'draft'; got: %v", rawTargets)
	}
	if !targets["blocked"] {
		t.Errorf("analyst allowed-targets from raw must include 'blocked'; got: %v", rawTargets)
	}
	if targets["raw"] {
		t.Errorf("analyst allowed-targets from raw must NOT include 'raw' (no self-transition); got: %v", rawTargets)
	}
}

// TestRawStatus_TransitionRawToDraft verifies that an analyst can transition
// a 'raw' artefact to 'draft', that the status is updated on disk, and that
// a git commit records the change.
//
// Run with: go test ./tests/integration/... -tags=integration -run TestRawStatus_TransitionRawToDraft
func TestRawStatus_TransitionRawToDraft(t *testing.T) {
	const relPath = "lifecycle/ideas/raw-to-draft.md"
	seeds := []seedArtifact{{
		relPath: relPath,
		content: makeArtifact("Raw To Draft", "idea", "raw", "raw-to-draft", "", "Quick capture."),
	}}
	env := newTestEnv(t, seeds)
	// admin@test.local has analyst role

	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+relPath+"/transition", map[string]any{
		"to": "draft",
	})
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	art, _ := data["artifact"].(map[string]any)
	if status, _ := art["status"].(string); status != "draft" {
		t.Errorf("expected status 'draft' after transition, got %q", status)
	}

	// Verify on disk.
	fileBytes, err := os.ReadFile(filepath.Join(env.projectRoot, relPath))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(fileBytes), "status: draft") {
		t.Error("status not updated to 'draft' on disk")
	}

	// Verify a git commit was recorded.
	commits, err := env.proj.Git.Log(relPath, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) < 2 {
		t.Error("expected at least 2 commits (initial + transition)")
	}
}

// TestRawStatus_WSEventOnTransition verifies that transitioning raw → draft
// broadcasts an artifact.indexed WebSocket event whose payload indicates the
// status reached "draft".
//
// Run with: go test ./tests/integration/... -tags=integration -run TestRawStatus_WSEventOnTransition
func TestRawStatus_WSEventOnTransition(t *testing.T) {
	const relPath = "lifecycle/ideas/raw-ws.md"
	seeds := []seedArtifact{{
		relPath: relPath,
		content: makeArtifact("Raw WS", "idea", "raw", "raw-ws", "", "Quick capture."),
	}}
	env := newTestEnv(t, seeds)

	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	t.Cleanup(func() { env.proj.Hub.Unregister(ch) })

	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+relPath+"/transition", map[string]any{
		"to": "draft",
	})
	requireStatus(t, resp, http.StatusOK)

	type wsEvent struct {
		Type    string         `json:"type"`
		Payload map[string]any `json:"payload"`
	}

	// Wait up to 2s for the artifact.indexed event. Using channel select rather
	// than time.Sleep so the assertion completes as soon as the event arrives.
	deadline := time.After(2 * time.Second)
	for {
		select {
		case raw := <-ch:
			var evt wsEvent
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Type != "artifact.indexed" {
				continue
			}
			p, _ := evt.Payload["path"].(string)
			action, _ := evt.Payload["action"].(string)
			to, _ := evt.Payload["to"].(string)
			if p == relPath && action == "transitioned" && to == "draft" {
				return // success
			}
		case <-deadline:
			t.Error("did not receive artifact.indexed{action:transitioned,to:draft} within 2s")
			return
		}
	}
}

// TestRawStatus_NonAnalystCantTransitionToDraft verifies that a user without
// the analyst role receives 403 when attempting to transition raw → draft.
//
// Run with: go test ./tests/integration/... -tags=integration -run TestRawStatus_NonAnalystCantTransitionToDraft
func TestRawStatus_NonAnalystCantTransitionToDraft(t *testing.T) {
	const relPath = "lifecycle/ideas/raw-forbidden.md"
	seeds := []seedArtifact{{
		relPath: relPath,
		content: makeArtifact("Raw Forbidden", "idea", "raw", "raw-forbidden", "", "Quick capture."),
	}}
	env := newTestEnv(t, seeds)

	env.login("dev@test.local", "dev-pass-123") // backend-developer only; no analyst role
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+relPath+"/transition", map[string]any{
		"to": "draft",
	})
	requireStatus(t, resp, http.StatusForbidden)
	data := readJSON(t, resp)
	errData, _ := data["error"].(map[string]any)
	if code, _ := errData["code"].(string); code != "forbidden" {
		t.Errorf("expected error code 'forbidden', got %q", code)
	}
}
