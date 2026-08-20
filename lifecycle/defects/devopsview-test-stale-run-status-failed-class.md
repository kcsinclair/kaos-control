---
title: DevOpsView.test.ts asserts removed .run-status--failed class instead of .latest-run-badge--failed
type: defect
status: approved
lineage: devops-pipeline-run-history
created: "2026-08-20T12:25:00+10:00"
parent: lifecycle/tests/devops-pipeline-run-history-9-test.md
labels:
    - defect
release: KC-Release5
assignees:
    - role: test-developer
      who: agent
---

# DevOpsView.test.ts asserts removed .run-status--failed class instead of .latest-run-badge--failed

## Reproduction Steps

1. `cd tests/web && pnpm test` (or `pnpm exec vitest run DevOpsView.test.ts`)
2. Observe `PipelineCard — run status styling > shows Failed run-status badge when run has failed` fail.

## Expected Behaviour

`tests/web/DevOpsView.test.ts` asserts against `PipelineCard.vue`'s current rendering: a failed run is surfaced via the historical `.latest-run-badge--failed` element (driven by `latestRun` from the devops store), not a static `activeRun.overallStatus`-driven badge.

## Actual Behaviour

Commit `4ce82a68` ("fix(devops-pipeline-run-history): make .latest-run-badge reachable on PipelineCard") removed `PipelineCard.vue`'s static terminal-status badges (previously rendered from `activeRun?.overallStatus === 'failed'` as `<span class="run-status run-status--failed">`) because they permanently shadowed the newer `.latest-run-badge` element. Failed status is now only rendered as `.latest-run-badge--failed` (`PipelineCard.vue:107`), sourced from historical run data (`latestRun`), not `activeRun.overallStatus`. The test at `tests/web/DevOpsView.test.ts:318-324` still sets `activeRun.overallStatus = 'failed'` and asserts `wrapper.find('.run-status--failed').exists()` — a class that no longer exists anywhere in the component.

## Logs / Output

```
 FAIL  DevOpsView.test.ts > PipelineCard — run status styling > shows Failed run-status badge when run has failed
AssertionError: expected false to be true // Object.is equality
 ❯ DevOpsView.test.ts:324:58
    322|       steps: buildPipeline.steps.map((s) => ({ name: s.name, status: '…
    323|     })
    324|     expect(wrapper.find('.run-status--failed').exists()).toBe(true)
```

## Fix guidance

Update the test to mock historical run data so `latestRun.status === 'failed'` (matching how the passing-status sibling tests / `run-history.spec.ts` e2e flow already exercise `.latest-run-badge--passed`), and assert `.latest-run-badge--failed` instead of the removed `.run-status--failed`.
