// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/devops"
)

var pipelineSlugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]*[a-z0-9]$|^[a-z0-9]$`)

// devopsDir returns the absolute path to the lifecycle/devops/ directory for
// the given project root.
func devopsDir(projectRoot string) string {
	return filepath.Join(projectRoot, "lifecycle", "devops")
}

// handleListPipelines handles GET /api/p/{project}/devops/pipelines.
// It discovers all valid pipeline YAML files in lifecycle/devops/ and returns
// them grouped by type. Access is restricted to product-owner and devops roles.
func (s *Server) handleListPipelines(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if !requireRole(w, r, p, RolesDevopsOrAdmin...) {
		return
	}

	dir := devopsDir(p.Entry.Path)
	pipelines, _ := devops.Discover(dir)

	type stepOut struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
	}
	type pipelineOut struct {
		Slug      string    `json:"slug"`
		Name      string    `json:"name"`
		Type      string    `json:"type"`
		StepCount int       `json:"step_count"`
		Steps     []stepOut `json:"steps"`
	}

	out := make([]pipelineOut, 0, len(pipelines))
	for _, pl := range pipelines {
		steps := make([]stepOut, len(pl.Steps))
		for i, st := range pl.Steps {
			steps[i] = stepOut{Name: st.Name, Description: st.Description}
		}
		out = append(out, pipelineOut{
			Slug:      pl.Slug,
			Name:      pl.Name,
			Type:      pl.Type,
			StepCount: len(pl.Steps),
			Steps:     steps,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"pipelines": out})
}

// handleRunPipeline handles POST /api/p/{project}/devops/pipelines/{slug}/run.
// Validates the role, checks the pipeline is not already running, starts
// execution, and returns a run_id.
func (s *Server) handleRunPipeline(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if !requireRole(w, r, p, RolesDevopsOrAdmin...) {
		return
	}

	slug := chi.URLParam(r, "slug")

	dir := devopsDir(p.Entry.Path)
	pipelines, _ := devops.Discover(dir)

	var found *devops.Pipeline
	for i := range pipelines {
		if pipelines[i].Slug == slug {
			found = &pipelines[i]
			break
		}
	}
	if found == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "pipeline not found: "+slug))
		return
	}

	if p.DevopsRunner.IsRunning(slug) {
		writeJSON(w, http.StatusConflict, apiError("conflict", "pipeline is already running: "+slug))
		return
	}

	runID, err := p.DevopsRunner.Start(*found, p.Entry.Path, p.Hub, p.Entry.Name)
	if err != nil {
		writeJSON(w, http.StatusConflict, apiError("conflict", err.Error()))
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{"run_id": runID})
}

// handleCancelPipeline handles POST /api/p/{project}/devops/pipelines/{slug}/cancel.
// Cancels the active run for the given pipeline slug, or returns 404 if none.
func (s *Server) handleCancelPipeline(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if !requireRole(w, r, p, RolesDevopsOrAdmin...) {
		return
	}

	slug := chi.URLParam(r, "slug")
	runID, ok := p.DevopsRunner.ActiveRunID(slug)
	if !ok {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "no active run for pipeline: "+slug))
		return
	}

	if err := p.DevopsRunner.Cancel(runID); err != nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"cancelled": true})
}

// handleGetRunLog handles GET /api/p/{project}/devops/runs/{run_id}.
// Returns the JSON-lines log for a completed or in-progress run.
func (s *Server) handleGetRunLog(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if !requireRole(w, r, p, RolesDevopsOrAdmin...) {
		return
	}

	runID := chi.URLParam(r, "run_id")
	data, err := p.DevopsLogs.ReadLogNDJSON(p.Entry.Name, runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "run log not found: "+runID))
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// handleGetPipeline handles GET /api/p/{project}/devops/pipelines/{slug}.
// It returns the raw YAML content of the named pipeline file so the frontend
// can populate the editor. Access is restricted to product-owner and devops roles.
func (s *Server) handleGetPipeline(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if !requireRole(w, r, p, RolesDevopsOrAdmin...) {
		return
	}

	slug := chi.URLParam(r, "slug")
	if !pipelineSlugRe.MatchString(slug) {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid pipeline slug"))
		return
	}

	path := filepath.Join(devopsDir(p.Entry.Path), slug+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, apiError("not_found", "pipeline not found: "+slug))
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	w.Header().Set("Content-Type", "text/yaml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// handleUpdatePipeline handles PUT /api/p/{project}/devops/pipelines/{slug}.
// It validates and atomically persists an updated YAML pipeline definition.
// Returns 409 if the pipeline slug is currently running, 400 for invalid YAML,
// 404 if the pipeline does not exist, and 200 with a summary on success.
func (s *Server) handleUpdatePipeline(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if !requireRole(w, r, p, RolesDevopsOrAdmin...) {
		return
	}

	slug := chi.URLParam(r, "slug")
	if !pipelineSlugRe.MatchString(slug) {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid pipeline slug"))
		return
	}

	var req struct {
		Definition string `json:"definition"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}

	dir := devopsDir(p.Entry.Path)
	destPath := filepath.Join(dir, slug+".yaml")
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "pipeline not found: "+slug))
		return
	}

	// Active-run guard (first check, before validation).
	if p.DevopsRunner.IsRunning(slug) {
		writeJSON(w, http.StatusConflict, apiError("conflict", "cannot edit pipeline while it is running: "+slug))
		return
	}

	// Validate the YAML definition.
	pl, err := devops.ValidateDefinition([]byte(req.Definition))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid pipeline definition: "+err.Error()))
		return
	}
	pl.Slug = slug

	// Second active-run check immediately before write to close the TOCTOU window.
	if p.DevopsRunner.IsRunning(slug) {
		writeJSON(w, http.StatusConflict, apiError("conflict", "cannot edit pipeline while it is running: "+slug))
		return
	}

	// Atomic write: write to a temp file then rename.
	tmp, err := os.CreateTemp(dir, slug+".yaml.tmp*")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}
	tmpPath := tmp.Name()
	// Deferred removal is a no-op after a successful rename.
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmp.Write([]byte(req.Definition)); err != nil {
		_ = tmp.Close()
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}
	if err := tmp.Close(); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"slug":       pl.Slug,
		"name":       pl.Name,
		"type":       pl.Type,
		"step_count": len(pl.Steps),
	})
}

// handleCreatePipeline handles POST /api/p/{project}/devops/pipelines.
// It validates the slug and YAML definition, rejects duplicates, and writes
// the new pipeline file to devops/{slug}.yaml under the project root.
func (s *Server) handleCreatePipeline(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if !requireRole(w, r, p, RolesDevopsOrAdmin...) {
		return
	}

	var req struct {
		Slug       string `json:"slug"`
		Definition string `json:"definition"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}

	if !pipelineSlugRe.MatchString(req.Slug) {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "slug must be lowercase alphanumeric with hyphens"))
		return
	}

	destPath := filepath.Join(devopsDir(p.Entry.Path), req.Slug+".yaml")
	if _, err := os.Stat(destPath); err == nil {
		writeJSON(w, http.StatusConflict, apiError("conflict", "pipeline already exists: "+req.Slug))
		return
	}

	pl, err := devops.ValidateDefinition([]byte(req.Definition))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid pipeline definition: "+err.Error()))
		return
	}
	pl.Slug = req.Slug

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}
	if err := os.WriteFile(destPath, []byte(req.Definition), 0o644); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"slug":       pl.Slug,
		"name":       pl.Name,
		"type":       pl.Type,
		"step_count": len(pl.Steps),
	})
}
