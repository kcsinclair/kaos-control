/**
 * Milestone 2 — Store unit tests for layout state in useGraphStore
 *
 * Covers:
 *   1. `activeLayout` defaults to 'fcose'.
 *   2. `directed` defaults to false.
 *   3. `setLayout('breadthfirst')` updates `activeLayout`.
 *   4. `setLayout('invalid-key')` does not change state (validation).
 *   5. `toggleDirected()` flips the `directed` boolean.
 *   6. Layout state persists across simulated route changes (store singleton
 *      is not destroyed when the component re-renders).
 *
 * Testing approach
 * ────────────────
 * Pure Pinia store tests — no DOM rendering.  A fresh pinia is created per
 * describe block (some tests share a pinia within the same group to simulate
 * persistence).  `rawNodes` / `rawEdges` are not relevant here; only the
 * layout slice of the store is exercised.
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
// 1 & 2 — Default state
// ===========================================================================

describe('GraphStore — layout defaults (Milestone 2 AC1 & AC2)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('1. activeLayout defaults to "fcose"', () => {
    const store = useGraphStore()
    expect(store.activeLayout).toBe('fcose')
  })

  it('2. directed defaults to false', () => {
    const store = useGraphStore()
    expect(store.directed).toBe(false)
  })

  it('layoutAnimating defaults to false', () => {
    const store = useGraphStore()
    expect(store.layoutAnimating).toBe(false)
  })
})

// ===========================================================================
// 3 — setLayout updates activeLayout for valid keys
// ===========================================================================

describe('GraphStore — setLayout with valid keys (Milestone 2 AC3)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('3a. setLayout("breadthfirst") updates activeLayout to "breadthfirst"', () => {
    const store = useGraphStore()
    store.setLayout('breadthfirst')
    expect(store.activeLayout).toBe('breadthfirst')
  })

  it('3b. setLayout("concentric") updates activeLayout to "concentric"', () => {
    const store = useGraphStore()
    store.setLayout('concentric')
    expect(store.activeLayout).toBe('concentric')
  })

  it('3c. setLayout("circle") updates activeLayout to "circle"', () => {
    const store = useGraphStore()
    store.setLayout('circle')
    expect(store.activeLayout).toBe('circle')
  })

  it('3d. setLayout("dagre") updates activeLayout to "dagre"', () => {
    const store = useGraphStore()
    store.setLayout('dagre')
    expect(store.activeLayout).toBe('dagre')
  })

  it('3e. setLayout("fcose") keeps activeLayout as "fcose" (idempotent)', () => {
    const store = useGraphStore()
    store.setLayout('fcose')
    expect(store.activeLayout).toBe('fcose')
  })
})

// ===========================================================================
// 4 — setLayout rejects invalid keys
// ===========================================================================

describe('GraphStore — setLayout with invalid keys (Milestone 2 AC4)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('4a. setLayout("invalid-key") does not change activeLayout', () => {
    const store = useGraphStore()
    store.setLayout('invalid-key')
    expect(store.activeLayout).toBe('fcose')
  })

  it('4b. setLayout("") does not change activeLayout', () => {
    const store = useGraphStore()
    store.setLayout('')
    expect(store.activeLayout).toBe('fcose')
  })

  it('4c. setLayout("FCOSE") (wrong case) does not change activeLayout', () => {
    const store = useGraphStore()
    store.setLayout('FCOSE')
    expect(store.activeLayout).toBe('fcose')
  })

  it('4d. setLayout("force-directed") does not change activeLayout', () => {
    const store = useGraphStore()
    store.setLayout('force-directed')
    expect(store.activeLayout).toBe('fcose')
  })
})

// ===========================================================================
// 5 — toggleDirected flips the directed boolean
// ===========================================================================

describe('GraphStore — toggleDirected (Milestone 2 AC5)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('5a. toggleDirected() changes directed from false to true', () => {
    const store = useGraphStore()
    expect(store.directed).toBe(false)
    store.toggleDirected()
    expect(store.directed).toBe(true)
  })

  it('5b. toggleDirected() changes directed from true back to false', () => {
    const store = useGraphStore()
    store.toggleDirected()
    expect(store.directed).toBe(true)
    store.toggleDirected()
    expect(store.directed).toBe(false)
  })

  it('5c. toggleDirected() called three times ends with directed=true', () => {
    const store = useGraphStore()
    store.toggleDirected()
    store.toggleDirected()
    store.toggleDirected()
    expect(store.directed).toBe(true)
  })
})

// ===========================================================================
// 6 — State persists across simulated route changes (store not destroyed)
// ===========================================================================

describe('GraphStore — layout state persistence (Milestone 2 AC6)', () => {
  it('6. activeLayout persists when the same pinia instance is used across route changes', () => {
    // Simulate: user navigates to graph view, changes layout, navigates away,
    // navigates back.  The Pinia store is a singleton for the lifetime of the
    // app — it is NOT re-created on route change, so the layout choice persists.
    const pinia = createPinia()
    setActivePinia(pinia)

    // Simulate first visit — user changes layout
    const storeFirstVisit = useGraphStore()
    storeFirstVisit.setLayout('dagre')
    expect(storeFirstVisit.activeLayout).toBe('dagre')

    // Simulate route change — navigate away (store is NOT destroyed)
    // No action needed; Pinia stores are singletons.

    // Simulate return — same pinia, same store instance
    const storeSecondVisit = useGraphStore()
    expect(storeSecondVisit.activeLayout).toBe('dagre')

    // directed flag also persists
    storeFirstVisit.toggleDirected()
    expect(storeSecondVisit.directed).toBe(true)
  })

  it('6b. directed persists across re-access of the same store', () => {
    const pinia = createPinia()
    setActivePinia(pinia)

    const store = useGraphStore()
    store.toggleDirected()
    expect(store.directed).toBe(true)

    // Re-access via useGraphStore() returns the same singleton
    const sameStore = useGraphStore()
    expect(sameStore.directed).toBe(true)
  })
})
