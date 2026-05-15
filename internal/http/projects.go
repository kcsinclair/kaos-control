// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
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
