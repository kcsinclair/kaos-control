---
title: "Sortable Table Columns — Frontend Plan"
type: plan-frontend
status: approved
lineage: sortable-table-columns
parent: lifecycle/requirements/sortable-table-columns-2.md
---

# Sortable Table Columns — Frontend Plan

This plan implements client-side column sorting across all three data tables in the UI per [[sortable-table-columns]]. The core sorting logic lives in a reusable `useSortableTable` composable. Each view opts in by declaring which columns are sortable and what sort type each uses.

## Milestone 1 — Create `useSortableTable` composable

### Description

Build a composable at `web/src/composables/useSortableTable.ts` that encapsulates all sorting state and logic. It accepts a reactive array of rows and a column-sort-type map, and exposes sorted rows, the current sort column, sort direction, and a toggle function.

The composable must:

1. Track `sortColumn` (string | null) and `sortDirection` (`'asc' | 'desc' | null`).
2. Implement a three-state toggle cycle: unsorted -> ascending -> descending -> unsorted. Switching columns resets to ascending.
3. Provide comparators for four sort types:
   - `string` — case-insensitive lexicographic (`localeCompare`).
   - `date` — chronological via `new Date()` parsing.
   - `number` — numeric comparison.
   - `text` — alias for string, used for enum-like fields like `status` and `priority` that the requirement says should just sort by text for now.
4. Return a computed `sortedRows` array. When sort is inactive, return the original order.
5. Expose a `resetSort()` function for external callers (filter changes, pagination resets).

### Files to change

- `web/src/composables/useSortableTable.ts` — **New file**. The composable.

### Acceptance criteria

- [ ] `useSortableTable` is a standalone composable with no view-specific logic
- [ ] Three-state toggle cycle works: unsorted -> asc -> desc -> unsorted
- [ ] Switching to a new column resets direction to ascending
- [ ] String sort is case-insensitive
- [ ] Date sort handles ISO 8601 strings chronologically
- [ ] Number sort handles numeric comparison (not string-of-digits)
- [ ] `resetSort()` clears sort state back to unsorted
- [ ] `sortedRows` is a Vue computed that reacts to changes in source data, column, and direction
- [ ] Sorting 1,000 rows completes in under 100 ms (no perceptible delay)

## Milestone 2 — Create sort header indicator component

### Description

Build a small `SortHeader` component (or use scoped slots in the composable) that renders a clickable `<th>` with:

- An up-arrow icon when the column is sorted ascending.
- A down-arrow icon when sorted descending.
- A subtle neutral indicator (e.g. `ArrowUpDown` from lucide) when the column is sortable but not the active sort column.
- No indicator for non-sortable columns — they render as plain `<th>` elements.

The header must be keyboard-accessible: focusable via `tabindex="0"` and activatable with Enter or Space.

### Files to change

- `web/src/components/SortHeader.vue` — **New file**. The sort header component.

### Acceptance criteria

- [ ] Ascending state shows an up-arrow icon
- [ ] Descending state shows a down-arrow icon
- [ ] Sortable-but-inactive columns show a neutral up-down icon
- [ ] Non-sortable columns render as plain `<th>` with no icon or click handler
- [ ] Click toggles sort via the composable's toggle function
- [ ] `tabindex="0"` is set; Enter and Space activate the sort toggle
- [ ] Hover and active styles use existing design tokens (`--color-text-muted`, `--color-text`, `--color-accent`)
- [ ] Icons come from `lucide-vue-next` (already in the project)

## Milestone 3 — Integrate sorting into `ArtifactListView`

### Description

Wire `useSortableTable` and `SortHeader` into the artifact list table. Define sortable columns:

| Column    | Key       | Sort type |
|-----------|-----------|-----------|
| Path      | `title`   | string    |
| Stage     | `stage`   | string    |
| Status    | `status`  | string    |
| Type      | `type`    | string    |
| Created   | `created` | date      |
| Modified  | `mtime`   | date      |

All six data columns are sortable.

The view must fetch all artifacts (`limit=0`) so sorting operates on the full dataset (coordinating with [[sortable-table-columns]] backend plan). The existing pagination controls from [[table-pagination]] must page over `sortedRows`. Changing a filter must call `resetSort()`.

### Files to change

- `web/src/views/project/ArtifactListView.vue` — Replace static `<th>` elements with `SortHeader`. Use `useSortableTable` to wrap `store.items`. Update pagination to slice `sortedRows`. Call `resetSort()` in `applyFilters()` and `resetFilters()`.
- `web/src/stores/artifacts.ts` — Update `fetchList` to use `limit: 0` when the view requests all data for client-side sorting.

### Acceptance criteria

- [ ] All six column headers are sortable and show sort indicators
- [ ] Clicking a column header sorts the full artifact list, not just the visible page
- [ ] Pagination resets to page 1 after sorting
- [ ] Changing any filter dropdown resets sort state to default order
- [ ] Navigating away and back resets sort state (component instance teardown)
- [ ] Existing row click navigation still works

## Milestone 4 — Integrate sorting into `AgentsRunsView`

### Description

Wire sorting into the agent runs table. Define sortable columns:

| Column   | Key            | Sort type |
|----------|----------------|-----------|
| Run ID   | `run_id`       | string    |
| Agent    | `agent_name`   | string    |
| Target   | `target_path`  | string    |
| Status   | `status`       | string    |
| Started  | `started_at`   | date      |
| Elapsed  | (computed)     | number    |

The empty actions/expand column (`<th></th>`) is **not sortable**.

### Files to change

- `web/src/views/project/AgentsRunsView.vue` — Replace static `<th>` with `SortHeader`. Use `useSortableTable` with `store.runs`. Leave the last `<th>` (actions) as a plain header.

### Acceptance criteria

- [ ] Six data columns are sortable; actions column is not
- [ ] Sort indicators display correctly on each sortable header
- [ ] Expanding a run detail row still works after sorting
- [ ] Elapsed column sorts numerically (computed from `started_at`/`finished_at`)
- [ ] Keyboard accessibility works on all sortable headers

## Milestone 5 — Integrate sorting into `ParseErrorsView`

### Description

Wire sorting into the parse errors table. Define sortable columns:

| Column | Key       | Sort type |
|--------|-----------|-----------|
| File   | `path`    | string    |
| Error  | `message` | string    |

Both columns are sortable.

### Files to change

- `web/src/views/project/ParseErrorsView.vue` — Replace static `<th>` with `SortHeader`. Use `useSortableTable` with the local `errors` ref.

### Acceptance criteria

- [ ] Both columns are sortable with correct sort indicators
- [ ] Keyboard accessibility works on both headers
- [ ] Reload button still works after sorting (data refresh should preserve or reset sort)
