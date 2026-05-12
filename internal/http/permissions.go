// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"net/http"
	"strings"

	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/project"
)

// Role name constants.
const (
	RoleProductOwner      = "product-owner"
	RoleAnalyst           = "analyst"
	RoleBackendDeveloper  = "backend-developer"
	RoleFrontendDeveloper = "frontend-developer"
	RoleTestDeveloper     = "test-developer"
	RoleQA                = "qa"
	RoleReviewer          = "reviewer"
	RoleApprover          = "approver"
	RoleDevops            = "devops"
)

// Role group variables derived from the permission matrix.
var (
	// RolesArtifactAuthors are the five authoring roles that may create artifacts.
	RolesArtifactAuthors = []string{RoleProductOwner, RoleAnalyst, RoleBackendDeveloper, RoleFrontendDeveloper, RoleTestDeveloper}
	// RolesArtifactEditors extends authors to include QA for update access.
	RolesArtifactEditors = []string{RoleProductOwner, RoleAnalyst, RoleBackendDeveloper, RoleFrontendDeveloper, RoleTestDeveloper, RoleQA}
	// RolesAdminOnly restricts access to the product-owner role only.
	RolesAdminOnly = []string{RoleProductOwner}
	// RolesDevopsOrAdmin allows product-owner and devops roles.
	RolesDevopsOrAdmin = []string{RoleProductOwner, RoleDevops}
	// RolesPriorityEditors may adjust artifact priority.
	RolesPriorityEditors = []string{RoleProductOwner, RoleAnalyst}
	// RolesReleaseEditors may assign or clear the release field on an artifact.
	RolesReleaseEditors = []string{RoleProductOwner, RoleAnalyst}
)

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

// requireRole checks that the authenticated user holds at least one of the
// allowed roles in the given project. It writes 401 or 403 and returns false
// if the check fails; returns true when the caller may proceed.
func requireRole(w http.ResponseWriter, r *http.Request, p *project.Project, allowed ...string) bool {
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return false
	}
	roles := p.Cfg.RolesFor(user.Email)
	if !hasAnyRole(roles, allowed...) {
		writeJSON(w, http.StatusForbidden, apiError("forbidden", "role required: "+strings.Join(allowed, ",")))
		return false
	}
	return true
}

// requireAppRole checks that the authenticated user holds at least one of the
// allowed roles across any configured project. Used for app-level endpoints
// that are mounted outside the /api/p/:project/ block.
func (s *Server) requireAppRole(w http.ResponseWriter, r *http.Request, allowed ...string) bool {
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return false
	}
	var union []string
	for _, p := range s.projects {
		union = append(union, p.Cfg.RolesFor(user.Email)...)
	}
	if !hasAnyRole(union, allowed...) {
		writeJSON(w, http.StatusForbidden, apiError("forbidden", "role required: "+strings.Join(allowed, ",")))
		return false
	}
	return true
}

// appUserHasRole reports whether user holds role in at least one configured project.
func (s *Server) appUserHasRole(user *auth.User, role string) bool {
	for _, p := range s.projects {
		for _, r := range p.Cfg.RolesFor(user.Email) {
			if r == role {
				return true
			}
		}
	}
	return false
}
