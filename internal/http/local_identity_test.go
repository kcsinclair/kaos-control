// SPDX-License-Identifier: AGPL-3.0-or-later

package http

// Milestone 2 — Unit tests for the X-Kaos-Local-User loopback identity path.
//
// Tests the sessionMiddleware's loopback-trusted-identity branch using httptest.
// Key behaviours verified:
//   - Loopback RemoteAddr + known email → request authenticated as that user.
//   - Non-loopback RemoteAddr + header → header silently ignored; caller is
//     unauthenticated and the downstream handler returns 401.
//   - Unknown email on loopback → unauthenticated (hard 401 from handler).
//   - Existing session cookie wins over X-Kaos-Local-User (no silent elevation).
//   - Existing bearer token wins over X-Kaos-Local-User (no silent elevation).
//
// Run with: go test ./internal/http/ -run TestLocalIdentity

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/auth"
)

// buildLocalIdentityServer builds a minimal *Server backed by a real auth.Store
// that has one pre-created user (localid@test.local / "pass").
func buildLocalIdentityServer(t *testing.T) (*Server, func()) {
	t.Helper()
	dir := t.TempDir()
	store, err := auth.Open(filepath.Join(dir, "auth.db"), 24*time.Hour)
	if err != nil {
		t.Fatalf("auth.Open: %v", err)
	}
	if err := store.CreateUser("localid@test.local", "Local ID", "pass", false); err != nil {
		_ = store.Close()
		t.Fatalf("CreateUser: %v", err)
	}
	s := &Server{
		cfg:      ServerConfig{Auth: store},
		projects: nil,
	}
	return s, func() { _ = store.Close() }
}

// whoHandler is a minimal HTTP handler that returns 200 + the authenticated
// user's email, or 401 if no user is in context.
func whoHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFromCtx(r.Context())
		if u == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(u.Email))
	})
}

// TestLocalIdentity_LoopbackKnownEmail_Authenticated verifies that a request
// from loopback carrying a known email is authenticated as that user.
func TestLocalIdentity_LoopbackKnownEmail_Authenticated(t *testing.T) {
	s, cleanup := buildLocalIdentityServer(t)
	defer cleanup()

	handler := s.sessionMiddleware(whoHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	req.Header.Set("X-Kaos-Local-User", "localid@test.local")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("want 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}
	if got := rr.Body.String(); got != "localid@test.local" {
		t.Errorf("want email=localid@test.local in body, got %q", got)
	}
}

// TestLocalIdentity_LoopbackIPv6_Authenticated verifies the ::1 loopback address.
func TestLocalIdentity_LoopbackIPv6_Authenticated(t *testing.T) {
	s, cleanup := buildLocalIdentityServer(t)
	defer cleanup()

	handler := s.sessionMiddleware(whoHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = "[::1]:54321"
	req.Header.Set("X-Kaos-Local-User", "localid@test.local")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("want 200 for ::1 loopback, got %d", rr.Code)
	}
}

// TestLocalIdentity_NonLoopback_HeaderIgnored verifies that the header is
// silently ignored (no 401 from the middleware itself) on non-loopback
// connections, leaving the caller unauthenticated so the handler returns 401.
func TestLocalIdentity_NonLoopback_HeaderIgnored(t *testing.T) {
	s, cleanup := buildLocalIdentityServer(t)
	defer cleanup()

	handler := s.sessionMiddleware(whoHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = "10.0.0.1:54321"
	req.Header.Set("X-Kaos-Local-User", "localid@test.local")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// The middleware ignores the header; whoHandler returns 401 (no user in ctx).
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("want 401 (non-loopback header ignored → no auth), got %d", rr.Code)
	}
}

// TestLocalIdentity_UnknownEmail_Unauthenticated verifies that an unrecognised
// email on loopback is not silently elevated — the caller stays unauthenticated.
func TestLocalIdentity_UnknownEmail_Unauthenticated(t *testing.T) {
	s, cleanup := buildLocalIdentityServer(t)
	defer cleanup()

	handler := s.sessionMiddleware(whoHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	req.Header.Set("X-Kaos-Local-User", "ghost@test.local") // not in store
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("want 401 (unknown email → no auth), got %d", rr.Code)
	}
}

// TestLocalIdentity_SessionWinsOverHeader verifies that a valid session cookie
// takes precedence over X-Kaos-Local-User (no silent elevation, NF3).
func TestLocalIdentity_SessionWinsOverHeader(t *testing.T) {
	s, cleanup := buildLocalIdentityServer(t)
	defer cleanup()

	// Add a second user so the session belongs to a distinct identity.
	if err := s.cfg.Auth.CreateUser("other@test.local", "Other", "pass", false); err != nil {
		t.Fatalf("CreateUser other: %v", err)
	}
	sessID, err := s.cfg.Auth.CreateSession("other@test.local")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	handler := s.sessionMiddleware(whoHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	req.Header.Set("X-Kaos-Local-User", "localid@test.local")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessID})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rr.Code)
	}
	// Session user wins; X-Kaos-Local-User is ignored because the context
	// already has a user after step 1 of the middleware.
	if got := rr.Body.String(); got != "other@test.local" {
		t.Errorf("want other@test.local (session wins), got %q", got)
	}
}

// TestLocalIdentity_BearerWinsOverHeader verifies that a valid bearer token
// takes precedence over X-Kaos-Local-User (no silent elevation, NF3).
func TestLocalIdentity_BearerWinsOverHeader(t *testing.T) {
	s, cleanup := buildLocalIdentityServer(t)
	defer cleanup()

	if err := s.cfg.Auth.CreateUser("bearer@test.local", "Bearer User", "pass", false); err != nil {
		t.Fatalf("CreateUser bearer: %v", err)
	}
	token, err := s.cfg.Auth.CreateToken("bearer@test.local", nil)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	handler := s.sessionMiddleware(whoHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	req.Header.Set("X-Kaos-Local-User", "localid@test.local")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rr.Code)
	}
	// Bearer token was processed first (step 2); X-Kaos-Local-User is ignored
	// because userFromCtx is already non-nil when step 3 runs.
	if got := rr.Body.String(); got != "bearer@test.local" {
		t.Errorf("want bearer@test.local (bearer wins), got %q", got)
	}
}
