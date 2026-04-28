/**
 * Milestone 5a — Performance tests for `useSortableTable`
 *
 * Validates that sorting large datasets completes within the time budgets
 * specified in the test plan: <100 ms for 1,000 rows, <500 ms for 5,000 rows.
 *
 * These tests use `performance.now()` to measure wall-clock time of the
 * computed sort, exercising all four sort types.
 */

import { describe, it, expect } from 'vitest'
import { ref } from 'vue'
import { useSortableTable } from '../../web/src/composables/useSortableTable'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function generateRows(count: number) {
  return Array.from({ length: count }, (_, i) => ({
    title:   `Artifact ${String(i).padStart(6, '0')}`,
    status:  i % 5 === 0 ? 'done' : i % 3 === 0 ? 'draft' : 'in-development',
    created: new Date(Date.now() - i * 60_000).toISOString(),
    count:   Math.floor(Math.random() * count),
  }))
}

function measureSortMs(
  rows: ReturnType<typeof generateRows>,
  column: string,
  type: 'string' | 'date' | 'number' | 'text',
): number {
  const rowsRef = ref(rows)
  const { sortedRows, toggleSort } = useSortableTable(rowsRef, { [column]: type })

  const t0 = performance.now()
  toggleSort(column)
  // Access the computed to trigger evaluation
  void sortedRows.value.length
  return performance.now() - t0
}

// ---------------------------------------------------------------------------
// 1,000 rows (blocking requirement: <100 ms)
// ---------------------------------------------------------------------------

describe('useSortableTable — performance: 1,000 rows', () => {
  const N = 1_000

  it('string sort completes in under 100 ms', () => {
    const elapsed = measureSortMs(generateRows(N), 'title', 'string')
    expect(elapsed).toBeLessThan(100)
  })

  it('date sort completes in under 100 ms', () => {
    const elapsed = measureSortMs(generateRows(N), 'created', 'date')
    expect(elapsed).toBeLessThan(100)
  })

  it('number sort completes in under 100 ms', () => {
    const elapsed = measureSortMs(generateRows(N), 'count', 'number')
    expect(elapsed).toBeLessThan(100)
  })

  it('text sort completes in under 100 ms', () => {
    const elapsed = measureSortMs(generateRows(N), 'status', 'text')
    expect(elapsed).toBeLessThan(100)
  })
})

// ---------------------------------------------------------------------------
// 5,000 rows (stretch goal: <500 ms — non-blocking)
// ---------------------------------------------------------------------------

describe('useSortableTable — performance: 5,000 rows (stretch goal)', () => {
  const N = 5_000

  it('string sort completes in under 500 ms', () => {
    const elapsed = measureSortMs(generateRows(N), 'title', 'string')
    expect(elapsed).toBeLessThan(500)
  })

  it('date sort completes in under 500 ms', () => {
    const elapsed = measureSortMs(generateRows(N), 'created', 'date')
    expect(elapsed).toBeLessThan(500)
  })

  it('number sort completes in under 500 ms', () => {
    const elapsed = measureSortMs(generateRows(N), 'count', 'number')
    expect(elapsed).toBeLessThan(500)
  })
})
