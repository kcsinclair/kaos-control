---
title: "Test Coverage — Rename Analyst Agents to Phase-First Convention"
type: test
status: approved
lineage: rename-analyst-agents
parent: lifecycle/test-plans/rename-analyst-agents-5-test.md
---

# Test Coverage — Rename Analyst Agents

Covers the rename of `analyst-requirements` → `requirements-analyst` and `analyst-planner` → `planning-analyst` across all integration and frontend test files.

## Scenarios covered

### Integration test helper config (`tests/integration/agent_helpers_test.go`)

- Embedded YAML config updated to declare `requirements-analyst` (active_status=clarifying) and `planning-analyst` (active_status=planning) agents.
- Agent git identities updated: email addresses use new names (`requirements-analyst@test.local`, `planning-analyst@test.local`).
- Helper comment at `newAgentTestEnv` updated to list new names.

### Integration test cases

**`tests/integration/agent_status_test.go`**
- `TestAnalystRequirementsActivatesStatus` — verifies `requirements-analyst` synchronously sets artifact status to `clarifying`.
- `TestAnalystPlannerActivatesStatus` — verifies `planning-analyst` synchronously sets artifact status to `planning`.
- `TestAnalystStatusPersistsAfterSuccess` — verifies `requirements-analyst` run retains `clarifying` status after exit 0.
- `TestAnalystStatusPersistsAfterFailure` — verifies `requirements-analyst` run retains `clarifying` status after non-zero exit.

**`tests/integration/agent_ws_test.go`**
- `TestAnalystRunBroadcastsStatusChange` — verifies WebSocket `agent.started` event fires with correct run_id and lineage when `requirements-analyst` starts.
- `TestAgentWSEvents_IncludeTargetPath` — verifies both `agent.started` and terminal events carry `target_path` for `requirements-analyst` runs.

**`tests/integration/agents_api_test.go`**
- `TestStartAgentRun_Success` — POST to `requirements-analyst/run` returns 202 with run_id.
- `TestStartAgentRun_BadRequest` — POST with malformed JSON to `requirements-analyst/run` returns 400.
- `TestListAgentRunsByTargetPath_ReturnsMatchingRuns` — seeded runs use `requirements-analyst` agent name.
- `TestListAgentRunsByTargetPath_OrderNewestFirst` — seeded runs use `requirements-analyst` agent name.
- `TestListAgentRunsByTargetPath_NoParam_ReturnsAll` — seeded runs use `requirements-analyst` agent name.

### Frontend test fixtures (`tests/web/helpers/agent_launch_fixtures.ts`)

- `AGENT_INPUT_TYPE_MAP` updated: `'requirements-analyst': 'idea'` and `'planning-analyst': 'requirement'`.

### Frontend test cases

**`tests/web/ArtifactRunHistory.test.ts`**
- Default `makeRun()` fixture uses `requirements-analyst` as `agent_name`.
- Run list rendering test asserts agent name column shows `requirements-analyst`.

**`tests/web/AgentsRunsView.sort.test.ts`**
- Agent sort fixture includes `requirements-analyst` as one of three agents.
- Ascending/descending sort assertions reference `requirements-analyst` as alphabetically first agent.
- Started-at and Elapsed sort assertions reference `requirements-analyst` as the chronologically/numerically first entry.

**`tests/web/agent-launch/defect-inclusion.test.ts`**
- `requirements-analyst does not see defects` — verifies approved defects assigned to analyst role are excluded from `requirements-analyst` results.

**`tests/web/agent-launch/input-status-filter.test.ts`**
- `requirements-analyst input filtering` — sees only approved ideas from mixed-status set.
- `planning-analyst input filtering` — sees only approved requirements from mixed-status set.
- `predecessorMap is not used` — `requirements-analyst` with active_status=draft still sees only approved artifacts.

**`tests/web/agent-launch/empty-state.test.ts`**
- `requirements-analyst sees empty list when all ideas are draft`.
- `requirements-analyst sees empty list when only approved requirements exist`.
- `requirements-analyst list becomes non-empty when an approved idea is added` (reactive test).

## Test files

- `tests/integration/agent_helpers_test.go`
- `tests/integration/agent_status_test.go`
- `tests/integration/agent_ws_test.go`
- `tests/integration/agents_api_test.go`
- `tests/web/helpers/agent_launch_fixtures.ts`
- `tests/web/ArtifactRunHistory.test.ts`
- `tests/web/AgentsRunsView.sort.test.ts`
- `tests/web/agent-launch/defect-inclusion.test.ts`
- `tests/web/agent-launch/input-status-filter.test.ts`
- `tests/web/agent-launch/empty-state.test.ts`
