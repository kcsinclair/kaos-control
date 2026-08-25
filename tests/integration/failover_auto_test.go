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
	"strings"
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
    active_status: clarifying
    source_types: [idea]
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
    active_status: clarifying
    source_types: [idea]
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

// TestFailover_AutoSwitch_HTTP529 (FA1): a job fails with an HTTP 529
// "overloaded_error" stream event. With auto_switch enabled and a healthy
// fallback, the dispatcher must: switch the agent's config on disk (provider
// -> fallback, primary_provider stashed), commit the change to git,
// broadcast provider.switched, and immediately retry the stalled job at the
// head of the queue without pausing — the retried job completes on the
// fallback.
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

	// (1) lifecycle/config.yaml updated with primary_provider/provider.
	ag, ok := findFailoverAgentConfig(env.loadConfig(), "requirements-analyst")
	if !ok {
		t.Fatal("requirements-analyst missing from disk config after failover")
	}
	if ag.PrimaryProvider != "anthropic-cloud" {
		t.Errorf("expected lifecycle/config.yaml to record primary_provider: anthropic-cloud, got %q", ag.PrimaryProvider)
	}
	if ag.Provider != "gemini-cloud" {
		t.Errorf("expected lifecycle/config.yaml to switch provider to gemini-cloud, got %q", ag.Provider)
	}

	// (2) git commit created.
	found := false
	for _, msg := range env.gitLogMessages(10) {
		if strings.Contains(msg, "failover(agent):") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected a failover(agent): commit in lifecycle/config.yaml history")
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

	ag, ok := findFailoverAgentConfig(env.loadConfig(), "requirements-analyst")
	if !ok {
		t.Fatal("requirements-analyst missing from disk config after failover")
	}
	if ag.Provider != "gemini-cloud" {
		t.Errorf("expected lifecycle/config.yaml to switch provider to gemini-cloud, got %q", ag.Provider)
	}
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

	// The agent's provider must NOT have been switched.
	ag, ok := findFailoverAgentConfig(env.loadConfig(), "requirements-analyst")
	if !ok {
		t.Fatal("requirements-analyst missing from disk config")
	}
	if ag.Provider != "anthropic-cloud" {
		t.Errorf("provider should not have switched with auto_switch disabled, got %q", ag.Provider)
	}
}
