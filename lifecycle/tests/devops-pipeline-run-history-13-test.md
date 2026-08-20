---
title: "Tests — DevOpsView.test.ts fix for stale .run-status--failed assertion"
type: test
status: draft
lineage: devops-pipeline-run-history
parent: lifecycle/defects/devopsview-test-stale-run-status-failed-class.md
created: 2026-08-20T14:35:00Z
---

# Tests — DevOpsView.test.ts fix for stale `.run-status--failed` assertion

Fixes the defect in
[devopsview-test-stale-run-status-failed-class](../defects/devopsview-test-stale-run-status-failed-class.md):
`PipelineCard.vue` no longer renders a static `.run-status--failed` badge from
`activeRun.overallStatus`; failed runs are now surfaced only via the
historical `.latest-run-badge--failed` element, driven by
`devops.latestRunForPipeline()` (backed by `pipelineHistory`, populated by
`RunHistory.vue`'s `listPipelineRuns` call on mount).

## Test file

**`tests/web/DevOpsView.test.ts`**

- Added a `listPipelineRuns` mock to the existing `@/api/devops` module mock
  (`mockListPipelineRuns`, default `{ runs: [] }`) so that `RunHistory.vue`,
  mounted inside every `PipelineCard`, has something to call on `onMounted`
  without throwing. The default keeps all pre-existing tests in
  `describe('PipelineCard — run status styling', …)` and elsewhere unaffected
  (no history → no `.latest-run-badge`).
- Rewrote `shows Failed run-status badge when run has failed` →
  `shows the .latest-run-badge--failed badge when the latest historical run
  failed`: mounts `PipelineCard` with no active run and
  `mockListPipelineRuns` resolving one `RunHistoryRow` with
  `status: 'failed'`, then asserts:
  - `.latest-run-badge--failed` exists (the current rendering path).
  - `.run-status--failed` does not exist (guards against regressing back to
    the removed static badge).

This mirrors the equivalent, already-passing coverage in
`tests/web/RunHistory.spec.ts` (`describe('PipelineCard.vue (F7 — latest-run
summary badge)')`), which exercises the same `.latest-run-badge--failed` /
`--passed` behaviour via both direct store pre-population and a mocked
`listPipelineRuns` response.

## Verification

```
cd tests/web && pnpm exec vitest run DevOpsView.test.ts RunHistory.spec.ts
```

20 tests pass in `DevOpsView.test.ts`, 10 in `RunHistory.spec.ts` (30 total).
An unrelated pre-existing failure in `AppSidebar.test.ts` (nav-link count
mismatch) is out of scope for this defect and was not touched.
