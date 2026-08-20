// SPDX-License-Identifier: AGPL-3.0-or-later

package http

// Onboarding: adding an EXISTING, already-initialised directory must register
// it as a project rather than erroring. Regression guard for the "…is already
// an initialised kaos-control project" dead end (POST /projects, mode=existing).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/initcmd"
	"github.com/kaos-control/kaos-control/internal/project"
)

// TestCreateProject_ExistingInitialisedRegisters verifies that pointing "add
// existing" at a directory that is already a kaos-control project registers it
// (HTTP 201, alreadyInitialised=true) instead of rejecting it.
func TestCreateProject_ExistingInitialisedRegisters(t *testing.T) {
	dir := t.TempDir()

	// Make the target a fully-initialised kaos-control project on disk.
	if _, err := initcmd.ScaffoldProject(initcmd.ScaffoldOptions{
		ProjectRoot: dir,
		ProjectName: "image-metadata1",
		OwnerEmail:  "owner@test",
	}); err != nil {
		t.Fatalf("ScaffoldProject: %v", err)
	}
	if !config.IsInitialised(dir) {
		t.Fatalf("precondition: %s should be initialised after scaffold", dir)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := &Server{
		projects:       make(map[string]*project.Project),
		projectCancels: make(map[string]context.CancelFunc),
		servCtx:        ctx,
		projectsDir:    t.TempDir(),
		dataDir:        t.TempDir(),
	}
	defer func() {
		if _, ok := s.getProject("image-metadata1"); ok {
			_ = s.UnregisterProject("image-metadata1")
		}
	}()

	body, _ := json.Marshal(map[string]string{
		"name": "image-metadata1",
		"mode": "existing",
		"path": dir,
	})
	r := httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(string(body)))
	r = r.WithContext(context.WithValue(r.Context(), userContextKey, &auth.User{Email: "owner@test"}))
	w := httptest.NewRecorder()

	s.handleCreateProject(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", w.Code, w.Body.String())
	}
	var res createProjectResult
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if !res.AlreadyInitialised {
		t.Errorf("alreadyInitialised = false, want true (dir was already a project)")
	}
	if _, ok := s.getProject("image-metadata1"); !ok {
		t.Errorf("project was not registered")
	}
}
