---
title: "Test: Verify Per-Agent Role-Specific Ready Counts"
type: plan-test
status: draft
lineage: agent-panel-ready-count-not-role-specific
parent: lifecycle/defects/agent-panel-ready-count-not-role-specific.md
---

# Test: Verify Per-Agent Role-Specific Ready Counts

## Problem Summary

The defect reports that all agent panels display the same ready count. Tests must verify that:
1. The backend endpoint returns distinct counts per agent based on status + type filtering.
2. The frontend renders per-agent counts (not a shared value).

---

## Milestone 1: Backend Integration Test — Ready Counts Endpoint

### Description

Write integration tests for `GET /api/p/:project/agents/ready-counts` that seed artifacts with different types and statuses, then verify each agent gets a distinct, correct count.

### Files to Change

- `tests/ready_counts_test.go` (new file) — integration test hitting the ready-counts endpoint.

### Acceptance Criteria

- [ ] Test seeds at least 3 artifacts: one `idea` with status `clarifying`, one `plan-backend` with status `in-development`, one `plan-frontend` with status `in-development`.
- [ ] Asserts `requirements-analyst` count equals the number of `clarifying` ideas.
- [ ] Asserts `backend-developer` count equals the number of `in-development` + `plan-backend` artifacts.
- [ ] Asserts `frontend-developer` count equals the number of `in-development` + `plan-frontend` artifacts.
- [ ] Asserts `backend-developer` count ≠ `frontend-developer` count when seed data differs.
- [ ] Test passes with `go test ./tests/... -run TestReadyCounts`.

---

## Milestone 2: Backend Unit Test — Count with Type Filter

### Description

Add a unit test in the index package confirming that `Count(Filter{Status: "in-development", Type: "plan-backend"})` returns only artifacts matching both predicates.

### Files to Change

- `internal/index/index_test.go` — add `TestCountWithTypeFilter` (or extend existing count tests).

### Acceptance Criteria

- [ ] Test inserts artifacts with varied status/type combos into an in-memory index.
- [ ] Verifies `Count(Filter{Status: "in-development", Type: "plan-backend"})` returns the correct subset count.
- [ ] Verifies `Count(Filter{Status: "in-development"})` (no type) returns all `in-development` artifacts regardless of type.
- [ ] Test passes with `go test ./internal/index/... -run TestCountWithTypeFilter`.

---

## Milestone 3: Frontend Component Test — Distinct Badge Values

### Description

Write a component test for `AgentPanelRow.vue` that mounts two agent panels with different ready counts and verifies they display different numbers.

### Files to Change

- `web/src/components/agent/__tests__/AgentPanelRow.spec.ts` (new or extend existing) — component test using Vitest + Vue Test Utils.

### Acceptance Criteria

- [ ] Mounts `AgentPanelRow` with a mocked agents store where `readyCounts['backend-developer'] = 3` and `readyCounts['frontend-developer'] = 7`.
- [ ] Asserts the backend-developer panel badge text contains "3 ready".
- [ ] Asserts the frontend-developer panel badge text contains "7 ready".
- [ ] Asserts badges are NOT the same value.
- [ ] Test passes with `pnpm test -- --run AgentPanelRow`.

---

## Milestone 4: E2E Smoke Test — Agents Screen Shows Distinct Counts

### Description

If an E2E framework is available, add a smoke test that navigates to the Agents screen and verifies not all badges share the same value. Otherwise, document a manual test procedure in the test artifact.

### Files to Change

- `tests/e2e/agents_ready_counts_test.go` or `tests/agents_smoke_test.go` — E2E or API-level smoke test.

### Acceptance Criteria

- [ ] Test starts the application, seeds distinct artifacts, navigates to agents (or hits the API).
- [ ] Verifies at least two agents show different ready counts.
- [ ] Test is tagged appropriately (not run in `-short` mode).

---

## Cross-References

- [[agent-panel-ready-count-not-role-specific]] backend plan — provides the `source_types` config field and updated endpoint logic being tested.
- [[agent-panel-ready-count-not-role-specific]] frontend plan — provides the component changes verified in Milestone 3.
