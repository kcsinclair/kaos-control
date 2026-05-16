---
title: "Test Plan: Artefacts Agent Run Count Column"
type: plan-test
status: in-development
lineage: artefacts-agent-run-count-column
parent: lifecycle/requirements/artefacts-agent-run-count-column-2.md
---

# Test Plan: Artefacts Agent Run Count Column

This plan covers testing for the [[artefacts-agent-run-count-column]] backend and frontend changes. Tests are split between Go unit tests (index + handler) and Playwright e2e tests (column rendering, sorting, pill, WebSocket refresh).

## Milestone 1 — Unit test: `AgentRunCountsByTargetPath`

### Description

Test the new index method that returns run counts grouped by `target_path`. Cover zero-run, single-run, and multi-run scenarios across different statuses.

### Files to change

- `internal/index/index_test.go` — add test function `TestAgentRunCountsByTargetPath`:
  1. Insert agent runs with varying `target_path` and `status` values (done, failed, killed, running, queued).
  2. Call `AgentRunCountsByTargetPath()`.
  3. Assert counts match expected values.
  4. Assert artefacts with no runs are absent from the map (caller treats missing as 0).

### Acceptance criteria

- [ ] Test passes with `go test ./internal/index/ -run TestAgentRunCountsByTargetPath`.
- [ ] Covers: 0 runs (missing key), 1 run, N runs, mixed statuses all counted.
- [ ] Confirms a single query is used (no N+1) — validate via count of SQL executions if the test harness supports it, or by code review.

---

## Milestone 2 — Unit test: `ActiveAgentStatusByTargetPath`

### Description

Test the new index method that returns the active agent status per target path.

### Files to change

- `internal/index/index_test.go` — add test function `TestActiveAgentStatusByTargetPath`:
  1. Insert runs with `status = 'running'` and `status = 'queued'` for different paths.
  2. Call `ActiveAgentStatusByTargetPath()`.
  3. Assert: path with a running job returns `"running"`, path with only queued returns `"queued"`, path with only completed runs returns empty/missing.

### Acceptance criteria

- [ ] Test passes with `go test ./internal/index/ -run TestActiveAgentStatusByTargetPath`.
- [ ] Covers: running trumps queued, queued-only, no active runs.

---

## Milestone 3 — Unit/integration test: `GET /api/p/:project/artifacts` response enrichment

### Description

Test that the artefact list API endpoint includes `agent_run_count` and `active_agent_status` fields with correct values.

### Files to change

- `internal/http/artifacts_test.go` (or create if absent) — add test function `TestListArtifacts_AgentRunCount`:
  1. Set up a test project with indexed artefacts and seeded agent runs.
  2. Call `GET /api/p/:project/artifacts`.
  3. Decode JSON response and assert:
     - Every item has `agent_run_count` as an integer.
     - An artefact with 0 runs has `agent_run_count: 0`.
     - An artefact with 3 runs has `agent_run_count: 3`.
     - An artefact with a running agent has `active_agent_status: "running"`.
     - An artefact with no active agent omits `active_agent_status`.

### Acceptance criteria

- [ ] Test passes with `go test ./internal/http/ -run TestListArtifacts_AgentRunCount`.
- [ ] Validates JSON field names match frontend expectations (`agent_run_count`, `active_agent_status`).
- [ ] Confirms `agent_run_count` is always present (never null/omitted), even when 0.

---

## Milestone 4 — E2E test: "Runs" column rendering and sorting

### Description

Playwright test that verifies the "Runs" column appears in the artefacts table, displays correct counts, and is sortable.

### Files to change

- `tests/e2e/` — add or extend an artefact-list test file (e.g. `tests/e2e/specs/artefact-run-count-column.spec.ts`):
  1. **Setup**: Seed the test project with artefacts and agent runs (some with 0 runs, some with N runs) via API or fixture.
  2. **Column presence**: Navigate to the artefacts list, assert a column header with text "Runs" exists.
  3. **Column position**: Assert "Runs" header appears after "Type" and before "Created" in DOM order.
  4. **Count display**: Assert cells in the "Runs" column show the expected integer values including `0`.
  5. **Sort ascending**: Click the "Runs" header, assert rows are ordered by count ascending.
  6. **Sort descending**: Click again, assert rows are ordered by count descending.

### Acceptance criteria

- [ ] Playwright test passes in CI.
- [ ] Validates column presence, position, count values, and both sort directions.
- [ ] Zero-count rows display `0` not blank.

---

## Milestone 5 — E2E test: active-agent status pill

### Description

Playwright test that verifies the "Agent Running" and "Work Queued" pills appear and disappear correctly.

### Files to change

- `tests/e2e/specs/artefact-run-count-column.spec.ts` (extend from Milestone 4):
  1. **Running pill**: Trigger an agent run against an artefact (or mock the API response to include `active_agent_status: "running"`). Assert a pill with text "Agent Running" appears in the artefact's row.
  2. **Queued pill**: Set up a queued run. Assert a pill with text "Work Queued" appears.
  3. **Pill removal**: After the agent run completes (simulate `agent.finished` WS event or wait for completion), assert the pill is no longer visible.
  4. **No pill**: For artefacts with no active runs, assert no pill is rendered.

### Acceptance criteria

- [ ] Playwright test passes in CI.
- [ ] Validates pill text, appearance, and disappearance.
- [ ] Pill styling is visually distinct for running vs. queued (screenshot comparison or class assertion).

---

## Milestone 6 — E2E test: WebSocket-driven count refresh

### Description

Playwright test that verifies the run count increments after an agent run finishes, without a full page reload.

### Files to change

- `tests/e2e/specs/artefact-run-count-column.spec.ts` (extend):
  1. Navigate to artefacts list, note the current run count for a target artefact.
  2. Trigger an agent run against that artefact via the API.
  3. Wait for the `agent.finished` event to propagate (poll the DOM for count change, with timeout).
  4. Assert the run count has incremented by 1.
  5. Assert no full page navigation occurred (check `page.url()` remains the same, or listen for no `load` event).

### Acceptance criteria

- [ ] Playwright test passes in CI.
- [ ] Count increments without page reload (AC6 from requirement).
- [ ] Test is resilient to timing — uses `expect(locator).toHaveText()` with Playwright auto-retry rather than hard sleeps.

---

## Milestone 7 — Static analysis gates

### Description

Verify all static analysis checks pass after the combined backend and frontend changes.

### Files to change

- None — this milestone runs existing tooling.

### Steps

1. `go vet ./...` — must pass.
2. `staticcheck ./...` — must pass.
3. `cd web && pnpm exec vue-tsc --noEmit` — must pass.
4. `cd web && pnpm build` — must produce a clean build.

### Acceptance criteria

- [ ] All four commands exit 0.
- [ ] No new warnings introduced.
