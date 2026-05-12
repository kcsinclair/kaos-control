---
title: "Test Plan: Agent Run Summary Panel"
type: plan-test
status: draft
lineage: agent-run-summary-panel
parent: lifecycle/requirements/agent-run-summary-panel-2.md
---

# Test Plan: Agent Run Summary Panel

relates-to: [[agent-run-summary-panel]]

## Overview

Integration and unit tests covering the backend result parser and API, frontend summary card and log modal components, and end-to-end WebSocket behaviour delivered by [[agent-run-summary-panel-3-be]] and [[agent-run-summary-panel-4-fe]].

---

## Milestone 1 — Backend unit tests: result line parser

### Description

Unit tests for `ParseResultLine` in `internal/agent/result.go`, covering parsing correctness and edge cases.

### Files to change

- `internal/agent/result_test.go` — tests defined in the backend plan Milestone 4. This milestone tracks their execution and verification:

  1. **`TestParseResultLine_ValidResult`** — multi-line log with valid `type:result` JSON. Assert all fields (cost, duration, turns, usage, permission_denials) are parsed correctly.
  2. **`TestParseResultLine_NoResultLine`** — log without `type:result`. Assert `nil` result, non-nil error.
  3. **`TestParseResultLine_MalformedJSON`** — corrupt JSON with `type:result` string. Assert graceful failure.
  4. **`TestParseResultLine_ResultNotLastLine`** — result appears mid-log. Assert it is still found.
  5. **`TestParseResultLine_EmptyLog`** — empty input. Assert `nil` result.
  6. **`TestParseResultLine_ZeroUsage`** — all usage fields zero. Assert parsed correctly with zero values (important for cache ratio "N/A" case).
  7. **`TestParseResultLine_PermissionDenials`** — non-empty permission_denials array. Assert entries are preserved as raw JSON.

### Acceptance criteria

- All tests pass with `go test ./internal/agent/ -run TestParseResultLine`.
- Tests cover the edge cases enumerated in FR-1 and NFR-2.
- No external dependencies — pure in-memory tests.

---

## Milestone 2 — Backend integration tests: result API endpoint

### Description

Test the `GET /api/p/{project}/agents/runs/{run_id}/result` endpoint for correct responses across different run states and driver types.

### Files to change

- `tests/integration/agents_api_test.go` — add new test functions:

  1. **`TestGetAgentRunResult_CompletedRun`**
     - Seed a completed run with a log file containing a valid `type:result` line.
     - `GET .../agents/runs/{run_id}/result` → assert 200 with `{"result": {...}}` containing correct `total_cost_usd`, `usage`, `duration_ms`, `num_turns`.

  2. **`TestGetAgentRunResult_RunningRun`**
     - Seed a run with status `running`.
     - `GET .../agents/runs/{run_id}/result` → assert 409 with `{"error": "run is still in progress"}`.

  3. **`TestGetAgentRunResult_NoResultLine`**
     - Seed a completed run with a log file that has no `type:result` line (simulates Ollama driver).
     - `GET .../agents/runs/{run_id}/result` → assert 200 with `{"result": null, "reason": "..."}`.

  4. **`TestGetAgentRunResult_UnknownRunId`**
     - `GET .../agents/runs/nonexistent/result` → assert 404.

  5. **`TestGetAgentRunResult_FieldAccuracy`**
     - Seed a run with a known result line. Assert every parsed field in the response matches the raw JSON values (spot-check requirement from acceptance criteria).

### Acceptance criteria

- All tests pass with `go test ./tests/integration/ -run TestGetAgentRunResult -tags integration`.
- Tests use the existing integration test harness.
- No flaky timing dependencies.

---

## Milestone 3 — Backend integration test: WebSocket result payload

### Description

Verify that `agent.finished` and `agent.failed` WebSocket events include the `result` field when a result line is present.

### Files to change

- `tests/integration/agent_ws_test.go` — add or extend:

  1. **`TestAgentWSFinished_IncludesResult`**
     - Start a Claude Code agent run (or simulate one with a fake binary that writes a `type:result` line).
     - Listen on the project WebSocket.
     - Wait for `agent.finished` event.
     - Assert the payload contains `"result"` with `total_cost_usd`, `usage`, and other fields.

  2. **`TestAgentWSFinished_NoResultLine_ResultNull`**
     - Start a run whose process produces no `type:result` line.
     - Wait for the terminal WS event.
     - Assert the payload contains `"result": null` — no error in the event.

### Acceptance criteria

- Tests confirm `result` is present (or null) in terminal WebSocket events.
- Tests use existing WS test helpers.
- No reliance on real Claude Code binary — use fake/mock processes.

---

## Milestone 4 — Frontend unit tests: RunSummaryCard component

### Description

Unit tests for the `RunSummaryCard.vue` component covering all rendering states, calculations, and threshold labels.

### Files to change

- `tests/web/RunSummaryCard.test.ts` — new test file:

  1. **`renders all summary fields for a valid result`**
     - Provide a complete `RunResult` prop. Assert the card displays: cost (4 decimal places with $), duration (Xm Ys format), turns, and four token usage rows.

  2. **`calculates cache hit ratio correctly`**
     - Provide result with `cache_read: 800, cache_creation: 100, input: 100`. Assert "80.0%" is displayed.

  3. **`displays Excellent label for ratio >= 90%`**
     - Provide result where ratio = 92%. Assert "Excellent" label with green styling.

  4. **`displays Good label for ratio >= 75%`**
     - Provide result where ratio = 80%. Assert "Good" label with blue styling.

  5. **`displays Fair label for ratio >= 50%`**
     - Provide result where ratio = 55%. Assert "Fair" label with amber styling.

  6. **`displays Poor label for ratio < 50%`**
     - Provide result where ratio = 30%. Assert "Poor" label with red styling.

  7. **`displays N/A when denominator is zero`**
     - Provide result with all usage fields at zero. Assert "N/A" is displayed instead of a percentage.

  8. **`displays fallback for null result with Claude driver`**
     - Pass `result: null, driverAvailable: true`. Assert "Summary unavailable" text.

  9. **`displays driver unavailable message for non-Claude runs`**
     - Pass `result: null, driverAvailable: false`. Assert "Token metrics not available for this driver" text.

  10. **`renders permission denials when present`**
      - Provide result with non-empty `permission_denials`. Assert the denials section is visible and lists entries.

  11. **`hides permission denials section when empty`**
      - Provide result with empty `permission_denials`. Assert the section is not rendered.

  12. **`formats token counts with thousands separators`**
      - Provide result with `input_tokens: 12345`. Assert "12,345" is displayed.

### Acceptance criteria

- All tests pass with `pnpm --prefix tests/web test` (or project Vitest config).
- Tests use Vitest + Vue Test Utils.
- No network calls — component receives data via props.
- All four cache quality threshold bands are tested.

---

## Milestone 5 — Frontend unit tests: RawLogModal component

### Description

Unit tests for the `RawLogModal.vue` component covering log display, dismiss behaviour, and edge cases.

### Files to change

- `tests/web/RawLogModal.test.ts` — new test file:

  1. **`displays log content in monospaced pre-formatted block`**
     - Mock `agentsApi.getRunLog` to return multi-line log text. Assert content is inside a `<pre>` element with monospace font.

  2. **`modal has minimum 90vh height`**
     - Mount the modal. Assert the log container has `min-height: 90vh` styling.

  3. **`scroll position starts at top`**
     - Mount with log content. Assert `scrollTop` is 0.

  4. **`dismisses on close button click`**
     - Click the close button. Assert `close` event is emitted.

  5. **`dismisses on Escape key`**
     - Trigger `keydown.escape`. Assert `close` event is emitted.

  6. **`displays loading state while fetching`**
     - Mount with delayed API mock. Assert a loading indicator is visible.

  7. **`displays error state on fetch failure`**
     - Mock `agentsApi.getRunLog` to reject. Assert an error message is displayed.

  8. **`displays empty state for empty log`**
     - Mock `agentsApi.getRunLog` to return empty string. Assert "No log content available" message.

### Acceptance criteria

- All tests pass.
- Tests cover FR-4 requirements (full-height, scrollable, dismiss methods).
- Consistent with existing test patterns in `tests/web/`.

---

## Milestone 6 — Frontend integration tests: RunDetailModal with summary

### Description

Tests for the integration between `RunDetailModal` and `RunSummaryCard`, verifying the summary appears for terminal runs and is absent for running runs.

### Files to change

- `tests/web/RunDetailModal.test.ts` — add to existing or new test file:

  1. **`shows summary card for completed run`**
     - Mock `agentsApi.getRun` to return a run with status `done`. Mock `agentsApi.getRunResult` to return a valid result. Assert `RunSummaryCard` is rendered with correct data.

  2. **`does not show summary card for running run`**
     - Mock `agentsApi.getRun` to return a run with status `running`. Assert `RunSummaryCard` is not rendered.

  3. **`shows summary unavailable when API returns null result`**
     - Mock `agentsApi.getRunResult` to return `{ result: null, reason: "..." }`. Assert fallback message is displayed.

  4. **`uses cached result from store when available`**
     - Pre-populate `agentsStore.runResults` with a result for the run ID. Mount the modal. Assert `getRunResult` API is not called.

  5. **`View Full Log button opens RawLogModal`**
     - Mount with a completed run. Click "View Full Log". Assert `RawLogModal` component is rendered.

  6. **`summary appears when run finishes via WebSocket`**
     - Mount modal for a running run. Simulate the store receiving an `agent.finished` event with a `result` payload. Assert the summary card appears without remounting.

### Acceptance criteria

- All tests pass.
- Tests verify both API-driven and WebSocket-driven summary display paths.
- Tests confirm running runs never show a summary.
- Tests verify the "View Full Log" → `RawLogModal` interaction.
