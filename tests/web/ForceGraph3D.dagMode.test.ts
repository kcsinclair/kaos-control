// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * ForceGraph3D — DAG mode cycle tolerance.
 *
 * Regression guard for defect `roadmap-3d-graph-dag-cycle`: the Roadmap is the
 * only caller that enables DAG layout (`dag-mode="lr"`) on the shared
 * ForceGraph3D component. 3d-force-graph throws / fails to lay out a cyclic
 * graph under DAG mode unless an `onDagError` handler is registered, so the
 * roadmap's (frequently cyclic) graph collapsed to the origin.
 *
 * The fix registers `graph.onDagError(() => {})` *before* `graph.dagMode(...)`.
 * 3d-force-graph is mocked here, so these tests lock in that the component
 * wires the handler correctly (presence + ordering + tolerance), which is the
 * exact thing a regression would remove.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import type { GraphNode, GraphEdge } from '../../web/src/types/api'

// graphConstants → useGraphTheme reads a Pinia store; stub it so the component
// can initialise without an active Pinia.
vi.mock('@/components/map/graphConstants', async () => {
  const { computed, ref } = await import('vue')
  const palette = {
    nodeColors: { requirement: '#3b82f6', release: '#93c5fd' },
    priorityColors: { high: '#ef4444', medium: '#f97316', normal: '#22c55e', low: '#3b82f6' },
    activeStatusColors: {},
    edgeColors: { related_to: '#94a3b8', timeline: '#3b82f6', assigned: '#475569' },
    approvedTestRingColor: '#2563eb',
    canvasBg: '#0f172a',
    labelColor: '#f1f5f9',
    labelNodeBg: '#2e1a4a', labelNodeText: '#d8b4fe', labelNodeBorder: '#a855f7',
    releaseText: '#1e3a5f', releaseBorderColor: '#60a5fa', backlogText: '#d1d5db',
    edgeLabelBg: '#1e293b', edgeLabelText: '#94a3b8',
    timelineEdgeColor: '#3b82f6', timelineEdgeTextColor: '#93c5fd', assignedEdgeColor: '#475569',
    borderDefault: 'rgba(255,255,255,0.25)', selectedBorderColor: '#ffffff',
    searchHighlight: '#facc15', dimBlend: '#1e2535',
  }
  return { useGraphTheme: () => ({ palette: computed(() => palette), isDark: ref(true) }) }
})

// THREE — lightweight stubs (the component builds node objects with THREE).
class MockVector3 {
  x = 1; y = 1; z = 1
  set(x: number, y: number, z: number) { this.x = x; this.y = y; this.z = z; return this }
  setScalar = vi.fn((s: number) => { this.x = s; this.y = s; this.z = s; return this })
}
class MockGroup { children: any[] = []; add(o: any) { this.children.push(o); return this } }
class MockMesh { scale = new MockVector3(); constructor(public geometry: any, public material: any) {} }
vi.mock('three', () => ({
  Group: MockGroup,
  TorusGeometry: class { constructor(public radius: number, public tube: number) {} },
  SphereGeometry: class { constructor(public radius: number) {} },
  MeshLambertMaterial: class { constructor(public params: any = {}) {} },
  SpriteMaterial: class { constructor(public params: any = {}) {} },
  Sprite: class { scale = new MockVector3(); position = new MockVector3(); constructor(public material: any) {} },
  CanvasTexture: class { constructor(public canvas: any) {} },
  Mesh: MockMesh,
}))

const { mockGraphInstance, getOnDagErrorCb } = vi.hoisted(() => {
  let _onDagErrorCb: ((loop: unknown) => void) | null = null

  const instance: any = {}
  const fluent = () => vi.fn().mockImplementation(() => instance)

  for (const m of [
    'nodeId', 'nodeLabel', 'nodeColor', 'nodeVal', 'nodeThreeObjectExtend',
    'nodeThreeObject', 'linkSource', 'linkTarget', 'linkColor', 'linkLabel',
    'linkWidth', 'linkDirectionalArrowLength', 'linkDirectionalArrowRelPos',
    'linkCurvature', 'linkOpacity', 'backgroundColor', 'showNavInfo',
    'onNodeClick', 'graphData', 'onEngineTick', 'width', 'height', 'zoomToFit',
    'dagMode',
  ]) {
    instance[m] = fluent()
  }
  instance.onDagError = vi.fn().mockImplementation((cb: (loop: unknown) => void) => {
    _onDagErrorCb = cb
    return instance
  })
  instance._destructor = vi.fn()

  return { mockGraphInstance: instance, getOnDagErrorCb: () => _onDagErrorCb }
})

vi.mock('3d-force-graph', () => ({
  default: () => () => mockGraphInstance,
}))

// Regular function (not arrow): ForceGraph3D.vue does `new ResizeObserver(...)`,
// and Vitest 4 won't `new` an arrow-implemented vi.fn().
const MockResizeObserver = vi.fn().mockImplementation(function () {
  return { observe: vi.fn(), unobserve: vi.fn(), disconnect: vi.fn() }
})

function makeNode(id: string): GraphNode {
  return {
    id,
    title: id,
    type: 'requirement',
    status: 'draft',
    stage: 'requirements',
    lineage: 'lin',
    slug: 'lin',
    index: 2,
  }
}

// A 2-cycle: n1 → n2 → n1 (symmetric related_to edges, exactly what the roadmap
// graph produces and what breaks DAG layout without onDagError).
const cyclicNodes = [makeNode('n1'), makeNode('n2')]
const cyclicEdges: GraphEdge[] = [
  { source: 'n1', target: 'n2', kind: 'related_to' },
  { source: 'n2', target: 'n1', kind: 'related_to' },
]

async function mountWith(props: Record<string, unknown>) {
  vi.stubGlobal('ResizeObserver', MockResizeObserver)
  vi.useFakeTimers()
  const ForceGraph3DVue = (await import('../../web/src/components/map/ForceGraph3D.vue')).default
  const wrapper = mount(ForceGraph3DVue, { props, attachTo: document.body })
  await flushPromises()
  vi.advanceTimersByTime(1100) // let the zoomToFit setTimeout fire
  await flushPromises()
  vi.useRealTimers()
  return wrapper
}

beforeEach(() => {
  mockGraphInstance.dagMode.mockClear()
  mockGraphInstance.onDagError.mockClear()
  MockResizeObserver.mockClear()
})

describe('ForceGraph3D — DAG mode cycle tolerance (roadmap-3d-graph-dag-cycle)', () => {
  it('registers onDagError before enabling dagMode when dag-mode is set', async () => {
    await mountWith({ nodes: cyclicNodes, edges: cyclicEdges, dagMode: 'lr' })

    expect(mockGraphInstance.dagMode).toHaveBeenCalledWith('lr')
    expect(mockGraphInstance.onDagError).toHaveBeenCalled()

    // onDagError must be registered BEFORE dagMode, so the handler is in place
    // when dagMode triggers DAG processing.
    const errOrder = mockGraphInstance.onDagError.mock.invocationCallOrder[0]
    const dagOrder = mockGraphInstance.dagMode.mock.invocationCallOrder[0]
    expect(errOrder).toBeLessThan(dagOrder)
  })

  it('registers an onDagError handler that tolerates a cycle (does not throw)', async () => {
    await mountWith({ nodes: cyclicNodes, edges: cyclicEdges, dagMode: 'lr' })

    const cb = getOnDagErrorCb()
    expect(typeof cb).toBe('function')
    // Simulate 3d-force-graph reporting the looped node ids — must be swallowed.
    expect(() => cb!(['n1', 'n2'])).not.toThrow()
  })

  it('does not touch dagMode/onDagError when dag-mode is absent (regular map path)', async () => {
    await mountWith({ nodes: cyclicNodes, edges: cyclicEdges })

    expect(mockGraphInstance.dagMode).not.toHaveBeenCalled()
    expect(mockGraphInstance.onDagError).not.toHaveBeenCalled()
  })
})
