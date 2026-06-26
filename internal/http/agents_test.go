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
