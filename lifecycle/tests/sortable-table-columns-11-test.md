---
title: ArtifactListView and AgentsRunsView sort tests — useRoute mock query fix
type: test
status: in-qa
lineage: sortable-table-columns
parent: lifecycle/defects/sortable-table-columns-8-defect.md
---

# ArtifactListView and AgentsRunsView sort tests — useRoute mock query fix

Fixes the two sort test suites that crashed on mount because the `vue-router`
mock's `useRoute` return value was missing the `query` property, causing
`usePagination` to throw `TypeError: Cannot read properties of undefined
(reading 'page')` on line 22 of `web/src/composables/usePagination.ts`.

Additionally fixes a secondary failure in the keyboard-activation tests of
`ArtifactListView.sort.test.ts` where `pathHeader!.find('[tabindex="0"]') ??
pathHeader!` used nullish coalescing against an empty DOMWrapper object
(truthy) rather than falling through to the `<th>` itself.

## Files changed

| File | Change |
|------|--------|
| `tests/web/ArtifactListView.sort.test.ts` | Add `query: {}` to `useRoute` mock; add `replace: vi.fn()` to `useRouter` mock; fix keyboard tests to trigger on `<th>` directly |
| `tests/web/AgentsRunsView.sort.test.ts` | Add `query: {}` to `useRoute` mock; add `replace: vi.fn()` to `useRouter` mock |

## Root cause

`usePagination` reads `route.query[pageKey]` synchronously during component
setup. When `useRoute` is mocked without a `query` property the read throws
before any test body executes, failing the entire suite.

`usePagination` also calls `router.replace(...)` when syncing the URL, so the
`useRouter` mock needed a `replace` stub to avoid a second crash path.

The keyboard tests additionally used `pathHeader!.find('[tabindex="0"]') ??
pathHeader!` which never fell back to `pathHeader!` because Vue Test Utils
`find()` returns an empty-but-truthy DOMWrapper on no match. Since
`SortHeader.vue` renders `tabindex="0"` on the `<th>` root element (not a
child), the correct approach is to trigger the `keydown` event directly on the
header wrapper.

## Scenarios covered

All 21 tests across the two suites now pass:

### ArtifactListView.sort.test.ts (12 tests)

| Scenario | Status |
|----------|--------|
| Path asc — rows sorted alphabetically by title | passing |
| Path desc — second click reverses order | passing |
| Path reset — third click restores original order | passing |
| Created asc — rows sorted chronologically by `created` | passing |
| Modified asc — rows sorted chronologically by `mtime` | passing |
| Single active indicator — exactly one `aria-sort` active | passing |
| Indicator updates to desc on second click | passing |
| Indicator resets to none on third click | passing |
| Filter change resets sort state | passing |
| Pagination resets to page 1 after sort | passing |
| Enter key activates sort | passing |
| Space key activates sort | passing |

### AgentsRunsView.sort.test.ts (9 tests)

| Scenario | Status |
|----------|--------|
| Agent asc — rows sorted alphabetically by `agent_name` | passing |
| Agent desc — second click reverses order | passing |
| Started asc — rows sorted chronologically by `started_at` | passing |
| Elapsed asc — rows sorted by computed elapsed time (numeric) | passing |
| Actions column — no `aria-sort` indicator | passing |
| Actions column — click does not trigger sort | passing |
| Expand after sort — detail row opens correctly | passing |
| Single active indicator | passing |
| Indicator moves on column switch | passing |
