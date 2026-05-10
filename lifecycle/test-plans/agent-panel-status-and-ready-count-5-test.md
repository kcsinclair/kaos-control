---
title: 'Test Plan: Agent Panel Status and Ready Count'
type: plan-test
status: draft
lineage: agent-panel-status-and-ready-count
parent: lifecycle/requirements/agent-panel-status-and-ready-count-2.md
---

## Overview

Integration and unit tests for the [[agent-panel-status-and-ready-count]] feature covering the backend ready-counts endpoint, frontend store logic, and UI behaviour.

## Milestone 1: Backend Endpoint Tests

**Description:** Unit tests for the `handleGetReadyCounts` handler verifying correct counts, empty-status exclusion, and edge cases.

**Files to change:**
- `internal/http/agents_test.go` — add test functions for the ready-counts endpoint

**Test cases:**
1. **Happy path** — configure 3 agents with different `active_status` values, seed index with artifacts at various statuses, assert response counts match expected
2. **Empty active_status excluded** — agent with no `active_status` must not appear in response
3. **Zero count returned** — agent with `active_status` but no matching artifacts returns count of 0
4. **Multiple artifacts same status** — verify correct aggregation
5. **Status change reflected** — index an artifact, change its status via API, re-query counts, verify update

**Acceptance criteria:**
- [ ] All test cases pass with `go test ./internal/http/ -run TestReadyCounts`
- [ ] Tests do not depend on filesystem state (use test fixtures or in-memory index)
- [ ] Response shape validated: `{"counts": {...}}` with correct types

## Milestone 2: Frontend Store Tests

**Description:** Unit tests for the Pinia agents store extensions: `fetchReadyCounts`, `readyCounts` state, and `runningCountByAgent` computed.

**Files to change:**
- `web/src/stores/__tests__/agents.test.ts` (create or extend)

**Test cases:**
1. **fetchReadyCounts populates state** — mock API response, call action, assert `readyCounts` matches
2. **runningCountByAgent computation** — set `runs` with multiple running entries for same agent, assert grouped counts
3. **WebSocket agent.started increments running count** — call `onWsEvent('agent.started', ...)`, verify `runningCountByAgent` updates
4. **WebSocket agent.finished decrements running count** — call `onWsEvent('agent.finished', ...)`, verify count drops
5. **readyCounts cleared on project switch** — verify state resets when fetching for new project

**Acceptance criteria:**
- [ ] All tests pass with `pnpm test`
- [ ] No flaky timing-dependent assertions (use `await nextTick()` or `flushPromises()`)
- [ ] Mocks are minimal and focused on API boundary

## Milestone 3: Component Rendering Tests

**Description:** Component tests for `AgentPanelRow.vue` verifying badge rendering, accessibility, and running-state styling.

**Files to change:**
- `web/src/components/agent/__tests__/AgentPanelRow.test.ts` (create or extend)

**Test cases:**
1. **Badge rendered with count** — mount component with store containing `readyCounts`, assert badge shows correct number
2. **Badge hidden for no active_status** — agent without `active_status` has no badge element
3. **Zero count displayed** — badge shows literal `0` text, not hidden
4. **aria-label present** — badge element has `aria-label` matching pattern `"N artifacts ready"`
5. **Running highlight applied** — when `runningCountByAgent` > 0, assert `.agent-panel--running` class present
6. **Running highlight removed** — when count drops to 0, class removed
7. **Run count badge shown** — when running, small badge with run count is visible
8. **No layout shift** — snapshot test confirming badge doesn't change container dimensions (compare with/without badge)

**Acceptance criteria:**
- [ ] All tests pass with `pnpm test`
- [ ] Component mounts without errors in test environment
- [ ] Accessibility assertions validate `aria-label` content

## Milestone 4: Integration / E2E Smoke Test

**Description:** End-to-end test verifying the full flow: artifact created → index updated → badge reflects count → agent started → card shows running state.

**Files to change:**
- `tests/agent_panel_status_test.go` (new integration test file)

**Test cases:**
1. **Ready count reflects indexed artifacts** — create a markdown artifact with status matching an agent's `active_status`, wait for index, call ready-counts endpoint, verify count incremented
2. **Real-time update via WebSocket** — connect WebSocket, create artifact, receive `artifact.indexed` event, re-fetch counts, verify update
3. **Agent start/finish lifecycle** — start an agent run (inline driver for speed), verify `agent.started` event received, verify running state, wait for completion, verify `agent.finished` event

**Acceptance criteria:**
- [ ] Integration tests pass with `make test-unit` or dedicated integration target
- [ ] Tests are hermetic (use temp project directory, isolated SQLite DB)
- [ ] WebSocket assertions have reasonable timeout (5s max)
