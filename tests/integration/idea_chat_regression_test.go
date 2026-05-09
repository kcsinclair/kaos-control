// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// Milestone 6 – Coexistence and Regression Tests
//
// These tests verify that the conversational idea-capture feature does not break
// existing artifact creation, update, or WebSocket functionality.
// The manual-artifact and update tests do NOT require ANTHROPIC_API_KEY.
// The WebSocket event test DOES require ANTHROPIC_API_KEY.

// TestIdeaChatManualArtifactCreationUnchanged verifies that the existing
// POST /api/p/:project/artifacts endpoint still works after the idea-capture
// feature is in place.
func TestIdeaChatManualArtifactCreationUnchanged(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "ideas",
		"slug":  "regression-manual",
		"frontmatter": map[string]any{
			"title":   "Regression Manual Idea",
			"type":    "idea",
			"status":  "draft",
			"lineage": "regression-manual",
		},
		"body": "This idea was created via the direct artifact API.",
	})
	requireStatus(t, resp, 201)
	data := readJSON(t, resp)

	path, _ := data["path"].(string)
	if path == "" {
		t.Fatal("expected path in create response")
	}
	if path != "lifecycle/ideas/regression-manual.md" {
		t.Errorf("expected path lifecycle/ideas/regression-manual.md, got %s", path)
	}
}

// TestIdeaChatArtifactUpdateUnchanged verifies that PUT /api/p/:project/artifacts/*
// still works correctly alongside the new converse endpoint.
func TestIdeaChatArtifactUpdateUnchanged(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/regression-update.md",
			content: makeArtifact("Regression Update", "idea", "draft", "regression-update", "", "Original body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const artifactPath = "lifecycle/ideas/regression-update.md"
	resp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+artifactPath, map[string]any{
		"frontmatter": map[string]any{
			"title":   "Regression Update",
			"type":    "idea",
			"status":  "draft",
			"lineage": "regression-update",
		},
		"body": "Updated body after idea-capture feature was added.",
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	artifact, _ := data["artifact"].(map[string]any)
	if artifact == nil {
		t.Fatal("expected artifact in PUT response")
	}
	// Confirm the file on disk contains the updated body.
	content, err := os.ReadFile(filepath.Join(env.projectRoot, artifactPath))
	if err != nil {
		t.Fatalf("reading updated artifact: %v", err)
	}
	if !strings.Contains(string(content), "Updated body") {
		t.Error("updated artifact file does not contain 'Updated body'")
	}
}

// TestIdeaChatAgentEndpointAccessible verifies that GET /api/p/:project/agents
// is reachable and returns a valid response structure. This confirms the agent
// routing was not broken by the addition of the idea-capture handler.
func TestIdeaChatAgentEndpointAccessible(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/agents", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	if _, ok := data["agents"]; !ok {
		t.Error("expected 'agents' key in GET /agents response")
	}
}

// TestIdeaChatWebSocketEventsAfterAccept verifies that after a conversation
// leads to an accepted idea, a WebSocket client receives an "artifact.indexed"
// event containing the new artifact path.
// Requires ANTHROPIC_API_KEY.
func TestIdeaChatWebSocketEventsAfterAccept(t *testing.T) {
	skipIfNoAPIKey(t)
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Connect WebSocket before accepting so we don't miss the broadcast.
	wsURL := "ws" + strings.TrimPrefix(env.baseURL, "http") + "/api/p/testproject/ws"
	dialCtx, dialCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dialCancel()

	conn, _, err := websocket.Dial(dialCtx, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Cookie": []string{cookieHeader(env.cookies)},
		},
	})
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer conn.CloseNow()

	// Buffer WS messages in a goroutine.
	events := make(chan map[string]any, 32)
	wsCtx, wsCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer wsCancel()

	go func() {
		defer close(events)
		for {
			var msg map[string]any
			if err := wsjson.Read(wsCtx, conn, &msg); err != nil {
				return
			}
			events <- msg
		}
	}()

	// Drive conversation to proposal and accept.
	sessionID, _ := convergeToProposal(t, env, uniqueIdeaMessage("ws-events"))

	acceptResp := converseAPI(env, sessionID, "__accept__")
	requireStatus(t, acceptResp, 200)
	acceptData := readJSON(t, acceptResp)

	artifactPath, _ := acceptData["artifact_path"].(string)
	if artifactPath == "" {
		t.Fatal("missing artifact_path in accept response")
	}

	// Wait for artifact.indexed WS event.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case evt, ok := <-events:
			if !ok {
				t.Fatal("WebSocket events channel closed before receiving artifact.indexed")
			}
			evtType, _ := evt["type"].(string)
			if evtType != "artifact.indexed" {
				continue
			}
			// Payload may be nested inside "payload" key as a JSON-decoded object.
			payload := wsEventPath(evt)
			if payload == artifactPath {
				return // success
			}
		case <-time.After(deadline.Sub(time.Now())):
			t.Fatalf("timed out waiting for artifact.indexed WS event for %s", artifactPath)
		}
	}
	t.Fatalf("did not receive artifact.indexed event for %s within timeout", artifactPath)
}

// wsEventPath extracts the "path" from a WebSocket event message. The hub
// broadcasts JSON where the payload may be directly in the event map or nested
// under a "payload" key.
func wsEventPath(evt map[string]any) string {
	// Try top-level "path" field first.
	if p, ok := evt["path"].(string); ok {
		return p
	}
	// Try "payload" sub-object.
	if payload, ok := evt["payload"]; ok {
		switch v := payload.(type) {
		case map[string]any:
			if p, ok := v["path"].(string); ok {
				return p
			}
		case string:
			var m map[string]any
			if err := json.Unmarshal([]byte(v), &m); err == nil {
				if p, ok := m["path"].(string); ok {
					return p
				}
			}
		}
	}
	return ""
}
