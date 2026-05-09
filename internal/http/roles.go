// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"net/http"
)

// handleGetRoles returns the project's configured roles and user bindings.
func (s *Server) handleGetRoles(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	type userBinding struct {
		Email string   `json:"email"`
		Roles []string `json:"roles"`
	}

	users := make([]userBinding, 0, len(p.Cfg.Users))
	for _, u := range p.Cfg.Users {
		users = append(users, userBinding{Email: u.Email, Roles: u.Roles})
	}

	// Ensure roles is never null in JSON output.
	roles := p.Cfg.Roles
	if roles == nil {
		roles = []string{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"roles": roles,
		"users": users,
	})
}
