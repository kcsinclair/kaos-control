---
title: Sort tests for ArtifactListView and AgentsRunsView crash on mount — useRoute mock missing query
type: defect
status: done
lineage: sortable-table-columns
parent: lifecycle/tests/sortable-table-columns-6-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
release: May2026
---

# Sort tests for ArtifactListView and AgentsRunsView crash on mount — useRoute mock missing query

## Reproduction Steps

1. Run `cd tests/web && pnpm test ArtifactListView.sort --reporter=verbose`.
2. Observe all 12 tests fail with `TypeError: Cannot read properties of undefined (reading 'page')`.
3. Run `pnpm test AgentsRunsView.sort --reporter=verbose`.
4. Observe all 9 tests fail with `TypeError: Cannot read properties of undefined (reading 'runs_page')`.
5. In both test files the `vue-router` mock returns `useRoute` as:
   ```ts
   useRoute: vi.fn(() => ({ params: { project: 'testproject' } }))
   ```
   There is no `query` property on the returned object.
6. At mount, `usePagination` (used by both views) reads `route.query[pageKey]` at line 22 of `web/src/composables/usePagination.ts`.
7. `route.query` is `undefined` → accessing a keyed property throws.

## Expected Behaviour

All mount operations succeed and tests exercise sorting behaviour as described in Milestone 2 (ArtifactListView) and Milestone 3 (AgentsRunsView) of the test plan. The `useRoute` mock should return a `query` object (at minimum an empty `{}`) so that `usePagination` can read query params without crashing.

## Actual Behaviour

Every test in both suites fails immediately during component mount before any sort interaction occurs:

```
TypeError: Cannot read properties of undefined (reading 'page')
  ❯ Module.usePagination  web/src/composables/usePagination.ts:22:50
  ❯ setup  web/src/views/project/ArtifactListView.vue:27:79

TypeError: Cannot read properties of undefined (reading 'runs_page')
  ❯ Module.usePagination  web/src/composables/usePagination.ts:22:50
  ❯ setup  web/src/views/project/AgentsRunsView.vue:26:79
```

## Logs / Output

```
 FAIL  ArtifactListView.sort.test.ts — 12 tests, all: Cannot read properties of undefined (reading 'page')
 FAIL  AgentsRunsView.sort.test.ts   — 9  tests, all: Cannot read properties of undefined (reading 'runs_page')
```

Fix required in `tests/web/ArtifactListView.sort.test.ts` and `tests/web/AgentsRunsView.sort.test.ts`:

```ts
// Change from:
useRoute: vi.fn(() => ({ params: { project: 'testproject' } })),

// To (at minimum):
useRoute: vi.fn(() => ({ params: { project: 'testproject' }, query: {} })),
```

Note: the pagination test files use a real `createRouter` / `createMemoryHistory` instead of a mock, which avoids this issue. The sort tests should adopt the same pattern or at least supply a `query: {}` default.
