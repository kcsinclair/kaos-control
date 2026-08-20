---
created: "2026-07-14T19:34:44+10:00"
title: ProjectQueuePanel realtime tests — persistent WS mock and store-driven event simulation
type: test
status: draft
lineage: project-queue-view
parent: lifecycle/defects/project-queue-view-7-defect.md
---

# ProjectQueuePanel realtime tests — fix mock plumbing

Fixes the defect described in
[project-queue-view-7-defect.md](../defects/project-queue-view-7-defect.md): all
8 cases in `ProjectQueuePanel.realtime.test.ts` failed with
`TypeError: Cannot read properties of undefined (reading '0')` because
`getAppWs: vi.fn(() => ({ on: vi.fn(...) }))` returned a brand-new mock object
(and a brand-new `on` mock) on every call, so `wsMock.on.mock.calls` inspected
by the test was never the same `on` the code under test had called.

## Root cause (deeper than the defect's literal description)

Making `getAppWs()` return a persistent object was necessary but not
sufficient. `ProjectQueuePanel.vue` itself never calls `getAppWs()` or
`queueStore.fetch()` — in the running app, an ancestor view
(`AgentsRunsView.vue`, `QueueView.vue`) calls `queueStore.fetch()` on mount,
which is what subscribes the **store** to the app WS via `getAppWs().on(...)`.
The test file's old `vi.mock('@/stores/queue', ...)` replaced the store with a
hand-rolled fake whose `fetch` was a no-op mock that never touched
`@/api/ws` — so no handler was ever registered, regardless of the mock
persistence issue.

## Fix

- `@/api/ws`'s `getAppWs()` now returns an object whose `.on` is a single
  `vi.hoisted` mock function (`_wsOnMock`) shared across every call.
- `@/stores/queue` is **no longer mocked**. The suite now exercises the real
  Pinia store so that `queue.*` WS events flow through its actual reducer
  logic into reactive state, exactly as in production. Only the network
  boundary (`@/api/queue`'s `listQueue`, via a hoisted `_listQueueMock`, and
  `@/api/ws`) is mocked.
- `mountProjectQueuePanel()` now calls `useQueueStore().fetch()` after
  mounting, mirroring what `AgentsRunsView.vue`/`QueueView.vue` do on mount,
  so the store actually subscribes and `_wsOnMock.mock.calls[0][0]` yields the
  real registered handler.
- Two assertions (`queue.added`, `queue.started`) asserted on the job's `id`
  (e.g. `'p1'`, `'new-job-1'`) which `ProjectQueuePanel.vue` never renders in
  the DOM (only `artifact_path`/`agent_name`). These were dead assertions that
  the previous mock bug had been masking; rewritten to assert on rendered
  `artifact_path`/`agent_name` text instead.
- The "different project" `queue.started` case was rewritten: the store's
  `queue.started` handler matches purely by job `id` and has no notion of
  project, so a same-project job never actually exercised cross-project
  scoping. The corrected scenario has two pending jobs in different projects
  and fires `queue.started` for the *other* project's job id, asserting the
  current project's job stays pending and no running-row appears (scoping
  happens via the panel's `project`-filtered computed properties).

## Scenarios covered

- `queue.added` for the current project adds a new pending row.
- `queue.added` for a different project is filtered out (no new row).
- `queue.started` for the current project's job moves it into the running
  section.
- `queue.started` for a different project's job leaves the current project's
  pending job untouched and shows no running row.
- `queue.finished` / `queue.cancelled` remove the job from the pending list.
- `queue.paused` shows the paused note; `queue.resumed` hides it.

## Test Files

- `tests/web/ProjectQueuePanel.realtime.test.ts` — all 8 cases pass
  (`pnpm test ProjectQueuePanel.realtime.test.ts` from `tests/web`).

## Out of scope

`tests/web/AgentsRunsView.projectQueue.test.ts` has 7 pre-existing failures
(`TypeError: Cannot read properties of undefined (reading 'length')` on
`store.runs.length`) unrelated to this defect. Confirmed via `git stash` that
these predate this fix; not modified here.
