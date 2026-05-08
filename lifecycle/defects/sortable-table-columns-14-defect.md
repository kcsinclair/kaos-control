---
title: ParseErrorsView sort tests crash — useRoute mock missing query property
type: defect
status: done
lineage: sortable-table-columns
parent: lifecycle/tests/sortable-table-columns-10-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
release: May2026
---

# ParseErrorsView sort tests crash — useRoute mock missing query property

All 10 scenarios in `tests/web/ParseErrorsView.sort.test.ts` fail to run.
Every test aborts in the `setup()` phase with:

```
TypeError: Cannot read properties of undefined (reading 'pe_page')
```

## Reproduction Steps

1. `cd tests/web`
2. `pnpm exec vitest run ParseErrorsView.sort.test.ts`
3. Observe all 10 tests fail immediately — none reach their first assertion.

## Expected Behaviour

All 10 sorting scenarios from Milestone 4 should collect and run; the
vi.mock hoisting fix described in the parent test artifact was intended to
make these tests executable.

## Actual Behaviour

Every test fails in component `setup()` before any sorting interaction is
attempted. The error originates in `usePagination.ts:22` when it reads
`route.query['pe_page']`, but `route.query` is `undefined` because the
`useRoute` mock in the test file only provides `params`, not `query`:

```ts
// tests/web/ParseErrorsView.sort.test.ts — current (broken)
useRoute: vi.fn(() => ({ params: { project: 'testproject' } })),
//                                                            ^ no `query` property
```

`usePagination` (called from `ParseErrorsView.vue:18`) then attempts:

```ts
const currentPage = ref(parsePositiveInt(route.query[pageKey], 1))
//                                                ^ TypeError: undefined['pe_page']
```

## Logs / Output

```
 ❯ ParseErrorsView.sort.test.ts  (10 tests | 10 failed) 18ms
   ❯ … clicking File header sorts errors alphabetically by path (ascending)
     → Cannot read properties of undefined (reading 'pe_page')
   ❯ … clicking File header again sorts descending
     → Cannot read properties of undefined (reading 'pe_page')
   ❯ … (all 10 tests, same error)

TypeError: Cannot read properties of undefined (reading 'pe_page')
 ❯ Module.usePagination ../../web/src/composables/usePagination.ts:22:50
     22|   const currentPage = ref(parsePositiveInt(route.query[pageKey], 1))
 ❯ setup ../../web/src/views/project/ParseErrorsView.vue:18:79
```

## Fix

Update the `useRoute` mock to include `query: {}`, matching how the
companion pagination test (`ParseErrorsView.pagination.test.ts`) handles
it by providing a real router that returns a full route object. The
simplest targeted fix in the sort test:

```ts
// tests/web/ParseErrorsView.sort.test.ts — fixed
useRoute: vi.fn(() => ({
  params: { project: 'testproject' },
  query:  {},
})),
```

Alternatively, switch `mountView()` to use a real `createRouter` +
`router.push` (as the pagination test does) so that `usePagination` gets
a fully-formed route object automatically.
