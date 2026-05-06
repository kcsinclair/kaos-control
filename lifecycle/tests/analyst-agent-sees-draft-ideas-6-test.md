---
title: "Agent Launcher Input Status Filtering — Tests"
type: test
status: draft
lineage: analyst-agent-sees-draft-ideas
parent: lifecycle/test-plans/analyst-agent-sees-draft-ideas-5-test.md
---

# Agent Launcher Input Status Filtering — Tests

Integration tests for the agent launch modal's artifact filtering logic. Verifies that the modal always uses `status: 'approved'` when selecting input artifacts, that developer agents receive approved defects assigned to their role, and that the empty state is shown when no eligible artifacts exist.

## Test files

- `tests/web/helpers/agent_launch_fixtures.ts` — shared factory helpers
- `tests/web/agent-launch/input-status-filter.test.ts` — Milestone 2 tests (12 tests)
- `tests/web/agent-launch/defect-inclusion.test.ts` — Milestone 3 tests (14 tests)
- `tests/web/agent-launch/empty-state.test.ts` — Milestone 4 tests (18 tests)

**Total: 44 tests, all passing.**

## Scenarios covered

### Helpers (`agent_launch_fixtures.ts`)

- `makeAgentSummary(name, activeStatus, roles)` — builds a valid `AgentSummary`
- `makeArtifactsByStatusAndType(type, statuses[])` — builds `ArtifactRow[]` with one entry per status
- `makeDefect(assigneeRoles, status, overrides)` — builds a defect `ArtifactRow` with `frontmatter.assignees`
- `AGENT_INPUT_TYPE_MAP` — mirrors the `agentInputTypeMap` constant from `AgentLaunchModal.vue`
- `applyAgentLaunchFilter(agentName, roles, candidates, defects)` — pure function reproducing the modal's filtering logic for unit testing

### Milestone 2 — Input Status Filtering (`input-status-filter.test.ts`)

- `requirements-analyst` sees only `approved` ideas from a mixed-status set; excludes all other statuses
- `planning-analyst` sees only `approved` requirements; returns empty when none are approved
- `backend-developer` sees only `approved` plan-backend artifacts
- `frontend-developer` sees only `approved` plan-frontend artifacts
- `test-developer` sees only `approved` plan-test artifacts
- `qa` sees only `approved` test artifacts
- Filter status is always `approved` regardless of agent `active_status` (covers the removed `predecessorMap` path): tested for `active_status = 'draft'` and `active_status = 'in-development'`
- Parametric test: all agents in `AGENT_INPUT_TYPE_MAP` yield an empty list when no approved artifacts exist
- Parametric test: all agents yield exactly 1 result when exactly 1 approved artifact exists

### Milestone 3 — Defect Inclusion (`defect-inclusion.test.ts`)

- `backend-developer` sees an approved defect assigned to `backend-developer` alongside plan-backend
- `frontend-developer` sees an approved defect assigned to `frontend-developer` alongside plan-frontend
- `test-developer` sees an approved defect assigned to `test-developer` alongside plan-test
- Developer agent sees defect even when no approved plans exist (defect is the only result)
- `draft`, `rejected`, and `in-development` defects are excluded even when assigned to the correct role
- Defects assigned to a different developer role are excluded (all three developer role combinations)
- Defect assigned to multiple roles including the agent's own role is included
- `requirements-analyst` never sees defects even with approved defects for every role
- `qa` never sees defects

### Milestone 4 — Empty State (`empty-state.test.ts`)

- Empty list when all ideas are `draft`/`clarifying`/`rejected` (requirements-analyst)
- Empty list when all plan-backend are `draft`/`in-development` (backend-developer)
- Empty list when all tests are `draft`/`rejected` (qa)
- Empty list when approved artifacts exist but are the wrong type (three scenarios)
- Reactive: list transitions from empty → non-empty when an approved idea is added (requirements-analyst)
- Reactive: list transitions from empty → non-empty when an approved plan-backend is added (backend-developer)
- Developer agent empty state: empty when no approved plans AND no approved defects for its role (three agent roles)
- Developer agent empty state clears reactively when an approved defect for the correct role is added
- Parametric: all six agent types return empty list when candidate array is empty
