---
title: "Queue Page Project Navigation — Backend Plan"
type: plan-backend
status: done
lineage: queue-page-project-navigation
parent: lifecycle/requirements/queue-page-project-navigation-2.md
created: "2026-05-13T00:00:00+10:00"
---

# Queue Page Project Navigation — Backend Plan

This feature requires **no new API endpoints**. The existing `GET /api/projects` and `GET /api/queue` endpoints already provide all data the frontend needs. This plan covers verification of existing behaviour and one minor backend improvement to ensure the project field is consistently populated in queue jobs.

## Milestone 1 — Verify Project Field Consistency in Queue Jobs

### Description

Confirm that every `QueueJob` emitted via the snapshot and WebSocket events includes a non-empty `project` field. The frontend sidebar filter relies on `job.project` matching a project name from the project list. If any code path produces a job with an empty or mismatched project name, the filter will silently drop those jobs.

### Files to review

- `internal/queue/dispatcher.go` — job creation and state transitions
- `internal/http/queue.go` — `handleEnqueue` validation, snapshot serialisation

### Acceptance Criteria

- [ ] `handleEnqueue` rejects requests where `project` is empty or does not match a registered project name (returns 400).
- [ ] Every `QueueJob` in the snapshot (`running`, `pending`, `recent`) has a `project` value that matches a key in the registered projects map.
- [ ] WebSocket `queue.*` event payloads include the `project` field unchanged from enqueue time.

## Milestone 2 — Verify Project List API Returns Stable Names

### Description

Confirm that `GET /api/projects` returns project names that exactly match the `project` field stored on queue jobs. The frontend will use strict string equality to filter jobs by project. Any casing or whitespace discrepancy would break filtering.

### Files to review

- `internal/http/server.go` (lines 342–358) — `GET /api/projects` handler
- `internal/project/` — project name resolution

### Acceptance Criteria

- [ ] Project names returned by `GET /api/projects` use the same canonical form stored in `QueueJob.project`.
- [ ] No trailing whitespace, path separators, or casing transformations are applied between registration and API output.

## Milestone 3 — Ensure Queue Snapshot Serialises Null-Safe for Frontend

### Description

The frontend defensive-codes empty arrays (`pending: []`, `recent: []`) but the sidebar job-count logic will iterate these arrays per-project. Verify the backend never returns `null` for array fields in the snapshot JSON, avoiding a class of runtime errors when the frontend counts jobs.

### Files to review

- `internal/http/queue.go` — `handleListQueue` serialisation
- `internal/queue/dispatcher.go` — `StateSnapshot()` return values

### Acceptance Criteria

- [ ] `GET /api/queue` returns `pending: []` (not `null`) when no jobs are pending.
- [ ] `GET /api/queue` returns `recent: []` (not `null`) when no recent jobs exist.
- [ ] `running` is serialised as `null` (not omitted) when no job is running.

## Cross-references

- [[queue-page-project-navigation]] frontend plan depends on the project name consistency verified in Milestones 1–2.
- [[queue-page-project-navigation]] test plan should include integration tests for empty-project and mismatched-project edge cases.
