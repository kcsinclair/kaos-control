// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// priority_inline_edit_test.go covers the gaps in priority test coverage
// identified by lifecycle/test-plans/artefact-priority-inline-edit-5-test.md
// that were not already addressed in priority_patch_test.go,
// priority_dropdown_test.go, or priority_roundtrip_test.go.
//
// Milestone map:
//   M1 — TestPriorityPatchIdempotent, TestPriorityPatchDiskConfirmation,
//         TestPriorityPatchWebSocketActionField
//   M2 — TestPriorityAbsentOmittedFromResponse
//   M5 — TestPriorityExternalFileWriteBroadcastsUpdate
//   M6 — TestPriorityPatchLockedByOtherUser, TestPriorityPatchWorksAfterLockRelease

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ── Milestone 1 ───────────────────────────────────────────────────────────────

// TestPriorityPatchIdempotent verifies that PATCHing the same priority value
// that is already set returns 200 with no error (idempotent write).
func TestPriorityPatchIdempotent(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/prio-idempotent.md",
			content: makeArtifactWithPriority("Priority Idempotent", "idea", "draft", "prio-idempotent", "medium", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const path = "lifecycle/ideas/prio-idempotent.md"

	// PATCH with the same value that is already set ("medium").
	resp := env.doRequest("PATCH", "/api/p/testproject/artifacts/"+path+"/priority", map[string]any{
		"priority": "medium",
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	artifact, _ := data["artifact"].(map[string]any)
	if artifact == nil {
		t.Fatal("expected artifact in response body")
	}
	fm, _ := artifact["frontmatter"].(map[string]any)
	if got, _ := fm["priority"].(string); got != "medium" {
		t.Errorf("idempotent PATCH priority: want %q, got %q", "medium", got)
	}

	// A second idempotent PATCH should also succeed.
	resp2 := env.doRequest("PATCH", "/api/p/testproject/artifacts/"+path+"/priority", map[string]any{
		"priority": "medium",
	})
	requireStatus(t, resp2, 200)
	resp2.Body.Close()
}

// TestPriorityPatchDiskConfirmation verifies that after a PATCH, the artifact
// file on disk contains the updated priority field in its YAML frontmatter.
// (Milestone 1 acceptance criterion: "reading the artifact file from disk
// confirms the YAML frontmatter contains the updated priority field".)
func TestPriorityPatchDiskConfirmation(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/prio-disk.md",
			content: makeArtifactWithPriority("Priority Disk Confirm", "idea", "draft", "prio-disk", "normal", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const path = "lifecycle/ideas/prio-disk.md"

	resp := env.doRequest("PATCH", "/api/p/testproject/artifacts/"+path+"/priority", map[string]any{
		"priority": "high",
	})
	requireStatus(t, resp, 200)
	resp.Body.Close()

	raw, err := os.ReadFile(filepath.Join(env.projectRoot, path))
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	if !strings.Contains(content, "priority: high") {
		t.Errorf("disk file does not contain 'priority: high' after PATCH:\n%s", content)
	}
}

// TestPriorityPatchWebSocketActionField verifies that the artifact.indexed
// WebSocket event emitted after a successful PATCH contains an "action"
// field set to "updated".
func TestPriorityPatchWebSocketActionField(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/prio-ws-action.md",
			content: makeArtifact("Priority WS Action", "idea", "draft", "prio-ws-action", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Register a hub channel to receive broadcast events directly.
	ch := make(chan []byte, 32)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	const path = "lifecycle/ideas/prio-ws-action.md"

	resp := env.doRequest("PATCH", "/api/p/testproject/artifacts/"+path+"/priority", map[string]any{
		"priority": "low",
	})
	requireStatus(t, resp, 200)
	resp.Body.Close()

	type wsEvent struct {
		Type    string         `json:"type"`
		Payload map[string]any `json:"payload"`
	}

	deadline := time.After(5 * time.Second)
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
			evtPath, _ := evt.Payload["path"].(string)
			if evtPath != path {
				continue
			}
			action, _ := evt.Payload["action"].(string)
			if action != "updated" {
				t.Errorf("artifact.indexed action: want %q, got %q", "updated", action)
			}
			return // success
		case <-deadline:
			t.Fatal("timed out waiting for artifact.indexed event after PATCH")
		}
	}
}

// ── Milestone 2 (backend-observable) ─────────────────────────────────────────

// TestPriorityAbsentOmittedFromResponse verifies that when an artifact has no
// priority field in its frontmatter, the GET endpoint returns an empty or absent
// priority value — the UI layer is responsible for displaying this as "normal".
func TestPriorityAbsentOmittedFromResponse(t *testing.T) {
	seeds := []seedArtifact{
		{
			// makeArtifact writes no priority field.
			relPath: "lifecycle/ideas/prio-absent.md",
			content: makeArtifact("Priority Absent", "idea", "draft", "prio-absent", "", "No priority set."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const path = "lifecycle/ideas/prio-absent.md"

	fm := artifactFrontmatterJSON(t, env, path)
	// priority should be absent entirely or an empty string — never a
	// non-empty default injected by the backend.
	if got, _ := fm["priority"].(string); got != "" {
		t.Errorf("absent priority: want empty string or absent, got %q", got)
	}

	// Disk file must not contain a priority key.
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, path))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "priority:") {
		t.Errorf("disk file unexpectedly contains 'priority:' when none was set:\n%s", raw)
	}
}

// ── Milestone 5 — WebSocket sync ─────────────────────────────────────────────

// TestPriorityExternalFileWriteBroadcastsUpdate verifies that modifying the
// priority field directly on disk (simulating an external editor or agent run)
// causes the watcher to re-index the artifact and broadcast a file.changed
// event.  The API must then return the updated priority value.
//
// Note: the watcher broadcasts "file.changed" (not "artifact.indexed") for
// external disk writes.  "artifact.indexed" is only emitted by the HTTP write
// handlers (PUT / PATCH).  The frontend uses "file.changed" to know that a
// remote refresh may be needed.
func TestPriorityExternalFileWriteBroadcastsUpdate(t *testing.T) {
	const relPath = "lifecycle/ideas/prio-ext-write.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			content: makeArtifactWithPriority("Priority External Write", "idea", "draft", "prio-ext-write", "normal", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Register hub channel before the disk write so no events are missed.
	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	// Write updated content directly to disk (simulating external edit).
	updated := makeArtifactWithPriority("Priority External Write", "idea", "draft", "prio-ext-write", "high", "Body.")
	absPath := filepath.Join(env.projectRoot, relPath)
	if err := os.WriteFile(absPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for the watcher to emit a file.changed event for this path.
	// The watcher uses a 150 ms debounce before re-indexing.
	type wsEvent struct {
		Type    string         `json:"type"`
		Payload map[string]any `json:"payload"`
	}

	deadline := time.After(5 * time.Second)
	gotEvent := false
	for !gotEvent {
		select {
		case raw := <-ch:
			var evt wsEvent
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Type != "file.changed" {
				continue
			}
			evtPath, _ := evt.Payload["path"].(string)
			if evtPath == relPath {
				gotEvent = true
			}
		case <-deadline:
			t.Fatal("timed out waiting for file.changed event after external file write")
		}
	}

	// API must now return the updated priority (watcher re-indexed the file).
	fm := artifactFrontmatterJSON(t, env, relPath)
	if got, _ := fm["priority"].(string); got != "high" {
		t.Errorf("GET priority after external write: want %q, got %q", "high", got)
	}
}

// TestPriorityExternalWriteClosedDropdownUpdatesAPI verifies the backend half
// of the "dropdown closed when external update arrives" scenario: the artifact
// index is updated by the watcher so the next GET returns the new priority.
func TestPriorityExternalWriteClosedDropdownUpdatesAPI(t *testing.T) {
	const relPath = "lifecycle/ideas/prio-ext-closed.md"
	seeds := []seedArtifact{
		{
			relPath: relPath,
			content: makeArtifactWithPriority("Priority Ext Closed", "idea", "draft", "prio-ext-closed", "low", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Verify initial priority via API.
	fm := artifactFrontmatterJSON(t, env, relPath)
	if got, _ := fm["priority"].(string); got != "low" {
		t.Fatalf("initial priority: want %q, got %q", "low", got)
	}

	// Simulate an external write (e.g. another session via the PATCH endpoint or
	// a direct agent edit on disk).
	updated := makeArtifactWithPriority("Priority Ext Closed", "idea", "draft", "prio-ext-closed", "high", "Body.")
	absPath := filepath.Join(env.projectRoot, relPath)
	if err := os.WriteFile(absPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}

	// Poll until the index reflects the change (watcher debounce ~150 ms).
	deadline := time.Now().Add(5 * time.Second)
	var finalPriority string
	for time.Now().Before(deadline) {
		fm = artifactFrontmatterJSON(t, env, relPath)
		finalPriority, _ = fm["priority"].(string)
		if finalPriority == "high" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if finalPriority != "high" {
		t.Errorf("GET priority after external disk write: want %q, got %q", "high", finalPriority)
	}
}

// ── Milestone 6 — Read-only mode (lock-based) ────────────────────────────────

// TestPriorityPatchLockedByOtherUser verifies that when another user holds
// the lineage lock, PATCH /priority returns HTTP 423 Locked with a "locked"
// error code.
func TestPriorityPatchLockedByOtherUser(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/prio-locked.md",
			content: makeArtifactWithPriority("Priority Locked", "idea", "draft", "prio-locked", "normal", "Body."),
		},
	}
	env := newTestEnv(t, seeds)

	// dev acquires the lineage lock.
	env.login("dev@test.local", "dev-pass-123")
	lockResp := env.doRequest("POST", "/api/p/testproject/locks", map[string]any{
		"lineage": "prio-locked",
		"kind":    "editor",
	})
	requireStatus(t, lockResp, 200)
	lockResp.Body.Close()

	devCookies := env.cookies
	devCSRF := env.csrfToken

	// admin tries to PATCH priority while dev holds the lock → 423.
	env.login("admin@test.local", "admin-pass-123")
	const path = "lifecycle/ideas/prio-locked.md"
	resp := env.doRequest("PATCH", "/api/p/testproject/artifacts/"+path+"/priority", map[string]any{
		"priority": "high",
	})
	requireStatus(t, resp, 423)
	data := readJSON(t, resp)

	errData, _ := data["error"].(map[string]any)
	if code, _ := errData["code"].(string); code != "locked" {
		t.Errorf("error code: want %q, got %q", "locked", code)
	}
	if _, ok := data["lock"]; !ok {
		t.Error("response should contain a 'lock' field describing who holds the lock")
	}

	// Release the lock (switch back to dev).
	env.cookies = devCookies
	env.csrfToken = devCSRF
	releaseResp := env.doRequest("DELETE", "/api/p/testproject/locks/prio-locked", nil)
	requireStatus(t, releaseResp, 204)
	releaseResp.Body.Close()
}

// TestPriorityPatchWorksAfterLockRelease verifies that once the lineage lock
// is released, PATCH /priority succeeds again (HTTP 200).
func TestPriorityPatchWorksAfterLockRelease(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/prio-lock-release.md",
			content: makeArtifactWithPriority("Priority Lock Release", "idea", "draft", "prio-lock-release", "normal", "Body."),
		},
	}
	env := newTestEnv(t, seeds)

	// dev acquires then releases the lock.
	env.login("dev@test.local", "dev-pass-123")
	lockResp := env.doRequest("POST", "/api/p/testproject/locks", map[string]any{
		"lineage": "prio-lock-release",
		"kind":    "editor",
	})
	requireStatus(t, lockResp, 200)
	lockResp.Body.Close()

	releaseResp := env.doRequest("DELETE", "/api/p/testproject/locks/prio-lock-release", nil)
	requireStatus(t, releaseResp, 204)
	releaseResp.Body.Close()

	// admin can now PATCH successfully.
	env.login("admin@test.local", "admin-pass-123")
	const path = "lifecycle/ideas/prio-lock-release.md"
	resp := env.doRequest("PATCH", "/api/p/testproject/artifacts/"+path+"/priority", map[string]any{
		"priority": "low",
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	artifact, _ := data["artifact"].(map[string]any)
	if artifact == nil {
		t.Fatal("expected artifact in response after lock-release PATCH")
	}
	fm, _ := artifact["frontmatter"].(map[string]any)
	if got, _ := fm["priority"].(string); got != "low" {
		t.Errorf("priority after lock-release PATCH: want %q, got %q", "low", got)
	}
}

// TestPriorityGetReturnsLockHolder verifies that when an artifact's lineage is
// locked, the GET artifact endpoint still returns the artifact (the lock does
// not prevent reads).  The UI uses the separate GET /locks endpoint to
// determine interactivity, but this test confirms the artifact itself is always
// readable.
func TestPriorityGetReturnsLockHolder(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/prio-get-locked.md",
			content: makeArtifactWithPriority("Priority Get Locked", "idea", "draft", "prio-get-locked", "medium", "Body."),
		},
	}
	env := newTestEnv(t, seeds)

	// Acquire lock as dev.
	env.login("dev@test.local", "dev-pass-123")
	lockResp := env.doRequest("POST", "/api/p/testproject/locks", map[string]any{
		"lineage": "prio-get-locked",
		"kind":    "editor",
	})
	requireStatus(t, lockResp, 200)
	lockResp.Body.Close()

	// GET the artifact as admin — must succeed and return the artifact.
	env.login("admin@test.local", "admin-pass-123")
	const path = "lifecycle/ideas/prio-get-locked.md"
	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+path, nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	artifact, _ := data["artifact"].(map[string]any)
	if artifact == nil {
		t.Fatal("GET artifact should succeed even when lineage is locked")
	}
	fm, _ := artifact["frontmatter"].(map[string]any)
	if got, _ := fm["priority"].(string); got != "medium" {
		t.Errorf("GET priority while locked: want %q, got %q", "medium", got)
	}

	// GET /locks confirms the lineage is locked.
	locksResp := env.doRequest("GET", "/api/p/testproject/locks", nil)
	requireStatus(t, locksResp, 200)
	locksData := readJSON(t, locksResp)
	locks, _ := locksData["locks"].([]any)
	if len(locks) == 0 {
		t.Error("expected at least one lock in GET /locks while dev holds prio-get-locked")
	}

	// Release.
	env.login("dev@test.local", "dev-pass-123")
	env.doRequest("DELETE", "/api/p/testproject/locks/prio-get-locked", nil).Body.Close()
}
