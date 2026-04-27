---
title: "Test Plan: Agent Launcher Panels"
type: plan-test
status: in-development
lineage: agent-launcher-panels
parent: lifecycle/requirements/agent-launcher-panels-2.md
---

## Overview

Integration tests for the agent launcher panels feature, covering the backend API changes from [[agent-launcher-panels]] backend plan and the end-to-end launch flow exercised by the [[agent-launcher-panels]] frontend plan. Tests target the HTTP API layer; frontend component tests are out of scope (owned by the frontend developer).

## Milestone 1 — Test `GET /agents` returns `model` and `active_status`

### Description

Verify that the agent list endpoint exposes the two new fields added in the backend plan (Milestone 1). Tests should cover agents that have these fields set and agents that omit them.

### Files to change

- `tests/agents_api_test.go` — Add or extend test functions:
  - `TestListAgents_ModelAndActiveStatus`: register a project whose `config.yaml` includes agents with and without `model`/`active_status`. Call `GET /api/p/:project/agents`. Assert:
    - Each agent object in the response has `name`, `roles`, `driver`.
    - Agents with a model return `"model": "<value>"`.
    - Agents with `active_status` return `"active_status": "<value>"`.
    - Agents without model/active_status omit those keys (or return empty string — match backend `omitempty` behaviour).
  - `TestListAgents_InlineDriver`: verify that the `idea-capture` agent (driver `inline`) appears in the list with `driver: "inline"` so the frontend can identify non-launchable agents.

### Acceptance criteria

- [ ] Test asserts `model` field is present and correct for agents that define it.
- [ ] Test asserts `active_status` field is present and correct for agents that define it.
- [ ] Test asserts fields are omitted for agents that don't define them.
- [ ] Test asserts inline-driver agents are included in the response with `driver: "inline"`.
- [ ] `go test ./tests/... -run TestListAgents` passes.

## Milestone 2 — Test artifact filtering by status and type

### Description

Verify that `GET /artifacts?status=<s>` and `GET /artifacts?status=<s>&type=<t>` return only matching artifacts. This validates the eligibility query the frontend will use in the launch modal (FR-6).

### Files to change

- `tests/artifacts_api_test.go` — Add or extend test functions:
  - `TestListArtifacts_FilterByStatus`: seed the lifecycle directory with artifacts in different statuses (e.g. `draft`, `clarifying`, `planning`). Call `GET /api/p/:project/artifacts?status=draft`. Assert only draft artifacts are returned.
  - `TestListArtifacts_FilterByStatusAndType`: seed with artifacts of different types and statuses. Call `GET /api/p/:project/artifacts?status=draft&type=idea`. Assert only draft ideas are returned.
  - `TestListArtifacts_FilterNoMatch`: call with a status that has no matching artifacts. Assert empty `items` array and `total: 0`.

### Acceptance criteria

- [ ] Status-only filter returns exactly the artifacts with that status.
- [ ] Combined status + type filter returns only artifacts matching both.
- [ ] Empty result set returns `{ "items": [], "total": 0 }`.
- [ ] `go test ./tests/... -run TestListArtifacts_Filter` passes.

## Milestone 3 — Test agent run start via `POST /agents/:name/run`

### Description

Verify the run-start endpoint works correctly when called with a target artifact path, as the launch modal will call it. Cover success, not-found agent, and missing target scenarios.

### Files to change

- `tests/agents_api_test.go` — Add or extend test functions:
  - `TestStartAgentRun_Success`: create a valid artifact, call `POST /api/p/:project/agents/analyst-requirements/run` with `{"target_path": "<artifact_path>"}`. Assert HTTP 202 and a non-empty `run_id`.
  - `TestStartAgentRun_AgentNotFound`: call with a non-existent agent name. Assert HTTP 404 with `"not_found"` error code.
  - `TestStartAgentRun_BadRequest`: call with invalid JSON body. Assert HTTP 400.

### Acceptance criteria

- [ ] Valid run start returns 202 with `run_id`.
- [ ] Non-existent agent returns 404.
- [ ] Invalid request body returns 400.
- [ ] `go test ./tests/... -run TestStartAgentRun` passes.

## Milestone 4 — Test workflow predecessor mapping for eligibility

### Description

Verify that the workflow state machine's transition rules support the predecessor mapping the frontend relies on. This ensures the eligibility logic in the frontend plan is grounded in the actual workflow engine.

### Files to change

- `tests/workflow_test.go` or `internal/workflow/workflow_test.go` — Add test:
  - `TestWorkflowPredecessors`: for each agent `active_status` value used in the project config (`clarifying`, `planning`, `in-development`, `in-qa`), verify that the expected predecessor status has a valid transition rule to it. Specifically:
    - `draft` → `clarifying` is a valid transition.
    - `clarifying` → `planning` is a valid transition.
    - `planning` → `in-development` is a valid transition.
    - `in-development` → `in-qa` is a valid transition.
  - Use `Engine.CanTransition()` with appropriate roles to assert each.

### Acceptance criteria

- [ ] Each predecessor → active_status pair is confirmed as a valid workflow transition.
- [ ] Test uses the actual workflow engine (not hardcoded assumptions).
- [ ] `go test ./... -run TestWorkflowPredecessors` passes.
