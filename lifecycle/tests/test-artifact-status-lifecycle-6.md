---
title: "Test Suite: Test Artifact Status Lifecycle"
type: test
status: draft
lineage: test-artifact-status-lifecycle
parent: lifecycle/test-plans/test-artifact-status-lifecycle-5-test.md
---

# Test Suite: Test Artifact Status Lifecycle

Integration tests implementing [[test-artifact-status-lifecycle-5-test]] for the
`approved → in-qa → approved` cyclical lifecycle for test artifacts.

## Files

- `tests/integration/workflow_type_conditional_test.go` — Milestone 1
- `tests/integration/transition_api_type_test.go` — Milestone 2
- `tests/integration/agent_lifecycle_type_test.go` — Milestones 3 & 4
- `tests/integration/defect_traceability_test.go` — Milestone 5
- `tests/integration/crash_recovery_test.go` — Milestone 6
- `tests/integration/test_lifecycle_e2e_test.go` — Milestone 7

## Scenarios Covered

### Milestone 1 — Workflow engine type-conditional rules (10 cases)

Direct calls to `workflow.New(nil).CanTransition(...)` and `AllowedTargets(...)`.
No HTTP or filesystem I/O; compatible with `-short`.

- `approved → in-qa` allowed for `qa` on `test` type (TC1)
- `approved → in-qa` denied for `qa` on `requirement` type — rule is type-restricted (TC2)
- `approved → in-qa` denied for `backend-developer` on `test` — wrong role (TC3)
- `in-qa → approved` allowed for `system` on `test` — post-run reset rule (TC4)
- `in-qa → approved` denied for `system` on `requirement` — type-restricted (TC5)
- `in-qa → approved` for `qa` on `requirement` unchanged — existing rule has no type restriction (TC6)
- `in-development → in-qa` for `backend-developer` unchanged (TC7)
- `AllowedTargets` includes `in-qa` for test artifact + qa (TC8)
- `AllowedTargets` excludes `in-qa` for requirement + qa (TC9)
- `product-owner` bypasses all type restrictions (TC10)

### Milestone 2 — HTTP transition endpoint type-conditional (4 cases)

- `POST .../transition {"to":"in-qa"}` as qa on `type:test` in `approved` → 200 (TC1)
- Same endpoint on `type:requirement` → 403 with `allowed_targets` not containing `in-qa` (TC2)
- `GET .../allowed-targets` as qa on `type:test` → includes `in-qa` (TC3)
- `GET .../allowed-targets` as qa on `type:requirement` → excludes `in-qa` (TC4)

### Milestone 3 — Agent runner lifecycle (5 of 6 cases)

- Pre-run: `approved → in-qa` is synchronous before driver is launched (TC 3.1)
- Pre-run rejection: draft-status test artifact → 409, status unchanged (TC 3.2)
- Post-run success: `in-qa → approved` after exit 0 (TC 3.3)
- Post-run failure: artifact stays `in-qa` after non-zero exit (TC 3.4)
- Non-test artifact uses `DoneOnSuccess` path, not the test reset path (TC 3.5)

TC 3.6 (atomicity on file-write failure) is not implemented: the current
implementation logs a warning and continues if `setArtifactStatus` fails; there
is no rollback guard for file-write errors. A failing test for this scenario
would document the gap.

### Milestone 4 — Concurrent run guard (3 cases)

- Second run rejected when artifact is `in-qa` (status guard, TC 4.1)
- Run allowed after first run completes and artifact returns to `approved` (TC 4.2)
- Different lineage runs concurrently without interference (TC 4.3)

### Milestone 5 — Defect-to-test traceability (3 cases)

Uses a stub `claude` binary that writes a defect with `related_to` pointing to the
test artifact.

- Defect created during QA run has `related_to` = test artifact path (TC 5.1)
- `related_to` is preserved through GET → PUT → GET round-trip (TC 5.2)
- Non-test artifact run does not auto-inject `related_to` into defects (TC 5.3)

### Milestone 6 — Crash recovery (3 of 4 cases)

Recovery fires in `agent.New()` during project open (before HTTP server starts).

- Orphaned `type:test` artifact in `in-qa` with no active run → reset to `approved` (TC 6.1)
- `type:requirement` artifact in `in-qa` is NOT reset (only test artifacts recover) (TC 6.3)
- Multiple orphaned test artifacts all reset to `approved` (TC 6.4)

TC 6.2 ("active run not reset") is not implemented: `index.RecoverRunningRuns()`
runs before orphan recovery and marks all "running" records as "failed", making it
impossible to have a legitimately active run at recovery time in a normal test setup.

### Milestone 7 — End-to-end lifecycle (3 cases)

- Full `approved → in-qa → approved` cycle with re-run eligibility (TC 7.1)
- Full cycle including defect creation with `related_to` and hub event verification (TC 7.2)
- Stale detection data condition: `in-qa` artifact with `mtime > 60min` meets threshold (TC 7.3)
  Note: the `test.stale` WebSocket event requires waiting for the 60-second reaper tick;
  TC 7.3 verifies the index condition only and documents this limitation.

## Open Limitations

- TC 3.6 (atomicity on file-write failure) not implemented — implementation gap.
- TC 6.2 (active run not reset) not implemented — startup ordering prevents setup.
- TC 7.3 (stale WebSocket event) only verifies data condition, not the broadcast.
