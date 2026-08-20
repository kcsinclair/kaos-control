---
created: "2026-08-15T11:32:40+10:00"
title: Pipeline card latest-run summary badge not visible after run completion
type: defect
status: done
lineage: devops-pipeline-run-history
parent: lifecycle/tests/devops-pipeline-run-history-8-test.md
labels:
    - defect
release: KC-Release5
assignees:
    - role: frontend-developer
      who: agent
---

## Reopened (2026-07-08, test-runner)

`make test-e2e` still reproduces this exact failure today —
`flows/run-history.spec.ts:114:3 › Run History smoke › pipeline card shows
the latest-run summary badge after a run` times out on the same
`.latest-run-badge` locator, unchanged since the 2026-07-07 analysis below.
Status was left as `done` after the initial product-owner triage even
though the follow-up test-developer analysis (below) refuted that triage
and left three blocking questions unanswered — reopening to `blocked` so
it surfaces for product-owner instead of filing a duplicate. No new
defect file created; see root-cause analysis already captured here.

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

## Open Questions (test-developer, 2026-07-07)

Re-triaged this before implementing/adjusting the E2E test. Static analysis
of the current code contradicts the 2026-07-07 triage's conclusion —
raising this back rather than writing a test that papers over it.

**The triage claims this is E2E timing, not a rendering gap. It is not
timing — the code makes `.latest-run-badge` permanently unreachable once a
pipeline has run at least once, no matter how long the test waits:**

- `web/src/stores/devops.ts`: `activeRuns` (a `Map<slug, ActiveRun>`) is
  populated in `runPipeline`/`handleRunStarted` and mutated in place by
  `handleRunCompleted` (`run.overallStatus = finalStatus`, line ~284). There
  is **no code path anywhere that deletes or resets an `activeRuns` entry**
  (`activeRuns.value.delete` does not appear in the file, confirmed by grep
  across `web/src`). So after the first run, the map entry for that
  pipeline slug permanently holds a terminal `overallStatus` (`'passed'` /
  `'failed'` / `'cancelled'`).
- `web/src/components/devops/PipelineCard.vue` (lines 102–117) renders the
  meta badges as a `v-if`/`v-else-if` chain: `passed` → `failed` →
  `cancelled` → `isActive` (running) → `latestRun` (`.latest-run-badge`).
  Since `activeRun?.overallStatus` never reverts to a non-terminal/unset
  value, one of the first three branches always wins after any run has
  completed, and the `v-else-if="latestRun"` branch — the one that renders
  `.latest-run-badge` — becomes dead code.

Increasing the Playwright `waitFor`/timeout (the triage's proposed next
step) cannot fix `flows/run-history.spec.ts:131`: the locator isn't slow to
appear, it can never appear once `activeRun.overallStatus` is set, which
happens deterministically on the first `pipeline.run.completed` WS event
for that pipeline in this same test run.

Blocking questions for product-owner:

1. Is the intended UX that the **static** `run-status--passed/failed`
   badge is what should persist on the card after a run (i.e. the E2E test
   at `run-history.spec.ts:114-144` is wrong to assert on
   `.latest-run-badge`, and it should assert on `.run-status--passed`
   instead)? Or is `.latest-run-badge` (with relative time) supposed to
   supersede the static badge once a run has completed?

> The latest-run-badge should win. It's the run-history feature's deliverable (status color/icon + "5m ago"), and the static "Passed/Failed" text badge is redundant with it. So the test is correct; the code is wrong. No change to run-history.spec.ts.

2. If `.latest-run-badge` should supersede: where should the transition
   happen — should `PipelineCard.vue`'s `v-else-if` chain drop the
   terminal-status branches once `latestRun` is available (i.e. only show
   `passed`/`failed`/`cancelled` while `isActive` is meaningfully "just
   finished"), or should `devops.ts` clear/expire the `activeRuns` entry
   for a slug some time after `handleRunCompleted` fires?

> In PipelineCard.vue (view layer), not by clearing activeRuns. Clearing the store entry on completion would also nuke the step list (showSteps = activeRun != null, line 42) and the card coloring (lines 92-93) the instant a run finishes — a real regression. So keep activeRuns as the "last-run detail" holder, and fix the badge chain: show "Running" while active, otherwise the latest-run-badge; drop the persistent static terminal text badges. That makes .latest-run-badge reachable and keeps the steps. One consequence: DevOpsView.test.ts:324 must switch its assertion from .run-status--failed to the latest-run-badge (or the card's failed class).

3. This fix lands in `web/src/stores/devops.ts` and/or
   `PipelineCard.vue`, which is outside test-developer's
   `allowed_write_paths`. Should this defect be reassigned to
   frontend-developer for the code fix, with test-developer only updating
   `tests/e2e/flows/run-history.spec.ts` once the intended behaviour (Q1)
   is confirmed?

> Yes — frontend-developer. The entire fix is PipelineCard.vue plus that one component-test update (both frontend paths). The existing E2E test is the verification — test-developer has nothing to change.

No test code changes made pending answers — the correct assertion for
`run-history.spec.ts:131` depends entirely on which behaviour is intended.
