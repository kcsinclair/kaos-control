---
title: "Test Artifact Management — Test Plan"
type: plan-test
status: done
lineage: test-artifact-management
parent: lifecycle/requirements/test-artifact-management-2.md
assignees:
    - role: test-developer
      who: agent
---

# Test Artifact Management — Test Plan

Defines integration tests for the Test Artifact Management feature covering the backend API behaviour, frontend UI interactions, and end-to-end workflows. Tests are written as Go integration tests in `tests/integration/` and corresponding test artifacts in `lifecycle/tests/`.

Cross-references: [[test-artifact-management-3-be]] (backend changes under test), [[test-artifact-management-4-fe]] (frontend changes under test).

---

## Milestone 1 — Test artifact filtering via API

### Description
Verify that the artifact listing API correctly filters by `type=test` and that the response meets performance requirements. These tests exercise the backend index and HTTP layer from [[test-artifact-management-3-be]] milestones 1–2.

### Files to change
- `tests/integration/test_artifact_filter_test.go` (new) — integration tests against a running kaos-control instance with a test project containing a mix of artifact types.
- `lifecycle/tests/test-artifact-management-filter.md` (new) — test artifact documenting coverage for this milestone.

### Test cases
1. **Filter by type=test** — `GET /api/p/:project/artifacts?type=test` returns only test artifacts, no other types.
2. **Filter by type=test and status=approved** — `GET /api/p/:project/artifacts?type=test&status=approved` returns only approved test artifacts.
3. **Badge count accuracy** — the `total` field in the filtered response matches the actual number of test artifacts in the project.
4. **Empty project** — filtering by `type=test` on a project with no test artifacts returns an empty list with `total: 0`.
5. **Performance** — the filtered query completes within 200 ms for a project seeded with 500 test artifacts (NF1).

### Acceptance criteria
- [ ] All 5 test cases pass against a running instance.
- [ ] Test fixtures include at least 3 artifact types (test, ticket, idea) to verify filtering exclusion.
- [ ] Performance test uses a seeded project with 500 test artifacts.

---

## Milestone 2 — Agent run for single test artifact

### Description
Verify that invoking the QA agent against a single approved test artifact works correctly via the API, and that the WebSocket events include the expected `target_path` field. Exercises [[test-artifact-management-3-be]] milestones 3–4.

### Files to change
- `tests/integration/test_artifact_run_test.go` (new) — integration tests for single test execution.
- `lifecycle/tests/test-artifact-management-run.md` (new) — test artifact documenting coverage.

### Test cases
1. **Run approved test** — `POST /api/p/:project/agents/qa/run` with `{ target_path: "<approved test>" }` returns 200 and an `agent.started` WS event is received.
2. **Run completion event** — after the agent finishes, an `agent.finished` or `agent.failed` WS event is received with `target_path` matching the request.
3. **Run non-approved test** — attempting to run a test artifact with `status: draft` via the QA agent should either succeed (agent decides) or fail gracefully; verify no crash or orphaned state.
4. **Concurrent run prevention** — if a run is already active for the same lineage, a second `POST .../agents/qa/run` returns an appropriate error (lineage lock).

### Acceptance criteria
- [ ] All 4 test cases pass.
- [ ] WS event payloads are validated for the presence of `target_path`.
- [ ] No orphaned `running` records remain in `agent_runs` after test completion.

---

## Milestone 3 — Serial batch execution

### Description
Verify that multiple test artifacts can be executed serially via repeated API calls, simulating the frontend batch execution flow from [[test-artifact-management-4-fe]] milestone 5.

### Files to change
- `tests/integration/test_artifact_batch_test.go` (new) — integration tests for batch execution.
- `lifecycle/tests/test-artifact-management-batch.md` (new) — test artifact documenting coverage.

### Test cases
1. **Serial execution** — submit 3 approved test artifacts sequentially (wait for `agent.finished` before starting the next). All 3 complete successfully.
2. **Failure does not halt batch** — if one test produces a defect (agent reports failure), the next test can still be started. Verify by checking that `POST .../agents/qa/run` succeeds for the subsequent test after a failed run.
3. **Lock release timing** — after receiving `agent.finished`, immediately calling `POST .../agents/qa/run` for the next test succeeds without lock contention errors.
4. **Agent run records** — after batch completion, `agent_runs` contains one record per test, each with a terminal status.

### Acceptance criteria
- [ ] All 4 test cases pass.
- [ ] Tests use real (or mock) agent runs with actual WS event flow.
- [ ] No timing-dependent flakiness — tests wait for explicit WS events, not sleeps.

---

## Milestone 4 — Kanban board test visibility

### Description
Verify that the Kanban board API response includes test artifacts (so the frontend toggle works) and that the existing artifact listing does not silently exclude them. This is a backend contract test supporting [[test-artifact-management-4-fe]] milestone 6.

### Files to change
- `tests/integration/test_artifact_kanban_test.go` (new) — integration tests for Kanban data.
- `lifecycle/tests/test-artifact-management-kanban.md` (new) — test artifact documenting coverage.

### Test cases
1. **Unfiltered listing includes tests** — `GET /api/p/:project/artifacts` (no type filter) includes `type: test` artifacts in the results.
2. **Kanban config unchanged** — `GET /api/p/:project/config/kanban` response structure is unchanged by this feature (no regressions).
3. **Type filter exclusion** — `GET /api/p/:project/artifacts?type=ticket` does NOT include test artifacts (negative filter check).

### Acceptance criteria
- [ ] All 3 test cases pass.
- [ ] Tests confirm that the backend does not apply any implicit test-artifact exclusion.

---

## Milestone 5 — End-to-end Testing board workflow

### Description
An end-to-end test that exercises the full workflow: navigate to the Testing board, verify cards render, select tests, run a batch, and verify completion. This may be a documented manual test procedure if browser automation is not available, or a scripted API-level simulation.

### Files to change
- `lifecycle/tests/test-artifact-management-e2e.md` (new) — test artifact describing the end-to-end workflow and expected outcomes.

### Test procedure
1. Seed a project with 5 test artifacts: 3 approved, 1 draft, 1 done.
2. Verify `GET /api/p/:project/artifacts?type=test` returns all 5.
3. Verify `GET /api/p/:project/artifacts?type=test&status=approved` returns 3.
4. Execute the 3 approved tests serially via the API, waiting for WS events between each.
5. Verify all 3 `agent_runs` records exist with terminal statuses.
6. Verify the artifact index reflects any status changes made by the QA agent.

### Acceptance criteria
- [ ] The full workflow completes without errors.
- [ ] Test artifact describing this procedure exists at `lifecycle/tests/test-artifact-management-e2e.md`.
- [ ] All acceptance criteria from the requirement (lifecycle/requirements/test-artifact-management-2.md) are traceable to at least one test case across milestones 1–5.
