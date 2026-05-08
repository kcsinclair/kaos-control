---
title: Sortable Table Columns — Integration Tests
type: test
status: approved
lineage: sortable-table-columns
parent: lifecycle/test-plans/sortable-table-columns-5-test.md
---

# Sortable Table Columns — Integration Tests

Tests for the client-side column sorting feature across the `useSortableTable`
composable, the `SortHeader` component, and all three data views that use them.

All test files live under `tests/web/` and run with Vitest.

## Test infrastructure

| File | Purpose |
|------|---------|
| `tests/web/package.json` | Vitest + `@vue/test-utils` + `happy-dom` dependencies |
| `tests/web/vitest.config.ts` | Vitest config with Vue plugin and `@` path alias pointing to `web/src/` |

Run the suite:
```sh
cd tests/web && pnpm install && pnpm test
```

## Milestone 1 — Composable unit tests

**File:** `tests/web/useSortableTable.test.ts`

Tests the `useSortableTable` composable in isolation — no DOM, no component
mounting. Exercises the reactive logic directly.

### Scenarios covered

| Scenario | Description |
|----------|-------------|
| Three-state toggle | `toggleSort` cycles null → asc → desc → null on the same column |
| Column switch | Toggling a new column resets direction to ascending; previous column is deactivated |
| String sort | Case-insensitive `localeCompare`: `["Banana","apple","Cherry"]` → `["apple","Banana","Cherry"]` |
| Date sort | ISO 8601 strings sorted chronologically |
| Number sort | Numeric comparison (`9 < 10`, not string-of-digits) |
| Text sort | `'text'` type treated identically to `'string'` |
| Null handling (ascending) | `null` values sort to the end |
| Empty string handling (ascending) | Empty strings sort to the end |
| Null handling (descending) | `null` values remain at the end in both directions |
| `resetSort()` — clears state | `sortColumn` and `sortDirection` return to `null` |
| `resetSort()` — original order | `sortedRows` returns rows in their original order |
| `resetSort()` — idempotent | Calling `resetSort()` when already unsorted is a no-op |
| Reactivity (sorted) | Adding items to source array while sorted updates `sortedRows` without re-toggling |
| Reactivity (unsorted) | Replacing source array while unsorted mirrors the new order in `sortedRows` |

## Milestone 5a — Performance tests

**File:** `tests/web/useSortableTable.perf.test.ts`

Benchmarks sorting via `performance.now()`.

| Dataset | Sort types | Budget |
|---------|-----------|--------|
| 1,000 rows | string, date, number, text | < 100 ms (blocking) |
| 5,000 rows | string, date, number | < 500 ms (stretch goal) |

## Milestone 5b — SortHeader accessibility tests

**File:** `tests/web/SortHeader.a11y.test.ts`

Mounts the `SortHeader` component (`web/src/components/SortHeader.vue`) in
isolation and validates accessibility semantics.

### Scenarios covered

| Scenario | Expected outcome |
|----------|-----------------|
| Focusability | Sortable header has `tabindex="0"` |
| Not focusable when non-sortable | No `tabindex="0"` on non-sortable columns |
| `aria-sort="none"` | When sortable but not the active column |
| `aria-sort="ascending"` | When this column is the active ascending sort |
| `aria-sort="descending"` | When this column is the active descending sort |
| `aria-sort` for sibling column | Shows `"none"` when a different column is active |
| Non-sortable — no `aria-sort` | Non-sortable columns have no `aria-sort` attribute |
| Enter key | Emits toggle with column key on `keydown Enter` |
| Enter key (non-sortable) | No emission when column is non-sortable |
| Space key | Emits toggle with column key on `keydown Space` |
| Click | Emits toggle on click |
| Click (non-sortable) | No emission when column is non-sortable |
| Label text | Renders the supplied `label` string |
| Root element | Renders as a `<th>` |

## Milestone 2 — ArtifactListView integration tests

**File:** `tests/web/ArtifactListView.sort.test.ts`

Mounts `ArtifactListView` with a Pinia store pre-loaded with fixture data. API
calls are mocked; WebSocket is stubbed.

### Scenarios covered

| Scenario | Expected outcome |
|----------|-----------------|
| Path asc | Rows sorted alphabetically by title, ascending |
| Path desc | Second click reverses order |
| Path reset | Third click restores original order |
| Created asc | Rows sorted chronologically by `created` field |
| Modified asc | Rows sorted chronologically by `mtime` field |
| Single active indicator | Only one column shows `aria-sort="ascending"` or `"descending"` |
| Indicator updates to desc | After second click indicator switches to `aria-sort="descending"` |
| Indicator resets to none | Third click removes all active indicators |
| Filter resets sort | Changing a filter `<select>` clears sort state |
| Pagination resets to page 1 | Sort triggers jump back to page 1 |
| Enter key activates sort | `keydown Enter` on a column header produces the same sort as a click |
| Space key activates sort | `keydown Space` on a column header produces the same sort as a click |

## Milestone 3 — AgentsRunsView integration tests

**File:** `tests/web/AgentsRunsView.sort.test.ts`

Mounts `AgentsRunsView` with fixture agent run data. API calls mocked.

### Scenarios covered

| Scenario | Expected outcome |
|----------|-----------------|
| Agent asc | Rows sorted alphabetically by `agent_name` |
| Agent desc | Second click reverses order |
| Started asc | Rows sorted chronologically by `started_at` |
| Elapsed asc | Rows sorted by computed elapsed time (numeric) |
| Actions column — no indicator | Last `<th>` has no `aria-sort` attribute |
| Actions column — no sort on click | Clicking last header does not reorder rows or set indicators |
| Expand after sort | Clicking a run row after sorting still opens the detail row |
| Single active indicator | Exactly one `aria-sort` active at a time |
| Indicator moves on column switch | Switching sort column moves the active indicator |

## Milestone 4 — ParseErrorsView integration tests

**File:** `tests/web/ParseErrorsView.sort.test.ts`

Mounts `ParseErrorsView` with fixture parse error data. API is mocked.

### Scenarios covered

| Scenario | Expected outcome |
|----------|-----------------|
| File asc | Errors sorted alphabetically by `path` |
| File desc | Second click reverses order |
| File reset | Third click restores original order |
| Error asc | Errors sorted alphabetically by `message` |
| Error desc | Second click reverses order |
| Three-state cycle (File) | Full asc → desc → reset cycle verified via `aria-sort` |
| Single active indicator | At most one column active at a time |
| Asc indicator visible | `aria-sort="ascending"` shown after first click |
| Desc indicator visible | `aria-sort="descending"` shown after second click; ascending gone |
| Reload after sort | Reload button re-fetches data without crashing; table repopulates |
