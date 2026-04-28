---
title: "Hide Done Items by Default — Frontend Plan"
type: plan-frontend
status: done
lineage: hide-done-items-by-default
parent: lifecycle/requirements/hide-done-items-by-default-2.md
---

# Hide Done Items by Default — Frontend Plan

Add a "Show completed" toggle to `ArtifactListView`, `KanbanBoardView`, and `GraphView`. When unchecked (the default on every page load), artifacts with status `done`, `rejected`, or `abandoned` are excluded from rendering. No backend changes are needed — see [[hide-done-items-by-default]] backend plan.

## Milestone 1: Define Terminal Statuses Constant

### Description

Create a shared constant `TERMINAL_STATUSES` containing `['done', 'rejected', 'abandoned']` so all three views reference a single source of truth.

### Files to Change

- `web/src/types/api.ts` — add an exported `TERMINAL_STATUSES` array constant at the end of the file.

### Acceptance Criteria

- [ ] `TERMINAL_STATUSES` is exported from `web/src/types/api.ts` and contains exactly `['done', 'rejected', 'abandoned']`.
- [ ] No other files duplicate this list.

## Milestone 2: Add Toggle to ArtifactListView

### Description

Add a `showCompleted` ref (default `false`) and a labelled checkbox toggle in the `.list-header` area. When unchecked, the view must filter out terminal-status artifacts client-side before rendering the table rows. The `total` count shown in the header must reflect the visible (filtered) set.

Since `ArtifactListView` uses server-side pagination via `store.fetchList()`, the filtering must be applied after the store populates `items`. Add a computed `visibleItems` that filters `store.items`, and bind the table to `visibleItems` instead of `store.items`. Update the displayed count accordingly.

### Files to Change

- `web/src/views/project/ArtifactListView.vue` — add `showCompleted` ref, computed `visibleItems`, checkbox in header, bind table to `visibleItems`, update count display.

### Acceptance Criteria

- [ ] A "Show completed" checkbox appears in the `.list-header` area, after the count badge.
- [ ] The checkbox is unchecked on initial page load and on every navigation to this view.
- [ ] When unchecked, rows with status `done`, `rejected`, or `abandoned` are hidden from the table.
- [ ] When checked, all rows (including terminal-status) are shown.
- [ ] The displayed artifact count reflects only the visible rows, not `store.total`.
- [ ] The checkbox has an associated `<label>` element for accessibility.
- [ ] The checkbox is keyboard-focusable with a visible focus indicator.

## Milestone 3: Add Toggle to KanbanBoardView

### Description

Add a `showCompleted` ref (default `false`) and a labelled checkbox in the `.board-header` area. When unchecked, columns whose statuses are all terminal (i.e. the "Done" column with statuses `done`, `abandoned`, `rejected`) must be hidden entirely. The `useKanbanBoard` composable's `columns` computed already builds columns from `allArtifacts`; add client-side filtering in the composable by accepting a `hideTerminal` flag that excludes terminal-status artifacts from `applyClientFilters` and suppresses columns that become empty as a result of this filtering.

### Files to Change

- `web/src/composables/useKanbanBoard.ts` — add a `hideTerminal` reactive ref; incorporate it into `applyClientFilters` so terminal-status cards are excluded when active; in the `columns` computed, skip columns where all mapped statuses are terminal and the column has zero cards after filtering.
- `web/src/views/project/KanbanBoardView.vue` — add `showCompleted` ref, labelled checkbox in `.board-header`, bind `showCompleted` to the composable's `hideTerminal` (inverted).

### Acceptance Criteria

- [ ] A "Show completed" checkbox appears in the `.board-header` area.
- [ ] The checkbox is unchecked on initial page load.
- [ ] When unchecked, the "Done" column (mapping `done`, `abandoned`, `rejected`) is hidden entirely from the board.
- [ ] When checked, the "Done" column and its cards appear normally.
- [ ] Column card counts reflect only the visible (filtered) cards.
- [ ] Other columns are unaffected when the toggle is unchecked (they never contain terminal-status cards in the current kanban config).
- [ ] The checkbox is keyboard-accessible with a visible focus indicator.

## Milestone 4: Add Toggle to GraphView

### Description

Add a `showCompleted` ref (default `false`) and a labelled checkbox in the `GraphFilters` sidebar. When unchecked, the graph store's filtering pipeline must exclude nodes with terminal statuses and prune their edges.

The graph store (`web/src/stores/graph.ts`) already has a `filteredNodes` computed that applies `filter.statuses`. The cleanest approach is to add a `hideTerminal` ref to the store and incorporate it into `filteredNodes`: when `hideTerminal` is true and no explicit status filter is active, exclude nodes whose status is in `TERMINAL_STATUSES`. The existing `filteredEdges` computed already prunes edges based on `filteredNodes`, so edge removal is automatic.

### Files to Change

- `web/src/stores/graph.ts` — add `hideTerminal` ref (default `true`); modify `filteredNodes` computed to exclude terminal-status nodes when `hideTerminal` is true; export a `toggleHideTerminal` action.
- `web/src/components/graph/GraphFilters.vue` — accept new `hideTerminal` prop; add a "Show completed" checkbox toggle; emit a `toggleHideTerminal` event.
- `web/src/views/project/GraphView.vue` — pass `hideTerminal` to `GraphFilters`; wire up the `toggleHideTerminal` event to the store action.

### Acceptance Criteria

- [ ] A "Show completed" checkbox appears in the `GraphFilters` sidebar, near the existing "Show label nodes" toggle.
- [ ] The checkbox is unchecked on initial page load.
- [ ] When unchecked, nodes with status `done`, `rejected`, or `abandoned` are absent from the graph.
- [ ] Edges connected to hidden nodes are also removed (handled automatically by `filteredEdges`).
- [ ] When checked, all nodes and their edges reappear.
- [ ] The node count display (`N / M nodes`) reflects the currently visible set.
- [ ] Navigating away and returning resets the toggle to unchecked.
- [ ] The checkbox is keyboard-accessible with a visible focus indicator.

## Milestone 5: Verify Consistent Reset on Navigation

### Description

Confirm that because `showCompleted` / `hideTerminal` are local refs (not persisted in the store or URL), they reset to their default (hidden) state whenever the user navigates away and returns. The graph store's `hideTerminal` ref needs to be reset in the `GraphView`'s `onMounted` or via the composable setup.

### Files to Change

- `web/src/views/project/GraphView.vue` — ensure `store.hideTerminal` is reset to `true` in `onMounted`.
- No changes needed for `ArtifactListView` or `KanbanBoardView` since their `showCompleted` refs are component-local and reset on mount.

### Acceptance Criteria

- [ ] Navigate to `ArtifactListView`, check "Show completed", navigate away, return — toggle is unchecked, terminal items hidden.
- [ ] Navigate to `KanbanBoardView`, check "Show completed", navigate away, return — toggle is unchecked, Done column hidden.
- [ ] Navigate to `GraphView`, check "Show completed", navigate away, return — toggle is unchecked, terminal nodes hidden.
