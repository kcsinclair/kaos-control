// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
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

// Milestone 4 (rice-scoring test plan): PATCH .../rice endpoint. Covers
// requirement §15 (write + broadcast), §19 (validation), lineage locking,
// auth/role gating, and idempotency.

// TestRicePatch_HappyPath verifies a valid PATCH persists, returns 200 with
// the re-indexed row including rice_score, and broadcasts an
// artifact.indexed action:"updated" event (requirement §15).
func TestRicePatch_HappyPath(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rice-patch-happy.md",
			content: makeArtifact("Rice Patch Happy", "idea", "draft", "rice-patch-happy", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)

	const path = "lifecycle/ideas/rice-patch-happy.md"
	resp := env.doRequest("PATCH", "/api/p/testproject/artifacts/"+path+"/rice", map[string]any{
		"rice_reach":      1000,
		"rice_impact":     2,
		"rice_confidence": 80,
		"rice_effort":     4,
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	artifactObj, _ := data["artifact"].(map[string]any)
	if artifactObj == nil {
		t.Fatal("expected artifact in response")
	}
	want := (1000.0 * 2.0 * (80.0 / 100.0)) / 4.0
	got, present := artifactObj["rice_score"]
	if !present {
		t.Fatal("expected rice_score present in PATCH response")
	}
	if got != want {
		t.Errorf("rice_score: want %v, got %v", want, got)
	}

	// GET should now reflect the same score.
	resp2 := env.doRequest("GET", "/api/p/testproject/artifacts/"+path, nil)
	requireStatus(t, resp2, 200)
	data2 := readJSON(t, resp2)
	artifact2, _ := data2["artifact"].(map[string]any)
	if artifact2["rice_score"] != want {
		t.Errorf("GET rice_score after PATCH: want %v, got %v", want, artifact2["rice_score"])
	}
}

// TestRicePatch_WebSocketEvent verifies that a successful PATCH .../rice
// emits an artifact.indexed WebSocket event (requirement §15).
func TestRicePatch_WebSocketEvent(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rice-patch-ws.md",
			content: makeArtifact("Rice Patch WS", "idea", "draft", "rice-patch-ws", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)

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

	const path = "lifecycle/ideas/rice-patch-ws.md"
	resp := env.doRequest("PATCH", "/api/p/testproject/artifacts/"+path+"/rice", map[string]any{
		"rice_reach": 100,
	})
	requireStatus(t, resp, 200)
	resp.Body.Close()

	deadline := time.After(5 * time.Second)
	for {
		select {
		case event := <-eventCh:
			typ, _ := event["type"].(string)
			if typ == "artifact.indexed" {
				payload, _ := event["payload"].(map[string]any)
				if eventPath, _ := payload["path"].(string); eventPath == path {
					if action, _ := payload["action"].(string); action != "updated" {
						t.Errorf("artifact.indexed action: want %q, got %q", "updated", action)
					}
					return
				}
			}
		case <-deadline:
			t.Fatal("timed out waiting for artifact.indexed WebSocket event after PATCH .../rice")
		}
	}
}

// TestRicePatch_InvalidInput verifies that invalid input (non-numeric,
// negative reach/impact, confidence outside 0-100, effort <= 0) is rejected
// with a field-level message and the file is left unchanged (requirement §19).
func TestRicePatch_InvalidInput(t *testing.T) {
	cases := []struct {
		name       string
		body       map[string]any
		wantStatus int
		wantField  string
	}{
		{"non-numeric reach", map[string]any{"rice_reach": "not-a-number"}, http.StatusBadRequest, ""},
		{"negative reach", map[string]any{"rice_reach": -1}, http.StatusUnprocessableEntity, "rice_reach"},
		{"negative impact", map[string]any{"rice_impact": -0.5}, http.StatusUnprocessableEntity, "rice_impact"},
		{"confidence below 0", map[string]any{"rice_confidence": -0.1}, http.StatusUnprocessableEntity, "rice_confidence"},
		{"confidence above 100", map[string]any{"rice_confidence": 100.1}, http.StatusUnprocessableEntity, "rice_confidence"},
		{"effort zero", map[string]any{"rice_effort": 0}, http.StatusUnprocessableEntity, "rice_effort"},
		{"effort negative", map[string]any{"rice_effort": -2}, http.StatusUnprocessableEntity, "rice_effort"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			slug := "rice-patch-invalid-" + strings.ReplaceAll(strings.ReplaceAll(tc.name, " ", "-"), "/", "-")
			relPath := "lifecycle/ideas/" + slug + ".md"
			seeds := []seedArtifact{
				{relPath: relPath, content: makeArtifact("Invalid Case", "idea", "draft", slug, "", "Body.")},
			}
			env := newTestEnv(t, seeds)

			before, err := os.ReadFile(filepath.Join(env.projectRoot, relPath))
			if err != nil {
				t.Fatal(err)
			}

			resp := env.doRequest("PATCH", "/api/p/testproject/artifacts/"+relPath+"/rice", tc.body)
			requireStatus(t, resp, tc.wantStatus)
			data := readJSON(t, resp)

			errObj, _ := data["error"].(map[string]any)
			if errObj == nil {
				t.Fatal("expected error object in response")
			}
			if msg, _ := errObj["message"].(string); msg == "" {
				t.Error("expected a non-empty field-level error message")
			}
			if tc.wantField != "" {
				if field, _ := errObj["field"].(string); field != tc.wantField {
					t.Errorf("error field: want %q, got %q", tc.wantField, field)
				}
			}

			after, err := os.ReadFile(filepath.Join(env.projectRoot, relPath))
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) {
				t.Errorf("file was modified despite invalid PATCH:\nbefore:\n%s\nafter:\n%s", before, after)
			}
		})
	}
}

// TestRicePatch_ForeignLineageLock verifies that a lineage locked by another
// user causes PATCH .../rice to return 423 Locked.
func TestRicePatch_ForeignLineageLock(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rice-patch-locked.md",
			content: makeArtifact("Rice Patch Locked", "idea", "draft", "rice-patch-locked", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)

	if _, err := env.proj.Locks.Acquire("rice-patch-locked", "someone-else@test.local", "editor"); err != nil {
		t.Fatalf("failed to seed foreign lock: %v", err)
	}

	resp := env.doRequest("PATCH", "/api/p/testproject/artifacts/lifecycle/ideas/rice-patch-locked.md/rice", map[string]any{
		"rice_reach": 100,
	})
	requireStatus(t, resp, http.StatusLocked)
	data := readJSON(t, resp)
	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "locked" {
		t.Errorf("error code: want %q, got %q", "locked", code)
	}
}

// TestRicePatch_Unauthenticated verifies that an anonymous request is
// rejected with 401.
func TestRicePatch_Unauthenticated(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rice-patch-anon.md",
			content: makeArtifact("Rice Patch Anon", "idea", "draft", "rice-patch-anon", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.logout()

	resp := env.doRequest("PATCH", "/api/p/testproject/artifacts/lifecycle/ideas/rice-patch-anon.md/rice", map[string]any{
		"rice_reach": 100,
	})
	requireStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

// TestRicePatch_UnauthorizedRole verifies that a user without a
// RolesPriorityEditors role (product-owner, analyst) is rejected with 403.
func TestRicePatch_UnauthorizedRole(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rice-patch-forbidden.md",
			content: makeArtifact("Rice Patch Forbidden", "idea", "draft", "rice-patch-forbidden", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("dev@test.local", "dev-pass-123") // backend/frontend/test-developer — not a priority editor

	resp := env.doRequest("PATCH", "/api/p/testproject/artifacts/lifecycle/ideas/rice-patch-forbidden.md/rice", map[string]any{
		"rice_reach": 100,
	})
	requireStatus(t, resp, http.StatusForbidden)
	resp.Body.Close()
}

// TestRicePatch_Idempotent verifies that PATCHing identical bodies twice
// leaves the file unchanged the second time.
func TestRicePatch_Idempotent(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rice-patch-idempotent.md",
			content: makeArtifact("Rice Patch Idempotent", "idea", "draft", "rice-patch-idempotent", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)

	const path = "lifecycle/ideas/rice-patch-idempotent.md"
	body := map[string]any{
		"rice_reach":      100,
		"rice_impact":     1,
		"rice_confidence": 50,
		"rice_effort":     1,
	}

	resp1 := env.doRequest("PATCH", "/api/p/testproject/artifacts/"+path+"/rice", body)
	requireStatus(t, resp1, 200)
	resp1.Body.Close()

	first, err := os.ReadFile(filepath.Join(env.projectRoot, path))
	if err != nil {
		t.Fatal(err)
	}

	resp2 := env.doRequest("PATCH", "/api/p/testproject/artifacts/"+path+"/rice", body)
	requireStatus(t, resp2, 200)
	resp2.Body.Close()

	second, err := os.ReadFile(filepath.Join(env.projectRoot, path))
	if err != nil {
		t.Fatal(err)
	}

	if string(first) != string(second) {
		t.Errorf("identical PATCH bodies produced different file contents:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}
