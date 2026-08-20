---
created: "2026-07-14T19:34:44+10:00"
title: Pipeline card latest-run summary badge not visible after run completion
type: defect
status: abandoned
lineage: devops-pipeline-run-history
parent: lifecycle/tests/devops-pipeline-run-history-7-test.md
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

1. Build the frontend assets and server binary:
   ```bash
   make build-web && make build
   ```
2. Navigate to the `tests/e2e` directory and install dependencies if necessary:
   ```bash
   cd tests/e2e && pnpm install
   ```
3. Run the Playwright run-history flow tests:
   ```bash
   pnpm playwright test flows/run-history.spec.ts
   ```
4. Observe that the test `pipeline card shows the latest-run summary badge after a run` fails due to a timeout waiting for the `.latest-run-badge` element.

## Expected Behaviour

When a pipeline run completes, the pipeline card should render the `.latest-run-badge` class element, displaying the relative time since the run finished and indicating the run's success or failure (e.g. using class `.latest-run-badge--passed`).

## Actual Behaviour

The pipeline card retains the completed run state, but instead of displaying the latest-run summary badge (`.latest-run-badge`), the conditional rendering (`v-if` / `v-else-if`) in `web/src/components/devops/PipelineCard.vue` evaluates `activeRun?.overallStatus === 'passed'` as true. Consequently, it shows the static `<span class="run-status run-status--passed">Passed</span>` badge and skips the `v-else-if="latestRun"` block entirely.

## Logs / Output

```
  3) flows/run-history.spec.ts:114:3 › Run History smoke › pipeline card shows the latest-run summary badge after a run 

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
