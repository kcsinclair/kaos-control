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
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/project"
	"github.com/kaos-control/kaos-control/internal/queue"
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
    provider: anthropic-cloud
    model: claude-3-7-sonnet
    fallback_provider: gemini-cloud
    fallback_model: gemini-2.5-flash
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

	if err := p.SwitchAgentProvider("failed-over-agent", "gemini-cloud", "gemini-2.5-flash", "seed failover", true); err != nil {
		t.Fatalf("seeding failover state: %v", err)
	}

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
	if failedOver.SwitchedAt == "" || failedOver.Reason != "seed failover" {
		t.Errorf("expected switched_at/reason populated from operations.yaml, got %+v", failedOver)
	}
}

// TestHandleGetFailoverStatus_ReachabilityPartialPauseAwaitingDecision
// verifies Milestone 8: the status response surfaces provider reachability
// (Milestone 6, all modes), FR-3.4 partial pause, and FR-7.3
// awaiting-operator-decision — all sourced from operations.yaml, not a live
// probe.
func TestHandleGetFailoverStatus_ReachabilityPartialPauseAwaitingDecision(t *testing.T) {
	p := newTestProjectForProviderSwitch(t)
	s := testServerWithAppProviders()

	if err := p.Operations().SetReachability("anthropic-cloud", true, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := p.Operations().SetAgentState(project.AgentOperationalState{
		Agent:        "failed-over-agent",
		Primary:      project.ProviderModel{Provider: "anthropic-cloud", Model: "claude-3-7-sonnet"},
		Active:       project.ProviderModel{Provider: "anthropic-cloud", Model: "claude-3-7-sonnet"},
		PartialPause: true,
		Reason:       "no secondary",
	}); err != nil {
		t.Fatal(err)
	}
	if err := p.Operations().SetAwaitingOperatorDecision("analyst-agent", "job-123"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withProjectAndUser(req, p, "po@test")
	w := httptest.NewRecorder()
	s.handleGetFailoverStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Agents       []providerSwitchAgentStatus           `json:"agents"`
		Reachability map[string]providerReachabilityStatus `json:"reachability"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	reach, ok := resp.Reachability["anthropic-cloud"]
	if !ok || !reach.Healthy {
		t.Errorf("expected anthropic-cloud reachability recorded healthy, got %+v ok=%v", reach, ok)
	}

	var partiallyPaused, awaitingDecision *providerSwitchAgentStatus
	for i := range resp.Agents {
		switch resp.Agents[i].Agent {
		case "failed-over-agent":
			partiallyPaused = &resp.Agents[i]
		case "analyst-agent":
			awaitingDecision = &resp.Agents[i]
		}
	}
	if partiallyPaused == nil || !partiallyPaused.PartialPause {
		t.Fatalf("expected failed-over-agent partial_pause=true, got %+v", partiallyPaused)
	}
	if partiallyPaused.IsFailover {
		t.Errorf("a partially-paused agent (active==primary) is not in failover, got %+v", partiallyPaused)
	}
	if awaitingDecision == nil || !awaitingDecision.AwaitingDecision || awaitingDecision.AwaitingDecisionJobID != "job-123" {
		t.Fatalf("expected analyst-agent awaiting_decision for job-123, got %+v", awaitingDecision)
	}
}

// TestHandleGetSwitchoverPolicy verifies FR-2.4: the effective event->action
// policy is retrievable and lists an action for every classified reason,
// with automated_switchover defaulting to disabled (FR-2.1).
func TestHandleGetSwitchoverPolicy(t *testing.T) {
	p := newTestProjectForProviderSwitch(t)
	s := testServerWithAppProviders()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withProjectAndUser(req, p, "po@test")
	w := httptest.NewRecorder()
	s.handleGetSwitchoverPolicy(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		AutomatedSwitchover bool              `json:"automated_switchover"`
		Actions             map[string]string `json:"actions"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.AutomatedSwitchover {
		t.Error("expected automated_switchover to default to disabled")
	}
	if len(resp.Actions) == 0 {
		t.Fatal("expected a defaulted action for every classified reason")
	}
	for _, reason := range []string{"rate_limit", "overloaded", "timeout", "provider_disconnected"} {
		if _, ok := resp.Actions[reason]; !ok {
			t.Errorf("expected an effective action for reason %q, got %+v", reason, resp.Actions)
		}
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

	// lifecycle/config.yaml (declared intent) is never mutated by a switch —
	// the active override lives in operations.yaml.
	ag, ok := p.Agents.GetAgent("analyst-agent")
	if !ok {
		t.Fatal("analyst-agent not found after switch")
	}
	if ag.Provider != "anthropic-cloud" || ag.Model != "claude-3-7-sonnet" {
		t.Errorf("expected declared config unchanged, got %+v", ag)
	}
	state, ok := p.Operations().AgentState("analyst-agent")
	if !ok {
		t.Fatal("expected operations.yaml to record the active override")
	}
	if state.Active.Provider != "local-ollama" || state.Active.Model != "llama3" {
		t.Errorf("expected switched provider/model, got %+v", state)
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

// serverWithRunningJob returns a Server wired with a queue.Dispatcher that
// has exactly one running job for (project, agent) — used to exercise the
// FR-8.2 in-flight-run guard on manual switch routes.
func serverWithRunningJob(t *testing.T, projectName, agentName string) *Server {
	t.Helper()
	s := testServerWithAppProviders()

	store, err := queue.Open(filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	if err := store.Enqueue(queue.Job{
		Project:      projectName,
		ArtifactPath: "lifecycle/ideas/a.md",
		AgentName:    agentName,
		EnqueuedBy:   "po@test",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Dequeue(); err != nil { // pending -> running
		t.Fatal(err)
	}

	lookup := func(string) (queue.ProjectAccess, bool) { return queue.ProjectAccess{}, false }
	s.queue = queue.New(store, lookup, hub.New(), queue.Config{})
	return s
}

// TestHandleAgentSwitchProvider_RejectedWhileRunInProgress verifies FR-8.2:
// a manual switch is rejected with the running jobs named when any run is
// executing for the project, and no switch occurs.
func TestHandleAgentSwitchProvider_RejectedWhileRunInProgress(t *testing.T) {
	p := newTestProjectForProviderSwitch(t)
	s := serverWithRunningJob(t, p.Entry.Name, "analyst-agent")

	body, _ := json.Marshal(map[string]string{"provider": "local-ollama", "model": "llama3"})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req = withProjectAndUser(req, p, "po@test")
	req = withChiParam(req, "name", "analyst-agent")
	w := httptest.NewRecorder()
	s.handleAgentSwitchProvider(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status: got %d, want 409, body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		RunningJobs []map[string]any `json:"running_jobs"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.RunningJobs) != 1 || resp.RunningJobs[0]["agent"] != "analyst-agent" {
		t.Errorf("expected the running job named in the rejection, got %+v", resp.RunningJobs)
	}
	if _, ok := p.Operations().AgentState("analyst-agent"); ok {
		t.Error("expected no switch to have occurred")
	}
}

// TestHandleSwitchAllProviders_RejectedWhileRunInProgress mirrors the above
// for the project-wide switch-all route.
func TestHandleSwitchAllProviders_RejectedWhileRunInProgress(t *testing.T) {
	p := newTestProjectForProviderSwitch(t)
	s := serverWithRunningJob(t, p.Entry.Name, "analyst-agent")

	body, _ := json.Marshal(map[string]string{
		"from_provider": "anthropic-cloud", "to_provider": "local-ollama", "to_model": "llama3",
	})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req = withProjectAndUser(req, p, "po@test")
	w := httptest.NewRecorder()
	s.handleSwitchAllProviders(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status: got %d, want 409, body: %s", w.Code, w.Body.String())
	}
	if _, ok := p.Operations().AgentState("analyst-agent"); ok {
		t.Error("expected no switch to have occurred")
	}
}

func TestHandleAgentRestoreProvider(t *testing.T) {
	p := newTestProjectForProviderSwitch(t)
	s := testServerWithAppProviders()

	if err := p.SwitchAgentProvider("failed-over-agent", "gemini-cloud", "gemini-2.5-flash", "seed failover", true); err != nil {
		t.Fatalf("seeding failover state: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = withProjectAndUser(req, p, "po@test")
	req = withChiParam(req, "name", "failed-over-agent")
	w := httptest.NewRecorder()
	s.handleAgentRestoreProvider(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, body: %s", w.Code, w.Body.String())
	}

	if _, ok := p.Operations().AgentState("failed-over-agent"); ok {
		t.Error("expected operations.yaml override cleared after restore")
	}
	ag, ok := p.Agents.GetAgent("failed-over-agent")
	if !ok {
		t.Fatal("failed-over-agent not found after restore")
	}
	if ag.Provider != "anthropic-cloud" || ag.Model != "claude-3-7-sonnet" {
		t.Errorf("expected declared config (primary) unchanged, got %+v", ag)
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

	if err := p.SwitchAgentProvider("failed-over-agent", "gemini-cloud", "gemini-2.5-flash", "seed failover", true); err != nil {
		t.Fatalf("seeding failover state: %v", err)
	}

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

	if _, ok := p.Operations().AgentState("failed-over-agent"); ok {
		t.Error("expected failed-over-agent's operations override cleared")
	}
	if _, ok := p.Operations().AgentState("analyst-agent"); ok {
		t.Error("expected analyst-agent (not in failover) untouched")
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

	state, ok := p.Operations().AgentState("analyst-agent")
	if !ok {
		t.Fatal("expected operations.yaml to record the template-applied override")
	}
	if state.Active.Provider != "local-ollama" || state.Active.Model != "llama3" {
		t.Errorf("expected template-applied provider/model, got %+v", state)
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
