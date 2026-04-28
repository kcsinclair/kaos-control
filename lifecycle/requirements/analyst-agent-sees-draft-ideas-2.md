---
title: Agent Launcher Must Filter Input Artifacts by Approved Status
type: requirement
status: draft
lineage: analyst-agent-sees-draft-ideas
parent: defects/analyst-agent-sees-draft-ideas.md
labels:
    - agent
    - workflow
    - frontend
---

# Agent Launcher Must Filter Input Artifacts by Approved Status

## Problem

The agent launcher modal presents candidate artifacts to the user based on a hardcoded `predecessorMap` that determines which input status to filter by, derived from the agent's `active_status`. For the `analyst-requirements` agent (`active_status: clarifying`), this map resolves to `draft`, causing unapproved ideas to appear as valid targets. An operator can then invoke the analyst against an idea that has not been reviewed or approved, producing requirements from potentially incomplete or rejected input.

The root cause is in `web/src/components/agent/AgentLaunchModal.vue` (lines 26-32): the `predecessorMap` assumes a single fixed predecessor status per `active_status`, but the correct input status for any agent should always be `approved` — an artifact must be approved before the next lifecycle phase can begin.

## Goals / Non-goals

### Goals

- Ensure every agent in the launcher modal is only offered artifacts whose status is `approved`.
- Make the filtering logic consistent with the lifecycle state machine: an artifact must be explicitly approved before it can be consumed by the next phase's agent.
- Maintain the existing defect-fetching behaviour for developer agents (approved defects assigned to the agent's role should still appear).

### Non-goals

- Changing the workflow state machine or transition authority rules.
- Adding server-side validation of agent input status (a desirable hardening measure, but out of scope for this fix).
- Modifying agent configuration schema or `active_status` semantics.

## Detailed Requirements

### Functional

1. **FR-1: Input status filtering.** The agent launch modal MUST filter candidate artifacts to only those with `status: approved`. The current `predecessorMap` lookup MUST be replaced or corrected so that all agents receive `approved` as the input status filter, regardless of their `active_status`.

2. **FR-2: Input type filtering.** The existing `agentInputTypeMap` filtering (idea, requirement, plan-backend, etc.) MUST be preserved. Only the status filter is affected by this change.

3. **FR-3: Defect inclusion for developer agents.** Developer agents (`backend-developer`, `frontend-developer`, `test-developer`) MUST continue to see approved defects assigned to their role, in addition to their primary plan type. This existing behaviour must not regress.

4. **FR-4: Empty state.** When no artifacts match the filters (correct type + `approved` status), the modal MUST display the existing "No eligible artifacts for this agent." message.

### Non-functional

5. **NFR-1: No new API calls.** The fix should not require additional API endpoints or backend changes. The existing `listArtifacts` query parameter `status=approved` is sufficient.

6. **NFR-2: No agent config changes.** The `active_status` field in `lifecycle/config.yaml` retains its current meaning (the status the agent sets on the artifact it produces). It is not used for input filtering after this fix.

## Acceptance Criteria

- [ ] Opening the agent launcher for `analyst-requirements` shows only ideas with `status: approved`; ideas with `status: draft`, `clarifying`, `rejected`, or any other status do not appear.
- [ ] Opening the agent launcher for `analyst-planner` shows only requirements with `status: approved`.
- [ ] Opening the agent launcher for `backend-developer` shows only `plan-backend` artifacts with `status: approved`, plus any `defect` artifacts with `status: approved` assigned to the `backend-developer` role.
- [ ] Opening the agent launcher for `frontend-developer` shows only `plan-frontend` artifacts with `status: approved`, plus any `defect` artifacts with `status: approved` assigned to the `frontend-developer` role.
- [ ] Opening the agent launcher for `test-developer` shows only `plan-test` artifacts with `status: approved`, plus any `defect` artifacts with `status: approved` assigned to the `test-developer` role.
- [ ] Opening the agent launcher for `qa` shows only `test` artifacts with `status: approved`.
- [ ] When no artifacts of the correct type have `status: approved`, the modal displays "No eligible artifacts for this agent."
- [ ] The `predecessorMap` is either removed or corrected so that no agent can be offered non-approved input artifacts.
- [ ] Existing integration tests (if any) for the agent launcher continue to pass.
- [ ] Related: [[analyst-agent-sees-draft-ideas]]

## Open Questions

- Should the backend also enforce input status validation when an agent run is started via `POST /agents/runs`, as a defence-in-depth measure? (Deferred as a non-goal for this fix, but worth tracking.)
