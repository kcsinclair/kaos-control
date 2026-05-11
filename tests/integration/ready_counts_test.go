// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestone 1 — Per-Agent Role-Specific Ready Counts: Backend Integration Tests
//
// Tests for GET /api/p/:project/agents/ready-counts that verify each agent
// returns a count scoped to its configured source_types (artifact type filter)
// AND the ready-input status, which is always "approved" — the status that
// gates an artifact into an agent run. (active_status is the *during-run*
// status the agent transitions the artifact INTO, not the status it picks
// from, so it's the wrong column to count for a "ready" badge.)
//
// Configuration used by these tests defines three agents with source_types:
//
//	requirements-analyst  active_status=clarifying     source_types=[idea]
//	backend-developer     active_status=in-development source_types=[plan-backend]
//	frontend-developer    active_status=in-development source_types=[plan-frontend]
//
// The active_status values differ between agents but are irrelevant to the
// count; what matters is that the agent's source_types matches the artifact
// type AND the artifact is in status "approved".

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
// type in status="approved" and asserts that each agent receives a count
// derived exclusively from its own source_types.
//
// Acceptance criteria covered:
//   - Seeds idea/approved, plan-backend/approved, plan-frontend/approved
//   - requirements-analyst count == number of approved ideas (1)
//   - backend-developer count == number of approved plan-backend artifacts (1)
//   - frontend-developer count == number of approved plan-frontend artifacts (2)
//   - backend-developer count != frontend-developer count
func TestReadyCounts_PerAgentSourceTypes(t *testing.T) {
	seeds := []seedArtifact{
		// One approved idea — requirements-analyst should count this.
		{
			relPath: "lifecycle/ideas/src-types-idea-1.md",
			content: makeArtifact("Source Types Idea 1", "idea", "approved", "src-types-idea-1", "", "Body."),
		},
		// One approved plan-backend — backend-developer should count this.
		{
			relPath: "lifecycle/backend-plans/src-types-be-1-3-be.md",
			content: makeArtifact("Source Types BE Plan 1", "plan-backend", "approved", "src-types-be-1", "", "Body."),
		},
		// Two approved plan-frontend — frontend-developer should count both.
		{
			relPath: "lifecycle/frontend-plans/src-types-fe-1-4-fe.md",
			content: makeArtifact("Source Types FE Plan 1", "plan-frontend", "approved", "src-types-fe-1", "", "Body."),
		},
		{
			relPath: "lifecycle/frontend-plans/src-types-fe-2-4-fe.md",
			content: makeArtifact("Source Types FE Plan 2", "plan-frontend", "approved", "src-types-fe-2", "", "Body."),
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

	// requirements-analyst: 1 approved idea.
	if got := countFor(counts, "requirements-analyst"); got != 1 {
		t.Errorf("requirements-analyst: want count 1, got %d", got)
	}

	// backend-developer: 1 approved plan-backend.
	if got := countFor(counts, "backend-developer"); got != 1 {
		t.Errorf("backend-developer: want count 1, got %d", got)
	}

	// frontend-developer: 2 approved plan-frontend.
	if got := countFor(counts, "frontend-developer"); got != 2 {
		t.Errorf("frontend-developer: want count 2, got %d", got)
	}

	// backend-developer and frontend-developer must have different counts
	// because their source_types do not overlap.
	backendCount := countFor(counts, "backend-developer")
	frontendCount := countFor(counts, "frontend-developer")
	if backendCount == frontendCount {
		t.Errorf("backend-developer (%d) and frontend-developer (%d) must have distinct counts; "+
			"different source_types must produce role-specific results", backendCount, frontendCount)
	}
}

// TestReadyCounts_SourceTypesExcludesWrongType verifies that an agent with
// source_types=[plan-backend] does NOT count plan-frontend artifacts even when
// they share the same approved status.
func TestReadyCounts_SourceTypesExcludesWrongType(t *testing.T) {
	seeds := []seedArtifact{
		// Two approved plan-frontend (wrong type for backend-developer).
		{
			relPath: "lifecycle/frontend-plans/excl-fe-1-4-fe.md",
			content: makeArtifact("Exclude FE Plan 1", "plan-frontend", "approved", "excl-fe-1", "", "Body."),
		},
		{
			relPath: "lifecycle/frontend-plans/excl-fe-2-4-fe.md",
			content: makeArtifact("Exclude FE Plan 2", "plan-frontend", "approved", "excl-fe-2", "", "Body."),
		},
	}

	env := newAgentTestEnvWithCfg(t, readyCountsCfgYAML, seeds)
	env.login("admin@test.local", "admin-pass-123")

	counts := getReadyCounts(t, env)

	// backend-developer must see 0: no plan-backend artifacts exist.
	if got := countFor(counts, "backend-developer"); got != 0 {
		t.Errorf("backend-developer: want count 0 (no plan-backend artifacts), got %d", got)
	}

	// frontend-developer must see 2: two approved plan-frontend exist.
	if got := countFor(counts, "frontend-developer"); got != 2 {
		t.Errorf("frontend-developer: want count 2, got %d", got)
	}
}

// TestReadyCounts_SourceTypesExcludesWrongStatus verifies that an agent with
// source_types=[plan-backend] does NOT count plan-backend artifacts that have
// a non-approved status (e.g. draft, planning, in-development).
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
		// One plan-backend already in-development (agent already running) — must NOT count.
		{
			relPath: "lifecycle/backend-plans/ws-in-dev-3-be.md",
			content: makeArtifact("WS In-Dev BE", "plan-backend", "in-development", "ws-in-dev", "", "Body."),
		},
	}

	env := newAgentTestEnvWithCfg(t, readyCountsCfgYAML, seeds)
	env.login("admin@test.local", "admin-pass-123")

	counts := getReadyCounts(t, env)

	if got := countFor(counts, "backend-developer"); got != 0 {
		t.Errorf("backend-developer: want count 0 (no plan-backend with approved status), got %d", got)
	}
}

// TestReadyCounts_RequirementsAnalystCountsOnlyIdeas verifies that
// requirements-analyst with source_types=[idea] counts only approved ideas,
// not approved tickets or other types.
func TestReadyCounts_RequirementsAnalystCountsOnlyIdeas(t *testing.T) {
	seeds := []seedArtifact{
		// An approved idea — should be counted.
		{
			relPath: "lifecycle/ideas/ra-idea-1.md",
			content: makeArtifact("RA Idea 1", "idea", "approved", "ra-idea-1", "", "Body."),
		},
		// An approved ticket — must NOT be counted (wrong type for requirements-analyst).
		{
			relPath: "lifecycle/requirements/ra-ticket-2.md",
			content: makeArtifact("RA Ticket 1", "ticket", "approved", "ra-ticket", "", "Body."),
		},
	}

	env := newAgentTestEnvWithCfg(t, readyCountsCfgYAML, seeds)
	env.login("admin@test.local", "admin-pass-123")

	counts := getReadyCounts(t, env)

	// Only 1 approved idea — the approved ticket must not be counted.
	if got := countFor(counts, "requirements-analyst"); got != 1 {
		t.Errorf("requirements-analyst: want count 1 (only approved ideas), got %d", got)
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

// TestReadyCounts_DeveloperIncludesAssignedDefects verifies that a developer
// agent's ready count includes approved defect artifacts whose frontmatter
// assignees include the agent's role — matching the plan-* branch in
// web/src/components/agent/AgentLaunchModal.vue so badge == dialog list size.
//
// Seeds:
//   - One approved plan-backend (the primary source_type for backend-developer).
//   - One approved defect assigned to role: backend-developer.
//   - One approved defect assigned to role: qa (must NOT count for backend-developer).
//   - One approved defect with no assignees (must NOT count).
//   - One in-development defect assigned to backend-developer (must NOT count;
//     wrong status).
//
// Expected: backend-developer count = 1 (plan-backend) + 1 (assigned defect) = 2.
// frontend-developer count = 0 (no plan-frontend, no defects assigned to it).
func TestReadyCounts_DeveloperIncludesAssignedDefects(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/backend-plans/dev-defects-be-1-3-be.md",
			content: makeArtifact("Dev-Defects BE 1", "plan-backend", "approved", "dev-defects-be-1", "", "Body."),
		},
		{
			relPath: "lifecycle/defects/dev-defects-be-assigned.md",
			content: "---\ntitle: \"BE-Assigned Defect\"\ntype: defect\nstatus: approved\nlineage: dev-defects-be-assigned\nassignees:\n  - role: backend-developer\n    who: agent\n---\nBody.\n",
		},
		{
			relPath: "lifecycle/defects/dev-defects-qa-assigned.md",
			content: "---\ntitle: \"QA-Assigned Defect\"\ntype: defect\nstatus: approved\nlineage: dev-defects-qa-assigned\nassignees:\n  - role: qa\n    who: agent\n---\nBody.\n",
		},
		{
			relPath: "lifecycle/defects/dev-defects-noassignee.md",
			content: "---\ntitle: \"Unassigned Defect\"\ntype: defect\nstatus: approved\nlineage: dev-defects-noassignee\n---\nBody.\n",
		},
		{
			relPath: "lifecycle/defects/dev-defects-be-wrongstatus.md",
			content: "---\ntitle: \"BE Defect Wrong Status\"\ntype: defect\nstatus: in-development\nlineage: dev-defects-be-wrongstatus\nassignees:\n  - role: backend-developer\n    who: agent\n---\nBody.\n",
		},
	}

	env := newAgentTestEnvWithCfg(t, readyCountsCfgYAML, seeds)
	env.login("admin@test.local", "admin-pass-123")

	counts := getReadyCounts(t, env)

	if got := countFor(counts, "backend-developer"); got != 2 {
		t.Errorf("backend-developer: want 2 (1 plan-backend + 1 assigned defect), got %d", got)
	}
	if got := countFor(counts, "frontend-developer"); got != 0 {
		t.Errorf("frontend-developer: want 0 (no plan-frontend, no defects assigned to frontend-developer), got %d", got)
	}
	if got := countFor(counts, "requirements-analyst"); got != 0 {
		t.Errorf("requirements-analyst: want 0 (analyst is not a developer agent; assigned defects must not count), got %d", got)
	}
}
