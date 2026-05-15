// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/agent"
	"github.com/kaos-control/kaos-control/internal/hub"
)

// hookPermissionRequest is the body Claude Code sends for every PreToolUse event.
type hookPermissionRequest struct {
	ToolName  string         `json:"tool_name"`
	ToolInput map[string]any `json:"tool_input"`
}

// hookPermissionResponse is what the hook helper writes to stdout.
type hookPermissionResponse struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason,omitempty"`
}

// handleHookPermission handles POST /api/agent/{run_id}/permission.
//
// This endpoint is called by the kaos-control hook-helper binary on every
// PreToolUse event from Claude Code. It is exempt from session auth and CSRF
// protection — it authenticates instead via the per-run secret sent in the
// Authorization: Bearer header (or X-Hook-Secret).
func (s *Server) handleHookPermission(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "run_id")

	// --- Extract and validate the per-run secret (FR8) ---
	secret := extractBearerToken(r)
	if secret == "" {
		secret = r.Header.Get("X-Hook-Secret")
	}

	// Find the project manager that owns this run.
	mgr := s.managerForRun(runID)
	if mgr == nil {
		writeJSON(w, http.StatusBadRequest, apiError("unknown_run", "run_id not found or not a claude-mediated run"))
		return
	}

	if !mgr.ValidateRunSecret(runID, secret) {
		writeJSON(w, http.StatusForbidden, apiError("forbidden", "invalid or missing run secret"))
		return
	}

	// --- Decode request body ---
	var req hookPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON body: "+err.Error()))
		return
	}

	// --- Fetch policy config ---
	policy, err := mgr.PolicyForRun(runID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("no_policy", err.Error()))
		return
	}

	// --- Evaluate permission ---
	decision := agent.Evaluate(*policy, req.ToolName, req.ToolInput)

	// --- Structured log line (FR19) ---
	targetPath, _ := req.ToolInput["file_path"].(string)
	command, _ := req.ToolInput["command"].(string)
	slog.Info("agent.permission",
		"run_id", runID,
		"tool_name", req.ToolName,
		"target_path", targetPath,
		"command", command,
		"decision", decision.Action,
		"reason", decision.Reason,
		"policy_rule", decision.Rule,
		"timestamp", time.Now().UTC().Format(time.RFC3339),
	)

	// --- Broadcast WS event (FR20) ---
	wsPayload := map[string]any{
		"run_id":      runID,
		"tool_name":   req.ToolName,
		"target_path": targetPath,
		"command":     command,
		"decision":    decision.Action,
		"reason":      decision.Reason,
		"policy_rule": decision.Rule,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}
	// Find the project hub to broadcast on.
	if projHub := s.hubForRun(runID); projHub != nil {
		projHub.Broadcast(hub.Event{Type: "agent.permission", Payload: wsPayload})
	}

	// --- Observe-only mode: log but always allow (FR17) ---
	if policy.ObserveOnly {
		writeJSON(w, http.StatusOK, hookPermissionResponse{Decision: "allow", Reason: "observe_only mode"})
		return
	}

	// --- Handle denial ---
	if decision.Action == "deny" {
		mgr.RecordDenial(runID, decision, req.ToolName, req.ToolInput)

		// If on_denial=abort, kill the run immediately (FR14).
		agentCfg := s.agentCfgForRun(runID)
		if agentCfg != nil && agentCfg.OnDenial == "abort" {
			_ = mgr.Kill(runID)
		}
	}

	writeJSON(w, http.StatusOK, hookPermissionResponse{
		Decision: decision.Action,
		Reason:   decision.Reason,
	})
}

// extractBearerToken returns the token from "Authorization: Bearer <token>", or "".
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// managerForRun searches all registered project agent managers and returns the
// one that holds a permission policy for the given runID. Returns nil if not found.
func (s *Server) managerForRun(runID string) *agent.Manager {
	for _, p := range s.projects {
		if p.Agents == nil {
			continue
		}
		if _, err := p.Agents.PolicyForRun(runID); err == nil {
			return p.Agents
		}
	}
	return nil
}

// hubForRun returns the per-project hub for the project owning runID.
func (s *Server) hubForRun(runID string) *hub.Hub {
	for _, p := range s.projects {
		if p.Agents == nil {
			continue
		}
		if _, err := p.Agents.PolicyForRun(runID); err == nil {
			return p.Hub
		}
	}
	return nil
}

// agentCfgForRun returns the AgentConfig for the agent running runID so the
// handler can read fields like OnDenial. Returns nil if not found.
func (s *Server) agentCfgForRun(runID string) *agentRunMeta {
	for _, p := range s.projects {
		if p.Agents == nil {
			continue
		}
		row, err := p.Agents.GetRun(runID)
		if err != nil || row == nil {
			continue
		}
		cfg, ok := p.Agents.GetAgent(row.AgentName)
		if !ok {
			continue
		}
		return &agentRunMeta{OnDenial: cfg.OnDenial}
	}
	return nil
}

// agentRunMeta carries the agent config fields needed by the permission handler.
type agentRunMeta struct {
	OnDenial string
}
