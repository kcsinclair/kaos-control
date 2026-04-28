---
title: "Test Plan: Artifact Agent Run History"
type: plan-test
status: done
lineage: artifact-agent-run-history
parent: lifecycle/requirements/artifact-agent-run-history-2.md
created: "2026-04-28"
---

# Test Plan: Artifact Agent Run History

relates-to: [[artifact-agent-run-history]]

## Overview

Integration and unit tests covering the backend API, frontend components, and end-to-end behaviour delivered by [[artifact-agent-run-history-3-be]] and [[artifact-agent-run-history-4-fe]].

---

## Milestone 1 — Backend integration tests: runs-by-target-path API

### Description

Test the `GET /api/p/{project}/agents/runs?target_path={path}` endpoint for correctness, edge cases, and performance.

### Files to change

- `tests/integration/agents_api_test.go` — add new test functions:

  1. **`TestListAgentRunsByTargetPath_ReturnsMatchingRuns`**
     - Seed the `agent_runs` table with 3 runs: 2 targeting `lifecycle/requirements/foo-2.md` and 1 targeting `lifecycle/ideas/bar.md`.
     - `GET …/agents/runs?target_path=lifecycle/requirements/foo-2.md` → assert response contains exactly 2 runs, ordered by `started_at DESC`.

  2. **`TestListAgentRunsByTargetPath_EmptyResult`**
     - `GET …/agents/runs?target_path=lifecycle/nonexistent.md` → assert response is `{"runs": []}` with HTTP 200, not an error.

  3. **`TestListAgentRunsByTargetPath_NoParam_ReturnsAll`**
     - Verify that omitting `target_path` still returns all runs (existing behaviour preserved).

  4. **`TestListAgentRunsByTargetPath_OrderNewestFirst`**
     - Seed 3 runs with distinct `started_at` values for the same target path.
     - Assert the returned array is sorted by `started_at DESC`.

### Acceptance criteria

- All four tests pass with `go test ./tests/integration/ -run TestListAgentRunsByTargetPath -tags integration`.
- Tests use the existing integration test harness (test server, test project setup).
- No flaky timing dependencies — use deterministic timestamps.

---

## Milestone 2 — Backend integration test: SQLite index existence

### Description

Verify that the `idx_agent_runs_target_path` index exists after the index is initialised.

### Files to change

- `tests/integration/agents_api_test.go` — add:

  1. **`TestAgentRunsTargetPathIndexExists`**
     - Query `SELECT name FROM sqlite_master WHERE type='index' AND name='idx_agent_runs_target_path'`.
     - Assert exactly one row is returned.

### Acceptance criteria

- Test passes, confirming the index is created automatically at startup.

---

## Milestone 3 — Backend integration test: WebSocket payloads include `target_path`

### Description

Verify that `agent.started`, `agent.finished`, and `agent.failed` WebSocket events include the `target_path` field.

### Files to change

- `tests/integration/agent_ws_test.go` — add or extend:

  1. **`TestAgentWSEvents_IncludeTargetPath`**
     - Start an agent run with a known target path.
     - Listen on the project WebSocket.
     - Assert the `agent.started` event payload contains `"target_path"` matching the run's target.
     - Wait for completion and assert the terminal event (`agent.finished` or `agent.failed`) also contains `"target_path"`.

### Acceptance criteria

- Test confirms `target_path` is present in both start and terminal WS events.
- Test uses the existing WS test helpers in `agent_ws_test.go` / `agent_helpers_test.go`.

---

## Milestone 4 — Frontend unit test: ArtifactRunHistory component

### Description

Unit tests for the `ArtifactRunHistory.vue` component covering rendering states and interaction.

### Files to change

- `tests/web/ArtifactRunHistory.test.ts` — new test file:

  1. **`renders loading state while fetching`**
     - Mount component, mock store action to delay. Assert a loading indicator is visible.

  2. **`renders empty state when no runs`**
     - Mock store to return empty `artifactRuns`. Assert "No agent runs for this artifact" text is displayed.

  3. **`renders run list with correct fields`**
     - Mock store with 2 sample runs. Assert each row shows truncated run ID (8 chars), agent name, formatted date, and status badge.

  4. **`status badges have accessible text`**
     - Mock store with runs of each status. Assert badges have aria-label or visible text, not colour alone.

  5. **`emits select-run on row click`**
     - Mock store with 1 run. Click the row. Assert `select-run` is emitted with the correct run ID.

  6. **`fetches runs with correct target path on mount`**
     - Mount with `targetPath="lifecycle/requirements/foo-2.md"`. Assert the store's `fetchRunsByTargetPath` was called with that path.

### Acceptance criteria

- All tests pass with `pnpm --prefix tests/web test` (or the project's Vitest config).
- Tests use Vitest + Vue Test Utils, consistent with existing test files in `tests/web/`.
- No network calls — store actions are mocked.

---

## Milestone 5 — Frontend unit test: RunDetailModal component

### Description

Unit tests for the `RunDetailModal.vue` component covering data display and dismiss behaviour.

### Files to change

- `tests/web/RunDetailModal.test.ts` — new test file:

  1. **`displays all AgentRunRow fields`**
     - Mock `agentsApi.getRun` to return a complete run object. Assert the modal renders: run ID, agent name, role, target path, started_at, finished_at, status, exit code, stderr tail (in `<pre>`), and artifacts produced.

  2. **`stderr tail renders in monospace scrollable block`**
     - Mock a run with multi-line stderr. Assert the stderr is inside a `<pre>` element with overflow styling.

  3. **`dismisses on close button click`**
     - Click the close button. Assert the component emits `close`.

  4. **`dismisses on Escape key`**
     - Trigger `keydown.escape`. Assert the component emits `close`.

  5. **`dismisses on backdrop click`**
     - Click the overlay backdrop (not the modal content). Assert the component emits `close`.

  6. **`traps focus while open`**
     - Mount the modal. Assert focus is within the modal container. Tab through elements and assert focus does not leave the modal.

### Acceptance criteria

- All tests pass.
- Accessibility tests (focus trap, Escape dismiss) are present.
- Consistent with existing test patterns in `tests/web/`.

---

## Milestone 6 — Frontend test: live update via WebSocket

### Description

Verify that the run list in `ArtifactRunHistory` updates reactively when the Pinia store's `artifactRuns` changes in response to a simulated WebSocket event.

### Files to change

- `tests/web/ArtifactRunHistory.test.ts` — add:

  1. **`updates list when store artifactRuns changes`**
     - Mount component with an initial set of runs. Programmatically push a new run into `agentsStore.artifactRuns`. Assert the new run appears in the rendered list without re-mounting.

  2. **`updates status when existing run changes`**
     - Mount with a "running" run. Update its status to "done" in the store. Assert the badge text changes to "done".

### Acceptance criteria

- Tests confirm reactive updates without component remount.
- No flaky async issues — use `nextTick` or `flushPromises` as needed.
