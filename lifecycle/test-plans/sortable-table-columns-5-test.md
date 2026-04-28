---
title: "Sortable Table Columns — Test Plan"
type: plan-test
status: in-development
lineage: sortable-table-columns
parent: lifecycle/requirements/sortable-table-columns-2.md
---

# Sortable Table Columns — Test Plan

This plan covers integration tests for the client-side column sorting feature described in [[sortable-table-columns]]. Tests validate the `useSortableTable` composable in isolation and the sorting behaviour as integrated into each view.

## Milestone 1 — Unit tests for `useSortableTable` composable

### Description

Write tests for the composable in isolation (no DOM rendering required — test the reactive logic directly). Cover all sort types, the three-state toggle cycle, column switching, and the reset function.

### Files to change

- `tests/web/useSortableTable.test.ts` — **New file**. Unit tests for the composable.

### Acceptance criteria

- [ ] **Three-state toggle**: calling toggle on the same column cycles through asc -> desc -> null (unsorted)
- [ ] **Column switch**: toggling a new column resets direction to ascending and clears the previous column
- [ ] **String sort**: verifies case-insensitive lexicographic ordering (e.g. `["Banana", "apple", "Cherry"]` -> `["apple", "Banana", "Cherry"]`)
- [ ] **Date sort**: verifies chronological ordering of ISO 8601 date strings
- [ ] **Number sort**: verifies numeric ordering (not string-of-digits, i.e. `9 < 10`, not `"10" < "9"`)
- [ ] **Null/empty handling**: rows with missing or empty values for the sorted column sort to the end (or beginning — just consistently)
- [ ] **resetSort()**: clears sort state; `sortedRows` returns original order
- [ ] **Reactivity**: changing the source data array updates `sortedRows` without re-toggling

## Milestone 2 — Integration tests for `ArtifactListView` sorting

### Description

Test sorting behaviour in the artifact list table with a rendered component. Use a fixture dataset of artifacts with varying titles, stages, statuses, types, and dates.

### Files to change

- `tests/web/ArtifactListView.sort.test.ts` — **New file**. Integration tests for artifact table sorting.

### Acceptance criteria

- [ ] Clicking the "Path" column header sorts artifact rows alphabetically by title (ascending), then descending, then resets
- [ ] Clicking the "Created" header sorts rows chronologically
- [ ] Clicking the "Modified" header sorts rows chronologically
- [ ] Sort indicator icon changes to reflect current direction (up, down, neutral)
- [ ] Only one column shows an active sort indicator at a time
- [ ] Changing a filter dropdown resets the sort state — rows return to default order
- [ ] After sorting, pagination resets to page 1 and pages over the sorted dataset
- [ ] Keyboard activation (Enter/Space on a column header) triggers the same sort cycle as a click

## Milestone 3 — Integration tests for `AgentsRunsView` sorting

### Description

Test sorting on the agent runs table. Verify that sortable columns work and the non-sortable actions column is inert.

### Files to change

- `tests/web/AgentsRunsView.sort.test.ts` — **New file**. Integration tests for agent runs table sorting.

### Acceptance criteria

- [ ] Clicking "Agent" header sorts runs alphabetically by agent name
- [ ] Clicking "Started" header sorts runs chronologically
- [ ] Clicking "Elapsed" header sorts runs by computed elapsed time (numeric)
- [ ] The last column (actions/expand) does not respond to clicks for sorting and shows no sort indicator
- [ ] Expanding a run detail row works correctly after the table has been sorted
- [ ] Sort indicators are correct across all sortable columns

## Milestone 4 — Integration tests for `ParseErrorsView` sorting

### Description

Test sorting on the parse errors table.

### Files to change

- `tests/web/ParseErrorsView.sort.test.ts` — **New file**. Integration tests for parse errors table sorting.

### Acceptance criteria

- [ ] Clicking "File" header sorts errors alphabetically by path
- [ ] Clicking "Error" header sorts errors alphabetically by message
- [ ] Three-state toggle cycle works (asc -> desc -> reset)
- [ ] Sort indicators display correctly

## Milestone 5 — Performance and accessibility validation

### Description

Add targeted tests for the non-functional requirements: sorting performance on large datasets and keyboard accessibility compliance.

### Files to change

- `tests/web/useSortableTable.perf.test.ts` — **New file**. Performance benchmark test.
- `tests/web/SortHeader.a11y.test.ts` — **New file**. Accessibility tests for the sort header component.

### Acceptance criteria

- [ ] Sorting a dataset of 1,000 rows completes in under 100 ms (measured via `performance.now()`)
- [ ] Sorting a dataset of 5,000 rows completes in under 500 ms (stretch goal, not blocking)
- [ ] `SortHeader` renders with `role="button"` or equivalent semantics
- [ ] `SortHeader` is focusable via Tab
- [ ] `SortHeader` activates on Enter key press
- [ ] `SortHeader` activates on Space key press
- [ ] `aria-sort` attribute reflects current sort direction ("ascending", "descending", or "none")
