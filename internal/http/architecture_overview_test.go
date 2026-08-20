// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestHandleArchitectureOverview_NoProject verifies the same nil-project
// error contract as handleArchitectureMap: a request with no project in
// context returns 500 with the no_project error code, never a panic.
func TestHandleArchitectureOverview_NoProject(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/p/nope/architecture/overview", nil)
	w := httptest.NewRecorder()

	s.handleArchitectureOverview(w, req)

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

// TestHandleArchitectureOverview_PopulatedProject verifies 200 with the
// classified model for a project with a promoted architecture + stack, and
// that a reader with no editor role still receives it (read-mostly, NFR-2).
// It also asserts the handler makes no artifact-content write and leaves the
// index unchanged.
func TestHandleArchitectureOverview_PopulatedProject(t *testing.T) {
	p, cleanup := newTestProject(t)
	defer cleanup()

	writeTestArchitectureHTTP(t, p, "modular-monolith", "Body.")
	rootRelPath := "lifecycle/architecture/modular-monolith.md"
	rootAbsPath := filepath.Join(p.Entry.Path, rootRelPath)
	if err := os.WriteFile(rootAbsPath,
		[]byte("---\ntitle: Modular Monolith\ntype: architecture\nstatus: approved\n---\n\nBody.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := p.Idx.IndexFile(rootAbsPath); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	before := listArchitectureFiles(t, p.Entry.Path)

	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/p/test/architecture/overview", nil)
	// analyst@test carries no editor role in newTestProject's fixture config —
	// a non-editor reader must still receive the model.
	req = withProjectAndUser(req, p, "analyst@test")
	w := httptest.NewRecorder()

	s.handleArchitectureOverview(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["has_chosen_architecture"] != true {
		t.Errorf("has_chosen_architecture: want true, got %v", body["has_chosen_architecture"])
	}
	chosen, ok := body["chosen_architecture"].(map[string]any)
	if !ok {
		t.Fatalf("chosen_architecture: want object, got %v", body["chosen_architecture"])
	}
	if chosen["catalog_role"] != "chosen-architecture" {
		t.Errorf("chosen_architecture.catalog_role: want %q, got %v", "chosen-architecture", chosen["catalog_role"])
	}
	catalog, ok := body["catalog"].([]any)
	if !ok || len(catalog) != 1 {
		t.Fatalf("catalog: want 1 item, got %v", body["catalog"])
	}
	item := catalog[0].(map[string]any)
	if item["catalog_role"] != "catalog" {
		t.Errorf("catalog[0].catalog_role: want %q, got %v", "catalog", item["catalog_role"])
	}

	after := listArchitectureFiles(t, p.Entry.Path)
	if len(before) != len(after) {
		t.Errorf("architecture zone file count changed: before=%d after=%d", len(before), len(after))
	}
	for name, content := range before {
		if after[name] != content {
			t.Errorf("file %q changed by a read-only request", name)
		}
	}
}

// TestHandleArchitectureOverview_EmptyProject verifies 200 with a degraded
// (empty/null) model when nothing has been promoted — never a 500.
func TestHandleArchitectureOverview_EmptyProject(t *testing.T) {
	p, cleanup := newTestProject(t)
	defer cleanup()

	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/p/test/architecture/overview", nil)
	req = withProjectAndUser(req, p, "po@test")
	w := httptest.NewRecorder()

	s.handleArchitectureOverview(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["has_chosen_architecture"] != false {
		t.Errorf("has_chosen_architecture: want false, got %v", body["has_chosen_architecture"])
	}
	if body["chosen_architecture"] != nil {
		t.Errorf("chosen_architecture: want null, got %v", body["chosen_architecture"])
	}
	if body["summary"] != nil {
		t.Errorf("summary: want null, got %v", body["summary"])
	}
	standards, ok := body["standards"].([]any)
	if !ok || len(standards) != 0 {
		t.Errorf("standards: want empty array, got %v", body["standards"])
	}
	adrs, ok := body["adrs"].([]any)
	if !ok || len(adrs) != 0 {
		t.Errorf("adrs: want empty array, got %v", body["adrs"])
	}
}

// TestHandleArchitectureOverview_DiskChangeReflectedOnNextCall pins FR-12 at
// the endpoint: a standard written directly to disk between two GETs, with
// no reindex call in between, appears in the second response. This is the
// disk-fallback half of the freshness contract; TestWatcher_ArchitectureSubdirsEmitFileChanged
// (internal/watcher) pins the live fsnotify half the frontend relies on to
// know when to re-fetch.
func TestHandleArchitectureOverview_DiskChangeReflectedOnNextCall(t *testing.T) {
	p, cleanup := newTestProject(t)
	defer cleanup()

	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/p/test/architecture/overview", nil)
	req = withProjectAndUser(req, p, "po@test")
	w := httptest.NewRecorder()
	s.handleArchitectureOverview(w, req)

	var before map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &before); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if standards, _ := before["standards"].([]any); len(standards) != 0 {
		t.Fatalf("expected no standards before the disk write, got %v", standards)
	}

	// Write a standard directly to disk — no p.Idx.IndexFile call, simulating
	// an external edit the watcher hasn't (yet) caught up with.
	standardPath := filepath.Join(p.Entry.Path, "lifecycle/architecture/standards/secrets.md")
	if err := os.MkdirAll(filepath.Dir(standardPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(standardPath,
		[]byte("---\ntitle: Secrets Handling\ntype: doc\nstatus: approved\n---\n\nBody.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/p/test/architecture/overview", nil)
	req2 = withProjectAndUser(req2, p, "po@test")
	w2 := httptest.NewRecorder()
	s.handleArchitectureOverview(w2, req2)

	var after map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &after); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	standards, ok := after["standards"].([]any)
	if !ok || len(standards) != 1 {
		t.Fatalf("expected the unindexed standard to appear, got %v", after["standards"])
	}
	item := standards[0].(map[string]any)
	if item["title"] != "Secrets Handling" {
		t.Errorf("standards[0].title: got %v", item["title"])
	}
}

// listArchitectureFiles returns the content of every .md file under
// lifecycle/architecture/, keyed by repo-relative path.
func listArchitectureFiles(t *testing.T, projectRoot string) map[string]string {
	t.Helper()
	root := filepath.Join(projectRoot, "lifecycle", "architecture")
	out := map[string]string{}
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			t.Fatalf("ReadFile: %v", rerr)
		}
		rel, rerr := filepath.Rel(projectRoot, path)
		if rerr != nil {
			t.Fatalf("Rel: %v", rerr)
		}
		out[filepath.ToSlash(rel)] = string(raw)
		return nil
	})
	return out
}
