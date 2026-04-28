---
title: "Table Pagination — Test Plan"
type: plan-test
status: draft
lineage: table-pagination
parent: lifecycle/requirements/table-pagination-2.md
---

# Table Pagination — Test Plan

This plan covers the testing strategy for the table pagination feature ([[table-pagination]]). Tests cover the reusable `TablePagination` component, the `usePagination` composable, and integration with all three table views. Both automated integration tests and manual verification steps are included.

## Milestone 1 — `TablePagination` component unit tests

### Description

Write tests for the `TablePagination` component in isolation, verifying rendering, user interactions, boundary conditions, and accessibility attributes.

### Files to change

- `tests/web/TablePagination.test.ts` — New file (or `web/src/components/common/__tests__/TablePagination.spec.ts` depending on test runner setup)

### Test cases

1. **Renders with defaults** — Mount with `totalItems=100`, verify default page size 25, "Showing 1–25 of 100" text, page 1 of 4.
2. **Page-size dropdown** — Change dropdown to 50, verify `update:pageSize(50)` and `update:currentPage(1)` emitted.
3. **Next/Previous navigation** — Click Next, verify `update:currentPage(2)` emitted. On last page, verify Next is disabled. On page 1, verify Previous is disabled.
4. **Page-jump input** — Enter "3", verify `update:currentPage(3)` emitted. Enter "999", verify clamped to last page. Enter "0", verify clamped to 1. Enter non-numeric, verify no emission or clamp to 1.
5. **Position summary accuracy** — With 142 items and size 25: page 1 shows "1–25 of 142", page 6 shows "126–142 of 142".
6. **Single-page dataset** — Mount with `totalItems=10`, `pageSize=25`. Controls render, Previous/Next disabled, page-jump disabled.
7. **Empty dataset** — Mount with `totalItems=0`. Verify graceful rendering.
8. **ARIA attributes** — Verify Previous button has `aria-label="Previous page"`, Next has `aria-label="Next page"`, page-jump input has associated label.
9. **Keyboard navigation** — Tab through controls, verify focus order is logical (page-size, previous, page-jump, next).

### Acceptance criteria

- [ ] All 9 test cases pass
- [ ] Tests cover emitted events, disabled states, and boundary conditions
- [ ] Accessibility assertions verify ARIA attributes

## Milestone 2 — `usePagination` composable tests

### Description

Write tests for the `usePagination` composable, verifying URL sync, computed slice indices, and multi-instance isolation.

### Files to change

- `tests/web/usePagination.test.ts` — New file

### Test cases

1. **Default values** — With no query params, `currentPage=1`, `pageSize=25`, `sliceStart=0`, `sliceEnd=25`.
2. **Read from URL** — With `?page=3&size=50`, verify `currentPage=3`, `pageSize=50`, `sliceStart=100`, `sliceEnd=150`.
3. **Set page updates URL** — Call `setPage(2)`, verify `router.replace` called with `?page=2&size=25`.
4. **Set page size resets page** — Call `setPageSize(50)`, verify page resets to 1 and URL updated to `?page=1&size=50`.
5. **Invalid query params** — With `?page=abc&size=-1`, verify fallback to defaults.
6. **Prefix isolation** — Two instances with prefixes `a` and `b` read/write `?a_page=`, `?b_page=` independently.

### Acceptance criteria

- [ ] All 6 test cases pass
- [ ] URL sync uses `router.replace` not `router.push`
- [ ] Slice indices are correct for all tested scenarios

## Milestone 3 — Integration tests for `ArtifactListView` pagination

### Description

Test that `ArtifactListView` correctly paginates its artifact rows, preserves state through navigation, and interacts properly with filters.

### Files to change

- `tests/web/ArtifactListView.pagination.test.ts` — New file (or extend existing test file)

### Test cases

1. **Paginated rendering** — Load 60 artifacts, verify only 25 rows visible on page 1. Navigate to page 3, verify 10 rows visible.
2. **Filter + pagination** — Apply a stage filter that reduces results to 15, verify single page with navigation disabled.
3. **State preservation** — Navigate to page 2, click into an artifact detail, go back — verify page 2 is restored via URL params.
4. **Page reset on filter change** — Go to page 3, change a filter dropdown — verify page resets to 1.
5. **Row interaction** — Verify clicking a row on page 2 navigates to the correct artifact (not an artifact from page 1).
6. **URL deep link** — Mount view with `?page=2&size=10` in route, verify page 2 with 10 rows rendered.

### Acceptance criteria

- [ ] All 6 test cases pass
- [ ] No regressions in existing artifact list functionality
- [ ] Row click targets match the visible (paginated) data

## Milestone 4 — Integration tests for `AgentsRunsView` pagination

### Description

Test that `AgentsRunsView` correctly paginates its runs table and that expanded row details work within paginated results.

### Files to change

- `tests/web/AgentsRunsView.pagination.test.ts` — New file

### Test cases

1. **Paginated rendering** — Load 30 runs, verify 25 visible on page 1, 5 on page 2.
2. **Expand row on page 2** — Navigate to page 2, expand a run row, verify detail panel shows correct run data.
3. **URL deep link** — Mount with `?runs_page=2&runs_size=10`, verify correct page rendered.

### Acceptance criteria

- [ ] All 3 test cases pass
- [ ] Expanded run details display correct data after pagination

## Milestone 5 — Integration tests for `ParseErrorsView` pagination

### Description

Test that `ParseErrorsView` correctly paginates its error rows.

### Files to change

- `tests/web/ParseErrorsView.pagination.test.ts` — New file

### Test cases

1. **Paginated rendering** — Load 30 parse errors, verify 25 visible on page 1.
2. **Reload preserves pagination** — Go to page 2, click Reload, verify page 2 is preserved.
3. **Empty state** — Load 0 errors, verify "No parse errors" message and no pagination controls crash.

### Acceptance criteria

- [ ] All 3 test cases pass
- [ ] Reload does not reset pagination state

## Milestone 6 — Build and type-check verification

### Description

Verify the full frontend build pipeline passes with the new component and all integrations.

### Files to change

- None (verification only)

### Test cases

1. `pnpm exec vue-tsc --noEmit` — passes with no type errors
2. `pnpm build` — completes successfully
3. Manual viewport test at 768px — pagination controls do not overflow

### Acceptance criteria

- [ ] `vue-tsc --noEmit` passes
- [ ] `pnpm build` passes
- [ ] Visual inspection confirms no layout breaks at 768px
