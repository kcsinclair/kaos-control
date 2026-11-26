// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/agent"
	"github.com/kaos-control/kaos-control/internal/config"
)

// ----- helpers -----

// maskedProviders returns a copy of providers with api_key masked for client consumption.
func maskedProviders(providers []config.Provider) []map[string]any {
	out := make([]map[string]any, len(providers))
	for i, p := range providers {
		m := map[string]any{
			"name":          p.Name,
			"base_url":      p.BaseURL,
			"driver":        p.Driver,
			"has_api_key":   p.APIKey != "",
			"extra_headers": p.ExtraHeaders,
		}
		if p.APIKey != "" {
			m["api_key"] = "***"
		}
		out[i] = m
	}
	return out
}

// findProvider returns the index of the named provider, or -1.
func findProvider(providers []config.Provider, name string) int {
	for i, p := range providers {
		if p.Name == name {
			return i
		}
	}
	return -1
}

// findProviderReferences checks whether any project agent uses the named provider,
// returning formatted reference strings.
func (s *Server) findProviderReferences(name string) []string {
	var refs []string
	s.projectsMu.RLock()
	defer s.projectsMu.RUnlock()
	for pName, p := range s.projects {
		for _, ag := range p.Config().Agents {
			if ag.Provider == name {
				refs = append(refs, fmt.Sprintf("project %q (agent %q)", pName, ag.Name))
			}
		}
	}
	return refs
}

// ----- CRUD Handlers -----

// handleListProviders returns all registered providers with secret fields masked (NFR-1).
func (s *Server) handleListProviders(w http.ResponseWriter, r *http.Request) {
	if s.appCfg == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError("not_configured", "app config unavailable"))
		return
	}
	s.appCfgMu.RLock()
	providers := make([]config.Provider, len(s.appCfg.Providers))
	copy(providers, s.appCfg.Providers)
	s.appCfgMu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{"providers": maskedProviders(providers)})
}

// handleGetProvider returns a single registered provider with secret fields masked (NFR-1).
func (s *Server) handleGetProvider(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	prov, ok := s.resolveProvider(w, name)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"provider": maskedProviders([]config.Provider{prov})[0]})
}

// handleCreateProvider registers a new provider in app config.
func (s *Server) handleCreateProvider(w http.ResponseWriter, r *http.Request) {
	if s.appCfg == nil || s.appCfgPath == "" {
		writeJSON(w, http.StatusServiceUnavailable, apiError("not_configured", "app config unavailable"))
		return
	}
	if !s.requireAppRole(w, r, RolesDevopsOrAdmin...) {
		return
	}

	var req struct {
		Name         string            `json:"name"`
		BaseURL      string            `json:"base_url"`
		Driver       string            `json:"driver"`
		APIKey       string            `json:"api_key"`
		ExtraHeaders map[string]string `json:"extra_headers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "name is required"))
		return
	}
	if err := config.ValidateProviderSlug(req.Name); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", fmt.Sprintf("invalid provider name %q: %v", req.Name, err)))
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

	if req.Driver == "" {
		req.Driver = "openai-compatible"
	}
	if req.Driver != "openai-compatible" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", fmt.Sprintf("unsupported driver %q: only openai-compatible is supported", req.Driver)))
		return
	}

	s.appCfgMu.Lock()
	defer s.appCfgMu.Unlock()

	if findProvider(s.appCfg.Providers, req.Name) >= 0 {
		writeJSON(w, http.StatusConflict, apiError("conflict", fmt.Sprintf("provider %q already exists", req.Name)))
		return
	}

	prov := config.Provider{
		Name:         req.Name,
		BaseURL:      req.BaseURL,
		Driver:       req.Driver,
		APIKey:       req.APIKey,
		ExtraHeaders: req.ExtraHeaders,
	}
	s.appCfg.Providers = append(s.appCfg.Providers, prov)

	if err := config.SaveApp(s.appCfgPath, *s.appCfg); err != nil {
		// Rollback in-memory change.
		s.appCfg.Providers = s.appCfg.Providers[:len(s.appCfg.Providers)-1]
		writeJSON(w, http.StatusInternalServerError, apiError("save_error", err.Error()))
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"provider": maskedProviders([]config.Provider{prov})[0]})
}

// handleUpdateProvider updates an existing provider in app config.
func (s *Server) handleUpdateProvider(w http.ResponseWriter, r *http.Request) {
	if s.appCfg == nil || s.appCfgPath == "" {
		writeJSON(w, http.StatusServiceUnavailable, apiError("not_configured", "app config unavailable"))
		return
	}
	if !s.requireAppRole(w, r, RolesDevopsOrAdmin...) {
		return
	}

	name := chi.URLParam(r, "name")

	var req struct {
		BaseURL      string            `json:"base_url"`
		Driver       string            `json:"driver"`
		APIKey       *string           `json:"api_key"`
		ExtraHeaders map[string]string `json:"extra_headers"`
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
	if req.Driver == "" {
		req.Driver = "openai-compatible"
	}
	if req.Driver != "openai-compatible" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", fmt.Sprintf("unsupported driver %q: only openai-compatible is supported", req.Driver)))
		return
	}

	s.appCfgMu.Lock()
	defer s.appCfgMu.Unlock()

	idx := findProvider(s.appCfg.Providers, name)
	if idx < 0 {
		writeJSON(w, http.StatusNotFound, apiError("not_found", fmt.Sprintf("provider %q not found", name)))
		return
	}

	old := s.appCfg.Providers[idx]
	apiKey := old.APIKey
	if req.APIKey != nil {
		if *req.APIKey != "***" {
			apiKey = *req.APIKey
		}
	}

	updated := config.Provider{
		Name:         name,
		BaseURL:      req.BaseURL,
		Driver:       req.Driver,
		APIKey:       apiKey,
		ExtraHeaders: req.ExtraHeaders,
	}
	s.appCfg.Providers[idx] = updated

	if err := config.SaveApp(s.appCfgPath, *s.appCfg); err != nil {
		s.appCfg.Providers[idx] = old
		writeJSON(w, http.StatusInternalServerError, apiError("save_error", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"provider": maskedProviders([]config.Provider{updated})[0]})
}

// handleDeleteProvider removes a provider, rejecting if it is in use by any project agent.
func (s *Server) handleDeleteProvider(w http.ResponseWriter, r *http.Request) {
	if s.appCfg == nil || s.appCfgPath == "" {
		writeJSON(w, http.StatusServiceUnavailable, apiError("not_configured", "app config unavailable"))
		return
	}
	if !s.requireAppRole(w, r, RolesDevopsOrAdmin...) {
		return
	}

	name := chi.URLParam(r, "name")

	if refs := s.findProviderReferences(name); len(refs) > 0 {
		writeJSON(w, http.StatusConflict, apiError("conflict", fmt.Sprintf("cannot delete provider %q: in use by %s", name, strings.Join(refs, ", "))))
		return
	}

	s.appCfgMu.Lock()
	defer s.appCfgMu.Unlock()

	idx := findProvider(s.appCfg.Providers, name)
	if idx < 0 {
		writeJSON(w, http.StatusNotFound, apiError("not_found", fmt.Sprintf("provider %q not found", name)))
		return
	}

	removed := s.appCfg.Providers[idx]
	s.appCfg.Providers = append(s.appCfg.Providers[:idx], s.appCfg.Providers[idx+1:]...)

	if err := config.SaveApp(s.appCfgPath, *s.appCfg); err != nil {
		s.appCfg.Providers = append(s.appCfg.Providers[:idx], append([]config.Provider{removed}, s.appCfg.Providers[idx:]...)...)
		writeJSON(w, http.StatusInternalServerError, apiError("save_error", err.Error()))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ----- Connectivity & Models -----

// handleTestProvider tests connectivity and tool-calling capability for a provider.
func (s *Server) handleTestProvider(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string            `json:"name"`
		BaseURL      string            `json:"base_url"`
		Driver       string            `json:"driver"`
		APIKey       string            `json:"api_key"`
		ExtraHeaders map[string]string `json:"extra_headers"`
		Model        string            `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}

	var prov config.Provider
	if req.BaseURL != "" {
		prov = config.Provider{
			Name:         req.Name,
			BaseURL:      req.BaseURL,
			Driver:       req.Driver,
			APIKey:       req.APIKey,
			ExtraHeaders: req.ExtraHeaders,
		}
	} else if req.Name != "" {
		p, ok := s.resolveProvider(w, req.Name)
		if !ok {
			return
		}
		prov = p
	} else {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "name or base_url is required"))
		return
	}

	client := &http.Client{Timeout: 15 * time.Second}
	start := time.Now()

	// Testing a *connection* means asking the provider what it serves — GET
	// /v1/models. That validates base_url, credentials and reachability without
	// needing a model name at all.
	//
	// This previously POSTed a tool-calling chat completion using a hardcoded
	// "test-model" placeholder whenever no model was supplied. Providers answer
	// an unknown model with HTTP 400, which the preflight then reported as
	// "model does not support tools" — so testing a perfectly good provider
	// produced a tools error naming a model the user never entered.
	models, err := listProviderModels(r.Context(), client, prov)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":         false,
			"error":      err.Error(),
			"latency_ms": latency,
		})
		return
	}

	// No model named: the connection itself is what was under test.
	if req.Model == "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":         true,
			"latency_ms": latency,
			"models":     models,
			"message":    fmt.Sprintf("Connected to %s — %d model(s) available", prov.Name, len(models)),
		})
		return
	}

	// A model was named: confirm the provider actually serves it before making
	// any claim about tool support, so a typo reads as "not found" rather than
	// "does not support tools".
	found := false
	for _, m := range models {
		if m == req.Model {
			found = true
			break
		}
	}
	if !found {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":         false,
			"latency_ms": latency,
			"models":     models,
			"error": fmt.Sprintf("model %q not found on provider %q (%d model(s) available)",
				req.Model, prov.Name, len(models)),
		})
		return
	}

	if err := agent.VerifyToolCapability(r.Context(), client, prov, req.Model, agent.DefaultOpenAITools()); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":         false,
			"latency_ms": time.Since(start).Milliseconds(),
			"models":     models,
			"error":      err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"latency_ms": time.Since(start).Milliseconds(),
		"models":     models,
		"message":    fmt.Sprintf("Successfully connected and verified tool support on %s with model %s", prov.Name, req.Model),
	})
}

// listProviderModels performs the GET <base_url>/v1/models listing used by the
// connection test. Returns the model IDs the provider advertises.
func listProviderModels(ctx context.Context, client *http.Client, prov config.Provider) ([]string, error) {
	modelsURL := prov.BaseURL + "/v1/models"
	if strings.HasSuffix(prov.BaseURL, "/v1") {
		modelsURL = prov.BaseURL + "/models"
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request to %s: %w", modelsURL, err)
	}
	if prov.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+prov.APIKey)
	}
	for k, v := range prov.ExtraHeaders {
		httpReq.Header.Set(k, v)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("could not reach %s: %w", modelsURL, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("provider returned HTTP %d from %s: %s",
			resp.StatusCode, modelsURL, strings.TrimSpace(string(body)))
	}

	var listing struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &listing); err != nil {
		return nil, fmt.Errorf("could not parse /v1/models response from %s: %w", prov.Name, err)
	}

	ids := make([]string, 0, len(listing.Data))
	for _, m := range listing.Data {
		ids = append(ids, m.ID)
	}
	return ids, nil
}

// handleProviderHealth checks connectivity to the provider with a 5s timeout.
func (s *Server) handleProviderHealth(w http.ResponseWriter, r *http.Request) {
	prov, ok := s.resolveProvider(w, chi.URLParam(r, "name"))
	if !ok {
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	modelsURL := prov.BaseURL + "/v1/models"
	if strings.HasSuffix(prov.BaseURL, "/v1") {
		modelsURL = prov.BaseURL + "/models"
	}

	httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, modelsURL, nil)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"healthy": false, "error": err.Error(), "latency_ms": 0})
		return
	}
	if prov.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+prov.APIKey)
	}
	for k, v := range prov.ExtraHeaders {
		httpReq.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := client.Do(httpReq)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"healthy":    false,
			"error":      err.Error(),
			"latency_ms": latency,
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		writeJSON(w, http.StatusOK, map[string]any{
			"healthy":    false,
			"error":      fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
			"latency_ms": latency,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"healthy":    true,
		"latency_ms": latency,
	})
}

// handleProviderModels queries /v1/models from the provider and returns model list.
func (s *Server) handleProviderModels(w http.ResponseWriter, r *http.Request) {
	prov, ok := s.resolveProvider(w, chi.URLParam(r, "name"))
	if !ok {
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	modelsURL := prov.BaseURL + "/v1/models"
	if strings.HasSuffix(prov.BaseURL, "/v1") {
		modelsURL = prov.BaseURL + "/models"
	}

	httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, modelsURL, nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("request_error", err.Error()))
		return
	}
	if prov.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+prov.APIKey)
	}
	for k, v := range prov.ExtraHeaders {
		httpReq.Header.Set(k, v)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, apiError("upstream_error", err.Error()))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		writeJSON(w, http.StatusBadGateway, apiError("upstream_error", fmt.Sprintf("provider returned status %d: %s", resp.StatusCode, string(body))))
		return
	}

	var modelsResp struct {
		Data []struct {
			ID                  string   `json:"id"`
			Name                string   `json:"name"`
			SupportedParameters []string `json:"supported_parameters,omitempty"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		writeJSON(w, http.StatusBadGateway, apiError("parse_error", "could not parse provider models response: "+err.Error()))
		return
	}

	type modelOut struct {
		ID                  string   `json:"id"`
		Name                string   `json:"name"`
		SupportedParameters []string `json:"supported_parameters,omitempty"`
	}
	out := make([]modelOut, len(modelsResp.Data))
	for i, m := range modelsResp.Data {
		name := m.Name
		if name == "" {
			name = m.ID
		}
		out[i] = modelOut{
			ID:                  m.ID,
			Name:                name,
			SupportedParameters: m.SupportedParameters,
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"models": out})
}

// resolveProvider looks up the named provider; writes 404 and returns false on miss.
func (s *Server) resolveProvider(w http.ResponseWriter, name string) (config.Provider, bool) {
	if s.appCfg == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError("not_configured", "app config unavailable"))
		return config.Provider{}, false
	}
	s.appCfgMu.RLock()
	defer s.appCfgMu.RUnlock()
	idx := findProvider(s.appCfg.Providers, name)
	if idx < 0 {
		writeJSON(w, http.StatusNotFound, apiError("not_found", fmt.Sprintf("provider %q not found", name)))
		return config.Provider{}, false
	}
	return s.appCfg.Providers[idx], true
}
