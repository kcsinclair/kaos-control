---
title: Test Suite — Agent Run Summary Panel
type: test
status: in-qa
lineage: agent-run-summary-panel
parent: lifecycle/test-plans/agent-run-summary-panel-5-test.md
---

# Test Suite: Agent Run Summary Panel

Companion artifact for the test implementation of
[[agent-run-summary-panel-5-test]].

---

## Overview

This suite implements all six milestones from the test plan, covering the
backend result parser, the result API endpoint, WebSocket result payloads, and
the three Vue frontend components introduced by the agent-run-summary-panel
feature.

---

## Milestone 1 — Backend unit tests: result line parser

**File:** `internal/agent/result_test.go`

Pre-existing at implementation time. All seven test functions were found
complete and passing:

| Test | Scenario |
|---|---|
| `TestParseResultLine_ValidResult` | Multi-line log with a valid `type:result` JSON line; all fields asserted |
| `TestParseResultLine_NoResultLine` | Log without a result line → nil result, non-nil error |
| `TestParseResultLine_MalformedJSON` | Corrupt JSON containing `"type":"result"` string → graceful error |
| `TestParseResultLine_ResultNotLastLine` | Result line mid-log; still found scanning backwards |
| `TestParseResultLine_EmptyLog` | Empty input → nil result, non-nil error |
| `TestParseResultLine_ZeroUsage` | All usage fields zero → parsed correctly (N/A cache ratio case) |
| `TestParseResultLine_PermissionDenials` | Non-empty `permission_denials` array → raw JSON preserved |

Run: `go test ./internal/agent/ -run TestParseResultLine`

---

## Milestone 2 — Backend integration tests: result API endpoint

**File:** `tests/integration/agents_api_test.go` (appended)

Five new test functions added to the existing `agents_api_test.go`:

| Test | Scenario |
|---|---|
| `TestGetAgentRunResult_CompletedRun` | Completed run with a valid result line → 200 with non-null result |
| `TestGetAgentRunResult_RunningRun` | Run still in "running" state → 409 with error message |
| `TestGetAgentRunResult_NoResultLine` | Completed run with no result line (Ollama case) → 200 with `result: null` and `reason` |
| `TestGetAgentRunResult_UnknownRunId` | Non-existent run ID → 404 |
| `TestGetAgentRunResult_FieldAccuracy` | Known result line → every field in the response matches the raw JSON values |

Helper `seedCompletedRun` inserts a run record directly into the SQLite index
and writes the log file at `<dataDir>/testproject/runs/<runID>.log`, avoiding
any timing dependencies on real process execution.

Run: `go test ./tests/integration/ -run TestGetAgentRunResult -tags integration`

---

## Milestone 3 — Backend integration test: WebSocket result payload

**File:** `tests/integration/agent_ws_test.go` (appended)

Two new test functions and a new helper `setupFakeClaudeWithOutput`:

| Test | Scenario |
|---|---|
| `TestAgentWSFinished_IncludesResult` | Fake claude emits a result line to stdout; `agent.finished` WS event carries non-null `result` with correct fields |
| `TestAgentWSFinished_NoResultLine_ResultNull` | Fake claude exits with no output; `agent.finished` event carries `"result": null` |

`setupFakeClaudeWithOutput` writes the desired stdout content to a temp file
and creates a `claude` shell script that cats it — avoiding shell-quoting
issues with JSON strings inside the script.

Run: `go test ./tests/integration/ -run TestAgentWSFinished -tags integration`

---

## Milestone 4 — Frontend unit tests: RunSummaryCard component

**File:** `tests/web/RunSummaryCard.test.ts` (new)

Twelve test scenarios across five describe blocks:

| Describe | Scenarios |
|---|---|
| Field rendering | Cost ($0.0234), duration (1m 15s / 45s), turns, token labels |
| Cache hit ratio | 80.0% calculation; N/A when denominator is zero |
| Cache quality thresholds | Excellent (≥90%, green), Good (≥75%, blue), Fair (≥50%, amber), Poor (<50%, red) |
| Fallback states | "Summary unavailable" for null result + Claude driver; driver-unavailable message for non-Claude driver |
| Permission denials | Section visible with non-empty list; hidden when empty |
| Token count formatting | Thousands separators via `toLocaleString()` |

No network calls — the component is mounted with prop fixtures only.

Run: `pnpm --prefix tests/web test RunSummaryCard`

---

## Milestone 5 — Frontend unit tests: RawLogModal component

**File:** `tests/web/RawLogModal.test.ts` (new)

Eight test scenarios across five describe blocks:

| Describe | Scenarios |
|---|---|
| Log content | `<pre>` element present with log text; `rlm-content` class (monospace) |
| Panel layout | `.rlm-panel` class presence (CSS sets `min-height: 90vh`) |
| Loading state | Loading indicator visible while API call is pending |
| Error state | Error message shown when `getRunLog` rejects |
| Empty log state | "No log content available" when API returns empty string |
| Dismiss (close button) | Emits `close` on button click |
| Dismiss (Escape key) | Emits `close` on Escape; non-Escape keys do not emit |

Run: `pnpm --prefix tests/web test RawLogModal`

---

## Milestone 6 — Frontend integration tests: RunDetailModal with summary

**File:** `tests/web/RunDetailModal.test.ts` (extended)

The existing `vi.mock('@/api/agents')` factory was updated to include a default
`getRunResult` mock returning `{ result: null }`, ensuring existing tests
continue to pass as the component now calls `getRunResult` for terminal runs.

Six new test scenarios in the `RunDetailModal — Milestone 6` describe block:

| Test | Scenario |
|---|---|
| shows summary for completed run | `getRunResult` resolves with a valid result → `RunSummaryCard` renders with `rsc-card` and cost displayed |
| no summary for running run | `getRun` returns status `running` → `RunSummaryCard` absent; `getRunResult` never called |
| summary unavailable for null result | `getRunResult` resolves `{ result: null }` → `rsc-card` absent; "unavailable" text visible |
| cached result from store | `store.runResults` pre-populated → `getRunResult` API not called; card still shown |
| View Full Log opens RawLogModal | Click `.rdm-btn-log` → `.rlm-overlay` appears in DOM |
| summary via WebSocket result | Store result set while run is open → card appears via reactive watcher |

Run: `pnpm --prefix tests/web test RunDetailModal`

---

## Running all tests

```sh
# Backend unit tests (no build tag needed)
go test ./internal/agent/ -run TestParseResultLine

# Backend integration tests
go test ./tests/integration/ -run 'TestGetAgentRunResult|TestAgentWSFinished' -tags integration

# Frontend tests
pnpm --prefix tests/web test --reporter=verbose
```
