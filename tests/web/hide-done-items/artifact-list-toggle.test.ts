// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 2 — ArtifactListView toggle logic tests.
 *
 * ArtifactListView computes `visibleItems` as:
 *   showCompleted ? store.items : store.items.filter(r => !TERMINAL_STATUSES.includes(r.status))
 *
 * Since this is an inline computed in a component that has many heavy
 * dependencies (router, websocket, lucide icons), these tests validate the
 * filtering logic directly using Vue's `ref` + `computed` primitives and the
 * shared `TERMINAL_STATUSES` constant — exactly the code the component uses.
 */

import { describe, it, expect, beforeEach } from 'vitest'
import { ref, computed } from 'vue'
import { TERMINAL_STATUSES } from '@/types/api'
import {
  makeArtifactsForAllStatuses,
  makeArtifactRow,
  TERMINAL_STATUSES as TEST_TERMINAL_STATUSES,
  ACTIVE_STATUSES,
} from '../helpers/seed_artifacts'

// ---------------------------------------------------------------------------
// Helper — reproduce the visibleItems computed from ArtifactListView.
// ---------------------------------------------------------------------------
function makeVisibleItems(storeItems: ReturnType<typeof makeArtifactsForAllStatuses>) {
  const items = ref(storeItems)
  const showCompleted = ref(false)
  const visibleItems = computed(() =>
    showCompleted.value
      ? items.value
      : items.value.filter(
          (r) => !(TERMINAL_STATUSES as readonly string[]).includes(r.status),
        ),
  )
  return { items, showCompleted, visibleItems }
}

// ---------------------------------------------------------------------------
// TERMINAL_STATUSES constant
// ---------------------------------------------------------------------------
describe('TERMINAL_STATUSES constant', () => {
  it('contains exactly done, rejected, abandoned', () => {
    expect([...TERMINAL_STATUSES].sort()).toEqual(['abandoned', 'done', 'rejected'])
  })

  it('matches the test helper constant', () => {
    expect([...TERMINAL_STATUSES].sort()).toEqual([...TEST_TERMINAL_STATUSES].sort())
  })
})

// ---------------------------------------------------------------------------
// Default state — showCompleted = false
// ---------------------------------------------------------------------------
describe('ArtifactListView visibleItems — default state (showCompleted=false)', () => {
  let storeItems: ReturnType<typeof makeArtifactsForAllStatuses>

  beforeEach(() => {
    storeItems = makeArtifactsForAllStatuses() // 5 active + 3 terminal
  })

  it('hides all three terminal-status artifacts', () => {
    const { visibleItems } = makeVisibleItems(storeItems)
    const statuses = visibleItems.value.map((r) => r.status)

    expect(statuses).not.toContain('done')
    expect(statuses).not.toContain('rejected')
    expect(statuses).not.toContain('abandoned')
  })

  it('shows all active-status artifacts', () => {
    const { visibleItems } = makeVisibleItems(storeItems)
    const statuses = visibleItems.value.map((r) => r.status)

    for (const active of ACTIVE_STATUSES) {
      expect(statuses).toContain(active)
    }
  })

  it('count matches only the active artifacts (5 of 8)', () => {
    const { visibleItems } = makeVisibleItems(storeItems)
    expect(visibleItems.value.length).toBe(ACTIVE_STATUSES.length)
  })

  it('returns empty list when all artifacts are terminal', () => {
    const terminalOnly = TEST_TERMINAL_STATUSES.map((s) => makeArtifactRow({ status: s }))
    const { visibleItems } = makeVisibleItems(terminalOnly)
    expect(visibleItems.value).toHaveLength(0)
  })

  it('returns all artifacts when none are terminal', () => {
    const activeOnly = ACTIVE_STATUSES.map((s) => makeArtifactRow({ status: s }))
    const { visibleItems } = makeVisibleItems(activeOnly)
    expect(visibleItems.value).toHaveLength(ACTIVE_STATUSES.length)
  })
})

// ---------------------------------------------------------------------------
// Toggle reveals terminal items
// ---------------------------------------------------------------------------
describe('ArtifactListView visibleItems — toggle reveals terminal items', () => {
  it('shows all 8 artifacts after setting showCompleted=true', () => {
    const storeItems = makeArtifactsForAllStatuses()
    const { showCompleted, visibleItems } = makeVisibleItems(storeItems)

    showCompleted.value = true

    expect(visibleItems.value.length).toBe(storeItems.length)
    const statuses = visibleItems.value.map((r) => r.status)
    expect(statuses).toContain('done')
    expect(statuses).toContain('rejected')
    expect(statuses).toContain('abandoned')
  })

  it('count reflects full set (not filtered) when showCompleted=true', () => {
    const storeItems = makeArtifactsForAllStatuses()
    const { showCompleted, visibleItems } = makeVisibleItems(storeItems)

    showCompleted.value = true

    // count must equal the total, not just active
    expect(visibleItems.value.length).toBe(storeItems.length)
  })
})

// ---------------------------------------------------------------------------
// Toggle hides terminal items again
// ---------------------------------------------------------------------------
describe('ArtifactListView visibleItems — toggle hides terminal items again', () => {
  it('re-hides terminal artifacts when showCompleted toggled back to false', () => {
    const storeItems = makeArtifactsForAllStatuses()
    const { showCompleted, visibleItems } = makeVisibleItems(storeItems)

    // show all
    showCompleted.value = true
    expect(visibleItems.value.length).toBe(storeItems.length)

    // hide again
    showCompleted.value = false
    const statuses = visibleItems.value.map((r) => r.status)
    expect(statuses).not.toContain('done')
    expect(statuses).not.toContain('rejected')
    expect(statuses).not.toContain('abandoned')
    expect(visibleItems.value.length).toBe(ACTIVE_STATUSES.length)
  })
})

// ---------------------------------------------------------------------------
// Count reflects visible set
// ---------------------------------------------------------------------------
describe('ArtifactListView visibleItems — count reflects visible set', () => {
  it('visible count is less than total when terminal items exist and toggle is off', () => {
    const storeItems = makeArtifactsForAllStatuses()
    const { visibleItems } = makeVisibleItems(storeItems)

    // visible should be 5, total is 8
    expect(visibleItems.value.length).toBeLessThan(storeItems.length)
  })

  it('visible count equals total when toggle is on', () => {
    const storeItems = makeArtifactsForAllStatuses()
    const { showCompleted, visibleItems } = makeVisibleItems(storeItems)

    showCompleted.value = true
    expect(visibleItems.value.length).toBe(storeItems.length)
  })
})

// ---------------------------------------------------------------------------
// Default resets on each "mount" (new showCompleted ref = false)
// ---------------------------------------------------------------------------
describe('ArtifactListView visibleItems — default state resets per instance', () => {
  it('showCompleted starts false on each new setup (simulated mount)', () => {
    const storeItems = makeArtifactsForAllStatuses()

    // First mount simulation
    const first = makeVisibleItems(storeItems)
    first.showCompleted.value = true
    expect(first.visibleItems.value.length).toBe(storeItems.length)

    // Second mount simulation — fresh refs, toggle should be false
    const second = makeVisibleItems(storeItems)
    expect(second.showCompleted.value).toBe(false)
    expect(second.visibleItems.value.length).toBe(ACTIVE_STATUSES.length)
  })
})
