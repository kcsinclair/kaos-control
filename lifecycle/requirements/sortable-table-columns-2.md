---
title: Sortable Table Columns
type: requirement
status: planning
lineage: sortable-table-columns
created: "2026-04-27"
priority: normal
parent: ideas/sortable-table-columns.md
labels:
    - enhancement
    - frontend
    - usability
    - vue
---

# Sortable Table Columns

## Problem

All data tables in the Innovation Maker UI currently display rows in a fixed server-default order. Users browsing artifact lists, agent-run logs, and parse-error tables have no way to reorder rows by a column of interest (e.g. sort artifacts by status, date, or title). This forces users to visually scan entire tables to locate or compare entries, which becomes increasingly painful as the number of artifacts grows.

## Goals / Non-goals

### Goals

- Allow users to sort any data table by clicking a column header.
- Provide a clear three-state toggle cycle: **unsorted -> ascending -> descending -> unsorted**.
- Display a visual indicator on the active sort column showing the current direction.
- Keep the sort interaction fast and responsive for typical table sizes (hundreds of rows).
- Implement sorting as a reusable mechanism so all current and future tables inherit it with minimal effort.

### Non-goals

- Server-side sorting or new API query parameters (sort is client-side only for now).
- Persisting sort preferences across navigation or sessions (state resets on navigate-away).
- Multi-column (compound) sorting.
- Drag-to-reorder or column reordering.

## Detailed Requirements

### Functional

1. **Sort toggle behaviour** — Each sortable column header acts as a three-state toggle:
   - **First click**: sort ascending (A-Z, oldest-first, lowest-first).
   - **Second click**: sort descending (Z-A, newest-first, highest-first).
   - **Third click**: reset to the table's default (unsorted / original) order.
   Only one column may be the active sort column at a time; activating a new column resets the previous one.

2. **Sort direction indicator** — The active sort column header must display an arrow icon (up for ascending, down for descending). Inactive sortable columns should show a subtle neutral indicator (e.g. an up-down chevron or no icon) so users know sorting is available.

3. **Applicable tables** — Sorting must be enabled on every data table currently in the UI:
   - Artifact list table (`ArtifactListView`)
   - Agent runs table (`AgentsRunsView`)
   - Parse errors table (`ParseErrorsView`)
   Any table added in the future should be able to opt in by declaring which columns are sortable.

4. **Column type awareness** — The sort comparator must handle at least:
   - Strings (case-insensitive lexicographic).
   - Dates / timestamps (chronological).
   - Numbers (numeric).
   - Enum-like values such as `status` and `priority` (sort by a defined display order, not alphabetically).
   The column definition should declare its sort type so the correct comparator is selected automatically.

5. **Interaction with pagination** — When the table is paginated, sorting applies to the full dataset available on the client, not just the visible page. After sorting, the view resets to page 1.

6. **Interaction with filters** — Sorting operates on the already-filtered result set. Changing a filter resets the sort state to default.

7. **State lifetime** — Sort state is local to the table component instance. It resets when the user navigates away from the view and returns.

8. **Keyboard accessibility** — Column headers must be focusable and activatable via Enter or Space, following standard button semantics.

### Non-functional

1. **Performance** — Sorting up to 1 000 rows must complete in under 100 ms on a mid-range device. No perceptible UI jank.
2. **Reusability** — Sorting logic should live in a shared composable or component (e.g. `useSortableTable` or a `<SortableTable>` wrapper) so individual views do not duplicate sort code.
3. **Visual consistency** — Sort icons and header hover/active styles must follow the existing design-token palette and spacing conventions used in the app.
4. **No new runtime dependencies** — Sorting must be implemented with plain TypeScript; do not add a table library.

## Acceptance Criteria

- [ ] Clicking a column header in [[sortable-table-columns]] artifact list table sorts rows ascending on first click, descending on second, and resets on third.
- [ ] An arrow icon on the sorted column header reflects the current direction; other headers show no active arrow.
- [ ] String columns sort case-insensitively; date columns sort chronologically; numeric columns sort numerically.
- [ ] Status and priority columns sort by their logical display order, not alphabetically.
- [ ] Sorting a paginated table resets the view to page 1 and sorts across the full client-side dataset.
- [ ] Changing any filter dropdown resets the sort state to default order.
- [ ] Sort state resets when navigating away from the view and returning.
- [ ] Column headers are keyboard-accessible (focusable, activatable with Enter/Space).
- [ ] Sorting 1 000 rows completes without perceptible delay.
- [ ] All three existing table views (`ArtifactListView`, `AgentsRunsView`, `ParseErrorsView`) support column sorting.
- [ ] Sort logic is extracted into a reusable composable or component, not duplicated per view.
- [ ] No new runtime dependencies are introduced.

## Open Questions

1. Should enum sort orders for `status` and `priority` be defined in the frontend only, or derived from the project's `config.yaml` workflow definition?

> sortable based on client side data, just go with the text for now.

2. Are there columns in any table that should explicitly be **non-sortable** (e.g. an actions column)?

> yes, look for the obvious ones which do not have data in them.

3. When server-side pagination is in use (artifact list fetches pages from the API), client-side sorting can only sort the current page's data — should we defer sorting on paginated API tables until a `sort` query parameter is added to the backend, or fetch all rows client-side first?

> We will use client side pagination initially, the number of items is relatively small.
