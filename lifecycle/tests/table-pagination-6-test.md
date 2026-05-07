---
title: "Table Pagination — Test Suite"
type: test
status: approved
lineage: table-pagination
parent: lifecycle/test-plans/table-pagination-5-test.md
---

# Table Pagination — Test Suite

This artifact documents the integration tests written for the `table-pagination` feature.
All tests live in `tests/web/` and run via Vitest + @vue/test-utils with a happy-dom environment.

## Files Created

| File | Tests | Milestone |
|------|-------|-----------|
| `tests/web/TablePagination.test.ts` | 33 | 1 — Component unit tests |
| `tests/web/usePagination.test.ts` | 30 | 2 — Composable unit tests |
| `tests/web/ArtifactListView.pagination.test.ts` | 15 | 3 — ArtifactListView integration |
| `tests/web/AgentsRunsView.pagination.test.ts` | 12 | 4 — AgentsRunsView integration |
| `tests/web/ParseErrorsView.pagination.test.ts` | 15 | 5 — ParseErrorsView integration |

**Total: 105 tests, all passing.**

## Infrastructure Change

Added `'vue-router'` to the `dedupe` list in `tests/web/vitest.config.ts`. Without deduplication,
the web source files and test files resolved different `vue-router` instances, causing `useRoute()`
to return `undefined` (broken injection chain). Deduplication ensures both contexts share one
canonical instance so `provide/inject` works across the boundary.

## Scenarios Covered

### Milestone 1 — `TablePagination` component (`TablePagination.test.ts`)

- **Default rendering**: mounts with `totalItems=100`, verifies "Showing 1–25 of 100" and page 1-of-4.
- **Page-size dropdown**: changing to 50 emits `update:pageSize(50)` and `update:currentPage(1)`.
- **Next/Previous navigation**: Next emits the incremented page; Previous is disabled on page 1; Next is disabled on last page.
- **Page-jump input**: correct page on valid entry; clamps "0" → 1 and "999" → lastPage; non-numeric keeps current page; Enter key commits the jump.
- **Position summary accuracy**: tested at page 1 and page 6 of 142 items (last page shows partial range).
- **Single-page dataset** (`totalItems=10, pageSize=25`): both buttons and the jump input disabled.
- **Empty dataset** (`totalItems=0`): renders without crash, shows "No items", controls disabled.
- **ARIA attributes**: Previous/Next buttons have correct `aria-label`; jump input has `aria-label` and associated `<label>`; container has `role="navigation"`.
- **Keyboard navigation order**: DOM order of focusable controls is `size-select → prev → jump-input → next`.

### Milestone 2 — `usePagination` composable (`usePagination.test.ts`)

Uses `vi.hoisted()` to create reactive router mocks that avoid temporal dead zone issues with `vi.mock` hoisting.

- **Default values**: `currentPage=1`, `pageSize=25`, `sliceStart=0`, `sliceEnd=25` with no query params.
- **Read from URL**: `page=3&size=50` → `currentPage=3`, `pageSize=50`, `sliceStart=100`, `sliceEnd=150`.
- **`setPage` updates URL**: calls `router.replace` with correct `page` and preserves `size`; does not call replace when value is unchanged; updates `currentPage` ref reactively.
- **`setPageSize` resets page**: resets `currentPage` to 1 and calls `router.replace` with `page=1&size=N`.
- **Invalid query params**: `page=abc`, `page=0`, `size=-1`, `size=0` all fall back to defaults.
- **Prefix isolation**: `queryPrefix='a'` reads/writes `a_page`/`a_size`; `queryPrefix='runs'` and `queryPrefix='pe'` are also verified; prefix-less uses plain `page`/`size`.
- **Slice index correctness**: spot-checked at page 2 and page 3; verified reactive update after `setPage`.

### Milestone 3 — `ArtifactListView` pagination (`ArtifactListView.pagination.test.ts`)

Uses a real Vue Router with `createMemoryHistory`. A `mountWithItems(count, url)` helper mounts the component, waits for `onMounted` fetch to complete, then patches the store with test fixtures.

- **Paginated rendering**: 60 items → 25 on page 1, 10 on page 3; first row on page 2 is item 26.
- **Filter resets page**: changing the stage dropdown calls `applyFilters()` → `setPage(1)`, URL query updates; show-completed toggle also resets page.
- **State preservation**: clicking Next writes `page=2` to the URL; navigating away and back (`router.go(-1)`) restores `page=2`.
- **Reset filters**: "Reset" button resets to page 1.
- **Row click targets paginated artifact**: first row on page 2 navigates to item-26, last row to item-50.
- **URL deep link**: mounting with `?page=2&size=10` renders items 11–20; `?page=3&size=25` renders items 51–60.

### Milestone 4 — `AgentsRunsView` pagination (`AgentsRunsView.pagination.test.ts`)

Uses `queryPrefix: 'runs'` → query keys `runs_page` and `runs_size`.

- **Paginated rendering**: 30 runs → 25 on page 1, 5 on page 2; `TablePagination` receives correct `totalItems`; not rendered when runs list is empty.
- **Expand row on page 2**: clicking a run row on page 2 shows a detail panel with correct `stderr_tail` content; returning to page 1 hides the detail.
- **URL deep link**: `?runs_page=2&runs_size=10` renders 10 rows starting at run-0011; `runs_page` is independent of plain `page`.
- **Navigation**: Next/Previous update `runs_page` in the URL.

### Milestone 5 — `ParseErrorsView` pagination (`ParseErrorsView.pagination.test.ts`)

Uses `queryPrefix: 'pe'` → query keys `pe_page` and `pe_size`.

- **Paginated rendering**: 30 errors → 25 on page 1; `TablePagination` appears with correct total; not rendered when empty; URL deep link `?pe_page=2&pe_size=10` renders 10 rows.
- **Reload preserves pagination**: clicking Reload re-fetches data without calling `setPage`; `pe_page` stays at 2 in the URL; if 35 errors are returned, page 2 now shows 10 rows.
- **Reload button disabled while loading**: the button is disabled during the initial fetch.
- **Empty state**: "No parse errors" message shown; table is absent; Reload button always present.
- **Navigation**: Next/Previous update `pe_page`; `pe_page` is independent of `runs_page`.
