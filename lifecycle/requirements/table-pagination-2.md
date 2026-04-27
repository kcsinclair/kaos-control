---
title: Table Pagination Controls
type: requirement
status: draft
lineage: table-pagination
parent: lifecycle/ideas/table-pagination.md
---

# Table Pagination Controls

## Problem

All tables in the application render their full dataset in a single scroll. As the number of artifacts, agent runs, or parse errors grows, this degrades rendering performance, increases DOM size, and makes it harder for users to locate specific rows. There is no way to control how many rows are visible or to jump to a specific page of results.

## Goals / Non-goals

### Goals

1. Provide a consistent, reusable pagination component that can be dropped into any table view with minimal integration effort.
2. Allow users to control page size and navigate between pages via previous/next buttons, direct page-jump input, and a page-size selector.
3. Display a summary of the current position within the dataset (e.g. "Showing 1–25 of 142").
4. Preserve pagination state (current page, page size) within a session so that navigating away and returning keeps the user's position.
5. Apply pagination to all existing table views: `ArtifactListView`, `AgentsRunsView`, and `ParseErrorsView`.

### Non-goals

- Server-side / API-level pagination — all data is already loaded client-side; pagination is purely a rendering concern.
- Infinite scroll or virtual scrolling as an alternative to discrete pages.
- Persisting pagination preferences across browser sessions (localStorage) — session-level retention is sufficient for now.
- Sorting or filtering — these are orthogonal features and out of scope.

## Detailed Requirements

### Functional

1. **Reusable `TablePagination` component** — a single Vue 3 SFC in `web/src/components/` that accepts total item count and emits page/size changes. All table views must use this component rather than implementing their own pagination logic.
2. **Page-size selector** — a dropdown offering at least the options: 10, 25, 50, 100. The default page size must be 25.
3. **Previous / Next buttons** — disabled at the first and last page respectively.
4. **Page-jump input** — a numeric input allowing the user to type a page number and jump directly to it. Out-of-range values must be clamped to the valid range (1 to last page).
5. **Position summary** — text showing "Showing X–Y of Z" where X is the first row index on the current page, Y is the last, and Z is the total count.
6. **Empty / single-page behaviour** — when the total item count fits within one page, the pagination controls must still render but the Previous/Next buttons and page-jump input must be disabled.
7. **Page reset on size change** — when the user changes the page size, the current page must reset to 1.
8. **Session-scoped state retention** — each table's current page and page size must survive in-app navigation (e.g. clicking into an artifact detail and returning to the list). Use a Pinia store or Vue Router query params — not localStorage.
9. **Integration with existing views** — `ArtifactListView`, `AgentsRunsView`, and `ParseErrorsView` must slice their displayed rows based on pagination state. No backend API changes are required.

### Non-functional

1. **Performance** — switching pages must not re-fetch or re-parse the full dataset; it must only re-slice the already-loaded array. Page transitions must feel instantaneous (< 50 ms DOM update).
2. **Accessibility** — pagination controls must be keyboard-navigable. Buttons must have `aria-label` attributes. The page-jump input must have an associated label.
3. **Responsive layout** — pagination controls must not overflow or break on viewports down to 768 px wide.
4. **Consistency** — the component must use the existing application design tokens (colours, spacing, font sizes) so it looks native to the rest of the UI.

## Acceptance Criteria

- [ ] A `TablePagination` Vue component exists in `web/src/components/` and is used by all three table views ([[table-pagination]])
- [ ] Changing the page-size dropdown updates the displayed rows and resets to page 1
- [ ] Previous/Next buttons navigate pages correctly and are disabled at boundaries
- [ ] Page-jump input accepts a valid page number and navigates to it; invalid input is clamped
- [ ] "Showing X–Y of Z" text is accurate for every page, including the last page where Y ≤ Z
- [ ] When total items ≤ page size, controls render but navigation is disabled
- [ ] Pagination state (page, size) is retained when navigating away from and back to a table view within the same session
- [ ] All pagination controls are keyboard-accessible with appropriate ARIA attributes
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass with no errors
- [ ] No regressions in existing table functionality (data still displays correctly, row clicks still work)

## Open Questions

1. Should the default page size be configurable per-table, or is a single global default (25) sufficient?
2. Are there plans to add server-side pagination to any API endpoints in the near future? If so, the component interface should account for async page fetches now.
3. Should the URL reflect the current page (e.g. `?page=3&size=25`) to support shareable deep links, or is session-only state acceptable?
