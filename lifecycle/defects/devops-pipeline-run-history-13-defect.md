---
title: Pipeline card latest-run summary badge still not visible after run completion (recurrence of -10)
type: defect
status: draft
lineage: devops-pipeline-run-history
parent: lifecycle/tests/devops-pipeline-run-history-8-test.md
labels:
    - defect
assignees:
    - role: frontend-developer
      who: agent
---

# Pipeline card latest-run summary badge still not visible after run completion (recurrence of -10)

## Reproduction Steps

1. Build the frontend assets: `make build-web`
2. Build the server binary: `make build`
3. Run the Playwright E2E tests: `make test-e2e`, or in isolation:
   `cd tests/e2e && npx playwright test flows/run-history.spec.ts -g "latest-run summary badge"`
4. Observe test `flows/run-history.spec.ts:114:3 › Run History smoke › pipeline card shows the latest-run summary badge after a run` times out waiting for `.latest-run-badge`.

Reproduced twice in a row (not a one-off flake): once as part of the full `make test-e2e` run and once when re-run in isolation.

## Expected Behaviour

After a pipeline run completes, the pipeline card renders `.latest-run-badge` (with `.latest-run-badge--passed`/`--failed` modifier) per `devops-pipeline-run-history-10-defect.md`'s agreed design.

## Actual Behaviour

`devops-pipeline-run-history-10-defect.md` is marked `status: done` — `web/src/components/devops/PipelineCard.vue` (commit `4ce82a68`) was updated so the `v-else-if="latestRun"` branch is reachable (the terminal `passed`/`failed`/`cancelled` static badges were removed from the chain; confirmed present in the current tree at `PipelineCard.vue` lines ~102-113). Despite that fix being in place, the E2E test still times out waiting for `.latest-run-badge` — this is a fresh regression or an unresolved race, not the original dead-code bug.

Suspected root cause (from code reading, not yet confirmed with a debug run): a race in `web/src/stores/devops.ts` between two writers of `pipelineHistory`:
- `RunHistory.vue`'s `onMounted` calls `devops.fetchPipelineHistory(project, slug)` (`devops.ts:105-116`), which unconditionally does `pipelineHistory.value.set(slug, res.runs ?? [])` once the REST GET resolves.
- `handleRunCompleted` (`devops.ts:276-311`, triggered by the `pipeline.run.completed` WS event) also unconditionally does `pipelineHistory.value.set(slug, deduped.slice(0, 50))`.

`latestRunForPipeline(slug)` (`devops.ts:101-103`) just reads `pipelineHistory.value.get(slug)?.[0]`. If the mount-time GET (issued when the card first renders, before any run has occurred) resolves *after* `handleRunCompleted` has already populated the slug's entry — plausible if the GET is still in flight when a fast/stub pipeline run starts and completes — it clobbers the just-written run with an empty/stale list, and `.latest-run-badge` never renders (or renders and then vanishes).

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

## Fix guidance

Verify the suspected race (e.g. add a temporary log/breakpoint around both `pipelineHistory.value.set` call sites, or run with network throttling on the `GET .../runs` request). If confirmed, make the two writers order-safe — e.g. have `fetchPipelineHistory` merge/de-dupe by `run_id` instead of blind-overwriting, or have it no-op if a WS-driven update for that slug happened more recently. If the race isn't the cause, re-open the investigation from `devops-pipeline-run-history-10-defect.md`.
