---
title: End-to-End Testing Board Workflow — Procedure
type: test
status: approved
lineage: test-artifact-management
parent: lifecycle/test-plans/test-artifact-management-5-test.md
release: KC-Release1
---

# End-to-End Testing Board Workflow — Procedure

Documents the **Milestone 5** end-to-end test procedure for the test-artifact-management feature. Because browser automation is not available in this project's test harness, this milestone is expressed as an API-level simulation that exercises the full workflow from seeding through batch completion.

The procedure is implemented as a Go integration test at `tests/integration/test_artifact_e2e_test.go` (`TestTestArtifactE2E_FullWorkflow`) and can be run with `go test -tags=integration ./tests/integration/ -run TestTestArtifactE2E`.

## Pre-conditions

- A running `kaos-control` instance with a project registered.
- `qa@test.local` (or equivalent user with the `qa` role) credentials available.
- The `qa` agent is configured with `driver: claude-code-cli` and `active_status: in-qa`.

## Test procedure

### Step 1 — Seed the project

Create 5 test artifacts in `lifecycle/tests/`:

| File | status |
|---|---|
| `lifecycle/tests/e2e-test-a.md` | approved |
| `lifecycle/tests/e2e-test-b.md` | approved |
| `lifecycle/tests/e2e-test-c.md` | approved |
| `lifecycle/tests/e2e-test-d.md` | draft |
| `lifecycle/tests/e2e-test-e.md` | done |

### Step 2 — Verify unfiltered listing

```
GET /api/p/:project/artifacts?type=test
```

Expected: `total=5`, `items` contains all 5 artifacts.

### Step 3 — Verify approved filter

```
GET /api/p/:project/artifacts?type=test&status=approved
```

Expected: `total=3`, all items have `status=approved`.

### Step 4 — Execute the 3 approved tests serially

For each approved test artifact (e2e-test-a, e2e-test-b, e2e-test-c):

1. `POST /api/p/:project/agents/qa/run` with `{ "target_path": "<path>" }` → expect 202 + `run_id`.
2. Subscribe to the WebSocket hub and wait for `agent.finished` or `agent.failed` carrying the matching `run_id`.
3. Proceed to the next test only after the terminal event is received.

### Step 5 — Verify agent run records

For each `run_id` obtained in step 4:

```
GET /api/p/:project/agents/runs/:run_id
```

Expected: `run.status` is `done` or `failed` (never `running`).

### Step 6 — Verify index reflects status changes

```
GET /api/p/:project/artifacts?type=test&status=in-qa
```

Expected: the 3 artifacts that were run now have `status=in-qa` (set by the QA agent's `active_status`), confirming the index was updated.

## Implementation notes

The Go integration test (`TestTestArtifactE2E_FullWorkflow`) uses a failing stub (`exit 1`) for the `claude` binary so that artifacts remain in `in-qa` after each run. The post-run reset to `approved` only fires on exit-0 (successful) completion; for failed runs the `active_status` (`in-qa`) set at run-start persists, which is what step 6 verifies. This is consistent with the production behaviour where a QA failure leaves the artifact under investigation.

## Acceptance criteria

- [x] Step 2 returns `total=5`.
- [x] Step 3 returns `total=3`.
- [x] All 3 `POST /agents/qa/run` calls in step 4 return 202.
- [x] Terminal WS events are received for all 3 runs before the next run starts.
- [x] Step 5 shows terminal status for all 3 run records.
- [x] Step 6 shows `total=3` for `type=test&status=in-qa` after the runs.

## Traceability

This procedure covers the requirement acceptance criteria from `lifecycle/requirements/test-artifact-management-2.md`:

- F2 (Testing Board View): Steps 2–3 verify the filtering API the board depends on.
- F4 (Single Test Execution): Step 4 verifies the `POST /agents/qa/run` endpoint and WS event flow.
- F5 (Multi-select and Batch Execution): Step 4 verifies serial execution with event-driven sequencing.
- F6 (Backend — No New Endpoints Required): All steps use existing API endpoints without additions.
