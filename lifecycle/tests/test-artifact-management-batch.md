---
title: Serial Batch Test Execution — Integration Tests
type: test
status: draft
lineage: test-artifact-management
parent: lifecycle/test-plans/test-artifact-management-5-test.md
---

# Serial Batch Test Execution — Integration Tests

Integration tests covering **Milestone 3** of the test-artifact-management feature: verifying that multiple test artifacts can be executed serially and that the lineage lock is released promptly after each run.

## Test file

`tests/integration/test_artifact_batch_test.go`

## Setup

Tests use the same `qaAgentCfgYAML` and stub `claude` binary as the Milestone 2 tests. Hub WS events are collected via `env.proj.Hub.Register(ch)` to wait for terminal events without sleep-based polling. The helper `waitForWSTerminalEvent` blocks on the channel until `agent.finished` or `agent.failed` arrives for a specific `run_id`.

## Scenarios covered

| Test function | Scenario |
|---|---|
| `TestTestArtifactBatch_SerialExecution` | 3 approved tests submitted sequentially; each waits for `agent.finished` before starting the next. All 3 must complete with `agent.finished`. |
| `TestTestArtifactBatch_FailureDoesNotHaltBatch` | With a failing stub (exit 1), run 1 produces `agent.failed`. Run 2 is then started and must get 202 — failure in run 1 does not block run 2. |
| `TestTestArtifactBatch_LockReleaseAfterFinished` | Immediately after receiving `agent.finished` for run 1, starting run 2 (different lineage) must return 202 — no residual lock from run 1. |
| `TestTestArtifactBatch_RunRecordsHaveTerminalStatus` | After 3 serial runs complete, `GET /agents/runs/{run_id}` for each must return a non-`running` status (`done` or `failed`). |

## No timing-dependent flakiness

All tests block on explicit hub WS events (`waitForWSTerminalEvent`) rather than `time.Sleep`, eliminating race conditions between run completion and the next API call.
