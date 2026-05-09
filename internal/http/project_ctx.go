// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/project"
)

type contextKey string

const projectKey contextKey = "project"

// projectMiddleware resolves the :project URL parameter and injects the
// project into the request context. Returns 404 if the project is unknown.
func (s *Server) projectMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "project")
		p, ok := s.projects[name]
		if !ok {
			writeJSON(w, http.StatusNotFound, apiError("project_not_found", "project not found: "+name))
			return
		}
		ctx := context.WithValue(r.Context(), projectKey, p)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func projectFromCtx(ctx context.Context) *project.Project {
	p, _ := ctx.Value(projectKey).(*project.Project)
	return p
}

func apiError(code, message string) map[string]any {
	return map[string]any{"error": map[string]any{"code": code, "message": message}}
}
