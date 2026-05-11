// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/agent"
	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/index"
)

// handleListAgents returns all configured agents for the current project.
func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p.Agents == nil {
		writeJSON(w, http.StatusOK, map[string]any{"agents": []any{}})
		return
	}
	type agentSummary struct {
		Name               string   `json:"name"`
		Roles              []string `json:"roles"`
		Driver             string   `json:"driver"`
		Model              string   `json:"model,omitempty"`
		ActiveStatus       string   `json:"active_status,omitempty"`
		AllowedPaths       []string `json:"allowed_write_paths,omitempty"`
		OllamaInstanceName string   `json:"ollama_instance,omitempty"`
		OllamaEndpoint     string   `json:"ollama_endpoint,omitempty"`
	}
	var out []agentSummary
	for _, ag := range p.Agents.Agents() {
		out = append(out, agentSummary{
			Name:               ag.Name,
			Roles:              ag.Roles,
			Driver:             ag.Driver,
			Model:              ag.Model,
			ActiveStatus:       ag.ActiveStatus,
			AllowedPaths:       ag.AllowedPaths,
			OllamaInstanceName: ag.OllamaInstanceName,
			OllamaEndpoint:     ag.OllamaEndpoint,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"agents": out})
}

// handleStartAgentRun triggers a new agent run.
// POST /api/p/:project/agents/:name/run  body: {target_path, role?}
func (s *Server) handleStartAgentRun(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}
	if p.Agents == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError("not_configured", "agents not configured for this project"))
		return
	}

	name := chi.URLParam(r, "name")

	// Derive allowed roles: product-owner always permitted, plus the agent's own configured roles.
	var agentCfg *config.AgentConfig
	for i := range p.Cfg.Agents {
		if p.Cfg.Agents[i].Name == name {
			agentCfg = &p.Cfg.Agents[i]
			break
		}
	}
	if agentCfg == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "agent "+name+" not configured"))
		return
	}
	allowed := append([]string{RoleProductOwner}, agentCfg.Roles...)
	if !requireRole(w, r, p, allowed...) {
		return
	}

	var req struct {
		TargetPath string `json:"target_path"`
		Role       string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}

	runID, err := p.Agents.StartRun(r.Context(), name, req.TargetPath, req.Role, user)
	if err != nil {
		switch err {
		case agent.ErrNotFound:
			writeJSON(w, http.StatusNotFound, apiError("not_found", "agent "+name+" not configured"))
		case agent.ErrBusy:
			writeJSON(w, http.StatusServiceUnavailable, apiError("busy", "agent concurrency limit reached"))
		default:
			writeJSON(w, http.StatusConflict, apiError("run_error", err.Error()))
		}
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{"run_id": runID})
}

// handleListAgentRuns lists run records, filtered by optional ?status= and ?target_path= query params.
func (s *Server) handleListAgentRuns(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p.Agents == nil {
		writeJSON(w, http.StatusOK, map[string]any{"runs": []any{}})
		return
	}
	if targetPath := r.URL.Query().Get("target_path"); targetPath != "" {
		runs, err := p.Agents.ListRunsByTargetPath(targetPath)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
		return
	}
	status := r.URL.Query().Get("status")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	runs, err := p.Agents.ListRuns(status, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
}

// handleGetAgentRun returns detail for a single run.
func (s *Server) handleGetAgentRun(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p.Agents == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "agents not configured"))
		return
	}
	runID := chi.URLParam(r, "run_id")
	run, err := p.Agents.GetRun(runID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	if run == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "run not found"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"run": run})
}

// handleKillAgentRun sends SIGTERM to a running agent.
func (s *Server) handleKillAgentRun(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}
	if p.Agents == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "agents not configured"))
		return
	}
	runID := chi.URLParam(r, "run_id")
	if err := p.Agents.Kill(runID); err != nil {
		writeJSON(w, http.StatusConflict, apiError("kill_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "run_id": runID})
}

// readyInputStatus is the artifact status that signals "ready for an agent to
// pick up". The agent runner transitions the artifact out of this state and
// into its configured ActiveStatus on run start, so ActiveStatus is the
// *during-run* status — the wrong column to count for a "ready" badge.
const readyInputStatus = "approved"

// hasDeveloperSourceType reports whether any of the given source types is a
// plan-* type, identifying a developer agent that also picks up assigned
// defects (matching the AgentLaunchModal's plan-* branch).
func hasDeveloperSourceType(types []string) bool {
	for _, t := range types {
		if strings.HasPrefix(t, "plan-") {
			return true
		}
	}
	return false
}

// countAssignedDefects returns the number of approved defect artifacts whose
// frontmatter assignees include at least one of the given agent roles. Mirrors
// the JS-side filter in web/src/components/agent/AgentLaunchModal.vue so the
// badge count agrees with the dialog's list size.
func countAssignedDefects(idx *index.Index, agentRoles []string) (int, error) {
	if len(agentRoles) == 0 {
		return 0, nil
	}
	defects, _, err := idx.List(index.Filter{
		Status:    readyInputStatus,
		Type:      "defect",
		Unlimited: true,
	})
	if err != nil {
		return 0, err
	}
	want := make(map[string]struct{}, len(agentRoles))
	for _, r := range agentRoles {
		want[r] = struct{}{}
	}
	var n int
	for _, d := range defects {
		for _, a := range d.FM.Assignees {
			if _, ok := want[a.Role]; ok {
				n++
				break
			}
		}
	}
	return n, nil
}

// handleGetReadyCounts returns per-agent counts of artifacts whose status is
// the ready-for-pickup status ("approved"), filtered by each agent's
// source_types when set. For developer agents (any plan-* source type), the
// count also includes approved defect artifacts whose assignees match the
// agent's roles — matching the AgentLaunchModal's plan-* branch so the badge
// agrees with the launch dialog.
// GET /api/p/:project/agents/ready-counts
func (s *Server) handleGetReadyCounts(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p.Agents == nil {
		writeJSON(w, http.StatusOK, map[string]any{"counts": map[string]int{}})
		return
	}
	counts := make(map[string]int)
	for _, ag := range p.Agents.Agents() {
		// Agents with no ActiveStatus are not part of the run lifecycle; skip
		// them so we don't add spurious badge counts for misconfigured agents.
		if ag.ActiveStatus == "" {
			continue
		}
		var total int
		if len(ag.SourceTypes) == 0 {
			n, err := p.Idx.Count(index.Filter{Status: readyInputStatus})
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
				return
			}
			total = n
		} else {
			for _, t := range ag.SourceTypes {
				n, err := p.Idx.Count(index.Filter{Status: readyInputStatus, Type: t})
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
					return
				}
				total += n
			}
		}
		// Developer agents also pick up approved defects assigned to their role.
		if hasDeveloperSourceType(ag.SourceTypes) {
			n, err := countAssignedDefects(p.Idx, ag.Roles)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
				return
			}
			total += n
		}
		counts[ag.Name] = total
	}
	writeJSON(w, http.StatusOK, map[string]any{"counts": counts})
}

// handleGetAgentRunLog streams the per-run log file as text/plain.
func (s *Server) handleGetAgentRunLog(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p.Agents == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "agents not configured"))
		return
	}
	runID := chi.URLParam(r, "run_id")
	logPath := p.Agents.LogPath(runID)
	if logPath == "" {
		writeJSON(w, http.StatusNotFound, apiError("no_log", "log files are not enabled for this project"))
		return
	}
	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, apiError("no_log", "no log file for this run yet"))
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError("read_error", err.Error()))
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, f)
}
