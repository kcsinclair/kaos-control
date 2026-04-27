---
title: "Test Plan: Analyst Agent Active Status Transitions"
type: plan-test
status: draft
lineage: analyst-missing-in-progress-status
parent: lifecycle/defects/analyst-missing-in-progress-status.md
labels:
    - agent
    - workflow
    - test
---

# Test Plan: Analyst Agent Active Status Transitions

This plan defines integration tests that verify analyst agents correctly set an in-progress status on their target artifacts during execution, matching the behaviour of developer agents (`in-development`) and the QA agent (`in-qa`).

## Milestone 1: Test `analyst-requirements` Sets `clarifying` on Run Start

### Description

Write an integration test that triggers an `analyst-requirements` agent run against an idea artifact in `draft` status and verifies the artifact's status is updated to `clarifying` before the agent process begins its work. This mirrors the existing pattern where `backend-developer` sets `in-development`.

### Files to Change

- `tests/agent_status_test.go` (new or extend existing) — add test case `TestAnalystRequirementsActivatesStatus`

### Acceptance Criteria

- [ ] Test creates a minimal idea artifact with `status: draft` in a temporary lifecycle directory
- [ ] Test triggers the `analyst-requirements` agent (or simulates the `StartRun` flow via the agent manager)
- [ ] After `StartRun` returns, the idea artifact's frontmatter on disk has `status: clarifying`
- [ ] A git commit exists with the message pattern `status(<lineage>): draft → clarifying [run:<id>]`
- [ ] Test passes with `go test ./tests/ -run TestAnalystRequirementsActivatesStatus`

## Milestone 2: Test `analyst-planner` Sets `planning` on Run Start

### Description

Write an integration test that triggers an `analyst-planner` agent run against a requirement artifact in `clarifying` status and verifies the artifact's status is updated to `planning` before the agent process begins.

### Files to Change

- `tests/agent_status_test.go` — add test case `TestAnalystPlannerActivatesStatus`

### Acceptance Criteria

- [ ] Test creates a minimal requirement artifact with `status: clarifying` in a temporary lifecycle directory
- [ ] Test triggers the `analyst-planner` agent (or simulates the `StartRun` flow)
- [ ] After `StartRun` returns, the requirement artifact's frontmatter on disk has `status: planning`
- [ ] A git commit exists with the message pattern `status(<lineage>): clarifying → planning [run:<id>]`
- [ ] Test passes with `go test ./tests/ -run TestAnalystPlannerActivatesStatus`

## Milestone 3: Test Status Reverts or Persists After Agent Completion

### Description

Verify behaviour when the analyst agent completes (both success and failure). If `done_on_success` is configured, the status should transition to `done`. If not configured, the status should remain at the active value (`clarifying` or `planning`) — it is not automatically reverted. This test documents the actual behaviour regardless of the `done_on_success` decision made in [[analyst-missing-in-progress-status-2-be]] Milestone 3.

### Files to Change

- `tests/agent_status_test.go` — add test cases:
  - `TestAnalystStatusPersistsAfterSuccess` (when `done_on_success` is false/unset)
  - `TestAnalystStatusSetsDoneAfterSuccess` (when `done_on_success` is true)
  - `TestAnalystStatusPersistsAfterFailure` (status stays at active value on failure)

### Acceptance Criteria

- [ ] When `done_on_success` is unset and the agent succeeds, the target artifact retains its `clarifying`/`planning` status
- [ ] When `done_on_success: true` and the agent succeeds, the target artifact status is `done`
- [ ] When the agent fails (non-zero exit), the target artifact retains its active status (`clarifying`/`planning`) — it is NOT reverted to the prior status
- [ ] Each scenario is a separate, independently runnable test case

## Milestone 4: Test WebSocket Event Includes Status Change

### Description

Verify that when an analyst agent starts and the target status changes, the WebSocket hub broadcasts an `artifact.indexed` event reflecting the new status. This ensures the frontend receives real-time updates.

### Files to Change

- `tests/agent_ws_test.go` (new or extend existing) — add test case `TestAnalystRunBroadcastsStatusChange`

### Acceptance Criteria

- [ ] Test establishes a WebSocket connection to the hub before triggering an analyst run
- [ ] After `StartRun`, the test receives an `agent.started` event with the correct `run_id` and `lineage`
- [ ] The re-indexed artifact (from `setArtifactStatus → idx.IndexFile`) triggers a file change event observable by the watcher or directly via index query
- [ ] Test passes with `go test ./tests/ -run TestAnalystRunBroadcastsStatusChange`

## Milestone 5: Test Config Validation

### Description

Verify that the `active_status` values in `lifecycle/config.yaml` for analyst agents are valid statuses recognised by the artifact parser. This prevents configuration drift.

### Files to Change

- `tests/config_validation_test.go` (new or extend existing) — add test case `TestAgentActiveStatusIsKnown`

### Acceptance Criteria

- [ ] Test loads `lifecycle/config.yaml` and iterates over all agent configs
- [ ] For each agent with a non-empty `active_status`, the test asserts the value exists in `artifact.KnownStatuses`
- [ ] Test fails if any agent is configured with an unrecognised active status
- [ ] Test passes with `go test ./tests/ -run TestAgentActiveStatusIsKnown`

## Cross-References

- [[analyst-missing-in-progress-status-2-be]] — backend config changes that enable the statuses being tested
- [[analyst-missing-in-progress-status-3-fe]] — frontend changes verified indirectly via WebSocket event tests in Milestone 4
