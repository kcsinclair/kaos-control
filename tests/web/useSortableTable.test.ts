/**
 * Milestone 1 — Unit tests for `useSortableTable` composable
 *
 * Tests the composable in isolation: no DOM rendering, no component mounting.
 * All acceptance criteria from the test plan are covered here.
 *
 * Expected composable location: web/src/composables/useSortableTable.ts
 * Expected API:
 *   useSortableTable<T>(rows: Ref<T[]>, columns: Record<string, SortType>)
 *   => { sortedRows, sortColumn, sortDirection, toggleSort, resetSort }
 *
 * where SortType = 'string' | 'date' | 'number' | 'text'
 */

import { describe, it, expect, beforeEach } from 'vitest'
import { ref, nextTick } from 'vue'
import { useSortableTable } from '../../web/src/composables/useSortableTable'

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

function makeArtifacts() {
  return [
    { title: 'Banana', created: '2024-03-15T00:00:00Z', count: 9 },
    { title: 'apple',  created: '2023-01-01T00:00:00Z', count: 10 },
    { title: 'Cherry', created: '2025-06-20T00:00:00Z', count: 2 },
  ]
}

// ---------------------------------------------------------------------------
// Three-state toggle
// ---------------------------------------------------------------------------

describe('useSortableTable — three-state toggle', () => {
  it('starts in unsorted state (sortColumn and sortDirection are null)', () => {
    const rows = ref(makeArtifacts())
    const { sortColumn, sortDirection } = useSortableTable(rows, { title: 'string' })

    expect(sortColumn.value).toBeNull()
    expect(sortDirection.value).toBeNull()
  })

  it('first toggle → ascending', () => {
    const rows = ref(makeArtifacts())
    const { sortColumn, sortDirection, toggleSort } = useSortableTable(rows, { title: 'string' })

    toggleSort('title')

    expect(sortColumn.value).toBe('title')
    expect(sortDirection.value).toBe('asc')
  })

  it('second toggle on same column → descending', () => {
    const rows = ref(makeArtifacts())
    const { sortColumn, sortDirection, toggleSort } = useSortableTable(rows, { title: 'string' })

    toggleSort('title')
    toggleSort('title')

    expect(sortColumn.value).toBe('title')
    expect(sortDirection.value).toBe('desc')
  })

  it('third toggle on same column → unsorted (null)', () => {
    const rows = ref(makeArtifacts())
    const { sortColumn, sortDirection, toggleSort } = useSortableTable(rows, { title: 'string' })

    toggleSort('title')
    toggleSort('title')
    toggleSort('title')

    expect(sortColumn.value).toBeNull()
    expect(sortDirection.value).toBeNull()
  })

  it('sortedRows returns original order when unsorted', () => {
    const data = makeArtifacts()
    const rows = ref(data)
    const { sortedRows, toggleSort } = useSortableTable(rows, { title: 'string' })

    // Cycle back to unsorted
    toggleSort('title')
    toggleSort('title')
    toggleSort('title')

    expect(sortedRows.value.map(r => r.title)).toEqual(data.map(r => r.title))
  })
})

// ---------------------------------------------------------------------------
// Column switching
// ---------------------------------------------------------------------------

describe('useSortableTable — column switching', () => {
  it('switching to a new column resets direction to ascending', () => {
    const rows = ref(makeArtifacts())
    const { sortColumn, sortDirection, toggleSort } = useSortableTable(rows, {
      title: 'string',
      created: 'date',
    })

    toggleSort('title')          // title asc
    toggleSort('title')          // title desc
    toggleSort('created')        // new column → created asc

    expect(sortColumn.value).toBe('created')
    expect(sortDirection.value).toBe('asc')
  })

  it('switching columns clears the previous column sort indicator', () => {
    const rows = ref(makeArtifacts())
    const { sortColumn, toggleSort } = useSortableTable(rows, {
      title: 'string',
      created: 'date',
    })

    toggleSort('title')
    toggleSort('created')

    // Only 'created' should be the active column; 'title' is no longer active
    expect(sortColumn.value).toBe('created')
    expect(sortColumn.value).not.toBe('title')
  })
})

// ---------------------------------------------------------------------------
// Sort types — string (case-insensitive)
// ---------------------------------------------------------------------------

describe('useSortableTable — string sort', () => {
  it('sorts ascending: case-insensitive lexicographic order', () => {
    const rows = ref([
      { title: 'Banana' },
      { title: 'apple' },
      { title: 'Cherry' },
    ])
    const { sortedRows, toggleSort } = useSortableTable(rows, { title: 'string' })

    toggleSort('title')

    expect(sortedRows.value.map(r => r.title)).toEqual(['apple', 'Banana', 'Cherry'])
  })

  it('sorts descending: reverse case-insensitive lexicographic order', () => {
    const rows = ref([
      { title: 'Banana' },
      { title: 'apple' },
      { title: 'Cherry' },
    ])
    const { sortedRows, toggleSort } = useSortableTable(rows, { title: 'string' })

    toggleSort('title')
    toggleSort('title')

    expect(sortedRows.value.map(r => r.title)).toEqual(['Cherry', 'Banana', 'apple'])
  })

  it('treats "text" type identically to "string"', () => {
    const rows = ref([
      { status: 'planning' },
      { status: 'approved' },
      { status: 'draft' },
    ])
    const { sortedRows, toggleSort } = useSortableTable(rows, { status: 'text' })

    toggleSort('status')

    expect(sortedRows.value.map(r => r.status)).toEqual(['approved', 'draft', 'planning'])
  })
})

// ---------------------------------------------------------------------------
// Sort types — date
// ---------------------------------------------------------------------------

describe('useSortableTable — date sort', () => {
  it('sorts ascending: chronological order of ISO 8601 strings', () => {
    const rows = ref([
      { created: '2024-03-15T00:00:00Z' },
      { created: '2023-01-01T00:00:00Z' },
      { created: '2025-06-20T00:00:00Z' },
    ])
    const { sortedRows, toggleSort } = useSortableTable(rows, { created: 'date' })

    toggleSort('created')

    const dates = sortedRows.value.map(r => r.created)
    expect(dates[0]).toBe('2023-01-01T00:00:00Z')
    expect(dates[1]).toBe('2024-03-15T00:00:00Z')
    expect(dates[2]).toBe('2025-06-20T00:00:00Z')
  })

  it('sorts descending: most recent first', () => {
    const rows = ref([
      { created: '2024-03-15T00:00:00Z' },
      { created: '2023-01-01T00:00:00Z' },
      { created: '2025-06-20T00:00:00Z' },
    ])
    const { sortedRows, toggleSort } = useSortableTable(rows, { created: 'date' })

    toggleSort('created')
    toggleSort('created')

    const dates = sortedRows.value.map(r => r.created)
    expect(dates[0]).toBe('2025-06-20T00:00:00Z')
    expect(dates[2]).toBe('2023-01-01T00:00:00Z')
  })
})

// ---------------------------------------------------------------------------
// Sort types — number
// ---------------------------------------------------------------------------

describe('useSortableTable — number sort', () => {
  it('sorts numerically, not lexicographically (9 < 10)', () => {
    const rows = ref([
      { count: 9 },
      { count: 10 },
      { count: 2 },
      { count: 100 },
    ])
    const { sortedRows, toggleSort } = useSortableTable(rows, { count: 'number' })

    toggleSort('count')

    expect(sortedRows.value.map(r => r.count)).toEqual([2, 9, 10, 100])
  })

  it('sorts descending numerically', () => {
    const rows = ref([
      { count: 9 },
      { count: 10 },
      { count: 2 },
    ])
    const { sortedRows, toggleSort } = useSortableTable(rows, { count: 'number' })

    toggleSort('count')
    toggleSort('count')

    expect(sortedRows.value.map(r => r.count)).toEqual([10, 9, 2])
  })
})

// ---------------------------------------------------------------------------
// Null/empty handling
// ---------------------------------------------------------------------------

describe('useSortableTable — null and empty value handling', () => {
  it('rows with null values sort to the end (ascending)', () => {
    const rows = ref([
      { title: 'Banana' },
      { title: null as unknown as string },
      { title: 'Apple' },
    ])
    const { sortedRows, toggleSort } = useSortableTable(rows, { title: 'string' })

    toggleSort('title')

    const titles = sortedRows.value.map(r => r.title)
    // Non-null values come first in sorted order
    expect(titles[0]).toBe('Apple')
    expect(titles[1]).toBe('Banana')
    // Null at the end
    expect(titles[2]).toBeNull()
  })

  it('rows with empty-string values sort to the end (ascending)', () => {
    const rows = ref([
      { title: 'Banana' },
      { title: '' },
      { title: 'Apple' },
    ])
    const { sortedRows, toggleSort } = useSortableTable(rows, { title: 'string' })

    toggleSort('title')

    const titles = sortedRows.value.map(r => r.title)
    expect(titles[0]).toBe('Apple')
    expect(titles[1]).toBe('Banana')
    expect(titles[2]).toBe('')
  })

  it('null handling is consistent: nulls remain at end in descending order too', () => {
    const rows = ref([
      { title: 'Banana' },
      { title: null as unknown as string },
      { title: 'Apple' },
    ])
    const { sortedRows, toggleSort } = useSortableTable(rows, { title: 'string' })

    toggleSort('title')  // asc: Apple, Banana, null
    toggleSort('title')  // desc: Banana, Apple, null

    const titles = sortedRows.value.map(r => r.title)
    // Non-null values first (in desc order), null still at end
    expect(titles[0]).toBe('Banana')
    expect(titles[1]).toBe('Apple')
    expect(titles[2]).toBeNull()
  })
})

// ---------------------------------------------------------------------------
// resetSort
// ---------------------------------------------------------------------------

describe('useSortableTable — resetSort()', () => {
  it('clears sortColumn and sortDirection', () => {
    const rows = ref(makeArtifacts())
    const { sortColumn, sortDirection, toggleSort, resetSort } = useSortableTable(
      rows,
      { title: 'string' },
    )

    toggleSort('title')
    resetSort()

    expect(sortColumn.value).toBeNull()
    expect(sortDirection.value).toBeNull()
  })

  it('sortedRows returns original row order after reset', () => {
    const data = makeArtifacts()
    const rows = ref([...data])
    const { sortedRows, toggleSort, resetSort } = useSortableTable(rows, { title: 'string' })

    toggleSort('title')  // sorts alphabetically
    resetSort()

    expect(sortedRows.value.map(r => r.title)).toEqual(data.map(r => r.title))
  })

  it('resetSort is idempotent when already unsorted', () => {
    const rows = ref(makeArtifacts())
    const { sortColumn, sortDirection, resetSort } = useSortableTable(rows, { title: 'string' })

    resetSort()
    resetSort()

    expect(sortColumn.value).toBeNull()
    expect(sortDirection.value).toBeNull()
  })
})

// ---------------------------------------------------------------------------
// Reactivity — source data updates
// ---------------------------------------------------------------------------

describe('useSortableTable — reactivity', () => {
  it('sortedRows updates when source data changes (without re-toggling)', async () => {
    const rows = ref([{ title: 'B' }, { title: 'A' }])
    const { sortedRows, toggleSort } = useSortableTable(rows, { title: 'string' })

    toggleSort('title')  // sort asc: A, B
    expect(sortedRows.value.map(r => r.title)).toEqual(['A', 'B'])

    // Add a new row to source data — no re-toggle needed
    rows.value = [{ title: 'B' }, { title: 'A' }, { title: 'C' }]
    await nextTick()

    expect(sortedRows.value.map(r => r.title)).toEqual(['A', 'B', 'C'])
  })

  it('sortedRows updates when source data changes while unsorted', async () => {
    const rows = ref([{ title: 'B' }, { title: 'A' }])
    const { sortedRows } = useSortableTable(rows, { title: 'string' })

    // No sort active; sortedRows mirrors source
    expect(sortedRows.value.map(r => r.title)).toEqual(['B', 'A'])

    rows.value = [{ title: 'X' }, { title: 'Y' }, { title: 'Z' }]
    await nextTick()

    expect(sortedRows.value.map(r => r.title)).toEqual(['X', 'Y', 'Z'])
  })
})
