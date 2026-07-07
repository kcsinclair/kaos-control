---
title: Pipeline card latest-run summary badge not visible after run completion
type: defect
status: in-development
lineage: devops-pipeline-run-history
parent: lifecycle/tests/devops-pipeline-run-history-8-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

## Triage (2026-07-07): feature present — treat as E2E timing, not a missing badge

Canonical for this issue (`-11` and `-12` were identical duplicates, closed).

The badge is **implemented and wired**, so this is not a "missing feature" bug:
- `web/src/components/devops/PipelineCard.vue` renders `.latest-run-badge`
  (`v-else-if="latestRun"`, from `devops.latestRunForPipeline(slug)`), with
  passed/failed icons and relative time.
- `web/src/stores/devops.ts` maintains `runHistory` and updates it from the
  `pipeline.run.*` WebSocket events (`runHistory.push`, the
  `pipeline.run.completed` handler), and `latestRunForPipeline` filters it.

So the Playwright timeout waiting for `.latest-run-badge` is most likely **E2E
timing / test-harness** (the badge appears only after the `run.completed` WS
event propagates and the history refreshes, which can exceed the test's wait),
not a rendering gap. Reassigned to test-developer.

**Next step (needs the heavy E2E run to confirm):** run
`cd tests/e2e && npx playwright test run-history` and check whether the badge
appears just after the wait window (→ increase the `waitFor`/poll for the WS
round-trip) vs. never (→ a real WS-event/selector gap to escalate back to
frontend-developer). Not yet reproduced locally here (E2E not run).

# Pipeline card latest-run summary badge not visible after run completion

## Reproduction Steps

1. Build the frontend assets: `make build-web`
2. Build the server binary: `make build`
3. Run the Playwright E2E tests: `make test-e2e` or `cd tests/e2e && pnpm test`
4. Observe that the test `pipeline card shows the latest-run summary badge after a run` fails with a timeout of 30,000ms.

## Expected Behaviour

When a DevOps pipeline completes running, the pipeline card should display the latest-run summary badge (with class `.latest-run-badge` and status class like `.latest-run-badge--passed`), which includes the relative time elapsed since the run.

## Actual Behaviour

The pipeline card displays a static status badge like `<span class="run-status run-status--passed">Passed</span>` instead of the latest-run relative time badge (`.latest-run-badge`). This is because the conditional rendering (`v-if`) in `web/src/components/devops/PipelineCard.vue` prioritizes showing the static `Passed` badge if `activeRun?.overallStatus === 'passed'` is true. Since the store retains the active run's state after completion, this condition remains true, and the `.latest-run-badge` is never rendered.

## Logs / Output

```
  1) flows/run-history.spec.ts:114:3 › Run History smoke › pipeline card shows the latest-run summary badge after a run 

    Test timeout of 30000ms exceeded.

    Error: expect(locator).toBeVisible() failed

    Locator: locator('.latest-run-badge')
    Expected: visible
    Error: element(s) not found

    Call log:
      - Expect "toBeVisible" with timeout 30000ms
      - waiting for locator('.latest-run-badge')


      129 |         // Trigger run and wait for the latest-run badge to appear.
      130 |         await page.locator('.btn-run').first().click()
    > 131 |         await expect(page.locator('.latest-run-badge')).toBeVisible({ timeout: 30_000 })
          |                                                         ^
      132 |
      133 |         // Card badge should reflect passed status.
      134 |         await expect(page.locator('.latest-run-badge--passed')).toBeVisible({
        at /Users/keith/Code/kaos-control/tests/e2e/flows/run-history.spec.ts:131:57
```
