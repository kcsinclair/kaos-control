package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/config"
)

// ----- helpers -----

// maskedInstances returns a copy of instances with api_key replaced by "***" when set.
func maskedInstances(instances []config.OllamaInstance) []map[string]any {
	out := make([]map[string]any, len(instances))
	for i, inst := range instances {
		m := map[string]any{
			"name":     inst.Name,
			"base_url": inst.BaseURL,
		}
		if inst.APIKey != "" {
			m["api_key"] = "***"
		}
		out[i] = m
	}
	return out
}

// findOllamaInstance returns the index of the named instance, or -1.
func findOllamaInstance(instances []config.OllamaInstance, name string) int {
	for i, inst := range instances {
		if inst.Name == name {
			return i
		}
	}
	return -1
}

// isInstanceReferencedByProjects checks whether any project agent uses the named instance.
func (s *Server) isInstanceReferencedByProjects(name string) bool {
	for _, p := range s.projects {
		for _, ag := range p.Cfg.Agents {
			if ag.OllamaInstanceName == name {
				return true
			}
		}
	}
	return false
}

// ----- CRUD handlers -----

// handleListOllamaInstances returns all registered Ollama instances with api_key masked.
func (s *Server) handleListOllamaInstances(w http.ResponseWriter, r *http.Request) {
	if s.appCfg == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError("not_configured", "app config unavailable"))
		return
	}
	s.appCfgMu.RLock()
	instances := make([]config.OllamaInstance, len(s.appCfg.OllamaInstances))
	copy(instances, s.appCfg.OllamaInstances)
	s.appCfgMu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{"instances": maskedInstances(instances)})
}

// handleCreateOllamaInstance adds a new Ollama instance to app config.
func (s *Server) handleCreateOllamaInstance(w http.ResponseWriter, r *http.Request) {
	if s.appCfg == nil || s.appCfgPath == "" {
		writeJSON(w, http.StatusServiceUnavailable, apiError("not_configured", "app config unavailable"))
		return
	}

	var req struct {
		Name    string `json:"name"`
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "name is required"))
		return
	}
	if req.BaseURL == "" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "base_url is required"))
		return
	}
	if u, err := url.ParseRequestURI(req.BaseURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", fmt.Sprintf("base_url %q is not a valid http/https URL", req.BaseURL)))
		return
	}

	s.appCfgMu.Lock()
	defer s.appCfgMu.Unlock()

	if findOllamaInstance(s.appCfg.OllamaInstances, req.Name) >= 0 {
		writeJSON(w, http.StatusConflict, apiError("conflict", fmt.Sprintf("instance %q already exists", req.Name)))
		return
	}

	inst := config.OllamaInstance{Name: req.Name, BaseURL: req.BaseURL, APIKey: req.APIKey}
	s.appCfg.OllamaInstances = append(s.appCfg.OllamaInstances, inst)

	if err := config.SaveApp(s.appCfgPath, *s.appCfg); err != nil {
		// Rollback in-memory change.
		s.appCfg.OllamaInstances = s.appCfg.OllamaInstances[:len(s.appCfg.OllamaInstances)-1]
		writeJSON(w, http.StatusInternalServerError, apiError("save_error", err.Error()))
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"instance": maskedInstances([]config.OllamaInstance{inst})[0]})
}

// handleUpdateOllamaInstance updates an existing Ollama instance.
func (s *Server) handleUpdateOllamaInstance(w http.ResponseWriter, r *http.Request) {
	if s.appCfg == nil || s.appCfgPath == "" {
		writeJSON(w, http.StatusServiceUnavailable, apiError("not_configured", "app config unavailable"))
		return
	}

	name := chi.URLParam(r, "name")

	var req struct {
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}
	if req.BaseURL == "" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "base_url is required"))
		return
	}
	if u, err := url.ParseRequestURI(req.BaseURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", fmt.Sprintf("base_url %q is not a valid http/https URL", req.BaseURL)))
		return
	}

	s.appCfgMu.Lock()
	defer s.appCfgMu.Unlock()

	idx := findOllamaInstance(s.appCfg.OllamaInstances, name)
	if idx < 0 {
		writeJSON(w, http.StatusNotFound, apiError("not_found", fmt.Sprintf("instance %q not found", name)))
		return
	}

	old := s.appCfg.OllamaInstances[idx]
	s.appCfg.OllamaInstances[idx] = config.OllamaInstance{Name: name, BaseURL: req.BaseURL, APIKey: req.APIKey}

	if err := config.SaveApp(s.appCfgPath, *s.appCfg); err != nil {
		s.appCfg.OllamaInstances[idx] = old
		writeJSON(w, http.StatusInternalServerError, apiError("save_error", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"instance": maskedInstances([]config.OllamaInstance{s.appCfg.OllamaInstances[idx]})[0]})
}

// handleDeleteOllamaInstance removes an Ollama instance, rejecting if it is referenced.
func (s *Server) handleDeleteOllamaInstance(w http.ResponseWriter, r *http.Request) {
	if s.appCfg == nil || s.appCfgPath == "" {
		writeJSON(w, http.StatusServiceUnavailable, apiError("not_configured", "app config unavailable"))
		return
	}

	name := chi.URLParam(r, "name")

	if s.isInstanceReferencedByProjects(name) {
		writeJSON(w, http.StatusConflict, apiError("conflict", fmt.Sprintf("instance %q is referenced by one or more project agents", name)))
		return
	}

	s.appCfgMu.Lock()
	defer s.appCfgMu.Unlock()

	idx := findOllamaInstance(s.appCfg.OllamaInstances, name)
	if idx < 0 {
		writeJSON(w, http.StatusNotFound, apiError("not_found", fmt.Sprintf("instance %q not found", name)))
		return
	}

	removed := s.appCfg.OllamaInstances[idx]
	s.appCfg.OllamaInstances = append(s.appCfg.OllamaInstances[:idx], s.appCfg.OllamaInstances[idx+1:]...)

	if err := config.SaveApp(s.appCfgPath, *s.appCfg); err != nil {
		// Rollback.
		s.appCfg.OllamaInstances = append(s.appCfg.OllamaInstances[:idx], append([]config.OllamaInstance{removed}, s.appCfg.OllamaInstances[idx:]...)...)
		writeJSON(w, http.StatusInternalServerError, apiError("save_error", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted": name})
}

// ----- Health & Models -----

// handleOllamaHealth proxies GET {base_url}/api/tags with a 10-second timeout
// and returns {"ok": true, "latency_ms": N} or {"ok": false, "error": "..."}.
func (s *Server) handleOllamaHealth(w http.ResponseWriter, r *http.Request) {
	inst, ok := s.resolveOllamaInstance(w, chi.URLParam(r, "name"))
	if !ok {
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, inst.BaseURL+"/api/tags", nil)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if inst.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+inst.APIKey)
	}

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "latency_ms": latency})
	} else {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": fmt.Sprintf("unexpected status %d", resp.StatusCode)})
	}
}

// handleOllamaModels proxies GET {base_url}/api/tags and extracts model list.
func (s *Server) handleOllamaModels(w http.ResponseWriter, r *http.Request) {
	inst, ok := s.resolveOllamaInstance(w, chi.URLParam(r, "name"))
	if !ok {
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, inst.BaseURL+"/api/tags", nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("request_error", err.Error()))
		return
	}
	if inst.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+inst.APIKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, apiError("upstream_error", err.Error()))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		writeJSON(w, http.StatusBadGateway, apiError("upstream_error", fmt.Sprintf("Ollama returned status %d", resp.StatusCode)))
		return
	}

	var tagsResp struct {
		Models []struct {
			Name    string `json:"name"`
			Size    int64  `json:"size"`
			Details struct {
				ParameterSize string `json:"parameter_size"`
			} `json:"details"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		writeJSON(w, http.StatusBadGateway, apiError("parse_error", "could not parse Ollama response: "+err.Error()))
		return
	}

	type modelOut struct {
		Name string `json:"name"`
		Size int64  `json:"size"`
	}
	out := make([]modelOut, len(tagsResp.Models))
	for i, m := range tagsResp.Models {
		out[i] = modelOut{Name: m.Name, Size: m.Size}
	}
	writeJSON(w, http.StatusOK, map[string]any{"models": out})
}

// resolveOllamaInstance looks up the named instance; writes 404 and returns false on miss.
func (s *Server) resolveOllamaInstance(w http.ResponseWriter, name string) (config.OllamaInstance, bool) {
	if s.appCfg == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError("not_configured", "app config unavailable"))
		return config.OllamaInstance{}, false
	}
	s.appCfgMu.RLock()
	defer s.appCfgMu.RUnlock()
	idx := findOllamaInstance(s.appCfg.OllamaInstances, name)
	if idx < 0 {
		writeJSON(w, http.StatusNotFound, apiError("not_found", fmt.Sprintf("instance %q not found", name)))
		return config.OllamaInstance{}, false
	}
	return s.appCfg.OllamaInstances[idx], true
}
