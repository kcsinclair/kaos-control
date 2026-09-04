// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Suite 2.3 — Recovery probing & alerts integration test (FR1).

import (
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
)

// failoverRecoveryCfgYAML wires agent-a with a normal anthropic-cloud
// primary and a gemini-cloud secondary, and configures a 1-second probe
// interval so the background RecoveryProber cycles fast enough for an
// integration test. Failover state (if any) is seeded via operations.yaml
// at test time — agent-switchover-and-failover Milestone 6 reads
// failed-over status exclusively from there, never from config.
const failoverRecoveryCfgYAML = `git:
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

provider_failover:
  probe_interval_seconds: 1

agents:
  - name: agent-a
    role: [analyst]
    driver: claude-code-cli
    provider: anthropic-cloud
    model: claude-3-7-sonnet
    fallback_provider: gemini-cloud
    fallback_model: gemini-2.5-flash
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: Agent A
      email: agent-a@test.local
    prompt_templates:
      analyst: "x"
`

// failoverSingleProviderCfgYAML wires agent-a with only a primary provider
// configured — no fallback_provider at all (single-provider mode, FR-1).
const failoverSingleProviderCfgYAML = `git:
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

provider_failover:
  probe_interval_seconds: 1

agents:
  - name: agent-a
    role: [analyst]
    driver: claude-code-cli
    provider: anthropic-cloud
    model: claude-3-7-sonnet
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: Agent A
      email: agent-a@test.local
    prompt_templates:
      analyst: "x"
`

// TestRecovery_ProbeAndAlert (FR1): with agent-a's operations.yaml state
// recording a failover away from anthropic-cloud, the background
// RecoveryProber re-probes every provider bound to any agent (primary or
// fallback) every probe_interval_seconds. Once the primary (initially down)
// starts responding 200 OK, two consecutive healthy probes must trigger a
// provider.primary_recovered broadcast over the project WebSocket.
func TestRecovery_ProbeAndAlert(t *testing.T) {
	anthropic := newMockProvider(t, false) // primary starts down
	gemini := newMockProvider(t, true)
	providers := []config.Provider{
		{Name: "anthropic-cloud", BaseURL: anthropic.URL, Driver: "openai-compatible"},
		{Name: "gemini-cloud", BaseURL: gemini.URL, Driver: "openai-compatible"},
	}

	env := newFailoverTestEnv(t, failoverRecoveryCfgYAML, providers, nil)
	env.seedFailoverState("agent-a", "gemini-cloud", "gemini-2.5-flash", "seed-fr1")
	ws := env.connectProjectWS()

	env.proj.StartRecoveryProber(env.ctx)

	// Give the prober a moment to run its first (unhealthy) probe cycle,
	// then bring the primary back online so the next two consecutive probes
	// are healthy.
	time.Sleep(1200 * time.Millisecond)
	anthropic.setHealthy(true)

	payload := ws.waitForEventType(t, 10*time.Second, "provider.primary_recovered")
	if payload["provider"] != "anthropic-cloud" {
		t.Errorf("provider.primary_recovered provider: got %v, want anthropic-cloud", payload["provider"])
	}
}

// TestRecovery_SingleProviderReachability (Milestone 6/FR-5.1): even with no
// agent ever having failed over — single-provider mode, no fallback
// configured at all — the background prober still probes every provider
// bound to any agent and records its reachability, surfaced through
// GET .../provider-switch/status. Reachability must not be gated on
// failover ever having happened.
func TestRecovery_SingleProviderReachability(t *testing.T) {
	anthropic := newMockProvider(t, true)
	providers := []config.Provider{
		{Name: "anthropic-cloud", BaseURL: anthropic.URL, Driver: "openai-compatible"},
	}

	env := newFailoverTestEnv(t, failoverSingleProviderCfgYAML, providers, nil)
	env.proj.StartRecoveryProber(env.ctx)

	env.waitFor(10*time.Second, "anthropic-cloud reachability to appear in provider-switch/status", func() bool {
		resp := env.doRequest("GET", "/api/p/testproject/provider-switch/status", nil)
		requireStatus(t, resp, 200)
		data := readJSON(t, resp)
		reachability, _ := data["reachability"].(map[string]any)
		entry, ok := reachability["anthropic-cloud"].(map[string]any)
		if !ok {
			return false
		}
		healthy, _ := entry["healthy"].(bool)
		return healthy
	})

	// Confirm failover_active stays false throughout — reachability tracking
	// is independent of ever having failed over.
	resp := env.doRequest("GET", "/api/p/testproject/provider-switch/status", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	if active, _ := data["failover_active"].(bool); active {
		t.Error("expected failover_active=false in single-provider mode")
	}
}

// TestRecovery_QuotaGatedRecovery_SuppressedUntilReset (Milestone 6/FR-9.3):
// after a rate-limit-triggered failover with a future resets_at_unix, the
// primary must not be announced as recovered — even once /v1/models probes
// come back healthy — until that reset time has passed.
func TestRecovery_QuotaGatedRecovery_SuppressedUntilReset(t *testing.T) {
	anthropic := newMockProvider(t, true) // healthy from the start
	gemini := newMockProvider(t, true)
	providers := []config.Provider{
		{Name: "anthropic-cloud", BaseURL: anthropic.URL, Driver: "openai-compatible"},
		{Name: "gemini-cloud", BaseURL: gemini.URL, Driver: "openai-compatible"},
	}

	env := newFailoverTestEnv(t, failoverRecoveryCfgYAML, providers, nil)

	// Seed a quota failover whose reset time is safely in the future: even
	// though anthropic-cloud responds healthy immediately, recovery must be
	// suppressed until this time passes.
	farFuture := time.Now().Add(1 * time.Hour).Unix()
	if _, _, err := env.proj.FailoverProviderWide("anthropic-cloud", "rate_limit", farFuture, "five_hour"); err != nil {
		t.Fatalf("seeding quota failover: %v", err)
	}

	ws := env.connectProjectWS()
	env.proj.StartRecoveryProber(env.ctx)

	// Several healthy probe cycles pass; provider.primary_recovered must not
	// fire because the recorded reset time is still in the future.
	deadline := time.After(3 * time.Second)
drain:
	for {
		select {
		case msg := <-ws.events:
			if msg["type"] == "provider.primary_recovered" {
				t.Fatal("provider.primary_recovered fired before the recorded reset time — recovery should be quota-gated")
			}
		case <-deadline:
			break drain
		}
	}

	statusResp := env.doRequest("GET", "/api/p/testproject/provider-switch/status", nil)
	requireStatus(t, statusResp, 200)
	statusData := readJSON(t, statusResp)
	if active, _ := statusData["failover_active"].(bool); !active {
		t.Error("expected failover_active=true while the quota-gated failover holds")
	}
}
