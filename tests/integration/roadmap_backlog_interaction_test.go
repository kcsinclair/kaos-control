// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// ── Milestone 5: Backlog Interaction and Reactive Update Tests ────────────────
//
// These tests verify that assigning or clearing a release field on an artifact
// via PUT /artifacts/* immediately changes its membership in the backlog query,
// and that a WebSocket "artifact.indexed" event is broadcast so the client can
// reactively update the Backlog panel without a page reload.

// TestBacklogInteraction_AssignReleaseRemovesFromBacklog verifies that after
// assigning a release to an unassigned artifact (FR4.2):
//   - The artifact no longer appears in GET /artifacts?release=__unassigned__.
//   - The artifact appears in GET /artifacts?release=<name>.
func TestBacklogInteraction_AssignReleaseRemovesFromBacklog(t *testing.T) {
	const artifactPath = "lifecycle/ideas/bi-assign-1.md"

	seeds := []seedArtifact{
		{
			relPath: artifactPath,
			content: makeArtifact("BI Assign 1", "idea", "draft", "bi-assign-1", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Confirm the artifact is initially in the backlog.
	resp := env.doRequest("GET", "/api/p/testproject/artifacts?release=__unassigned__", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)
	items, _ := body["items"].([]any)
	if !containsPath(items, artifactPath) {
		t.Fatalf("artifact %q should be in backlog before assignment", artifactPath)
	}

	// Assign a release.
	resp = env.doRequest("PUT", "/api/p/testproject/artifacts/"+artifactPath, map[string]any{
		"frontmatter": map[string]any{
			"title":   "BI Assign 1",
			"type":    "idea",
			"status":  "draft",
			"lineage": "bi-assign-1",
			"release": "v-bi-rel",
		},
		"body": "Body.",
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Artifact must no longer appear in the backlog.
	resp = env.doRequest("GET", "/api/p/testproject/artifacts?release=__unassigned__", nil)
	requireStatus(t, resp, http.StatusOK)
	body = readJSON(t, resp)
	items, _ = body["items"].([]any)
	if containsPath(items, artifactPath) {
		t.Errorf("artifact %q should NOT be in backlog after release assignment", artifactPath)
	}

	// Artifact must appear in the release-specific filter.
	resp = env.doRequest("GET", "/api/p/testproject/artifacts?release=v-bi-rel", nil)
	requireStatus(t, resp, http.StatusOK)
	body = readJSON(t, resp)
	items, _ = body["items"].([]any)
	if !containsPath(items, artifactPath) {
		t.Errorf("artifact %q should appear in release filter after assignment", artifactPath)
	}
}

// TestBacklogInteraction_ClearReleaseAddsToBacklog verifies that after clearing
// the release field from a previously assigned artifact (FR4.3):
//   - The artifact appears in GET /artifacts?release=__unassigned__.
//   - The artifact no longer appears in the old release filter.
func TestBacklogInteraction_ClearReleaseAddsToBacklog(t *testing.T) {
	const artifactPath = "lifecycle/ideas/bi-clear-1.md"

	seeds := []seedArtifact{
		{
			relPath: artifactPath,
			content: makeArtifactWithRelease("BI Clear 1", "idea", "draft", "bi-clear-1", "v-bi-clear-rel", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Confirm the artifact is NOT in the backlog initially (has release assignment).
	resp := env.doRequest("GET", "/api/p/testproject/artifacts?release=__unassigned__", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)
	items, _ := body["items"].([]any)
	if containsPath(items, artifactPath) {
		t.Fatalf("artifact %q should NOT be in backlog before clearing release", artifactPath)
	}

	// Clear the release field by omitting it from frontmatter.
	resp = env.doRequest("PUT", "/api/p/testproject/artifacts/"+artifactPath, map[string]any{
		"frontmatter": map[string]any{
			"title":   "BI Clear 1",
			"type":    "idea",
			"status":  "draft",
			"lineage": "bi-clear-1",
		},
		"body": "Body.",
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Artifact must now appear in the backlog.
	resp = env.doRequest("GET", "/api/p/testproject/artifacts?release=__unassigned__", nil)
	requireStatus(t, resp, http.StatusOK)
	body = readJSON(t, resp)
	items, _ = body["items"].([]any)
	if !containsPath(items, artifactPath) {
		t.Errorf("artifact %q should be in backlog after clearing release", artifactPath)
	}

	// Artifact must no longer appear under the old release filter.
	resp = env.doRequest("GET", "/api/p/testproject/artifacts?release=v-bi-clear-rel", nil)
	requireStatus(t, resp, http.StatusOK)
	body = readJSON(t, resp)
	items, _ = body["items"].([]any)
	if containsPath(items, artifactPath) {
		t.Errorf("artifact %q should NOT appear under old release filter after clearing", artifactPath)
	}
}

// TestBacklogInteraction_ArtifactIndexedWebSocketFires verifies that updating
// an artifact (assigning a release) broadcasts an "artifact.indexed" WebSocket
// event so the client can reactively refresh the Backlog panel (FR4.2, FR4.3).
func TestBacklogInteraction_ArtifactIndexedWebSocketFires(t *testing.T) {
	const artifactPath = "lifecycle/ideas/bi-ws-1.md"

	seeds := []seedArtifact{
		{
			relPath: artifactPath,
			content: makeArtifact("BI WS 1", "idea", "draft", "bi-ws-1", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	// Update the artifact to assign a release.
	resp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+artifactPath, map[string]any{
		"frontmatter": map[string]any{
			"title":   "BI WS 1",
			"type":    "idea",
			"status":  "draft",
			"lineage": "bi-ws-1",
			"release": "v-bi-ws-rel",
		},
		"body": "Body.",
	})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	payload := collectArtifactIndexedEvent(ch, artifactPath, 3*time.Second)
	if payload == nil {
		t.Fatal("did not receive artifact.indexed event within 3 seconds after assigning release")
	}
}

// TestBacklogInteraction_ClickNavigatesRoute verifies that the artifact path
// returned in the backlog list is a valid routable path (FR4.1).
// The frontend uses the path field to construct /p/:project/artifacts/:path.
// We verify the path field is non-empty and matches the expected format.
func TestBacklogInteraction_ClickNavigatesRoute(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/bi-nav-1.md",
			content: makeArtifact("BI Nav 1", "idea", "draft", "bi-nav-1", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?release=__unassigned__", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)
	items, _ := body["items"].([]any)

	if len(items) == 0 {
		t.Fatal("expected at least 1 backlog item for navigation route test")
	}

	for _, raw := range items {
		item, _ := raw.(map[string]any)
		path, _ := item["path"].(string)
		if path == "" {
			t.Errorf("backlog item missing path field (required for navigation): %v", item)
			continue
		}
		// The frontend constructs /p/:project/artifacts/:path using this field.
		// Verify it resolves via the artifact GET endpoint.
		artifactResp := env.doRequest("GET", "/api/p/testproject/artifacts/"+path, nil)
		requireStatus(t, artifactResp, http.StatusOK)
		artifactResp.Body.Close()
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// containsPath returns true if the items slice contains an artifact with the
// given path field.
func containsPath(items []any, path string) bool {
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if p, _ := item["path"].(string); p == path {
			return true
		}
	}
	return false
}

// collectArtifactIndexedEvent drains the hub channel until an "artifact.indexed"
// event for the given artifact path arrives, or the timeout elapses.
func collectArtifactIndexedEvent(ch <-chan []byte, artifactPath string, timeout time.Duration) map[string]any {
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
			if evt.Type != "artifact.indexed" {
				continue
			}
			// Payload may contain "path" or "artifact.path".
			if p, _ := evt.Payload["path"].(string); p == artifactPath {
				return evt.Payload
			}
			if artifact, ok := evt.Payload["artifact"].(map[string]any); ok {
				if p, _ := artifact["path"].(string); p == artifactPath {
					return evt.Payload
				}
			}
		case <-deadline:
			return nil
		}
	}
}
