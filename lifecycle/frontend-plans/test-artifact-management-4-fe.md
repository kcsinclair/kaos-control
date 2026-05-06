---
title: "Test Artifact Management — Frontend Plan"
type: plan-frontend
status: draft
lineage: test-artifact-management
parent: lifecycle/requirements/test-artifact-management-2.md
assignees:
    - role: frontend-developer
      who: agent
---

# Test Artifact Management — Frontend Plan

Implements the Testing board view, the Kanban "Show Tests" toggle, single test execution from the detail screen, and batch multi-select execution. Follows existing patterns from `KanbanBoardView`, `useKanbanBoard`, and the agent run dialog.

Cross-references: [[test-artifact-management-3-be]] (backend API and index), [[test-artifact-management-5-test]] (integration tests).

---

## Milestone 1 — Testing store and API integration

### Description
Create a Pinia store for the Testing board that manages test artifact listing, selection state, and batch execution. Add API helper functions for fetching test artifacts (using the existing `artifactsApi.listArtifacts` with `type=test` filter) and for fetching the approved-test badge count.

### Files to change
- `web/src/stores/testing.ts` (new) — `useTestingStore` with state: `tests: Artifact[]`, `loading: boolean`, `selectedPaths: Set<string>`, `batchQueue: string[]`, `batchCurrentIndex: number`, `batchRunning: boolean`, `filters: { status, lineage, label, priority }`. Actions: `fetchTests()`, `fetchApprovedCount()`, `toggleSelection(path)`, `selectAll()`, `clearSelection()`, `startBatch()`, `cancelBatch()`. Getters: `approvedTests`, `approvedCount`, `selectedTests`, `batchProgress`.
- `web/src/api/artifacts.ts` — verify `listArtifacts` accepts `type` filter parameter. If not, add it to the query params.

### Acceptance criteria
- [ ] `fetchTests()` retrieves all `type=test` artifacts in a single paginated call (up to 500 per NF1).
- [ ] `approvedCount` getter returns the number of tests with `status === 'approved'`.
- [ ] Selection state only allows selecting artifacts with `status === 'approved'`.
- [ ] Store re-fetches on `artifact.indexed` WebSocket events (using `useWebSocket`).

---

## Milestone 2 — Testing board view and route

### Description
Create the Testing board view component and register it in the Vue Router. The view displays test artifacts as cards in a filterable grid/list, grouped or filterable by status. Non-approved tests are visually distinguished.

### Files to change
- `web/src/views/project/TestingBoardView.vue` (new) — uses `useTestingStore`. Renders a toolbar with filter dropdowns (status, lineage, label, priority) and a "Run All Approved" button. Below, renders test artifact cards in a responsive grid. Each card shows title, status badge, priority, labels, and age. Approved cards include a selection checkbox. Non-approved cards have reduced opacity (0.6) and no checkbox.
- `web/src/router/index.ts` — add route `{ path: 'testing', component: () => import('../views/project/TestingBoardView.vue') }` as a child of the project layout route, alongside existing routes like `board` and `artifacts`.

### Acceptance criteria
- [ ] Navigating to `/p/:project/testing` renders the Testing board.
- [ ] All `type=test` artifacts appear as styled cards matching the `KanbanCard` visual style.
- [ ] Filter dropdowns filter the displayed cards client-side (status, lineage, label, priority per resolved question).
- [ ] Non-approved test cards have reduced opacity and are not selectable.
- [ ] Approved test cards display a checkbox for multi-select.
- [ ] Clicking a card (not the checkbox) navigates to the existing artifact detail screen.

---

## Milestone 3 — Testing navigation menu item with badge

### Description
Add a "Testing" item to the left sidebar navigation with a badge showing the count of approved tests.

### Files to change
- `web/src/components/layout/AppSidebar.vue` — add a `NavItem` with label "Testing", icon `FlaskConical` (from lucide-vue-next), and route `/p/:project/testing`. Add a computed badge count that reads from `useTestingStore().approvedCount`. Fetch the count on mount and refresh on `artifact.indexed` WS events.

### Acceptance criteria
- [ ] "Testing" appears in the left navigation below the existing menu items.
- [ ] The menu item displays a badge with the count of approved tests (e.g. "3").
- [ ] Badge updates reactively when test artifacts change status (via WS event refresh).
- [ ] Clicking the item navigates to the Testing board view.
- [ ] The badge is hidden when the count is 0.

---

## Milestone 4 — Single test execution from detail screen

### Description
When viewing an artifact detail screen for a `type: test` artifact with `status: approved`, show a "Run Test" button. Clicking it invokes the QA agent against this artifact and shows a running state until completion.

### Files to change
- `web/src/views/project/ArtifactEditorView.vue` — add a conditional "Run Test" button in the topbar actions area. Conditions: `artifact.type === 'test' && artifact.status === 'approved'`. On click, call `POST /api/p/:project/agents/qa/run` with `{ target_path: artifact.path }`. While running, replace the button with a disabled "Running..." indicator. Listen for `agent.finished` / `agent.failed` WS events with matching `target_path` to restore the button state. On completion, the existing `artifact.indexed` handler will refresh the artifact data.

### Acceptance criteria
- [ ] "Run Test" button appears only for `type: test` artifacts in `approved` status.
- [ ] Clicking "Run Test" sends the correct API request to the QA agent.
- [ ] Button transitions to disabled "Running..." state immediately on click.
- [ ] On `agent.finished`/`agent.failed` with matching target, the button state resets.
- [ ] Updated artifact status is reflected via the existing `artifact.indexed` WS refresh.

---

## Milestone 5 — Multi-select and batch execution

### Description
Implement multi-select on the Testing board and batch serial execution of selected tests. Include a "Run All Approved" convenience button. Show progress during batch execution.

### Files to change
- `web/src/views/project/TestingBoardView.vue` — wire checkbox state to `useTestingStore().toggleSelection(path)`. Add a toolbar area with: "Run Tests (N selected)" button (enabled when `selectedTests.length > 0`), and "Run All Approved" button (selects all approved then starts batch). During batch execution, show a progress bar or indicator: "Running test 2 of 5: <test title>".
- `web/src/stores/testing.ts` — implement `startBatch()`: copies selected paths into `batchQueue`, sets `batchCurrentIndex = 0`, `batchRunning = true`. Uses a sequential loop: for each path, calls `POST /api/p/:project/agents/qa/run`, waits for the corresponding `agent.finished`/`agent.failed` WS event, then advances `batchCurrentIndex`. On completion or all tests done, sets `batchRunning = false` and clears the queue. `cancelBatch()` stops after the current test finishes (does not abort the running agent).

### Acceptance criteria
- [ ] Checking/unchecking approved test cards updates the selection count in the toolbar.
- [ ] "Run Tests" button triggers serial execution of all selected tests.
- [ ] Tests execute one at a time; the next starts only after the previous completes.
- [ ] Progress indicator shows which test is running and how many remain (e.g. "Running 2/5: Login validation").
- [ ] If a test produces a defect, execution continues with the next test.
- [ ] "Run All Approved" selects all approved tests and starts batch execution.
- [ ] User can navigate away and return — batch continues in background (NF2) because state is in the Pinia store, not component-local. The requirement mentions using [[agent-task-scheduler]] for background execution; if the scheduler is available, batch execution should be submitted as a scheduler job instead of running in-browser.
- [ ] "Cancel" stops the batch after the current test completes.

---

## Milestone 6 — Kanban board "Show Tests" toggle

### Description
Add a "Show Tests" checkbox to the Kanban board toolbar. When unchecked (default), test artifacts are filtered out of all columns. Toggle state persists in the Pinia UI store for the session.

### Files to change
- `web/src/stores/ui.ts` (or `web/src/stores/uiStore.ts`) — add `showTestsOnKanban: boolean` state field, default `false`.
- `web/src/composables/useKanbanBoard.ts` — in the computed that assigns artifacts to columns, add a filter: if `!uiStore.showTestsOnKanban`, exclude artifacts where `type === 'test'`.
- `web/src/views/project/KanbanBoardView.vue` — add a checkbox in the toolbar/header area bound to `uiStore.showTestsOnKanban`. Label: "Show Tests".

### Acceptance criteria
- [ ] "Show Tests" checkbox appears in the Kanban board toolbar.
- [ ] Default state is unchecked; no `type: test` artifacts appear in Kanban columns.
- [ ] Checking the box reactively shows test artifacts in their correct status columns without a page reload (NF3).
- [ ] Unchecking removes them reactively.
- [ ] Toggle state persists across navigations within the same session (Pinia store, not localStorage per F3).
- [ ] Toggle does not trigger a new API call — it filters the already-loaded artifact data client-side.
