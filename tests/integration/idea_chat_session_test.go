// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"io"
	"net/http"
	"testing"
)

// Milestone 1 – Session Lifecycle Tests
//
// These tests cover session creation, reuse, expiry detection, input validation,
// and authentication enforcement for POST /api/p/:project/ideas/converse.

// TestIdeaChatUnknownSession verifies that sending a message with a fabricated
// session_id returns HTTP 404 with code "session_not_found".
// Does NOT require ANTHROPIC_API_KEY.
func TestIdeaChatUnknownSession(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/ideas/converse", map[string]any{
		"session_id": "00000000000000000000000000000000",
		"message":    "hello",
	})
	requireStatus(t, resp, 404)
	data := readJSON(t, resp)

	errData, _ := data["error"].(map[string]any)
	if code, _ := errData["code"].(string); code != "session_not_found" {
		t.Errorf("expected error code 'session_not_found', got %q", code)
	}
}

// TestIdeaChatEmptyMessageRejected verifies that a POST with an empty message
// returns HTTP 400 without requiring an LLM call.
// Does NOT require ANTHROPIC_API_KEY.
func TestIdeaChatEmptyMessageRejected(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/ideas/converse", map[string]any{
		"message": "",
	})
	requireStatus(t, resp, 400)
	resp.Body.Close()
}

// TestIdeaChatAuthRequired verifies that calling the converse endpoint without
// a valid session cookie returns HTTP 401 or HTTP 403 (CSRF middleware fires
// before auth, so an unauthenticated request without a CSRF token yields 403).
// Does NOT require ANTHROPIC_API_KEY.
func TestIdeaChatAuthRequired(t *testing.T) {
	env := newTestEnv(t, nil)
	// Deliberately NOT calling env.login – no cookies set.

	resp, err := http.Post(
		env.baseURL+"/api/p/testproject/ideas/converse",
		"application/json",
		stringReader(`{"message":"hello"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	// CSRF middleware runs before auth: without a CSRF token the request gets
	// 403 (csrf_missing). Either 401 or 403 confirms the request is rejected
	// before reaching the LLM or creating any session state.
	if resp.StatusCode != 401 && resp.StatusCode != 403 {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected status 401 or 403, got %d: %s", resp.StatusCode, b)
	}
	resp.Body.Close()
}

// TestIdeaChatNewSession verifies that POST with session_id absent (new session)
// and a valid message returns HTTP 200 with a non-empty session_id and
// status: "conversing".
// Requires ANTHROPIC_API_KEY.
func TestIdeaChatNewSession(t *testing.T) {
	skipIfNoAPIKey(t)
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := converseAPI(env, "", "I have an idea about a new feature")
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	sessionID, _ := data["session_id"].(string)
	if sessionID == "" {
		t.Error("expected non-empty session_id in response")
	}

	status, _ := data["status"].(string)
	if status != "conversing" && status != "proposed" {
		t.Errorf("expected status 'conversing' or 'proposed', got %q", status)
	}
}

// TestIdeaChatSessionReuse verifies that sending a second message with the
// returned session_id produces HTTP 200 and preserves the same session_id.
// Requires ANTHROPIC_API_KEY.
func TestIdeaChatSessionReuse(t *testing.T) {
	skipIfNoAPIKey(t)
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// First turn – create a session.
	resp1 := converseAPI(env, "", "I have a vague idea")
	requireStatus(t, resp1, 200)
	data1 := readJSON(t, resp1)
	sessionID, _ := data1["session_id"].(string)
	if sessionID == "" {
		t.Fatal("first turn: missing session_id")
	}

	// Second turn – reuse the session.
	resp2 := converseAPI(env, sessionID, "It involves notifications for users")
	requireStatus(t, resp2, 200)
	data2 := readJSON(t, resp2)

	returnedID, _ := data2["session_id"].(string)
	if returnedID == "" {
		t.Error("second turn: missing session_id")
	}
	if returnedID != sessionID {
		t.Errorf("second turn: expected session_id %q, got %q", sessionID, returnedID)
	}
}
