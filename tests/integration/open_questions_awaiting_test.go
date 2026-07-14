// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// TestOpenQuestionsAwaiting_ListReturnsExactBlockedSet verifies that
// GET .../artifacts?awaiting_answers=true returns exactly the artifacts that
// are "blocked" with a non-empty "## Open Questions" section, excluding a
// draft artifact and a done artifact that carry no open questions.
func TestOpenQuestionsAwaiting_ListReturnsExactBlockedSet(t *testing.T) {
	seeds := []seedArtifact{
		{relPath: "lifecycle/requirements/awaiting-target-1.md",
			content: makeBlockedArtifact("Awaiting Target 1", "ticket", "awaiting-target-1", "")},
		{relPath: "lifecycle/requirements/awaiting-target-2.md",
			content: makeBlockedArtifact("Awaiting Target 2", "ticket", "awaiting-target-2", "")},
		{relPath: "lifecycle/requirements/awaiting-decoy-draft.md",
			content: makeArtifact("Awaiting Decoy Draft", "ticket", "draft", "awaiting-decoy-draft", "", "No open questions here.")},
		{relPath: "lifecycle/requirements/awaiting-decoy-done.md",
			content: makeArtifact("Awaiting Decoy Done", "ticket", "done", "awaiting-decoy-done", "", "Also no open questions.")},
	}
	env := newTestEnv(t, seeds)

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?awaiting_answers=true", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	got := map[string]bool{}
	for _, it := range items {
		row, _ := it.(map[string]any)
		path, _ := row["path"].(string)
		if status, _ := row["status"].(string); status != "blocked" {
			t.Errorf("awaiting_answers item %q has status %q, want blocked", path, status)
		}
		got[path] = true
	}

	want := []string{
		"lifecycle/requirements/awaiting-target-1.md",
		"lifecycle/requirements/awaiting-target-2.md",
	}
	if len(got) != len(want) {
		t.Fatalf("expected exactly %d awaiting-answers artifacts, got %d: %v", len(want), len(got), got)
	}
	for _, p := range want {
		if !got[p] {
			t.Errorf("expected %q in awaiting_answers list, missing", p)
		}
	}
	for _, decoy := range []string{
		"lifecycle/requirements/awaiting-decoy-draft.md",
		"lifecycle/requirements/awaiting-decoy-done.md",
	} {
		if got[decoy] {
			t.Errorf("decoy %q unexpectedly present in awaiting_answers list", decoy)
		}
	}
}

// TestOpenQuestionsAwaiting_CountOnly verifies that adding count_only=true to
// the same awaiting_answers=true query returns just {"count":N} matching the
// number of blocked-with-open-questions artifacts.
func TestOpenQuestionsAwaiting_CountOnly(t *testing.T) {
	seeds := []seedArtifact{
		{relPath: "lifecycle/requirements/awaiting-count-1.md",
			content: makeBlockedArtifact("Awaiting Count 1", "ticket", "awaiting-count-1", "")},
		{relPath: "lifecycle/requirements/awaiting-count-2.md",
			content: makeBlockedArtifact("Awaiting Count 2", "ticket", "awaiting-count-2", "")},
		{relPath: "lifecycle/requirements/awaiting-count-3.md",
			content: makeBlockedArtifact("Awaiting Count 3", "ticket", "awaiting-count-3", "")},
		{relPath: "lifecycle/requirements/awaiting-count-decoy.md",
			content: makeArtifact("Awaiting Count Decoy", "ticket", "draft", "awaiting-count-decoy", "", "No open questions.")},
	}
	env := newTestEnv(t, seeds)

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?awaiting_answers=true&count_only=true", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	count, ok := data["count"].(float64)
	if !ok {
		t.Fatalf("expected numeric 'count' field, got %v", data["count"])
	}
	if _, hasItems := data["items"]; hasItems {
		t.Errorf("count_only=true response should not include 'items', got %v", data["items"])
	}
	if int(count) != 3 {
		t.Errorf("expected count 3, got %d", int(count))
	}
}

// TestOpenQuestionsAwaiting_ResolvingLastDropsCountToZero verifies that once
// the single remaining blocked-with-open-questions artifact is fully
// resolved (preview complete=true, then persisted via PUT), the
// awaiting_answers count drops from 1 to 0.
func TestOpenQuestionsAwaiting_ResolvingLastDropsCountToZero(t *testing.T) {
	const relPath = "lifecycle/requirements/awaiting-last.md"
	seeds := []seedArtifact{
		{relPath: relPath, content: makeBlockedArtifact("Awaiting Last", "ticket", "awaiting-last", "")},
	}
	env := newTestEnv(t, seeds)

	countBefore := awaitingAnswersCount(t, env)
	if countBefore != 1 {
		t.Fatalf("expected initial awaiting_answers count 1, got %d", countBefore)
	}

	// Preview a complete resolution (single question in makeBlockedArtifact's body).
	previewResp := env.doRequest("POST", "/api/p/testproject/artifacts/"+relPath+"/open-questions/preview", map[string]any{
		"answers":  map[string]string{"0": "Rayleigh scattering."},
		"complete": true,
	})
	requireStatus(t, previewResp, 200)
	previewData := readJSON(t, previewResp)
	newBody, _ := previewData["body"].(string)
	if !strings.Contains(newBody, "## Resolved Questions") {
		t.Fatalf("expected preview body to rename heading to '## Resolved Questions', got: %q", newBody)
	}

	// Persist via the existing PUT (no status field authored by the client).
	putResp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPath, map[string]any{
		"frontmatter": map[string]any{
			"title":   "Awaiting Last",
			"type":    "ticket",
			"status":  "blocked",
			"lineage": "awaiting-last",
		},
		"body": newBody,
	})
	requireStatus(t, putResp, 200)
	putData := readJSON(t, putResp)
	art, _ := putData["artifact"].(map[string]any)
	fm, _ := art["frontmatter"].(map[string]any)
	if status, _ := fm["status"].(string); status != "draft" {
		t.Errorf("expected auto-unblock to 'draft' after complete resolve, got %q", status)
	}

	countAfter := awaitingAnswersCount(t, env)
	if countAfter != 0 {
		t.Errorf("expected awaiting_answers count 0 after resolving the last artifact, got %d", countAfter)
	}
}

// awaitingAnswersCount fetches the awaiting_answers=true&count_only=true count.
func awaitingAnswersCount(t *testing.T, env *testEnv) int {
	t.Helper()
	resp := env.doRequest("GET", "/api/p/testproject/artifacts?awaiting_answers=true&count_only=true", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	count, ok := data["count"].(float64)
	if !ok {
		t.Fatalf("expected numeric 'count' field, got %v", data["count"])
	}
	return int(count)
}

// TestOpenQuestionsAwaiting_WebSocketEventOnResolve verifies that resolving a
// blocked-with-open-questions artifact (preview complete=true, then PUT)
// fires the existing artifact.indexed WebSocket event for that path, so the
// frontend awaiting-answers badge can update live (NFR2).
func TestOpenQuestionsAwaiting_WebSocketEventOnResolve(t *testing.T) {
	const relPath = "lifecycle/requirements/awaiting-ws.md"
	seeds := []seedArtifact{
		{relPath: relPath, content: makeBlockedArtifact("Awaiting WS", "ticket", "awaiting-ws", "")},
	}
	env := newTestEnv(t, seeds)

	previewResp := env.doRequest("POST", "/api/p/testproject/artifacts/"+relPath+"/open-questions/preview", map[string]any{
		"answers":  map[string]string{"0": "Rayleigh scattering."},
		"complete": true,
	})
	requireStatus(t, previewResp, 200)
	previewData := readJSON(t, previewResp)
	newBody, _ := previewData["body"].(string)

	// Connect the WebSocket before issuing the resolving PUT.
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

	time.Sleep(50 * time.Millisecond)

	putResp := env.doRequest("PUT", "/api/p/testproject/artifacts/"+relPath, map[string]any{
		"frontmatter": map[string]any{
			"title":   "Awaiting WS",
			"type":    "ticket",
			"status":  "blocked",
			"lineage": "awaiting-ws",
		},
		"body": newBody,
	})
	requireStatus(t, putResp, 200)
	putResp.Body.Close()

	deadline := time.After(5 * time.Second)
	for {
		select {
		case event := <-eventCh:
			typ, _ := event["type"].(string)
			if typ == "artifact.indexed" {
				payload, _ := event["payload"].(map[string]any)
				if eventPath, _ := payload["path"].(string); eventPath == relPath {
					return // success
				}
			}
		case <-deadline:
			t.Fatal("timed out waiting for artifact.indexed WebSocket event after resolving PUT")
		}
	}
}

// TestOpenQuestionsAwaitingPermissions_PreviewRequiresProductOwner verifies
// that the open-questions preview endpoint returns 403 (and performs no work)
// for a session without the product-owner role, and succeeds for a
// product-owner session (FR8, Resolved Q2).
func TestOpenQuestionsAwaitingPermissions_PreviewRequiresProductOwner(t *testing.T) {
	const relPath = "lifecycle/requirements/awaiting-perm.md"
	seeds := []seedArtifact{
		{relPath: relPath, content: makeBlockedArtifact("Awaiting Perm", "ticket", "awaiting-perm", "")},
	}
	env := newTestEnv(t, seeds)

	// dev@test.local holds backend-developer/frontend-developer/test-developer
	// roles only (see defaultCfgYAML) — not product-owner.
	env.login("dev@test.local", "dev-pass-123")
	nonOwnerResp := env.doRequest("POST", "/api/p/testproject/artifacts/"+relPath+"/open-questions/preview", map[string]any{
		"answers":  map[string]string{"0": "Rayleigh scattering."},
		"complete": true,
	})
	requireStatus(t, nonOwnerResp, http.StatusForbidden)
	nonOwnerResp.Body.Close()

	// Confirm no work was performed: the artifact is still blocked with the
	// original "## Open Questions" heading.
	getResp := env.doRequest("GET", "/api/p/testproject/artifacts/"+relPath, nil)
	requireStatus(t, getResp, 200)
	getData := readJSON(t, getResp)
	art, _ := getData["artifact"].(map[string]any)
	fm, _ := art["frontmatter"].(map[string]any)
	if status, _ := fm["status"].(string); status != "blocked" {
		t.Errorf("expected artifact to remain 'blocked' after forbidden preview attempt, got %q", status)
	}

	// admin@test.local holds product-owner and should succeed.
	env.login("admin@test.local", "admin-pass-123")
	ownerResp := env.doRequest("POST", "/api/p/testproject/artifacts/"+relPath+"/open-questions/preview", map[string]any{
		"answers":  map[string]string{"0": "Rayleigh scattering."},
		"complete": true,
	})
	requireStatus(t, ownerResp, 200)
	ownerData := readJSON(t, ownerResp)
	if _, ok := ownerData["body"].(string); !ok {
		t.Errorf("expected 'body' field in successful preview response, got %v", ownerData)
	}
}
