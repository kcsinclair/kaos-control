---
title: Project Queue View — Backend Plan
type: plan-backend
status: approved
lineage: project-queue-view
parent: lifecycle/requirements/project-queue-view-2.md
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
- [ ] `GET /api/queue` returns each job with a non-empty `project` field for
      running, pending, and recent slots (asserted by an integration test —
      see `project-queue-view-5-test` M1).
- [ ] `queue.added` / `queue.started` / `queue.finished` / `queue.cancelled`
      events allow a client to determine the affected job's project (either the
      payload carries `project`, or it carries an `id` already present in the
      client's project-filtered snapshot).
- [ ] `DELETE /api/queue/{id}` cancels a pending job irrespective of the caller's
      current project route, and emits `queue.cancelled` (existing behaviour,
      re-asserted).
- [ ] A written note is added to this plan (or the test plan) recording the
      verification result: **"client-side filter is sufficient — no endpoint
      required"** OR **"insufficient because X — proceed to Milestone 2"**.
- [ ] No files under `internal/` are modified.

---

## Milestone 2 — CONTINGENCY: `GET /api/p/{project}/queue` (build only if M1 fails)

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
