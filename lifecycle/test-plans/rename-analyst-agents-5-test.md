---
title: "Test Plan — Rename Analyst Agents to Phase-First Convention"
type: plan-test
status: done
lineage: rename-analyst-agents
parent: lifecycle/requirements/rename-analyst-agents-2.md
---

# Test Plan — Rename Analyst Agents

This plan covers all integration test and frontend test fixture updates required for the `analyst-requirements` → `requirements-analyst` and `analyst-planner` → `planning-analyst` rename.

Cross-links: [[rename-analyst-agents]] (backend plan), [[rename-analyst-agents]] (frontend plan).

---

## Milestone 1 — Update integration test helper config

### Description

The embedded YAML config in `tests/integration/agent_helpers_test.go` defines test agent entries. Update agent names, comments, and email addresses.

### Files to change

- `tests/integration/agent_helpers_test.go`

### Changes

1. Line 26 comment: `analyst-requirements` → `requirements-analyst`.
2. Line 27 comment: `analyst-planner` → `planning-analyst`.
3. Line 68: `name: analyst-requirements` → `name: requirements-analyst`.
4. Line 77: `email: analyst-requirements@test.local` → `email: requirements-analyst@test.local`.
5. Line 81: `name: analyst-planner` → `name: planning-analyst`.
6. Line 92: `email: analyst-planner@test.local` → `email: planning-analyst@test.local`.
7. Line 112 comment: update the parenthetical list to use new names.

### Acceptance criteria

- [ ] `grep -c 'analyst-requirements\|analyst-planner' tests/integration/agent_helpers_test.go` returns 0.
- [ ] The embedded config correctly defines `requirements-analyst` and `planning-analyst` agents.

---

## Milestone 2 — Update integration test cases

### Description

Update all string literals and comments referencing old agent names in integration test files.

### Files to change

- `tests/integration/agent_status_test.go`
- `tests/integration/agent_ws_test.go`
- `tests/integration/agents_api_test.go`

### Changes

#### `agent_status_test.go`

1. Line 13 comment: `analyst-requirements` → `requirements-analyst`.
2. Line 26: `startAgentRun(t, env, "analyst-requirements", ...)` → `"requirements-analyst"`.
3. Line 54 comment: `analyst-planner` → `planning-analyst`.
4. Line 68: `startAgentRun(t, env, "analyst-planner", ...)` → `"planning-analyst"`.
5. Line 106: `"analyst-requirements"` → `"requirements-analyst"`.
6. Line 167: `"analyst-requirements"` → `"requirements-analyst"`.

#### `agent_ws_test.go`

1. Line 35: `"analyst-requirements"` → `"requirements-analyst"`.
2. Line 103: `"analyst-requirements"` → `"requirements-analyst"`.

#### `agents_api_test.go`

1. Line 230: `"/api/p/testproject/agents/analyst-requirements/run"` → `"requirements-analyst"`.
2. Line 274: URL string `analyst-requirements` → `requirements-analyst`.
3. Lines 320, 324, 328: `AgentName: "analyst-requirements"` → `"requirements-analyst"`.
4. Line 387: `AgentName: "analyst-requirements"` → `"requirements-analyst"`.
5. Lines 415, 419, 423: `AgentName: "analyst-requirements"` → `"requirements-analyst"`.

### Acceptance criteria

- [ ] `grep -c 'analyst-requirements\|analyst-planner' tests/integration/agent_status_test.go tests/integration/agent_ws_test.go tests/integration/agents_api_test.go` returns 0 for each file.
- [ ] All integration tests compile: `go test -c ./tests/integration/...`

---

## Milestone 3 — Update frontend test fixtures

### Description

Update the agent-to-type mapping in the shared test fixtures file.

### Files to change

- `tests/web/helpers/agent_launch_fixtures.ts`

### Changes

1. Line 88: `'analyst-requirements': 'idea'` → `'requirements-analyst': 'idea'`.
2. Line 89: `'analyst-planner': 'requirement'` → `'planning-analyst': 'requirement'`.

### Acceptance criteria

- [ ] `grep -c 'analyst-requirements\|analyst-planner' tests/web/helpers/agent_launch_fixtures.ts` returns 0.

---

## Milestone 4 — Update frontend test cases

### Description

Update all agent name string literals and comments in frontend test files.

### Files to change

- `tests/web/ArtifactRunHistory.test.ts`
- `tests/web/AgentsRunsView.sort.test.ts`
- `tests/web/agent-launch/defect-inclusion.test.ts`
- `tests/web/agent-launch/input-status-filter.test.ts`
- `tests/web/agent-launch/empty-state.test.ts`

### Changes

#### `ArtifactRunHistory.test.ts`

1. Line 46: `agent_name: 'analyst-requirements'` → `'requirements-analyst'`.
2. Line 114: `agent_name: 'analyst-requirements'` → `'requirements-analyst'`.
3. Line 132: assertion string `'analyst-requirements'` → `'requirements-analyst'`.

#### `AgentsRunsView.sort.test.ts`

1. Line 78: `agent_name: 'analyst-requirements'` → `'requirements-analyst'`.
2. Lines 145, 163, 183, 184, 206, 209: assertion/comment strings `'analyst-requirements'` → `'requirements-analyst'`.

#### `defect-inclusion.test.ts`

1. Line 162 comment: `analyst-requirements` → `requirements-analyst`.
2. Line 164: describe string `'analyst-requirements does not see defects'` → `'requirements-analyst does not see defects'`.
3. Lines 166, 177: `makeAgentSummary('analyst-requirements', ...)` → `'requirements-analyst'`.

#### `input-status-filter.test.ts`

1. Line 46 comment: `analyst-requirements` → `requirements-analyst`.
2. Line 48: describe string → `'requirements-analyst input filtering'`.
3. Lines 50, 60: `makeAgentSummary('analyst-requirements', ...)` → `'requirements-analyst'`.
4. Line 72 comment: `analyst-planner` → `planning-analyst`.
5. Line 74: describe string → `'planning-analyst input filtering'`.
6. Lines 76, 86: `makeAgentSummary('analyst-planner', ...)` → `'planning-analyst'`.
7. Line 161: `makeAgentSummary('analyst-requirements', ...)` → `'requirements-analyst'`.

#### `empty-state.test.ts`

1. Lines 48, 80, 82, 116: test description/comment strings `analyst-requirements` → `requirements-analyst`.
2. Lines 49, 81, 117: `makeAgentSummary('analyst-requirements', ...)` → `'requirements-analyst'`.

### Acceptance criteria

- [ ] `grep -rc 'analyst-requirements\|analyst-planner' tests/web/` returns 0 for every file.
- [ ] Frontend tests pass: `pnpm --prefix web test` (or equivalent).

---

## Milestone 5 — Run full test suites

### Description

Execute all test suites to verify zero regressions.

### Commands

```sh
make test-unit
go test ./tests/integration/... -v
cd web && pnpm test
```

### Acceptance criteria

- [ ] `make test-unit` passes.
- [ ] All integration tests in `tests/integration/` pass.
- [ ] All web tests in `tests/web/` pass.
- [ ] Existing artifacts produced before the rename continue to render and index correctly.
