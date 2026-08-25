// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Suite 2.3 — Recovery probing & alerts integration test (FR1).

import (
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
)

// failoverRecoveryCfgYAML seeds a single agent already in a failover state
// (provider = gemini-cloud, primary_provider = anthropic-cloud stashed) and
// configures a 1-second probe interval so the background RecoveryProber
// cycles fast enough for an integration test.
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
    provider: gemini-cloud
    model: gemini-2.5-flash
    primary_provider: anthropic-cloud
    primary_model: claude-3-7-sonnet
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: Agent A
      email: agent-a@test.local
    prompt_templates:
      analyst: "x"
`

// TestRecovery_ProbeAndAlert (FR1): with agent-a stashed in failover
// (primary_provider=anthropic-cloud), the background RecoveryProber
// re-probes the primary provider every probe_interval_seconds. Once the
// primary (initially down) starts responding 200 OK, two consecutive
// healthy probes must trigger a provider.primary_recovered broadcast over
// the project WebSocket.
func TestRecovery_ProbeAndAlert(t *testing.T) {
	anthropic := newMockProvider(t, false) // primary starts down
	gemini := newMockProvider(t, true)
	providers := []config.Provider{
		{Name: "anthropic-cloud", BaseURL: anthropic.URL, Driver: "openai-compatible"},
		{Name: "gemini-cloud", BaseURL: gemini.URL, Driver: "openai-compatible"},
	}

	env := newFailoverTestEnv(t, failoverRecoveryCfgYAML, providers, nil)
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
