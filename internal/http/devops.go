package http

import (
	"net/http"
	"path/filepath"

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
