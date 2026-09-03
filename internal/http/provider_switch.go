// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/project"
)

// providerExists reports whether name is a registered app-level provider.
func (s *Server) providerExists(name string) bool {
	if s.appCfg == nil {
		return false
	}
	s.appCfgMu.RLock()
	defer s.appCfgMu.RUnlock()
	return findProvider(s.appCfg.Providers, name) >= 0
}

// providerConfig returns the named app-level provider config, or ok=false.
func (s *Server) providerConfig(name string) (config.Provider, bool) {
	if s.appCfg == nil {
		return config.Provider{}, false
	}
	s.appCfgMu.RLock()
	defer s.appCfgMu.RUnlock()
	idx := findProvider(s.appCfg.Providers, name)
	if idx < 0 {
		return config.Provider{}, false
	}
	return s.appCfg.Providers[idx], true
}

// providerSwitchAgentStatus is one agent's entry in the
// GET .../provider-switch/status response — built entirely from
// operations.yaml (agent-switchover-and-failover Milestone 8), never from a
// live probe: reachability is populated by the background RecoveryProber
// (Milestone 6), which runs in every mode, not just while failed over.
type providerSwitchAgentStatus struct {
	Agent            string `json:"agent"`
	IsFailover       bool   `json:"is_failover"`
	PrimaryProvider  string `json:"primary_provider,omitempty"`
	PrimaryModel     string `json:"primary_model,omitempty"`
	ActiveProvider   string `json:"active_provider"`
	ActiveModel      string `json:"active_model"`
	FallbackProvider string `json:"fallback_provider,omitempty"`
	FallbackModel    string `json:"fallback_model,omitempty"`
	// SwitchedAt (RFC3339) and Reason describe the most recent switch away
	// from Primary; both are empty when the agent is on its primary.
	SwitchedAt string `json:"switched_at,omitempty"`
	Reason     string `json:"reason,omitempty"`
	// ResetsAtUnix + Bucket carry rate-limit context (FR-3.3) so the UI can
	// show when the primary is expected to become usable again.
	ResetsAtUnix int64  `json:"resets_at_unix,omitempty"`
	Bucket       string `json:"bucket,omitempty"`
	// PartialPause (FR-3.4): this agent has no secondary to fail over to and
	// its jobs are paused while the rest of the project proceeds.
	PartialPause bool `json:"partial_pause,omitempty"`
	// AwaitingDecision (FR-7.3): a job for this agent was interrupted with a
	// suspected partial commit and needs an operator decision.
	AwaitingDecision      bool   `json:"awaiting_decision,omitempty"`
	AwaitingDecisionJobID string `json:"awaiting_decision_job_id,omitempty"`
}

// providerReachabilityStatus is one provider's entry in the top-level
// reachability map of the status response (Milestone 6: every provider
// bound to any agent, in every mode — not only ones currently failed over).
type providerReachabilityStatus struct {
	Healthy      bool  `json:"healthy"`
	LastProbedAt int64 `json:"last_probed_at,omitempty"`
	Since        int64 `json:"since,omitempty"`
}

// handleGetFailoverStatus returns project-wide provider-failover status,
// built from operations.yaml: active-vs-primary per agent, reason,
// switched_at, resets_at_unix, bucket, partial pause, awaiting-operator-
// decision, and provider reachability in every mode (FR-8.4).
// GET /api/p/{project}/provider-switch/status
func (s *Server) handleGetFailoverStatus(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p.Agents == nil {
		writeJSON(w, http.StatusOK, map[string]any{"failover_active": false, "agents": []any{}, "reachability": map[string]any{}})
		return
	}

	out := []providerSwitchAgentStatus{}
	active := false
	for _, ag := range p.Agents.Agents() {
		// The effective active provider is an operations.yaml override, if
		// recorded, else the agent's declared config (agent-switchover-and-
		// failover Milestone 2) — lifecycle/config.yaml is never mutated by a
		// switch, so failover state must be read from the operations store,
		// not from ag.PrimaryProvider/ag.Provider directly.
		state, hasState := p.Operations().AgentState(ag.Name)
		isFailover := hasState && state.IsFailedOver()
		activeProvider, activeModel := ag.Provider, ag.Model
		if hasState {
			activeProvider, activeModel = state.Active.Provider, state.Active.Model
		}
		if isFailover {
			active = true
		}
		status := providerSwitchAgentStatus{
			Agent:                 ag.Name,
			IsFailover:            isFailover,
			ActiveProvider:        activeProvider,
			ActiveModel:           activeModel,
			FallbackProvider:      ag.FallbackProvider,
			FallbackModel:         ag.FallbackModel,
			PartialPause:          hasState && state.PartialPause,
			AwaitingDecision:      hasState && state.AwaitingOperatorDecision,
			AwaitingDecisionJobID: state.AwaitingDecisionJobID,
		}
		if hasState {
			status.Reason = state.Reason
			if state.SwitchedAt > 0 {
				status.SwitchedAt = time.Unix(state.SwitchedAt, 0).UTC().Format(time.RFC3339)
			}
			status.ResetsAtUnix = state.ResetsAtUnix
			status.Bucket = state.Bucket
		}
		if isFailover {
			status.PrimaryProvider = state.Primary.Provider
			status.PrimaryModel = state.Primary.Model
		}
		out = append(out, status)
	}

	reachability := map[string]providerReachabilityStatus{}
	for name, reach := range p.Operations().AllReachability() {
		reachability[name] = providerReachabilityStatus{Healthy: reach.Healthy, LastProbedAt: reach.LastProbedAt, Since: reach.Since}
	}

	writeJSON(w, http.StatusOK, map[string]any{"failover_active": active, "agents": out, "reachability": reachability})
}

// handleGetSwitchoverPolicy returns the project's effective event->action
// switchover policy (FR-2.4): automated_switchover and an explicit action
// for every classified reason, configured entries overriding the FR-2.3
// defaults.
// GET /api/p/{project}/provider-switch/policy
func (s *Server) handleGetSwitchoverPolicy(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	writeJSON(w, http.StatusOK, p.Config().EffectiveSwitchoverPolicy())
}

// runningJobsRejection guards a manual provider switch against in-flight
// runs (FR-8.2): if any run is currently executing for the project, the
// switch is rejected with a 409 naming the running jobs, and ok=false. When
// the queue isn't wired (nil, e.g. minimal test/deploy configurations) the
// guard is a no-op, matching the rest of the queue-optional handlers in
// this package.
func (s *Server) runningJobsRejection(w http.ResponseWriter, p *project.Project) bool {
	if s.queue == nil {
		return true
	}
	running, err := s.queue.RunningJobs(p.Entry.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("queue_error", err.Error()))
		return false
	}
	if len(running) == 0 {
		return true
	}
	jobs := make([]map[string]any, 0, len(running))
	for _, j := range running {
		jobs = append(jobs, map[string]any{
			"id":            j.ID,
			"agent":         j.AgentName,
			"artifact_path": j.ArtifactPath,
		})
	}
	writeJSON(w, http.StatusConflict, map[string]any{
		"error":        map[string]any{"code": "runs_in_progress", "message": "cannot switch provider while runs are in progress"},
		"running_jobs": jobs,
	})
	return false
}

// handleAgentSwitchProvider manually switches an individual agent's
// provider/model.
// POST /api/p/{project}/agents/{name}/switch-provider
// Request: {"provider": "...", "model": "...", "reason": "..."}
func (s *Server) handleAgentSwitchProvider(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if !requireRole(w, r, p, RolesDevopsOrAdmin...) {
		return
	}
	if p.Agents == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError("not_configured", "agents not configured for this project"))
		return
	}
	name := chi.URLParam(r, "name")
	if _, ok := p.Agents.GetAgent(name); !ok {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "agent "+name+" not configured"))
		return
	}

	var req struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
		Reason   string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}
	if req.Provider == "" || req.Model == "" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "provider and model are required"))
		return
	}
	if !s.providerExists(req.Provider) {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "unknown provider: "+req.Provider))
		return
	}
	if !s.runningJobsRejection(w, p) {
		return
	}

	if err := p.SwitchAgentProvider(name, req.Provider, req.Model, req.Reason, false); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("switch_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "agent": name, "provider": req.Provider, "model": req.Model})
}

// handleAgentRestoreProvider restores an agent to its primary provider/model.
// POST /api/p/{project}/agents/{name}/restore-provider
func (s *Server) handleAgentRestoreProvider(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if !requireRole(w, r, p, RolesDevopsOrAdmin...) {
		return
	}
	if p.Agents == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError("not_configured", "agents not configured for this project"))
		return
	}
	name := chi.URLParam(r, "name")
	if _, ok := p.Agents.GetAgent(name); !ok {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "agent "+name+" not configured"))
		return
	}

	if err := p.RestoreAgentProvider(name); err != nil {
		writeJSON(w, http.StatusConflict, apiError("restore_error", err.Error()))
		return
	}

	ag, _ := p.Agents.GetAgent(name)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "agent": name, "provider": ag.Provider, "model": ag.Model})
}

// handleSwitchAllProviders batch-switches every agent currently on
// from_provider to to_provider/to_model.
// POST /api/p/{project}/provider-switch/switch-all
// Request: {"from_provider": "...", "to_provider": "...", "to_model": "...", "reason": "..."}
func (s *Server) handleSwitchAllProviders(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if !requireRole(w, r, p, RolesDevopsOrAdmin...) {
		return
	}

	var req struct {
		FromProvider string `json:"from_provider"`
		ToProvider   string `json:"to_provider"`
		ToModel      string `json:"to_model"`
		Reason       string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}
	if req.FromProvider == "" || req.ToProvider == "" || req.ToModel == "" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "from_provider, to_provider, and to_model are required"))
		return
	}
	if !s.providerExists(req.ToProvider) {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "unknown provider: "+req.ToProvider))
		return
	}
	if !s.runningJobsRejection(w, p) {
		return
	}

	n, err := p.SwitchAllAgentProviders(req.FromProvider, req.ToProvider, req.ToModel, req.Reason)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("switch_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "switched_agents": n, "from_provider": req.FromProvider, "to_provider": req.ToProvider})
}

// handleRestoreAllProviders restores every agent currently in a failover state.
// POST /api/p/{project}/provider-switch/restore-all
func (s *Server) handleRestoreAllProviders(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if !requireRole(w, r, p, RolesDevopsOrAdmin...) {
		return
	}

	n, err := p.RestoreAllAgentProviders()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("restore_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "restored_agents": n})
}

// handleListProviderTemplates lists the project's configured provider presets.
// GET /api/p/{project}/provider-templates
func (s *Server) handleListProviderTemplates(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	templates := p.Config().ProviderTemplates
	if templates == nil {
		templates = []config.ProviderTemplate{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"templates": templates})
}

// handleApplyProviderTemplate applies a named provider template across project agents.
// POST /api/p/{project}/provider-templates/apply
// Request: {"template": "..."}
func (s *Server) handleApplyProviderTemplate(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if !requireRole(w, r, p, RolesDevopsOrAdmin...) {
		return
	}

	var req struct {
		Template string `json:"template"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}
	if req.Template == "" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "template is required"))
		return
	}

	n, err := p.ApplyProviderTemplate(req.Template)
	if err != nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "template": req.Template, "updated_agents": n})
}
