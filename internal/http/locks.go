// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/lock"
)

// handleListLocks handles GET /api/p/:project/locks
func (s *Server) handleListLocks(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil || p.Locks == nil {
		writeJSON(w, http.StatusOK, map[string]any{"locks": []any{}})
		return
	}
	locks, err := p.Idx.ListActiveLocks()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"locks": locks})
}

// handleAcquireLock handles POST /api/p/:project/locks
func (s *Server) handleAcquireLock(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}
	if p == nil || p.Locks == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError("unavailable", "lock manager not configured"))
		return
	}

	var req struct {
		Lineage string `json:"lineage"`
		Kind    string `json:"kind"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}
	if req.Lineage == "" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "lineage is required"))
		return
	}
	if req.Kind == "" {
		req.Kind = "editor"
	}

	lockRow, err := p.Locks.Acquire(req.Lineage, user.Email, req.Kind)
	if err != nil {
		if err == lock.ErrLocked {
			existing, _ := p.Locks.Get(req.Lineage)
			writeJSON(w, http.StatusConflict, map[string]any{
				"error": map[string]any{"code": "locked", "message": "lineage is already locked"},
				"lock":  existing,
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lock": lockRow})
}

// handleReleaseLock handles DELETE /api/p/:project/locks/:lineage
func (s *Server) handleReleaseLock(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil || p.Locks == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	lineage := chi.URLParam(r, "lineage")
	_ = p.Locks.Release(lineage)
	w.WriteHeader(http.StatusNoContent)
}

// handleHeartbeatLock handles POST /api/p/:project/locks/:lineage/heartbeat
func (s *Server) handleHeartbeatLock(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil || p.Locks == nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	lineage := chi.URLParam(r, "lineage")
	if err := p.Locks.Heartbeat(lineage); err != nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "lock not found"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
