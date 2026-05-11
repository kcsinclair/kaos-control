// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestone 4 — E2E Smoke Test: Agents Screen Shows Distinct Ready Counts
//
// An API-level smoke test that exercises the full ready-counts flow:
//   1. Start a server with agents configured to use source_types.
//   2. Seed distinct artifact populations per agent type.
//   3. Hit GET /agents/ready-counts.
//   4. Assert that at least two agents report different counts.
//
// These tests are skipped in -short mode (go test -short) because they spin up
// a full HTTP server and hit a live SQLite index.

import (
	"testing"
)

// TestAgentsReadyCounts_SmokeDistinctCounts is an API-level smoke test that
// verifies the full agents → ready-counts pipeline returns distinct values for
// agents whose source_types do not overlap.
//
// Test is skipped in -short mode.
func TestAgentsReadyCounts_SmokeDistinctCounts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping smoke test in -short mode")
	}

	// Seed one approved plan-backend and three approved plan-frontend so that
	// backend-developer (1) and frontend-developer (3) are distinct.
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/backend-plans/smoke-be-1-3-be.md",
			content: makeArtifact("Smoke BE Plan 1", "plan-backend", "approved", "smoke-be-1", "", "Body."),
		},
		{
			relPath: "lifecycle/frontend-plans/smoke-fe-1-4-fe.md",
			content: makeArtifact("Smoke FE Plan 1", "plan-frontend", "approved", "smoke-fe-1", "", "Body."),
		},
		{
			relPath: "lifecycle/frontend-plans/smoke-fe-2-4-fe.md",
			content: makeArtifact("Smoke FE Plan 2", "plan-frontend", "approved", "smoke-fe-2", "", "Body."),
		},
		{
			relPath: "lifecycle/frontend-plans/smoke-fe-3-4-fe.md",
			content: makeArtifact("Smoke FE Plan 3", "plan-frontend", "approved", "smoke-fe-3", "", "Body."),
		},
	}

	env := newAgentTestEnvWithCfg(t, readyCountsCfgYAML, seeds)
	env.login("admin@test.local", "admin-pass-123")

	counts := getReadyCounts(t, env)

	// Collect all numeric count values.
	var values []int
	for _, v := range counts {
		switch n := v.(type) {
		case float64:
			values = append(values, int(n))
		case int:
			values = append(values, n)
		}
	}

	if len(values) < 2 {
		t.Fatalf("smoke test: expected at least 2 agents in ready-counts, got %d", len(values))
	}

	// At least one pair of agents must have different counts.
	allSame := true
	for i := 1; i < len(values); i++ {
		if values[i] != values[0] {
			allSame = false
			break
		}
	}
	if allSame {
		t.Errorf("smoke test: all ready-counts are equal (%v); expected at least two agents with distinct values", values)
	}
}

// TestAgentsReadyCounts_SmokeBackendVsFrontend is a targeted smoke test that
// confirms the two developer agents (with distinct source_types) report
// different counts after targeted seeding of approved artifacts.
func TestAgentsReadyCounts_SmokeBackendVsFrontend(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping smoke test in -short mode")
	}

	seeds := []seedArtifact{
		// Two plan-backend in-development.
		{
			relPath: "lifecycle/backend-plans/bvf-be-1-3-be.md",
			content: makeArtifact("BvF BE Plan 1", "plan-backend", "approved", "bvf-be-1", "", "Body."),
		},
		{
			relPath: "lifecycle/backend-plans/bvf-be-2-3-be.md",
			content: makeArtifact("BvF BE Plan 2", "plan-backend", "approved", "bvf-be-2", "", "Body."),
		},
		// One plan-frontend in-development.
		{
			relPath: "lifecycle/frontend-plans/bvf-fe-1-4-fe.md",
			content: makeArtifact("BvF FE Plan 1", "plan-frontend", "approved", "bvf-fe-1", "", "Body."),
		},
	}

	env := newAgentTestEnvWithCfg(t, readyCountsCfgYAML, seeds)
	env.login("admin@test.local", "admin-pass-123")

	counts := getReadyCounts(t, env)

	be := countFor(counts, "backend-developer")
	fe := countFor(counts, "frontend-developer")

	if be != 2 {
		t.Errorf("smoke: backend-developer want 2, got %d", be)
	}
	if fe != 1 {
		t.Errorf("smoke: frontend-developer want 1, got %d", fe)
	}
	if be == fe {
		t.Errorf("smoke: backend-developer (%d) == frontend-developer (%d); counts must differ", be, fe)
	}
}
