// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/project"
)

// projectSummary is the JSON representation of a registered project.
type projectSummary struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description"`
	Owner       string `json:"owner"`
	Initialised bool   `json:"initialised"`
}

func entryToSummary(e *config.ProjectEntry) projectSummary {
	return projectSummary{
		Name:        e.Name,
		Path:        e.Path,
		Description: e.Description,
		Owner:       e.Owner,
		Initialised: config.IsInitialised(e.Path),
	}
}

func projectToSummary(p *project.Project) projectSummary {
	return entryToSummary(p.Entry)
}

// handleListProjects returns all registered projects.
func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	s.projectsMu.RLock()
	out := make([]projectSummary, 0, len(s.projects))
	for _, p := range s.projects {
		out = append(out, projectToSummary(p))
	}
	s.projectsMu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{"projects": out})
}

// handleGetProject returns a single project by name.
func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "project")
	p, ok := s.getProject(name)
	if !ok {
		writeJSON(w, http.StatusNotFound, apiError("project_not_found", "project not found: "+name))
		return
	}
	writeJSON(w, http.StatusOK, projectToSummary(p))
}

// handleCreateProject registers a new project and persists it to the registry.
func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		Path        string `json:"path"`
		Description string `json:"description"`
		Owner       string `json:"owner"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid_body", "invalid JSON: "+err.Error()))
		return
	}

	if err := config.ValidateProjectName(body.Name); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid_name", err.Error()))
		return
	}

	if _, exists := s.getProject(body.Name); exists {
		writeJSON(w, http.StatusConflict, apiError("conflict", "project already exists: "+body.Name))
		return
	}

	if body.Path == "" {
		writeJSON(w, http.StatusBadRequest, apiError("invalid_path", "path must not be empty"))
		return
	}
	resolved, err := config.ValidatePath(body.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid_path", err.Error()))
		return
	}

	entry := &config.ProjectEntry{
		Name:        body.Name,
		Path:        resolved,
		Description: body.Description,
		Owner:       body.Owner,
	}

	if err := config.SaveProjectEntry(s.projectsDir, entry); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("save_failed", "saving project entry: "+err.Error()))
		return
	}

	if err := s.RegisterProject(entry); err != nil {
		// Roll back: remove the saved YAML file since registration failed.
		_ = config.DeleteProjectEntry(s.projectsDir, entry.Name)
		writeJSON(w, http.StatusInternalServerError, apiError("register_failed", "registering project: "+err.Error()))
		return
	}

	p, _ := s.getProject(entry.Name)
	writeJSON(w, http.StatusCreated, projectToSummary(p))
}
