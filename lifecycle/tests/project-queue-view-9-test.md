---
created: "2026-07-14T19:34:44+10:00"
title: ProjectQueuePanel unit tests — fix ID-based assertions to use artifact_path
type: test
status: draft
lineage: project-queue-view
parent: lifecycle/defects/project-queue-view-6-defect.md
---

# ProjectQueuePanel unit tests — fix ID-based assertions

Fixes the defect described in
[project-queue-view-6-defect.md](../defects/project-queue-view-6-defect.md): four
`ProjectQueuePanel.test.ts` cases asserted on the internal `QueueJob.id` (e.g.
`p1`, `p3`, `run-1`), which `ProjectQueuePanel.vue` never renders. All jobs
built via the `makeJob` helper defaulted to the same `artifact_path`
(`lifecycle/ideas/test.md`), so once the component rendered `artifact_path`
instead of `id`, the tests could no longer distinguish which job was actually
present in the DOM.

## Fix

Each affected test now overrides `artifact_path` with a unique value per job
and asserts on that rendered path instead of the job `id`. No changes were
made to `ProjectQueuePanel.vue` — the component's behaviour was already
correct.

## Scenarios covered (updated)

- `shows only jobs belonging to the current project` — three jobs across two
  other projects and the current project, each given a distinct
  `artifact_path`; asserts only the current project's artifact path appears.
- `shows running section when job belongs to project` — running job given a
  unique `artifact_path`; asserts it and the agent name are rendered.
- `shows pending jobs in position order` — three pending jobs with distinct
  positions and `artifact_path` values; asserts `.pending-row` elements
  render in position order by checking each row's artifact path.
- `filters out jobs from other projects` — jobs across two projects, each
  with a unique `artifact_path`; asserts only current-project paths appear.

All other pre-existing cases in the file were unaffected and continue to
pass.

## Test Files

- `tests/web/ProjectQueuePanel.test.ts` — all 12 cases pass
  (`pnpm test ProjectQueuePanel.test.ts` from `tests/web`).

## Out of scope

`tests/web/ProjectQueuePanel.realtime.test.ts` has 8 pre-existing failures
unrelated to this defect (mock `wsMock.on.mock.calls[0]` is undefined,
suggesting the component's WS subscription wiring changed independently of
this issue). Confirmed via `git stash` that these failures predate this fix
and are outside the defect's reproduction steps; not modified here.
