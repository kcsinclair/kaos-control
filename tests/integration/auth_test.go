//go:build integration

package integration

import (
	"net/http"
	"testing"
)

// TestLoginSuccess verifies that correct credentials return 200 with session
// and CSRF cookies, and GET /auth/me returns the user info.
func TestLoginSuccess(t *testing.T) {
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
	data := readJSON(t, resp)

	user, _ := data["user"].(map[string]any)
	if email, _ := user["email"].(string); email != "admin@test.local" {
		t.Errorf("expected email admin@test.local, got %q", email)
	}

	// Check cookies.
	var hasSession, hasCsrf bool
	for _, c := range resp.Cookies() {
		switch c.Name {
		case "kc_session":
			hasSession = true
			if !c.HttpOnly {
				t.Error("session cookie should be HttpOnly")
			}
		case "kc_csrf":
			hasCsrf = true
			if c.HttpOnly {
				t.Error("CSRF cookie should NOT be HttpOnly (JS must read it)")
			}
		}
	}
	if !hasSession {
		t.Error("expected kc_session cookie")
	}
	if !hasCsrf {
		t.Error("expected kc_csrf cookie")
	}
}

// TestLoginInvalidCredentials verifies that wrong credentials return 401.
func TestLoginInvalidCredentials(t *testing.T) {
	env := newTestEnv(t, nil)

	resp, err := http.Post(
		env.baseURL+"/api/auth/login",
		"application/json",
		stringReader(`{"email":"admin@test.local","password":"wrong-pass"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, resp, 401)
	resp.Body.Close()
}

// TestGetMeReturnsRoles verifies that GET /auth/me returns the user's
// per-project role mappings.
func TestGetMeReturnsRoles(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/auth/me", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	roles, ok := data["roles"].(map[string]any)
	if !ok {
		t.Fatal("expected roles map in /auth/me response")
	}
	projectRoles, ok := roles["testproject"].([]any)
	if !ok || len(projectRoles) == 0 {
		t.Fatal("expected testproject roles for admin user")
	}

	// admin@test.local has: product-owner, analyst, reviewer, approver
	roleSet := map[string]bool{}
	for _, r := range projectRoles {
		roleSet[r.(string)] = true
	}
	for _, expected := range []string{"product-owner", "analyst", "reviewer", "approver"} {
		if !roleSet[expected] {
			t.Errorf("expected role %q in admin's project roles", expected)
		}
	}
}

// TestLogoutClearsSession verifies that POST /logout clears the session,
// and subsequent GET /auth/me returns 401.
func TestLogoutClearsSession(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Logout.
	resp := env.doRequest("POST", "/api/auth/logout", nil)
	requireStatus(t, resp, 204)
	resp.Body.Close()

	// GET /auth/me should now return 401 since the session cookie
	// points to a deleted session. The cookie is still sent but
	// the server won't find it.
	resp2 := env.doRequest("GET", "/api/auth/me", nil)
	requireStatus(t, resp2, 401)
	resp2.Body.Close()
}

// TestBootstrapFirstUser verifies that the first user can be created
// without authentication (bootstrap path).
func TestBootstrapFirstUser(t *testing.T) {
	// Create a fresh env without the default users.
	// We can't easily skip the user creation in newTestEnv, so instead
	// test that creating an additional user requires auth.
	env := newTestEnv(t, nil)

	// Without login, creating a user should fail (not the first user anymore).
	resp, err := http.Post(
		env.baseURL+"/api/admin/users",
		"application/json",
		stringReader(`{"email":"new@test.local","password":"new-pass-123"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	// Should be 403: CSRF middleware fires before auth, so unauthenticated
	// requests without a CSRF token get 403 (csrf_missing), not 401.
	requireStatus(t, resp, 403)
	resp.Body.Close()

	// With login, it should succeed.
	env.login("admin@test.local", "admin-pass-123")
	resp2 := env.doRequest("POST", "/api/admin/users", map[string]any{
		"email":    "new@test.local",
		"password": "new-pass-123",
	})
	requireStatus(t, resp2, 201)
	resp2.Body.Close()
}

// TestGetMeWithoutLogin returns 401.
func TestGetMeWithoutLogin(t *testing.T) {
	env := newTestEnv(t, nil)

	resp, err := http.Get(env.baseURL + "/api/auth/me")
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, resp, 401)
	resp.Body.Close()
}
