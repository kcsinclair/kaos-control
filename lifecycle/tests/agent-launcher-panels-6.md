---
title: 'Tests: Agent Launcher Panels API'
type: test
status: approved
lineage: agent-launcher-panels
parent: lifecycle/test-plans/agent-launcher-panels-5-test.md
---

# Tests: Agent Launcher Panels API

Integration tests covering the backend API changes and end-to-end run-start
flow introduced by the agent-launcher-panels feature.

## Test Files

- `tests/integration/agents_api_test.go` — agent list and run-start endpoint tests (Milestones 1 + 3)
- `tests/integration/artifacts_api_test.go` — artifact filtering tests (Milestone 2)
- `internal/workflow/workflow_test.go` — workflow predecessor mapping unit test (Milestone 4)
- `tests/integration/agent_helpers_test.go` — extended with `newAgentTestEnvWithCfg` helper

## Scenarios Covered

### Milestone 1 — `GET /agents` returns `model` and `active_status`

#### `TestListAgents_ModelAndActiveStatus`

Uses a custom lifecycle config (`agentPanelCfgYAML`) registering four agents
with varying field combinations. Calls `GET /api/p/testproject/agents`. Asserts:

- Every agent object carries `name`, `roles`, `driver`.
- `agent-with-model`: `model="claude-opus-4-6"` and `active_status="clarifying"` are present.
- `agent-no-model`: `model` is absent/empty (`omitempty`); `active_status="planning"` is present.
- `agent-no-active-status`: `model="claude-sonnet-4-6"` is present; `active_status` is absent/empty.

#### `TestListAgents_InlineDriver`

Same custom config. Asserts that the `idea-capture` agent (driver=inline) appears
in the list with `driver: "inline"`.

### Milestone 2 — Artifact filtering by status and type

#### `TestListArtifacts_FilterByStatus`

Seeds two draft ideas and two non-draft requirements. Calls
`GET /api/p/testproject/artifacts?status=draft`. Asserts:

- Only the two draft artifacts are returned.
- All returned items have `status="draft"`.

#### `TestListArtifacts_FilterByStatusAndType`

Seeds two draft ideas, one draft ticket, and one clarifying idea. Calls
`GET /api/p/testproject/artifacts?status=draft&type=idea`. Asserts:

- Exactly two items returned (the draft ideas only).
- All items have `status="draft"` and `type="idea"`.

#### `TestListArtifacts_FilterNoMatch`

Seeds one draft idea and queries `?status=approved`. Asserts:

- `items` is an empty array.
- `total` is 0.

### Milestone 3 — Agent run start via `POST /agents/:name/run`

#### `TestStartAgentRun_Success`

Uses `setupFakeClaude(0)` (stub exits 0). Creates a valid draft idea artifact.
Calls `POST /api/p/testproject/agents/analyst-requirements/run` with
`{"target_path": "lifecycle/ideas/launch-target.md"}`. Asserts:

- HTTP 202 response.
- Non-empty `run_id` in the response body.

#### `TestStartAgentRun_AgentNotFound`

Calls the run endpoint with agent name `does-not-exist`. Asserts:

- HTTP 404 response.
- Error code `"not_found"` in the response body.

#### `TestStartAgentRun_BadRequest`

Sends raw malformed JSON (`{not valid json`) directly via `http.NewRequest`
(bypassing the test helper's marshalling). Asserts:

- HTTP 400 response.
- Error code `"bad_request"` in the response body.

### Milestone 4 — `TestWorkflowPredecessors`

Unit test in `internal/workflow/workflow_test.go`. Constructs a default
`Engine` (no project overrides). Asserts each predecessor → active_status
transition used by the feature is valid for the expected role:

| From | To | Role |
|------|----|------|
| `draft` | `clarifying` | `analyst` |
| `clarifying` | `planning` | `analyst` |
| `planning` | `in-development` | `approver` |
| `in-development` | `in-qa` | `backend-developer` |

## Test Infrastructure

All integration tests use `//go:build integration`. The `newAgentTestEnvWithCfg`
helper (added to `agent_helpers_test.go`) accepts an arbitrary config YAML string,
enabling tests to define custom agent configurations without modifying shared
constants.

Run integration tests:

```sh
go test ./tests/integration/ -tags integration -run 'TestListAgents|TestStartAgentRun|TestListArtifacts_Filter'
```

Run the workflow unit test:

```sh
go test ./internal/workflow/... -run TestWorkflowPredecessors
```
