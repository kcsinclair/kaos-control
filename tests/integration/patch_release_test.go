// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// cfgYAMLWithQAAsProductOwner is the default config but gives qa@test.local
// the product-owner role so it can call the PATCH /release endpoint. This is
// needed for the lock-conflict test, which requires two users that both pass
// the RolesReleaseEditors check.
const cfgYAMLWithQAAsProductOwner = `git:
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
    roles: [product-owner, analyst, reviewer, approver]
  - email: dev@test.local
    roles: [backend-developer, frontend-developer, test-developer]
  - email: qa@test.local
    roles: [qa, product-owner]

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []
`

// patchRelease sends PATCH .../release with the given body and returns the
// HTTP response. Pass releaseName="" to set release to the zero-value string;
// pass nil to omit the field entirely (which results in null in JSON and clears
// the release field on the server).
func patchRelease(env *testEnv, artifactPath string, release *string) *http.Response {
	var body map[string]any
	if release == nil {
		body = map[string]any{"release": nil}
	} else {
		body = map[string]any{"release": *release}
	}
	return env.doRequest("PATCH", "/api/p/testproject/artifacts/"+artifactPath+"/release", body)
}

// strPtr is a convenience helper to obtain a *string from a string literal.
func strPtr(s string) *string { return &s }

// ── Milestone 1: PATCH /artifacts/*/release ───────────────────────────────────

// TestReleasePatch_SetRelease verifies the happy path: create an artifact with
// no release, create a release in the DB, PATCH with that release name →
// 200, response body has the updated release, and GET confirms the update.
func TestReleasePatch_SetRelease(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rp-set.md",
			content: makeArtifact("Release Patch Set", "idea", "draft", "rp-set", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{"name": "v1.0-set", "status": "planned"})

	const path = "lifecycle/ideas/rp-set.md"
	resp := patchRelease(env, path, strPtr("v1.0-set"))
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	artifact, _ := data["artifact"].(map[string]any)
	if artifact == nil {
		t.Fatal("expected artifact in response")
	}
	fm, _ := artifact["frontmatter"].(map[string]any)
	if release, _ := fm["release"].(string); release != "v1.0-set" {
		t.Errorf("response release: want %q, got %q", "v1.0-set", release)
	}

	// GET confirms the update.
	resp2 := env.doRequest("GET", "/api/p/testproject/artifacts/"+path, nil)
	requireStatus(t, resp2, 200)
	data2 := readJSON(t, resp2)
	artifact2, _ := data2["artifact"].(map[string]any)
	fm2, _ := artifact2["frontmatter"].(map[string]any)
	if release, _ := fm2["release"].(string); release != "v1.0-set" {
		t.Errorf("GET release after PATCH: want %q, got %q", "v1.0-set", release)
	}
}

// TestReleasePatch_ChangeRelease verifies that patching from release A to
// release B updates the artifact correctly.
func TestReleasePatch_ChangeRelease(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rp-change.md",
			content: makeArtifactWithRelease("Release Patch Change", "idea", "draft", "rp-change", "v-change-a", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{"name": "v-change-a", "status": "planned"})
	createRelease(t, env, map[string]any{"name": "v-change-b", "status": "planned"})

	const path = "lifecycle/ideas/rp-change.md"
	resp := patchRelease(env, path, strPtr("v-change-b"))
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	artifact, _ := data["artifact"].(map[string]any)
	fm, _ := artifact["frontmatter"].(map[string]any)
	if release, _ := fm["release"].(string); release != "v-change-b" {
		t.Errorf("response release after change: want %q, got %q", "v-change-b", release)
	}
}

// TestReleasePatch_ClearRelease verifies that PATCHing with null clears the
// release field from frontmatter.
func TestReleasePatch_ClearRelease(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rp-clear.md",
			content: makeArtifactWithRelease("Release Patch Clear", "idea", "draft", "rp-clear", "v-clear-1", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{"name": "v-clear-1", "status": "planned"})

	const path = "lifecycle/ideas/rp-clear.md"
	// Pass nil → JSON body is {"release": null} → clears the field.
	resp := patchRelease(env, path, nil)
	requireStatus(t, resp, 200)
	resp.Body.Close()

	// Read file from disk: release field must be absent.
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, path))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "release:") {
		t.Errorf("expected release field to be removed from disk, but found it in:\n%s", string(raw))
	}
}

// TestReleasePatch_InvalidReleaseName verifies that PATCH with a release name
// that does not exist in the project returns 422 with error code invalid_release.
func TestReleasePatch_InvalidReleaseName(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rp-invalid.md",
			content: makeArtifact("Release Patch Invalid", "idea", "draft", "rp-invalid", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const path = "lifecycle/ideas/rp-invalid.md"
	resp := patchRelease(env, path, strPtr("does-not-exist"))
	requireStatus(t, resp, 422)
	data := readJSON(t, resp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "invalid_release" {
		t.Errorf("expected error code %q, got %q", "invalid_release", code)
	}
}

// TestReleasePatch_ArtifactNotFound verifies that PATCH on a non-existent
// artifact path returns 404.
func TestReleasePatch_ArtifactNotFound(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("PATCH", "/api/p/testproject/artifacts/lifecycle/ideas/does-not-exist.md/release", map[string]any{
		"release": "v1.0",
	})
	requireStatus(t, resp, 404)
	resp.Body.Close()
}

// TestReleasePatch_InvalidJSONBody verifies that sending malformed JSON returns
// 400.
func TestReleasePatch_InvalidJSONBody(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rp-badjson.md",
			content: makeArtifact("Release Patch Bad JSON", "idea", "draft", "rp-badjson", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const path = "lifecycle/ideas/rp-badjson.md"
	const url = "/api/p/testproject/artifacts/" + path + "/release"

	req, err := http.NewRequest("PATCH", env.baseURL+url, bytes.NewReader([]byte(`{not valid json`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	for _, c := range env.cookies {
		req.AddCookie(c)
	}
	req.Header.Set("X-CSRF-Token", env.csrfToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, resp, 400)
	resp.Body.Close()
}

// TestReleasePatch_LockConflict verifies that PATCHing an artifact whose
// lineage is locked by a different user returns 423 Locked with error code
// "locked".
func TestReleasePatch_LockConflict(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rp-lock.md",
			content: makeArtifact("Release Patch Lock", "idea", "draft", "rp-lock", "", "Body."),
		},
	}
	// Use a custom config where qa@test.local also has product-owner so they
	// pass the RolesReleaseEditors check and reach the lock check.
	env := newTestEnvWithCfgYAML(t, seeds, cfgYAMLWithQAAsProductOwner)
	env.login("admin@test.local", "admin-pass-123")

	// Admin acquires the lineage lock.
	lockResp := env.doRequest("POST", "/api/p/testproject/locks", map[string]any{
		"lineage": "rp-lock",
		"kind":    "editor",
	})
	requireStatus(t, lockResp, 200)
	lockResp.Body.Close()

	// Create a release so the PATCH body passes release validation.
	createRelease(t, env, map[string]any{"name": "v-lock-test", "status": "planned"})

	// Switch to qa (product-owner in this config) and attempt PATCH.
	env.login("qa@test.local", "qa-pass-123")

	const path = "lifecycle/ideas/rp-lock.md"
	resp := patchRelease(env, path, strPtr("v-lock-test"))
	requireStatus(t, resp, 423)
	data := readJSON(t, resp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "locked" {
		t.Errorf("expected error code %q, got %q", "locked", code)
	}
	if _, ok := data["lock"]; !ok {
		t.Error("expected lock info in 423 response body")
	}
}

// TestReleasePatch_ReindexVerification verifies that after a successful PATCH
// the index reflects the updated release value (queries through the GET endpoint
// which reads from the index).
func TestReleasePatch_ReindexVerification(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rp-reindex.md",
			content: makeArtifact("Release Patch Reindex", "idea", "draft", "rp-reindex", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{"name": "v-reindex-1", "status": "planned"})

	const path = "lifecycle/ideas/rp-reindex.md"
	resp := patchRelease(env, path, strPtr("v-reindex-1"))
	requireStatus(t, resp, 200)
	resp.Body.Close()

	// Query the index via the list endpoint with a release filter.
	listResp := env.doRequest("GET", "/api/p/testproject/artifacts?release=v-reindex-1", nil)
	requireStatus(t, listResp, 200)
	listData := readJSON(t, listResp)

	items, _ := listData["items"].([]any)
	found := false
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if p, _ := item["path"].(string); p == path {
			found = true
			fm, _ := item["frontmatter"].(map[string]any)
			if release, _ := fm["release"].(string); release != "v-reindex-1" {
				t.Errorf("index release: want %q, got %q", "v-reindex-1", release)
			}
		}
	}
	if !found {
		t.Errorf("artifact %q not found in index after PATCH", path)
	}
}

// TestReleasePatch_WebSocketEvent verifies that a successful PATCH emits an
// artifact.indexed WebSocket event with action "updated" and the correct path.
func TestReleasePatch_WebSocketEvent(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rp-ws.md",
			content: makeArtifact("Release Patch WS", "idea", "draft", "rp-ws", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{"name": "v-ws-event", "status": "planned"})

	// Connect WebSocket before issuing the PATCH.
	wsURL := "ws://" + strings.TrimPrefix(env.baseURL, "http://") + "/api/p/testproject/ws"
	cookieHeader := buildCookieHeader(env.cookies)

	wsCtx, wsCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer wsCancel()

	conn, _, err := websocket.Dial(wsCtx, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{"Cookie": []string{cookieHeader}},
	})
	if err != nil {
		t.Fatalf("websocket dial failed: %v", err)
	}
	defer conn.CloseNow()

	// Buffer incoming events in a goroutine.
	eventCh := make(chan map[string]any, 20)
	go func() {
		for {
			var msg map[string]any
			if err := wsjson.Read(wsCtx, conn, &msg); err != nil {
				return
			}
			eventCh <- msg
		}
	}()

	// Small delay to ensure the WS subscription is registered before PATCH.
	time.Sleep(50 * time.Millisecond)

	const path = "lifecycle/ideas/rp-ws.md"
	resp := patchRelease(env, path, strPtr("v-ws-event"))
	requireStatus(t, resp, 200)
	resp.Body.Close()

	// Wait for the artifact.indexed event with the matching path.
	deadline := time.After(5 * time.Second)
	for {
		select {
		case event := <-eventCh:
			typ, _ := event["type"].(string)
			if typ == "artifact.indexed" {
				payload, _ := event["payload"].(map[string]any)
				if eventPath, _ := payload["path"].(string); eventPath == path {
					if action, _ := payload["action"].(string); action != "updated" {
						t.Errorf("expected action %q in event, got %q", "updated", action)
					}
					return // success
				}
			}
		case <-deadline:
			t.Fatal("timed out waiting for artifact.indexed WebSocket event after release PATCH")
		}
	}
}

// TestReleasePatch_FrontmatterPreservation verifies that PATCH only changes the
// release field and leaves all other frontmatter intact.
func TestReleasePatch_FrontmatterPreservation(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rp-preserve.md",
			content: makeArtifact("Release Patch Preserve", "idea", "draft", "rp-preserve", "", "Body.", "auth", "backend"),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	createRelease(t, env, map[string]any{"name": "v-preserve-1", "status": "planned"})

	const path = "lifecycle/ideas/rp-preserve.md"
	resp := patchRelease(env, path, strPtr("v-preserve-1"))
	requireStatus(t, resp, 200)
	resp.Body.Close()

	fm := artifactFrontmatterJSON(t, env, path)

	checks := map[string]string{
		"title":   "Release Patch Preserve",
		"type":    "idea",
		"status":  "draft",
		"lineage": "rp-preserve",
		"release": "v-preserve-1",
	}
	for field, want := range checks {
		if got, _ := fm[field].(string); got != want {
			t.Errorf("frontmatter %s: want %q, got %q", field, want, got)
		}
	}

	labels, _ := fm["labels"].([]any)
	if len(labels) != 2 {
		t.Errorf("expected 2 labels after release PATCH, got %d", len(labels))
	}
}
