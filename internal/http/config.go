// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/kaos-control/kaos-control/internal/config"
	"gopkg.in/yaml.v3"
)

const configRelPath = "lifecycle/config.yaml"

// handleGetConfig returns the raw YAML text of lifecycle/config.yaml.
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}
	path := filepath.Join(p.Entry.Path, configRelPath)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		writeJSON(w, http.StatusOK, map[string]any{"raw": ""})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("read_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"raw": string(data)})
}

// handleGetKanbanConfig returns the parsed kanban section of lifecycle/config.yaml as JSON.
// It reloads the config from disk on every request so that edits via the config editor
// are reflected immediately without a server restart.
func (s *Server) handleGetKanbanConfig(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}
	cfg, err := config.LoadProject(p.Entry.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("config_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"kanban": cfg.Kanban})
}

// handleGetRoadmapConfig returns the parsed roadmap section of lifecycle/config.yaml as JSON.
// It reloads the config from disk on every request so that edits are reflected immediately.
func (s *Server) handleGetRoadmapConfig(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}
	cfg, err := config.LoadProject(p.Entry.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("config_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"roadmap": cfg.Roadmap})
}

// handleGetOpenQuestionsConfig returns the parsed open_questions section of
// lifecycle/config.yaml as JSON. It reloads the config from disk on every
// request so that edits via the config editor are reflected immediately.
func (s *Server) handleGetOpenQuestionsConfig(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}
	cfg, err := config.LoadProject(p.Entry.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("config_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"answer_format": cfg.OpenQuestions.EffectiveFormat()})
}

// handleGetConfigHealth returns the config self-repair notes recorded the
// last time this project's lifecycle/config.yaml was loaded (see
// config.Project.ValidateAndRepair). An empty "repairs" list means the
// on-disk config already satisfies the required generation capabilities.
// Never includes secret fields (e.g. auth_token) — RepairNote only carries
// agent name, template key, and reason.
func (s *Server) handleGetConfigHealth(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}
	notes := p.Config().RepairNotes
	repairs := make([]map[string]string, 0, len(notes))
	for _, n := range notes {
		repairs = append(repairs, map[string]string{
			"agent":        n.Agent,
			"template_key": n.TemplateKey,
			"reason":       n.Reason,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"repairs": repairs})
}

// handleUpdateConfig validates and writes lifecycle/config.yaml.
func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}
	if !requireRole(w, r, p, RolesAdminOnly...) {
		return
	}

	var body struct {
		Raw string `json:"raw"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", err.Error()))
		return
	}

	// Validate: must parse as valid YAML.
	var probe any
	if err := yaml.Unmarshal([]byte(body.Raw), &probe); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, apiError("invalid_yaml", err.Error()))
		return
	}

	path := filepath.Join(p.Entry.Path, configRelPath)
	if err := os.WriteFile(path, []byte(body.Raw), 0o644); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("write_error", err.Error()))
		return
	}

	// Reload the live config (agent roster, roles, etc.) so the write takes
	// effect immediately rather than waiting for the debounced watcher event
	// or a restart. The file is already written at this point; if the new
	// config fails full validation (e.g. an agent missing a required field),
	// the previous config stays active and the caller is told reload failed
	// so they know to fix and re-save rather than assume it's live.
	resp := map[string]any{"ok": true}
	if err := p.ReloadConfig(); err != nil {
		resp["reload_error"] = err.Error()
	}
	writeJSON(w, http.StatusOK, resp)
}
