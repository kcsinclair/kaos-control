---
title: "Test Plan: Status Check Advance Flow"
type: plan-test
status: in-development
lineage: status-checker-button-no-status-update
parent: lifecycle/defects/status-checker-button-no-status-update.md
---

# Test Plan: Status Check Advance Flow

This plan covers integration tests that verify the status-check GET and advance POST endpoints return the correct response shapes and actually mutate artifact statuses on disk.

## Milestone 1: Test Children Field Shape in Status Check Response

**Description:** Verify that `GET /api/p/{project}/status-check` returns `children` as an array of objects with `path` and `status` fields (not bare strings).

**Files to change:**
- `tests/integration/status_check_test.go`

**Changes:**
1. Add a test case that creates a lineage with a parent artifact at `in-development` and a child at `done`.
2. Call the status-check endpoint and assert that the response `stale[0].children` is an array of objects, each with non-empty `path` and `status` string fields.
3. Assert `stale[0].children[0].status` equals `"done"`.

**Acceptance criteria:**
- Test passes against the updated backend (after backend plan Milestone 1).
- Test fails if `children` reverts to `[]string`.

## Milestone 2: Test Advance Endpoint Response Contract

**Description:** Verify that `POST /api/p/{project}/status-check/advance` returns each result with `ok: bool` and `advanced_to: string` fields, and that the artifact's on-disk status actually changes.

**Files to change:**
- `tests/integration/status_check_test.go`

**Changes:**
1. Add a test case that sets up a stale artifact (parent at `in-development`, child at `done`), then calls the advance endpoint with the parent's path.
2. Assert the response contains `ok: true` and `advanced_to` matching the expected target status.
3. Re-read the artifact file from disk and assert its frontmatter `status` field has been updated.
4. Add a negative test: attempt to advance an artifact where the user lacks the required role. Assert `ok: false` and a non-empty `error`/`reason` field.

**Acceptance criteria:**
- Advance test passes and confirms disk mutation.
- Permission-denied test confirms the artifact is NOT mutated and the response indicates failure.

## Milestone 3: Test Staleness Detection Edge Cases

**Description:** The defect scenario involves an idea and requirement stuck at `in-development` while downstream plans/tests are `done`. Verify the staleness algorithm handles multi-level lineages correctly.

**Files to change:**
- `tests/integration/status_check_test.go` or `internal/statuscheck/statuscheck_test.go`

**Changes:**
1. Test a 3-level lineage: idea (`in-development`) → requirement (`done`) → plan (`done`). The idea should be detected as stale because its direct child (requirement) is ahead.
2. Test a lineage where one child is `done` and another is `in-development` (same parent). The parent should NOT be detected as stale since not ALL children are ahead.
3. Test a lineage where a child is in a terminal status (`rejected`). The terminal child should be excluded from the "all ahead" check.

**Acceptance criteria:**
- Multi-level staleness is detected correctly (parent reports stale when direct children are all ahead).
- Mixed-progress siblings correctly prevent false-positive staleness detection.
- Terminal children are excluded from the comparison.

## Milestone 4: E2E Test for Status Check Button User Flow

**Description:** Verify the full flow from clicking the status check button through to artifact status update, as described in the defect's reproduction steps.

**Files to change:**
- `tests/integration/status_check_e2e_test.go`

**Changes:**
1. Programmatically create a project with a stale lineage.
2. Call the GET status-check endpoint; assert at least one stale result with `can_advance: true`.
3. Call the POST advance endpoint with the stale paths.
4. Call the GET status-check endpoint again; assert the previously-stale artifact is no longer in the results (or has a new `current_status`).
5. Read the artifact from disk to confirm the frontmatter was written.

**Acceptance criteria:**
- Full round-trip test passes: detect stale → advance → verify updated.
- Test is deterministic (no timing dependencies on fsnotify or WebSocket).
- `go test ./tests/integration/... -run StatusCheck` passes.

## Cross-references

- [[status-checker-button-no-status-update]] (backend plan): Tests depend on Milestones 1–2 for correct response shapes.
- [[status-checker-button-no-status-update]] (frontend plan): E2E test validates the contract the frontend relies on.
