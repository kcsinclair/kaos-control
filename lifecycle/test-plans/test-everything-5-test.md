---
title: 'Test Plan: Run All Tests and Auto-file Defects'
type: plan-test
status: approved
lineage: test-everything
parent: lifecycle/requirements/test-everything-2.md
---

## Overview

Integration and unit tests covering the test-everything feature: test output parsing for Go/Vitest/Playwright, artifact mapping, deduplication, defect creation with correct lineage and routing, the `test-runner` agent orchestrator, the `test-all.yaml` DevOps pipeline, and frontend UI elements (agent launcher, run summary, auto-filed badge). Tests validate all acceptance criteria from the requirement.

Related: [[test-everything]]

## Milestone 1 — Parser Unit Tests

### Description

Unit tests for each test output parser, covering valid output, edge cases, and malformed input (NF4).

### Files to change

- `internal/testrunner/parse_go_test.go` (new) — Test cases:
  - Valid `go test -json` output with mixed pass/fail/skip results is parsed correctly.
  - Failed test extracts correct package, test name, output lines, and elapsed time.
  - Skipped tests are counted but do not appear in failures.
  - Subtests (`TestFoo/subcase`) are parsed with full name.
  - Compilation error (non-JSON output) sets `SuiteResult.RawError` and returns zero failures.
  - Empty input returns zero totals.
  - Interleaved output from parallel tests is correctly attributed.
- `internal/testrunner/parse_vitest_test.go` (new) — Test cases:
  - Valid Vitest JSON with passing and failing assertion results.
  - Failure extracts `ancestorTitles + title` as full test name, `failureMessages`, and `location.line`.
  - Skipped tests are counted correctly.
  - Empty `testResults` array returns zero totals.
  - Malformed JSON sets `RawError`.
- `internal/testrunner/parse_playwright_test.go` (new) — Test cases:
  - Valid Playwright JSON with nested suites/specs/tests/results.
  - Failure extracts title, error message, and location.
  - Skipped/expected-failure tests counted correctly.
  - Empty suites array returns zero totals.
  - Malformed JSON sets `RawError`.
- `tests/fixtures/testrunner/` (new directory) — Fixture files:
  - `go_mixed_results.json` — Go test output with passes, failures, and skips.
  - `go_compile_error.txt` — Non-JSON compilation error output.
  - `vitest_results.json` — Vitest output with mixed results.
  - `playwright_results.json` — Playwright output with nested failures.

### Acceptance criteria

- [ ] Each parser has tests for success, failure, skip, and malformed input scenarios.
- [ ] Fixture files provide realistic test framework output.
- [ ] `go test ./internal/testrunner/ -short -run TestParse` passes.
- [ ] Edge cases (empty input, parallel tests, subtests) are covered.

## Milestone 2 — Artifact Mapping Tests

### Description

Unit tests for the `ArtifactMapper` covering all three lookup tiers (filename, label, lineage) and orphan detection.

### Files to change

- `internal/testrunner/mapper_test.go` (new) — Test cases:
  - **Filename match**: a failure in `artifact_store_test.go` maps to a test artifact with `source_file: artifact_store_test.go`.
  - **Filename slug match**: a failure in `project_crud_test.go` maps to `projects-crud-*-test.md` by slug derivation.
  - **Label match**: a failure in `internal/http/` maps to a test artifact labelled `http` or `backend`.
  - **Lineage match**: a failure maps when test artifact lineage matches the derived slug.
  - **No match**: a failure with no corresponding test artifact returns `nil`.
  - **Priority**: filename match takes precedence over label match.
  - **Orphan detection**: `DetectOrphans` returns test files that have no corresponding `lifecycle/tests/*.md` artifact.
  - **Orphan detection**: all tests with matching artifacts return an empty orphan list.
- `tests/fixtures/testrunner/test_artifacts/` (new directory) — Minimal test artifact fixtures for mapping tests.

### Acceptance criteria

- [ ] All three lookup tiers are individually tested.
- [ ] Lookup priority (filename → label → lineage) is verified.
- [ ] Orphan detection correctly identifies unmatched tests.
- [ ] `go test ./internal/testrunner/ -short -run TestMap` passes.

## Milestone 3 — Deduplication Tests

### Description

Unit tests for the deduplication engine covering duplicate detection by test identifier, assertion location, and error similarity, plus assertion grouping.

### Files to change

- `internal/testrunner/dedup_test.go` (new) — Test cases:
  - **Same test identifier**: existing open defect with matching package + test name is found.
  - **Same assertion location**: existing open defect at same file:line is found.
  - **Similar error message**: errors differing only in timestamp/pointer/UUID are matched.
  - **Dissimilar error**: errors with different first 100 characters are not matched.
  - **Closed defects ignored**: `done` and `abandoned` defects are not returned as duplicates.
  - **Assertion grouping**: 5 failures at the same file:line produce 1 group with 5 entries.
  - **Distinct locations**: failures at different file:line produce separate groups.
  - **`NormaliseError`**: strips `2026-05-15`, `0x1a2b3c`, UUID patterns; truncates to 100 chars.
  - **`NormaliseError` idempotent**: normalising an already-normalised string returns the same result.

### Acceptance criteria

- [ ] Duplicate detection works for all three matching criteria.
- [ ] Closed defects are excluded from duplicate matches.
- [ ] Assertion grouping produces correct cluster sizes.
- [ ] Error normalisation handles timestamps, pointers, UUIDs, and truncation.
- [ ] `go test ./internal/testrunner/ -short -run TestDedup` passes.

## Milestone 4 — Defect Filing Integration Tests

### Description

Integration tests that verify defect artifacts are created with correct frontmatter, body structure, lineage, role routing, and witness entries.

### Files to change

- `tests/testrunner_defect_test.go` (new) — Test cases:
  - Filing a Go test failure creates a defect in `lifecycle/defects/` with correct frontmatter fields (`type: defect`, `status: draft`, `lineage`, `parent`, `labels` including `auto-filed`, `assignees`).
  - Defect body includes test name, file:line, error message, suite name, and `## Reproduction` section.
  - Filing a failure matched to a test artifact with `backend` label routes to `backend-developer`.
  - Filing a failure from `tests/web/` routes to `frontend-developer`.
  - Filing an orphaned failure uses `tests-orphaned` lineage and notes the missing artifact.
  - The `tests-orphaned` idea artifact is auto-created if absent.
  - Grouped failures (same file:line) produce one defect with all witnesses listed.
  - Appending a witness to an existing defect adds the entry without corrupting frontmatter.
  - Lineage index increments correctly and does not reuse indices.
  - Defect `release` field is inherited from the matched test artifact.
- `tests/testrunner_defect_test.go` — Cleanup: tests remove created defect files after completion.

### Acceptance criteria

- [ ] Defect frontmatter matches F6 specification exactly.
- [ ] Role routing matches F7 rules for backend, frontend, and ambiguous cases.
- [ ] Orphaned failures use `tests-orphaned` lineage with auto-created idea artifact.
- [ ] Witness append does not break YAML frontmatter parsing.
- [ ] Lineage indices are monotonically increasing.
- [ ] Tests clean up created artifacts.

## Milestone 5 — Agent Orchestrator Integration Tests

### Description

End-to-end integration tests for the full `test-runner` agent flow: execution → parsing → mapping → deduplication → defect filing → summary.

### Files to change

- `tests/testrunner_agent_test.go` (new) — Test cases:
  - **Full flow**: invoke the `test-runner` orchestrator against a project with known test failures; verify defects are created and summary is correct.
  - **Idempotency (NF2)**: run the orchestrator twice; second run finds duplicates and creates no new defects.
  - **Suite-level failure (NF4)**: a suite with compilation errors produces a single suite-level defect and does not block other suites.
  - **Performance (NF1)**: parsing + mapping + dedup + filing overhead is under 10 seconds (measured separately from actual test execution).
  - **Skip on no changes**: when scheduled and no source files changed, the orchestrator skips execution.
  - **Coverage gaps**: the summary lists tests with no corresponding `lifecycle/tests/*.md` artifact.
  - **Mixed suites**: Go failures and Vitest failures in the same run produce correctly routed defects.
- `tests/fixtures/testrunner/project/` (new directory) — Minimal project fixture with Go test files and Vitest test files that produce known failures.

### Acceptance criteria

- [ ] Full flow creates the expected defects with correct content.
- [ ] Second run produces zero new defects (idempotency).
- [ ] Suite-level failures produce one defect and do not halt the run.
- [ ] Overhead is under 10 seconds.
- [ ] Coverage gaps are reported in the summary.
- [ ] Tests use fixture projects, not the live `kaos-control` test suites.

## Milestone 6 — DevOps Pipeline Integration Test

### Description

Integration test for the `test-all.yaml` pipeline, verifying it is discovered, can be triggered, and reflects the correct outcome.

### Files to change

- `tests/devops_test_all_test.go` (new) — Test cases:
  - `test-all.yaml` is returned by the pipeline discovery endpoint.
  - The pipeline has type `test` and one step.
  - Triggering the pipeline via `POST .../run` returns a `run_id`.
  - Pipeline status reflects the outcome of the test-runner agent (pass/fail).
  - Pipeline appears in run history after completion.

### Acceptance criteria

- [ ] Pipeline is discovered and has the correct metadata.
- [ ] Pipeline can be triggered and returns a run ID.
- [ ] Pipeline run history shows the completed run.
- [ ] Pipeline status correctly reflects test outcomes.

## Milestone 7 — Frontend UI Tests

### Description

Vitest component tests for the frontend changes: target-less agent launcher, run summary display, and auto-filed badge.

### Files to change

- `tests/web/agent-launcher-targetless.test.ts` (new) — Test cases:
  - Agent with empty `source_types` hides the target picker.
  - Informational message is displayed for target-less agents.
  - "Run" button is enabled without a target selection.
  - Agent with non-empty `source_types` still shows the target picker.
- `tests/web/agent-run-summary.test.ts` (new) — Test cases:
  - Run summary for `test-runner` renders per-suite statistics table.
  - Defect/duplicate/orphan counts are displayed.
  - Coverage gaps section is collapsible and lists gap items.
  - Duration is formatted correctly.
- `tests/web/auto-filed-badge.test.ts` (new) — Test cases:
  - Defect with `auto-filed` label shows bot icon badge.
  - Badge has correct tooltip text.
  - Defect without `auto-filed` label has no badge.
  - Non-defect artifact with `auto-filed` label has no badge.

### Acceptance criteria

- [ ] All component tests pass with `pnpm test`.
- [ ] Target-less agent launcher behaviour is verified for both target-less and target-requiring agents.
- [ ] Run summary rendering is tested with realistic mock data.
- [ ] Auto-filed badge visibility logic is fully covered.
