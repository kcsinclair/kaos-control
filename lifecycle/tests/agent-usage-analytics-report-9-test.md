---
title: "Test Suite: Duplicate Helper Declaration Compile-Error Fix"
type: test
status: draft
lineage: agent-usage-analytics-report
parent: lifecycle/defects/agent-usage-report-test-compile-error-7-defect.md
---

This artifact documents the resolution of the duplicate-helper compile error described in the parent defect. No new test logic was added; the fix was entirely a rename of three helper functions so each has a unique name within the `integration` package.

## What was fixed

### `setupFakeClaudeWithOutput` → `setupFakeClaudeWithLines` (`agent_ws_test.go`)

`agent_ws_test.go` had its own `setupFakeClaudeWithOutput(t, []string, int)` that conflicted with the single-string variant of the same name in `agent_metrics_test.go`. The `agent_ws_test.go` variant was renamed to `setupFakeClaudeWithLines` (accepting a slice of output lines and an exit code) and all call sites inside that file were updated.

### `setupFakeClaudeWithScript` → `setupFakeClaudeWithRawScript` (`agent_metrics_test.go`)

`agent_metrics_test.go` declared `setupFakeClaudeWithScript(t, script string)` which collided with the identically-named helper in `queue_helpers_test.go`. The `agent_metrics_test.go` copy was renamed `setupFakeClaudeWithRawScript`; all usages inside `agent_metrics_test.go` (`TestSupervisor_PersistsMetricsOnFinish`, `TestSupervisor_RecordsTTFT`, `TestSupervisor_RecordsTTFTOnce`) were updated accordingly.

### `seedAgentRun` → `seedAgentRunRow` (`agents_api_test.go`)

`agents_api_test.go` declared `seedAgentRun(t, env, *index.AgentRunRow)` which conflicted with `reports_api_test.go`'s `seedAgentRun(t, env, id, agent, startedAt, status)`. The `agents_api_test.go` variant was renamed `seedAgentRunRow`; all call sites in that file were updated.

## Verification

Running `go test -tags=integration ./tests/integration/... -run="TestReportsAgentUsage|TestSupervisor|TestBackfill"` completes without compilation errors and all tests pass.

## Test files affected

- `tests/integration/agent_ws_test.go` — renamed helper and updated call sites
- `tests/integration/agent_metrics_test.go` — renamed helper and updated call sites
- `tests/integration/agents_api_test.go` — renamed helper and updated call sites
