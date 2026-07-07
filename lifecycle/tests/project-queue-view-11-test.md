---
title: AgentsRunsView projectQueue tests — complete the agents store mock
type: test
status: draft
lineage: project-queue-view
parent: lifecycle/defects/project-queue-view-8-defect.md
---

# AgentsRunsView projectQueue tests — complete the agents store mock

Fixes the defect described in
[project-queue-view-8-defect.md](../defects/project-queue-view-8-defect.md): all
7 cases in `AgentsRunsView.projectQueue.test.ts` failed with
`TypeError: Cannot read properties of undefined (reading 'length')` on
`store.runs.length` because the inline `vi.mock('@/stores/agents', ...)` only
defined `activeRuns`, `agents`, `fetchAgents`, `fetchRuns` and `killRun` —
missing `runs` and `loading`, both read directly by `AgentsRunsView.vue`'s
template.

## Root cause (deeper than the defect's literal description)

Beyond `runs`/`loading`, `AgentsRunsView.vue` also:
- calls `store.fetchReadyCounts(project)` unconditionally in `onMounted`,
- reads `store.progressLines.get(...)`, `store.permissionEvents.get(...)` and
  `store.runResults.get(...)`/`.has(...)`/`.set(...)` when rendering run rows.

The mock needed all of these to match the real store's shape
(`web/src/stores/agents.ts`), not just the two properties named in the
defect's reproduction, otherwise the same class of crash would resurface as
soon as a test populated `store.runs`.

## Fix

`vi.mock('@/stores/agents', ...)` in `AgentsRunsView.projectQueue.test.ts` now
returns `runs: []`, `loading: false`, `readyCounts: {}`,
`progressLines: new Map()`, `permissionEvents: new Map()`,
`runResults: new Map()`, and a `fetchReadyCounts` mock, alongside the
previously-defined `activeRuns`, `agents`, `fetchAgents`, `fetchRuns` and
`killRun`.

While fixing this, one test — `maintains access to global queue features` —
was still failing after the mock fix, but with a different error: it asserted
`wrapper.text()).toContain('p1')`, where `'p1'` was the pending job's `id`.
`ProjectQueuePanel.vue` never renders a job's `id`, only `artifact_path` and
`agent_name` (the same class of dead assertion documented in
[project-queue-view-9-test.md](project-queue-view-9-test.md) and
[project-queue-view-10-test.md](project-queue-view-10-test.md)). The assertion
was rewritten to check for the job's `artifact_path`
(`'lifecycle/ideas/test.md'`) instead. No changes were made to
`AgentsRunsView.vue` or `ProjectQueuePanel.vue`.

## Scenarios covered

- `AgentsRunsView` renders `ProjectQueuePanel` alongside existing agent rows
  and the run-history table.
- The panel receives the correct `project` route param, including when
  navigating to a different project.
- The panel renders with the expected `.runs-queue-panel` class and "Project
  Queue" header text.
- Regression guard: mounting the view still renders `.runs-main` and the
  queue panel together without crashing.
- Responsive layout classes (`.runs-layout`, `.runs-main`) are present.
- A pending job for the current project is visible in the panel (asserted via
  its `artifact_path`) and the panel's cancel button is present.

## Test Files

- `tests/web/AgentsRunsView.projectQueue.test.ts` — all 7 cases pass
  (`pnpm test AgentsRunsView.projectQueue.test.ts` from `tests/web`).

Full suite verified clean: `pnpm test` from `tests/web` — 103 files, 1536
tests passed.
