// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"io"
	"net/http"
	"testing"
	"testing/fstest"
)

// stubFrontendFS returns a minimal in-memory FS that satisfies the server's
// catch-all handler. The handler calls fs.Sub(frontend, "dist") and then opens
// "index.html", so the stub must contain "dist/index.html".
func stubFrontendFS() fstest.MapFS {
	return fstest.MapFS{
		"dist/index.html": &fstest.MapFile{
			Data: []byte(`<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"/></head><body><div id="app"></div></body></html>`),
		},
	}
}

// ── Milestone 6: Navigation & Default Route ───────────────────────────────────

// TestDashboardE2E_ProjectRouteReturns200 verifies that GET /p/:project
// returns HTTP 200 with the SPA shell.
//
// Background: the router's catch-all handler (handleFrontend) serves
// index.html for every unknown path when a frontend FS is configured.
// The Vue Router redirect from /p/:project → /p/:project/dashboard is
// client-side only; the Go server never emits a 302. This test validates
// the server-side behaviour (option a from Q1 resolution).
//
// HTML content assertions (widget containers, sidebar nav items) require
// JavaScript execution in a real browser and are covered by a Playwright
// E2E suite (option b from Q2 resolution).
func TestDashboardE2E_ProjectRouteReturns200(t *testing.T) {
	env := newTestEnvWithFrontend(t, nil, stubFrontendFS())
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/p/testproject", nil)
	defer resp.Body.Close()

	requireStatus(t, resp, 200)
}

// TestDashboardE2E_DashboardSubRouteReturns200 verifies that the explicit
// /p/:project/dashboard path also returns HTTP 200 (same catch-all handler).
func TestDashboardE2E_DashboardSubRouteReturns200(t *testing.T) {
	env := newTestEnvWithFrontend(t, nil, stubFrontendFS())
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/p/testproject/dashboard", nil)
	defer resp.Body.Close()

	requireStatus(t, resp, 200)
}

// TestDashboardE2E_ResponseBodyContainsSPAShell verifies that the 200 response
// body contains the SPA mount point (<div id="app">), confirming index.html
// content is served and not an empty or error response.
func TestDashboardE2E_ResponseBodyContainsSPAShell(t *testing.T) {
	env := newTestEnvWithFrontend(t, nil, stubFrontendFS())
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/p/testproject/dashboard", nil)
	requireStatus(t, resp, 200)

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}

	if bodyStr := string(body); len(bodyStr) == 0 {
		t.Error("expected non-empty body for dashboard route")
	}
}

// TestDashboardE2E_FrontendUnavailableReturns500 verifies that without a
// frontend FS the catch-all handler returns HTTP 500 (not a redirect or 404).
// This documents the expected failure mode when the binary is built without
// an embedded frontend (e.g. in unit-test builds that pass nil).
func TestDashboardE2E_FrontendUnavailableReturns500(t *testing.T) {
	// newTestEnv passes nil for the frontend FS.
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Use a custom client that does NOT follow redirects so we observe the
	// exact status the server emits.
	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest(http.MethodGet, env.baseURL+"/p/testproject", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range env.cookies {
		req.AddCookie(c)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 when frontend FS is nil, got %d", resp.StatusCode)
	}
}

// TestDashboardE2E_ArbitrarySubRouteReturns200 verifies that any unrecognised
// path under /p/:project falls back to index.html (HTML5 pushState support).
func TestDashboardE2E_ArbitrarySubRouteReturns200(t *testing.T) {
	env := newTestEnvWithFrontend(t, nil, stubFrontendFS())
	env.login("admin@test.local", "admin-pass-123")

	for _, path := range []string{
		"/p/testproject/artifacts",
		"/p/testproject/graph",
		"/p/testproject/agents",
	} {
		resp := env.doRequest("GET", path, nil)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s: expected 200, got %d", path, resp.StatusCode)
		}
	}
}
