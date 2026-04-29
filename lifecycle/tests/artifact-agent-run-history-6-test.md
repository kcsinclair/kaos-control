---
title: "Tests: Artifact Agent Run History"
type: test
status: approved
lineage: artifact-agent-run-history
parent: lifecycle/test-plans/artifact-agent-run-history-5-test.md
created: "2026-04-28"
---

# Tests: Artifact Agent Run History

Tests covering the backend API, WebSocket events, and frontend components
delivered by the artifact-agent-run-history feature.

## Test files

### Backend integration tests

**`tests/integration/agents_api_test.go`** — appended functions:

| Test | Milestone | Description |
|------|-----------|-------------|
| `TestListAgentRunsByTargetPath_ReturnsMatchingRuns` | 1 | Seeds 3 runs (2 for `lifecycle/requirements/foo-2.md`, 1 for another path), queries by target_path, asserts exactly 2 matching runs returned |
| `TestListAgentRunsByTargetPath_EmptyResult` | 1 | Queries a non-existent path; asserts HTTP 200 with `{"runs": []}` |
| `TestListAgentRunsByTargetPath_NoParam_ReturnsAll` | 1 | Omits target_path param; asserts all seeded runs are returned |
| `TestListAgentRunsByTargetPath_OrderNewestFirst` | 1 | Seeds 3 runs with distinct timestamps for the same path; asserts response order is `started_at DESC` |
| `TestAgentRunsTargetPathIndexExists` | 2 | Opens the project SQLite DB directly and queries `sqlite_master` to confirm `idx_agent_runs_target_path` index was created at startup |

Runs are seeded directly via `env.proj.Idx.InsertAgentRun(...)` using deterministic timestamps — no timing flakiness.

**`tests/integration/agent_ws_test.go`** — appended function:

| Test | Milestone | Description |
|------|-----------|-------------|
| `TestAgentWSEvents_IncludeTargetPath` | 3 | Starts an agent run, registers a hub channel, collects `agent.started` and the terminal event (`agent.finished` or `agent.failed`); asserts both carry `target_path` matching the run's target artifact |

### Frontend unit tests

**`tests/web/ArtifactRunHistory.test.ts`** — new file (Milestones 4 & 6):

| Test | Milestone | Description |
|------|-----------|-------------|
| renders loading state while fetching | 4 | Mocks `listRunsByTargetPath` to hang; asserts no run rows are visible before resolution |
| renders empty state when no runs | 4 | Resolves with `[]`; asserts "No agent runs for this artifact." text |
| renders run list rows with correct fields | 4 | Sets store.artifactRuns; asserts truncated run ID (8 chars), agent name, status badge per row |
| run ID is truncated to exactly 8 characters | 4 | Verifies `.arh-run-id` text length |
| status badges have accessible text or aria-label | 4 | Checks all four statuses (running, done, failed, killed) have visible text or aria-label |
| emits select-run on row click | 4 | Clicks a row; asserts `select-run` emitted with full run ID |
| calls fetchRunsByTargetPath with correct args on mount | 4 | Spies on `listRunsByTargetPath`; asserts called with matching project and targetPath |
| updates list when store artifactRuns changes | 6 | Pushes a new run into store after mount; asserts new row appears without remounting |
| updates status badge when existing run status changes | 6 | Updates run status in store; asserts badge text changes reactively |

**`tests/web/RunDetailModal.test.ts`** — new file (Milestone 5):

| Test | Milestone | Description |
|------|-----------|-------------|
| displays all AgentRunRow fields after loading | 5 | Asserts run ID, agent, role, target path, status, stderr, artifacts all appear in rendered HTML |
| displays run ID in full (not truncated) | 5 | Full run ID in `.rdm-mono` |
| displays agent name and role | 5 | Both fields rendered |
| displays target path | 5 | Path appears in body |
| shows loading state before getRun resolves | 5 | Mocks getRun to hang; asserts "Loading" text visible |
| renders stderr tail inside `<pre>` | 5 | Locates `<pre>` element containing all stderr lines |
| `<pre>` has overflow styling (rdm-log class) | 5 | Verifies class for scrollable monospace block |
| emits "close" on close button click | 5 | Clicks `button[aria-label="Close"]`; asserts `close` emitted |
| emits "close" on Escape key | 5 | Triggers `keydown` with key=Escape on overlay; asserts `close` emitted |
| emits "close" on backdrop click | 5 | Dispatches click whose target is the overlay element; asserts `close` emitted |
| does NOT emit "close" on panel click | 5 | Clicks `.rdm-panel`; asserts `close` not emitted |
| overlay has tabindex="-1" | 5 | Focus trap infrastructure in place |
| Tab key does not emit "close" | 5 | Confirms Tab is handled by focus trap, not close |
| role="dialog" and aria-modal="true" | 5 | Screen reader semantics |
| close button is focusable inside panel | 5 | No disabled attribute on close button |

## Running the tests

```sh
# Backend (all new tests)
go test ./tests/integration/ -tags integration \
  -run 'TestListAgentRunsByTargetPath|TestAgentRunsTargetPathIndexExists|TestAgentWSEvents_IncludeTargetPath' \
  -v -count=1

# Frontend
cd tests/web
npx vitest run ArtifactRunHistory.test.ts RunDetailModal.test.ts --reporter=verbose
```
