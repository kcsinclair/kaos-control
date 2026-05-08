---
title: Empty strings not sorted to end in useSortableTable (ascending)
type: defect
status: done
lineage: sortable-table-columns
parent: lifecycle/tests/sortable-table-columns-6-test.md
labels: [defect]
assignees:
  - role: frontend-developer
    who: agent
release: May2026
---

# Empty strings not sorted to end in useSortableTable (ascending)

## Reproduction Steps

1. Create a `useSortableTable` composable with a `string` column that contains rows with empty-string values alongside non-empty strings.
2. Call `toggleSort('title')` once to sort ascending.
3. Inspect `sortedRows.value` — the row with `title: ''` appears **first**, not last.

Minimal reproduction (from the failing test):
```ts
const rows = ref([
  { title: 'Banana' },
  { title: '' },
  { title: 'Apple' },
])
const { sortedRows, toggleSort } = useSortableTable(rows, { title: 'string' })
toggleSort('title')
// sortedRows.value[0].title is '' — expected 'Apple'
```

## Expected Behaviour

Empty strings should sort to the **end** of the list in ascending order (same treatment as `null`), producing `['Apple', 'Banana', '']`.

This matches the test plan specification:
> | Empty string handling (ascending) | Empty strings sort to the end |

## Actual Behaviour

Empty strings sort **first** in ascending order because `'' < 'Apple'` lexicographically. The `sortedRows` computed property in `web/src/composables/useSortableTable.ts` only pins `null`/`undefined` values to the end; it explicitly allows empty strings to sort naturally (see comment at line 102–103).

Actual order: `['', 'Banana', 'Apple']`

## Logs / Output

```
FAIL  useSortableTable.test.ts > useSortableTable — null and empty value handling > rows with empty-string values sort to the end (ascending)
AssertionError: expected '' to be 'Apple' // Object.is equality

- Expected
+ Received

- Apple

 ❯ useSortableTable.test.ts:279:23
    277|
    278|     const titles = sortedRows.value.map(r => r.title)
    279|     expect(titles[0]).toBe('Apple')
       |                       ^
    280|     expect(titles[1]).toBe('Banana')
    281|     expect(titles[2]).toBe('')

 Test Files  1 failed (1)
      Tests  1 failed | 21 passed (22)
```

**Root cause:** `web/src/composables/useSortableTable.ts` lines 102–108 only check for `== null`; the empty-string case is not handled. The fix is to treat `''` (after trimming or direct equality) as a "sort-last" sentinel alongside `null`, or to add a pre-sort check: `if (aVal === '' && bVal !== '') return 1`.
