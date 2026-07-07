---
title: Queue does not broadcast queue.added on normal enqueue, or queue.cancelled at all
type: defect
status: in-development
lineage: queue-events-missing-broadcasts
created: "2026-07-05T00:00:00+10:00"
priority: medium
labels:
    - defect
    - queue
    - websocket
    - realtime
assignees:
    - role: backend-developer
      who: agent
---

# Queue does not broadcast `queue.added` on normal enqueue, or `queue.cancelled` at all

## Summary

Two app-level queue WebSocket events that the frontend already listens for do
not reliably fire on the primary user-facing code paths, so a second connected
client's queue view does not update in real time when a job is enqueued or
cancelled. Surfaced while assessing the project-queue-view backend plan
([lifecycle/backend-plans/project-queue-view-3-be.md](../backend-plans/project-queue-view-3-be.md),
Open Questions §1–§3); the gap is **orthogonal to project-scoping** and blocks
that plan's Milestone 1 acceptance from being truthfully met.

## Reproduction Steps

1. Open the queue view in two separate browser tabs/clients.
2. In tab A, enqueue a job (`POST /api/queue`).
3. Observe tab B: the new job does **not** appear until a full manual refetch.
   (Tab A shows it only via the client-side optimistic insert in
   `web/src/stores/queue.ts: enqueue()`.)
4. In tab A, cancel a pending job (`DELETE /api/queue/{id}`).
5. Observe tab B: the job is **not** shown as cancelled in real time.

## Expected Behaviour

Enqueue and cancel broadcast app-level `queue.added` / `queue.cancelled` events
(carrying at least `id`, `project`, `artifact_path`, `agent_name`, `state`,
`position`) to every connected client, so all queue views stay live — matching
FR-5's assumption and the handlers already present in
`web/src/stores/queue.ts`.

## Actual Behaviour

- **`queue.added` is not broadcast on the normal enqueue path.**
  `POST /api/queue` (`internal/http/queue.go: handleEnqueue`) calls
  `Dispatcher.Enqueue`, a one-line proxy to `Store.Enqueue`
  ([internal/queue/dispatcher.go:533](../../internal/queue/dispatcher.go#L533))
  with no broadcast in that chain. `queue.added` is only broadcast from the two
  internal auto-retry paths — rate-limit re-enqueue
  ([dispatcher.go:462](../../internal/queue/dispatcher.go#L462)) and auth-error
  re-enqueue ([dispatcher.go:514](../../internal/queue/dispatcher.go#L514)) —
  and even there the payload carries only `id`, `position`, `attempts`,
  `reason` (no `project`/`artifact_path`/`agent_name`), unlike
  `broadcastJobEvent`.
- **`queue.cancelled` is never broadcast anywhere.**
  `DELETE /api/queue/{id}` (`internal/http/queue.go: handleCancelQueue`) calls
  `Dispatcher.Cancel`, a bare proxy to `Store.Cancel`
  ([dispatcher.go:544](../../internal/queue/dispatcher.go#L544)); no code path
  emits `queue.cancelled`.

The snapshot (`GET /api/queue`) is correct — a cancelled job shows in `recent`
with `state:"cancelled"` — so this is purely a **live-broadcast** gap, not a
state gap.

## Suggested Fix

Mirror the existing `broadcastJobEvent`
([dispatcher.go:613](../../internal/queue/dispatcher.go#L613)), whose payload
**already includes `project`** ([dispatcher.go:616](../../internal/queue/dispatcher.go#L616)):

1. In `Dispatcher.Enqueue`, after a successful `Store.Enqueue`, call
   `d.broadcastJobEvent("queue.added", &job, "pending")`.
2. In `Dispatcher.Cancel`, after a successful `Store.Cancel`, call
   `d.broadcastJobEvent("queue.cancelled", job, "cancelled")`.

Because that payload carries `project`, fixing this also makes the
project-queue-view client-side filter (FR-2 option 1 / Milestone 1) fully
real-time — reinforcing that **Milestone 2's contingency endpoint is not
needed**.

## Scope note (NFR-1)

This *adds* two broadcasts the frontend already expects; it does not change any
existing event payload shape or the snapshot. It should be treated as making the
global queue satisfy the real-time behaviour FR-5 already assumes, not as a
breaking change to global-queue behaviour.

## Verification

- Integration test: connect a WS client, `POST /api/queue`, assert a
  `queue.added` event arrives carrying `project`/`artifact_path`/`agent_name`;
  `DELETE /api/queue/{id}`, assert a `queue.cancelled` event arrives.
- Assert `GET /api/queue` and all existing global-queue tests are unchanged.
