// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/agent"
	"github.com/kaos-control/kaos-control/internal/config"
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
// GET .../provider-switch/status response.
type providerSwitchAgentStatus struct {
	Agent            string `json:"agent"`
	IsFailover       bool   `json:"is_failover"`
	PrimaryProvider  string `json:"primary_provider,omitempty"`
	PrimaryModel     string `json:"primary_model,omitempty"`
	ActiveProvider   string `json:"active_provider"`
	ActiveModel      string `json:"active_model"`
	FallbackProvider string `json:"fallback_provider,omitempty"`
	FallbackModel    string `json:"fallback_model,omitempty"`
	// PrimaryHealthy is a live reachability probe of the primary provider,
	// populated only for agents currently in a failover state.
	PrimaryHealthy *bool `json:"primary_healthy,omitempty"`
}

// handleGetFailoverStatus returns project-wide provider-failover status.
// GET /api/p/{project}/provider-switch/status
func (s *Server) handleGetFailoverStatus(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p.Agents == nil {
		writeJSON(w, http.StatusOK, map[string]any{"failover_active": false, "agents": []any{}})
		return
	}

	out := []providerSwitchAgentStatus{}
	active := false
	for _, ag := range p.Agents.Agents() {
		isFailover := ag.PrimaryProvider != ""
		if isFailover {
			active = true
		}
		status := providerSwitchAgentStatus{
			Agent:            ag.Name,
			IsFailover:       isFailover,
			PrimaryProvider:  ag.PrimaryProvider,
			PrimaryModel:     ag.PrimaryModel,
			ActiveProvider:   ag.Provider,
			ActiveModel:      ag.Model,
			FallbackProvider: ag.FallbackProvider,
			FallbackModel:    ag.FallbackModel,
		}
		if isFailover {
			if prov, ok := s.providerConfig(ag.PrimaryProvider); ok {
				healthy := agent.ProbeProviderHealth(r.Context(), nil, prov, 2*time.Second)
				status.PrimaryHealthy = &healthy
			}
		}
		out = append(out, status)
	}
	writeJSON(w, http.StatusOK, map[string]any{"failover_active": active, "agents": out})
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
