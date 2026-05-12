---
title: "Frontend Plan — Agent Work Queue UI"
type: plan-frontend
status: done
lineage: agent-rate-limit-queue
parent: lifecycle/requirements/agent-rate-limit-queue-2.md
created: "2026-05-12T10:45:00+10:00"
priority: high
labels:
    - agent
    - queue
    - frontend
    - vue
    - release-blocker
release: KC-Release1
---

# Frontend Plan — Agent Work Queue UI

Implements the FR14–FR16 UI requirements and the supporting Pinia store
that consumes the WS events defined in [[agent-rate-limit-queue-3-be]].

## File summary

```
web/src/
├── stores/
│   └── queue.ts                       NEW — Pinia store
├── api/
│   └── queue.ts                       NEW — REST wrappers
├── components/
│   ├── artifact/
│   │   └── QueueWorkButton.vue        NEW — the per-artefact button
│   ├── layout/
│   │   └── AppHeader.vue              EDIT — add badge
│   └── queue/
│       ├── QueueRunningPanel.vue      NEW
│       ├── QueuePendingTable.vue      NEW
│       ├── QueueRecentTable.vue       NEW
│       └── QueuePauseBanner.vue       NEW
├── views/project/
│   ├── ArtifactEditorView.vue         EDIT — render QueueWorkButton
│   └── QueueView.vue                  NEW — /queue page
└── router/
    └── index.ts                       EDIT — add /queue route
```

`AgentLaunchModal.vue` already holds the `agentInputTypeMap`; the queue
needs the same agent-name-for-type lookup. Export the map from a shared
module so both views stay in sync.

## Milestone 1 — Shared agent-for-type lookup

### Description

Move `agentInputTypeMap` out of `AgentLaunchModal.vue` into a shared
module so the Queue Work button uses the same mapping. Today the map
lives at `web/src/components/agent/AgentLaunchModal.vue:29-36`.

### Files to change

- **New** `web/src/composables/useAgentForArtifact.ts`:
  ```ts
  // typeToAgent: map of artefact type → agent name.
  export const typeToAgent: Record<string, string> = {
    idea: 'requirements-analyst',
    ticket: 'planning-analyst',
    requirement: 'planning-analyst',
    'plan-backend': 'backend-developer',
    'plan-frontend': 'frontend-developer',
    'plan-test': 'test-developer',
    test: 'qa',
  }

  // agentForArtifact returns the agent name to use for an artefact,
  // applying the defect→assignee-role rule. Returns null if no agent
  // would handle this artefact.
  export function agentForArtifact(
    artifact: { frontmatter: { type: string; assignees?: { role: string }[] } },
    agents: { name: string; roles: string[] }[],
  ): string | null { … }
  ```

- **Edit** `web/src/components/agent/AgentLaunchModal.vue` — import
  the map from the new module; delete the local copy. Verify the
  modal still works.

### Acceptance criteria

- Vitest unit test for `agentForArtifact` covering every type, the
  defect-assignee branch, and the no-match case.
- Existing `AgentLaunchModal` tests still pass.

---

## Milestone 2 — Queue REST + WS client

### Description

Thin wrappers around the new backend endpoints, plus a Pinia store
holding the queue state and reacting to WS events.

### Files to change

- **New** `web/src/api/queue.ts`:
  ```ts
  export interface QueueJob {
    id: string
    project: string
    artifact_path: string
    agent: string
    state: 'pending' | 'running' | 'completed' | 'failed' | 'skipped' | 'cancelled'
    reason?: string
    attempts: number
    enqueued_at: number
    started_at?: number
    finished_at?: number
    position: number
    enqueued_by: string
  }
  export interface QueueSnapshot {
    running: QueueJob | null
    pending: QueueJob[]
    recent: QueueJob[]
    paused: boolean
    paused_until: string | null
    pause_reason: 'rate_limit' | 'manual' | null
  }

  export const listQueue   = () => api.get<QueueSnapshot>('/queue')
  export const enqueue     = (b: { project: string; artifact_path: string; agent: string }) =>
                              api.post<{ id: string; position: number }>('/queue', b)
  export const cancelQueue = (id: string) => api.delete(`/queue/${id}`)
  export const pauseQueue  = () => api.post('/queue/pause', null)
  export const resumeQueue = () => api.post('/queue/resume', null)
  ```

- **New** `web/src/stores/queue.ts` (Pinia):
  - State: `snapshot` (the full `QueueSnapshot`), `loading`, `error`.
  - Getters: `pendingCount`, `isPaused`, `pausedUntilDate`.
  - Actions: `fetch()`, `enqueue(args)`, `cancel(id)`, `pause()`, `resume()`.
  - WS subscription (registered once on first use):
    - `queue.added` → push to `snapshot.pending` at correct position.
    - `queue.started` → move from `pending` to `running`.
    - `queue.finished` / `queue.skipped` / `queue.cancelled` → move to
      `recent` (cap at 10), clear `running`.
    - `queue.paused` → set `paused = true, paused_until = …`.
    - `queue.resumed` → set `paused = false, paused_until = null`.

### Acceptance criteria

- Vitest tests for the store covering each WS event type and verifying
  the snapshot mutates correctly without a refetch.
- `fetch()` is called once on store initialisation; subsequent state
  changes come from WS only.

---

## Milestone 3 — Queue Work button on artefact view

### Description

A new button rendered in the artefact editor's header next to the
existing "Change Status" / "Run Agent" controls. Visible only when:

- The artefact is in status `approved`.
- `agentForArtifact(artifact, agents)` returns non-null.
- The user has the role required to launch that agent (the backend
  returns 403 on enqueue otherwise; we still hide the button for users
  who clearly can't, by comparing roles client-side as a UX nicety).

If the artefact is already queued (matched via `queue.snapshot.pending`
or `queue.snapshot.running` with the same `(project, artifact_path)`),
the button is **replaced** by a "Queued — position N" badge.

### Files to change

- **New** `web/src/components/artifact/QueueWorkButton.vue`:
  - Props: `artifact`, `project`.
  - Uses `useQueueStore` and `useAgentsStore`.
  - Computes `agentName` via `agentForArtifact`.
  - Click → `queueStore.enqueue({ project, artifact_path, agent: agentName })`.
  - Renders disabled state with tooltip explaining why.

- **Edit** `web/src/views/project/ArtifactEditorView.vue`:
  - Import and render `<QueueWorkButton>` in the header action row
    (find the existing action-row template and append).

### Acceptance criteria

- Vitest:
  - Renders the button when artefact is `approved` and a matching agent
    exists; click triggers `enqueue` with the correct args.
  - Hides the button when status != `approved`.
  - Renders the "Queued — position N" badge when the artefact is
    already in the queue.

---

## Milestone 4 — `/queue` page

### Description

A new top-level page (not project-scoped) showing the full queue state.
Reachable from the new header badge.

Layout (top to bottom):

1. **Pause banner** (`QueuePauseBanner.vue`) — only when paused. Shows:
   - Reset time in user-local timezone with "in 3h 17m" relative tag.
   - Pause reason (`rate_limit` / `manual`).
   - "Resume now" button for users with product-owner / devops role.

2. **Running** (`QueueRunningPanel.vue`) — either empty state ("nothing
   running") or one row: agent, project, artefact link, started-at,
   elapsed timer, link to the run log on the existing agent runs view.

3. **Pending** (`QueuePendingTable.vue`) — FIFO table with columns
   position / project / artefact / agent / enqueued-at / enqueued-by /
   actions (Remove). Each artefact name is a clickable link to its
   artefact view.

4. **Recently finished** (`QueueRecentTable.vue`) — last 10 terminal
   items. Same columns plus terminal-state badge and reason.

### Files to change

- **New** files listed under "File summary" above.
- **Edit** `web/src/router/index.ts` — add a route:
  ```ts
  {
    path: '/queue',
    name: 'queue',
    component: () => import('@/views/QueueView.vue'),
    meta: { requiresAuth: true },
  }
  ```
  (Note: not under `/p/:project/` because the queue is global.)

### Acceptance criteria

- Vitest:
  - Renders running / pending / recent sections from a mocked store.
  - Pause banner appears only when `paused === true`.
  - "Resume now" button visible only for product-owner / devops.
  - "Remove" on a pending row calls `queueStore.cancel(id)`.

---

## Milestone 5 — Header badge

### Description

A small badge in `AppHeader.vue` showing:
- Number of pending jobs.
- A pause icon overlay when paused.
- Tooltip "Queue: N pending" or "Queue paused, resumes …" based on
  state.
- Click navigates to `/queue`.

Match the existing run-indicator badge styling already in the header.

### Files to change

- **Edit** `web/src/components/layout/AppHeader.vue` — add the badge,
  bind to `queueStore.pendingCount` and `queueStore.isPaused`.

### Acceptance criteria

- Vitest: badge renders count, switches to paused state when store
  emits `queue.paused`, click pushes the router to `/queue`.

---

## Milestone 6 — Live elapsed-time / countdown

### Description

Two small reactive timers needed for UX polish:

- The running-job panel needs an "elapsed" string that updates every
  second.
- The pause banner needs a "resumes in 3h 17m" string that updates
  every second (down to seconds when under a minute).

Use a single shared `useNow()` composable that returns a `Ref<Date>`
ticking once per second on mount, stopped on unmount. Avoid creating
multiple intervals.

### Files to change

- **New** `web/src/composables/useNow.ts`.
- **Edit** `QueueRunningPanel.vue` and `QueuePauseBanner.vue` — derive
  the displayed string from `useNow()`.

### Acceptance criteria

- Vitest: with fake timers, advance time by 65s and verify the elapsed
  string updates correctly (`1m 5s` or similar).

---

## Verification (end-to-end)

1. `pnpm build` clean.
2. `pnpm test` — all existing + new vitest cases pass.
3. Manual smoke:
   - Approve an idea in two projects.
   - Click Queue Work on both.
   - Verify the `/queue` page shows them in order.
   - Verify the header badge shows `2 pending`.
   - Pause manually; verify banner appears, "Resume now" works.
   - Trigger a real rate-limit (or use a test fixture) and verify the
     auto-pause banner shows the reset time + 5 min.
   - Cancel a pending item; verify it disappears from the list.

## Risk notes

- **Header badge clutter.** The header already has a run-indicator
  badge. Confirm with design (or pick the lower-priority position) so
  the two don't compete visually. If the run-indicator badge is for
  the same use case, consider merging.

- **Route depth.** `/queue` is a sibling of `/projects` rather than
  under `/p/:project/`. Make sure the `AppSidebar` doesn't try to
  render project-scoped nav while on `/queue`.

- **`agentForArtifact` divergence.** This must stay in sync with the
  backend's enqueue-side selection logic. Add a single fixture file
  shared by frontend and backend tests if drift becomes a concern.
