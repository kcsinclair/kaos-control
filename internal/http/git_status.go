// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"net/http"
)

// handleGetGitStatus handles GET /api/p/{project}/git/status.
// Returns a git status summary for the project, or {"available":false}
// when the project directory is not a git repository.
func (s *Server) handleGetGitStatus(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	if p.Git == nil {
		writeJSON(w, http.StatusOK, map[string]any{"available": false})
		return
	}

	summary, err := p.Git.Status()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("git_error", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"available":    true,
		"branch":       summary.Branch,
		"dirty":        summary.Dirty,
		"head_sha":     summary.HeadSHA,
		"head_message": summary.HeadMessage,
		"head_author":  summary.HeadAuthor,
		"head_when":    summary.HeadWhen,
	})
}
