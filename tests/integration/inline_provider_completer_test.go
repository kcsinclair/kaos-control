// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Integration tests for the inline conversational driver provider
// abstraction: POST /ideas/converse and POST /ideas/generate driven through
// a stub OpenAI-compatible provider (httptest), rather than the real `claude`
// CLI binary. These do NOT require ANTHROPIC_API_KEY.
//
// Test plan: lifecycle/test-plans/inline-driver-provider-abstraction-5-test.md
// Milestone 5.

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/project"
)

// ── Stub OpenAI-compatible completion server ─────────────────────────────────

// stubCapturedRequest is one recorded POST /v1/chat/completions call.
type stubCapturedRequest struct {
	Body    map[string]any
	Headers http.Header
}

// stubCompletionServer is a minimal httptest-backed stand-in for an
// OpenAI-compatible /v1/chat/completions endpoint, scripted with a queue of
// canned assistant message contents (one per call, repeating the last entry
// once exhausted). It matches exactly the request/response shape the inline
// completer (internal/ideachat/openai_completer.go) speaks: POST
// {base_url}/v1/chat/completions, body {model, messages, max_tokens?},
// response {"choices":[{"message":{"content": "..."}}]}.
type stubCompletionServer struct {
	server *httptest.Server

	mu        sync.Mutex
	responses []string
	requests  []stubCapturedRequest
}

// newStubCompletionServer starts a stub server that returns responses[i] as
// the assistant message content for the (i+1)th completion request it
// receives (1-indexed), repeating the last response for any further calls.
func newStubCompletionServer(t *testing.T, responses ...string) *stubCompletionServer {
	t.Helper()
	s := &stubCompletionServer{responses: responses}
	s.server = httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(s.server.Close)
	return s
}

func (s *stubCompletionServer) handle(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/v1/chat/completions" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	raw, _ := io.ReadAll(r.Body)
	r.Body.Close()
	var body map[string]any
	_ = json.Unmarshal(raw, &body)

	s.mu.Lock()
	s.requests = append(s.requests, stubCapturedRequest{Body: body, Headers: r.Header.Clone()})
	idx := len(s.requests) - 1
	var content string
	switch {
	case len(s.responses) == 0:
		content = `{"action":"propose","reply":"ok","slug":"stub-default","title":"Stub Default","labels":[],"body":"# Stub Default\n\nBody."}`
	case idx < len(s.responses):
		content = s.responses[idx]
	default:
		content = s.responses[len(s.responses)-1]
	}
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{"role": "assistant", "content": content},
			},
		},
	})
}

// URL returns the stub server's loopback base URL.
func (s *stubCompletionServer) URL() string { return s.server.URL }

// Requests returns a copy of every completion request captured so far.
func (s *stubCompletionServer) Requests() []stubCapturedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]stubCapturedRequest, len(s.requests))
	copy(out, s.requests)
	return out
}

// ── Test project wiring ───────────────────────────────────────────────────────

const stubProviderName = "stub-provider"
const stubProviderAPIKey = "stub-secret-abc123"

// inlineProviderCfgYAML binds both inline generation agents (idea-capture,
// docs-capture) to stubProviderName via driver: inline + provider: <name>,
// covering all four inline template keys: idea-capture, idea-generate,
// defect-generate, doc-generate.
const inlineProviderCfgYAML = `git:
  default_branch: main
  branch_template: "ticket/{slug}"

roles:
  - product-owner
  - analyst
  - backend-developer
  - frontend-developer
  - test-developer
  - qa
  - reviewer
  - approver
  - tech-writer

stages:
  - {name: ideas, dir: ideas}
  - {name: requirements, dir: requirements}
  - {name: backend-plans, dir: backend-plans}
  - {name: frontend-plans, dir: frontend-plans}
  - {name: test-plans, dir: test-plans}
  - {name: tests, dir: tests}
  - {name: prototypes, dir: prototypes}
  - {name: releases, dir: releases}
  - {name: sprints, dir: sprints}
  - {name: defects, dir: defects}
  - {name: docs, dir: docs}

users:
  - email: admin@test.local
    roles: [product-owner, analyst, reviewer, approver]
  - email: dev@test.local
    roles: [backend-developer, frontend-developer, test-developer, tech-writer]
  - email: qa@test.local
    roles: [qa]

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []

agents:
  - name: idea-capture
    role: [product-owner]
    driver: inline
    model: stub-model
    provider: stub-provider
    allowed_write_paths:
      - lifecycle/ideas
    prompt_templates:
      idea-capture: |
        You are an idea-capture assistant. Always reply with a single JSON
        object: {"action":"clarify"|"propose","reply":"...","slug":"...","title":"...","labels":[...],"body":"..."}.
      idea-generate: |
        Generate a single idea proposal as a JSON object with action="propose".
      defect-generate: |
        Generate a single defect proposal as a JSON object with action="propose".
  - name: docs-capture
    role: [tech-writer]
    driver: inline
    model: stub-model
    provider: stub-provider
    allowed_write_paths:
      - lifecycle/docs
    prompt_templates:
      doc-generate: |
        Generate a single doc proposal as a JSON object with action="propose".
`

// newInlineProviderTestEnv opens a testproject whose idea-capture and
// docs-capture agents route through stub's OpenAI-compatible endpoint,
// registered as the app-level provider "stub-provider" carrying a
// stubProviderAPIKey secret (used to assert the secret never leaks back over
// the API). stub.URL() is always a 127.0.0.1 loopback address (httptest
// default), so every test built on this helper exercises the "local-provider,
// no outbound network" path (NFR-5) implicitly.
func newInlineProviderTestEnv(t *testing.T, stub *stubCompletionServer) *testEnv {
	t.Helper()
	providers := []config.Provider{
		{
			Name:    stubProviderName,
			BaseURL: stub.URL(),
			Driver:  "openai-compatible",
			APIKey:  stubProviderAPIKey,
		},
	}
	return newTestEnvFull(t, nil, nil, inlineProviderCfgYAML, func(o *project.OpenOptions) {
		o.Providers = providers
	})
}

// ── Milestone 5, TC1: converse to a proposal, accept writes the artifact ───────

// TestInlineProvider_ConverseAcceptWritesArtifact drives a two-turn
// conversation (clarify, then propose) entirely through the stub provider and
// confirms __accept__ writes an idea artifact shaped the same way the
// CLI-default path does (type: idea, status: draft, lineage: <slug>).
func TestInlineProvider_ConverseAcceptWritesArtifact(t *testing.T) {
	stub := newStubCompletionServer(t,
		`{"action":"clarify","reply":"What should we call it?","slug":"","title":"","labels":[],"body":""}`,
		`{"action":"propose","reply":"Here is your idea proposal.","slug":"stub-dark-mode","title":"Stub Dark Mode Toggle","labels":[],"body":"# Stub Dark Mode Toggle\n\nLets users switch themes."}`,
	)
	env := newInlineProviderTestEnv(t, stub)

	resp := converseAPI(env, "", "I want a dark mode toggle for the settings page.")
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	sessionID, _ := data["session_id"].(string)
	if sessionID == "" {
		t.Fatal("missing session_id after first turn")
	}
	if status, _ := data["status"].(string); status != "conversing" {
		t.Fatalf("expected first-turn status 'conversing', got %q", status)
	}

	resp = converseAPI(env, sessionID, "Call it Stub Dark Mode Toggle.")
	requireStatus(t, resp, 200)
	data = readJSON(t, resp)
	if status, _ := data["status"].(string); status != "proposed" {
		t.Fatalf("expected second-turn status 'proposed', got %q", status)
	}
	preview, _ := data["preview"].(map[string]any)
	if preview == nil {
		t.Fatal("expected non-null preview when status is 'proposed'")
	}

	resp = converseAPI(env, sessionID, "__accept__")
	requireStatus(t, resp, 200)
	data = readJSON(t, resp)
	if status, _ := data["status"].(string); status != "created" {
		t.Fatalf("expected accept status 'created', got %q", status)
	}
	artifactPath, _ := data["artifact_path"].(string)
	if artifactPath == "" {
		t.Fatal("expected non-null artifact_path")
	}
	if !strings.HasPrefix(artifactPath, "lifecycle/ideas/") || !strings.HasSuffix(artifactPath, ".md") {
		t.Errorf("unexpected artifact_path shape: %q", artifactPath)
	}

	absPath := filepath.Join(env.projectRoot, artifactPath)
	content, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("artifact file missing on disk: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "type: idea") {
		t.Error("artifact missing 'type: idea' in frontmatter")
	}
	if !strings.Contains(text, "status: draft") {
		t.Error("artifact missing 'status: draft' in frontmatter")
	}
	slug := slugFromPath(artifactPath)
	if !strings.Contains(text, "lineage: "+slug) {
		t.Errorf("artifact missing 'lineage: %s' in frontmatter", slug)
	}

	// __accept__ must not call the LLM: exactly the two conversation turns'
	// worth of upstream requests.
	if got := len(stub.Requests()); got != 2 {
		t.Errorf("expected 2 upstream completion requests (accept makes none), got %d", got)
	}
}

// ── Milestone 5, TC2: all four consumers via the OpenAI-compatible completer ──

// TestInlineProvider_GenerateAllTypes drives POST /ideas/generate for
// type=idea, type=defect and type=doc through the stub provider, covering the
// idea-generate, defect-generate and doc-generate template keys (the fourth,
// idea-capture, is covered by TestInlineProvider_ConverseAcceptWritesArtifact).
func TestInlineProvider_GenerateAllTypes(t *testing.T) {
	cases := []struct {
		name          string
		artifactType  string
		wantTargetDir string
		response      string
	}{
		{
			name: "idea", artifactType: "idea", wantTargetDir: "lifecycle/ideas",
			response: `{"action":"propose","reply":"ok","slug":"stub-idea","title":"Stub Idea","labels":[],"body":"# Stub Idea\n\nBody."}`,
		},
		{
			name: "defect", artifactType: "defect", wantTargetDir: "lifecycle/defects",
			response: `{"action":"propose","reply":"ok","slug":"stub-defect","title":"Stub Defect","labels":[],"body":"# Stub Defect\n\nSteps to reproduce."}`,
		},
		{
			name: "doc", artifactType: "doc", wantTargetDir: "lifecycle/docs",
			response: `{"action":"propose","reply":"ok","slug":"stub-doc","title":"Stub Doc","labels":[],"body":"# Stub Doc\n\nDocumentation body."}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := newStubCompletionServer(t, tc.response)
			env := newInlineProviderTestEnv(t, stub)

			input := "This is a sufficiently long description of the " + tc.name + " so it clears the five word minimum easily"
			resp := generateAPI(env, input, tc.artifactType)
			requireStatus(t, resp, 200)
			data := readJSON(t, resp)

			if got, _ := data["target_dir"].(string); got != tc.wantTargetDir {
				t.Errorf("target_dir: want %q, got %q", tc.wantTargetDir, got)
			}
			fm, _ := data["frontmatter"].(map[string]any)
			if fm == nil {
				t.Fatal("missing frontmatter")
			}
			if got, _ := fm["type"].(string); got != tc.artifactType {
				t.Errorf("frontmatter.type: want %q, got %q", tc.artifactType, got)
			}
			if tc.artifactType == "defect" {
				labels, _ := data["labels"].([]any)
				found := false
				for _, l := range labels {
					if l == "defect" {
						found = true
					}
				}
				if !found {
					t.Errorf("defect generation must force the 'defect' label, got %v", labels)
				}
			}

			if got := len(stub.Requests()); got != 1 {
				t.Errorf("expected exactly 1 upstream completion request, got %d", got)
			}
		})
	}
}

// ── Milestone 5, TC3: request shape cross-check at the integration layer ──────

// TestInlineProvider_RequestShape_NoToolsAndMessageMapping cross-checks the
// unit-level M2 assertions end-to-end: the stub-captured requests never carry
// a "tools" key, and a multi-turn conversation's message array preserves
// system-first ordering and role/content mapping across turns.
func TestInlineProvider_RequestShape_NoToolsAndMessageMapping(t *testing.T) {
	stub := newStubCompletionServer(t,
		`{"action":"clarify","reply":"What should we call it?","slug":"","title":"","labels":[],"body":""}`,
		`{"action":"propose","reply":"done","slug":"stub-shape","title":"Stub Shape","labels":[],"body":"# Stub Shape\n\nBody."}`,
	)
	env := newInlineProviderTestEnv(t, stub)

	resp := converseAPI(env, "", "First turn message about a shape feature.")
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	sessionID, _ := data["session_id"].(string)
	if sessionID == "" {
		t.Fatal("missing session_id after first turn")
	}

	resp = converseAPI(env, sessionID, "Second turn follow-up message.")
	requireStatus(t, resp, 200)
	readJSON(t, resp)

	reqs := stub.Requests()
	if len(reqs) != 2 {
		t.Fatalf("expected 2 captured requests, got %d", len(reqs))
	}

	for i, r := range reqs {
		if _, hasTools := r.Body["tools"]; hasTools {
			t.Errorf("request %d: body must not contain a 'tools' key", i)
		}
		if model, _ := r.Body["model"].(string); model != "stub-model" {
			t.Errorf("request %d: model = %q, want 'stub-model'", i, model)
		}
	}

	// The second request carries the full turn history: system, user,
	// assistant (first raw reply), user (the follow-up).
	messages, _ := reqs[1].Body["messages"].([]any)
	if len(messages) < 4 {
		t.Fatalf("expected at least 4 messages in second request, got %d: %v", len(messages), messages)
	}
	roles := make([]string, len(messages))
	for i, m := range messages {
		mm, _ := m.(map[string]any)
		roles[i], _ = mm["role"].(string)
	}
	wantRoles := []string{"system", "user", "assistant", "user"}
	for i, want := range wantRoles {
		if roles[i] != want {
			t.Errorf("message[%d].role = %q, want %q (full: %v)", i, roles[i], want, roles)
		}
	}
	lastMsg, _ := messages[len(messages)-1].(map[string]any)
	if content, _ := lastMsg["content"].(string); !strings.Contains(content, "Second turn follow-up message.") {
		t.Errorf("last message content = %q, want to contain the follow-up text", content)
	}
}

// ── Milestone 5, TC4: offline / local-provider capability (NFR-5) ─────────────

// TestInlineProvider_OfflineLocalStub_NoExternalNetwork asserts the stub
// provider is loopback-only (no external network involved) and that a
// generation call completes successfully against it end-to-end, exercising
// the local-model-operability capability referenced by NFR-5.
func TestInlineProvider_OfflineLocalStub_NoExternalNetwork(t *testing.T) {
	stub := newStubCompletionServer(t,
		`{"action":"propose","reply":"ok","slug":"stub-offline","title":"Stub Offline","labels":[],"body":"# Stub Offline\n\nBody."}`,
	)
	env := newInlineProviderTestEnv(t, stub)

	u, err := url.Parse(stub.URL())
	if err != nil {
		t.Fatal(err)
	}
	switch u.Hostname() {
	case "127.0.0.1", "localhost", "::1":
		// loopback-only, as required.
	default:
		t.Fatalf("stub provider must be loopback-only, got host %q", u.Hostname())
	}

	input := "This description of the offline capability is long enough to pass validation easily"
	resp := generateAPI(env, input, "idea")
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	if slug, _ := data["slug"].(string); slug == "" {
		t.Error("expected non-empty slug from offline local-provider generation")
	}
	if got := len(stub.Requests()); got != 1 {
		t.Errorf("expected exactly 1 upstream request against the local stub, got %d", got)
	}
}

// ── Milestone 5, TC5: secret boundary (NFR-1) ──────────────────────────────────

// TestInlineProvider_NoSecretLeak asserts the provider's api_key never
// appears in the /ideas/generate response body nor in the GET /agents
// provider listing.
func TestInlineProvider_NoSecretLeak(t *testing.T) {
	stub := newStubCompletionServer(t,
		`{"action":"propose","reply":"ok","slug":"stub-secret","title":"Stub Secret","labels":[],"body":"# Stub Secret\n\nBody."}`,
	)
	env := newInlineProviderTestEnv(t, stub)

	input := "This description is long enough for the secret boundary check to pass validation cleanly"
	resp := generateAPI(env, input, "idea")
	requireStatus(t, resp, 200)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), stubProviderAPIKey) {
		t.Error("generate response leaked the provider api_key")
	}

	agentsResp := env.doRequest("GET", "/api/p/testproject/agents", nil)
	requireStatus(t, agentsResp, 200)
	agentsBody, err := io.ReadAll(agentsResp.Body)
	agentsResp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(agentsBody), stubProviderAPIKey) {
		t.Error("GET /agents leaked the provider api_key")
	}
}
