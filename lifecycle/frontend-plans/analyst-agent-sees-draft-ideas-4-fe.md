---
title: "Agent Launcher Input Status Filtering — Frontend Plan"
type: plan-frontend
status: approved
lineage: analyst-agent-sees-draft-ideas
parent: lifecycle/requirements/analyst-agent-sees-draft-ideas-2.md
---

# Agent Launcher Input Status Filtering — Frontend Plan

Replace the `predecessorMap` in `AgentLaunchModal.vue` with a hardcoded `approved` status filter so that every agent is only offered artifacts that have been explicitly approved. The `agentInputTypeMap` and defect-fetching logic are preserved. See [[analyst-agent-sees-draft-ideas]] backend plan for API confirmation.

## Milestone 1: Remove predecessorMap and Hardcode Approved Status

### Description

Delete the `predecessorMap` object (lines 26–32 of `AgentLaunchModal.vue`) and the `inputStatus` computed property that derives from it. Replace with a simple constant: the input status for all agents is always `'approved'`. This directly addresses FR-1.

### Files to Change

- `web/src/components/agent/AgentLaunchModal.vue` — remove the `predecessorMap` object and the `inputStatus` computed property. Replace with `const inputStatus = 'approved'` (or inline `'approved'` directly in the `fetchArtifacts` filter).

### Acceptance Criteria

- [ ] The `predecessorMap` object no longer exists in `AgentLaunchModal.vue`.
- [ ] The `inputStatus` computed property is removed or replaced with a constant `'approved'`.
- [ ] The `fetchArtifacts` function passes `status: 'approved'` in its filter object for the primary artifact query.
- [ ] The `agentInputTypeMap` is unchanged — type filtering (FR-2) continues to work as before.
- [ ] No new API endpoints are called (NFR-1).

## Milestone 2: Verify Defect Inclusion for Developer Agents

### Description

Confirm that the existing defect-fetching block (lines 70–79) continues to work correctly after Milestone 1. This block already hardcodes `status: 'approved'` for defects, so no change is needed. This milestone is a verification pass to ensure FR-3 is preserved.

### Files to Change

- `web/src/components/agent/AgentLaunchModal.vue` — no changes expected; review and confirm the defect-fetching block still uses `status: 'approved'` and filters by agent role.

### Acceptance Criteria

- [ ] Developer agents (`backend-developer`, `frontend-developer`, `test-developer`) see approved defects assigned to their role in the artifact list.
- [ ] The defect query uses `status: 'approved'` and `type: 'defect'` (already the case — confirm no regression).
- [ ] Non-developer agents (e.g. `analyst-requirements`, `qa`) do not see defect artifacts in their list.

## Milestone 3: Verify Empty State Behaviour

### Description

Confirm that the "No eligible artifacts for this agent." message (FR-4) continues to display when no artifacts match the filters. Since the filtering is now stricter (only `approved` instead of potentially `draft`), more agents may encounter the empty state. No template changes are needed — the existing `v-else` branch handles this.

### Files to Change

- `web/src/components/agent/AgentLaunchModal.vue` — no changes expected; confirm the empty state rendering path is intact.

### Acceptance Criteria

- [ ] When no artifacts of the correct type have `status: approved`, the modal displays "No eligible artifacts for this agent."
- [ ] The empty state message is shown for all agent types when their input type has no approved artifacts.
- [ ] The loading state ("Loading artifacts…") still displays while the API call is in flight.
