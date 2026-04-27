---
title: 'Tests: Analyst Agent Active Status Transitions'
type: test
status: approved
lineage: analyst-missing-in-progress-status
parent: lifecycle/test-plans/analyst-missing-in-progress-status-4-test.md
---

# Tests: Analyst Agent Active Status Transitions

Integration tests verifying that analyst agents correctly set and retain an
in-progress status on their target artifacts, mirroring the behaviour of
developer and QA agents.

## Test Files

- `tests/integration/agent_helpers_test.go` — shared helpers: `newAgentTestEnv`,
  `setupFakeClaude`, `startAgentRun`, `waitForRunCompletion`
- `tests/integration/agent_status_test.go` — status-lifecycle test cases
- `tests/integration/agent_ws_test.go` — WebSocket event test case
- `tests/integration/config_validation_test.go` — config drift detection

## Scenarios Covered

### Milestone 1 — `TestAnalystRequirementsActivatesStatus`

Starts an `analyst-requirements` run against an idea artifact with
`status: draft`.  Asserts:

- The artifact's `status` field on disk is updated to `clarifying`
  synchronously (before the agent process begins).
- A git commit exists with message pattern
  `status(activate-req): draft → clarifying [run:<hex>]`.

### Milestone 2 — `TestAnalystPlannerActivatesStatus`

Starts an `analyst-planner` run against a requirement artifact with
`status: clarifying`.  Asserts:

- The artifact's `status` on disk is updated to `planning` synchronously.
- A git commit exists with the corresponding `draft → planning` message
  pattern.

### Milestone 3 — Status retention and done_on_success

Three independent test cases:

| Test | Agent | Exit | Expected artifact status |
|------|-------|------|--------------------------|
| `TestAnalystStatusPersistsAfterSuccess` | `analyst-requirements` | 0 (success) | `clarifying` (retained, not reverted) |
| `TestAnalystStatusSetsDoneAfterSuccess` | `stub-done-agent` (done_on_success=true) | 0 (success) | `done` |
| `TestAnalystStatusPersistsAfterFailure` | `analyst-requirements` | 1 (failure) | `clarifying` (retained, not reverted to `draft`) |

### Milestone 4 — `TestAnalystRunBroadcastsStatusChange`

Subscribes directly to the project hub before starting a run.  Asserts:

- An `agent.started` WebSocket event is received with the correct `run_id`
  and `lineage`.
- The SQLite index reflects `status: clarifying` after `StartRun` returns
  (confirming `setArtifactStatus → idx.IndexFile` executed).

### Milestone 5 — `TestAgentActiveStatusIsKnown`

Loads the real `lifecycle/config.yaml` and iterates over all configured
agents.  For each agent with a non-empty `active_status`, asserts the value
exists in `artifact.KnownStatuses`.  Fails if any agent is configured with an
unrecognised status string, preventing silent configuration drift.

## Test Infrastructure

All tests use a `//go:build integration` build tag.  A `setupFakeClaude`
helper writes a minimal shell script (`#!/bin/sh; exit N`) to a temp directory
and prepends it to PATH via `t.Setenv`, so no real `claude` CLI is required.
`newAgentTestEnv` creates an isolated temp project with agents configured in
`lifecycle/config.yaml`.

Run with:

```sh
go test ./tests/integration/ -tags integration -run 'TestAnalyst|TestAgentActiveStatus'
```
