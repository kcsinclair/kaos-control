---
title: "Agent Launcher Input Status Filtering — Test Plan"
type: plan-test
status: in-development
lineage: analyst-agent-sees-draft-ideas
parent: lifecycle/requirements/analyst-agent-sees-draft-ideas-2.md
---

# Agent Launcher Input Status Filtering — Test Plan

Tests verifying that the agent launch modal only offers `approved` artifacts as input, regardless of agent type. Tests cover the filtering logic extracted from `AgentLaunchModal.vue`, defect inclusion for developer agents, and the empty state. See [[analyst-agent-sees-draft-ideas]] frontend plan for implementation details.

## Milestone 1: Test Helpers for Agent Launch Modal

### Description

Create shared test fixtures that produce mock agent configurations and artifact rows spanning multiple statuses and types. These helpers will be used by all subsequent test suites to avoid duplicating setup logic.

### Files to Change

- `tests/web/helpers/agent_launch_fixtures.ts` — add factory functions: `makeAgentSummary(name, activeStatus, roles)` that returns an `AgentSummary`, and `makeArtifactsByStatusAndType(type, statuses[])` that returns `ArtifactRow[]` with the given type and one artifact per status.

### Acceptance Criteria

- [ ] `makeAgentSummary` produces a valid `AgentSummary` with configurable `name`, `active_status`, and `roles`.
- [ ] `makeArtifactsByStatusAndType` produces `ArtifactRow[]` with the specified type and one artifact per requested status.
- [ ] Helpers are importable by test files in `tests/web/agent-launch/`.

## Milestone 2: Input Status Filtering Tests

### Description

Test that the artifact filtering logic always uses `status: 'approved'`, not the `predecessorMap`-derived status. Since the fix removes `predecessorMap` and hardcodes `'approved'`, these tests validate the filtering function that `fetchArtifacts` relies on. Tests should use Vue `ref`/`computed` primitives to reproduce the filtering logic, consistent with the test pattern established in `tests/web/hide-done-items/`.

### Files to Change

- `tests/web/agent-launch/input-status-filter.test.ts` — unit tests for input status filtering logic.

### Acceptance Criteria

- [ ] **Test: analyst-requirements sees only approved ideas** — given ideas with statuses `draft`, `clarifying`, `approved`, `rejected`, only the `approved` idea appears.
- [ ] **Test: analyst-planner sees only approved requirements** — given requirements with mixed statuses, only `approved` requirements appear.
- [ ] **Test: backend-developer sees only approved plan-backend** — given plan-backend artifacts with mixed statuses, only `approved` ones appear.
- [ ] **Test: frontend-developer sees only approved plan-frontend** — given plan-frontend artifacts with mixed statuses, only `approved` ones appear.
- [ ] **Test: test-developer sees only approved plan-test** — given plan-test artifacts with mixed statuses, only `approved` ones appear.
- [ ] **Test: qa sees only approved tests** — given test artifacts with mixed statuses, only `approved` ones appear.
- [ ] **Test: predecessorMap no longer influences filtering** — the filter status is `'approved'` for all agents, regardless of `active_status` value.

## Milestone 3: Defect Inclusion Tests

### Description

Test that developer agents see approved defects assigned to their role in addition to their primary plan type, and that non-developer agents do not see defects.

### Files to Change

- `tests/web/agent-launch/defect-inclusion.test.ts` — unit tests for defect inclusion logic.

### Acceptance Criteria

- [ ] **Test: backend-developer sees approved defects assigned to backend-developer role** — defects with `status: approved` and `assignees` containing `role: backend-developer` appear alongside plan-backend artifacts.
- [ ] **Test: frontend-developer sees approved defects assigned to frontend-developer role** — same pattern for frontend.
- [ ] **Test: test-developer sees approved defects assigned to test-developer role** — same pattern for test.
- [ ] **Test: developer agent does not see unapproved defects** — defects with `status: draft` or other non-approved statuses are excluded even if assigned to the correct role.
- [ ] **Test: developer agent does not see defects assigned to other roles** — defects assigned to a different developer role are excluded.
- [ ] **Test: analyst-requirements does not see defects** — non-developer agents get no defect results.
- [ ] **Test: qa does not see defects** — qa agent gets no defect results.

## Milestone 4: Empty State Tests

### Description

Test that the modal displays the empty state message when no artifacts match the combined type + approved status filter.

### Files to Change

- `tests/web/agent-launch/empty-state.test.ts` — unit tests for empty state behaviour.

### Acceptance Criteria

- [ ] **Test: empty state when no approved artifacts of correct type exist** — given only `draft` ideas, the analyst-requirements agent sees an empty list.
- [ ] **Test: empty state when artifacts exist but wrong type** — given approved requirements but no approved ideas, analyst-requirements sees an empty list.
- [ ] **Test: empty state clears when approved artifact is added** — after adding an approved artifact of the correct type, the list is no longer empty.
- [ ] **Test: developer agent empty state includes defect check** — a developer agent with no approved plans and no approved defects for its role sees the empty state.
