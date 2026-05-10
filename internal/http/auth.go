// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/kaos-control/kaos-control/internal/auth"
)

const (
	sessionCookieName = "kc_session"
	csrfCookieName    = "kc_csrf"
)

type authContextKey string

const userContextKey authContextKey = "auth_user"

// userFromCtx returns the authenticated user from the request context, or nil.
func userFromCtx(ctx context.Context) *auth.User {
	u, _ := ctx.Value(userContextKey).(*auth.User)
	return u
}

// bearerContextKey marks requests authenticated via bearer token (CSRF exempt).
type bearerContextKey struct{}

// isBearerAuth reports whether the request was authenticated by a bearer token.
func isBearerAuth(ctx context.Context) bool {
	v, _ := ctx.Value(bearerContextKey{}).(bool)
	return v
}

// sessionMiddleware reads the session cookie and injects the user into the context.
// If no valid session cookie is found, it falls back to checking an
// Authorization: Bearer <token> header. Bearer-authenticated requests are
// flagged in context so csrfMiddleware can skip CSRF enforcement.
// It is a no-op when the server has no auth store configured.
func (s *Server) sessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Auth != nil {
			// 1. Try session cookie.
			if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
				if user, _ := s.cfg.Auth.GetSession(cookie.Value); user != nil {
					ctx := context.WithValue(r.Context(), userContextKey, user)
					r = r.WithContext(ctx)
					next.ServeHTTP(w, r)
					return
				}
			}
			// 2. Try Authorization: Bearer <token>.
			if hdr := r.Header.Get("Authorization"); strings.HasPrefix(hdr, "Bearer ") {
				token := strings.TrimPrefix(hdr, "Bearer ")
				if user, _ := s.cfg.Auth.ValidateToken(token); user != nil {
					ctx := context.WithValue(r.Context(), userContextKey, user)
					ctx = context.WithValue(ctx, bearerContextKey{}, true)
					r = r.WithContext(ctx)
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// csrfMiddleware enforces the double-submit cookie pattern for non-GET mutations.
// Auth endpoints, bearer-token requests, and the bootstrap user-creation call are exempt.
func (s *Server) csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		// Bearer-authenticated requests are not vulnerable to CSRF.
		if isBearerAuth(r.Context()) {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/auth/") {
			next.ServeHTTP(w, r)
			return
		}
		// Bootstrap path: first-ever user creation requires no CSRF.
		if r.URL.Path == "/api/admin/users" && s.cfg.Auth != nil {
			if count, _ := s.cfg.Auth.UserCount(); count == 0 {
				next.ServeHTTP(w, r)
				return
			}
		}
		csrfCookie, err := r.Cookie(csrfCookieName)
		if err != nil || csrfCookie.Value == "" {
			writeJSON(w, http.StatusForbidden, apiError("csrf_missing", "CSRF cookie missing; re-login required"))
			return
		}
		if r.Header.Get("X-CSRF-Token") != csrfCookie.Value {
			writeJSON(w, http.StatusForbidden, apiError("csrf_invalid", "X-CSRF-Token header does not match cookie"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireAuth is a global middleware that returns 401 for unauthenticated requests
// on all paths except the explicitly exempt ones listed below.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Exempt: static SPA assets served by handleFrontend.
		if path == "/" || path == "/index.html" || path == "/favicon.ico" ||
			strings.HasPrefix(path, "/assets/") {
			next.ServeHTTP(w, r)
			return
		}

		// Exempt: login and health endpoints.
		if path == "/api/auth/login" || path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}

		// Exempt: bootstrap — first user creation when no users exist yet.
		if path == "/api/admin/users" && r.Method == http.MethodPost && s.cfg.Auth != nil {
			if count, _ := s.cfg.Auth.UserCount(); count == 0 {
				next.ServeHTTP(w, r)
				return
			}
		}

		if userFromCtx(r.Context()) == nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// handleLogin handles POST /api/auth/login
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Auth == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError("auth_disabled", "authentication not configured"))
		return
	}
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON"))
		return
	}
	user, err := s.cfg.Auth.Authenticate(req.Email, req.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("invalid_credentials", "invalid email or password"))
		return
	}
	sessionID, err := s.cfg.Auth.CreateSession(user.Email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	csrfToken := randomHex(16)
	secure := s.cfg.TLSOn
	maxAge := int(s.cfg.Auth.SessionTTL / time.Second)

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    csrfToken,
		Path:     "/",
		HttpOnly: false, // readable by JS for double-submit
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	})
	writeJSON(w, http.StatusOK, map[string]any{"user": user})
}

// handleLogout handles POST /api/auth/logout
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil && s.cfg.Auth != nil {
		_ = s.cfg.Auth.DeleteSession(cookie.Value)
	}
	// Even on logout, set the same security attributes the login cookies use.
	// Browsers honour Secure/SameSite on Set-Cookie deletion and gosec flags
	// the absence as a CWE-614 issue, so apply for hygiene.
	secure := s.cfg.TLSOn
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name: csrfCookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: false, Secure: secure, SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

// handleMe handles GET /api/auth/me
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "not logged in"))
		return
	}
	roleMap := map[string][]string{}
	for name, p := range s.projects {
		if roles := p.Cfg.RolesFor(user.Email); len(roles) > 0 {
			roleMap[name] = roles
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"email":        user.Email,
		"display_name": user.DisplayName,
		"roles":        roleMap,
	})
}

// handleCreateUser handles POST /api/admin/users
// The very first user can be created without authentication (bootstrap).
func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Auth == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError("auth_disabled", "authentication not configured"))
		return
	}
	count, err := s.cfg.Auth.UserCount()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	if count > 0 && userFromCtx(r.Context()) == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required to add more users"))
		return
	}
	var req struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		Password    string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON"))
		return
	}
	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "email and password are required"))
		return
	}
	if req.DisplayName == "" {
		req.DisplayName = req.Email
	}
	if err := s.cfg.Auth.CreateUser(req.Email, req.DisplayName, req.Password, false); err != nil {
		writeJSON(w, http.StatusConflict, apiError("conflict", err.Error()))
		return
	}
	user, _ := s.cfg.Auth.GetUser(req.Email)
	writeJSON(w, http.StatusCreated, map[string]any{"user": user})
}

// randomHex generates n random bytes and returns them as a lowercase hex string.
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
