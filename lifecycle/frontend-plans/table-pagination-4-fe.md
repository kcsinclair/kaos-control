---
title: "Table Pagination — Frontend Plan"
type: plan-frontend
status: in-development
lineage: table-pagination
parent: lifecycle/requirements/table-pagination-2.md
---

# Table Pagination — Frontend Plan

This plan covers the frontend implementation of client-side table pagination ([[table-pagination]]). The core deliverable is a reusable `TablePagination` Vue component used by all three table views, with pagination state reflected in URL query params for shareable deep links.

## Milestone 1 — Create `TablePagination` component

### Description

Build a reusable Vue 3 SFC at `web/src/components/common/TablePagination.vue`. The component accepts a total item count and the current page/size, and emits events when the user changes page or page size. It renders: page-size selector, previous/next buttons, page-jump input, and a "Showing X–Y of Z" position summary.

### Files to change

- `web/src/components/common/TablePagination.vue` — New file

### Implementation detail

Props:
- `totalItems: number` — total row count
- `currentPage: number` — 1-based current page (default 1)
- `pageSize: number` — rows per page (default 25)

Emits:
- `update:currentPage(page: number)` — when user navigates
- `update:pageSize(size: number)` — when user changes page size

Internal behaviour:
- Page-size dropdown options: 10, 25, 50, 100
- Changing page size emits both `update:pageSize` and `update:currentPage(1)` (reset to page 1)
- Previous/Next buttons disabled at boundaries (page 1 and last page)
- Page-jump input clamps out-of-range values to [1, lastPage]
- When `totalItems <= pageSize`, controls render but Previous/Next and page-jump are disabled
- All interactive elements have `aria-label` attributes
- Page-jump input has an associated `<label>`
- All controls are keyboard-navigable (native button/input elements)

### Acceptance criteria

- [ ] Component renders page-size dropdown with options 10, 25, 50, 100
- [ ] Previous/Next buttons navigate correctly and are disabled at boundaries
- [ ] Page-jump input accepts valid page numbers and clamps invalid input
- [ ] "Showing X–Y of Z" text is accurate for every page including the last
- [ ] When total items fit in one page, controls render but navigation is disabled
- [ ] Changing page size resets to page 1
- [ ] All controls have appropriate ARIA attributes and are keyboard-accessible
- [ ] Component uses existing design tokens (colours, spacing, font sizes)
- [ ] Layout does not overflow on viewports down to 768px

## Milestone 2 — Create `usePagination` composable for URL sync

### Description

Create a composable that manages pagination state (page, size) and syncs it bidirectionally with Vue Router query params (`?page=N&size=N`). Each table view will use this composable, keyed by a namespace to avoid collisions, and it will provide the computed slice indices for the current page.

### Files to change

- `web/src/composables/usePagination.ts` — New file

### Implementation detail

```ts
function usePagination(options?: { defaultSize?: number; queryPrefix?: string })
```

Returns:
- `currentPage: Ref<number>` — synced to `?page=` (or `?<prefix>_page=`)
- `pageSize: Ref<number>` — synced to `?size=` (or `?<prefix>_size=`)
- `sliceStart: ComputedRef<number>` — `(currentPage - 1) * pageSize`
- `sliceEnd: ComputedRef<number>` — `currentPage * pageSize`
- `setPage(n: number): void`
- `setPageSize(n: number): void` — also resets page to 1

Uses `useRoute` / `useRouter` to read initial values from query params and `router.replace` (not push) to update them without polluting browser history.

### Acceptance criteria

- [ ] Initial page/size are read from URL query params on mount
- [ ] Changing page or size updates URL query params via `router.replace`
- [ ] `sliceStart` and `sliceEnd` compute correct indices for array slicing
- [ ] Changing page size resets page to 1 and updates URL
- [ ] Missing or invalid query params fall back to defaults (page=1, size=25)
- [ ] Multiple instances with different prefixes do not collide

## Milestone 3 — Integrate into `ArtifactListView`

### Description

Replace the existing server-side offset/limit pagination in `ArtifactListView` with the new client-side `TablePagination` component. The view should fetch all artifacts (using `limit=0` from [[table-pagination]] backend plan) and slice them locally.

### Files to change

- `web/src/views/project/ArtifactListView.vue` — Replace pagination logic, add `TablePagination` component
- `web/src/stores/artifacts.ts` — Adjust `fetchList` usage: the view will request all items and paginate client-side

### Implementation detail

- On mount, call `store.fetchList(project, { limit: 0, ...filters })` to load all artifacts
- Use `usePagination()` composable for page/size state and URL sync
- Replace `v-for="row in store.items"` with `v-for="row in store.items.slice(sliceStart, sliceEnd)"`
- Remove existing `prevPage`/`nextPage` functions and inline pagination template
- Add `<TablePagination>` below the table, passing `totalItems`, `currentPage`, `pageSize`
- When filters change, reset page to 1

### Acceptance criteria

- [ ] `ArtifactListView` renders paginated rows using `TablePagination`
- [ ] All filter combinations work correctly with pagination
- [ ] URL reflects current page and size (e.g. `?page=2&size=25`)
- [ ] Navigating to artifact detail and back preserves page/size from URL
- [ ] Row click and keyboard enter still open artifact detail
- [ ] Page transitions are instantaneous (< 50ms, no re-fetch)
- [ ] `pnpm exec vue-tsc --noEmit` passes

## Milestone 4 — Integrate into `AgentsRunsView`

### Description

Add client-side pagination to the agent runs table using the `TablePagination` component and `usePagination` composable.

### Files to change

- `web/src/views/project/AgentsRunsView.vue` — Add pagination, slice `store.runs`

### Implementation detail

- Use `usePagination({ queryPrefix: 'runs' })` to avoid query param collision if the user navigates between views
- Slice `store.runs` with computed `sliceStart`/`sliceEnd`
- Place `<TablePagination>` after the table
- Expanded run detail rows remain functional within paginated results

### Acceptance criteria

- [ ] `AgentsRunsView` renders paginated rows using `TablePagination`
- [ ] Expanding a run row to view logs/details still works
- [ ] URL query params reflect pagination state
- [ ] Navigating away and back preserves page position via URL

## Milestone 5 — Integrate into `ParseErrorsView`

### Description

Add client-side pagination to the parse errors table using the `TablePagination` component and `usePagination` composable.

### Files to change

- `web/src/views/project/ParseErrorsView.vue` — Add pagination, slice `errors` array

### Implementation detail

- Use `usePagination({ queryPrefix: 'pe' })` for namespaced query params
- Slice `errors.value` with computed `sliceStart`/`sliceEnd`
- Place `<TablePagination>` after the table

### Acceptance criteria

- [ ] `ParseErrorsView` renders paginated rows using `TablePagination`
- [ ] URL query params reflect pagination state
- [ ] Reload button preserves pagination state
- [ ] "No parse errors" message still displays when list is empty

## Milestone 6 — Responsive layout and visual polish

### Description

Verify the `TablePagination` component layout on narrow viewports (down to 768px) and ensure it matches the application's visual style. Adjust flex-wrap, font sizes, or spacing if needed.

### Files to change

- `web/src/components/common/TablePagination.vue` — Add responsive CSS if needed

### Acceptance criteria

- [ ] Pagination controls do not overflow or break at 768px viewport width
- [ ] Component visually matches existing UI patterns (buttons, inputs, spacing)
- [ ] `pnpm build` completes with no errors
