package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kaos-control/kaos-control/internal/statuscheck"
)

// handleStatusCheck handles GET /api/p/{project}/status-check
//
// Optional query parameter:
//
//	?lineage=<slug>  — check a single lineage; when omitted, check all.
//
// Response:
//
//	{"stale": [<StatusResult>, ...]}
//
// Requires an authenticated user (roles are needed to determine can_advance).
func (s *Server) handleStatusCheck(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	lineageSlug := r.URL.Query().Get("lineage")
	userRoles := p.Cfg.RolesFor(user.Email)

	var allResults []statuscheck.Result

	if lineageSlug != "" {
		// Single-lineage check.
		artifacts, err := p.Idx.ListByLineage(lineageSlug)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
			return
		}
		allResults = statuscheck.Check(artifacts)
	} else {
		// Project-wide check: run algorithm per lineage.
		grouped, err := p.Idx.ListAllGroupedByLineage()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
			return
		}
		for _, artifacts := range grouped {
			allResults = append(allResults, statuscheck.Check(artifacts)...)
		}
	}

	// Annotate each result with can_advance / blocked_reason.
	for i := range allResults {
		r := &allResults[i]
		if p.Workflow.CanTransition(r.CurrentStatus, r.SuggestedStatus, userRoles) {
			r.CanAdvance = true
		} else {
			r.CanAdvance = false
			r.BlockedReason = fmt.Sprintf("requires role with permission to transition %q → %q",
				r.CurrentStatus, r.SuggestedStatus)
		}
	}

	if allResults == nil {
		allResults = []statuscheck.Result{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"stale": allResults})
}

// handleStatusCheckAdvance handles POST /api/p/{project}/status-check/advance
//
// Request body:
//
//	{"paths": ["lifecycle/ideas/foo.md", ...]}
//
// Each artifact is re-evaluated against the staleness algorithm at execution
// time. Transitions are applied sequentially. Artifacts that are already
// current, or whose transition is blocked, are reported without modification.
//
// Response:
//
//	{"results": [{"path": "...", "advanced_to": "planning", "ok": true}, ...]}
func (s *Server) handleStatusCheckAdvance(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	var req struct {
		Paths []string `json:"paths"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}
	if len(req.Paths) == 0 {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "field 'paths' must be a non-empty array"))
		return
	}

	userRoles := p.Cfg.RolesFor(user.Email)

	type advanceResult struct {
		Path       string `json:"path"`
		AdvancedTo string `json:"advanced_to,omitempty"`
		Ok         bool   `json:"ok"`
		Error      string `json:"error,omitempty"`
	}

	results := make([]advanceResult, 0, len(req.Paths))

	for _, relPath := range req.Paths {
		// Re-fetch the artifact from the index at execution time.
		row, err := p.Idx.Get(relPath)
		if err != nil {
			results = append(results, advanceResult{
				Path:  relPath,
				Ok:    false,
				Error: fmt.Sprintf("db error: %s", err.Error()),
			})
			continue
		}
		if row == nil {
			results = append(results, advanceResult{
				Path:  relPath,
				Ok:    false,
				Error: "artifact not found",
			})
			continue
		}

		// Re-evaluate staleness for this artifact's lineage.
		lineageArtifacts, err := p.Idx.ListByLineage(row.Lineage)
		if err != nil {
			results = append(results, advanceResult{
				Path:  relPath,
				Ok:    false,
				Error: fmt.Sprintf("db error fetching lineage: %s", err.Error()),
			})
			continue
		}

		staleResults := statuscheck.Check(lineageArtifacts)

		// Find the staleness result for this specific artifact.
		var suggested string
		for _, sr := range staleResults {
			if sr.Path == relPath {
				suggested = sr.SuggestedStatus
				break
			}
		}

		if suggested == "" {
			// Artifact is not stale — idempotent no-op.
			results = append(results, advanceResult{
				Path: relPath,
				Ok:   true,
			})
			continue
		}

		// Check permission.
		if !p.Workflow.CanTransition(row.Status, suggested, userRoles) {
			results = append(results, advanceResult{
				Path:  relPath,
				Ok:    false,
				Error: fmt.Sprintf("requires role with permission to transition %q → %q", row.Status, suggested),
			})
			continue
		}

		// Apply the transition.
		if err := applyTransition(p, row, relPath, suggested, user.Email, ""); err != nil {
			results = append(results, advanceResult{
				Path:  relPath,
				Ok:    false,
				Error: err.Error(),
			})
			continue
		}

		results = append(results, advanceResult{
			Path:       relPath,
			AdvancedTo: suggested,
			Ok:         true,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

