// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kaos-control/kaos-control/internal/config"
	kgit "github.com/kaos-control/kaos-control/internal/git"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
	"github.com/kaos-control/kaos-control/internal/project"
	"github.com/kaos-control/kaos-control/internal/release"
	"github.com/kaos-control/kaos-control/internal/workflow"
)

func newTestProjectWithGit(t *testing.T) (*project.Project, func()) {
	t.Helper()
	dir := t.TempDir()

	// Initialise a real git repo
	gr, err := gogit.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}

	for _, sub := range []string{"ideas", "requirements", "releases"} {
		if err := os.MkdirAll(filepath.Join(dir, "lifecycle", sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Create initial commit so HEAD exists
	readmePath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test Repo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wt, err := gr.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Add("README.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Commit("initial commit", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	}); err != nil {
		t.Fatal(err)
	}

	gitRepo, err := kgit.Open(dir)
	if err != nil {
		t.Fatalf("kgit.Open: %v", err)
	}

	dataDir := t.TempDir()
	h := hub.New()
	wf := workflow.New(nil)
	idx, err := index.Open(filepath.Join(dataDir, "test.db"), dir, nil,
		index.WithHub(h),
		index.WithWorkflow(wf),
		index.WithGit(gitRepo),
	)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}

	cfg := &config.Project{
		Roles: []string{
			"product-owner", "analyst", "backend-developer",
			"frontend-developer", "test-developer", "qa",
			"reviewer", "approver", "devops", "system",
		},
		Users: []config.UserBinding{
			{Email: "admin@test", Roles: []string{"product-owner", "analyst", "reviewer", "approver"}},
		},
	}
	entry := &config.ProjectEntry{Name: "test", Path: dir}

	expected := release.NewExpectedEvents()
	releaseSync := release.NewDiskSync(expected)

	p := &project.Project{
		Entry:       entry,
		Cfg:         cfg,
		Idx:         idx,
		Hub:         h,
		Workflow:    wf,
		Git:         gitRepo,
		ReleaseSync: releaseSync,
	}

	return p, func() { idx.Close() }
}

func withChiReleaseID(r *http.Request, releaseID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("releaseID", releaseID)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestRelease_GitCommits(t *testing.T) {
	p, cleanup := newTestProjectWithGit(t)
	defer cleanup()
	s := &Server{}

	// 1. Create Release
	createBody := `{"name":"Alpha 1","status":"planned","start_date":"2026-09-01","end_date":"2026-09-30"}`
	req := httptest.NewRequest(http.MethodPost, "/api/p/test/releases", bytes.NewBufferString(createBody))
	req = withProjectAndUser(req, p, "admin@test")
	w := httptest.NewRecorder()

	s.handleCreateRelease(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("handleCreateRelease: want 201, got %d: %s", w.Code, w.Body.String())
	}

	var createdResp struct {
		Release *release.Release `json:"release"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &createdResp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	rel := createdResp.Release
	if rel == nil || rel.Slug != "alpha-1" {
		t.Fatalf("unexpected release: %+v", rel)
	}

	commits, err := p.Git.Log("lifecycle/releases/alpha-1.md", 5)
	if err != nil {
		t.Fatalf("Git.Log: %v", err)
	}
	if len(commits) == 0 {
		t.Fatalf("expected git commit for create, got none")
	}
	if !strings.Contains(commits[0].Message, "create(releases): lifecycle/releases/alpha-1.md") {
		t.Errorf("commit message: want containing %q, got %q", "create(releases): lifecycle/releases/alpha-1.md", commits[0].Message)
	}

	// 2. In-place Edit Release (status / dates)
	updateBody := `{"name":"Alpha 1","status":"active","start_date":"2026-09-01","end_date":"2026-09-25"}`
	req = httptest.NewRequest(http.MethodPut, "/api/p/test/releases/alpha-1", bytes.NewBufferString(updateBody))
	req = withProjectAndUser(req, p, "admin@test")
	req = withChiReleaseID(req, "alpha-1")
	w = httptest.NewRecorder()

	s.handleUpdateRelease(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("handleUpdateRelease in-place: want 200, got %d: %s", w.Code, w.Body.String())
	}

	commits, err = p.Git.Log("lifecycle/releases/alpha-1.md", 5)
	if err != nil {
		t.Fatalf("Git.Log: %v", err)
	}
	if len(commits) < 2 {
		t.Fatalf("expected at least 2 git commits after update, got %d", len(commits))
	}
	if !strings.Contains(commits[0].Message, "update: lifecycle/releases/alpha-1.md") {
		t.Errorf("commit message: want containing %q, got %q", "update: lifecycle/releases/alpha-1.md", commits[0].Message)
	}

	// 3. Rename Release
	renameBody := `{"name":"Alpha 2","status":"active","start_date":"2026-09-01","end_date":"2026-09-25"}`
	req = httptest.NewRequest(http.MethodPut, "/api/p/test/releases/alpha-1", bytes.NewBufferString(renameBody))
	req = withProjectAndUser(req, p, "admin@test")
	req = withChiReleaseID(req, "alpha-1")
	w = httptest.NewRecorder()

	s.handleUpdateRelease(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("handleUpdateRelease rename: want 200, got %d: %s", w.Code, w.Body.String())
	}

	commits, err = p.Git.Log("lifecycle/releases/alpha-2.md", 5)
	if err != nil {
		t.Fatalf("Git.Log: %v", err)
	}
	if len(commits) == 0 {
		t.Fatalf("expected git commit for renamed release, got none")
	}
	wantRenameMsg := `chore(releases): rename "Alpha 1" → "Alpha 2"`
	if !strings.Contains(commits[0].Message, wantRenameMsg) {
		t.Errorf("commit message: want containing %q, got %q", wantRenameMsg, commits[0].Message)
	}

	// 4. Delete Release
	req = httptest.NewRequest(http.MethodDelete, "/api/p/test/releases/alpha-2", nil)
	req = withProjectAndUser(req, p, "admin@test")
	req = withChiReleaseID(req, "alpha-2")
	w = httptest.NewRecorder()

	s.handleDeleteRelease(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("handleDeleteRelease: want 200, got %d: %s", w.Code, w.Body.String())
	}

	commits, err = p.Git.Log("lifecycle/releases/alpha-2.md", 5)
	if err != nil {
		t.Fatalf("Git.Log: %v", err)
	}
	if len(commits) < 2 {
		t.Fatalf("expected at least 2 commits touching alpha-2.md, got %d", len(commits))
	}
	if !strings.Contains(commits[0].Message, "delete: lifecycle/releases/alpha-2.md") {
		t.Errorf("delete commit message: want containing %q, got %q", "delete: lifecycle/releases/alpha-2.md", commits[0].Message)
	}

	// Status should be clean
	status, err := p.Git.Status()
	if err != nil {
		t.Fatalf("Git.Status: %v", err)
	}
	if status.Dirty {
		mods, _ := p.Git.ModifiedFiles(nil)
		t.Errorf("expected clean working tree after all release operations, got dirty: %v", mods)
	}
}
