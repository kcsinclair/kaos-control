---
title: "Test Plan: Test Artifact Status Lifecycle"
type: plan-test
status: approved
lineage: test-artifact-status-lifecycle
parent: requirements/test-artifact-status-lifecycle-2.md
---

# Test Plan: Test Artifact Status Lifecycle

This plan defines integration tests for the [[test-artifact-status-lifecycle]] feature — verifying the cyclical `approved → in-qa → approved` lifecycle for test artifacts, including workflow rules, agent runner behaviour, concurrency guards, defect traceability, and crash recovery.

## Milestone 1: Workflow Engine Type-Conditional Transition Tests

### Description

Test that the workflow engine correctly enforces type-conditional transition rules added in the [[test-artifact-status-lifecycle]] backend plan (Milestone 1).

### Files to change

- `tests/workflow_test.go` (new or extend existing) — Add test cases for the type-aware `CanTransition` and `AllowedTargets` functions.

### Test cases

1. **`approved → in-qa` allowed for `qa` role on `type: test`** — Call `CanTransition("approved", "in-qa", ["qa"], "test")`, assert `true`.
2. **`approved → in-qa` denied for `qa` role on `type: requirement`** — Call `CanTransition("approved", "in-qa", ["qa"], "requirement")`, assert `false`.
3. **`approved → in-qa` denied for non-`qa` role on `type: test`** — Call `CanTransition("approved", "in-qa", ["backend-developer"], "test")`, assert `false`.
4. **`in-qa → approved` allowed for `system` role on `type: test`** — Call `CanTransition("in-qa", "approved", ["system"], "test")`, assert `true`.
5. **`in-qa → approved` denied for `system` role on `type: requirement`** — Call `CanTransition("in-qa", "approved", ["system"], "requirement")`, assert `false`.
6. **Existing `in-qa → approved` for `qa` role on non-test types unchanged** — Call `CanTransition("in-qa", "approved", ["qa"], "requirement")`, assert `true`.
7. **Existing `in-development → in-qa` unchanged** — Call `CanTransition("in-development", "in-qa", ["backend-developer"], "requirement")`, assert `true`.
8. **`AllowedTargets` includes `in-qa` for test artifacts in `approved` with `qa` role** — Call `AllowedTargets("approved", ["qa"], "test")`, assert `in-qa` is in the result.
9. **`AllowedTargets` excludes `in-qa` for non-test artifacts in `approved` with `qa` role** — Call `AllowedTargets("approved", ["qa"], "requirement")`, assert `in-qa` is NOT in the result.
10. **Product-owner bypasses type restrictions** — Call `CanTransition("approved", "in-qa", ["product-owner"], "requirement")`, assert `true` (product-owner is superuser).

### Acceptance criteria

- All 10 test cases pass.
- Tests run with `go test ./... -short` (unit-level, no I/O).
- No existing workflow tests break.

## Milestone 2: HTTP Transition Endpoint Tests for Type-Conditional Rules

### Description

Test that the HTTP transition endpoint correctly passes artifact type to the workflow engine, per [[test-artifact-status-lifecycle]] backend plan Milestone 2.

### Files to change

- `tests/transition_api_test.go` (new or extend existing) — Integration tests against the HTTP API.

### Test cases

1. **Transition test artifact `approved → in-qa` via API with `qa` role** — `POST /api/p/:project/artifacts/tests/foo-6.md/transition` with `{"to": "in-qa"}`, assert 200.
2. **Transition non-test artifact `approved → in-qa` via API with `qa` role rejected** — Same endpoint for a `type: requirement` artifact, assert 403 with `allowed_targets` in response.
3. **`allowed-targets` includes `in-qa` for test artifact in `approved` status** — `GET /api/p/:project/artifacts/tests/foo-6.md/allowed-targets` with `qa` role, assert `in-qa` in `targets`.
4. **`allowed-targets` excludes `in-qa` for non-test artifact** — Same endpoint for a requirement artifact, assert `in-qa` not in `targets`.

### Acceptance criteria

- All 4 test cases pass.
- Tests set up a temporary project directory with fixture artifacts.
- Tests clean up after themselves (no leftover state).

## Milestone 3: Agent Runner Lifecycle Tests

### Description

Test the agent runner's pre-run and post-run status transitions for test artifacts, per [[test-artifact-status-lifecycle]] backend plan Milestone 3.

### Files to change

- `tests/agent_lifecycle_test.go` (new) — Integration tests that exercise the agent runner with a mock/stub driver.

### Test cases

1. **Pre-run transition: test artifact `approved → in-qa`** — Start a QA agent run against a `type: test` artifact in `approved` status. Assert the artifact status is `in-qa` before the agent process is invoked.
2. **Pre-run rejection: test artifact not in `approved`** — Start a QA agent run against a `type: test` artifact in `draft` status. Assert the run is rejected with an error and the artifact status is unchanged.
3. **Post-run success: test artifact `in-qa → approved`** — Complete a QA agent run with exit code 0. Assert the test artifact returns to `approved` status.
4. **Post-run failure: test artifact stays in `in-qa`** — Complete a QA agent run with non-zero exit code. Assert the test artifact remains in `in-qa` status.
5. **Non-test artifact uses existing behaviour** — Start an agent run against a non-test artifact. Assert the existing `ActiveStatus`/`DoneOnSuccess` behaviour applies (status set to `done` on success, not `approved`).
6. **Atomicity: failed pre-run transition does not start agent** — Simulate a file-write failure during the `approved → in-qa` transition. Assert the agent process is never started and the artifact status is unchanged.

### Acceptance criteria

- All 6 test cases pass.
- Tests use a stub driver that can simulate success and failure exits.
- Each test starts from a clean state with a temporary project directory.

## Milestone 4: Concurrent Run Guard Tests

### Description

Test that a second QA run against a test artifact already in `in-qa` is rejected, per [[test-artifact-status-lifecycle]] backend plan Milestone 4.

### Files to change

- `tests/agent_lifecycle_test.go` (extend) — Add concurrency test cases.

### Test cases

1. **Second run rejected when test artifact in `in-qa`** — Start a QA agent run (artifact transitions to `in-qa`). Without completing the first run, attempt a second QA run against the same artifact. Assert the second run is rejected with a descriptive error.
2. **Run allowed after first run completes** — Complete the first QA run (artifact returns to `approved`). Start a new QA run. Assert it succeeds.
3. **Different lineage not affected** — Start a QA run on lineage A. Start a QA run on lineage B. Assert both succeed concurrently.

### Acceptance criteria

- All 3 test cases pass.
- Tests verify both the status-check guard and the lineage lock guard.

## Milestone 5: Defect-to-Test Traceability Tests

### Description

Test that defect artifacts raised during a QA run against a test artifact include a `related_to` link, per [[test-artifact-status-lifecycle]] backend plan Milestone 5.

### Files to change

- `tests/defect_traceability_test.go` (new) — Integration tests that verify defect frontmatter.

### Test cases

1. **Defect includes `related_to` for test artifact** — Run a QA agent against `tests/foo-6.md` that creates a defect. Parse the defect's frontmatter and assert `related_to` contains `tests/foo-6.md`.
2. **Defect `related_to` preserved through round-trip** — Read the defect artifact, write it back via the API, read again. Assert `related_to` is unchanged.
3. **Defect against non-test artifact has no `related_to` injection** — Run a QA agent against a requirement artifact. Assert defects do not have an auto-injected `related_to` field (only whatever the agent decides to include).

### Acceptance criteria

- All 3 test cases pass.
- Tests parse actual YAML frontmatter from disk files.

## Milestone 6: Crash Recovery Tests

### Description

Test that orphaned `in-qa` test artifacts are detected and reset on startup, per [[test-artifact-status-lifecycle]] backend plan Milestone 6.

### Files to change

- `tests/crash_recovery_test.go` (new) — Tests that simulate crash scenarios.

### Test cases

1. **Orphaned test artifact reset on startup** — Create a `type: test` artifact with `status: in-qa` and no corresponding running agent run in the index. Call `RecoverOrphanedTests`. Assert the artifact status is reset to `approved` and a warning is logged.
2. **Active run not reset** — Create a `type: test` artifact with `status: in-qa` AND a corresponding agent run with `status: running`. Call `RecoverOrphanedTests`. Assert the artifact status remains `in-qa`.
3. **Non-test artifact in `in-qa` not reset** — Create a `type: requirement` artifact with `status: in-qa` and no running agent run. Call `RecoverOrphanedTests`. Assert the artifact status remains `in-qa` (only test artifacts are recovered).
4. **Multiple orphans recovered** — Create 3 orphaned test artifacts. Call `RecoverOrphanedTests`. Assert all 3 are reset to `approved`.

### Acceptance criteria

- All 4 test cases pass.
- Tests use a temporary project directory with a fresh SQLite index.
- Warning log messages are verified (captured via test logger).

## Milestone 7: End-to-End Lifecycle Cycle Test

### Description

A full integration test that exercises the complete `approved → in-qa → approved` cycle, including defect creation and re-run eligibility. This validates the entire [[test-artifact-status-lifecycle]] feature end-to-end.

### Files to change

- `tests/test_lifecycle_e2e_test.go` (new) — End-to-end test.

### Test cases

1. **Full cycle: approved → in-qa → approved** — Create a test artifact in `approved` status. Start a QA agent run. Assert status transitions to `in-qa`. Complete the run successfully. Assert status returns to `approved`. Start another QA run. Assert it succeeds (re-run eligibility).
2. **Full cycle with defect** — Create a test artifact in `approved` status. Start a QA agent run that raises a defect. Assert the defect has `related_to` pointing to the test artifact. Assert the test artifact returns to `approved` after the run completes.
3. **Stale detection** — Create a test artifact, transition to `in-qa`, and advance the clock (or set mtime) to > 60 minutes ago. Trigger the stale-check routine. Assert a `test.stale` WebSocket event is broadcast with the correct artifact path.

### Acceptance criteria

- All 3 test cases pass.
- Tests exercise the real HTTP API and agent runner (with a stub driver).
- WebSocket events are captured and verified.
- No manual cleanup is required between test runs.
