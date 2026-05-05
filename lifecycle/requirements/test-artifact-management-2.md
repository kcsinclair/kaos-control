---
title: Test Artifact Management and Test Runner
type: requirement
status: draft
lineage: test-artifact-management
parent: lifecycle/ideas/test-artifact-management.md
labels:
    - feature
    - testing
    - frontend
    - backend
    - qa
    - agent
---

# Test Artifact Management and Test Runner

## Problem

Test artifacts (`type: test`) are a special class of lifecycle artifact that are executed frequently and require dedicated UX beyond the generic Kanban board. Today, test artifacts are mixed in with all other artifact types on the board, making them hard to find and manage. There is no way to trigger a test run from the UI, and no way to batch-run a selection of tests. Users must manually invoke the QA agent from the CLI or API, which breaks the flow of the graphical lifecycle tool.

## Goals / Non-goals

### Goals

- Provide a dedicated "Testing" view accessible from the left navigation menu that displays all `type: test` artifacts as cards.
- Allow users to run a single approved test artifact via the QA agent directly from the artifact detail screen.
- Allow users to multi-select approved tests and run them serially via the QA agent.
- Keep the main Kanban board uncluttered by hiding test artifacts behind an opt-in toggle.

### Non-goals

- Parallel/concurrent test execution (serial only in v1).
- Test result history or trend reporting (rely on defect artifacts and git history).
- Custom test ordering or priority-based execution order.
- Integration with external CI/CD systems.
- Real-time streaming of test output to the UI (agent run status via existing WebSocket events is sufficient).

## Detailed Requirements

### Functional

#### F1: Testing Menu Item

- A new left-navigation item labelled "Testing" must be added.
- Clicking it navigates to a dedicated Testing board view.
- The menu item must display a badge count of tests currently in `approved` status (ready to run).

#### F2: Testing Board View

- The board displays all artifacts where `type == "test"` as cards.
- Cards must match the visual style of existing Kanban board cards (title, status, priority, labels, age).
- Cards must be grouped or filterable by status (at minimum: `draft`, `approved`, `done`).
- Non-approved tests must be visually distinguished (e.g. reduced opacity or a greyed-out treatment) and must NOT be selectable for execution.

#### F3: Kanban Board Test Visibility Toggle

- A "Show Tests" checkbox must be added to the Kanban board header/toolbar.
- Default state: **unchecked** (test artifacts hidden from Kanban columns).
- When checked, test artifacts appear in their respective status columns alongside other artifact types.
- The toggle state must persist across page navigations within the same session (Pinia store, not localStorage in v1).

#### F4: Single Test Execution (Detail Screen)

- When viewing an artifact detail screen for a `type: test` artifact with `status: approved`, a "Run Test" button must be displayed.
- Clicking "Run Test" invokes `POST /api/p/:project/agents/qa/run` with `{ target_path: <artifact path> }`.
- While the agent run is active, the button must be replaced with a disabled "Running..." indicator.
- On completion, the UI must reflect the updated artifact status (via existing `artifact.indexed` WebSocket event).

#### F5: Multi-select and Batch Execution

- On the Testing board, each approved test card must have a selectable checkbox.
- A "Run Tests" action button appears when one or more tests are selected.
- Clicking "Run Tests" executes the selected tests **serially**: one `POST /api/p/:project/agents/qa/run` call per test, waiting for completion before starting the next.
- Progress must be indicated: show which test is currently running and how many remain.
- If a test run produces defects, execution continues with the next test (do not halt the batch).

#### F6: Backend — No New Endpoints Required

- The existing `POST /api/p/:project/agents/:name/run` endpoint is sufficient.
- The frontend must poll or subscribe to agent run status via the existing WebSocket hub (`agent.started`, `agent.completed`, `agent.failed` events) to know when each run finishes.

### Non-functional

- **NF1**: The Testing board must load within 200 ms for projects with up to 500 test artifacts (SQLite index query).
- **NF2**: Multi-select batch execution must not block the UI thread; the user must be able to navigate away and return to see progress.
- **NF3**: The "Show Tests" toggle must not cause a full page reload; it must reactively filter the existing Kanban data.

## Acceptance Criteria

- [ ] Left navigation contains a "Testing" menu item that routes to the Testing board.
- [ ] Testing board displays all `type: test` artifacts as styled cards.
- [ ] Non-approved test cards are visually distinguished and cannot be selected.
- [ ] Clicking a test card navigates to the existing artifact detail screen.
- [ ] Artifact detail screen shows a "Run Test" button for approved test artifacts.
- [ ] "Run Test" button invokes the QA agent via `POST /api/p/:project/agents/qa/run` and shows a running state.
- [ ] Multi-select checkboxes appear on approved test cards on the Testing board.
- [ ] "Run Tests" button triggers serial execution of all selected tests.
- [ ] Batch progress indicator shows current test and remaining count.
- [ ] Batch execution continues past failures (defects raised, next test starts).
- [ ] Kanban board has a "Show Tests" checkbox, unchecked by default.
- [ ] When "Show Tests" is unchecked, no `type: test` artifacts appear in Kanban columns.
- [ ] When "Show Tests" is checked, test artifacts appear in their correct status columns.
- [ ] Toggle state persists within the session (Pinia store).
- [ ] [[test-artifact-management]] lineage artifacts link correctly in the graph view.

## Open Questions

- Should the Testing board support filtering by lineage/label, or is status grouping sufficient for v1?
- Should there be a "Run All Approved" convenience button that selects all approved tests without manual multi-select?
- What should happen if the user navigates away mid-batch: cancel remaining runs, or continue in background?
