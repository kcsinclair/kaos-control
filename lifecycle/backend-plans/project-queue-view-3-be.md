---
title: Project Queue View — Backend Plan
type: plan-backend
status: done
lineage: project-queue-view
created: "2026-07-14T19:34:44+10:00"
parent: lifecycle/requirements/project-queue-view-2.md
assignees:
    - role: product-owner
      who: agent
---

# Project Queue View — Backend Plan

Backend companion to the frontend plan [[project-queue-view]] (`project-queue-view-4-fe`)
and the test plan (`project-queue-view-5-test`).

## Summary and stance

The requirement's preferred data source (**FR-2 option 1**) is a **client-side
filter** of the already-subscribed `stores/queue.ts` snapshot. The existing
app-level queue snapshot already carries everything the project panel needs:

- Each `QueueJob` already includes a `project` field
  (`internal/http` queue handler / `web/src/api/queue.ts`), so the client can
  filter `running` / `pending` / `recent` by `job.project === activeProject`
  with **no new endpoint**.
- Cancellation already flows through the app-level `DELETE /api/queue/{id}`
  (FR-4), which is project-agnostic and needs no change.
- Real-time updates already broadcast over the app-level WebSocket
  (`queue.added`, `queue.started`, `queue.finished`, `queue.skipped`,
  `queue.cancelled`, `queue.paused`, `queue.resumed`) which the store already
  consumes (FR-5).

**Therefore the default, expected outcome of this plan is: no backend code
changes.** The work here is (a) *verifying* the existing API already satisfies
the frontend's needs, and (b) a **contingency** endpoint that is built **only if**
Milestone 1 proves the client-side approach insufficient (per the requirement's
explicit "in order of preference" wording in FR-2 and NFR-3).

Building the contingency endpoint speculatively would violate the "purely
additive / no impact on global queue" spirit of the requirement and add an
untested surface. Do not implement Milestone 2 unless Milestone 1 fails.

---

## Milestone 1 — Verify the existing snapshot satisfies FR-2/FR-4/FR-5 (required)

**Description.** Confirm, by reading the code and adding assertions to the
integration suite, that the existing app-level queue API already exposes the
per-job `project` field and per-project semantics the frontend needs — so the
frontend can filter client-side and the backend stays untouched.

**Files to inspect (no edits expected).**
- `internal/http/` queue handler — the `GET /api/queue` snapshot serialiser and
  the `QueueJob` JSON shape.
- `internal/http/` WebSocket broadcast for `queue.*` events — confirm each event
  payload identifies the affected job's `project` (directly, or via a job `id`
  the client already holds in a project-tagged snapshot entry).
- `internal/hub/` broadcast hub — confirm queue events are app-level and reach
  every connected client regardless of the project route.
- `web/src/api/queue.ts` — confirm the `QueueJob` TypeScript type already
  includes `project`, `artifact_path`, `agent_name`, `enqueued_at`, `position`,
  `state`.

**Acceptance criteria.**
- [x] `GET /api/queue` returns each job with a non-empty `project` field for
      running, pending, and recent slots. `Job.Project` is serialised as
      `json:"project"` (`internal/queue/types.go:36`) and asserted throughout
      `tests/integration/queue_api_test.go`.
- [x] `queue.added` / `queue.started` / `queue.finished` / `queue.cancelled`
      events allow a client to determine the affected job's project.
      `broadcastJobEvent` carries `id`, `project`, `artifact_path` and
      `agent_name` (`internal/queue/dispatcher.go:786-789`), and `queue.added`
      broadcasts the full persisted job record.
- [x] `DELETE /api/queue/{id}` cancels a pending job irrespective of the caller's
      current project route, and emits `queue.cancelled`
      (`internal/queue/dispatcher.go:698-706`; covered by
      `TestQueue_Cancel_Pending` and `TestQueue_Cancel_Running`).
- [x] Verification result recorded — see **Verification result** below.
- [x] No files under `internal/` were modified by this plan.

---

## Milestone 2 — CONTINGENCY: `GET /api/p/{project}/queue` — NOT REQUIRED

**Not built, and must not be.** Milestone 1 passed, so the contingency does not
apply. No `/api/p/{project}/queue` route exists and none should be added; the
plan's own stance is that building it speculatively would add an untested
surface for no benefit. The description below is retained for the record only.

### Original contingency description

**Description.** *Only if* Milestone 1 shows the client-side filter cannot work
(e.g. a WS event payload omits both `project` and a client-resolvable `id`, or
NFR-3's in-memory-filter constraint cannot be met), add a project-scoped
read-only endpoint that returns a `QueueSnapshot` pre-filtered to `{project}`.
It must reuse the existing in-memory snapshot path — it must not introduce a new
query path, new storage, or polling.

**Files to change.**
- `internal/http/` — register `GET /api/p/{project}/queue` on the existing chi
  router alongside the current queue routes; add a handler that takes the
  existing app-level snapshot and filters `running` (nil unless it belongs to
  `{project}`), `pending`, and `recent` by `project`.
- `internal/http/` (or wherever the queue snapshot is assembled) — extract the
  filter into a small helper reused by both the global and per-project handlers
  so behaviour cannot drift.
- `web/src/api/queue.ts` — add `listProjectQueue(project)` calling the new route
  (consumed by the frontend plan only if it switches to option 2).

**Acceptance criteria.**
- [ ] `GET /api/p/{project}/queue` returns a `QueueSnapshot` containing only jobs
      whose `project` equals the path param; `running` is `null` when the global
      running job belongs to another project.
- [ ] The endpoint reads from the **existing** in-memory snapshot (no new DB
      query, no polling) and responds in **under 50 ms** (NFR-3), verified by a
      timing assertion in the integration test.
- [ ] The global `GET /api/queue` response is **byte-for-byte unchanged**; all
      existing `tests/integration/queue_api_test.go` and related global-queue
      tests pass unchanged (NFR-1).
- [ ] Path-param `{project}` is validated/escaped consistently with other
      `/api/p/{project}/…` routes; an unknown project returns an empty snapshot
      (not a 500).
- [ ] `go build ./...` and `go vet ./...` pass with no new errors (NFR-5).

---

## Cross-cutting acceptance (applies to whichever milestone path is taken)

- [ ] **NFR-1 — No impact on global queue.** No existing queue endpoint,
      WebSocket event, or payload shape changes. Existing global-queue
      integration tests pass unchanged.
- [ ] **NFR-5 — Build hygiene.** If any Go file changes, `go build ./...` and
      `go vet ./...` pass with no new errors. If no Go file changes (expected
      default), this is trivially satisfied.
- [ ] The chosen path (option 1 default, option 2 contingency) is recorded so the
      frontend plan [[project-queue-view]] and test plan know which data source
      to wire against.

## Verification result

**Client-side filter is sufficient — no endpoint required.** (Verified 2026-09-03.)

Milestone 1 passed on every criterion and no backend code was written, which was
the plan's stated expected outcome. The feature shipped on option 1:
`web/src/components/agent/ProjectQueuePanel.vue` filters the app-level queue
store in the browser —

```js
return r && r.project === props.project ? r : null
queueStore.snapshot.pending.filter((j) => j.project === props.project)
```

— and is mounted from `AgentsRunsView.vue`. Every sibling in this lineage
(idea, requirement, frontend plan, test plan, and defects 6/7/8) is `done`.

Evidence: all 14 queue integration tests pass, including
`TestQueue_HappyPath_MultiProject`, `TestQueue_Cancel_Pending` and
`TestQueue_Cancel_Running`.

### The Open Questions below are resolved

All three concerned missing WebSocket broadcasts. They were accurate when this
plan was written on 2026-07-14 and have since been fixed in the dispatcher,
independently of this plan:

| Question | Resolution |
|---|---|
| `queue.added` not broadcast on the normal enqueue path | Fixed. `Dispatcher.Enqueue` broadcasts `queue.added` carrying the full persisted job record (`internal/queue/dispatcher.go:679-687`). |
| `queue.cancelled` never broadcast anywhere | Fixed. `Dispatcher.Cancel` broadcasts it via `broadcastJobEvent` (`internal/queue/dispatcher.go:698-706`). |
| Second client does not learn of enqueue/cancel in real time | Follows from the two above; both events now fire with `project` in the payload. |

Consequently the product decision the questions asked for is no longer needed,
and Milestone 1's criteria are satisfiable as written. The questions are kept
below unedited as the record of why this plan was blocked.

## Open Questions (resolved — see above)

1. **`queue.added` is not broadcast on the normal enqueue path.**
   `POST /api/queue` (`internal/http/queue.go: handleEnqueue`) calls
   `Dispatcher.Enqueue`, which is a bare one-line proxy to `Store.Enqueue`
   (`internal/queue/dispatcher.go:533`) with no broadcast call anywhere in
   that call chain. The only places `queue.added` is actually broadcast are
   two internal auto-retry paths — rate-limit re-enqueue
   (`internal/queue/dispatcher.go:462`) and auth-error re-enqueue
   (`internal/queue/dispatcher.go:514`) — and even there the payload carries
   only `id`, `position`, `attempts`, `reason` (no `project`, `artifact_path`,
   or `agent_name`), unlike `broadcastJobEvent`'s payload used for
   `queue.started`/`queue.finished`.

2. **`queue.cancelled` is never broadcast anywhere in the codebase.**
   `DELETE /api/queue/{id}` (`internal/http/queue.go: handleCancelQueue`)
   calls `Dispatcher.Cancel`, itself a bare proxy to `Store.Cancel`
   (`internal/queue/dispatcher.go:544`) with no broadcast call. No other code
   path emits `queue.cancelled`.

3. Consequently, a second connected client's queue view does **not** learn
   about a newly enqueued or newly cancelled job in real time — only the
   originating tab sees it (via the client-side optimistic insert in
   `web/src/stores/queue.ts: enqueue()`), until that other client does a full
   manual refetch. `web/src/stores/queue.ts` already contains handlers for
   both events, i.e. the frontend was written assuming they fire.

This gap is orthogonal to project-scoping: it exists identically whether the
plan takes option 1 (client-side filter) or option 2 (contingency endpoint),
because Milestone 2's endpoint only reshapes the on-demand snapshot — it does
not add the missing broadcast calls. So neither milestone as currently
written can be completed and marked correct without a product decision:

1. Is fixing the missing `queue.added` (on normal enqueue) and
   `queue.cancelled` broadcasts **in scope for this plan**, or is it a
   pre-existing defect to raise and fix independently of project-queue-view?

2. If in scope: is the expected fix adding
   `d.broadcastJobEvent("queue.added", &job, "pending")` in
   `Dispatcher.Enqueue` and `d.broadcastJobEvent("queue.cancelled", job,
   "cancelled")` in `Dispatcher.Cancel` (mirroring the existing
   `broadcastJobEvent` payload shape used by `queue.started`/`queue.finished`,
   which already includes `project`)? Does making this change count as
   "impact on the global queue" under NFR-1, or is it welcomed because it
   makes the existing global queue actually satisfy the real-time behaviour
   FR-5 already assumes?

3. Should Milestone 1's acceptance criteria be reworded — as literally
   written they can never be satisfied, since the events in question do not
   reliably fire (or carry `project`) on the primary user-facing code paths?
