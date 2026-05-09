// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
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
		slog.Debug("status-check: evaluating lineage", "lineage", lineageSlug, "user", user.Email)
		artifacts, err := p.Idx.ListByLineage(lineageSlug)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
			return
		}
		allResults = statuscheck.Check(artifacts)
		slog.Debug("status-check: lineage evaluated", "lineage", lineageSlug, "stale_count", len(allResults))
	} else {
		// Project-wide check: run algorithm per lineage.
		grouped, err := p.Idx.ListAllGroupedByLineage()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
			return
		}
		for lineage, artifacts := range grouped {
			slog.Debug("status-check: evaluating lineage", "lineage", lineage, "artifact_count", len(artifacts))
			results := statuscheck.Check(artifacts)
			slog.Debug("status-check: lineage evaluated", "lineage", lineage, "stale_count", len(results))
			allResults = append(allResults, results...)
		}
	}

	// Annotate each result with can_advance / blocked_reason.
	for i := range allResults {
		r := &allResults[i]
		if p.Workflow.CanTransition(r.CurrentStatus, r.SuggestedStatus, userRoles, r.Type) {
			r.CanAdvance = true
			slog.Debug("status-check: artifact stale and can advance",
				"path", r.Path, "current_status", r.CurrentStatus, "suggested_status", r.SuggestedStatus)
		} else {
			r.CanAdvance = false
			r.BlockedReason = fmt.Sprintf("requires role with permission to transition %q → %q",
				r.CurrentStatus, r.SuggestedStatus)
			slog.Debug("status-check: artifact stale but advance blocked",
				"path", r.Path, "current_status", r.CurrentStatus, "suggested_status", r.SuggestedStatus,
				"reason", r.BlockedReason)
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
//	{"results": [
//	  {"path": "...", "outcome": "advanced", "ok": true, "advanced_to": "planning"},
//	  {"path": "...", "outcome": "skipped", "ok": false},
//	  {"path": "...", "outcome": "error", "ok": false, "reason": "requires role with permission to transition ..."}
//	]}
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
		Outcome    string `json:"outcome"`
		Ok         bool   `json:"ok"`
		AdvancedTo string `json:"advanced_to,omitempty"`
		Reason     string `json:"reason,omitempty"`
	}

	results := make([]advanceResult, 0, len(req.Paths))

	for _, relPath := range req.Paths {
		slog.Debug("status-check/advance: processing artifact", "path", relPath, "user", user.Email)

		// Re-fetch the artifact from the index at execution time.
		row, err := p.Idx.Get(relPath)
		if err != nil {
			slog.Debug("status-check/advance: db error fetching artifact", "path", relPath, "err", err)
			results = append(results, advanceResult{
				Path:    relPath,
				Outcome: "error",
				Reason:  fmt.Sprintf("db error: %s", err.Error()),
			})
			continue
		}
		if row == nil {
			slog.Debug("status-check/advance: artifact not found", "path", relPath)
			results = append(results, advanceResult{
				Path:    relPath,
				Outcome: "error",
				Reason:  "artifact not found",
			})
			continue
		}

		// Re-evaluate staleness for this artifact's lineage.
		lineageArtifacts, err := p.Idx.ListByLineage(row.Lineage)
		if err != nil {
			slog.Debug("status-check/advance: db error fetching lineage", "path", relPath, "lineage", row.Lineage, "err", err)
			results = append(results, advanceResult{
				Path:    relPath,
				Outcome: "error",
				Reason:  fmt.Sprintf("db error fetching lineage: %s", err.Error()),
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
			slog.Debug("status-check/advance: artifact not stale, skipping", "path", relPath, "status", row.Status)
			results = append(results, advanceResult{
				Path:    relPath,
				Outcome: "skipped",
			})
			continue
		}

		// Check permission.
		if !p.Workflow.CanTransition(row.Status, suggested, userRoles, row.Type) {
			reason := fmt.Sprintf("requires role with permission to transition %q → %q", row.Status, suggested)
			slog.Debug("status-check/advance: advance blocked by permissions",
				"path", relPath, "from", row.Status, "to", suggested, "reason", reason)
			results = append(results, advanceResult{
				Path:    relPath,
				Outcome: "error",
				Reason:  reason,
			})
			continue
		}

		// Apply the transition.
		if err := applyTransition(p, row, relPath, suggested, user.Email, ""); err != nil {
			slog.Debug("status-check/advance: transition error", "path", relPath, "from", row.Status, "to", suggested, "err", err)
			results = append(results, advanceResult{
				Path:    relPath,
				Outcome: "error",
				Reason:  err.Error(),
			})
			continue
		}

		slog.Info("status-check/advance: artifact advanced",
			"path", relPath, "from", row.Status, "to", suggested, "user", user.Email)
		results = append(results, advanceResult{
			Path:       relPath,
			Outcome:    "advanced",
			Ok:         true,
			AdvancedTo: suggested,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}
