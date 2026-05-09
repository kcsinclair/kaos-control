// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/devops"
)

// devopsDir returns the absolute path to the lifecycle/devops/ directory for
// the given project root.
func devopsDir(projectRoot string) string {
	return filepath.Join(projectRoot, "lifecycle", "devops")
}

// hasAnyRole reports whether userRoles contains at least one of the allowed roles.
func hasAnyRole(userRoles []string, allowed ...string) bool {
	for _, ur := range userRoles {
		for _, a := range allowed {
			if ur == a {
				return true
			}
		}
	}
	return false
}

// handleListPipelines handles GET /api/p/{project}/devops/pipelines.
// It discovers all valid pipeline YAML files in lifecycle/devops/ and returns
// them grouped by type. Access is restricted to product-owner and devops roles.
func (s *Server) handleListPipelines(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	roles := p.Cfg.RolesFor(user.Email)
	if !hasAnyRole(roles, "product-owner", "devops") {
		writeJSON(w, http.StatusForbidden, apiError("forbidden", "product-owner or devops role required"))
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
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	roles := p.Cfg.RolesFor(user.Email)
	if !hasAnyRole(roles, "product-owner", "devops") {
		writeJSON(w, http.StatusForbidden, apiError("forbidden", "product-owner or devops role required"))
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
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	roles := p.Cfg.RolesFor(user.Email)
	if !hasAnyRole(roles, "product-owner", "devops") {
		writeJSON(w, http.StatusForbidden, apiError("forbidden", "product-owner or devops role required"))
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
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	roles := p.Cfg.RolesFor(user.Email)
	if !hasAnyRole(roles, "product-owner", "devops") {
		writeJSON(w, http.StatusForbidden, apiError("forbidden", "product-owner or devops role required"))
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
