---
title: Pipeline card latest-run summary badge not visible after run completion
type: defect
status: abandoned
lineage: devops-pipeline-run-history
parent: lifecycle/tests/devops-pipeline-run-history-9-test.md
labels: [defect]
assignees:
  - role: frontend-developer
    who: agent
---

## Closed — duplicate (2026-07-07)

Duplicate of `devops-pipeline-run-history-10` (identical title/issue: pipeline
card latest-run badge not visible after run completion). Tracking under -10.

# Pipeline card latest-run summary badge not visible after run completion

## Reproduction Steps

1. Build the server binary: `make build`
2. Run the Playwright E2E smoke tests: `cd tests/e2e && pnpm install && npx playwright test run-history`
3. Observe that the test `pipeline card shows the latest-run summary badge after a run` fails due to a timeout waiting for the `.latest-run-badge` selector.

## Expected Behaviour

Upon completion of a pipeline run, the `.latest-run-badge` element should be rendered on the pipeline card, displaying the relative time and status of the latest run (e.g. using class `.latest-run-badge--passed`).

## Actual Behaviour

The pipeline card retains the completed run in `activeRuns` (which has `overallStatus = 'passed'`), and the conditional rendering logic in `web/src/components/devops/PipelineCard.vue` displays a static `.run-status--passed` badge (`Passed`) instead of the `.latest-run-badge`. Because `.run-status--passed` is matched in the `v-if/v-else-if` block first, the `latestRun` badge is not rendered when the completed active run is still in the store.

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
