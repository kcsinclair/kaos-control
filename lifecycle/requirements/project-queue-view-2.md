---
title: Project Queue View in Agents Panel
type: requirement
status: approved
lineage: project-queue-view
created: "2026-06-27T00:00:00+10:00"
priority: high
parent: lifecycle/ideas/project-queue-view.md
labels:
    - queue
    - frontend
    - feature
    - agent
    - vue
release: KC-Release4
assignees:
    - role: product-owner
      who: agent
---

## Problem

The queue is a single app-level structure: every job carries a `project` field, but the only place a user can see queued work is the global `QueueView` (route `/queue`), which lists jobs across all projects. Inside a project, the Agents panel (`AgentsRunsView.vue`, route `/p/:project/agents`) shows agent quick-launch rows and a run-history table, but nothing about what is *waiting* to run.

As a result, a user working inside a project must leave the project context, open the global queue, and visually filter by project to answer "what work is pending for the agents I'm looking at?" This breaks the feedback loop between the agents shown on screen and the jobs those agents will process, and discourages monitoring of queue depth from where the work is actually launched.

## Goals / Non-goals

### Goals

- Embed a **project-scoped queue view** on the right side of the Agents panel (`AgentsRunsView.vue`), showing only jobs whose `project` matches the current `:project` route param.
- Show the same lifecycle states the global queue shows — the currently running job, pending jobs in order, and recently finished jobs — but filtered to this project.
- Keep the project queue **live**, updating in real time from the existing queue WebSocket events without page refresh or polling.
- Provide a clear **link to the global queue** (`/queue`) for users who need the system-wide, cross-project view.
- Be **purely additive**: the global `QueueView` and all existing queue endpoints, events, and behaviour are unchanged.

### Non-goals

- Changing or removing the global queue view, its layout, or its routes.
- Adding new queue *control* behaviour beyond what already exists (e.g. no new pause/resume semantics; cancelling a pending job reuses the existing `DELETE /api/queue/{id}`).
- Adding per-project pause/resume — pause state remains a global queue property.
- Replacing the existing `AgentPanelRow` ready/running badges or the run-history table.
- Server-side push of a per-project filtered snapshot (filtering may be done client-side from the existing store — see FR-2).
- Re-architecting the queue from app-level to per-project ownership.

## Detailed Requirements

### Functional

#### FR-1: Project queue panel embedded in the Agents view

`AgentsRunsView.vue` must render a queue panel on the **right side** of the Agents view, alongside the existing agent rows and run-history table. The panel shows queue state filtered to the current project (the `:project` route param).

- The panel must be visually distinct as a queue region (heading such as "Project Queue").
- The panel must not displace or alter the existing agent quick-launch rows (`AgentPanelRow`) or run-history table — it is added beside them.
- Layout must be responsive: on narrow viewports the panel may stack below the main content rather than overflow horizontally (see NFR-2).

#### FR-2: Project-scoped data source

The panel must display only jobs whose `project` equals the active project. Two implementations are acceptable, in order of preference:

1. **Client-side filter** of the existing `stores/queue.ts` `snapshot` (`running`, `pending`, `recent`) by `job.project === activeProject`. This reuses the already-subscribed queue store and requires no new endpoint.
2. A new endpoint `GET /api/p/:project/queue` returning a `queueSnapshot` pre-filtered to the project, if the client-side approach proves insufficient.

Whichever is chosen, the `running` slot must show the running job **only if** that job belongs to the current project (the queue runs at most one job globally, so this slot is frequently empty for a given project).

#### FR-3: Sections shown

The project queue panel must show, scoped to the current project:

- **Running** — the single currently running job if it belongs to this project; otherwise an empty/idle state.
- **Pending** — jobs in `pending` state for this project, in queue order (by `position`). Each entry shows at minimum: artifact (from `artifact_path`), agent (`agent_name`), and enqueued time (`enqueued_at`).
- **Recent** — recently finished jobs for this project (`completed` / `failed` / `skipped` / `cancelled`), reusing the global queue's recent window.

A pending-count indicator (e.g. "3 pending") must be shown for the project.

#### FR-4: Cancel a pending job

Each pending job in the project panel must offer a cancel/remove action that calls the existing `DELETE /api/queue/{id}` (via the queue store `cancel(id)` action). No new cancellation path is introduced.

#### FR-5: Real-time updates

The panel must update live from the existing app-level queue WebSocket events already handled by `stores/queue.ts`: `queue.added`, `queue.started`, `queue.finished`, `queue.skipped`, `queue.cancelled`, `queue.paused`, `queue.resumed`. Because the panel derives from the shared store, a job added/started/finished/cancelled for the current project must appear in or leave the panel within 2 seconds without page refresh. Events for other projects must not appear in the panel.

#### FR-6: Link to global queue

The panel must include a clearly labelled link/affordance that navigates to the global queue view (`/queue`), so users can switch from the project-scoped subset to the system-wide perspective.

#### FR-7: Empty state

When the project has no running, pending, or recent jobs, the panel must show an explicit empty state (e.g. "No queued work for this project") rather than rendering blank — so users can distinguish "nothing queued" from "panel failed to load".

### Non-functional

#### NFR-1: No impact on global queue

No existing global queue endpoint, WebSocket event, store action, or component behaviour changes. The global `QueueView` renders identically to before. This must be verifiable by existing global-queue tests continuing to pass unchanged.

#### NFR-2: Layout integrity

Adding the panel must not cause horizontal overflow or break the existing Agents view layout on standard desktop widths. On narrow viewports the panel reflows (stacks) rather than clipping content.

#### NFR-3: Performance

Deriving the project-filtered view from the existing snapshot must be an in-memory filter; it must not add new polling. If FR-2 option 2 (a new endpoint) is used, it must reuse the existing snapshot path and respond in under 50 ms.

#### NFR-4: Accessibility

The pending-count indicator and cancel actions must be screen-reader accessible (e.g. `aria-label` such as "3 jobs pending for this project", "cancel queued job"). The running/idle distinction must not rely on colour alone.

#### NFR-5: Build hygiene

`pnpm build` and `vue-tsc --noEmit` pass with no new errors. If a backend endpoint is added, `go build ./...` and `go vet ./...` pass with no new errors.

## Acceptance Criteria

- [ ] The Agents view (`/p/:project/agents`) renders a queue panel on the right side, alongside the existing agent rows and run-history table
- [ ] The panel shows only jobs whose `project` equals the current `:project` route param
- [ ] The running job appears in the panel only when it belongs to the current project; otherwise an idle/empty running state is shown
- [ ] Pending jobs for the project are listed in `position` order, each showing artifact, agent, and enqueued time
- [ ] Recent finished jobs for the project are shown
- [ ] A pending-count indicator for the project is displayed
- [ ] Each pending job can be cancelled via the existing `DELETE /api/queue/{id}` path
- [ ] Enqueuing, starting, finishing, or cancelling a job for the current project updates the panel within 2 seconds with no page refresh
- [ ] Queue events for other projects do not appear in the panel
- [ ] The panel includes a link that navigates to the global queue view (`/queue`)
- [ ] An explicit empty state is shown when the project has no running, pending, or recent jobs
- [ ] The global `QueueView` and all existing global queue endpoints/events are unchanged; existing global-queue tests pass unchanged
- [ ] The panel reflows (stacks) on narrow viewports without horizontal overflow
- [ ] Pending-count and cancel actions expose screen-reader labels; running/idle state does not rely on colour alone
- [ ] `pnpm build` and `vue-tsc --noEmit` pass with no new errors; if a backend endpoint is added, `go build ./...` and `go vet ./...` pass with no new errors
- [ ] Originating idea: [[project-queue-view]]

## Resolved Questions

- **Q1**: Should the project queue panel be collapsible/dismissible, or always visible when the Agents view is open?

> Always visible.

- **Q2**: Should clicking a queued job's artifact navigate to that artifact (drill-down), consistent with the ready-count badge drill-down in [[agent-panel-status-and-ready-count]]?

> Yes

- **Q3**: When the queue is globally paused (rate-limit or manual), should the project panel surface the paused state and reason, or stay silent and leave pause status to the global view only?

> It should show that it is paused and include a link to view the global queue, e.g. "see here for details".

- **Q4**: Should the project panel show the user who enqueued each job (`enqueued_by`), as the global pending table does, or is that noise within a single-project context?

> No, not needed at the moment.

- **Q5**: How many "recent" jobs should the project panel show — the same global window of 10 (which may be mostly other projects' jobs and yield few project entries), or a project-scoped count of the last N for this project?
 
> The project level queue should only show the jobs waiting to be run, the completed jobs will be visibile in the current agent view.
