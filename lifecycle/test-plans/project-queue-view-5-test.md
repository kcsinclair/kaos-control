---
created: "2026-07-14T19:34:44+10:00"
title: Project Queue View — Test Plan
type: plan-test
status: done
lineage: project-queue-view
parent: lifecycle/requirements/project-queue-view-2.md
---

# Project Queue View — Test Plan

Verifies the backend plan [[project-queue-view]] (`project-queue-view-3-be`) and
frontend plan (`project-queue-view-4-fe`) against every FR/NFR and Resolved
Question in the requirement. Emphasis is on the frontend, since the default
implementation is a client-side filter with **no backend change**.

Test stacks in this repo:
- **Frontend component/store tests** — Vitest under `tests/web/` (e.g.
  `QueuePendingTable.filter.test.ts`, `queueStore.test.ts`, `QueueView.test.ts`),
  mounting SFCs with a Pinia store and asserting rendered output.
- **Backend integration tests** — Go under `tests/integration/` (e.g.
  `queue_api_test.go`); only exercised if the backend contingency endpoint is built.

---

## Milestone 1 — Backend verification / regression guard

**Description.** Assert the existing app-level queue API already provides the
per-project data the frontend needs, and guard that the global queue is
unchanged. If (and only if) the backend contingency endpoint
(`project-queue-view-3-be` M2) is built, add tests for it.

**Files to change / add.**
- `tests/integration/queue_api_test.go` (extend) or a new
  `tests/integration/project_queue_test.go`.

**Acceptance criteria.**
- [ ] A test asserts `GET /api/queue` returns each job (running/pending/recent)
      with a non-empty `project` field — the fact the client-side filter relies on.
- [ ] A test asserts `DELETE /api/queue/{id}` cancels a pending job and emits
      `queue.cancelled` regardless of any project context (FR-4 backing).
- [ ] Existing global-queue integration tests continue to pass **unchanged**
      (NFR-1).
- [ ] **Contingency only:** if `GET /api/p/{project}/queue` is added, tests assert
      it returns a snapshot filtered to `{project}` (running null when the global
      running job belongs elsewhere), responds < 50 ms (NFR-3), and leaves
      `GET /api/queue` byte-for-byte unchanged (NFR-1).

---

## Milestone 2 — `ProjectQueuePanel.vue` unit/component tests

**Description.** Mount `ProjectQueuePanel` with a seeded queue store snapshot
containing jobs across **multiple** projects and assert correct project-scoped
rendering, drill-down, cancel, empty, and paused states.

**Files to change / add.**
- `tests/web/ProjectQueuePanel.test.ts` (new), following the pattern of
  `QueuePendingTable.filter.test.ts`.

**Acceptance criteria.**
- [ ] Given a snapshot with pending jobs for project A and project B, the panel
      for project A shows only A's jobs, in `position` order (FR-2, FR-3).
- [ ] Each pending row renders artifact (`artifact_path`), agent (`agent_name`),
      and enqueued time (`enqueued_at`); it does **not** render `enqueued_by` (Q4)
      and there is **no** Recent/completed section (Q5).
- [ ] The Running section shows the running job only when its `project` matches
      the panel's project; with the running job owned by another project, the
      idle state renders (FR-2, AC running-only-when-belongs).
- [ ] The pending-count indicator shows the correct count and exposes an
      `aria-label` like "N jobs pending for this project" (FR-3, NFR-4).
- [ ] Clicking a pending job's artifact produces a `RouterLink`/navigation to
      `/p/:project/artifacts/:artifact_path` (Q2).
- [ ] The cancel action for a pending row calls the store's `cancel(id)` with the
      right job id and exposes an `aria-label` "cancel queued job" (FR-4, NFR-4).
- [ ] With no running and no pending jobs for the project, the explicit empty
      state "No queued work for this project" renders (FR-7).
- [ ] When the store reports `isPaused`, the panel shows the paused state and a
      link to `/queue` (Q3).
- [ ] A labelled link to `/queue` is present (FR-6).
- [ ] Running/idle is distinguishable without colour (assert on text/icon/aria,
      not just class/colour) (NFR-4).

---

## Milestone 3 — Real-time update tests (store-driven)

**Description.** Drive the queue store through the WS-event handlers it already
implements and assert the panel reflects changes for the current project only.

**Files to change / add.**
- `tests/web/ProjectQueuePanel.realtime.test.ts` (new), or extend
  `queueStore.test.ts` for the store side plus a mounted-panel assertion.

**Acceptance criteria.**
- [ ] Simulating `queue.added` for the current project adds the job to the panel's
      pending list (reactively, no refetch) (FR-5).
- [ ] Simulating `queue.started` moves the job into the panel's Running section
      when it belongs to the project (FR-5).
- [ ] Simulating `queue.finished` / `queue.cancelled` removes the job from the
      panel's waiting sections (FR-5); no lingering completed row (Q5).
- [ ] A `queue.added` event for a **different** project does **not** appear in the
      panel (FR-5, other-projects-excluded).
- [ ] `queue.paused` / `queue.resumed` toggle the panel's paused affordance (Q3).

---

## Milestone 4 — Layout, integration, and global-queue regression

**Description.** Verify the panel is embedded in `AgentsRunsView.vue` without
disturbing existing content, reflows responsively, and leaves the global queue
untouched.

**Files to change / add.**
- `tests/web/AgentsRunsView.projectQueue.test.ts` (new) — mount
  `AgentsRunsView` and assert the panel is present alongside `AgentPanelRow` and
  the runs table.
- Reuse existing `tests/web/QueueView.test.ts` and the global queue component
  filter tests as the NFR-1 regression guard (run unchanged).

**Acceptance criteria.**
- [ ] `AgentsRunsView` renders `ProjectQueuePanel` alongside the existing agent
      rows and run-history table; those existing elements are still present and
      unchanged (FR-1).
- [ ] The panel receives the current `:project` route param (asserted via the
      rendered project-scoped content).
- [ ] A responsive assertion (or documented manual check) confirms the panel
      stacks below main content on narrow viewports with no horizontal overflow
      (NFR-2). Where jsdom cannot measure layout, this is covered by a documented
      manual viewport check plus the presence of the stacking media-query class.
- [ ] All existing global-queue tests (`QueueView.test.ts`,
      `QueuePendingTable.filter.test.ts`, `QueueRunningPanel.filter.test.ts`,
      `QueueRecentTable.filter.test.ts`, `QueueView.projectNav.test.ts`) pass
      **unchanged** (NFR-1).

---

## Milestone 5 — Build/quality gates

**Description.** Ensure the whole change compiles and typechecks cleanly.

**Acceptance criteria.**
- [ ] `pnpm build` and `vue-tsc --noEmit` pass with no new errors (NFR-5).
- [ ] The Vitest suite (existing + new) passes (`pnpm test` in `web/` /
      `tests/web`).
- [ ] If any Go changed (contingency only), `go build ./...`, `go vet ./...`, and
      `go test ./... -short` pass with no new errors (NFR-5).

---

## Coverage map (requirement → milestone)

- FR-1 embedded right-side panel → M4
- FR-2 project-scoped source → M1 (data), M2 (filter)
- FR-3 sections + pending count (Running/Pending; Recent narrowed per Q5) → M2
- FR-4 cancel via existing DELETE → M1 (backend), M2 (wiring)
- FR-5 real-time updates → M3
- FR-6 link to global queue → M2
- FR-7 empty state → M2
- NFR-1 no global impact → M1, M4
- NFR-2 layout integrity → M4
- NFR-3 in-memory filter, no polling → M1, M3
- NFR-4 accessibility → M2
- NFR-5 build hygiene → M5
- Q1 always visible → M4; Q2 drill-down → M2; Q3 paused → M2/M3;
  Q4 no enqueued_by → M2; Q5 waiting-only → M2/M3

## Traceability

- Requirement: `lifecycle/requirements/project-queue-view-2.md`
- Under test: [[project-queue-view]] (`project-queue-view-3-be`, `project-queue-view-4-fe`)
- Originating idea: [[project-queue-view]]
