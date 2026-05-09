// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"net/http"
	"testing"
)

// TestPathTraversalBlocked verifies that API attempts with "../" or absolute
// paths are rejected with 400/403 — no filesystem leak.
// Test plan §11: "Path traversal" scenario.
func TestPathTraversalBlocked(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	traversalPaths := []string{
		"/api/p/testproject/artifacts/../../etc/passwd",
		"/api/p/testproject/artifacts/../../../etc/shadow",
		"/api/p/testproject/artifacts/lifecycle/ideas/../../.env",
	}

	for _, path := range traversalPaths {
		resp := env.doRequest("GET", path, nil)
		// Should be 400 (invalid_path) or 404 (not_found), never 200 with file content.
		if resp.StatusCode == 200 {
			t.Errorf("path traversal attempt succeeded for %s", path)
		}
		resp.Body.Close()
	}
}

// TestCsrfProtection verifies that mutations without the X-CSRF-Token header
// are rejected with 403.
// Test plan §11: "CSRF" scenario.
func TestCsrfProtection(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Try a POST with wrong CSRF token.
	req, err := http.NewRequest("POST", env.baseURL+"/api/p/testproject/artifacts", stringReader(`{
		"stage": "ideas",
		"slug": "csrf-test",
		"frontmatter": {"title": "CSRF", "type": "idea", "status": "draft", "lineage": "csrf-test"},
		"body": "test"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	for _, c := range env.cookies {
		req.AddCookie(c)
	}
	req.Header.Set("X-CSRF-Token", "wrong-token-value")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, resp, 403)
	data := readJSON(t, resp)
	errData, _ := data["error"].(map[string]any)
	if code, _ := errData["code"].(string); code != "csrf_invalid" {
		t.Errorf("expected error code 'csrf_invalid', got %q", code)
	}
}

// TestSessionCookieAttributes verifies that the session cookie has the
// correct security attributes: HttpOnly, SameSite=Lax.
// Test plan §11: "Auth" scenario.
func TestSessionCookieAttributes(t *testing.T) {
	env := newTestEnv(t, nil)

	resp, err := http.Post(
		env.baseURL+"/api/auth/login",
		"application/json",
		stringReader(`{"email":"admin@test.local","password":"admin-pass-123"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, resp, 200)
	resp.Body.Close()

	for _, c := range resp.Cookies() {
		if c.Name == "kc_session" {
			if !c.HttpOnly {
				t.Error("session cookie must be HttpOnly")
			}
			if c.SameSite != http.SameSiteLaxMode {
				t.Errorf("session cookie SameSite should be Lax, got %v", c.SameSite)
			}
			// Secure is false since we're testing over HTTP (not TLS).
			return
		}
	}
	t.Error("session cookie not found")
}

// TestUnauthorizedMutationReturns401 verifies that attempting a mutation
// that requires auth without login returns 401.
func TestUnauthorizedMutationReturns401(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/protected.md",
			content: makeArtifact("Protected", "idea", "draft", "protected", "", "Needs auth."),
		},
	}
	env := newTestEnv(t, seeds)

	// Transition requires authentication.
	resp, err := http.Post(
		env.baseURL+"/api/p/testproject/artifacts/lifecycle/ideas/protected.md/transition",
		"application/json",
		stringReader(`{"to":"clarifying"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	// Should be either 401 or 403 (CSRF missing since no login).
	if resp.StatusCode != 401 && resp.StatusCode != 403 {
		t.Errorf("expected 401 or 403 for unauthenticated mutation, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}
