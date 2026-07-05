// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kaos-control/kaos-control/internal/artifact"
)

func TestHandleGetOpenQuestions_ReturnsParsedQuestions(t *testing.T) {
	p, cleanup := newTestProject(t)
	defer cleanup()

	content := "---\ntitle: Test OQ\ntype: idea\nstatus: blocked\nlineage: oq-test\n---\n\n" +
		"## Open Questions\n\n- Q1?\n\n> A1.\n\n- Q2?\n"
	relPath := "lifecycle/ideas/oq-test.md"
	absPath := filepath.Join(p.Entry.Path, relPath)
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := p.Idx.IndexFile(absPath); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withProjectAndUser(req, p, "po@test")
	req = withChiWildcard(req, relPath+"/open-questions")

	w := httptest.NewRecorder()
	s.handleGetOpenQuestions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Heading   string `json:"heading"`
		Format    string `json:"format"`
		Questions []struct {
			Index  int    `json:"index"`
			Text   string `json:"text"`
			Answer string `json:"answer"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Heading != "## Open Questions" {
		t.Errorf("heading = %q", resp.Heading)
	}
	if resp.Format != "blockquote" {
		t.Errorf("format = %q, want blockquote", resp.Format)
	}
	if len(resp.Questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(resp.Questions))
	}
	if resp.Questions[0].Answer != "A1." {
		t.Errorf("questions[0].Answer = %q", resp.Questions[0].Answer)
	}
	if resp.Questions[1].Answer != "" {
		t.Errorf("questions[1].Answer = %q, want empty", resp.Questions[1].Answer)
	}
}

func TestHandleGetOpenQuestions_NoSectionReturnsEmptyList(t *testing.T) {
	p, cleanup := newTestProject(t)
	defer cleanup()

	relPath := writeTestArtifactHTTP(t, p, "no-oq", "draft")

	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withProjectAndUser(req, p, "po@test")
	req = withChiWildcard(req, relPath+"/open-questions")

	w := httptest.NewRecorder()
	s.handleGetOpenQuestions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Questions []any `json:"questions"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Questions == nil {
		t.Error("expected questions to be an empty array, got null")
	}
	if len(resp.Questions) != 0 {
		t.Errorf("expected 0 questions, got %d", len(resp.Questions))
	}
}

func TestHandleGetOpenQuestions_NotFound(t *testing.T) {
	p, cleanup := newTestProject(t)
	defer cleanup()

	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withProjectAndUser(req, p, "po@test")
	req = withChiWildcard(req, "lifecycle/ideas/does-not-exist.md/open-questions")

	w := httptest.NewRecorder()
	s.handleGetOpenQuestions(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestHandlePreviewOpenQuestions_PartialDoesNotWriteToDisk(t *testing.T) {
	p, cleanup := newTestProject(t)
	defer cleanup()

	content := "---\ntitle: Test OQ Preview\ntype: idea\nstatus: blocked\nlineage: oq-preview\n---\n\n" +
		"## Open Questions\n\n- Q1?\n\n- Q2?\n"
	relPath := "lifecycle/ideas/oq-preview.md"
	absPath := filepath.Join(p.Entry.Path, relPath)
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := p.Idx.IndexFile(absPath); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	before, err := os.Stat(absPath)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	reqBody, _ := json.Marshal(map[string]any{
		"answers":  map[string]string{"0": "A1."},
		"complete": false,
	})

	s := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(reqBody))
	req = withProjectAndUser(req, p, "po@test")
	req = withChiWildcard(req, relPath+"/open-questions/preview")

	w := httptest.NewRecorder()
	s.handlePreviewOpenQuestions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	after, err := os.Stat(absPath)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !before.ModTime().Equal(after.ModTime()) {
		t.Errorf("preview endpoint modified the file on disk: before=%v after=%v", before.ModTime(), after.ModTime())
	}

	var resp struct {
		Body string `json:"body"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	qs, ok := artifact.ParseOpenQuestions(resp.Body)
	if !ok || len(qs) != 2 {
		t.Fatalf("expected 2 questions in previewed body, got ok=%v qs=%v", ok, qs)
	}
	if qs[0].Answer != "A1." {
		t.Errorf("q0.Answer = %q, want %q", qs[0].Answer, "A1.")
	}
}

func TestHandlePreviewOpenQuestions_CompleteWithMissingAnswerReturns422(t *testing.T) {
	p, cleanup := newTestProject(t)
	defer cleanup()

	content := "---\ntitle: Test OQ Preview 2\ntype: idea\nstatus: blocked\nlineage: oq-preview-2\n---\n\n" +
		"## Open Questions\n\n- Q1?\n\n- Q2?\n"
	relPath := "lifecycle/ideas/oq-preview-2.md"
	absPath := filepath.Join(p.Entry.Path, relPath)
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := p.Idx.IndexFile(absPath); err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	reqBody, _ := json.Marshal(map[string]any{
		"answers":  map[string]string{"0": "A1."},
		"complete": true,
	})

	s := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(reqBody))
	req = withProjectAndUser(req, p, "po@test")
	req = withChiWildcard(req, relPath+"/open-questions/preview")

	w := httptest.NewRecorder()
	s.handlePreviewOpenQuestions(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d; body: %s", w.Code, w.Body.String())
	}
}
