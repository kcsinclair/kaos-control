---
title: Test pipeline does not run the web/ component suite (web/src/**/__tests__)
type: defect
status: draft
lineage: pipeline-missing-web-component-suite
created: "2026-08-24T11:00:00+10:00"
priority: high
labels:
    - defect
    - ci
    - testing
    - frontend
release: KC-Release6
assignees:
    - role: devops
      who: agent
---

# Test pipeline does not run the web/ component suite (web/src/**/__tests__)

## Reproduction Steps

1. Inspect `lifecycle/devops/all-tests.yaml` (and the test-runner agent's suite list).
2. Note the frontend step is `cd tests/web && pnpm test`.
3. Observe there is no step running `cd web && pnpm test`.

## Expected Behaviour

Every frontend unit/component suite runs in the pipeline and via the test-runner
agent, so a "green" run means all frontend tests pass.

## Actual Behaviour

The pipeline runs only `tests/web` (~1548 tests). The **`web/` component suite**
(`web/src/**/__tests__`, ~193 tests — e.g. `ScaffoldStep.spec.ts`,
`RunHistory.spec.ts`) is **never run by CI or the test-runner**. During the
KC-Release5 release prep the test-runner reported a clean run, yet the `web/`
suite had 3 failing `ScaffoldStep.spec.ts` tests (stale after the
wizard-skip-scaffolding Milestone 2 UI change). They were only found by running
`cd web && pnpm test` manually. This is a release-gate blind spot: the pipeline
can report green while frontend tests are red.

## Fix guidance

Add a **"Frontend component tests"** step to `lifecycle/devops/all-tests.yaml`
running `cd web && pnpm test`, and include it in the test-runner agent's suite
set so failures are surfaced (and auto-filed) like the others. Consider a single
root script that runs both `web` and `tests/web` so neither can be forgotten.

## Notes

Surfaced during KC-Release5 release prep. Deferred to KC-Release6; the 3
ScaffoldStep failures themselves were fixed on kc-dev so the release is not
blocked, but the pipeline gap remains.
