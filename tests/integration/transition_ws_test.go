// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestTransitionWebSocketArtifactIndexed connects a hub channel client before
// triggering a status transition, then verifies that an "artifact.indexed" event
// is delivered with the expected payload shape: path, action == "transitioned",
// from, and to fields.
//
// Uses the hub channel pattern (env.proj.Hub.Register) rather than a real HTTP
// WebSocket connection — the same approach used by TestFeedWebSocketNewEvent in
// feed_ws_test.go.
func TestTransitionWebSocketArtifactIndexed(t *testing.T) {
	const artifactPath = "lifecycle/requirements/ws-artifact-indexed.md"
	env := newTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("WS Artifact Indexed", "ticket", "draft", "ws-artifact-indexed", "", "Body."),
	}})

	// Register the hub channel BEFORE triggering the transition so no events are missed.
	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("admin@test.local", "admin-pass-123")
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]string{"to": "clarifying"})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Drain hub messages until we find an artifact.indexed event or timeout.
	var indexedPayload map[string]any
	timeout := time.After(2 * time.Second)

COLLECT_INDEXED:
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
			if evt.Type == "artifact.indexed" {
				indexedPayload = evt.Payload
				break COLLECT_INDEXED
			}
		case <-timeout:
			break COLLECT_INDEXED
		}
	}

	if indexedPayload == nil {
		t.Fatal("did not receive an artifact.indexed WebSocket event within 2 seconds")
	}

	// payload.path must equal the transitioned artifact path.
	if path, _ := indexedPayload["path"].(string); path != artifactPath {
		t.Errorf("artifact.indexed payload.path: expected %q, got %q", artifactPath, path)
	}

	// payload.action must be "transitioned".
	if action, _ := indexedPayload["action"].(string); action != "transitioned" {
		t.Errorf("artifact.indexed payload.action: expected \"transitioned\", got %q", action)
	}

	// payload.from must be the previous status.
	if from, _ := indexedPayload["from"].(string); from != "draft" {
		t.Errorf("artifact.indexed payload.from: expected \"draft\", got %q", from)
	}

	// payload.to must be the new status.
	if to, _ := indexedPayload["to"].(string); to != "clarifying" {
		t.Errorf("artifact.indexed payload.to: expected \"clarifying\", got %q", to)
	}
}

// TestTransitionWebSocketFeedNew verifies that a successful status transition
// also delivers a "feed.new" WebSocket event whose payload contains
// event_type == "status_transition", a non-zero id, a non-empty summary, and a
// non-zero timestamp.
//
// WebSocket connections are cleaned up via defer Unregister — no leaks.
func TestTransitionWebSocketFeedNew(t *testing.T) {
	const artifactPath = "lifecycle/requirements/ws-feed-new.md"
	env := newTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("WS Feed New", "ticket", "draft", "ws-feed-new", "", "Body."),
	}})

	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("admin@test.local", "admin-pass-123")
	resp := env.doRequest("POST", "/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]string{"to": "clarifying"})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	var feedPayload map[string]any
	timeout := time.After(2 * time.Second)

COLLECT_FEED:
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
			if evt.Type == "feed.new" {
				feedPayload = evt.Payload
				break COLLECT_FEED
			}
		case <-timeout:
			break COLLECT_FEED
		}
	}

	if feedPayload == nil {
		t.Fatal("did not receive a feed.new WebSocket event within 2 seconds")
	}

	// event_type must be "status_transition".
	if et, _ := feedPayload["event_type"].(string); et != "status_transition" {
		t.Errorf("feed.new event_type: expected \"status_transition\", got %q", et)
	}

	// id must be non-zero.
	if id, _ := feedPayload["id"].(float64); id == 0 {
		t.Errorf("feed.new payload: missing or zero id (got %v)", feedPayload["id"])
	}

	// summary must be non-empty.
	if summary, _ := feedPayload["summary"].(string); summary == "" {
		t.Error("feed.new payload: summary is empty")
	}

	// timestamp must be non-zero.
	if ts, _ := feedPayload["timestamp"].(float64); ts == 0 {
		t.Errorf("feed.new payload: missing or zero timestamp (got %v)", feedPayload["timestamp"])
	}
}
