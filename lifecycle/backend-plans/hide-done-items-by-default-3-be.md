---
title: "Hide Done Items by Default — Backend Plan"
type: plan-backend
status: approved
lineage: hide-done-items-by-default
parent: lifecycle/requirements/hide-done-items-by-default-2.md
---

# Hide Done Items by Default — Backend Plan

The requirement (§FR-8) explicitly states "No backend API changes are required; filtering is purely a frontend concern." The API must continue to return all artifacts regardless of status.

This plan documents the backend non-changes, defines the shared constant that the frontend will rely on, and ensures the backend does not inadvertently interfere.

## Milestone 1: Confirm No Backend Changes Required

### Description

Verify that the existing `GET /p/:project/artifacts`, `GET /p/:project/graph`, and `GET /p/:project/config/kanban` endpoints return artifacts of all statuses including `done`, `rejected`, and `abandoned`. No code changes are required.

### Files to Change

None.

### Acceptance Criteria

- [ ] The `GET /p/:project/artifacts` endpoint returns artifacts with `done`, `rejected`, and `abandoned` statuses in its response.
- [ ] The `GET /p/:project/graph` endpoint includes nodes with terminal statuses in its response.
- [ ] No backend Go code is modified for this feature.

## Milestone 2: Document Terminal Status Convention

### Description

The set of terminal statuses (`done`, `rejected`, `abandoned`) is already implicitly defined by the kanban config in `lifecycle/config.yaml` (the "Done" column maps to these three statuses). The frontend plan [[hide-done-items-by-default]] will define the `TERMINAL_STATUSES` constant client-side. No backend constant or API surface is needed since filtering is entirely a frontend concern.

### Files to Change

None.

### Acceptance Criteria

- [ ] The kanban configuration in `lifecycle/config.yaml` continues to list `done`, `abandoned`, `rejected` as the statuses for the "Done" column, serving as the canonical definition.
- [ ] No new backend API fields, query parameters, or response shape changes are introduced.
