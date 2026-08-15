// SPDX-License-Identifier: AGPL-3.0-or-later

package http

// T-5 (API surface) — secret hygiene for GET /api/p/:project/agents.
// Verifies that the response includes driver, model, and base_url for a
// claude-env agent, and that the auth_token never appears in the JSON body.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/agent"
	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
	"github.com/kaos-control/kaos-control/internal/lock"
	"github.com/kaos-control/kaos-control/internal/project"
	"github.com/kaos-control/kaos-control/internal/workflow"
)

// newTestProjectWithClaudeEnvAgent creates a minimal project backed by a real
// SQLite index and an agent.Manager configured with one claude-env agent.
func newTestProjectWithClaudeEnvAgent(t *testing.T, ag config.AgentConfig) (*project.Project, func()) {
	t.Helper()
	dir := t.TempDir()

	h := hub.New()
	wf := workflow.New(nil)
	idx, err := index.Open(filepath.Join(dir, "agents-test.db"), dir, nil,
		index.WithHub(h),
		index.WithWorkflow(wf),
	)
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}

	locks := lock.New(idx, h)
	cfg := &config.Project{
		Roles:  []string{"analyst", "product-owner"},
		Agents: []config.AgentConfig{ag},
		Users: []config.UserBinding{
			{Email: "po@test", Roles: []string{"product-owner"}},
		},
	}
	entry := &config.ProjectEntry{Name: "test", Path: dir}

	mgr := agent.New(
		[]config.AgentConfig{ag},
		4,
		idx,
		nil,
		h,
		locks,
		nil,
		dir,
		"",
		nil,
		config.AppAgentConfig{},
	)

	p := &project.Project{
		Entry:    entry,
		Cfg:      cfg,
		Idx:      idx,
		Hub:      h,
		Workflow: wf,
		Agents:   mgr,
	}
	return p, func() { idx.Close() }
}

// TestHandleListAgents_ClaudeEnvSecretHygiene verifies that GET /agents for a
// project with a claude-env agent returns driver/model/base_url but never
// exposes auth_token or its value in the JSON response (T-5 NFR-1).
func TestHandleListAgents_ClaudeEnvSecretHygiene(t *testing.T) {
	const token = "s3cr3t-token-must-not-appear"

	ag := config.AgentConfig{
		Name:      "claude-env-agent",
		Roles:     []string{"analyst"},
		Driver:    "claude-env",
		Model:     "claude-opus-4-6",
		BaseURL:   "http://localhost:11434",
		AuthToken: token,
		PromptTemplates: map[string]string{
			"analyst": "analyse {target_path}",
		},
	}

	p, cleanup := newTestProjectWithClaudeEnvAgent(t, ag)
	defer cleanup()

	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withProjectAndUser(req, p, "po@test")

	w := httptest.NewRecorder()
	s.handleListAgents(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	// Token literal must not appear anywhere in the response body.
	if strings.Contains(body, token) {
		t.Errorf("auth_token literal %q found in response body:\n%s", token, body)
	}
	// auth_token field must not be present at all.
	if strings.Contains(body, "auth_token") {
		t.Errorf(`"auth_token" field found in response body:\n%s`, body)
	}

	// Parse the response and verify driver, model, base_url are present.
	var resp struct {
		Agents []struct {
			Name    string `json:"name"`
			Driver  string `json:"driver"`
			Model   string `json:"model"`
			BaseURL string `json:"base_url"`
		} `json:"agents"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(resp.Agents))
	}
	got := resp.Agents[0]
	if got.Driver != "claude-env" {
		t.Errorf("driver: got %q, want claude-env", got.Driver)
	}
	if got.Model != "claude-opus-4-6" {
		t.Errorf("model: got %q, want claude-opus-4-6", got.Model)
	}
	if got.BaseURL != "http://localhost:11434" {
		t.Errorf("base_url: got %q, want http://localhost:11434", got.BaseURL)
	}
}

// TestListAgents_ReturnsFullNonSecretConfig is the backend leg of T-5
// (NFR-2 / NFR-3): GET /agents must return every non-secret AgentConfig
// field (Milestone 1's expanded agentSummary) and must never leak
// auth_token. Configures a claude-env agent exercising every field in the
// -3-be plan's Milestone 1 table.
func TestListAgents_ReturnsFullNonSecretConfig(t *testing.T) {
	const token = "s3cr3t-token-must-not-appear"

	ag := config.AgentConfig{
		Name:           "full-config-agent",
		Roles:          []string{"analyst"},
		Driver:         "claude-env",
		Model:          "claude-opus-4-6",
		Endpoint:       "https://legacy.example.com/endpoint",
		AllowedPaths:   []string{"lifecycle/requirements/"},
		TimeoutMinutes: 45,
		GitIdentity: config.GitIdentity{
			Name:  "Full Config Agent",
			Email: "full-config-agent@example.com",
		},
		PromptTemplates: map[string]string{
			"analyst": "analyse {target_path}",
		},
		ActiveStatus:       "in-development",
		DoneOnSuccess:      true,
		SourceTypes:        []string{"requirement"},
		OllamaInstanceName: "local-ollama",
		OllamaEndpoint:     "chat",
		BashAllowlist:      []string{"go test *"},
		BashDenylist:       []string{"rm *"},
		OnDenial:           "abort",
		ObserveOnly:        true,
		ShellCommand:       "./run-agent.sh",
		BaseURL:            "http://localhost:11434",
		AuthToken:          token,
	}

	p, cleanup := newTestProjectWithClaudeEnvAgent(t, ag)
	defer cleanup()

	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withProjectAndUser(req, p, "po@test")

	w := httptest.NewRecorder()
	s.handleListAgents(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	if strings.Contains(body, token) {
		t.Errorf("auth_token literal %q found in response body:\n%s", token, body)
	}
	if strings.Contains(body, "auth_token") {
		t.Errorf(`"auth_token" field found in response body:\n%s`, body)
	}

	var resp struct {
		Agents []struct {
			Name           string   `json:"name"`
			Roles          []string `json:"roles"`
			Driver         string   `json:"driver"`
			Model          string   `json:"model"`
			Endpoint       string   `json:"endpoint"`
			ActiveStatus   string   `json:"active_status"`
			AllowedPaths   []string `json:"allowed_write_paths"`
			TimeoutMinutes int      `json:"timeout_minutes"`
			GitIdentity    *struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"git_identity"`
			PromptTemplates    map[string]string `json:"prompt_templates"`
			DoneOnSuccess      bool              `json:"done_on_success"`
			SourceTypes        []string          `json:"source_types"`
			OllamaInstanceName string            `json:"ollama_instance"`
			OllamaEndpoint     string            `json:"ollama_endpoint"`
			BashAllowlist      []string          `json:"bash_allowlist"`
			BashDenylist       []string          `json:"bash_denylist"`
			OnDenial           string            `json:"on_denial"`
			ObserveOnly        bool              `json:"observe_only"`
			ShellCommand       string            `json:"shell_command"`
			BaseURL            string            `json:"base_url"`
		} `json:"agents"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(resp.Agents))
	}
	got := resp.Agents[0]

	if got.Name != ag.Name {
		t.Errorf("name: got %q, want %q", got.Name, ag.Name)
	}
	if len(got.Roles) != 1 || got.Roles[0] != "analyst" {
		t.Errorf("roles: got %v, want [analyst]", got.Roles)
	}
	if got.Driver != ag.Driver {
		t.Errorf("driver: got %q, want %q", got.Driver, ag.Driver)
	}
	if got.Model != ag.Model {
		t.Errorf("model: got %q, want %q", got.Model, ag.Model)
	}
	if got.Endpoint != ag.Endpoint {
		t.Errorf("endpoint: got %q, want %q", got.Endpoint, ag.Endpoint)
	}
	if got.ActiveStatus != ag.ActiveStatus {
		t.Errorf("active_status: got %q, want %q", got.ActiveStatus, ag.ActiveStatus)
	}
	if len(got.AllowedPaths) != 1 || got.AllowedPaths[0] != ag.AllowedPaths[0] {
		t.Errorf("allowed_write_paths: got %v, want %v", got.AllowedPaths, ag.AllowedPaths)
	}
	if got.TimeoutMinutes != ag.TimeoutMinutes {
		t.Errorf("timeout_minutes: got %d, want %d", got.TimeoutMinutes, ag.TimeoutMinutes)
	}
	if got.GitIdentity == nil {
		t.Fatal("git_identity: got nil, want non-nil")
	}
	if got.GitIdentity.Name != ag.GitIdentity.Name || got.GitIdentity.Email != ag.GitIdentity.Email {
		t.Errorf("git_identity: got %+v, want %+v", got.GitIdentity, ag.GitIdentity)
	}
	if got.PromptTemplates["analyst"] != ag.PromptTemplates["analyst"] {
		t.Errorf("prompt_templates[analyst]: got %q, want %q", got.PromptTemplates["analyst"], ag.PromptTemplates["analyst"])
	}
	if got.DoneOnSuccess != ag.DoneOnSuccess {
		t.Errorf("done_on_success: got %v, want %v", got.DoneOnSuccess, ag.DoneOnSuccess)
	}
	if len(got.SourceTypes) != 1 || got.SourceTypes[0] != ag.SourceTypes[0] {
		t.Errorf("source_types: got %v, want %v", got.SourceTypes, ag.SourceTypes)
	}
	if got.OllamaInstanceName != ag.OllamaInstanceName {
		t.Errorf("ollama_instance: got %q, want %q", got.OllamaInstanceName, ag.OllamaInstanceName)
	}
	if got.OllamaEndpoint != ag.OllamaEndpoint {
		t.Errorf("ollama_endpoint: got %q, want %q", got.OllamaEndpoint, ag.OllamaEndpoint)
	}
	if len(got.BashAllowlist) != 1 || got.BashAllowlist[0] != ag.BashAllowlist[0] {
		t.Errorf("bash_allowlist: got %v, want %v", got.BashAllowlist, ag.BashAllowlist)
	}
	if len(got.BashDenylist) != 1 || got.BashDenylist[0] != ag.BashDenylist[0] {
		t.Errorf("bash_denylist: got %v, want %v", got.BashDenylist, ag.BashDenylist)
	}
	if got.OnDenial != ag.OnDenial {
		t.Errorf("on_denial: got %q, want %q", got.OnDenial, ag.OnDenial)
	}
	if got.ObserveOnly != ag.ObserveOnly {
		t.Errorf("observe_only: got %v, want %v", got.ObserveOnly, ag.ObserveOnly)
	}
	if got.ShellCommand != ag.ShellCommand {
		t.Errorf("shell_command: got %q, want %q", got.ShellCommand, ag.ShellCommand)
	}
	if got.BaseURL != ag.BaseURL {
		t.Errorf("base_url: got %q, want %q", got.BaseURL, ag.BaseURL)
	}
}
