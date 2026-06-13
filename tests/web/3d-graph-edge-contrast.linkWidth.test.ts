// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * 3d-graph-edge-contrast — Milestone 3: link width hierarchy
 *
 * Verifies the `linkWidth` callback in ForceGraph3D.vue applies the correct
 * hierarchy across all edge kinds:
 *   timeline (2.0) > semantic edges (1.2) > assigned (0.8)
 *
 * and that every kind meets its specified minimum:
 *   timeline  >= 2.0
 *   parent    >= 1.0
 *   assigned  >= 0.6
 *
 * The callback is captured by mocking `3d-force-graph` and recording the
 * argument passed to `.linkWidth()` during component mount — the same technique
 * used in ForceGraph3D.approvedRing.test.ts for nodeThreeObject.
 *
 * Component: web/src/components/map/ForceGraph3D.vue
 * Test plan: lifecycle/test-plans/3d-graph-edge-contrast-5-test.md  §Milestone 3
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import type { GraphNode, GraphEdge } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Mock: graphConstants — minimal palette so the component can initialise
// ---------------------------------------------------------------------------

vi.mock('@/components/map/graphConstants', async () => {
  const { computed, ref } = await import('vue')
  const palette = {
    nodeColors: {
      idea: '#f59e0b', requirement: '#3b82f6', 'plan-backend': '#8b5cf6',
      'plan-frontend': '#a78bfa', 'plan-test': '#c084fc', test: '#06b6d4',
      prototype: '#14b8a6', defect: '#f43f5e', label: '#a855f7',
      release: '#93c5fd', backlog: '#6b7280',
    },
    priorityColors: { high: '#ef4444', medium: '#f97316', normal: '#22c55e', low: '#3b82f6' },
    activeStatusColors: {
      'in-development': '#4ade80', 'in-qa': '#fbbf24', 'in-progress': '#4ade80',
      clarifying: '#60a5fa', planning: '#a78bfa',
    },
    edgeColors: {
      parent: '#94a3b8', depends_on: '#f97316', blocks: '#ef4444',
      related_to: '#94a3b8', label: '#a855f7',
      timeline: '#3b82f6', assigned: '#475569',
    },
    approvedTestRingColor: '#2563eb',
    canvasBg: '#0f172a',
    labelColor: '#f1f5f9',
    labelNodeBg: '#2e1a4a',
    labelNodeText: '#d8b4fe',
    labelNodeBorder: '#a855f7',
    releaseText: '#1e3a5f',
    releaseBorderColor: '#60a5fa',
    backlogText: '#d1d5db',
    edgeLabelBg: '#1e293b',
    edgeLabelText: '#94a3b8',
    timelineEdgeColor: '#3b82f6',
    timelineEdgeTextColor: '#93c5fd',
    assignedEdgeColor: '#475569',
    borderDefault: 'rgba(255,255,255,0.25)',
    selectedBorderColor: '#ffffff',
    searchHighlight: '#facc15',
    dimBlend: '#1e2535',
  }
  return {
    useGraphTheme: () => ({
      palette: computed(() => palette),
      isDark: ref(true),
    }),
  }
})

// ---------------------------------------------------------------------------
// Mock: THREE — lightweight stubs (component uses THREE for node rendering)
// ---------------------------------------------------------------------------

class MockVector3 {
  x = 1; y = 1; z = 1
  set(x: number, y: number, z: number) { this.x = x; this.y = y; this.z = z; return this }
  setScalar = vi.fn((s: number) => { this.x = s; this.y = s; this.z = s; return this })
}
class MockGroup {
  children: any[] = []
  add(obj: any) { this.children.push(obj); return this }
}
class MockTorusGeometry {
  constructor(public radius: number, public tube: number) {}
}
class MockSphereGeometry {
  constructor(public radius: number) {}
}
class MockMeshLambertMaterial {
  constructor(public params: any = {}) {}
}
class MockSpriteMaterial {
  constructor(public params: any = {}) {}
}
class MockSprite {
  scale = new MockVector3()
  position = new MockVector3()
  constructor(public material: any) {}
}
class MockCanvasTexture {
  constructor(public canvas: any) {}
}
class MockMesh {
  scale = new MockVector3()
  constructor(public geometry: any, public material: any) {}
}

vi.mock('three', () => ({
  Group: MockGroup,
  TorusGeometry: MockTorusGeometry,
  SphereGeometry: MockSphereGeometry,
  MeshLambertMaterial: MockMeshLambertMaterial,
  SpriteMaterial: MockSpriteMaterial,
  Sprite: MockSprite,
  CanvasTexture: MockCanvasTexture,
  Mesh: MockMesh,
}))

// ---------------------------------------------------------------------------
// Mock: 3d-force-graph — captures the linkWidth callback
// ---------------------------------------------------------------------------

const { getLinkWidthCallback, mockGraphInstance } = vi.hoisted(() => {
  let _linkWidthCb: ((l: any) => number) | null = null

  const instance: any = {}
  const fluent = () => vi.fn().mockImplementation(() => instance)

  instance.nodeId = fluent()
  instance.nodeLabel = fluent()
  instance.nodeColor = fluent()
  instance.nodeVal = fluent()
  instance.nodeThreeObjectExtend = fluent()
  instance.nodeThreeObject = fluent()
  instance.linkSource = fluent()
  instance.linkTarget = fluent()
  instance.linkColor = fluent()
  instance.linkLabel = fluent()
  instance.linkWidth = vi.fn().mockImplementation((cb: (l: any) => number) => {
    _linkWidthCb = cb
    return instance
  })
  instance.linkDirectionalArrowLength = fluent()
  instance.linkDirectionalArrowRelPos = fluent()
  instance.linkCurvature = fluent()
  instance.linkOpacity = fluent()
  instance.backgroundColor = fluent()
  instance.showNavInfo = fluent()
  instance.onNodeClick = fluent()
  instance.graphData = fluent()
  instance.onEngineTick = fluent()
  instance.width = fluent()
  instance.height = fluent()
  instance.zoomToFit = fluent()
  instance._destructor = vi.fn()

  return {
    getLinkWidthCallback: () => _linkWidthCb,
    mockGraphInstance: instance,
  }
})

vi.mock('3d-force-graph', () => ({
  default: () => () => mockGraphInstance,
}))

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// Vitest 4 no longer allows `new` on a vi.fn() whose implementation is an arrow
// function (arrows can't be constructors). ForceGraph3D.vue does
// `new ResizeObserver(...)`, so the implementation must be a regular function.
const MockResizeObserver = vi.fn().mockImplementation(function () {
  return {
    observe: vi.fn(),
    unobserve: vi.fn(),
    disconnect: vi.fn(),
  }
})

function makeNode(overrides: Partial<GraphNode> = {}): GraphNode {
  return {
    id: 'n1',
    title: 'Node',
    type: 'requirement',
    status: 'draft',
    stage: 'requirements',
    lineage: 'test-feature',
    slug: 'test-feature',
    index: 2,
    ...overrides,
  }
}

function makeEdge(kind: GraphEdge['kind']): GraphEdge {
  return { source: 'n1', target: 'n2', kind }
}

async function mountForceGraph3D(nodes: GraphNode[] = [], edges: GraphEdge[] = []) {
  vi.stubGlobal('ResizeObserver', MockResizeObserver)
  vi.useFakeTimers()

  const ForceGraph3DVue = (await import('../../web/src/components/map/ForceGraph3D.vue')).default
  const wrapper = mount(ForceGraph3DVue, {
    props: { nodes, edges },
    attachTo: document.body,
  })
  await flushPromises()
  vi.advanceTimersByTime(1100)
  await flushPromises()
  vi.useRealTimers()

  return wrapper
}

/** Call the captured linkWidth callback with a fake edge of the given kind. */
function callLinkWidth(kind: string): number {
  const cb = getLinkWidthCallback()
  if (!cb) throw new Error('linkWidth callback was not registered — is ForceGraph3D mounted?')
  return cb(makeEdge(kind as GraphEdge['kind']))
}

// ---------------------------------------------------------------------------
// Cleanup
// ---------------------------------------------------------------------------

beforeEach(() => {
  mockGraphInstance.linkWidth.mockClear()
  MockResizeObserver.mockClear()
})

afterEach(() => {
  document.body.innerHTML = ''
  vi.clearAllMocks()
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

// ===========================================================================
// Milestone 3 — linkWidth hierarchy
// ===========================================================================

describe('3d-graph-edge-contrast — Milestone 3: linkWidth hierarchy', () => {
  it('linkWidth callback is registered on mount', async () => {
    await mountForceGraph3D([makeNode()])
    expect(getLinkWidthCallback()).toBeTypeOf('function')
  })

  it('timeline edge width >= 2.0', async () => {
    await mountForceGraph3D([makeNode()])
    expect(callLinkWidth('timeline')).toBeGreaterThanOrEqual(2.0)
  })

  it('parent edge width >= 1.0', async () => {
    await mountForceGraph3D([makeNode()])
    expect(callLinkWidth('parent')).toBeGreaterThanOrEqual(1.0)
  })

  it('depends_on edge width >= 1.0', async () => {
    await mountForceGraph3D([makeNode()])
    expect(callLinkWidth('depends_on')).toBeGreaterThanOrEqual(1.0)
  })

  it('blocks edge width >= 1.0', async () => {
    await mountForceGraph3D([makeNode()])
    expect(callLinkWidth('blocks')).toBeGreaterThanOrEqual(1.0)
  })

  it('related_to edge width >= 1.0', async () => {
    await mountForceGraph3D([makeNode()])
    expect(callLinkWidth('related_to')).toBeGreaterThanOrEqual(1.0)
  })

  it('assigned edge width >= 0.6', async () => {
    await mountForceGraph3D([makeNode()])
    expect(callLinkWidth('assigned')).toBeGreaterThanOrEqual(0.6)
  })

  it('hierarchy: timeline > parent (semantic) > assigned', async () => {
    await mountForceGraph3D([makeNode()])
    const wTimeline = callLinkWidth('timeline')
    const wParent   = callLinkWidth('parent')
    const wAssigned = callLinkWidth('assigned')
    expect(wTimeline).toBeGreaterThan(wParent)
    expect(wParent).toBeGreaterThan(wAssigned)
  })

  it('hierarchy: timeline > depends_on > assigned', async () => {
    await mountForceGraph3D([makeNode()])
    const wTimeline   = callLinkWidth('timeline')
    const wDependsOn  = callLinkWidth('depends_on')
    const wAssigned   = callLinkWidth('assigned')
    expect(wTimeline).toBeGreaterThan(wDependsOn)
    expect(wDependsOn).toBeGreaterThan(wAssigned)
  })

  it('semantic edge kinds return the same width (they share a default)', async () => {
    await mountForceGraph3D([makeNode()])
    const kinds = ['parent', 'depends_on', 'blocks', 'related_to', 'label']
    const widths = kinds.map(callLinkWidth)
    const first = widths[0]
    for (const w of widths) {
      expect(w).toBe(first)
    }
  })

  it('linkWidth returns a positive number for an unknown edge kind', async () => {
    await mountForceGraph3D([makeNode()])
    // Unknown kind falls through to the default branch — should not throw or return ≤ 0
    const w = callLinkWidth('unknown_kind' as any)
    expect(w).toBeGreaterThan(0)
  })
})
