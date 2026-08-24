// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/project"
)

func newTestServerWithProviders(t *testing.T, initialProviders []config.Provider, agents []config.AgentConfig) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	appCfg, err := config.LoadApp(cfgPath)
	if err != nil {
		t.Fatalf("LoadApp: %v", err)
	}
	appCfg.Providers = initialProviders
	if err := config.SaveApp(cfgPath, *appCfg); err != nil {
		t.Fatalf("SaveApp: %v", err)
	}

	pCfg := &config.Project{
		Roles:  []string{"devops", "product-owner"},
		Agents: agents,
		Users: []config.UserBinding{
			{Email: "admin@test", Roles: []string{"devops"}},
		},
	}
	p := &project.Project{
		Entry: &config.ProjectEntry{Name: "test-proj", Path: dir},
		Cfg:   pCfg,
	}

	s := &Server{
		appCfg:      appCfg,
		appCfgPath:  cfgPath,
		projects:    map[string]*project.Project{"test-proj": p},
		projectsDir: dir,
		dataDir:     dir,
	}
	return s, cfgPath
}

func authedRequest(method, target string, body any) *http.Request {
	var r *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		r = httptest.NewRequest(method, target, bytes.NewReader(b))
	} else {
		r = httptest.NewRequest(method, target, nil)
	}
	ctx := context.WithValue(r.Context(), userContextKey, &auth.User{Email: "admin@test"})
	return r.WithContext(ctx)
}

func TestProviders_CRUD(t *testing.T) {
	initial := []config.Provider{
		{
			Name:    "local-ollama",
			BaseURL: "http://localhost:11434",
			Driver:  "openai-compatible",
			APIKey:  "secret-ollama-key",
		},
	}
	s, cfgPath := newTestServerWithProviders(t, initial, nil)

	// 1. GET /api/providers -> secret is masked
	{
		r := chi.NewRouter()
		r.Get("/api/providers", s.handleListProviders)

		req := authedRequest(http.MethodGet, "/api/providers", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("GET /api/providers status %d: %s", w.Code, w.Body.String())
		}
		var resp struct {
			Providers []map[string]any `json:"providers"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if len(resp.Providers) != 1 {
			t.Fatalf("expected 1 provider, got %d", len(resp.Providers))
		}
		p0 := resp.Providers[0]
		if p0["name"] != "local-ollama" {
			t.Errorf("name mismatch: %v", p0["name"])
		}
		if p0["api_key"] != "***" {
			t.Errorf("api_key not masked: %v", p0["api_key"])
		}
		if p0["has_api_key"] != true {
			t.Errorf("has_api_key should be true")
		}
	}

	// 2. POST /api/providers -> create new provider
	{
		r := chi.NewRouter()
		r.Post("/api/providers", s.handleCreateProvider)

		createBody := map[string]any{
			"name":     "openrouter",
			"base_url": "https://openrouter.ai/api",
			"driver":   "openai-compatible",
			"api_key":  "sk-or-v1-secret",
			"extra_headers": map[string]string{
				"HTTP-Referer": "https://kaos-control.local",
			},
		}
		req := authedRequest(http.MethodPost, "/api/providers", createBody)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("POST /api/providers status %d: %s", w.Code, w.Body.String())
		}

		// Verify on-disk YAML
		savedCfg, err := config.LoadApp(cfgPath)
		if err != nil {
			t.Fatalf("loading saved app config: %v", err)
		}
		if len(savedCfg.Providers) != 2 {
			t.Fatalf("expected 2 providers on disk, got %d", len(savedCfg.Providers))
		}
		if savedCfg.Providers[1].Name != "openrouter" || savedCfg.Providers[1].APIKey != "sk-or-v1-secret" {
			t.Errorf("saved provider mismatch: %+v", savedCfg.Providers[1])
		}
	}

	// 3. POST /api/providers duplicate -> 409 Conflict
	{
		r := chi.NewRouter()
		r.Post("/api/providers", s.handleCreateProvider)

		createBody := map[string]any{
			"name":     "openrouter",
			"base_url": "https://openrouter.ai/api",
		}
		req := authedRequest(http.MethodPost, "/api/providers", createBody)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict, got %d", w.Code)
		}
	}

	// 4. PUT /api/providers/{name} -> update provider
	{
		r := chi.NewRouter()
		r.Put("/api/providers/{name}", s.handleUpdateProvider)

		updateBody := map[string]any{
			"base_url": "https://openrouter.ai/api/v2",
			"driver":   "openai-compatible",
		}
		req := authedRequest(http.MethodPut, "/api/providers/openrouter", updateBody)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("PUT /api/providers/openrouter status %d: %s", w.Code, w.Body.String())
		}

		savedCfg, _ := config.LoadApp(cfgPath)
		if savedCfg.Providers[1].BaseURL != "https://openrouter.ai/api/v2" {
			t.Errorf("base_url not updated: %s", savedCfg.Providers[1].BaseURL)
		}
		// API key should have been preserved
		if savedCfg.Providers[1].APIKey != "sk-or-v1-secret" {
			t.Errorf("APIKey was not preserved: %s", savedCfg.Providers[1].APIKey)
		}
	}

	// 5. DELETE /api/providers/{name}
	{
		r := chi.NewRouter()
		r.Delete("/api/providers/{name}", s.handleDeleteProvider)

		req := authedRequest(http.MethodDelete, "/api/providers/openrouter", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("DELETE /api/providers/openrouter status %d: %s", w.Code, w.Body.String())
		}

		savedCfg, _ := config.LoadApp(cfgPath)
		if len(savedCfg.Providers) != 1 {
			t.Errorf("expected 1 provider after deletion, got %d", len(savedCfg.Providers))
		}
	}
}

func TestProviders_DeleteReferencedRejected(t *testing.T) {
	initial := []config.Provider{
		{
			Name:    "in-use-provider",
			BaseURL: "http://localhost:11434",
			Driver:  "openai-compatible",
		},
	}
	agents := []config.AgentConfig{
		{
			Name:     "active-agent",
			Provider: "in-use-provider",
		},
	}
	s, _ := newTestServerWithProviders(t, initial, agents)

	r := chi.NewRouter()
	r.Delete("/api/providers/{name}", s.handleDeleteProvider)

	req := authedRequest(http.MethodDelete, "/api/providers/in-use-provider", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for referenced provider, got %d", w.Code)
	}
}

func TestProviders_TestAndModels(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":   "model-a",
						"name": "Model Alpha",
						"supported_parameters": []string{"tools"},
					},
				},
			})
			return
		}

		// Preflight chat completions
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		promptTokens := 10
		if _, hasTools := req["tools"]; hasTools {
			promptTokens = 50
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"usage": map[string]any{
				"prompt_tokens": promptTokens,
			},
		})
	}))
	defer ts.Close()

	initial := []config.Provider{
		{
			Name:    "mock-prov",
			BaseURL: ts.URL,
			Driver:  "openai-compatible",
		},
	}
	s, _ := newTestServerWithProviders(t, initial, nil)

	// Test POST /api/providers/test
	{
		r := chi.NewRouter()
		r.Post("/api/providers/test", s.handleTestProvider)

		testReqBody := map[string]any{
			"name":  "mock-prov",
			"model": "model-a",
		}
		req := authedRequest(http.MethodPost, "/api/providers/test", testReqBody)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("POST /api/providers/test status %d: %s", w.Code, w.Body.String())
		}
		var testResp map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &testResp)
		if testResp["ok"] != true {
			t.Errorf("expected ok: true, got: %+v", testResp)
		}
	}

	// Test GET /api/providers/{name}/models
	{
		r := chi.NewRouter()
		r.Get("/api/providers/{name}/models", s.handleProviderModels)

		req := authedRequest(http.MethodGet, "/api/providers/mock-prov/models", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("GET /api/providers/mock-prov/models status %d: %s", w.Code, w.Body.String())
		}
		var modelsResp struct {
			Models []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"models"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &modelsResp)
		if len(modelsResp.Models) != 1 || modelsResp.Models[0].ID != "model-a" {
			t.Errorf("unexpected models response: %+v", modelsResp)
		}
	}
}
