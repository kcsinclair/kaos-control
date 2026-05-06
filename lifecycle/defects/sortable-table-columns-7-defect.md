---
title: useSortableTable columns API mismatch — sort not applied for any type
type: defect
status: done
lineage: sortable-table-columns
parent: lifecycle/tests/sortable-table-columns-6-test.md
labels: [defect]
assignees:
  - role: frontend-developer
    who: agent
---

# useSortableTable columns API mismatch — sort not applied for any type

## Reproduction Steps

1. Run `cd tests/web && pnpm test useSortableTable.test.ts --reporter=verbose`.
2. Observe 11 failures in the "string sort", "date sort", "number sort", "null and empty value handling", and "reactivity" suites.
3. The failing tests pass column definitions as plain string type literals, e.g.:
   ```ts
   useSortableTable(rows, { title: 'string' })
   useSortableTable(rows, { created: 'date' })
   useSortableTable(rows, { count: 'number' })
   ```
4. The composable at `web/src/composables/useSortableTable.ts` types `columns` as `SortColumnMap = Record<string, SortColumnDef>`, where `SortColumnDef = { type: SortType; getValue?: ... }`.
5. When the test passes the string `'string'` as the column def, `def.type` is `undefined` inside `compareValues()`.
6. The `switch (type)` in `compareValues` matches no case and returns `undefined`.
7. `Array.prototype.sort` receives `undefined` from the comparator and leaves items in original order.

## Expected Behaviour

`useSortableTable(rows, { title: 'string' })` should sort rows by the `title` field using case-insensitive string comparison when `toggleSort('title')` is called. The test spec (line 9 of `useSortableTable.test.ts`) defines the API as:

```ts
useSortableTable<T>(rows: Ref<T[]>, columns: Record<string, SortType>)
// where SortType = 'string' | 'date' | 'number' | 'text'
```

String, date, and number columns must all sort correctly using type-specific comparators.

## Actual Behaviour

11 tests fail. `sortedRows` always returns rows in their original insertion order regardless of the active column and direction. Samples from test output:

```
× useSortableTable — string sort — sorts ascending: case-insensitive lexicographic order
  → expected [ 'Banana', 'apple', 'Cherry' ] to deeply equal [ 'apple', 'Banana', 'Cherry' ]

× useSortableTable — date sort — sorts ascending: chronological order of ISO 8601 strings
  → expected '2024-03-15T00:00:00Z' to be '2023-01-01T00:00:00Z'

× useSortableTable — number sort — sorts numerically, not lexicographically (9 < 10)
  → expected [ 9, 10, 2, 100 ] to deeply equal [ 2, 9, 10, 100 ]

× useSortableTable — null and empty value handling — rows with null values sort to the end (ascending)
  → expected 'Banana' to be 'Apple'

× useSortableTable — reactivity — sortedRows updates when source data changes (without re-toggling)
  → expected [ 'B', 'A' ] to deeply equal [ 'A', 'B' ]
```

## Logs / Output

```
 Test Files  1 failed | 10 passed (15)
       Tests  11 failed | 218 passed (229)
```

Root cause in `web/src/composables/useSortableTable.ts`:

```ts
// Actual type accepted (lines 12–15):
export type SortColumnMap = Record<string, SortColumnDef>
// SortColumnDef = { type: SortType; getValue?: ... }

// But tests pass bare strings:
useSortableTable(rows, { title: 'string' })
// → columns['title'] === 'string', so def.type === undefined
```

Fix: change `columns` parameter to accept `Record<string, SortType>` (bare string type literals) as specified by the test plan, or accept both forms. All `def.type` references must be updated to read the type correctly.
