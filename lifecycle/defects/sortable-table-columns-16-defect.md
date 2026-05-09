---
title: "useSortableTable: empty strings and nulls not sorted to end"
type: defect
status: done
lineage: sortable-table-columns
parent: lifecycle/tests/sortable-table-columns-13-test.md
labels: [defect]
assignees:
  - role: frontend-developer
    who: agent
release: KC-Feature-Sprint
---

# useSortableTable: empty strings and nulls not sorted to end

Two unit tests in `tests/web/useSortableTable.test.ts` fail because the `useSortableTable` composable does not place empty-string and `null` values at the end of sorted results.

## Reproduction Steps

1. `cd tests/web`
2. `pnpm exec vitest run useSortableTable.test.ts`
3. Observe failures in:
   - `useSortableTable — null and empty value handling › rows with empty-string values sort to the end (ascending)`
   - `useSortableTable — null and empty value handling › null handling is consistent: nulls remain at end in descending order too`

## Expected Behaviour

- **Ascending sort:** given rows `['Banana', '', 'Apple']`, after `toggleSort('title')` the order should be `['Apple', 'Banana', '']` — empty string last.
- **Descending sort:** given rows `['Banana', null, 'Apple']`, after two `toggleSort('title')` calls the order should be `['Banana', 'Apple', null]` — null last regardless of direction.

## Actual Behaviour

- **Ascending:** `sortedRows.value[0].title` is `''` instead of `'Apple'`. Empty strings sort before non-empty strings instead of after.
- **Descending:** `sortedRows.value[1].title` is `null` instead of `'Apple'`. Null values are not pinned to the end when sort direction is descending.

```
AssertionError: expected '' to be 'Apple' // Object.is equality
 ❯ useSortableTable.test.ts:279:23

AssertionError: expected null to be 'Banana' // Object.is equality
 ❯ useSortableTable.test.ts:296:23
```

The comparator in `web/src/composables/useSortableTable.ts` treats empty strings as ordinary values and applies the same direction multiplier to nulls, rather than unconditionally pushing them to the tail.

## Logs / Output

```
 FAIL  useSortableTable.test.ts > useSortableTable — null and empty value handling > rows with empty-string values sort to the end (ascending)
AssertionError: expected '' to be 'Apple' // Object.is equality
    279|     expect(titles[0]).toBe('Apple')

 FAIL  useSortableTable.test.ts > useSortableTable — null and empty value handling > null handling is consistent: nulls remain at end in descending order too
AssertionError: expected null to be 'Banana' // Object.is equality
    296|     expect(titles[1]).toBe('Apple')

 Test Files  1 failed (1)
      Tests  2 failed | 20 passed (22)
```
