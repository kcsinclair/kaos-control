// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/project"
)

const providerSwitchHTTPTestConfig = `stages:
  - {name: ideas, dir: ideas}
git:
  default_branch: main
roles:
  - analyst
  - product-owner
  - devops
users:
  - email: po@test
    roles: [product-owner]
  - email: viewer@test
    roles: [analyst]
agents:
  - name: analyst-agent
    role: [analyst]
    driver: openai-compatible
    provider: anthropic-cloud
    model: claude-3-7-sonnet
    fallback_provider: gemini-cloud
    fallback_model: gemini-2.5-flash
    prompt_templates:
      analyst: "x"
  - name: failed-over-agent
    role: [analyst]
    driver: openai-compatible
    provider: gemini-cloud
    model: gemini-2.5-flash
    primary_provider: anthropic-cloud
    primary_model: claude-3-7-sonnet
    prompt_templates:
      analyst: "x"
provider_templates:
  - name: local-ai
    agents:
      analyst-agent: {provider: local-ollama, model: llama3}
`

// newTestProjectForProviderSwitch opens a real project (real git repo +
// on-disk lifecycle/config.yaml) via project.Open, since the provider-switch
// handlers patch and re-read lifecycle/config.yaml from disk — the in-memory
// &project.Project{Cfg: ...} harness used elsewhere in this package doesn't
// have a file to patch.
func newTestProjectForProviderSwitch(t *testing.T) *project.Project {
	t.Helper()
	dir := t.TempDir()

	gr, err := gogit.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "lifecycle", "ideas"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lifecycle", "config.yaml"), []byte(providerSwitchHTTPTestConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	wt, err := gr.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Add("lifecycle/config.yaml"); err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Commit("initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	}); err != nil {
		t.Fatal(err)
	}

	entry := &config.ProjectEntry{Name: "provider-switch-http-test", Path: dir}
	dbDir := t.TempDir()
	p, err := project.Open(entry, dbDir, project.OpenOptions{SkipArchitectureScaffold: true})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = p.Close() })
	return p
}

func testServerWithAppProviders() *Server {
	return &Server{
		appCfg: &config.App{
			Providers: []config.Provider{
				{Name: "anthropic-cloud", BaseURL: "https://api.anthropic.example", Driver: "openai-compatible"},
				{Name: "gemini-cloud", BaseURL: "https://gemini.example", Driver: "openai-compatible"},
				{Name: "local-ollama", BaseURL: "http://localhost:11434", Driver: "openai-compatible"},
			},
		},
	}
}

func withChiParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestHandleGetFailoverStatus(t *testing.T) {
	p := newTestProjectForProviderSwitch(t)
	s := testServerWithAppProviders()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withProjectAndUser(req, p, "po@test")
	w := httptest.NewRecorder()
	s.handleGetFailoverStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		FailoverActive bool                        `json:"failover_active"`
		Agents         []providerSwitchAgentStatus `json:"agents"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.FailoverActive {
		t.Error("expected failover_active=true (failed-over-agent is in failover)")
	}
	var analyst, failedOver *providerSwitchAgentStatus
	for i := range resp.Agents {
		switch resp.Agents[i].Agent {
		case "analyst-agent":
			analyst = &resp.Agents[i]
		case "failed-over-agent":
			failedOver = &resp.Agents[i]
		}
	}
	if analyst == nil || analyst.IsFailover {
		t.Fatalf("expected analyst-agent not in failover, got %+v", analyst)
	}
	if failedOver == nil || !failedOver.IsFailover {
		t.Fatalf("expected failed-over-agent in failover, got %+v", failedOver)
	}
	if failedOver.PrimaryProvider != "anthropic-cloud" || failedOver.ActiveProvider != "gemini-cloud" {
		t.Errorf("unexpected failover fields: %+v", failedOver)
	}
}

func TestHandleAgentSwitchProvider(t *testing.T) {
	p := newTestProjectForProviderSwitch(t)
	s := testServerWithAppProviders()

	body, _ := json.Marshal(map[string]string{
		"provider": "local-ollama",
		"model":    "llama3",
		"reason":   "manual test",
	})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req = withProjectAndUser(req, p, "po@test")
	req = withChiParam(req, "name", "analyst-agent")
	w := httptest.NewRecorder()
	s.handleAgentSwitchProvider(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, body: %s", w.Code, w.Body.String())
	}

	ag, ok := p.Agents.GetAgent("analyst-agent")
	if !ok {
		t.Fatal("analyst-agent not found after switch")
	}
	if ag.Provider != "local-ollama" || ag.Model != "llama3" {
		t.Errorf("expected switched provider/model, got %+v", ag)
	}
}

func TestHandleAgentSwitchProvider_UnknownProviderRejected(t *testing.T) {
	p := newTestProjectForProviderSwitch(t)
	s := testServerWithAppProviders()

	body, _ := json.Marshal(map[string]string{"provider": "does-not-exist", "model": "m"})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req = withProjectAndUser(req, p, "po@test")
	req = withChiParam(req, "name", "analyst-agent")
	w := httptest.NewRecorder()
	s.handleAgentSwitchProvider(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400, body: %s", w.Code, w.Body.String())
	}
}

func TestHandleAgentSwitchProvider_RoleGating(t *testing.T) {
	p := newTestProjectForProviderSwitch(t)
	s := testServerWithAppProviders()

	body, _ := json.Marshal(map[string]string{"provider": "local-ollama", "model": "llama3"})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req = withProjectAndUser(req, p, "viewer@test") // analyst role only, not devops/product-owner
	req = withChiParam(req, "name", "analyst-agent")
	w := httptest.NewRecorder()
	s.handleAgentSwitchProvider(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403, body: %s", w.Code, w.Body.String())
	}
}

func TestHandleAgentRestoreProvider(t *testing.T) {
	p := newTestProjectForProviderSwitch(t)
	s := testServerWithAppProviders()

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = withProjectAndUser(req, p, "po@test")
	req = withChiParam(req, "name", "failed-over-agent")
	w := httptest.NewRecorder()
	s.handleAgentRestoreProvider(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, body: %s", w.Code, w.Body.String())
	}

	ag, ok := p.Agents.GetAgent("failed-over-agent")
	if !ok {
		t.Fatal("failed-over-agent not found after restore")
	}
	if ag.Provider != "anthropic-cloud" || ag.Model != "claude-3-7-sonnet" {
		t.Errorf("expected restored to primary, got %+v", ag)
	}
	if ag.PrimaryProvider != "" || ag.PrimaryModel != "" {
		t.Errorf("expected primary fields cleared, got %+v", ag)
	}
}

func TestHandleAgentRestoreProvider_NotInFailover(t *testing.T) {
	p := newTestProjectForProviderSwitch(t)
	s := testServerWithAppProviders()

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = withProjectAndUser(req, p, "po@test")
	req = withChiParam(req, "name", "analyst-agent")
	w := httptest.NewRecorder()
	s.handleAgentRestoreProvider(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status: got %d, want 409, body: %s", w.Code, w.Body.String())
	}
}

func TestHandleRestoreAllProviders(t *testing.T) {
	p := newTestProjectForProviderSwitch(t)
	s := testServerWithAppProviders()

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = withProjectAndUser(req, p, "po@test")
	w := httptest.NewRecorder()
	s.handleRestoreAllProviders(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		OK             bool `json:"ok"`
		RestoredAgents int  `json:"restored_agents"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.RestoredAgents != 1 {
		t.Errorf("expected 1 restored agent, got %d", resp.RestoredAgents)
	}

	ag, _ := p.Agents.GetAgent("failed-over-agent")
	if ag.PrimaryProvider != "" {
		t.Errorf("expected failed-over-agent restored, got %+v", ag)
	}
	other, _ := p.Agents.GetAgent("analyst-agent")
	if other.Provider != "anthropic-cloud" {
		t.Errorf("expected analyst-agent (not in failover) untouched, got %+v", other)
	}
}

func TestHandleApplyProviderTemplate(t *testing.T) {
	p := newTestProjectForProviderSwitch(t)
	s := testServerWithAppProviders()

	body, _ := json.Marshal(map[string]string{"template": "local-ai"})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req = withProjectAndUser(req, p, "po@test")
	w := httptest.NewRecorder()
	s.handleApplyProviderTemplate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		OK            bool   `json:"ok"`
		Template      string `json:"template"`
		UpdatedAgents int    `json:"updated_agents"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.UpdatedAgents != 1 {
		t.Errorf("expected 1 updated agent, got %d", resp.UpdatedAgents)
	}

	ag, _ := p.Agents.GetAgent("analyst-agent")
	if ag.Provider != "local-ollama" || ag.Model != "llama3" {
		t.Errorf("expected template-applied provider/model, got %+v", ag)
	}
}

func TestHandleApplyProviderTemplate_RoleGating(t *testing.T) {
	p := newTestProjectForProviderSwitch(t)
	s := testServerWithAppProviders()

	body, _ := json.Marshal(map[string]string{"template": "local-ai"})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req = withProjectAndUser(req, p, "viewer@test")
	w := httptest.NewRecorder()
	s.handleApplyProviderTemplate(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403, body: %s", w.Code, w.Body.String())
	}
}

func TestHandleListProviderTemplates(t *testing.T) {
	p := newTestProjectForProviderSwitch(t)
	s := testServerWithAppProviders()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withProjectAndUser(req, p, "po@test")
	w := httptest.NewRecorder()
	s.handleListProviderTemplates(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "local-ai") {
		t.Errorf("expected local-ai template in response, got %s", w.Body.String())
	}
}
