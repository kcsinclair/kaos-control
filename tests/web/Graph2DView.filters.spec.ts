// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 4 — Integration tests: filter + layout interaction in Graph2DView
 *
 * Covers:
 *   1. With activeLayout "circle", applying a type filter triggers relayout with
 *      { name: 'circle', ... }.
 *   2. With activeLayout "breadthfirst", toggling a status filter triggers relayout
 *      with breadthfirst options.
 *   3. Text search highlight (matchedNodeIds change) does NOT trigger a relayout
 *      (only visual dimming via applySearchHighlight).
 *
 * Testing approach
 * ────────────────
 * Graph2DView watches `[props.nodes, props.edges]` and calls `update()` (which
 * calls `runLayout(false)`) when the prop references change.  This simulates what
 * happens when the parent GraphView passes a freshly-filtered node/edge array from
 * the Pinia store's `augmentedNodes` / `augmentedEdges` computeds.
 *
 * The `matchedNodeIds` prop drives `applySearchHighlight()`, which sets node
 * opacity styles without calling `runLayout()` — so cy.layout() must not be
 * called when only `matchedNodeIds` changes.
 *
 * Component: web/src/components/graph/Graph2DView.vue
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { mount, flushPromises } from '@vue/test-utils'
import type { GraphNode, GraphEdge } from '../../web/src/types/api'
import { useGraphStore } from '../../web/src/stores/graph'

// ---------------------------------------------------------------------------
// Module mocks
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
        one: vi.fn((_event: string, cb: () => void) => cb()),
        run: vi.fn(),
      }
    }),
    nodes: vi.fn(() => ({
      length: _cyNodes.length,
      forEach: (cb: (n: any) => void) => _cyNodes.forEach(cb),
      map: (cb: (n: any) => any) => _cyNodes.map(cb),
      filter: vi.fn(() => ({ length: 0, forEach: vi.fn() })),
      style: vi.fn(),
    })),
    edges: vi.fn(() => ({
      forEach: vi.fn(),
      style: vi.fn(),
    })),
    getElementById: vi.fn(() => ({
      data: vi.fn(() => null),
      remove: vi.fn(),
    })),
    style: vi.fn(() => ({ fromJson: vi.fn(() => ({ update: vi.fn() })) })),
    destroy: vi.fn(),
    fit: vi.fn(),
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

vi.mock('cytoscape', () => ({ default: mockCyConstructor }))
vi.mock('cytoscape-fcose', () => ({ default: { name: 'fcoseMock' } }))
vi.mock('cytoscape-dagre', () => ({ default: { name: 'dagreMock' } }))

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

function makeNode(id: string, type = 'idea', status = 'draft'): GraphNode {
  return {
    id,
    title: `Node ${id}`,
    type,
    status,
    stage: 'ideas',
    lineage: 'test-lineage',
    slug: 'test-lineage',
    index: 1,
  }
}

function makeEdge(source: string, target: string): GraphEdge {
  return { source, target, kind: 'parent' }
}

const ALL_NODES: GraphNode[] = [
  makeNode('n1', 'idea'),
  makeNode('n2', 'requirement'),
  makeNode('n3', 'idea'),
]
const ALL_EDGES: GraphEdge[] = [makeEdge('n1', 'n2')]

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

async function mountGraph2D(nodes: GraphNode[], edges: GraphEdge[]) {
  const { default: Graph2DView } = await import(
    '../../web/src/components/graph/Graph2DView.vue'
  )
  const wrapper = mount(Graph2DView, {
    props: {
      nodes,
      edges,
      onNodeClick: vi.fn(),
      matchedNodeIds: new Set<string>(),
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
  mockCyInstance.layout.mockClear()
  mockCyInstance.stop.mockClear()
  setCyNodes([{ data: vi.fn(() => null), style: vi.fn() }])

  setActivePinia(createPinia())
  store = useGraphStore()
})

afterEach(() => {
  document.body.innerHTML = ''
  vi.clearAllTimers()
})

// ===========================================================================
// 1 — With activeLayout "circle", type-filter relayout uses circle options
// ===========================================================================

describe('Graph2DView — filter relayout uses current layout: circle (Milestone 4 AC1)', () => {
  it('applying a type filter (node prop change) triggers relayout with name "circle"', async () => {
    store.setLayout('circle')

    const wrapper = await mountGraph2D(ALL_NODES, ALL_EDGES)
    layoutCallOptions.length = 0
    mockCyInstance.layout.mockClear()

    // Simulate applying a type filter: parent passes filtered nodes as new prop
    const filteredNodes = ALL_NODES.filter((n) => n.type === 'idea')
    await wrapper.setProps({ nodes: filteredNodes, edges: [] })
    await flushPromises()

    expect(mockCyInstance.layout).toHaveBeenCalled()
    const opts = layoutCallOptions[layoutCallOptions.length - 1]
    expect(opts.name).toBe('circle')
  })

  it('filter relayout with circle does NOT use name "fcose"', async () => {
    store.setLayout('circle')

    const wrapper = await mountGraph2D(ALL_NODES, ALL_EDGES)
    layoutCallOptions.length = 0

    const filteredNodes = ALL_NODES.slice(0, 1)
    await wrapper.setProps({ nodes: filteredNodes, edges: [] })
    await flushPromises()

    const lastOpts = layoutCallOptions[layoutCallOptions.length - 1]
    expect(lastOpts.name).not.toBe('fcose')
  })

  it('filter relayout with circle passes animate: false (update path)', async () => {
    store.setLayout('circle')

    const wrapper = await mountGraph2D(ALL_NODES, ALL_EDGES)
    layoutCallOptions.length = 0

    const filteredNodes = ALL_NODES.filter((n) => n.type === 'idea')
    await wrapper.setProps({ nodes: filteredNodes, edges: [] })
    await flushPromises()

    // update() calls runLayout(false) → animate should be false
    const opts = layoutCallOptions[layoutCallOptions.length - 1]
    expect(opts.animate).toBe(false)
  })
})

// ===========================================================================
// 2 — With activeLayout "breadthfirst", status-filter relayout uses breadthfirst
// ===========================================================================

describe('Graph2DView — filter relayout uses current layout: breadthfirst (Milestone 4 AC2)', () => {
  it('toggling a status filter triggers relayout with name "breadthfirst"', async () => {
    store.setLayout('breadthfirst')

    const wrapper = await mountGraph2D(ALL_NODES, ALL_EDGES)
    layoutCallOptions.length = 0
    mockCyInstance.layout.mockClear()

    // Simulate status filter: parent passes nodes filtered by status
    const filteredNodes = ALL_NODES.filter((n) => n.status === 'draft')
    const filteredEdges = ALL_EDGES.filter(
      (e) => filteredNodes.some((n) => n.id === e.source) &&
             filteredNodes.some((n) => n.id === e.target)
    )
    await wrapper.setProps({ nodes: filteredNodes, edges: filteredEdges })
    await flushPromises()

    expect(mockCyInstance.layout).toHaveBeenCalled()
    const opts = layoutCallOptions[layoutCallOptions.length - 1]
    expect(opts.name).toBe('breadthfirst')
  })

  it('breadthfirst relayout includes spacing options', async () => {
    store.setLayout('breadthfirst')

    const wrapper = await mountGraph2D(ALL_NODES, ALL_EDGES)
    layoutCallOptions.length = 0

    await wrapper.setProps({ nodes: ALL_NODES.slice(0, 2), edges: [] })
    await flushPromises()

    const opts = layoutCallOptions[layoutCallOptions.length - 1]
    expect(opts.name).toBe('breadthfirst')
    // breadthfirst config includes spacingFactor and avoidOverlap
    expect(opts.spacingFactor).toBeDefined()
    expect(opts.avoidOverlap).toBeDefined()
  })
})

// ===========================================================================
// 3 — Text search highlight does NOT trigger a relayout
// ===========================================================================

describe('Graph2DView — searchText highlight does not relayout (Milestone 4 AC3)', () => {
  it('changing matchedNodeIds does not call cy.layout()', async () => {
    store.setLayout('circle')

    const wrapper = await mountGraph2D(ALL_NODES, ALL_EDGES)
    layoutCallOptions.length = 0
    mockCyInstance.layout.mockClear()

    // Simulate a text search match — update matchedNodeIds prop only
    const matched = new Set(['n1', 'n3'])
    await wrapper.setProps({ matchedNodeIds: matched })
    await flushPromises()

    expect(mockCyInstance.layout).not.toHaveBeenCalled()
  })

  it('changing matchedNodeIds does not change layoutCallOptions count', async () => {
    const wrapper = await mountGraph2D(ALL_NODES, ALL_EDGES)
    const countBefore = layoutCallOptions.length

    await wrapper.setProps({ matchedNodeIds: new Set(['n2']) })
    await flushPromises()

    expect(layoutCallOptions.length).toBe(countBefore)
  })

  it('clearing matchedNodeIds (empty set) does not call cy.layout()', async () => {
    const wrapper = await mountGraph2D(ALL_NODES, ALL_EDGES)

    // First set a non-empty matchedNodeIds
    await wrapper.setProps({ matchedNodeIds: new Set(['n1']) })
    await flushPromises()

    layoutCallOptions.length = 0
    mockCyInstance.layout.mockClear()

    // Clear the search
    await wrapper.setProps({ matchedNodeIds: new Set() })
    await flushPromises()

    expect(mockCyInstance.layout).not.toHaveBeenCalled()
  })

  it('nodes prop change and matchedNodeIds change together still uses current layout', async () => {
    store.setLayout('concentric')

    const wrapper = await mountGraph2D(ALL_NODES, ALL_EDGES)
    layoutCallOptions.length = 0

    // Simultaneously: filter nodes AND set search
    const filteredNodes = ALL_NODES.slice(0, 2)
    await wrapper.setProps({
      nodes: filteredNodes,
      edges: [],
      matchedNodeIds: new Set(['n1']),
    })
    await flushPromises()

    // A layout was run (due to node prop change)
    expect(mockCyInstance.layout).toHaveBeenCalled()
    // But it should use the current layout (concentric)
    const opts = layoutCallOptions[layoutCallOptions.length - 1]
    expect(opts.name).toBe('concentric')
  })
})
