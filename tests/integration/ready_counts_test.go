// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestone 1 — Per-Agent Role-Specific Ready Counts: Backend Integration Tests
//
// Tests for GET /api/p/:project/agents/ready-counts that verify each agent
// returns a count scoped to its configured source_types (artifact type filter)
// combined with its active_status.
//
// Configuration used by these tests defines three agents with source_types:
//
//	requirements-analyst  active_status=clarifying  source_types=[idea]
//	backend-developer     active_status=in-development  source_types=[plan-backend]
//	frontend-developer    active_status=in-development  source_types=[plan-frontend]
//
// Because backend-developer and frontend-developer share the same active_status
// but have different source_types, they must return distinct counts whenever the
// seed data contains a different number of matching artifacts per type.

import (
	"net/http"
	"testing"
)

// readyCountsCfgYAML is the lifecycle/config.yaml for ready-counts role-specific tests.
// It defines three agents, each with a distinct source_types list, so that their
// ready counts are derived from non-overlapping artifact subsets.
const readyCountsCfgYAML = `git:
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
    roles: [product-owner, analyst, reviewer, approver]
  - email: dev@test.local
    roles: [backend-developer, frontend-developer, test-developer]
  - email: qa@test.local
    roles: [qa]

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []

agents:
  - name: requirements-analyst
    role: [analyst]
    driver: claude-code-cli
    active_status: clarifying
    source_types: [idea]
    allowed_write_paths:
      - lifecycle/requirements
    git_identity:
      name: Requirements Analyst
      email: requirements-analyst@test.local
    prompt_templates:
      analyst: "Test prompt for {target_path}"

  - name: backend-developer
    role: [backend-developer]
    driver: claude-code-cli
    active_status: in-development
    source_types: [plan-backend]
    allowed_write_paths:
      - lifecycle/backend-plans
    git_identity:
      name: Backend Developer
      email: backend-developer@test.local
    prompt_templates:
      backend-developer: "Test prompt for {target_path}"

  - name: frontend-developer
    role: [frontend-developer]
    driver: claude-code-cli
    active_status: in-development
    source_types: [plan-frontend]
    allowed_write_paths:
      - lifecycle/frontend-plans
    git_identity:
      name: Frontend Developer
      email: frontend-developer@test.local
    prompt_templates:
      frontend-developer: "Test prompt for {target_path}"
`

// TestReadyCounts_PerAgentSourceTypes seeds one artifact of each relevant
// type/status combination and asserts that each agent receives a count derived
// exclusively from its own source_types.
//
// Acceptance criteria covered:
//   - Seeds idea/clarifying, plan-backend/in-development, plan-frontend/in-development
//   - requirements-analyst count == number of clarifying ideas (1)
//   - backend-developer count == number of in-development plan-backend artifacts (1)
//   - frontend-developer count == number of in-development plan-frontend artifacts (2)
//   - backend-developer count != frontend-developer count
func TestReadyCounts_PerAgentSourceTypes(t *testing.T) {
	seeds := []seedArtifact{
		// One clarifying idea — requirements-analyst should count this.
		{
			relPath: "lifecycle/ideas/src-types-idea-1.md",
			content: makeArtifact("Source Types Idea 1", "idea", "clarifying", "src-types-idea-1", "", "Body."),
		},
		// One in-development plan-backend — backend-developer should count this.
		{
			relPath: "lifecycle/backend-plans/src-types-be-1-3-be.md",
			content: makeArtifact("Source Types BE Plan 1", "plan-backend", "in-development", "src-types-be-1", "", "Body."),
		},
		// Two in-development plan-frontend — frontend-developer should count both.
		{
			relPath: "lifecycle/frontend-plans/src-types-fe-1-4-fe.md",
			content: makeArtifact("Source Types FE Plan 1", "plan-frontend", "in-development", "src-types-fe-1", "", "Body."),
		},
		{
			relPath: "lifecycle/frontend-plans/src-types-fe-2-4-fe.md",
			content: makeArtifact("Source Types FE Plan 2", "plan-frontend", "in-development", "src-types-fe-2", "", "Body."),
		},
		// A plan-backend with a different status — must NOT be counted by backend-developer.
		{
			relPath: "lifecycle/backend-plans/src-types-be-draft-3-be.md",
			content: makeArtifact("Source Types BE Draft", "plan-backend", "draft", "src-types-be-draft", "", "Body."),
		},
		// A plan-frontend with a different status — must NOT be counted by frontend-developer.
		{
			relPath: "lifecycle/frontend-plans/src-types-fe-draft-4-fe.md",
			content: makeArtifact("Source Types FE Draft", "plan-frontend", "draft", "src-types-fe-draft", "", "Body."),
		},
	}

	env := newAgentTestEnvWithCfg(t, readyCountsCfgYAML, seeds)
	env.login("admin@test.local", "admin-pass-123")

	counts := getReadyCounts(t, env)

	// requirements-analyst: 1 clarifying idea.
	if got := countFor(counts, "requirements-analyst"); got != 1 {
		t.Errorf("requirements-analyst: want count 1, got %d", got)
	}

	// backend-developer: 1 in-development plan-backend.
	if got := countFor(counts, "backend-developer"); got != 1 {
		t.Errorf("backend-developer: want count 1, got %d", got)
	}

	// frontend-developer: 2 in-development plan-frontend.
	if got := countFor(counts, "frontend-developer"); got != 2 {
		t.Errorf("frontend-developer: want count 2, got %d", got)
	}

	// backend-developer and frontend-developer share active_status=in-development but
	// must have different counts because their source_types do not overlap.
	backendCount := countFor(counts, "backend-developer")
	frontendCount := countFor(counts, "frontend-developer")
	if backendCount == frontendCount {
		t.Errorf("backend-developer (%d) and frontend-developer (%d) must have distinct counts; "+
			"shared active_status with different source_types must produce role-specific results",
			backendCount, frontendCount)
	}
}

// TestReadyCounts_SourceTypesExcludesWrongType verifies that an agent with
// source_types=[plan-backend] does NOT count plan-frontend artifacts even when
// they share the same active_status (in-development).
func TestReadyCounts_SourceTypesExcludesWrongType(t *testing.T) {
	seeds := []seedArtifact{
		// Two in-development plan-frontend (wrong type for backend-developer).
		{
			relPath: "lifecycle/frontend-plans/excl-fe-1-4-fe.md",
			content: makeArtifact("Exclude FE Plan 1", "plan-frontend", "in-development", "excl-fe-1", "", "Body."),
		},
		{
			relPath: "lifecycle/frontend-plans/excl-fe-2-4-fe.md",
			content: makeArtifact("Exclude FE Plan 2", "plan-frontend", "in-development", "excl-fe-2", "", "Body."),
		},
	}

	env := newAgentTestEnvWithCfg(t, readyCountsCfgYAML, seeds)
	env.login("admin@test.local", "admin-pass-123")

	counts := getReadyCounts(t, env)

	// backend-developer must see 0: no plan-backend artifacts exist.
	if got := countFor(counts, "backend-developer"); got != 0 {
		t.Errorf("backend-developer: want count 0 (no plan-backend artifacts), got %d", got)
	}

	// frontend-developer must see 2: two plan-frontend in-development exist.
	if got := countFor(counts, "frontend-developer"); got != 2 {
		t.Errorf("frontend-developer: want count 2, got %d", got)
	}
}

// TestReadyCounts_SourceTypesExcludesWrongStatus verifies that an agent with
// source_types=[plan-backend] and active_status=in-development does NOT count
// plan-backend artifacts that have a different status (e.g. draft).
func TestReadyCounts_SourceTypesExcludesWrongStatus(t *testing.T) {
	seeds := []seedArtifact{
		// One plan-backend in draft status — must NOT count.
		{
			relPath: "lifecycle/backend-plans/ws-draft-3-be.md",
			content: makeArtifact("WS Draft BE", "plan-backend", "draft", "ws-draft", "", "Body."),
		},
		// One plan-backend in planning status — must NOT count.
		{
			relPath: "lifecycle/backend-plans/ws-planning-3-be.md",
			content: makeArtifact("WS Planning BE", "plan-backend", "planning", "ws-planning", "", "Body."),
		},
	}

	env := newAgentTestEnvWithCfg(t, readyCountsCfgYAML, seeds)
	env.login("admin@test.local", "admin-pass-123")

	counts := getReadyCounts(t, env)

	if got := countFor(counts, "backend-developer"); got != 0 {
		t.Errorf("backend-developer: want count 0 (no plan-backend with in-development status), got %d", got)
	}
}

// TestReadyCounts_RequirementsAnalystCountsOnlyIdeas verifies that
// requirements-analyst with source_types=[idea] and active_status=clarifying
// counts only clarifying ideas, not clarifying tickets or requirements.
func TestReadyCounts_RequirementsAnalystCountsOnlyIdeas(t *testing.T) {
	seeds := []seedArtifact{
		// A clarifying idea — should be counted.
		{
			relPath: "lifecycle/ideas/ra-idea-1.md",
			content: makeArtifact("RA Idea 1", "idea", "clarifying", "ra-idea-1", "", "Body."),
		},
		// A clarifying ticket — must NOT be counted (wrong type for requirements-analyst).
		{
			relPath: "lifecycle/requirements/ra-ticket-2.md",
			content: makeArtifact("RA Ticket 1", "ticket", "clarifying", "ra-ticket", "", "Body."),
		},
	}

	env := newAgentTestEnvWithCfg(t, readyCountsCfgYAML, seeds)
	env.login("admin@test.local", "admin-pass-123")

	counts := getReadyCounts(t, env)

	// Only 1 clarifying idea — the clarifying ticket must not be counted.
	if got := countFor(counts, "requirements-analyst"); got != 1 {
		t.Errorf("requirements-analyst: want count 1 (only clarifying ideas), got %d", got)
	}
}

// TestReadyCounts_ResponseShape verifies the structural contract of the
// ready-counts endpoint when source_types agents are configured:
//   - HTTP 200
//   - Body is {"counts": {...}}
//   - Each present value is numeric
//   - Agents without active_status are absent
func TestReadyCounts_ResponseShape(t *testing.T) {
	env := newAgentTestEnvWithCfg(t, readyCountsCfgYAML, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/agents/ready-counts", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	countsRaw, ok := data["counts"]
	if !ok {
		t.Fatal("response missing 'counts' key")
	}
	countsMap, isMap := countsRaw.(map[string]any)
	if !isMap {
		t.Fatalf("'counts' must be a JSON object, got %T", countsRaw)
	}

	// All configured agents have active_status, so all three must appear.
	for _, name := range []string{"requirements-analyst", "backend-developer", "frontend-developer"} {
		v, present := countsMap[name]
		if !present {
			t.Errorf("agent %q: expected to appear in counts (has active_status configured)", name)
			continue
		}
		switch v.(type) {
		case float64, int:
			// ok — JSON numbers decode as float64
		default:
			t.Errorf("agent %q: count has unexpected type %T (want numeric)", name, v)
		}
	}
}
