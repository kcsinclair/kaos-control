// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"
)

// stubSPAFrontend returns a minimal fs.FS that the test server can use to serve
// the SPA catch-all route.  It contains only "dist/index.html" with enough HTML
// to satisfy isSPAResponse.
func stubSPAFrontend() fstest.MapFS {
	return fstest.MapFS{
		"dist/index.html": &fstest.MapFile{
			Data: []byte(`<!DOCTYPE html><html><head><title>Test SPA</title></head><body><div id="app"></div></body></html>`),
		},
	}
}

// isSPAResponse returns true when the response body looks like the SPA HTML shell
// (contains a <html> tag), which confirms the catch-all frontend handler served it.
func isSPAResponse(t *testing.T, resp *http.Response) bool {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}
	body := strings.ToLower(string(b))
	return strings.Contains(body, "<html")
}

// TestKanbanRouting_BoardRouteServesSPA verifies that GET /p/:project/artifacts/board
// returns 200 with the SPA HTML shell, confirming the new board route is not
// accidentally 404-ing at the server level.
// Covers Milestone 3, scenario 1.
func TestKanbanRouting_BoardRouteServesSPA(t *testing.T) {
	env := newTestEnvWithFrontend(t, nil, stubSPAFrontend())

	resp, err := http.Get(env.baseURL + "/p/testproject/artifacts/board")
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, resp, 200)

	if !isSPAResponse(t, resp) {
		t.Error("expected SPA HTML for /p/:project/artifacts/board, got non-HTML response")
	}
}

// TestKanbanRouting_ListRouteUnchanged verifies that the pre-existing
// GET /p/:project/artifacts route still returns 200 with the SPA HTML.
// Covers Milestone 3, scenario 2.
func TestKanbanRouting_ListRouteUnchanged(t *testing.T) {
	env := newTestEnvWithFrontend(t, nil, stubSPAFrontend())

	resp, err := http.Get(env.baseURL + "/p/testproject/artifacts")
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, resp, 200)

	if !isSPAResponse(t, resp) {
		t.Error("expected SPA HTML for /p/:project/artifacts, got non-HTML response")
	}
}

// TestKanbanRouting_ArtifactEditorRouteUnchanged verifies that the artifact
// editor route /p/:project/artifacts/<path> still serves the SPA and is not
// intercepted by any new board route pattern.
// Covers Milestone 3, scenario 3.
func TestKanbanRouting_ArtifactEditorRouteUnchanged(t *testing.T) {
	env := newTestEnvWithFrontend(t, nil, stubSPAFrontend())

	resp, err := http.Get(env.baseURL + "/p/testproject/artifacts/requirements/kanban-view-3.md")
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, resp, 200)

	if !isSPAResponse(t, resp) {
		t.Error("expected SPA HTML for artifact editor route, got non-HTML response")
	}
}

// TestKanbanRouting_KanbanConfigRequiresAuth verifies that
// GET /api/p/:project/config/kanban without an authenticated session returns 401.
// Covers Milestone 3, scenario 4.
func TestKanbanRouting_KanbanConfigRequiresAuth(t *testing.T) {
	env := newTestEnv(t, nil)
	// Deliberately do NOT call env.login — no session cookie.

	resp, err := http.Get(env.baseURL + "/api/p/testproject/config/kanban")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		b, _ := io.ReadAll(resp.Body)
		t.Errorf("expected 401 for unauthenticated request to /config/kanban, got %d: %s",
			resp.StatusCode, string(b))
	}
}
