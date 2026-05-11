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
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
