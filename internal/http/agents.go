package http

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/agent"
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
