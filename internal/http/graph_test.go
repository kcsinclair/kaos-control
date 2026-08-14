// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/project"
)

// writeTestArchitectureHTTP writes a minimal type: architecture fixture to
// disk under lifecycle/architecture/architectures/ and indexes it. Returns
// the project-relative path (== node id).
func writeTestArchitectureHTTP(t *testing.T, p *project.Project, slug, body string) string {
	t.Helper()
	relPath := "lifecycle/architecture/architectures/" + slug + ".md"
	absPath := filepath.Join(p.Entry.Path, relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\ntitle: " + slug + "\ntype: architecture\nstatus: approved\n---\n\n" + body + "\n"
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := p.Idx.IndexFile(absPath); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}
	return relPath
}

// TestHandleArchitectureMap_NoProject verifies the same nil-project error
// contract as handleGraph: a request with no project in context returns 500
// with the no_project error code, never a panic.
func TestHandleArchitectureMap_NoProject(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/p/nope/architecture-map", nil)
	w := httptest.NewRecorder()

	s.handleArchitectureMap(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d; body: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	errObj, _ := body["error"].(map[string]any)
	if errObj["code"] != "no_project" {
		t.Errorf("error.code: want %q, got %v", "no_project", body["error"])
	}
}

// TestHandleArchitectureMap_BaseMap verifies GET returns 200 with a
// { nodes, edges } payload scoped to architecture nodes, and that omitting
// stack_for does not add tech-stack nodes.
func TestHandleArchitectureMap_BaseMap(t *testing.T) {
	p, cleanup := newTestProject(t)
	defer cleanup()

	writeTestArchitectureHTTP(t, p, "foo", "See [[bar]].")
	writeTestArchitectureHTTP(t, p, "bar", "Body.")

	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/p/test/architecture-map", nil)
	req = withProjectAndUser(req, p, "po@test")
	w := httptest.NewRecorder()

	s.handleArchitectureMap(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	nodes, ok := body["nodes"].([]any)
	if !ok || len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got: %v", body["nodes"])
	}
	edges, ok := body["edges"].([]any)
	if !ok || len(edges) != 1 {
		t.Fatalf("expected 1 edge, got: %v", body["edges"])
	}
	edge := edges[0].(map[string]any)
	if edge["kind"] != "related" {
		t.Errorf("edge.kind: want %q, got %v", "related", edge["kind"])
	}
}

// TestHandleArchitectureMap_StackForQueryParam verifies that ?stack_for=<id>
// is read from the query string and forwarded to the read model, adding the
// named architecture's related_to tech-stack ring.
func TestHandleArchitectureMap_StackForQueryParam(t *testing.T) {
	p, cleanup := newTestProject(t)
	defer cleanup()

	fooID := writeTestArchitectureHTTP(t, p, "foo", "Body.")
	stackRelPath := "lifecycle/architecture/tech-stacks/go-postgres.md"
	stackAbsPath := filepath.Join(p.Entry.Path, stackRelPath)
	if err := os.MkdirAll(filepath.Dir(stackAbsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stackAbsPath,
		[]byte("---\ntitle: Go + Postgres\ntype: tech-stack\nstatus: approved\n---\n\nBody.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := p.Idx.IndexFile(stackAbsPath); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}
	// Re-write foo with a related_to pointing at the stack now that it's indexed.
	fooAbsPath := filepath.Join(p.Entry.Path, fooID)
	if err := os.WriteFile(fooAbsPath,
		[]byte("---\ntitle: foo\ntype: architecture\nstatus: approved\nrelated_to:\n  - architecture/tech-stacks/go-postgres.md\n---\n\nBody.\n"),
		0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := p.Idx.IndexFile(fooAbsPath); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/p/test/architecture-map?stack_for="+fooID, nil)
	req = withProjectAndUser(req, p, "po@test")
	w := httptest.NewRecorder()

	s.handleArchitectureMap(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	nodes, _ := body["nodes"].([]any)
	found := false
	for _, n := range nodes {
		node := n.(map[string]any)
		if node["id"] == stackRelPath {
			found = true
			if node["type"] != "tech-stack" {
				t.Errorf("stack node type: want tech-stack, got %v", node["type"])
			}
		}
	}
	if !found {
		t.Errorf("expected stack node %q in payload, got nodes: %v", stackRelPath, nodes)
	}
}

// TestArchitectureMapRoute_ReadOnly verifies at the router level that only
// GET is registered for /architecture-map — no POST/PUT/DELETE/PATCH variant
// exists (FR-10). Uses chi's route walk rather than a live request so no
// session/auth setup is required.
func TestArchitectureMapRoute_ReadOnly(t *testing.T) {
	s := &Server{}
	router := s.buildRouter()

	methods := map[string]bool{}
	if err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if route == "/api/p/{project}/architecture-map" {
			methods[method] = true
		}
		return nil
	}); err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}

	if !methods[http.MethodGet] {
		t.Errorf("expected GET registered for /architecture-map, got methods: %v", methods)
	}
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		if methods[m] {
			t.Errorf("unexpected %s registered for /architecture-map", m)
		}
	}
}
