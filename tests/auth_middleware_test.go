// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

// HTTP auth middleware integration tests.
// These tests start the compiled kaos-control binary as a subprocess on a
// random port, pre-populate the auth DB directly via the auth package, and
// then send HTTP requests to verify the middleware behaviour described in
// Milestone 4 and Milestone 5 of the test plan.
package cli_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/auth"
)

// ─── Server-start helpers ─────────────────────────────────────────────────────

// freePort binds a TCP listener on 127.0.0.1:0, records the chosen port,
// closes the listener, and returns the port. There is a small race window
// between Close and the server binding, but this is acceptable for tests.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return port
}

// middlewareTestSetup holds everything needed by a single middleware test:
// the server base URL, a direct handle on the auth store (for pre-populating
// users/tokens), and the data dir path.
type middlewareTestSetup struct {
	baseURL   string
	authStore *auth.Store
	dataDir   string
}

// startMiddlewareServer writes a config file, opens a direct auth store, starts
// the server subprocess on a free port, and waits until /api/health responds.
// If captureOutput is true, server stdout is returned via a buffer pointer
// (useful for log-assertion tests); otherwise stdout is discarded.
func startMiddlewareServer(t *testing.T, captureOutput bool) (*middlewareTestSetup, *bytes.Buffer) {
	t.Helper()

	port := freePort(t)
	dataDir := t.TempDir()
	cfgDir := t.TempDir()

	// Write a minimal server config.
	cfgPath := filepath.Join(cfgDir, "config.yaml")
	cfgContent := fmt.Sprintf(
		"data_dir: %q\nserver:\n  listen: \"127.0.0.1:%d\"\n",
		dataDir, port,
	)
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}

	// Open a direct auth store so tests can pre-populate the DB.
	store, err := auth.Open(filepath.Join(dataDir, "auth.db"), 24*time.Hour)
	if err != nil {
		t.Fatalf("auth.Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	// Start the server subprocess.
	cmd := newBinCmd(t, "serve", "--config", cfgPath)
	var outBuf bytes.Buffer
	if captureOutput {
		cmd.Stdout = &outBuf
	} else {
		cmd.Stdout = io.Discard
	}
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		t.Fatalf("starting server subprocess: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		_ = cmd.Wait()
	})

	// Poll /api/health until ready or timeout.
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	healthURL := baseURL + "/api/health"
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(healthURL) //nolint:gosec
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	// Final check: server must be reachable.
	resp, err := http.Get(healthURL) //nolint:gosec
	if err != nil {
		t.Fatalf("server did not become ready within 15s: %v", err)
	}
	resp.Body.Close()

	return &middlewareTestSetup{
		baseURL:   baseURL,
		authStore: store,
		dataDir:   dataDir,
	}, &outBuf
}

// login performs POST /api/auth/login with the given credentials and returns
// the response cookies on success.
func login(t *testing.T, baseURL, email, password string) []*http.Cookie {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	resp, err := http.Post(baseURL+"/api/auth/login", "application/json", bytes.NewReader(body)) //nolint:gosec
	if err != nil {
		t.Fatalf("POST /api/auth/login: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: want 200, got %d", resp.StatusCode)
	}
	return resp.Cookies()
}

// cookieJarHeader builds the Cookie header value from a slice of cookies.
func cookieJarHeader(cookies []*http.Cookie) string {
	parts := make([]string, len(cookies))
	for i, c := range cookies {
		parts[i] = c.Name + "=" + c.Value
	}
	return strings.Join(parts, "; ")
}

// csrfToken extracts the kc_csrf cookie value from a cookie slice.
func csrfToken(cookies []*http.Cookie) string {
	for _, c := range cookies {
		if c.Name == "kc_csrf" {
			return c.Value
		}
	}
	return ""
}

// ─── Milestone 4: HTTP Auth Middleware Tests ──────────────────────────────────

// TestUnauthenticatedRequest_Returns401 asserts that a protected API endpoint
// returns 401 with {"error":"unauthorized"} when no credentials are supplied.
func TestUnauthenticatedRequest_Returns401(t *testing.T) {
	setup, _ := startMiddlewareServer(t, false)
	// Use GET /api/projects — requires auth, no project context needed.
	resp, err := http.Get(setup.baseURL + "/api/projects") //nolint:gosec
	if err != nil {
		t.Fatalf("GET /api/projects: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", resp.StatusCode)
	}
	var body map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] != "unauthorized" {
		t.Errorf("want error=unauthorized, got %v", body)
	}
}

// TestSessionCookieAuth_Returns200 asserts that a valid session cookie grants
// access to a protected endpoint.
func TestSessionCookieAuth_Returns200(t *testing.T) {
	setup, _ := startMiddlewareServer(t, false)
	if err := setup.authStore.CreateUser("sess@test.com", "Sess", "pass", false); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	cookies := login(t, setup.baseURL, "sess@test.com", "pass")

	req, _ := http.NewRequest(http.MethodGet, setup.baseURL+"/api/projects", nil)
	req.Header.Set("Cookie", cookieJarHeader(cookies))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/projects with cookie: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("want 200, got %d", resp.StatusCode)
	}
}

// TestBearerTokenAuth_Returns200 asserts that a valid bearer token grants
// access to a protected endpoint.
func TestBearerTokenAuth_Returns200(t *testing.T) {
	setup, _ := startMiddlewareServer(t, false)
	if err := setup.authStore.CreateUser("bear@test.com", "Bear", "pass", false); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	token, err := setup.authStore.CreateToken("bear@test.com", nil)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	req, _ := http.NewRequest(http.MethodGet, setup.baseURL+"/api/projects", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/projects with bearer: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("want 200, got %d", resp.StatusCode)
	}
}

// TestExpiredSession_Returns401 manually expires a session in the DB and
// asserts the server returns 401 on the next request.
func TestExpiredSession_Returns401(t *testing.T) {
	setup, _ := startMiddlewareServer(t, false)
	if err := setup.authStore.CreateUser("exp@test.com", "Exp", "pass", false); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// Use a negative TTL so expires_at is stored as a Unix timestamp in the past.
	// GetSession checks time.Now().Unix() > expiresAt (second granularity),
	// so we need expires_at < now in whole seconds.
	shortStore, err := auth.Open(filepath.Join(setup.dataDir, "auth.db"), -2*time.Second)
	if err != nil {
		t.Fatalf("auth.Open (past TTL): %v", err)
	}
	defer shortStore.Close()

	sessID, err := shortStore.CreateSession("exp@test.com")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	req, _ := http.NewRequest(http.MethodGet, setup.baseURL+"/api/projects", nil)
	req.Header.Set("Cookie", "kc_session="+sessID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/projects with expired session: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("want 401 for expired session, got %d", resp.StatusCode)
	}
}

// TestExpiredToken_Returns401 creates a token with a past expiry and asserts
// the server returns 401.
func TestExpiredToken_Returns401(t *testing.T) {
	setup, _ := startMiddlewareServer(t, false)
	if err := setup.authStore.CreateUser("exptok@test.com", "ExpTok", "pass", false); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	past := time.Now().Add(-1 * time.Second)
	token, err := setup.authStore.CreateToken("exptok@test.com", &past)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	req, _ := http.NewRequest(http.MethodGet, setup.baseURL+"/api/projects", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/projects with expired token: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("want 401 for expired token, got %d", resp.StatusCode)
	}
}

// TestHealthEndpoint_NoAuth asserts GET /api/health returns 200 without auth.
func TestHealthEndpoint_NoAuth(t *testing.T) {
	setup, _ := startMiddlewareServer(t, false)
	resp, err := http.Get(setup.baseURL + "/api/health") //nolint:gosec
	if err != nil {
		t.Fatalf("GET /api/health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("want 200, got %d", resp.StatusCode)
	}
}

// TestStaticAssets_NoAuth asserts that /, /index.html, and /assets/* are
// accessible without any credentials (middleware exempts them).
func TestStaticAssets_NoAuth(t *testing.T) {
	setup, _ := startMiddlewareServer(t, false)
	paths := []string{"/", "/index.html", "/assets/nonexistent.js"}
	for _, path := range paths {
		resp, err := http.Get(setup.baseURL + path) //nolint:gosec
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusUnauthorized {
			t.Errorf("GET %s: want not-401, got 401", path)
		}
	}
}

// TestLoginEndpoint_NoAuth asserts that POST /api/auth/login is never blocked
// by the auth middleware (401), even when credentials are wrong.
func TestLoginEndpoint_NoAuth(t *testing.T) {
	setup, _ := startMiddlewareServer(t, false)
	body, _ := json.Marshal(map[string]string{"email": "nobody@test.com", "password": "wrong"})
	resp, err := http.Post(setup.baseURL+"/api/auth/login", "application/json", bytes.NewReader(body)) //nolint:gosec
	if err != nil {
		t.Fatalf("POST /api/auth/login: %v", err)
	}
	defer resp.Body.Close()
	// The middleware must not return 401. The handler may return 401
	// (invalid_credentials) or 503 (auth_disabled), but the code path is the
	// handler's choice, not the auth middleware.
	// We verify this by checking the response body does NOT contain "unauthorized"
	// (the middleware sentinel) when the status IS 401.
	if resp.StatusCode == http.StatusUnauthorized {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		if code, _ := errBody["code"].(string); code == "unauthorized" {
			t.Error("POST /api/auth/login returned 401 unauthorized from middleware; login endpoint must be exempt")
		}
	}
}

// TestWebSocketAuth_Rejected asserts that a WS upgrade attempt to a protected
// route without credentials is rejected with 401.
func TestWebSocketAuth_Rejected(t *testing.T) {
	setup, _ := startMiddlewareServer(t, false)
	// The auth middleware runs before the project middleware, so even an
	// unknown project will return 401 (not 404) when unauthenticated.
	req, _ := http.NewRequest(http.MethodGet, setup.baseURL+"/api/p/dummy/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("WS upgrade request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("want 401 for unauthenticated WS upgrade, got %d", resp.StatusCode)
	}
}

// TestBearerAuth_SkipsCsrf asserts that a POST request authenticated via
// bearer token succeeds (or fails for handler-level reasons) without a
// X-CSRF-Token header — i.e., the CSRF middleware is bypassed for bearer auth.
func TestBearerAuth_SkipsCsrf(t *testing.T) {
	setup, _ := startMiddlewareServer(t, false)
	if err := setup.authStore.CreateUser("bearer_csrf@test.com", "BearerCSRF", "pass", false); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	token, err := setup.authStore.CreateToken("bearer_csrf@test.com", nil)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	// POST /api/admin/users with bearer token and NO X-CSRF-Token.
	// The user already exists so the handler will return 409 (conflict), but
	// it must NOT be 403 (CSRF failure).
	body, _ := json.Marshal(map[string]string{
		"email":    "bearer_csrf@test.com",
		"password": "anypass",
	})
	req, _ := http.NewRequest(http.MethodPost, setup.baseURL+"/api/admin/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	// Deliberately omit X-CSRF-Token.

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/admin/users with bearer: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden {
		t.Error("bearer-authenticated POST returned 403 Forbidden (CSRF not skipped for bearer token)")
	}
}

// TestSessionAuth_RequiresCsrf asserts that a POST request authenticated via
// session cookie but lacking the X-CSRF-Token header is rejected with 403.
func TestSessionAuth_RequiresCsrf(t *testing.T) {
	setup, _ := startMiddlewareServer(t, false)
	if err := setup.authStore.CreateUser("sess_csrf@test.com", "SessCSRF", "pass", false); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	cookies := login(t, setup.baseURL, "sess_csrf@test.com", "pass")

	// POST /api/admin/users with session cookie but NO X-CSRF-Token.
	body, _ := json.Marshal(map[string]string{
		"email":    "newuser@test.com",
		"password": "anypass",
	})
	req, _ := http.NewRequest(http.MethodPost, setup.baseURL+"/api/admin/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookieJarHeader(cookies))
	// Deliberately omit X-CSRF-Token.

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/admin/users with session, no CSRF: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("want 403 (CSRF failure) for session POST without CSRF token, got %d", resp.StatusCode)
	}
}

// ─── Milestone 5: No-User Startup Warning Tests ───────────────────────────────

// TestNoUserWarning starts the server with an empty auth DB and asserts the
// startup log contains the `auth create-user` command hint.
func TestNoUserWarning(t *testing.T) {
	_, outBuf := startMiddlewareServer(t, true /* captureOutput */)

	// Give the server a moment to flush startup logs.
	time.Sleep(200 * time.Millisecond)

	got := outBuf.String()
	if !strings.Contains(got, "auth create-user") {
		t.Errorf("startup log missing 'auth create-user' warning\ngot stdout:\n%s", got)
	}
}

// TestNoWarningWithUsers starts the server after pre-populating the auth DB
// with one user and asserts the no-user warning is absent from startup logs.
func TestNoWarningWithUsers(t *testing.T) {
	port := freePort(t)
	dataDir := t.TempDir()
	cfgDir := t.TempDir()

	cfgPath := filepath.Join(cfgDir, "config.yaml")
	cfgContent := fmt.Sprintf(
		"data_dir: %q\nserver:\n  listen: \"127.0.0.1:%d\"\n",
		dataDir, port,
	)
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}

	// Pre-create a user BEFORE starting the server.
	store, err := auth.Open(filepath.Join(dataDir, "auth.db"), 24*time.Hour)
	if err != nil {
		t.Fatalf("auth.Open: %v", err)
	}
	if err := store.CreateUser("preexist@test.com", "PreExist", "pass", false); err != nil {
		_ = store.Close()
		t.Fatalf("CreateUser: %v", err)
	}
	_ = store.Close()

	// Start the server capturing stdout.
	cmd := newBinCmd(t, "serve", "--config", cfgPath)
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting server: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		_ = cmd.Wait()
	})

	// Wait for server to be ready.
	healthURL := fmt.Sprintf("http://127.0.0.1:%d/api/health", port)
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(healthURL) //nolint:gosec
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(200 * time.Millisecond)
	got := outBuf.String()
	if strings.Contains(got, "auth create-user") {
		t.Errorf("startup log contains 'auth create-user' warning when users already exist\ngot stdout:\n%s", got)
	}
}
