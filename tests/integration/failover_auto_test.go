// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Suite 2.1 — Automated failover & queue retry integration tests (FA1-FA3).
//
// Each test drives a real dispatcher + agent supervisor loop through a fake
// `claude` binary (see setupFakeClaudeWithScript in queue_helpers_test.go).
// The stub emits a transient-error stream event on its first invocation
// (marker file absent) and a normal success stream on every subsequent
// invocation (marker file present), so the re-enqueued job created by
// automated failover actually completes.

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
)

// failoverAutoCfgYAML wires requirements-analyst with an Anthropic primary
// and a Gemini fallback, with automated failover enabled.
const failoverAutoCfgYAML = `git:
  default_branch: main
  branch_template: "ticket/{slug}"

roles:
  - product-owner
  - analyst
  - devops

stages:
  - {name: ideas,          dir: ideas}
  - {name: requirements,   dir: requirements}
  - {name: backend-plans,  dir: backend-plans}
  - {name: frontend-plans, dir: frontend-plans}
  - {name: test-plans,     dir: test-plans}
  - {name: tests,          dir: tests}
  - {name: prototypes,     dir: prototypes}
  - {name: releases,       dir: releases}
  - {name: sprints,        dir: sprints}
  - {name: defects,        dir: defects}

users:
  - email: admin@test.local
    roles: [product-owner]
  - email: dev@test.local
    roles: [devops]
  - email: qa@test.local
    roles: [analyst]

required_plans:
  ticket: []
  epic: []

provider_failover:
  auto_switch: true
  max_failovers_per_run: 1

agents:
  - name: requirements-analyst
    role: [analyst]
    driver: claude-code-cli
    provider: anthropic-cloud
    model: claude-3-7-sonnet
    fallback_provider: gemini-cloud
    fallback_model: gemini-2.5-flash
    allowed_write_paths:
      - lifecycle/requirements
      - lifecycle/ideas
    git_identity:
      name: Requirements Analyst Agent
      email: requirements-analyst@test.local
    prompt_templates:
      analyst: "Test requirements analyst prompt for {target_path}"
`

// failoverDisabledCfgYAML is identical to failoverAutoCfgYAML except
// auto_switch is left at its default (false), so a transient failure must
// fall through to the standard rate-limit pause instead of switching.
const failoverDisabledCfgYAML = `git:
  default_branch: main
  branch_template: "ticket/{slug}"

roles:
  - product-owner
  - analyst
  - devops

stages:
  - {name: ideas,          dir: ideas}
  - {name: requirements,   dir: requirements}
  - {name: backend-plans,  dir: backend-plans}
  - {name: frontend-plans, dir: frontend-plans}
  - {name: test-plans,     dir: test-plans}
  - {name: tests,          dir: tests}
  - {name: prototypes,     dir: prototypes}
  - {name: releases,       dir: releases}
  - {name: sprints,        dir: sprints}
  - {name: defects,        dir: defects}

users:
  - email: admin@test.local
    roles: [product-owner]
  - email: dev@test.local
    roles: [devops]
  - email: qa@test.local
    roles: [analyst]

required_plans:
  ticket: []
  epic: []

agents:
  - name: requirements-analyst
    role: [analyst]
    driver: claude-code-cli
    provider: anthropic-cloud
    model: claude-3-7-sonnet
    fallback_provider: gemini-cloud
    fallback_model: gemini-2.5-flash
    allowed_write_paths:
      - lifecycle/requirements
      - lifecycle/ideas
    git_identity:
      name: Requirements Analyst Agent
      email: requirements-analyst@test.local
    prompt_templates:
      analyst: "Test requirements analyst prompt for {target_path}"
`

// failoverThenSucceedScript builds a fake-claude shell script body (no
// shebang — setupFakeClaudeWithScript prepends it) that emits errorJSON as a
// stream-json error line and exits 0 on its first invocation, then emits a
// normal success stream on every invocation after that. markerPath is a file
// path (inside a t.TempDir()) used to detect "first invocation".
func failoverThenSucceedScript(markerPath, errorJSON string) string {
	return fmt.Sprintf(`if [ -f %s ]; then
%sexit 0
else
touch %s
printf '%%s\n' '%s'
exit 0
fi
`, markerPath, fakeClaudeSuccessEvents, markerPath, errorJSON)
}

// assertAgentFailedOverTo fetches GET .../provider-switch/status and asserts
// agentName is recorded as failed over from primaryProvider to
// activeProvider — the operations.yaml-backed source of truth
// (agent-switchover-and-failover Milestone 2/8), since a switch/failover no
// longer touches lifecycle/config.yaml at all.
func assertAgentFailedOverTo(t *testing.T, env *failoverTestEnv, agentName, primaryProvider, activeProvider string) {
	t.Helper()
	resp := env.doRequest("GET", "/api/p/testproject/provider-switch/status", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	agents, _ := data["agents"].([]any)
	for _, raw := range agents {
		a, _ := raw.(map[string]any)
		if a["agent"] != agentName {
			continue
		}
		if isFailover, _ := a["is_failover"].(bool); !isFailover {
			t.Errorf("%s is_failover: got false, want true", agentName)
		}
		if a["primary_provider"] != primaryProvider {
			t.Errorf("%s primary_provider: got %v, want %v", agentName, a["primary_provider"], primaryProvider)
		}
		if a["active_provider"] != activeProvider {
			t.Errorf("%s active_provider: got %v, want %v", agentName, a["active_provider"], activeProvider)
		}
		return
	}
	t.Fatalf("expected %s in provider-switch/status response", agentName)
}

// TestFailover_AutoSwitch_HTTP529 (FA1): a job fails with an HTTP 529
// "overloaded_error" stream event. With auto_switch enabled and a healthy
// fallback, the dispatcher must: record the switch in operations.yaml only
// (provider -> fallback, primary stashed) — never touching
// lifecycle/config.yaml or git — broadcast provider.switched, and
// immediately retry the stalled job at the head of the queue without
// pausing. The retried job completes on the fallback.
func TestFailover_AutoSwitch_HTTP529(t *testing.T) {
	gemini := newMockProvider(t, true)
	providers := []config.Provider{
		{Name: "anthropic-cloud", BaseURL: "http://127.0.0.1:1", Driver: "openai-compatible"},
		{Name: "gemini-cloud", BaseURL: gemini.URL, Driver: "openai-compatible"},
	}

	markerPath := filepath.Join(t.TempDir(), "fa1-invoked")
	errorJSON := `{"error":{"type":"overloaded_error","message":"HTTP 529 Overloaded"}}`
	setupFakeClaudeWithScript(t, failoverThenSucceedScript(markerPath, errorJSON))

	env := newFailoverTestEnv(t, failoverAutoCfgYAML, providers, []seedArtifact{
		{relPath: "lifecycle/ideas/fa1-idea-1.md", content: makeApprovedArtifact("FA1 Idea", "idea", "fa1-idea")},
	})
	beforeSHA := env.headSHA()
	beforeCfg := env.readConfigYAML()

	ws := env.connectProjectWS()

	env.enqueue("lifecycle/ideas/fa1-idea-1.md", "requirements-analyst")

	// The re-enqueued job (attempts=2) should complete on the fallback
	// without the queue ever pausing.
	env.waitFor(15*time.Second, "requeued job to complete", func() bool {
		snap := env.queueSnapshot()
		if paused, _ := snap["paused"].(bool); paused {
			t.Fatal("queue should not pause on automated failover, but it did")
		}
		j := findJobByPath(snap, "lifecycle/ideas/fa1-idea-1.md")
		if j == nil {
			return false
		}
		attempts, _ := j["attempts"].(float64)
		return j["state"] == "completed" && attempts == 2
	})

	// (1) operations.yaml (not lifecycle/config.yaml) records the switch.
	assertAgentFailedOverTo(t, env, "requirements-analyst", "anthropic-cloud", "gemini-cloud")

	// (2) lifecycle/config.yaml is untouched and no git commit was made
	// (Milestone 2: switch/failover writes operations.yaml only).
	if got := env.readConfigYAML(); got != beforeCfg {
		t.Error("expected lifecycle/config.yaml to be byte-for-byte unchanged after automated failover")
	}
	if got := env.headSHA(); got != beforeSHA {
		t.Errorf("expected no new git commit after automated failover, HEAD moved from %s to %s", beforeSHA, got)
	}

	// (3) provider.switched WS event broadcast.
	payload := ws.waitForEventType(t, 5*time.Second, "provider.switched")
	if payload["agent"] != "requirements-analyst" {
		t.Errorf("provider.switched agent: got %v, want requirements-analyst", payload["agent"])
	}
	if payload["provider"] != "gemini-cloud" {
		t.Errorf("provider.switched provider: got %v, want gemini-cloud", payload["provider"])
	}
	if isFailover, _ := payload["is_failover"].(bool); !isFailover {
		t.Error("provider.switched is_failover: want true")
	}
}

// TestFailover_AutoSwitch_RateLimitQuota (FA2): an HTTP 429 quota-exhaustion
// stream event triggers the same automated failover path (rate_limit is in
// the default switch_on_kinds set) and the head-of-queue retry executes
// immediately.
func TestFailover_AutoSwitch_RateLimitQuota(t *testing.T) {
	gemini := newMockProvider(t, true)
	providers := []config.Provider{
		{Name: "anthropic-cloud", BaseURL: "http://127.0.0.1:1", Driver: "openai-compatible"},
		{Name: "gemini-cloud", BaseURL: gemini.URL, Driver: "openai-compatible"},
	}

	markerPath := filepath.Join(t.TempDir(), "fa2-invoked")
	errorJSON := `{"error":{"type":"rate_limit_error","message":"429 quota exceeded"}}`
	setupFakeClaudeWithScript(t, failoverThenSucceedScript(markerPath, errorJSON))

	env := newFailoverTestEnv(t, failoverAutoCfgYAML, providers, []seedArtifact{
		{relPath: "lifecycle/ideas/fa2-idea-1.md", content: makeApprovedArtifact("FA2 Idea", "idea", "fa2-idea")},
	})

	env.enqueue("lifecycle/ideas/fa2-idea-1.md", "requirements-analyst")

	env.waitFor(15*time.Second, "requeued job to complete after quota failover", func() bool {
		snap := env.queueSnapshot()
		if paused, _ := snap["paused"].(bool); paused {
			t.Fatal("queue should not pause on automated failover, but it did")
		}
		j := findJobByPath(snap, "lifecycle/ideas/fa2-idea-1.md")
		if j == nil {
			return false
		}
		attempts, _ := j["attempts"].(float64)
		return j["state"] == "completed" && attempts == 2
	})

	assertAgentFailedOverTo(t, env, "requirements-analyst", "anthropic-cloud", "gemini-cloud")
}

// TestFailover_Disabled_PausesQueue (FA3): the agent has a fallback_provider
// configured but the project's provider_failover.auto_switch is left at its
// default (false). A transient failure must fall through to the standard
// rate-limit pause-and-retry path instead of automated failover.
func TestFailover_Disabled_PausesQueue(t *testing.T) {
	gemini := newMockProvider(t, true)
	providers := []config.Provider{
		{Name: "anthropic-cloud", BaseURL: "http://127.0.0.1:1", Driver: "openai-compatible"},
		{Name: "gemini-cloud", BaseURL: gemini.URL, Driver: "openai-compatible"},
	}

	// Unparseable reset text -> dispatcher falls back to OverloadPause, but
	// what matters here is simply that it pauses at all rather than
	// switching provider.
	errorJSON := `{"error":{"type":"overloaded_error","message":"Overloaded"}}`
	setupFakeClaudeWithScript(t, "printf '%s\\n' '"+errorJSON+"'\nexit 0\n")

	env := newFailoverTestEnv(t, failoverDisabledCfgYAML, providers, []seedArtifact{
		{relPath: "lifecycle/ideas/fa3-idea-1.md", content: makeApprovedArtifact("FA3 Idea", "idea", "fa3-idea")},
	})

	env.enqueue("lifecycle/ideas/fa3-idea-1.md", "requirements-analyst")

	env.waitFor(15*time.Second, "queue to pause after disabled-failover transient failure", func() bool {
		snap := env.queueSnapshot()
		paused, _ := snap["paused"].(bool)
		return paused
	})

	snap := env.queueSnapshot()
	if until, _ := snap["paused_until"].(string); until == "" {
		t.Error("expected non-empty paused_until banner when auto_switch is disabled")
	}

	// The agent's provider must NOT have been switched: config unchanged...
	ag, ok := findFailoverAgentConfig(env.loadConfig(), "requirements-analyst")
	if !ok {
		t.Fatal("requirements-analyst missing from disk config")
	}
	if ag.Provider != "anthropic-cloud" {
		t.Errorf("provider should not have switched with auto_switch disabled, got %q", ag.Provider)
	}
	// ...and no operations.yaml override recorded either.
	resp := env.doRequest("GET", "/api/p/testproject/provider-switch/status", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	if active, _ := data["failover_active"].(bool); active {
		t.Error("expected failover_active=false with auto_switch disabled")
	}
}

// failoverProjectWideCfgYAML wires three analyst agents all primary on
// anthropic-cloud: agent-x and agent-y each have a gemini-cloud secondary;
// agent-z has none, exercising the FR-3.4 "no secondary -> partial pause"
// branch of a single project-wide failover action.
const failoverProjectWideCfgYAML = `git:
  default_branch: main
  branch_template: "ticket/{slug}"

roles:
  - product-owner
  - analyst

stages:
  - {name: ideas,          dir: ideas}
  - {name: requirements,   dir: requirements}
  - {name: backend-plans,  dir: backend-plans}
  - {name: frontend-plans, dir: frontend-plans}
  - {name: test-plans,     dir: test-plans}
  - {name: tests,          dir: tests}
  - {name: prototypes,     dir: prototypes}
  - {name: releases,       dir: releases}
  - {name: sprints,        dir: sprints}
  - {name: defects,        dir: defects}

users:
  - email: admin@test.local
    roles: [product-owner]

required_plans:
  ticket: []
  epic: []

switchover:
  automated_switchover: true

agents:
  - name: agent-x
    role: [analyst]
    driver: claude-code-cli
    provider: anthropic-cloud
    model: claude-3-7-sonnet
    fallback_provider: gemini-cloud
    fallback_model: gemini-2.5-flash
    allowed_write_paths: [lifecycle/requirements, lifecycle/ideas]
    git_identity:
      name: Agent X
      email: agent-x@test.local
    prompt_templates:
      analyst: "x"
  - name: agent-y
    role: [analyst]
    driver: claude-code-cli
    provider: anthropic-cloud
    model: claude-3-7-sonnet
    fallback_provider: gemini-cloud
    fallback_model: gemini-2.5-flash
    allowed_write_paths: [lifecycle/requirements, lifecycle/ideas]
    git_identity:
      name: Agent Y
      email: agent-y@test.local
    prompt_templates:
      analyst: "y"
  - name: agent-z
    role: [analyst]
    driver: claude-code-cli
    provider: anthropic-cloud
    model: claude-3-7-sonnet
    allowed_write_paths: [lifecycle/requirements, lifecycle/ideas]
    git_identity:
      name: Agent Z
      email: agent-z@test.local
    prompt_templates:
      analyst: "z"
`

// TestFailover_ProjectWide_MultiAgentDrain (Milestone 2/FR-3): a single
// transient failure on agent-x's provider must move every agent bound to
// that provider in one action — agent-y (which has its own secondary,
// switches too) and agent-z (no secondary configured, partially paused
// instead, FR-3.4) — while agent-x's interrupted job restarts and
// agent-y's already-queued job continues to drain normally on the
// secondary. agent-z's job, blocked by the partial pause, must not run.
func TestFailover_ProjectWide_MultiAgentDrain(t *testing.T) {
	gemini := newMockProvider(t, true)
	providers := []config.Provider{
		{Name: "anthropic-cloud", BaseURL: "http://127.0.0.1:1", Driver: "openai-compatible"},
		{Name: "gemini-cloud", BaseURL: gemini.URL, Driver: "openai-compatible"},
	}

	markerPath := filepath.Join(t.TempDir(), "drain-invoked")
	errorJSON := `{"error":{"type":"overloaded_error","message":"HTTP 529 Overloaded"}}`
	setupFakeClaudeWithScript(t, failoverThenSucceedScript(markerPath, errorJSON))

	env := newFailoverTestEnv(t, failoverProjectWideCfgYAML, providers, []seedArtifact{
		{relPath: "lifecycle/ideas/drain-x.md", content: makeApprovedArtifact("Drain X", "idea", "drain-x")},
		{relPath: "lifecycle/ideas/drain-y.md", content: makeApprovedArtifact("Drain Y", "idea", "drain-y")},
		{relPath: "lifecycle/ideas/drain-z.md", content: makeApprovedArtifact("Drain Z", "idea", "drain-z")},
	})

	env.enqueue("lifecycle/ideas/drain-x.md", "agent-x")

	env.waitFor(15*time.Second, "agent-x's requeued job to complete on the fallback", func() bool {
		snap := env.queueSnapshot()
		if paused, _ := snap["paused"].(bool); paused {
			t.Fatal("queue should not pause on automated project-wide failover, but it did")
		}
		j := findJobByPath(snap, "lifecycle/ideas/drain-x.md")
		if j == nil {
			return false
		}
		attempts, _ := j["attempts"].(float64)
		return j["state"] == "completed" && attempts == 2
	})

	// Every agent bound to anthropic-cloud moved in the same action, whether
	// or not it had a job in flight.
	assertAgentFailedOverTo(t, env, "agent-x", "anthropic-cloud", "gemini-cloud")
	assertAgentFailedOverTo(t, env, "agent-y", "anthropic-cloud", "gemini-cloud")

	statusResp := env.doRequest("GET", "/api/p/testproject/provider-switch/status", nil)
	requireStatus(t, statusResp, 200)
	statusData := readJSON(t, statusResp)
	agents, _ := statusData["agents"].([]any)
	var zPartialPause bool
	for _, raw := range agents {
		a, _ := raw.(map[string]any)
		if a["agent"] == "agent-z" {
			zPartialPause, _ = a["partial_pause"].(bool)
		}
	}
	if !zPartialPause {
		t.Error("expected agent-z (no secondary configured) to be partially paused after project-wide failover")
	}

	// The rest of the queue continues draining on the secondary: agent-y's
	// job, enqueued after the failover already engaged, completes normally.
	env.enqueue("lifecycle/ideas/drain-y.md", "agent-y")
	env.waitFor(15*time.Second, "agent-y's job to complete on the secondary", func() bool {
		snap := env.queueSnapshot()
		j := findJobByPath(snap, "lifecycle/ideas/drain-y.md")
		return j != nil && j["state"] == "completed"
	})

	// agent-z is partially paused: its job must not be picked up by the
	// dispatcher while the pause holds.
	env.enqueue("lifecycle/ideas/drain-z.md", "agent-z")
	time.Sleep(1 * time.Second)
	snap := env.queueSnapshot()
	j := findJobByPath(snap, "lifecycle/ideas/drain-z.md")
	if j == nil {
		t.Fatal("expected agent-z's job to still be queued")
	}
	if j["state"] != "pending" {
		t.Errorf("expected agent-z's job to remain pending while partially paused, got state=%v", j["state"])
	}
}

// TestFailover_AutoSwitch_AuthError (Milestone 2/FR-2.3): a classified
// auth_error failure (provider API key rejected — HTTP 401, distinct from
// Claude Code's own OAuth token rotation) triggers the same automated
// project-wide failover path as rate_limit/overloaded when
// automated_switchover is enabled.
func TestFailover_AutoSwitch_AuthError(t *testing.T) {
	gemini := newMockProvider(t, true)
	providers := []config.Provider{
		{Name: "anthropic-cloud", BaseURL: "http://127.0.0.1:1", Driver: "openai-compatible"},
		{Name: "gemini-cloud", BaseURL: gemini.URL, Driver: "openai-compatible"},
	}

	markerPath := filepath.Join(t.TempDir(), "auth-err-invoked")
	script := fmt.Sprintf(`if [ -f %s ]; then
%sexit 0
else
touch %s
printf '%%s\n' '{"type":"system","subtype":"init","permissionMode":"bypassPermissions","model":"claude-sonnet-4-6"}'
>&2 printf '%%s\n' 'authentication failed: HTTP 401 invalid api key'
exit 1
fi
`, markerPath, fakeClaudeSuccessEvents, markerPath)
	setupFakeClaudeWithScript(t, script)

	env := newFailoverTestEnv(t, failoverAutoCfgYAML, providers, []seedArtifact{
		{relPath: "lifecycle/ideas/auth-err-idea.md", content: makeApprovedArtifact("Auth Err Idea", "idea", "auth-err-idea")},
	})

	env.enqueue("lifecycle/ideas/auth-err-idea.md", "requirements-analyst")

	env.waitFor(15*time.Second, "requeued job to complete after auth_error failover", func() bool {
		snap := env.queueSnapshot()
		if paused, _ := snap["paused"].(bool); paused {
			t.Fatal("queue should not pause on an auth_error automated failover, but it did")
		}
		j := findJobByPath(snap, "lifecycle/ideas/auth-err-idea.md")
		if j == nil {
			return false
		}
		attempts, _ := j["attempts"].(float64)
		return j["state"] == "completed" && attempts == 2
	})

	assertAgentFailedOverTo(t, env, "requirements-analyst", "anthropic-cloud", "gemini-cloud")
}
