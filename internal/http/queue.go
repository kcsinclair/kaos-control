// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/queue"
)

// handleEnqueue adds a job to the app-level queue.
// POST /api/queue
// Request:  {"project": "kaos-control", "artifact_path": "lifecycle/ideas/foo.md", "agent": "requirements-analyst"}
// Response: 201 {"id": "...", "position": 3}
func (s *Server) handleEnqueue(w http.ResponseWriter, r *http.Request) {
	if s.queue == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError("not_configured", "queue not available"))
		return
	}
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	var req struct {
		Project      string `json:"project"`
		ArtifactPath string `json:"artifact_path"`
		Agent        string `json:"agent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}
	if req.Project == "" || req.ArtifactPath == "" || req.Agent == "" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "project, artifact_path, and agent are required"))
		return
	}

	// Look up the project to validate and check role.
	// An unrecognised project name is treated as a bad request (400) because
	// the client supplied an invalid value for the `project` field.
	p, ok := s.projects[req.Project]
	if !ok {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "unknown project: "+req.Project))
		return
	}

	// Find the agent config so we can check the required roles.
	var agentCfg *config.AgentConfig
	for i := range p.Cfg.Agents {
		if p.Cfg.Agents[i].Name == req.Agent {
			agentCfg = &p.Cfg.Agents[i]
			break
		}
	}
	if agentCfg == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "agent not configured: "+req.Agent))
		return
	}

	// product-owner may always enqueue; otherwise the user must hold one of
	// the agent's configured roles (same logic as handleStartAgentRun).
	allowed := append([]string{RoleProductOwner}, agentCfg.Roles...)
	userRoles := p.Cfg.RolesFor(user.Email)
	if !hasAnyRole(userRoles, allowed...) {
		writeJSON(w, http.StatusForbidden, apiError("forbidden", "insufficient role to enqueue this agent"))
		return
	}

	job := queue.Job{
		Project:      req.Project,
		ArtifactPath: req.ArtifactPath,
		AgentName:    req.Agent,
		EnqueuedBy:   user.Email,
	}
	if err := s.queue.Enqueue(job); err != nil {
		if err == queue.ErrDuplicateActive {
			writeJSON(w, http.StatusConflict, apiError("duplicate", "an active job for this artifact already exists"))
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError("queue_error", err.Error()))
		return
	}

	// Retrieve the assigned position from the store.
	active, findErr := s.queue.FindActiveByPath(req.Project, req.ArtifactPath)
	pos := int64(0)
	id := ""
	if findErr == nil && active != nil {
		pos = active.Position
		id = active.ID
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":       id,
		"position": pos,
	})
}

// handleListQueue returns the current queue state.
// GET /api/queue
func (s *Server) handleListQueue(w http.ResponseWriter, r *http.Request) {
	if s.queue == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"running": nil, "pending": []any{}, "recent": []any{},
			"paused": false, "paused_until": nil, "pause_reason": "",
		})
		return
	}
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	snap, err := s.queue.StateSnapshot()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("queue_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

// handleCancelQueue cancels a pending queue job.
// DELETE /api/queue/{id}
func (s *Server) handleCancelQueue(w http.ResponseWriter, r *http.Request) {
	if s.queue == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError("not_configured", "queue not available"))
		return
	}
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	id := chi.URLParam(r, "id")
	job, err := s.queue.GetByID(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("queue_error", err.Error()))
		return
	}
	if job == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "queue job not found"))
		return
	}

	// Only the enqueuer or a product-owner may cancel.
	if job.EnqueuedBy != user.Email && !s.appUserHasRole(user, RoleProductOwner) {
		writeJSON(w, http.StatusForbidden, apiError("forbidden", "only the enqueuer or product-owner may cancel"))
		return
	}

	if err := s.queue.Cancel(id); err != nil {
		if err == queue.ErrCannotCancelRunning {
			writeJSON(w, http.StatusConflict, apiError("running", "cannot cancel a running job"))
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError("queue_error", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handlePauseQueue manually pauses the queue.
// POST /api/queue/pause
func (s *Server) handlePauseQueue(w http.ResponseWriter, r *http.Request) {
	if s.queue == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError("not_configured", "queue not available"))
		return
	}
	if !s.requireAppRole(w, r, RoleProductOwner, RoleDevops) {
		return
	}
	s.queue.Pause("manual")
	w.WriteHeader(http.StatusNoContent)
}

// handleResumeQueue manually resumes the queue.
// POST /api/queue/resume
func (s *Server) handleResumeQueue(w http.ResponseWriter, r *http.Request) {
	if s.queue == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError("not_configured", "queue not available"))
		return
	}
	if !s.requireAppRole(w, r, RoleProductOwner, RoleDevops) {
		return
	}
	s.queue.Resume()
	w.WriteHeader(http.StatusNoContent)
}

