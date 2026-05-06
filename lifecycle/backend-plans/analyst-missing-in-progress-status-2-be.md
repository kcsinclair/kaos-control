---
title: "Backend Plan: Add Active Status to Analyst Agents"
type: plan-backend
status: done
lineage: analyst-missing-in-progress-status
parent: lifecycle/defects/analyst-missing-in-progress-status.md
labels:
    - agent
    - workflow
    - backend
---

# Backend Plan: Add Active Status to Analyst Agents

The analyst agents (`requirements-analyst` and `planning-analyst`) do not set an in-progress status on the target artifact when a run starts. The backend already supports this via the `active_status` config field — developer agents use `in-development` and QA uses `in-qa` — but neither analyst agent is configured with one.

This plan adds a new `clarifying` active status for `requirements-analyst` and `planning` for `planning-analyst`, using existing statuses from the vocabulary rather than introducing a new one. These statuses are semantically correct: the requirements-analyst agent clarifies an idea into a requirement, and the planning-analyst agent plans a requirement into implementation plans.

## Milestone 1: Configure `active_status` on Analyst Agents

### Description

Add `active_status` and `done_on_success` fields to the two analyst agent definitions in the project config. No Go code changes are required — the `agent.Manager.StartRun` method already reads `AgentConfig.ActiveStatus` and calls `setArtifactStatus` when it is non-empty (see `internal/agent/agent.go:397-405`).

### Files to Change

- `lifecycle/config.yaml` — add `active_status: clarifying` to `requirements-analyst`; add `active_status: planning` to `planning-analyst`. Optionally add `done_on_success: true` to both if the target artifact should be marked `done` when the agent finishes successfully.

### Acceptance Criteria

- [ ] `requirements-analyst` config block includes `active_status: clarifying`
- [ ] `planning-analyst` config block includes `active_status: planning`
- [ ] Running `requirements-analyst` against an idea artifact causes its status to change to `clarifying` before the agent process starts
- [ ] Running `planning-analyst` against a requirement artifact causes its status to change to `planning` before the agent process starts
- [ ] A git commit is produced for each status change with the message format `status(<lineage>): <old> → <new> [run:<id>]`
- [ ] No changes are needed to Go source code — the existing `active_status` machinery in `internal/agent/agent.go` handles this generically

## Milestone 2: Verify Workflow Transition Compatibility

### Description

Confirm that `setArtifactStatus` in `internal/agent/agent.go` bypasses the workflow engine's `CanTransition` check (it writes directly to disk), so no workflow rule changes are needed. However, verify that the workflow transitions still allow human users to perform the same transitions for manual runs — specifically `draft → clarifying` and `clarifying → planning` — which the default rules already permit for the `analyst` role.

### Files to Change

- No files to change — this is a verification milestone.

### Acceptance Criteria

- [ ] `internal/agent/agent.go:setArtifactStatus` writes directly to disk without calling `workflow.Engine.CanTransition` (confirmed by code inspection)
- [ ] `internal/workflow/workflow.go` default rules already include `draft → clarifying` (roles: `product-owner`, `analyst`) and `clarifying → planning` (roles: `product-owner`, `reviewer`, `analyst`)
- [ ] No workflow rule additions are needed

## Milestone 3: Decide `done_on_success` Behaviour for Analyst Agents

### Description

The developer agents set `done_on_success: true`, meaning the target plan artifact is marked `done` when the agent completes successfully. For analyst agents, the semantics are different: the requirements-analyst agent produces a *new* requirement artifact (it does not "complete" the idea), and the planning-analyst agent produces *new* plan artifacts (it does not "complete" the requirement).

Determine whether `done_on_success` should be set. If the product owner wants the source artifact (idea or requirement) to remain in its pre-run status after a successful agent run, leave `done_on_success` unset. If the source should be advanced (e.g. idea → `done` after requirement is produced), set it.

### Files to Change

- `lifecycle/config.yaml` — add or omit `done_on_success` on `requirements-analyst` and `planning-analyst` based on the decision.

### Acceptance Criteria

- [ ] A deliberate decision is recorded (in commit message or artifact) about whether analyst agents should set the target to `done` on success
- [ ] If `done_on_success: true` is added, the idea/requirement status transitions to `done` after a successful analyst run
- [ ] If `done_on_success` is omitted, the idea/requirement retains its `active_status` value (`clarifying` or `planning`) after the run — the human product-owner or a subsequent workflow step advances it

## Cross-References

- [[analyst-missing-in-progress-status-3-fe]] — frontend must ensure the `clarifying` and `planning` statuses render correctly in the UI, including badge colours and graph node colours
- [[analyst-missing-in-progress-status-4-test]] — integration tests must verify the status transitions during analyst agent runs
