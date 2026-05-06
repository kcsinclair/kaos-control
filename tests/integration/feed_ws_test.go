//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestFeedWebSocketNewEvent connects a hub client, triggers a status transition
// via the REST API, and verifies that a feed.new WebSocket event is delivered
// within 2 seconds. The payload must contain a non-zero id, the correct
// event_type (status_transition), a non-empty summary, and a non-zero timestamp.
//
// The test uses the hub channel pattern (env.proj.Hub.Register) rather than a
// real HTTP WebSocket connection, which is the same approach used by
// TestAnalystRunBroadcastsStatusChange in agent_ws_test.go.
func TestFeedWebSocketNewEvent(t *testing.T) {
	const artifactPath = "lifecycle/ideas/feed-ws-test.md"
	env := newTestEnv(t, []seedArtifact{{
		relPath: artifactPath,
		content: makeArtifact("Feed WS Test Idea", "idea", "draft", "feed-ws-test", "", "Body."),
	}})

	// Register the hub channel before triggering the action so no events are missed.
	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	env.login("admin@test.local", "admin-pass-123")

	// Trigger a status transition — this synchronously inserts a feed event and
	// calls Hub.Broadcast before the HTTP handler returns.
	resp := env.doRequest("POST",
		"/api/p/testproject/artifacts/"+artifactPath+"/transition",
		map[string]string{"to": "clarifying"},
	)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Drain hub messages until we find a feed.new event or timeout.
	var feedPayload map[string]any
	timeout := time.After(2 * time.Second)

COLLECT:
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
				break COLLECT
			}
		case <-timeout:
			break COLLECT
		}
	}

	if feedPayload == nil {
		t.Fatal("did not receive a feed.new WebSocket event within 2 seconds")
	}

	// id must be non-zero.
	if id, _ := feedPayload["id"].(float64); id == 0 {
		t.Errorf("feed.new payload: missing or zero id (got %v)", feedPayload["id"])
	}

	// event_type must be status_transition.
	if et, _ := feedPayload["event_type"].(string); et != "status_transition" {
		t.Errorf("feed.new event_type: expected status_transition, got %q", et)
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
