// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/index"
	"github.com/kaos-control/kaos-control/internal/triage"
)

// handleTriageIdea handles POST /api/p/:project/ideas/{slug}/triage.
//
// Requires authentication. The caller must have one of: product-owner,
// analyst, or reviewer role. Resolves the slug to the artifact under
// lifecycle/ideas/ and calls the triage manager synchronously for the
// trigger, returning the run ID immediately (the actual triage runs async).
//
// Responses:
//
//	202  {"run_id": "<id>"}                    — run started or coalesced
//	401  {"error": …}                          — not authenticated
//	403  {"error": …}                          — insufficient role
//	404  {"error": …}                          — no ideas/<slug>.md found
//	409  {"error":"not_eligible","reason":"…"} — artifact not eligible
//	409  {"error":"locked"}                     — lineage locked
//	503  {"error":"busy"}                       — semaphore at capacity
//	500  {"error": …}                           — unexpected error
func (s *Server) handleTriageIdea(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "slug is required"))
		return
	}

	// Role check: product-owner, analyst, or reviewer may trigger triage.
	roles := p.Cfg.RolesFor(user.Email)
	if !hasAnyRole(roles, "product-owner", "analyst", "reviewer") {
		writeJSON(w, http.StatusForbidden, apiError("forbidden", "role product-owner, analyst, or reviewer required"))
		return
	}

	// Look up the idea artifact for this slug under lifecycle/ideas/.
	rows, _, err := p.Idx.List(index.Filter{Lineage: slug, Type: "idea", Unlimited: true})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	var ideaRow *index.ArtifactRow
	for _, row := range rows {
		if len(row.Path) > len("lifecycle/ideas/") &&
			row.Path[:len("lifecycle/ideas/")] == "lifecycle/ideas/" {
			ideaRow = row
			break
		}
	}
	if ideaRow == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "no idea artifact found for slug "+slug))
		return
	}

	if p.TriageMgr == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("config_error", "triage manager not initialised"))
		return
	}

	runID, triggerErr := p.TriageMgr.Trigger(r.Context(), ideaRow.Path, triage.TriggerAPI)
	if triggerErr == nil {
		writeJSON(w, http.StatusAccepted, map[string]any{"run_id": runID})
		return
	}

	var ie triage.ErrIneligible
	if errors.As(triggerErr, &ie) {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":  "not_eligible",
			"reason": ie.Reason,
		})
		return
	}
	if errors.Is(triggerErr, triage.ErrLocked) {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "locked"})
		return
	}
	if errors.Is(triggerErr, triage.ErrBusy) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "busy"})
		return
	}
	writeJSON(w, http.StatusInternalServerError, apiError("triage_error", triggerErr.Error()))
}

