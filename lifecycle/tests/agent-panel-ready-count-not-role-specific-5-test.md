---
title: "Test: Per-Agent Role-Specific Ready Counts"
type: test
status: approved
lineage: agent-panel-ready-count-not-role-specific
parent: lifecycle/test-plans/agent-panel-ready-count-not-role-specific-4-test.md
---

# Test: Per-Agent Role-Specific Ready Counts

Verifies that the `GET /api/p/:project/agents/ready-counts` endpoint returns
counts scoped to each agent's `source_types` configuration, and that the
`internal/index` `Count()` function correctly honours both `Status` and `Type`
predicates when called with a combined filter.

## Scenarios Covered

### Milestone 1 — Backend Integration: `tests/integration/ready_counts_test.go`

Tests start a full HTTP server using a custom `readyCountsCfgYAML` config that
defines three agents, each with distinct `source_types`:

| Agent | `active_status` | `source_types` |
|---|---|---|
| `requirements-analyst` | `clarifying` | `[idea]` |
| `backend-developer` | `in-development` | `[plan-backend]` |
| `frontend-developer` | `in-development` | `[plan-frontend]` |

| Test | Description |
|---|---|
| `TestReadyCounts_PerAgentSourceTypes` | Seeds 1 clarifying idea, 1 in-dev plan-backend, 2 in-dev plan-frontend (plus out-of-status noise). Asserts requirements-analyst=1, backend-developer=1, frontend-developer=2, and backend-developer ≠ frontend-developer. |
| `TestReadyCounts_SourceTypesExcludesWrongType` | Seeds only plan-frontend in-development; asserts backend-developer=0, frontend-developer=2. |
| `TestReadyCounts_SourceTypesExcludesWrongStatus` | Seeds plan-backend in draft/planning status; asserts backend-developer=0. |
| `TestReadyCounts_RequirementsAnalystCountsOnlyIdeas` | Seeds 1 clarifying idea and 1 clarifying ticket; asserts requirements-analyst=1 (ticket not counted). |
| `TestReadyCounts_ResponseShape` | Asserts HTTP 200, body shape `{"counts":{...}}`, all values numeric, all agents with active_status present. |

Run with:
```
go test -tags integration ./tests/integration/... -run TestReadyCounts
```

### Milestone 2 — Index Unit Test: `internal/index/index_test.go`

Tests call `idx.Count(Filter{...})` directly against an in-memory SQLite index
seeded with four artifacts covering all combinations of type × status.

| Test | Description |
|---|---|
| `TestCountWithTypeFilter` | Table-driven: verifies 8 filter combinations including status+type, status-only, type-only, CSV type, and no-filter cases. |
| `TestCountWithTypeFilter_InDevelopmentNoTypeIsAllTypes` | Confirms that omitting Type returns all in-development artifacts across all types (no implicit restriction). |
| `TestCountWithTypeFilter_MultipleTypesCSV` | Confirms comma-separated Type value triggers `IN (...)` behaviour and matches multiple types. |

Run with:
```
go test ./internal/index/... -run TestCountWithTypeFilter
```

### Milestone 3 — Frontend Component Test (Manual)

Vitest and `@vue/test-utils` are not installed in `web/package.json`; no `test`
script exists. The component test described in the test plan cannot be run
automatically at this time.

**Manual test procedure for `AgentPanelRow.vue`:**

1. Start the development server: `make run`
2. Navigate to a project's Agents screen.
3. Ensure at least two agents have `active_status` configured with different
   numbers of matching artifacts (e.g. seed 3 `plan-backend/in-development` and
   1 `plan-frontend/in-development`).
4. Verify that the ready-count badge on the backend-developer panel shows **3**
   and the frontend-developer panel shows **1**.
5. Confirm the badges are not the same value.
6. Click a badge and confirm the artifact list page opens filtered to the
   correct `status` and `type` for that agent.

To enable automated tests in future, install:
```
pnpm add -D vitest @vue/test-utils @vitejs/plugin-vue jsdom
```
Then add `"test": "vitest"` to `web/package.json` scripts and create
`web/src/components/agent/__tests__/AgentPanelRow.spec.ts`.

### Milestone 4 — E2E Smoke Tests: `tests/integration/agents_ready_counts_smoke_test.go`

API-level smoke tests skipped in `-short` mode.

| Test | Description |
|---|---|
| `TestAgentsReadyCounts_SmokeDistinctCounts` | Seeds 1 plan-backend and 3 plan-frontend in-development; asserts at least two agents have different counts. |
| `TestAgentsReadyCounts_SmokeBackendVsFrontend` | Seeds 2 plan-backend and 1 plan-frontend in-development; asserts backend-developer=2, frontend-developer=1, and they differ. |

Run with:
```
go test -tags integration ./tests/integration/... -run TestAgentsReadyCounts_Smoke
```

## Test Files

- `tests/integration/ready_counts_test.go` — Milestones 1 integration tests
- `tests/integration/agents_ready_counts_smoke_test.go` — Milestone 4 smoke tests
- `internal/index/index_test.go` — Milestone 2 unit tests
