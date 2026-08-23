---
title: Revisit test artifacts and revise the test workflow
type: idea
status: draft
lineage: test-artifact-workflow-revision
created: "2026-08-23T09:00:00+10:00"
priority: normal
release: KC-Release6
labels:
    - idea
    - testing
    - workflow
    - lifecycle
---

# Revisit test artifacts and revise the test workflow

The current test model conflates two different things, and the lifecycle
treats a `type: test` artifact as a flow work-item when — after verification —
it is really standing reference. Revise the workflow so the model matches
reality.

## The two layers (today)

1. **Test code** — e.g. `tests/integration/architecture_wizard_scaffold_test.go`.
   Real, tagged (`//go:build integration`), committed. `make test-integration`
   is `go test ./... -tags=integration`, which **globs** it, so it runs on every
   pipeline test run and via the test-runner agent — **permanently, automatically.**
2. **Test artifact** — e.g. [[wizard-skip-scaffolding]]'s
   `lifecycle/tests/…-6-test.md`. A non-executable `type: test` record that
   *describes* what the code covers, carrying a lifecycle status.

The rough edge: the artifact flows `draft → … → in-qa → done` like a work item,
but its "run" is a **one-time QA gate** for that dev cycle (verify the new tests
pass, then close it). The *code* is what actually re-runs forever. New users
reasonably ask "do I need to run this again?" — the answer ("no, the code is in
the suite; the artifact is verified once") isn't expressed anywhere in the model.

## What to revise

- **Treat a verified test artifact as standing reference, not flow.** Once
  `done`, it is coverage documentation — the same conclusion reached for
  `feature` artifacts and the architecture zone. Consider hiding done test
  artifacts from the flow board by default (a reference zone), or otherwise
  signalling "this is a record, its code runs in the suite."
- **Regression traceability (the real gap).** Today the only test-staleness
  signal is `test.stale`, which fires only when an artifact *lingers in `in-qa`
  > 60 min* (`internal/lock/lock.go`, `staleTestThreshold`) — a "you forgot to
  close this" nudge, **not** drift detection. There is:
    - no "the code under test changed → re-verify this test artifact" signal;
    - no link from a pipeline/test-runner **failure** back to the test artifact
      that owns the failing test, so a regression does not re-open the right
      record.
  Wire failing test functions (from the test-runner run summary) back to the
  covering test artifact so a regression re-activates it and its defect links to
  it.
- **Clarify the "verify once vs. runs continuously" distinction** in the UI /
  docs so the QA gate isn't mistaken for ongoing execution.

## Explicitly out of scope

- **Do NOT** add per-artifact re-running of tests. The pipeline already runs the
  whole suite continuously; duplicating that per artifact would be wasteful. The
  goal is to model the *record* correctly and connect *failures* back to records
  — not to re-execute code the suite already covers.

## Considerations to resolve

- Should a done test artifact move to a dedicated reference zone (like
  `lifecycle/features/` / the architecture zone), or stay in `lifecycle/tests/`
  with board-visibility rules?
- Failure→artifact linkage: match by test-function name, by file path, or by an
  explicit `covers:`/`tests_file:` frontmatter field on the test artifact?
- Does a covered feature changing warrant auto-reopening the test artifact to
  `in-qa`, or just a non-blocking "may be stale" flag?

Captured for **KC-Release6**; to be fixed next release.
