// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/project"
	"github.com/kaos-control/kaos-control/internal/sandbox"
	"github.com/kaos-control/kaos-control/internal/scheduler"
)

// jobNameRe validates scheduler job names: alphanumeric + hyphens, 1–64 chars.
var jobNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\-]{0,62}[a-zA-Z0-9]$|^[a-zA-Z0-9]$`)

// ----- request/response helpers -----

// jobRequest is the JSON body for create/update.
type jobRequest struct {
	TargetType    string                `json:"target_type"`
	Target        string                `json:"target"`
	Args          map[string]string     `json:"args,omitempty"`
	Schedule      scheduler.ScheduleSpec `json:"schedule"`
	Preconditions []scheduler.Precondition `json:"preconditions,omitempty"`
	Enabled       bool                  `json:"enabled"`
	Priority      int                   `json:"priority"`
	TimeoutSec    int                   `json:"timeout_sec"`
}

// ----- GET /scheduler/jobs -----

func (s *Server) handleListSchedulerJobs(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p.SchedulerStore == nil {
		writeJSON(w, http.StatusOK, map[string]any{"jobs": []any{}})
		return
	}
	jobs, err := p.SchedulerStore.ListJobs()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	now := time.Now()
	for _, j := range jobs {
		if j.Enabled {
			last, _ := p.SchedulerStore.LastRunForJob(j.Name)
			var lastRunTime time.Time
			if last != nil && last.EndTime != nil {
				lastRunTime = *last.EndTime
			}
			next := scheduler.NextFireTime(j.Schedule, lastRunTime, now)
			if !next.IsZero() {
				j.NextRunAt = &next
			}
		}
	}
	if jobs == nil {
		jobs = []*scheduler.Job{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": jobs})
}

// ----- GET /scheduler/jobs/{name} -----

func (s *Server) handleGetSchedulerJob(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p.SchedulerStore == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "scheduler not configured"))
		return
	}
	name := chi.URLParam(r, "name")
	job, err := p.SchedulerStore.GetJob(name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	if job == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "job not found"))
		return
	}
	// Attach last 10 runs.
	runs, _, err := p.SchedulerStore.ListRuns(name, 1, 10)
	if err != nil {
		runs = nil
	}
	if runs == nil {
		runs = []*scheduler.Run{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"job": job, "runs": runs})
}

// ----- POST /scheduler/jobs -----

func (s *Server) handleCreateSchedulerJob(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p.SchedulerStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError("not_configured", "scheduler not configured"))
		return
	}
	if !requireRole(w, r, p, RolesDevopsOrAdmin...) {
		return
	}
	name := r.URL.Query().Get("name")
	// Name may also come from the JSON body; decode first then check.
	var req struct {
		Name string `json:"name"`
		jobRequest
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}
	if name == "" {
		name = req.Name
	}
	if !jobNameRe.MatchString(name) {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "job name must be alphanumeric+hyphens, 1–64 chars"))
		return
	}
	agentNames := configuredAgentNames(p)
	if err := validateJobRequest(req.jobRequest, p.Entry.Path, agentNames); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", err.Error()))
		return
	}
	if req.Priority == 0 {
		req.Priority = 5
	}

	// Check uniqueness.
	existing, err := p.SchedulerStore.GetJob(name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	if existing != nil {
		writeJSON(w, http.StatusConflict, apiError("conflict", "job already exists: "+name))
		return
	}

	job := &scheduler.Job{
		Name:          name,
		TargetType:    req.TargetType,
		Target:        req.Target,
		Args:          req.Args,
		Schedule:      req.Schedule,
		Preconditions: req.Preconditions,
		Enabled:       req.Enabled,
		Priority:      req.Priority,
		TimeoutSec:    req.TimeoutSec,
	}
	if err := p.SchedulerStore.CreateJob(job); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"job": job})
}

// ----- PUT /scheduler/jobs/{name} -----

func (s *Server) handleUpdateSchedulerJob(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p.SchedulerStore == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "scheduler not configured"))
		return
	}
	if !requireRole(w, r, p, RolesDevopsOrAdmin...) {
		return
	}
	name := chi.URLParam(r, "name")
	existing, err := p.SchedulerStore.GetJob(name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	if existing == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "job not found"))
		return
	}

	var req jobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}
	agentNames2 := configuredAgentNames(p)
	if err := validateJobRequest(req, p.Entry.Path, agentNames2); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", err.Error()))
		return
	}
	if req.Priority == 0 {
		req.Priority = existing.Priority
	}

	existing.TargetType = req.TargetType
	existing.Target = req.Target
	existing.Args = req.Args
	existing.Schedule = req.Schedule
	existing.Preconditions = req.Preconditions
	existing.Enabled = req.Enabled
	existing.Priority = req.Priority
	existing.TimeoutSec = req.TimeoutSec

	if err := p.SchedulerStore.UpdateJob(existing); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job": existing})
}

// ----- DELETE /scheduler/jobs/{name} -----

func (s *Server) handleDeleteSchedulerJob(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p.SchedulerStore == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "scheduler not configured"))
		return
	}
	if !requireRole(w, r, p, RolesDevopsOrAdmin...) {
		return
	}
	name := chi.URLParam(r, "name")
	existing, err := p.SchedulerStore.GetJob(name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	if existing == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "job not found"))
		return
	}
	if err := p.SchedulerStore.DeleteJob(name); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ----- POST /scheduler/jobs/{name}/trigger -----

func (s *Server) handleTriggerSchedulerJob(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p.Scheduler == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError("not_configured", "scheduler not running"))
		return
	}
	name := chi.URLParam(r, "name")
	if err := p.Scheduler.TriggerNow(name); err != nil {
		if errors.Is(err, scheduler.ErrJobNotFound) {
			writeJSON(w, http.StatusNotFound, apiError("not_found", err.Error()))
			return
		}
		writeJSON(w, http.StatusConflict, apiError("conflict", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"triggered": name})
}

// ----- POST /scheduler/jobs/{name}/pause -----

func (s *Server) handlePauseSchedulerJob(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p.SchedulerStore == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "scheduler not configured"))
		return
	}
	name := chi.URLParam(r, "name")
	j, err := p.SchedulerStore.GetJob(name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	if j == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "job not found"))
		return
	}
	if p.Scheduler != nil {
		_ = p.Scheduler.Pause(name)
	} else {
		_ = p.SchedulerStore.SetEnabled(name, false)
	}
	writeJSON(w, http.StatusOK, map[string]any{"paused": name})
}

// ----- POST /scheduler/jobs/{name}/resume -----

func (s *Server) handleResumeSchedulerJob(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p.SchedulerStore == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "scheduler not configured"))
		return
	}
	name := chi.URLParam(r, "name")
	j, err := p.SchedulerStore.GetJob(name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	if j == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "job not found"))
		return
	}
	if p.Scheduler != nil {
		_ = p.Scheduler.Resume(name)
	} else {
		_ = p.SchedulerStore.SetEnabled(name, true)
	}
	writeJSON(w, http.StatusOK, map[string]any{"resumed": name})
}

// ----- GET /scheduler/jobs/{name}/runs -----

func (s *Server) handleListSchedulerRuns(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p.SchedulerStore == nil {
		writeJSON(w, http.StatusOK, map[string]any{"runs": []any{}, "total": 0})
		return
	}
	name := chi.URLParam(r, "name")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	runs, total, err := p.SchedulerStore.ListRuns(name, page, perPage)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	if runs == nil {
		runs = []*scheduler.Run{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs, "total": total, "page": page, "per_page": perPage})
}

// ----- GET /scheduler/jobs/{name}/runs/{id}/log -----

func (s *Server) handleGetSchedulerRunLog(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p.SchedulerStore == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "scheduler not configured"))
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid run id"))
		return
	}
	run, err := p.SchedulerStore.GetRun(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	if run == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "run not found"))
		return
	}
	if run.LogPath == "" {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "no log file for this run"))
		return
	}
	f, err := os.Open(run.LogPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, apiError("not_found", "log file has been pruned"))
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, f)
}

// ----- validation -----

// validateJobRequest validates fields common to create and update.
// agentNames is the list of configured agent names for the project.
func validateJobRequest(req jobRequest, projectRoot string, agentNames []string) error {
	if req.TargetType != "agent" && req.TargetType != "shell" {
		return errors.New("target_type must be 'agent' or 'shell'")
	}
	if req.Target == "" {
		return errors.New("target must not be empty")
	}
	if req.Priority != 0 && (req.Priority < 1 || req.Priority > 10) {
		return errors.New("priority must be between 1 and 10")
	}
	if req.TargetType == "shell" {
		if _, err := sandbox.Resolve(projectRoot, req.Target); err != nil {
			return errors.New("shell target path is outside the project sandbox: " + err.Error())
		}
	}
	if req.TargetType == "agent" {
		found := false
		for _, n := range agentNames {
			if n == req.Target {
				found = true
				break
			}
		}
		if !found {
			return errors.New("agent target " + req.Target + " is not a configured agent")
		}
	}
	if err := scheduler.ValidateScheduleSpec(req.Schedule); err != nil {
		return err
	}
	return nil
}

// configuredAgentNames returns the list of configured agent names for a project.
func configuredAgentNames(p *project.Project) []string {
	if p == nil || p.Agents == nil {
		return nil
	}
	agents := p.Agents.Agents()
	names := make([]string, len(agents))
	for i, a := range agents {
		names[i] = a.Name
	}
	return names
}
