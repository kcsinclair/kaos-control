// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/index"
)

// TestListArtifacts_AgentRunCount verifies that GET /api/p/:project/artifacts
// enriches each artifact with agent_run_count (always present, even when 0)
// and active_agent_status (omitted when no active run).
func TestListArtifacts_AgentRunCount(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/arc-http-a.md",
			content: makeArtifact("ARC HTTP Idea A", "idea", "draft", "arc-http-a", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/arc-http-b.md",
			content: makeArtifact("ARC HTTP Idea B", "idea", "draft", "arc-http-b", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/arc-http-c.md",
			content: makeArtifact("ARC HTTP Idea C", "idea", "draft", "arc-http-c", "", "Body."),
		},
	}

	env := newTestEnv(t, seeds)
	now := time.Now()

	// Seed 3 runs for idea A (all completed statuses).
	for _, r := range []*index.AgentRunRow{
		{RunID: "http-a-0", AgentName: "test-agent", Role: "developer", TargetPath: "lifecycle/ideas/arc-http-a.md", StartedAt: now, Status: "done"},
		{RunID: "http-a-1", AgentName: "test-agent", Role: "developer", TargetPath: "lifecycle/ideas/arc-http-a.md", StartedAt: now, Status: "failed"},
		{RunID: "http-a-2", AgentName: "test-agent", Role: "developer", TargetPath: "lifecycle/ideas/arc-http-a.md", StartedAt: now, Status: "done"},
	} {
		if err := env.proj.Idx.InsertAgentRun(r); err != nil {
			t.Fatalf("InsertAgentRun: %v", err)
		}
	}

	// Seed 1 running run for idea B — active_agent_status must be "running".
	if err := env.proj.Idx.InsertAgentRun(&index.AgentRunRow{
		RunID: "http-b-0", AgentName: "test-agent", Role: "developer",
		TargetPath: "lifecycle/ideas/arc-http-b.md", StartedAt: now, Status: "running",
	}); err != nil {
		t.Fatalf("InsertAgentRun: %v", err)
	}

	// Idea C: zero runs.

	resp := env.doRequest("GET", "/api/p/testproject/artifacts", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	if len(items) == 0 {
		t.Fatal("expected items in response")
	}

	byPath := make(map[string]map[string]any, len(items))
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		path, _ := item["path"].(string)
		byPath[path] = item
	}

	// ── idea A: 3 completed runs, no active run ──────────────────────────────
	a, ok := byPath["lifecycle/ideas/arc-http-a.md"]
	if !ok {
		t.Fatal("arc-http-a.md not found in response")
	}
	if count, _ := a["agent_run_count"].(float64); int(count) != 3 {
		t.Errorf("arc-http-a: agent_run_count = %v, want 3", a["agent_run_count"])
	}
	if _, present := a["active_agent_status"]; present {
		t.Errorf("arc-http-a: active_agent_status should be omitted (all runs completed), got %v", a["active_agent_status"])
	}

	// ── idea B: 1 running run ────────────────────────────────────────────────
	b, ok := byPath["lifecycle/ideas/arc-http-b.md"]
	if !ok {
		t.Fatal("arc-http-b.md not found in response")
	}
	if count, _ := b["agent_run_count"].(float64); int(count) != 1 {
		t.Errorf("arc-http-b: agent_run_count = %v, want 1", b["agent_run_count"])
	}
	if status, _ := b["active_agent_status"].(string); status != "running" {
		t.Errorf("arc-http-b: active_agent_status = %q, want running", status)
	}

	// ── idea C: 0 runs — agent_run_count must be present as 0, never omitted ─
	c, ok := byPath["lifecycle/ideas/arc-http-c.md"]
	if !ok {
		t.Fatal("arc-http-c.md not found in response")
	}
	rawCount, present := c["agent_run_count"]
	if !present {
		t.Error("arc-http-c: agent_run_count field is absent; must be 0 not omitted")
	} else if count, _ := rawCount.(float64); int(count) != 0 {
		t.Errorf("arc-http-c: agent_run_count = %v, want 0", rawCount)
	}
	if _, present := c["active_agent_status"]; present {
		t.Errorf("arc-http-c: active_agent_status should be omitted (no runs), got %v", c["active_agent_status"])
	}
}
