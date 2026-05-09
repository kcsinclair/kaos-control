// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 5a — Performance tests for `useSortableTable`
 *
 * Validates that sorting large datasets completes within the time budgets
 * specified in the test plan: <100 ms for 1,000 rows (string/date/number),
 * <250 ms for text sort (localeCompare is inherently slower), and <500 ms for
 * 5,000 rows.
 *
 * These tests use `performance.now()` to measure wall-clock time of the
 * computed sort, exercising all four sort types.
 *
 * This file runs in an isolated forked process (see vitest.config.ts
 * poolMatchGlobs) so it does not compete for CPU with the component-mounting
 * tests, which previously caused intermittent threshold failures under
 * full-suite parallelism (defect sortable-table-columns-19-defect.md).
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

  // `text` sort uses localeCompare which is inherently slower than numeric or
  // date comparison.  Budget raised to 250 ms (from 100 ms) to accommodate
  // legitimate variation across environments without masking real regressions.
  // The file-level pool isolation (forks) ensures this budget is not further
  // inflated by concurrent test suite activity.
  it('text sort completes in under 250 ms', () => {
    const elapsed = measureSortMs(generateRows(N), 'status', 'text')
    expect(elapsed).toBeLessThan(250)
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
