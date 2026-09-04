// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Suite — Milestone 9: observability & reports aggregation (FR-10).
//
// Every switch/failover/restore transition is logged (secret-free) and
// aggregated by internal/reports.FailoverUsage into per-agent and
// per-provider summaries, exposed at GET .../reports/failover.

import (
	"testing"
	"time"
)

// TestFailoverReport_Aggregation (Milestone 9/FR-10.2): after a
// project-wide failover (agent-b/c/d, seeded via newProviderSwitchAPIEnv)
// and a restore for one of them, the reports API aggregates failover
// count, causing provider, time on secondary, and restore state correctly
// per agent and per provider.
func TestFailoverReport_Aggregation(t *testing.T) {
	env := newProviderSwitchAPIEnv(t)

	// Give agent-b a moment on the secondary before restoring it, so
	// TimeOnSecondaryMs/MeanTimeToRestoreMs are meaningfully non-zero.
	time.Sleep(1100 * time.Millisecond)
	restoreResp := env.doRequest("POST", "/api/p/testproject/agents/agent-b/restore-provider", nil)
	requireStatus(t, restoreResp, 200)

	resp := env.doRequest("GET", "/api/p/testproject/reports/failover", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	perAgent, _ := data["per_agent"].([]any)
	byAgent := map[string]map[string]any{}
	for _, raw := range perAgent {
		a, _ := raw.(map[string]any)
		byAgent[a["agent"].(string)] = a
	}

	b, ok := byAgent["agent-b"]
	if !ok {
		t.Fatal("expected agent-b in per_agent report")
	}
	if fc, _ := b["failover_count"].(float64); fc != 1 {
		t.Errorf("agent-b failover_count: got %v, want 1", fc)
	}
	if rc, _ := b["restore_count"].(float64); rc != 1 {
		t.Errorf("agent-b restore_count: got %v, want 1", rc)
	}
	if onSecondary, _ := b["currently_on_secondary"].(bool); onSecondary {
		t.Error("agent-b currently_on_secondary: got true, want false (restored)")
	}
	if ms, _ := b["time_on_secondary_ms"].(float64); ms <= 0 {
		t.Errorf("agent-b time_on_secondary_ms: got %v, want > 0", ms)
	}
	if ms, _ := b["mean_time_to_restore_ms"].(float64); ms <= 0 {
		t.Errorf("agent-b mean_time_to_restore_ms: got %v, want > 0", ms)
	}

	c, ok := byAgent["agent-c"]
	if !ok {
		t.Fatal("expected agent-c in per_agent report")
	}
	if fc, _ := c["failover_count"].(float64); fc != 1 {
		t.Errorf("agent-c failover_count: got %v, want 1", fc)
	}
	if rc, _ := c["restore_count"].(float64); rc != 0 {
		t.Errorf("agent-c restore_count: got %v, want 0 (never restored)", rc)
	}
	if onSecondary, _ := c["currently_on_secondary"].(bool); !onSecondary {
		t.Error("agent-c currently_on_secondary: got false, want true (still failed over)")
	}

	perProvider, _ := data["per_provider"].([]any)
	byProvider := map[string]map[string]any{}
	for _, raw := range perProvider {
		p, _ := raw.(map[string]any)
		byProvider[p["provider"].(string)] = p
	}
	anthropic, ok := byProvider["anthropic-cloud"]
	if !ok {
		t.Fatal("expected anthropic-cloud (the causing provider) in per_provider report")
	}
	if fc, _ := anthropic["failover_count"].(float64); fc != 3 {
		t.Errorf("anthropic-cloud failover_count: got %v, want 3 (agent-b/c/d)", fc)
	}
	agents, _ := anthropic["agents"].([]any)
	agentSet := map[string]bool{}
	for _, a := range agents {
		agentSet[a.(string)] = true
	}
	for _, name := range []string{"agent-b", "agent-c", "agent-d"} {
		if !agentSet[name] {
			t.Errorf("expected %s in anthropic-cloud's affected agents %v", name, agents)
		}
	}
}
