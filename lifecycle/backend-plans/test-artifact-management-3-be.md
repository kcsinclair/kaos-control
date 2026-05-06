---
title: "Test Artifact Management — Backend Plan"
type: plan-backend
status: draft
lineage: test-artifact-management
parent: lifecycle/requirements/test-artifact-management-2.md
assignees:
    - role: backend-developer
      who: agent
---

# Test Artifact Management — Backend Plan

The requirement states F6: "No New Endpoints Required" — the existing `POST /api/p/:project/agents/:name/run` endpoint and artifact listing API are sufficient. This plan covers the minor backend adjustments needed to support the frontend features: an optimised query path for test-type artifacts, the Kanban visibility default, and ensuring the agent runner correctly handles serial invocations from the frontend.

Cross-references: [[test-artifact-management-4-fe]] (frontend consumes the filtered API and WS events), [[test-artifact-management-5-test]] (integration tests).

---

## Milestone 1 — Verify type-filtered artifact listing performance

### Description
The frontend Testing board will call `GET /api/p/:project/artifacts?type=test`. The `buildWhere` function in `internal/index/index.go` already supports a `Type` field on the `Filter` struct, and the `artifacts` table has a `type` column. Verify that this column is indexed and that queries filtering by `type = 'test'` meet the NF1 requirement (< 200 ms for up to 500 test artifacts).

### Files to change
- `internal/index/index.go` — confirm or add `CREATE INDEX IF NOT EXISTS idx_artifacts_type ON artifacts(type)` to the schema init block.

### Acceptance criteria
- [ ] The `artifacts` table has an index on the `type` column.
- [ ] `GET /api/p/:project/artifacts?type=test` returns results within 200 ms for a project with 500 test artifacts (verified by test or benchmark).
- [ ] No changes to the `Filter` struct or `buildWhere` are needed (existing `type` filter works correctly).

---

## Milestone 2 — Add badge count endpoint for test artifacts

### Description
The left navigation "Testing" menu item needs a badge count of tests in `approved` status. Rather than forcing the frontend to fetch all test artifacts just for a count, add a lightweight count endpoint (or extend the existing listing endpoint to support a `count_only` query parameter). This avoids transferring full artifact payloads for a simple number.

### Files to change
- `internal/index/index.go` — add a `Count(filter Filter) (int, error)` method that runs `SELECT COUNT(*) FROM artifacts` with the same `buildWhere` logic but skips pagination and row scanning.
- `internal/http/artifacts.go` — if the `count_only=true` query parameter is present on `GET /api/p/:project/artifacts`, return `{ "count": N }` instead of the full list. Alternatively, the existing `total` field in the list response is already sufficient — verify this.

### Acceptance criteria
- [ ] The frontend can obtain the count of `type=test, status=approved` artifacts without fetching full artifact data.
- [ ] The count is consistent with the filtered list result's `total` field.
- [ ] Response time for the count query is under 50 ms for 500 artifacts.

---

## Milestone 3 — Ensure agent runner supports rapid serial invocations

### Description
Batch test execution calls `POST /api/p/:project/agents/qa/run` serially — the frontend waits for one run to complete before starting the next. Verify that the agent runner correctly handles this pattern: that the lineage lock is released on completion, that the semaphore slot is freed, and that the `agent_runs` row is updated before the `agent.finished` WebSocket event is broadcast (so the frontend can safely start the next run upon receiving the event).

### Files to change
- `internal/agent/agent.go` — audit the `supervise` goroutine to confirm that: (1) the `agent_runs` status is updated to a terminal state before `hub.Broadcast` of `agent.finished`/`agent.failed`; (2) the lineage lock is released before the broadcast; (3) the semaphore is released. Fix ordering if needed.

### Acceptance criteria
- [ ] After `agent.finished` is broadcast, a subsequent `POST .../agents/qa/run` for a different test artifact succeeds immediately (no lock/semaphore contention from the prior run).
- [ ] The `agent_runs` row has a terminal status (`finished` or `failed`) before the WS event is sent.
- [ ] No race condition exists between lock release and the next `StartRun` call.

---

## Milestone 4 — WebSocket event payload for agent runs includes target path

### Description
The frontend needs to correlate `agent.finished` / `agent.failed` events with specific test artifacts during batch execution. Verify that the existing WebSocket event payloads include the `target_path` that was passed to the agent run. If not, add it to the payload so the frontend can match completions to queued tests.

### Files to change
- `internal/agent/agent.go` — in the `supervise` goroutine, ensure the `agent.finished` and `agent.failed` event payloads include the `TargetPath` field from the run record.

### Acceptance criteria
- [ ] `agent.finished` and `agent.failed` WS event payloads contain a `target_path` field matching the value passed in the original `POST .../agents/qa/run` request.
- [ ] Existing frontend code that consumes these events is not broken by the additional field.

---

## Milestone 5 — Default Kanban filter excludes test artifacts

### Description
The requirement specifies that the Kanban board's "Show Tests" toggle defaults to unchecked, hiding test artifacts. The backend Kanban config endpoint (`GET /api/p/:project/config/kanban`) can include a default filter hint, or this can be handled purely on the frontend. Since the Kanban board already does client-side filtering, no backend change is strictly required here — but document the expectation that the frontend filters `type != 'test'` by default.

### Files to change
- No backend code changes required. This milestone exists to confirm the decision and avoid unnecessary backend work.

### Acceptance criteria
- [ ] Confirmed: Kanban column config does not need a backend change; test-artifact hiding is a frontend-only concern.
- [ ] The existing `GET /api/p/:project/artifacts` endpoint returns test artifacts in its results (no server-side exclusion) so that toggling "Show Tests" on the frontend works without a new API call.
