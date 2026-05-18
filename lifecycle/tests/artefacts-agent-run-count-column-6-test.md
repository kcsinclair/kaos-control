---
title: "Tests: Artefacts Agent Run Count Column"
type: test
status: done
lineage: artefacts-agent-run-count-column
parent: lifecycle/test-plans/artefacts-agent-run-count-column-5-test.md
---

# Tests: Artefacts Agent Run Count Column

Integration and e2e tests covering the "Runs" column, agent status pills, and WebSocket-driven count refresh introduced by the [[artefacts-agent-run-count-column]] feature.

## Scenarios Covered

### Milestone 1 — Index unit tests (`internal/index/index_test.go`)

**`TestAgentRunCountsByTargetPath`**

- Zero-run path is absent from the returned map (caller treats missing key as 0).
- Path with 1 run returns count 1 (running status).
- Path with 1 run returns count 1 (queued status).
- Path with 3 runs across `done`, `failed`, and `killed` statuses returns count 3 — all statuses are included in the tally.
- A single `GROUP BY` query is used (confirmed by code review; no N+1 paths in the implementation).

**`TestActiveAgentStatusByTargetPath`**

- Path with only a `running` run returns `"running"`.
- Path with both `running` and `queued` runs returns `"running"` (priority ordering).
- Path with only a `queued` run returns `"queued"`.
- Path with only completed runs (`done`, `failed`) is absent from the map.

### Milestone 2 — HTTP handler integration test (`tests/integration/agent_run_count_test.go`)

**`TestListArtifacts_AgentRunCount`**

- `GET /api/p/:project/artifacts` response includes `agent_run_count` as an integer on every item.
- Artefact with 3 completed runs returns `agent_run_count: 3`.
- Artefact with 1 active run returns `agent_run_count: 1` and `active_agent_status: "running"`.
- Artefact with 0 runs returns `agent_run_count: 0` — the field is always present, never omitted.
- Artefact with 0 runs omits `active_agent_status` from the JSON response.

### Milestone 4 — E2E: column rendering and sorting (`tests/e2e/flows/10-artefact-run-count-column.spec.ts`)

**TC1: Column presence, position, and counts**

- A column header with text `Runs` exists in the artefacts table.
- `Runs` header appears after `Type` and before `Created` in DOM column order.
- Row for `rc-idea-a.md` (2 seeded runs) displays `2` in the Runs cell.
- Row for `rc-idea-b.md` (1 seeded run) displays `1`.
- Row for `rc-idea-c.md` (0 runs) displays `0` — not blank.

**TC2: Sorting**

- Clicking the `Runs` header once sorts all rows in non-decreasing order.
- Clicking again sorts all rows in non-increasing order.

### Milestone 5 — E2E: active-agent status pill (`tests/e2e/flows/10-artefact-run-count-column.spec.ts`)

**TC3: Running pill lifecycle**

- An `agent-status-pill[data-status="running"]` element with text "Agent Running" appears in the row while a stub-agent run is in progress.
- The pill disappears after the run completes (confirmed via Playwright auto-retry).

### Milestone 6 — E2E: WebSocket-driven count refresh (`tests/e2e/flows/10-artefact-run-count-column.spec.ts`)

**TC4: Live count increment**

- Triggering an agent run increments the displayed run count for the target artefact after the `agent.finished` WebSocket event is received.
- No full page navigation occurs (URL is unchanged before and after the increment).

## Test Files

| File | Type | Command |
|------|------|---------|
| `internal/index/index_test.go` | Go unit | `go test ./internal/index/ -run TestAgentRunCountsByTargetPath\|TestActiveAgentStatusByTargetPath` |
| `tests/integration/agent_run_count_test.go` | Go integration | `go test -tags integration ./tests/integration/ -run TestListArtifacts_AgentRunCount` |
| `tests/e2e/flows/10-artefact-run-count-column.spec.ts` | Playwright e2e | `pnpm exec playwright test flows/10-artefact-run-count-column.spec.ts` |

## E2E Fixture Artifacts

Five new fixture ideas added to `tests/e2e/fixtures/lifecycle/ideas/`:

- `rc-idea-a.md` — seeded with 2 agent runs in TC1
- `rc-idea-b.md` — seeded with 1 agent run in TC1
- `rc-idea-c.md` — 0 runs (used to verify `0` is shown, not blank)
- `rc-pill.md` — target for pill appearance/disappearance test (TC3)
- `rc-ws.md` — target for WebSocket count-refresh test (TC4)
