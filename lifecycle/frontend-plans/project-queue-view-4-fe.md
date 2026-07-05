---
title: Project Queue View — Frontend Plan
type: plan-frontend
status: in-development
lineage: project-queue-view
parent: lifecycle/requirements/project-queue-view-2.md
---

# Project Queue View — Frontend Plan

Frontend implementation for embedding a project-scoped queue panel in the Agents
view. Pairs with the backend plan [[project-queue-view]] (`project-queue-view-3-be`,
expected outcome: **no backend change**, client-side filter) and the test plan
(`project-queue-view-5-test`).

## Design decisions grounded in the Resolved Questions

- **Q1 — always visible** (not collapsible). Render unconditionally when the
  Agents view is open.
- **Q2 — artifact drill-down: yes.** Job artifact links navigate to
  `/p/:project/artifacts/:artifact_path`.
- **Q3 — surface paused state** with a link to the global queue for details.
- **Q4 — no `enqueued_by`.** Omit the enqueuer within single-project context.
- **Q5 — waiting work only.** The panel shows **Running + Pending** (jobs waiting
  to run). It does **not** render a "Recent/completed" section — completed jobs
  are already visible in the run-history table on the same Agents view. This
  intentionally narrows FR-3's "Recent" bullet per the resolved decision; the
  pending-count indicator (FR-3) and all other sections remain.

A **new, self-contained `ProjectQueuePanel.vue`** is added rather than reusing the
global `QueuePendingTable` / `QueueRunningPanel` (which carry Project and
Enqueued-by columns sized for the full-page global view). This keeps the global
`QueueView` and its components **byte-for-byte unchanged** (NFR-1) and gives the
compact right-side panel its own responsive layout.

Data source: **FR-2 option 1** — client-side filter of the already-subscribed
`stores/queue.ts` snapshot by `job.project === project`. No new store, no new
endpoint, no polling (NFR-3). `AgentsRunsView.vue` already calls
`queueStore.fetch()` on mount and already reads `queueStore.isPaused`.

---

## Milestone 1 — `ProjectQueuePanel.vue` component (Running + Pending + empty/paused states)

**Description.** Create a presentational panel component that derives its data
from `useQueueStore()`, filtered to a `project` prop. Renders: a "Project Queue"
heading, a pending-count indicator, a Running section, a Pending list, a paused
banner affordance, an empty state, and a link to the global queue.

**Files to change / add.**
- `web/src/components/agent/ProjectQueuePanel.vue` (new). Props: `project: string`.
  - `running = computed(() => snapshot.running && snapshot.running.project === project ? snapshot.running : null)`.
  - `pending = computed(() => snapshot.pending.filter(j => j.project === project).sort((a,b) => a.position - b.position))`.
  - `pendingCount = computed(() => pending.value.length)`.
  - Heading "Project Queue"; pending-count indicator, e.g. `{{ pendingCount }} pending`.
  - **Running section:** when `running` is set, show agent (`agent_name`),
    artifact link, and started/elapsed; when null, show an explicit idle state
    ("Nothing running for this project"). The running/idle distinction must use
    text + icon, **not colour alone** (NFR-4).
  - **Pending section:** for each job show artifact (`artifact_path`, as a
    `RouterLink` to `/p/${encodeURIComponent(project)}/artifacts/${job.artifact_path}`
    — Q2), agent (`agent_name`), and enqueued time (`enqueued_at`, formatted).
    No Project column, no `enqueued_by` (Q4).
  - **Paused affordance (Q3):** when `queueStore.isPaused`, show a short paused
    note plus a link "see the global queue for details" → `/queue`.
  - **Empty state (FR-7):** when `!running && pending.length === 0`, show
    "No queued work for this project" (distinct from a load failure).
  - **Global-queue link (FR-6):** a clearly labelled `RouterLink` to `/queue`,
    e.g. "View global queue".
- `web/src/api/queue.ts` — reused as-is (no change) for the `QueueJob` type.

**Cancel wiring (FR-4).** Each pending row has a cancel/remove button calling
`queueStore.cancel(job.id)` (which already hits `DELETE /api/queue/{id}`), with
success/error surfaced via `useUiStore()` as in `QueuePendingTable.vue`.

**Acceptance criteria.**
- [ ] The panel shows the running job **only when** `running.project === project`;
      otherwise an explicit idle state (AC: running-only-when-belongs).
- [ ] Pending jobs for the project render in `position` order, each showing
      artifact, agent, and enqueued time.
- [ ] A pending-count indicator is shown with an `aria-label` such as
      "N jobs pending for this project" (NFR-4).
- [ ] Each pending job exposes a cancel action with `aria-label` "cancel queued
      job" that invokes `queueStore.cancel(id)` — no new cancellation path (FR-4).
- [ ] Clicking a job's artifact navigates to
      `/p/:project/artifacts/:artifact_path` (Q2).
- [ ] When globally paused, the panel shows the paused state and a link to
      `/queue` (Q3).
- [ ] When the project has no running and no pending jobs, the explicit empty
      state renders (FR-7).
- [ ] A labelled link navigates to `/queue` (FR-6).
- [ ] Running/idle status is conveyed by text/icon, not colour alone (NFR-4).
- [ ] No completed/recent section is rendered (Q5).

---

## Milestone 2 — Embed the panel on the right of `AgentsRunsView.vue` (responsive)

**Description.** Place `ProjectQueuePanel` on the **right side** of the Agents
view, beside the existing `AgentPanelRow` and the run-history table, without
displacing or altering them (FR-1). Make the layout reflow (stack) on narrow
viewports (NFR-2).

**Files to change.**
- `web/src/views/project/AgentsRunsView.vue`
  - Import and render `<ProjectQueuePanel :project="project" />`.
  - Introduce a two-column wrapper: main content (existing `AgentPanelRow` +
    runs table + pagination) on the left, `ProjectQueuePanel` on the right.
    Use a fl/grid layout with the panel given a fixed/max width and the main
    content `flex: 1`.
  - Add a `@media (max-width: …)` rule so the panel **stacks below** the main
    content on narrow widths rather than overflowing horizontally (NFR-2). Scope
    all new styles to this SFC; do not touch existing selectors' behaviour.
  - The existing pause banner block, `AgentPanelRow`, runs table, pagination, and
    all modals stay exactly as they are — the panel is added beside them.
  - `queueStore.fetch()` is already called on mount (kept). No new fetch/poll.

**Acceptance criteria.**
- [ ] `/p/:project/agents` renders the queue panel on the right, alongside the
      existing agent rows and run-history table (FR-1).
- [ ] The existing `AgentPanelRow` and run-history table are unchanged in
      structure and behaviour — the panel does not displace them (FR-1).
- [ ] `<ProjectQueuePanel>` receives the current `:project` route param so its
      filter matches the active project.
- [ ] On standard desktop widths there is no horizontal overflow; on narrow
      viewports the panel stacks below the main content (NFR-2).
- [ ] The global `QueueView.vue` and the global queue components are untouched;
      their existing tests pass unchanged (NFR-1).

---

## Milestone 3 — Real-time behaviour & build hygiene

**Description.** Because the panel derives from the shared, WS-subscribed store,
live updates require no extra wiring — but this milestone verifies and guards
that behaviour and the build.

**Files to change.**
- No new logic expected; adjustments only if reactivity gaps surface (ensure the
  panel's `computed`s read `queueStore.snapshot` reactively so `queue.added` /
  `queue.started` / `queue.finished` / `queue.cancelled` propagate).

**Acceptance criteria.**
- [ ] Enqueuing, starting, finishing, or cancelling a job for the current project
      updates the panel within 2 seconds with no page refresh (FR-5).
- [ ] Queue events for **other** projects do not appear in the panel (FR-5), since
      the `computed` filters by `project`.
- [ ] `pnpm build` and `vue-tsc --noEmit` pass with no new errors (NFR-5).
- [ ] No new polling is introduced; the panel reads only the existing store
      snapshot (NFR-3).

---

## Traceability

- Originating idea: [[project-queue-view]]
- Data-source contract with backend: [[project-queue-view]] (`project-queue-view-3-be`)
- Verified by: [[project-queue-view]] (`project-queue-view-5-test`)
