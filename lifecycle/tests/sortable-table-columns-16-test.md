---
title: AgentsRunsView sort tests — @/api/config mock eliminates unhandled rejections
type: test
status: in-qa
lineage: sortable-table-columns
parent: lifecycle/defects/sortable-table-columns-15-defect.md
---

# AgentsRunsView sort tests — @/api/config mock eliminates unhandled rejections

Fixes all 9 unhandled promise rejections emitted by `tests/web/AgentsRunsView.sort.test.ts`
that caused the suite to exit with code 1 despite every test assertion passing.

## Root cause

`AgentsRunsView` calls `configStore.fetchRoles(project)` in `onMounted`, which
internally calls `getRoles` from `@/api/config`. In the happy-dom test environment,
relative URLs have no base URL, so `new URL('/api/p/testproject/roles')` throws
`ERR_INVALID_URL`. The rejection was never caught, producing one unhandled rejection
per test (9 total). Vitest captures these as suite errors and exits with code 1.

## Fix applied

Added a `vi.mock('@/api/config', ...)` block alongside the existing
`vi.mock('@/api/agents', ...)` in `tests/web/AgentsRunsView.sort.test.ts`:

```ts
vi.mock('@/api/config', () => ({
  getRoles: vi.fn().mockResolvedValue({ roles: [] }),
}))
```

This prevents `configStore.fetchRoles` from issuing a real HTTP call, eliminating
all 9 unhandled rejections so the suite exits cleanly with code 0.

## Scenarios covered

All 9 existing sort scenarios in `AgentsRunsView.sort.test.ts` now pass cleanly
with no unhandled rejections:

| Describe | Scenario |
|----------|----------|
| Agent column sort | ascending — runs sorted alphabetically by agent name |
| Agent column sort | descending — second click reverses order |
| Started column sort | ascending — runs sorted chronologically |
| Elapsed column sort | ascending — runs sorted by computed elapsed time (numeric) |
| Actions column not sortable | last header has no `aria-sort` attribute |
| Actions column not sortable | clicking actions header does not trigger sort |
| Expanding run detail row | detail row appears after sorting |
| Sort indicators | exactly one column shows an active indicator at a time |
| Sort indicators | switching columns moves the active indicator |

## Files changed

| File | Change |
|------|--------|
| `tests/web/AgentsRunsView.sort.test.ts` | Added `vi.mock('@/api/config', ...)` block to suppress `fetchRoles` HTTP call in happy-dom environment |
