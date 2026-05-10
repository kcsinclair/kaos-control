// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 4 — Store unit tests for label-toggle state in useGraphStore.
 *
 * Verifies that:
 *   - showNodeTitles defaults to false.
 *   - showNodeLineage defaults to false.
 *   - toggleShowNodeTitles() flips the value.
 *   - toggleShowNodeLineage() flips the value.
 *   - The two refs are independent (toggling one does not affect the other).
 *   - State is session-scoped: no localStorage persistence (resets on new pinia).
 *
 * Acceptance criteria (from test plan Milestone 4):
 *   - showNodeTitles defaults to false.
 *   - showNodeLineage defaults to false.
 *   - toggleShowNodeTitles() flips the value.
 *   - toggleShowNodeLineage() flips the value.
 *
 * Store: web/src/stores/graph.ts
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useGraphStore } from '../../web/src/stores/graph'

// ---------------------------------------------------------------------------
// Module mocks — prevent real network I/O
// ---------------------------------------------------------------------------

vi.mock('@/api/graph', () => ({
  getGraph: vi.fn().mockResolvedValue({ nodes: [], edges: [] }),
}))

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: vi.fn(),
}))

// ===========================================================================
// Default state
// ===========================================================================

describe('GraphStore — label-toggle default state (M4)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('showNodeTitles defaults to false', () => {
    const store = useGraphStore()
    expect(store.showNodeTitles).toBe(false)
  })

  it('showNodeLineage defaults to false', () => {
    const store = useGraphStore()
    expect(store.showNodeLineage).toBe(false)
  })

  it('both label refs default to false simultaneously', () => {
    const store = useGraphStore()
    expect(store.showNodeTitles).toBe(false)
    expect(store.showNodeLineage).toBe(false)
  })
})

// ===========================================================================
// toggleShowNodeTitles
// ===========================================================================

describe('GraphStore — toggleShowNodeTitles (M4)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('toggleShowNodeTitles() flips showNodeTitles from false to true', () => {
    const store = useGraphStore()
    expect(store.showNodeTitles).toBe(false)
    store.toggleShowNodeTitles()
    expect(store.showNodeTitles).toBe(true)
  })

  it('toggleShowNodeTitles() flips showNodeTitles from true back to false', () => {
    const store = useGraphStore()
    store.toggleShowNodeTitles()
    expect(store.showNodeTitles).toBe(true)
    store.toggleShowNodeTitles()
    expect(store.showNodeTitles).toBe(false)
  })

  it('toggleShowNodeTitles() called three times ends with showNodeTitles=true', () => {
    const store = useGraphStore()
    store.toggleShowNodeTitles()
    store.toggleShowNodeTitles()
    store.toggleShowNodeTitles()
    expect(store.showNodeTitles).toBe(true)
  })

  it('toggleShowNodeTitles() does not affect showNodeLineage', () => {
    const store = useGraphStore()
    store.toggleShowNodeTitles()
    expect(store.showNodeLineage).toBe(false)
  })
})

// ===========================================================================
// toggleShowNodeLineage
// ===========================================================================

describe('GraphStore — toggleShowNodeLineage (M4)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('toggleShowNodeLineage() flips showNodeLineage from false to true', () => {
    const store = useGraphStore()
    expect(store.showNodeLineage).toBe(false)
    store.toggleShowNodeLineage()
    expect(store.showNodeLineage).toBe(true)
  })

  it('toggleShowNodeLineage() flips showNodeLineage from true back to false', () => {
    const store = useGraphStore()
    store.toggleShowNodeLineage()
    expect(store.showNodeLineage).toBe(true)
    store.toggleShowNodeLineage()
    expect(store.showNodeLineage).toBe(false)
  })

  it('toggleShowNodeLineage() called three times ends with showNodeLineage=true', () => {
    const store = useGraphStore()
    store.toggleShowNodeLineage()
    store.toggleShowNodeLineage()
    store.toggleShowNodeLineage()
    expect(store.showNodeLineage).toBe(true)
  })

  it('toggleShowNodeLineage() does not affect showNodeTitles', () => {
    const store = useGraphStore()
    store.toggleShowNodeLineage()
    expect(store.showNodeTitles).toBe(false)
  })
})

// ===========================================================================
// Independence — toggling one ref does not affect the other
// ===========================================================================

describe('GraphStore — label-toggle independence (M4)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('showNodeTitles and showNodeLineage can be toggled independently', () => {
    const store = useGraphStore()
    store.toggleShowNodeTitles()
    expect(store.showNodeTitles).toBe(true)
    expect(store.showNodeLineage).toBe(false)

    store.toggleShowNodeLineage()
    expect(store.showNodeTitles).toBe(true)
    expect(store.showNodeLineage).toBe(true)

    store.toggleShowNodeTitles()
    expect(store.showNodeTitles).toBe(false)
    expect(store.showNodeLineage).toBe(true)
  })

  it('toggling hideTerminal and hideTests does not affect label toggles', () => {
    const store = useGraphStore()
    store.toggleHideTerminal()
    store.toggleHideTests()
    expect(store.showNodeTitles).toBe(false)
    expect(store.showNodeLineage).toBe(false)
  })
})

// ===========================================================================
// Session scope — state resets on a fresh pinia (no localStorage persistence)
// ===========================================================================

describe('GraphStore — label-toggle session scope (M4)', () => {
  it('showNodeTitles resets to false when a new pinia is created', () => {
    // First session: user enables titles
    setActivePinia(createPinia())
    const store1 = useGraphStore()
    store1.toggleShowNodeTitles()
    expect(store1.showNodeTitles).toBe(true)

    // New page load / new pinia — state should not carry over
    setActivePinia(createPinia())
    const store2 = useGraphStore()
    expect(store2.showNodeTitles).toBe(false)
  })

  it('showNodeLineage resets to false when a new pinia is created', () => {
    setActivePinia(createPinia())
    const store1 = useGraphStore()
    store1.toggleShowNodeLineage()
    expect(store1.showNodeLineage).toBe(true)

    setActivePinia(createPinia())
    const store2 = useGraphStore()
    expect(store2.showNodeLineage).toBe(false)
  })
})

// ===========================================================================
// Both methods are exposed on the store return object
// ===========================================================================

describe('GraphStore — label-toggle methods are exported (M4)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('toggleShowNodeTitles is a callable function on the store', () => {
    const store = useGraphStore()
    expect(typeof store.toggleShowNodeTitles).toBe('function')
  })

  it('toggleShowNodeLineage is a callable function on the store', () => {
    const store = useGraphStore()
    expect(typeof store.toggleShowNodeLineage).toBe('function')
  })
})
