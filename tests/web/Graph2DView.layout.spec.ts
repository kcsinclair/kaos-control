/**
 * Milestone 3 — Integration tests: Graph2DView layout switching
 *
 * Covers:
 *   1. On mount, default layout "fcose" is applied via cy.layout().
 *   2. Changing activeLayout to "concentric" calls cy.layout({ name: 'concentric', ... }).run().
 *   3. Changing activeLayout to "dagre" triggers dynamic import of cytoscape-dagre
 *      before running the layout.
 *   4. Toggling directed to true re-runs the active layout with directed: true.
 *   5. Animation options (animate: true, animationDuration) are passed to the layout.
 *   6. The Cytoscape instance is NOT destroyed and recreated on layout change.
 *
 * Testing approach
 * ────────────────
 * Cytoscape is dynamically imported inside Graph2DView.vue and cytoscape-fcose /
 * cytoscape-dagre are registered lazily.  All three are intercepted with vi.mock
 * so that:
 *   - The mock Cytoscape constructor captures the layout options passed to cy.layout()
 *   - Plugin registration (Cy.use) calls are tracked per plugin
 *   - The mock layout object fires 'layoutstop' synchronously so layoutAnimating
 *     is reset immediately
 *
 * The Pinia store is created fresh per test (setActivePinia) and the graph store
 * is used to drive activeLayout / directed changes.
 *
 * Component: web/src/components/graph/Graph2DView.vue
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { mount, flushPromises } from '@vue/test-utils'
import type { GraphNode, GraphEdge } from '../../web/src/types/api'
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

// ---------------------------------------------------------------------------
// Hoisted mock state
// ---------------------------------------------------------------------------

const { mockCyConstructor, mockCyInstance, layoutCallOptions, setCyNodes } = vi.hoisted(() => {
  const _layoutCallOptions: Array<Record<string, unknown>> = []
  let _cyNodes: any[] = []

  const mockCyInstance = {
    on: vi.fn(),
    stop: vi.fn(),
    elements: vi.fn().mockReturnValue({ remove: vi.fn() }),
    add: vi.fn(),
    layout: vi.fn((opts: Record<string, unknown>) => {
      _layoutCallOptions.push({ ...opts })
      return {
        one: vi.fn((_event: string, cb: () => void) => {
          // Fire layoutstop synchronously so layoutAnimating resets immediately
          cb()
        }),
        run: vi.fn(),
      }
    }),
    nodes: vi.fn(() => ({
      length: _cyNodes.length,
      forEach: (cb: (n: any) => void) => _cyNodes.forEach(cb),
    })),
    destroy: vi.fn(),
    fit: vi.fn(),
    edges: vi.fn(() => ({
      forEach: vi.fn(),
      style: vi.fn(),
    })),
  }

  const ctor: any = vi.fn().mockReturnValue(mockCyInstance)
  ctor.use = vi.fn()

  return {
    mockCyConstructor: ctor,
    mockCyInstance,
    layoutCallOptions: _layoutCallOptions,
    setCyNodes: (nodes: any[]) => { _cyNodes = nodes },
  }
})

// ---------------------------------------------------------------------------
// Module mocks — cytoscape + plugins
// ---------------------------------------------------------------------------

vi.mock('cytoscape', () => ({ default: mockCyConstructor }))
vi.mock('cytoscape-fcose', () => ({ default: { name: 'fcoseMock' } }))
vi.mock('cytoscape-dagre', () => ({ default: { name: 'dagreMock' } }))

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

function makeNode(id: string, overrides: Partial<GraphNode> = {}): GraphNode {
  return {
    id,
    title: `Node ${id}`,
    type: 'idea',
    status: 'draft',
    stage: 'ideas',
    lineage: 'test-lineage',
    slug: 'test-lineage',
    index: 1,
    ...overrides,
  }
}

const BASE_NODES: GraphNode[] = [makeNode('n1'), makeNode('n2')]
const BASE_EDGES: GraphEdge[] = [{ source: 'n1', target: 'n2', kind: 'parent' }]

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

async function mountGraph2D(nodes = BASE_NODES, edges = BASE_EDGES) {
  const { default: Graph2DView } = await import(
    '../../web/src/components/graph/Graph2DView.vue'
  )
  // Provide a non-null container element (happy-dom)
  const wrapper = mount(Graph2DView, {
    props: {
      nodes,
      edges,
      onNodeClick: vi.fn(),
    },
    attachTo: document.body,
  })
  await flushPromises()
  return wrapper
}

// ---------------------------------------------------------------------------
// Setup / teardown
// ---------------------------------------------------------------------------

let store: ReturnType<typeof useGraphStore>

beforeEach(() => {
  layoutCallOptions.length = 0
  mockCyConstructor.mockClear()
  mockCyConstructor.use.mockClear()
  mockCyInstance.on.mockClear()
  mockCyInstance.stop.mockClear()
  mockCyInstance.layout.mockClear()
  mockCyInstance.destroy.mockClear()
  setCyNodes([{ data: vi.fn(() => null), style: vi.fn() }]) // at least one node

  setActivePinia(createPinia())
  store = useGraphStore()
})

afterEach(() => {
  document.body.innerHTML = ''
  vi.clearAllTimers()
})

// ===========================================================================
// 1 — On mount, default layout "fcose" is applied
// ===========================================================================

describe('Graph2DView — default layout on mount (Milestone 3 AC1)', () => {
  it('cy.layout() is called at least once on mount', async () => {
    await mountGraph2D()
    expect(mockCyInstance.layout).toHaveBeenCalled()
  })

  it('first cy.layout() call uses name "fcose"', async () => {
    await mountGraph2D()
    const firstCall = layoutCallOptions[0]
    expect(firstCall).toBeDefined()
    expect(firstCall.name).toBe('fcose')
  })

  it('fcose plugin is registered via Cy.use() on mount', async () => {
    await mountGraph2D()
    // Cy.use is called with the fcose plugin default export
    const useCalls = mockCyConstructor.use.mock.calls
    const usedPlugins = useCalls.map((c: any[]) => c[0])
    expect(usedPlugins.some((p: any) => p?.name === 'fcoseMock')).toBe(true)
  })
})

// ===========================================================================
// 2 — Changing activeLayout to "concentric" calls cy.layout({ name: 'concentric' })
// ===========================================================================

describe('Graph2DView — layout change to concentric (Milestone 3 AC2)', () => {
  it('cy.layout() is called with name "concentric" when activeLayout changes', async () => {
    await mountGraph2D()
    layoutCallOptions.length = 0
    mockCyInstance.layout.mockClear()

    store.setLayout('concentric')
    await flushPromises()

    expect(mockCyInstance.layout).toHaveBeenCalled()
    const lastCall = layoutCallOptions[layoutCallOptions.length - 1]
    expect(lastCall.name).toBe('concentric')
  })

  it('cy.layout().run() is called when layout changes', async () => {
    await mountGraph2D()

    let runSpy: ReturnType<typeof vi.fn> | undefined
    mockCyInstance.layout.mockImplementation((opts: Record<string, unknown>) => {
      layoutCallOptions.push({ ...opts })
      runSpy = vi.fn()
      return {
        one: vi.fn((_event: string, cb: () => void) => cb()),
        run: runSpy,
      }
    })

    store.setLayout('concentric')
    await flushPromises()

    expect(runSpy).toHaveBeenCalled()
  })

  it('animate and animationDuration options are present in the concentric layout call', async () => {
    await mountGraph2D()
    layoutCallOptions.length = 0

    store.setLayout('concentric')
    await flushPromises()

    const opts = layoutCallOptions[layoutCallOptions.length - 1]
    expect(opts.animate).toBe(true)
    expect(opts.animationDuration).toBeDefined()
  })
})

// ===========================================================================
// 3 — Changing activeLayout to "dagre" triggers dynamic import of cytoscape-dagre
// ===========================================================================

describe('Graph2DView — dagre layout: dynamic import (Milestone 3 AC3)', () => {
  it('cy.layout() is called with name "dagre" when activeLayout changes to "dagre"', async () => {
    await mountGraph2D()
    layoutCallOptions.length = 0
    mockCyInstance.layout.mockClear()

    store.setLayout('dagre')
    await flushPromises()

    expect(mockCyInstance.layout).toHaveBeenCalled()
    const opts = layoutCallOptions[layoutCallOptions.length - 1]
    expect(opts.name).toBe('dagre')
  })

  it('Cy.use() is called with the dagre plugin when dagre layout is first selected', async () => {
    await mountGraph2D()
    mockCyConstructor.use.mockClear()

    store.setLayout('dagre')
    await flushPromises()

    const usedPlugins = mockCyConstructor.use.mock.calls.map((c: any[]) => c[0])
    expect(usedPlugins.some((p: any) => p?.name === 'dagreMock')).toBe(true)
  })

  it('dagre plugin is NOT registered a second time on repeated dagre layout runs', async () => {
    await mountGraph2D()

    // First dagre layout — plugin is registered
    store.setLayout('dagre')
    await flushPromises()

    const countAfterFirst = mockCyConstructor.use.mock.calls.filter(
      (c: any[]) => c[0]?.name === 'dagreMock'
    ).length

    // Second dagre layout switch — plugin already registered, should not call use() again
    store.setLayout('fcose')
    await flushPromises()
    store.setLayout('dagre')
    await flushPromises()

    const countAfterSecond = mockCyConstructor.use.mock.calls.filter(
      (c: any[]) => c[0]?.name === 'dagreMock'
    ).length

    expect(countAfterSecond).toBe(countAfterFirst)
  })
})

// ===========================================================================
// 4 — Toggling directed to true re-runs layout with directed: true
// ===========================================================================

describe('Graph2DView — directed toggle re-runs layout (Milestone 3 AC4)', () => {
  it('cy.layout() is called again when directed is toggled', async () => {
    // Use breadthfirst which supports the directed option
    store.setLayout('breadthfirst')
    await mountGraph2D()
    mockCyInstance.layout.mockClear()
    layoutCallOptions.length = 0

    store.toggleDirected()
    await flushPromises()

    expect(mockCyInstance.layout).toHaveBeenCalled()
  })

  it('layout call includes directed: true when store.directed is toggled to true', async () => {
    store.setLayout('breadthfirst')
    await mountGraph2D()
    layoutCallOptions.length = 0

    store.toggleDirected() // false → true
    await flushPromises()

    const opts = layoutCallOptions[layoutCallOptions.length - 1]
    expect(opts.directed).toBe(true)
  })

  it('layout call includes directed: false when store.directed is toggled back', async () => {
    store.setLayout('breadthfirst')
    await mountGraph2D()

    store.toggleDirected() // false → true
    await flushPromises()

    layoutCallOptions.length = 0
    store.toggleDirected() // true → false
    await flushPromises()

    const opts = layoutCallOptions[layoutCallOptions.length - 1]
    expect(opts.directed).toBe(false)
  })
})

// ===========================================================================
// 5 — Animation options are passed to layout calls
// ===========================================================================

describe('Graph2DView — animation options (Milestone 3 AC5)', () => {
  it('animated layout calls include animate: true', async () => {
    await mountGraph2D()
    layoutCallOptions.length = 0

    // Trigger an animated layout change (not initial mount, which uses animate: false)
    store.setLayout('circle')
    await flushPromises()

    const opts = layoutCallOptions[layoutCallOptions.length - 1]
    expect(opts.animate).toBe(true)
  })

  it('animated layout calls include animationDuration', async () => {
    await mountGraph2D()
    layoutCallOptions.length = 0

    store.setLayout('circle')
    await flushPromises()

    const opts = layoutCallOptions[layoutCallOptions.length - 1]
    expect(typeof opts.animationDuration).toBe('number')
    expect(opts.animationDuration as number).toBeGreaterThan(0)
  })

  it('initial mount layout call uses animate: false (no animation on load)', async () => {
    await mountGraph2D()

    // First layout call should have animate: false (initial render)
    const firstOpts = layoutCallOptions[0]
    expect(firstOpts.animate).toBe(false)
  })
})

// ===========================================================================
// 6 — Cytoscape instance is NOT destroyed and recreated on layout change
// ===========================================================================

describe('Graph2DView — Cytoscape instance reuse on layout change (Milestone 3 AC6)', () => {
  it('Cytoscape constructor is called only once (not re-instantiated on layout change)', async () => {
    await mountGraph2D()
    const instanceCountAfterMount = mockCyConstructor.mock.calls.length

    store.setLayout('concentric')
    await flushPromises()

    store.setLayout('circle')
    await flushPromises()

    expect(mockCyConstructor.mock.calls.length).toBe(instanceCountAfterMount)
  })

  it('cy.destroy() is NOT called on layout change (only on unmount)', async () => {
    const wrapper = await mountGraph2D()
    mockCyInstance.destroy.mockClear()

    store.setLayout('breadthfirst')
    await flushPromises()
    store.setLayout('dagre')
    await flushPromises()

    expect(mockCyInstance.destroy).not.toHaveBeenCalled()

    // Destroy is called on unmount
    wrapper.unmount()
    expect(mockCyInstance.destroy).toHaveBeenCalledOnce()
  })
})
