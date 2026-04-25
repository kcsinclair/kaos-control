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

// TestPriorityPatchHappyPath verifies PATCH priority from "normal" to "high"
// returns the updated artifact and GET subsequently reflects the new value.
func TestPriorityPatchHappyPath(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/prio-happy.md",
			content: makeArtifactWithPriority("Priority Happy Path", "idea", "draft", "prio-happy", "normal", "Body text."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const path = "lifecycle/ideas/prio-happy.md"

	resp := env.doRequest("PATCH", "/api/p/testproject/artifacts/"+path+"/priority", map[string]any{
		"priority": "high",
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	artifact, _ := data["artifact"].(map[string]any)
	if artifact == nil {
		t.Fatal("expected artifact in response")
	}
	fm, _ := artifact["frontmatter"].(map[string]any)
	if priority, _ := fm["priority"].(string); priority != "high" {
		t.Errorf("response priority: want %q, got %q", "high", priority)
	}

	// GET should now return updated priority.
	resp2 := env.doRequest("GET", "/api/p/testproject/artifacts/"+path, nil)
	requireStatus(t, resp2, 200)
	data2 := readJSON(t, resp2)
	artifact2, _ := data2["artifact"].(map[string]any)
	fm2, _ := artifact2["frontmatter"].(map[string]any)
	if priority, _ := fm2["priority"].(string); priority != "high" {
		t.Errorf("GET priority after PATCH: want %q, got %q", "high", priority)
	}
}

// TestPriorityPatchUnset verifies that PATCHing with priority="" clears the
// priority field from frontmatter on disk.
func TestPriorityPatchUnset(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/prio-unset.md",
			content: makeArtifactWithPriority("Priority Unset", "idea", "draft", "prio-unset", "medium", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const path = "lifecycle/ideas/prio-unset.md"

	resp := env.doRequest("PATCH", "/api/p/testproject/artifacts/"+path+"/priority", map[string]any{
		"priority": "",
	})
	requireStatus(t, resp, 200)
	resp.Body.Close()

	// Read file from disk and confirm priority field is absent.
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, path))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "priority:") {
		t.Errorf("expected priority field to be removed from disk, but found it in:\n%s", string(raw))
	}
}

// TestPriorityPatchInvalidValue verifies that PATCH with an unrecognised
// priority value returns 400 bad_request.
func TestPriorityPatchInvalidValue(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/prio-invalid.md",
			content: makeArtifact("Priority Invalid", "idea", "draft", "prio-invalid", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("PATCH", "/api/p/testproject/artifacts/lifecycle/ideas/prio-invalid.md/priority", map[string]any{
		"priority": "critical",
	})
	requireStatus(t, resp, 400)
	data := readJSON(t, resp)
	errData, _ := data["error"].(map[string]any)
	if code, _ := errData["code"].(string); code != "bad_request" {
		t.Errorf("expected error code 'bad_request', got %q", code)
	}
}

// TestPriorityPatchNonExistent verifies that PATCHing a path that doesn't
// exist returns 404.
func TestPriorityPatchNonExistent(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("PATCH", "/api/p/testproject/artifacts/lifecycle/ideas/does-not-exist.md/priority", map[string]any{
		"priority": "high",
	})
	requireStatus(t, resp, 404)
	resp.Body.Close()
}

// TestPriorityPatchFrontmatterPreservation verifies that a PATCH only changes
// priority and leaves all other frontmatter fields intact.
func TestPriorityPatchFrontmatterPreservation(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/prio-preserve.md",
			content: makeArtifact("Preserve Frontmatter", "idea", "draft", "prio-preserve", "", "Body.", "auth", "backend"),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const path = "lifecycle/ideas/prio-preserve.md"

	resp := env.doRequest("PATCH", "/api/p/testproject/artifacts/"+path+"/priority", map[string]any{
		"priority": "low",
	})
	requireStatus(t, resp, 200)
	resp.Body.Close()

	// Read updated artifact.
	resp2 := env.doRequest("GET", "/api/p/testproject/artifacts/"+path, nil)
	requireStatus(t, resp2, 200)
	data := readJSON(t, resp2)

	artifact, _ := data["artifact"].(map[string]any)
	fm, _ := artifact["frontmatter"].(map[string]any)

	checks := map[string]string{
		"title":   "Preserve Frontmatter",
		"type":    "idea",
		"status":  "draft",
		"lineage": "prio-preserve",
	}
	for field, want := range checks {
		if got, _ := fm[field].(string); got != want {
			t.Errorf("frontmatter %s: want %q, got %q", field, want, got)
		}
	}

	// Labels should still be present.
	labels, _ := fm["labels"].([]any)
	if len(labels) != 2 {
		t.Errorf("expected 2 labels after PATCH, got %d", len(labels))
	}
}

// TestPriorityPatchBodyPreservation verifies the markdown body is byte-identical
// after a priority PATCH.
func TestPriorityPatchBodyPreservation(t *testing.T) {
	const body = "This is the exact body.\n\nWith multiple paragraphs.\n"
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/prio-body.md",
			content: makeArtifact("Body Preserve", "idea", "draft", "prio-body", "", body),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const path = "lifecycle/ideas/prio-body.md"

	// Capture body before PATCH.
	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+path, nil)
	requireStatus(t, resp, 200)
	before := readJSON(t, resp)
	bodyBefore, _ := before["body"].(string)

	// Apply PATCH.
	resp2 := env.doRequest("PATCH", "/api/p/testproject/artifacts/"+path+"/priority", map[string]any{
		"priority": "high",
	})
	requireStatus(t, resp2, 200)
	resp2.Body.Close()

	// Capture body after PATCH.
	resp3 := env.doRequest("GET", "/api/p/testproject/artifacts/"+path, nil)
	requireStatus(t, resp3, 200)
	after := readJSON(t, resp3)
	bodyAfter, _ := after["body"].(string)

	if bodyBefore != bodyAfter {
		t.Errorf("body changed after priority PATCH:\nbefore: %q\nafter:  %q", bodyBefore, bodyAfter)
	}
}

// TestPriorityPatchWebSocketEvent verifies that a successful PATCH emits an
// artifact.indexed WebSocket event.
func TestPriorityPatchWebSocketEvent(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/prio-ws.md",
			content: makeArtifact("Priority WS Event", "idea", "draft", "prio-ws", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

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

	const path = "lifecycle/ideas/prio-ws.md"
	resp := env.doRequest("PATCH", "/api/p/testproject/artifacts/"+path+"/priority", map[string]any{
		"priority": "medium",
	})
	requireStatus(t, resp, 200)
	resp.Body.Close()

	// Wait for the artifact.indexed event.
	deadline := time.After(5 * time.Second)
	for {
		select {
		case event := <-eventCh:
			typ, _ := event["type"].(string)
			if typ == "artifact.indexed" {
				payload, _ := event["payload"].(map[string]any)
				if eventPath, _ := payload["path"].(string); eventPath == path {
					return // success
				}
			}
		case <-deadline:
			t.Fatal("timed out waiting for artifact.indexed WebSocket event after PATCH")
		}
	}
}

// buildCookieHeader formats a slice of cookies into a Cookie header string.
func buildCookieHeader(cookies []*http.Cookie) string {
	parts := make([]string, 0, len(cookies))
	for _, c := range cookies {
		parts = append(parts, c.Name+"="+c.Value)
	}
	return strings.Join(parts, "; ")
}

// decodeGraphNodes is a convenience helper that decodes the "nodes" field from
// a raw JSON body (as map[string]any) into a typed slice.
func decodeGraphNodes(t *testing.T, data map[string]any) []any {
	t.Helper()
	nodes, ok := data["nodes"].([]any)
	if !ok {
		t.Fatal("graph response missing 'nodes' array")
	}
	return nodes
}

// graphNodeLabels extracts the labels field from a graph node map.
// Returns nil only if the field is entirely absent — the caller should
// distinguish nil vs []any{}.
func graphNodeLabels(node map[string]any) []any {
	raw, exists := node["labels"]
	if !exists {
		return nil
	}
	labels, _ := raw.([]any)
	return labels
}

// graphResponseForProject calls GET /graph on the testproject and returns the
// decoded JSON map.
func graphResponseForProject(t *testing.T, env *testEnv) map[string]any {
	t.Helper()
	resp, err := http.Get(env.baseURL + "/api/p/testproject/graph")
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, resp, 200)
	return readJSON(t, resp)
}

// createArtifactViaAPI creates an artifact using the POST /artifacts endpoint
// and returns its relative path.
func createArtifactViaAPI(t *testing.T, env *testEnv, stage, slug string, fm map[string]any, body string) string {
	t.Helper()
	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage":       stage,
		"slug":        slug,
		"frontmatter": fm,
		"body":        body,
	})
	requireStatus(t, resp, 201)
	data := readJSON(t, resp)
	path, _ := data["path"].(string)
	if path == "" {
		t.Fatal("POST /artifacts response missing 'path'")
	}
	return path
}

// artifactFrontmatterJSON reads the frontmatter_json column via the GET endpoint.
// Returns the artifact's frontmatter as a Go map decoded from JSON.
func artifactFrontmatterJSON(t *testing.T, env *testEnv, path string) map[string]any {
	t.Helper()
	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+path, nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	artifact, _ := data["artifact"].(map[string]any)
	fm, _ := artifact["frontmatter"].(map[string]any)
	return fm
}

// roundTripJSON round-trips a value through JSON encoding/decoding so that
// nested structures all become map[string]any / []any.
func roundTripJSON(v any) (map[string]any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	err = json.Unmarshal(b, &out)
	return out, err
}
