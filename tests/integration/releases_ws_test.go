// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// collectReleaseEvent drains the hub channel until an event of the given type
// arrives or the timeout elapses.  Returns nil on timeout.
func collectReleaseEvent(ch <-chan []byte, eventType string, timeout time.Duration) map[string]any {
	deadline := time.After(timeout)
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
			if evt.Type == eventType {
				return evt.Payload
			}
		case <-deadline:
			return nil
		}
	}
}

// ── Milestone 6: WebSocket event tests ───────────────────────────────────────

// TestReleaseWebSocket_Created verifies that creating a release broadcasts a
// "release.created" hub event whose payload contains the release data.
func TestReleaseWebSocket_Created(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	resp := env.doRequest("POST", "/api/p/testproject/releases", map[string]any{
		"name":   "v-ws-created",
		"status": "planned",
	})
	requireStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	payload := collectReleaseEvent(ch, "release.created", 2*time.Second)
	if payload == nil {
		t.Fatal("did not receive release.created event within 2 seconds")
	}

	rel, _ := payload["release"].(map[string]any)
	if name, _ := rel["name"].(string); name != "v-ws-created" {
		t.Errorf("release.created payload.release.name: want %q, got %q", "v-ws-created", name)
	}
	if id, _ := rel["id"].(float64); id == 0 {
		t.Error("release.created payload.release.id must be non-zero")
	}
}

// TestReleaseWebSocket_Updated verifies that updating a release broadcasts a
// "release.updated" hub event with the updated release data.
func TestReleaseWebSocket_Updated(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Create the release before registering the hub so the create event doesn't
	// interfere with draining.
	data := createRelease(t, env, map[string]any{"name": "v-ws-upd", "status": "planned"})
	id := releaseID(t, data)

	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	resp := env.doRequest("PUT", releasePath(id), map[string]any{
		"name":   "v-ws-upd",
		"status": "active",
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	payload := collectReleaseEvent(ch, "release.updated", 2*time.Second)
	if payload == nil {
		t.Fatal("did not receive release.updated event within 2 seconds")
	}

	rel, _ := payload["release"].(map[string]any)
	if status, _ := rel["status"].(string); status != "active" {
		t.Errorf("release.updated payload.release.status: want %q, got %q", "active", status)
	}
}

// TestReleaseWebSocket_Deleted verifies that deleting a release broadcasts a
// "release.deleted" hub event containing the release ID.
func TestReleaseWebSocket_Deleted(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{"name": "v-ws-del", "status": "planned"})
	wantID := releaseID(t, data)

	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	resp := env.doRequest("DELETE", releasePath(wantID), nil)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	payload := collectReleaseEvent(ch, "release.deleted", 2*time.Second)
	if payload == nil {
		t.Fatal("did not receive release.deleted event within 2 seconds")
	}

	gotID, _ := payload["id"].(float64)
	if int64(gotID) != wantID {
		t.Errorf("release.deleted payload.id: want %d, got %d", wantID, int64(gotID))
	}
}

// TestReleaseWebSocket_RenamePropagate verifies that renaming a release that
// has assigned artifacts broadcasts a "release.updated" event followed by at
// least one "artifact.indexed" event for the updated artifact.
func TestReleaseWebSocket_RenamePropagate(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rws-rename-idea.md",
			content: makeArtifactWithRelease("RWS Rename Idea", "idea", "draft", "rws-rename-idea", "v-ws-rename-old", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	data := createRelease(t, env, map[string]any{"name": "v-ws-rename-old", "status": "planned"})
	id := releaseID(t, data)

	ch := make(chan []byte, 128)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	resp := env.doRequest("PUT", releasePath(id), map[string]any{
		"name":   "v-ws-rename-new",
		"status": "planned",
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Collect events for up to 3 seconds, looking for both types.
	gotUpdated := false
	gotIndexed := false
	deadline := time.After(3 * time.Second)

DRAIN:
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
			switch evt.Type {
			case "release.updated":
				gotUpdated = true
			case "artifact.indexed":
				gotIndexed = true
			}
			if gotUpdated && gotIndexed {
				break DRAIN
			}
		case <-deadline:
			break DRAIN
		}
	}

	if !gotUpdated {
		t.Error("did not receive release.updated event after rename propagation")
	}
	if !gotIndexed {
		t.Error("did not receive artifact.indexed event after rename propagation")
	}
}
