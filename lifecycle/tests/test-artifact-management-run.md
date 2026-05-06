---
title: Single Test Artifact Run — Integration Tests
type: test
status: draft
lineage: test-artifact-management
parent: lifecycle/test-plans/test-artifact-management-5-test.md
---

# Single Test Artifact Run — Integration Tests

Integration tests covering **Milestone 2** of the test-artifact-management feature: invoking the QA agent against a single test artifact and verifying WebSocket event payloads.

## Test file

`tests/integration/test_artifact_run_test.go`

## Setup

Tests use a custom `qaAgentCfgYAML` lifecycle config that registers a `qa` agent with `driver: claude-code-cli` and `active_status: in-qa`. A stub `claude` binary is installed via `setupFakeClaude` or `setupSlowFakeClaude` to control exit timing deterministically. Tests authenticate as `qa@test.local` which holds the `qa` role required to trigger the agent.

## Scenarios covered

| Test function | Scenario |
|---|---|
| `TestTestArtifactRun_ApprovedTestStarted` | `POST /agents/qa/run` with an approved test artifact returns 202 with a `run_id`, and the hub broadcasts `agent.started` with the correct `target_path`. |
| `TestTestArtifactRun_TerminalEventHasTargetPath` | After the stub agent exits, the hub broadcasts `agent.finished` (or `agent.failed`) carrying a `target_path` matching the original request. |
| `TestTestArtifactRun_DraftTestNoOrphan` | Running the QA agent against a `draft` test artifact does not crash the server (no 500). Any run that starts must reach a terminal state — no orphaned `running` records. |
| `TestTestArtifactRun_ConcurrentRunPrevented` | A second `POST /agents/qa/run` for the same lineage while the first is still active returns 409 Conflict. Uses a slow (5 s sleep) stub to hold the lineage lock. |

## Fixtures

- `lifecycle/tests/run-approved-test.md` — type=test, status=approved
- `lifecycle/tests/run-terminal-test.md` — type=test, status=approved
- `lifecycle/tests/run-draft-test.md` — type=test, status=draft
- `lifecycle/tests/run-concurrent-test.md` — type=test, status=approved
