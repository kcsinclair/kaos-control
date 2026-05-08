---
title: ParseErrorsView sort tests — useRoute mock query fix
type: test
status: draft
lineage: sortable-table-columns
parent: lifecycle/defects/sortable-table-columns-14-defect.md
---

# ParseErrorsView sort tests — useRoute mock query fix

Fixes the two issues that prevented all 10 Milestone 4 sort scenarios in
`tests/web/ParseErrorsView.sort.test.ts` from passing:

1. **Crash fix** — `useRoute` mock was missing `query: {}`, causing a
   `TypeError` in `usePagination.ts:22` during every test's `setup()` phase.
2. **Fixture fix** — test fixture used mixed directory prefixes
   (`lifecycle/defects/`, `lifecycle/ideas/`, etc.), so full-path alphabetical
   sort produced a different order than the filename-based expectations.

## Changes made

### `tests/web/ParseErrorsView.sort.test.ts`

| Change | Detail |
|--------|--------|
| `useRoute` mock | Added `query: {}` alongside `params` so `usePagination` can read `route.query['pe_page']` without crashing |
| Fixture `makeErrors()` | Changed all four fixture paths to use the same `lifecycle/requirements/` prefix so alphabetical sort by full path matches alphabetical sort by filename |

## Scenarios covered

All 10 scenarios from Milestone 4 now collect and pass:

| Describe | Scenario | Status |
|----------|----------|--------|
| File column sort | ascending — errors sorted by path ascending | passing |
| File column sort | descending — second click reverses | passing |
| File column sort | reset — third click restores original order | passing |
| Error column sort | ascending — errors sorted by message ascending | passing |
| Error column sort | descending — second click reverses | passing |
| Three-state toggle cycle | asc → desc → unsorted via `aria-sort` | passing |
| Sort indicators | ascending indicator shows after first click | passing |
| Sort indicators | descending indicator shows after second click; ascending gone | passing |
| Sort indicators | only one column has an active indicator at a time | passing |
| Reload after sort | Reload button re-fetches without crash; rows remain | passing |

## Files changed

| File | Change |
|------|--------|
| `tests/web/ParseErrorsView.sort.test.ts` | Added `query: {}` to `useRoute` mock; normalised fixture paths to same directory |
