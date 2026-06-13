// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 3 — Component tests for ForceGraph3D.vue — approved-test blue ring
 *
 * Covers:
 *   1. buildNodeObject returns a group with a torus mesh coloured
 *      APPROVED_TEST_RING_COLOR for type='test', status='approved' nodes.
 *   2. The approved-test torus is NOT added to activeRings (no scale animation).
 *   3. Approved test nodes that also have a priority get two torus meshes at
 *      different radii (priority ring + approved-test ring).
 *   4. A non-approved test node (e.g. in-qa) gets no blue torus.
 *   5. A non-test approved node (e.g. requirement/approved) gets no blue torus.
 *
 * Testing approach
 * ───────────────
 * buildNodeObject is a private function inside ForceGraph3D.vue's <script setup>.
 * We access it indirectly by mocking the '3d-force-graph' library and capturing
 * the callback registered via .nodeThreeObject(...).  Calling that callback with
 * a test GraphNode invokes the real buildNodeObject implementation.
 *
 * Three.js is mocked with lightweight class stand-ins whose constructors record
 * arguments, allowing assertions on geometry type, torus radius, and material colour
 * without requiring a real WebGL context.
 *
 * The onEngineTick callback is also captured so test 2 can verify that the
 * approved-test torus is not animated (scale.setScalar is not called on it).
 *
 * Component: web/src/components/graph/ForceGraph3D.vue
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import type { GraphNode, GraphEdge } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Sentinel value
// ---------------------------------------------------------------------------

const APPROVED_TEST_RING_COLOR = '#2563eb'

// ---------------------------------------------------------------------------
// Mock: graphConstants
// ---------------------------------------------------------------------------

vi.mock('@/components/map/graphConstants', async () => {
  const { computed, ref } = await import('vue')
  return {
    useGraphTheme: () => ({
      palette: computed(() => ({
        nodeColors: {
          idea: '#f59e0b',
          requirement: '#3b82f6',
          'plan-backend': '#8b5cf6',
          'plan-frontend': '#a78bfa',
          'plan-test': '#c084fc',
          test: '#06b6d4',
          prototype: '#14b8a6',
          defect: '#f43f5e',
          label: '#a855f7',
          release: '#93c5fd',
          backlog: '#6b7280',
        },
        priorityColors: { high: '#ef4444', medium: '#f97316', normal: '#22c55e', low: '#3b82f6' },
        activeStatusColors: {
          'in-development': '#4ade80',
          'in-qa': '#fbbf24',
          'in-progress': '#4ade80',
          clarifying: '#60a5fa',
          planning: '#a78bfa',
        },
        edgeColors: {
          parent: '#94a3b8',
          depends_on: '#f97316',
          blocks: '#ef4444',
          related_to: '#64748b',
          label: '#a855f7',
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
        assignedEdgeColor: '#334155',
        borderDefault: 'rgba(255,255,255,0.25)',
        selectedBorderColor: '#ffffff',
        searchHighlight: '#facc15',
        dimBlend: '#1e2535',
      })),
      isDark: ref(true),
    }),
  }
})

// ---------------------------------------------------------------------------
// Mock: THREE — minimal stand-ins that record constructor arguments
// ---------------------------------------------------------------------------

// Shared registry — reset between tests
let _createdMeshes: MockMesh[] = []
let _createdTorusGeometries: MockTorusGeometry[] = []

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
  readonly isTorusGeometry = true
  constructor(
    public radius: number,
    public tube: number,
    public radialSegments: number,
    public tubularSegments: number,
  ) {
    _createdTorusGeometries.push(this)
  }
}

class MockSphereGeometry {
  readonly isSphereGeometry = true
  constructor(public radius: number) {}
}

class MockMeshLambertMaterial {
  readonly isMeshLambertMaterial = true
  readonly color: string | number | undefined
  readonly transparent: boolean
  readonly opacity: number
  constructor(params: { color?: string | number; transparent?: boolean; opacity?: number } = {}) {
    this.color = params.color
    this.transparent = params.transparent ?? false
    this.opacity = params.opacity ?? 1
  }
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
  readonly scale = new MockVector3()
  constructor(public geometry: any, public material: any) {
    _createdMeshes.push(this)
  }
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
// Mock: 3d-force-graph — captures nodeThreeObject and onEngineTick callbacks
// ---------------------------------------------------------------------------

const { getNodeThreeObjectCallback, getOnEngineTickCallback, mockGraphInstance } = vi.hoisted(() => {
  let _nodeThreeObjectCb: ((n: any) => any) | null = null
  let _onEngineTickCb: (() => void) | null = null

  // Build the fluent mock instance using explicit mockReturnValue(instance)
  // rather than mockReturnThis() — the latter relies on 'this' binding which
  // can be lost when Vitest's proxy layer wraps the call.
  const instance: any = {}
  const fluent = () => vi.fn().mockImplementation(() => instance)

  instance.nodeId = fluent()
  instance.nodeLabel = fluent()
  instance.nodeColor = fluent()
  instance.nodeVal = fluent()
  instance.nodeThreeObjectExtend = fluent()
  instance.nodeThreeObject = vi.fn().mockImplementation((cb: (n: any) => any) => {
    _nodeThreeObjectCb = cb
    return instance
  })
  instance.linkSource = fluent()
  instance.linkTarget = fluent()
  instance.linkColor = fluent()
  instance.linkLabel = fluent()
  instance.linkWidth = fluent()
  instance.linkOpacity = fluent()
  instance.linkDirectionalArrowLength = fluent()
  instance.linkDirectionalArrowRelPos = fluent()
  instance.linkCurvature = fluent()
  instance.backgroundColor = fluent()
  instance.showNavInfo = fluent()
  instance.onNodeClick = fluent()
  instance.graphData = fluent()
  instance.onEngineTick = vi.fn().mockImplementation((cb: () => void) => {
    _onEngineTickCb = cb
    return instance
  })
  instance.width = fluent()
  instance.height = fluent()
  instance.zoomToFit = fluent()
  instance._destructor = vi.fn()

  return {
    getNodeThreeObjectCallback: () => _nodeThreeObjectCb,
    getOnEngineTickCallback: () => _onEngineTickCb,
    mockGraphInstance: instance,
  }
})

vi.mock('3d-force-graph', () => ({
  default: () => () => mockGraphInstance,
}))

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

function makeNode(overrides: Partial<GraphNode> = {}): GraphNode {
  return {
    id: 'test-node-1',
    title: 'Some test artifact',
    type: 'test',
    status: 'approved',
    stage: 'tests',
    lineage: 'some-feature',
    slug: 'some-feature',
    index: 2,
    ...overrides,
  }
}

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

// ResizeObserver is not available in happy-dom — provide a no-op stub.
// Vitest 4 no longer allows `new` on a vi.fn() whose implementation is an arrow
// function (arrows can't be constructors), so use a regular function.
const MockResizeObserver = vi.fn().mockImplementation(function () {
  return {
    observe: vi.fn(),
    unobserve: vi.fn(),
    disconnect: vi.fn(),
  }
})

async function mountForceGraph3D(nodes: GraphNode[] = [], edges: GraphEdge[] = []) {
  // Stub ResizeObserver before mounting
  vi.stubGlobal('ResizeObserver', MockResizeObserver)
  // Stub setTimeout so zoomToFit delay does not interfere
  vi.useFakeTimers()

  const ForceGraph3DVue = (await import('../../web/src/components/map/ForceGraph3D.vue')).default
  const wrapper = mount(ForceGraph3DVue, {
    props: { nodes, edges },
    attachTo: document.body,
  })
  await flushPromises()

  // Advance past the 1000 ms zoomToFit timeout
  vi.advanceTimersByTime(1100)
  await flushPromises()

  vi.useRealTimers()

  return wrapper
}

/** Invoke buildNodeObject indirectly via the captured nodeThreeObject callback. */
function callBuildNodeObject(node: GraphNode): MockGroup {
  const cb = getNodeThreeObjectCallback()
  if (!cb) throw new Error('nodeThreeObject callback was not registered — is ForceGraph3D mounted?')
  return cb(node) as MockGroup
}

/** All torus meshes within a group (recursively). */
function torusMeshesIn(group: MockGroup): MockMesh[] {
  return group.children.filter(
    (c): c is MockMesh => c instanceof MockMesh && c.geometry instanceof MockTorusGeometry,
  )
}

/** Torus meshes whose material colour matches the given colour string. */
function torusMeshesWithColor(group: MockGroup, color: string): MockMesh[] {
  return torusMeshesIn(group).filter(
    (m) => (m.material as MockMeshLambertMaterial).color === color,
  )
}

// ---------------------------------------------------------------------------
// Cleanup
// ---------------------------------------------------------------------------

beforeEach(() => {
  _createdMeshes = []
  _createdTorusGeometries = []
  MockResizeObserver.mockClear()
  mockGraphInstance.nodeThreeObject.mockClear()
  mockGraphInstance.onEngineTick.mockClear()
})

afterEach(() => {
  document.body.innerHTML = ''
  // vi.clearAllMocks() resets call history only; vi.restoreAllMocks() would
  // also strip the mockImplementation from vi.fn() instances inside mockGraphInstance,
  // causing nodeId() and friends to return undefined on subsequent tests.
  vi.clearAllMocks()
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

// ===========================================================================
// Test 1 — Approved test node gets a static torus with APPROVED_TEST_RING_COLOR
// ===========================================================================

describe('ForceGraph3D — approved-test torus ring', () => {
  it('buildNodeObject returns a group containing at least one torus mesh', async () => {
    await mountForceGraph3D([makeNode()])
    const group = callBuildNodeObject(makeNode())
    const tori = torusMeshesIn(group)
    expect(tori.length).toBeGreaterThan(0)
  })

  it('the torus mesh has material colour matching APPROVED_TEST_RING_COLOR', async () => {
    await mountForceGraph3D([makeNode()])
    const group = callBuildNodeObject(makeNode())
    const blueTori = torusMeshesWithColor(group, APPROVED_TEST_RING_COLOR)
    expect(
      blueTori.length,
      `expected a torus with colour ${APPROVED_TEST_RING_COLOR} in the group`,
    ).toBeGreaterThan(0)
  })

  it('the approved-test torus uses TorusGeometry (not SphereGeometry)', async () => {
    await mountForceGraph3D([makeNode()])
    const group = callBuildNodeObject(makeNode())
    const blueTori = torusMeshesWithColor(group, APPROVED_TEST_RING_COLOR)
    expect(blueTori[0].geometry).toBeInstanceOf(MockTorusGeometry)
  })

  it('the approved-test torus material is not transparent', async () => {
    await mountForceGraph3D([makeNode()])
    const group = callBuildNodeObject(makeNode())
    const blueTori = torusMeshesWithColor(group, APPROVED_TEST_RING_COLOR)
    const mat = blueTori[0].material as MockMeshLambertMaterial
    expect(mat.transparent).toBeFalsy()
  })
})

// ===========================================================================
// Test 2 — Approved-test torus is NOT in activeRings (no animation)
// ===========================================================================

describe('ForceGraph3D — approved-test torus is not animated', () => {
  it('onEngineTick does not call scale.setScalar on the approved-test torus', async () => {
    const wrapper = await mountForceGraph3D([makeNode()])

    const group = callBuildNodeObject(makeNode())
    const blueTori = torusMeshesWithColor(group, APPROVED_TEST_RING_COLOR)
    expect(blueTori.length).toBeGreaterThan(0)

    const approvedTestTorus = blueTori[0]

    // Fire the engine tick — it will iterate activeRings and call scale.setScalar
    // on every mesh in that map.  If our torus was added to activeRings it would
    // be called here.
    const tickCb = getOnEngineTickCallback()
    if (tickCb) tickCb()

    expect(
      approvedTestTorus.scale.setScalar,
      'approved-test torus must not be animated via activeRings',
    ).not.toHaveBeenCalled()

    wrapper.unmount()
  })
})

// ===========================================================================
// Test 3 — Approved test with priority gets two torus rings
// ===========================================================================

describe('ForceGraph3D — approved test with priority gets two rings', () => {
  it('group contains two torus meshes when priority is set', async () => {
    await mountForceGraph3D([makeNode({ priority: 'high' })])
    const group = callBuildNodeObject(makeNode({ priority: 'high' }))
    const tori = torusMeshesIn(group)
    expect(tori.length, 'expected two torus meshes (priority + approved-test)').toBe(2)
  })

  it('one torus has the priority colour and one has the approved-test colour', async () => {
    await mountForceGraph3D([makeNode({ priority: 'high' })])
    const group = callBuildNodeObject(makeNode({ priority: 'high' }))

    const blueTori = torusMeshesWithColor(group, APPROVED_TEST_RING_COLOR)
    const redTori = torusMeshesWithColor(group, '#ef4444') // high priority colour
    expect(blueTori.length, 'expected one blue approved-test torus').toBe(1)
    expect(redTori.length, 'expected one red priority torus').toBe(1)
  })

  it('the two torus rings have different radii', async () => {
    await mountForceGraph3D([makeNode({ priority: 'high' })])
    const group = callBuildNodeObject(makeNode({ priority: 'high' }))
    const tori = torusMeshesIn(group)
    // Prerequisite: two tori must exist before checking their radii
    expect(tori.length, 'need two torus meshes to compare radii').toBe(2)
    const [r1, r2] = tori.map((m) => (m.geometry as MockTorusGeometry).radius)
    expect(r1).not.toBe(r2)
  })
})

// ===========================================================================
// Test 4 — Non-approved test node gets no blue torus
// ===========================================================================

describe('ForceGraph3D — non-approved test node gets no blue torus', () => {
  it('type=test, status=in-qa produces no approved-test coloured torus', async () => {
    await mountForceGraph3D([makeNode({ status: 'in-qa' })])
    const group = callBuildNodeObject(makeNode({ status: 'in-qa' }))
    const blueTori = torusMeshesWithColor(group, APPROVED_TEST_RING_COLOR)
    expect(blueTori.length, 'expected no blue torus for an in-qa test node').toBe(0)
  })

  it('type=test, status=draft produces no approved-test coloured torus', async () => {
    await mountForceGraph3D([makeNode({ status: 'draft' })])
    const group = callBuildNodeObject(makeNode({ status: 'draft' }))
    const blueTori = torusMeshesWithColor(group, APPROVED_TEST_RING_COLOR)
    expect(blueTori.length).toBe(0)
  })
})

// ===========================================================================
// Test 5 — Non-test approved node gets no blue torus
// ===========================================================================

describe('ForceGraph3D — non-test approved node gets no blue torus', () => {
  it('type=requirement, status=approved produces no approved-test coloured torus', async () => {
    await mountForceGraph3D([makeNode({ type: 'requirement' })])
    const group = callBuildNodeObject(makeNode({ type: 'requirement' }))
    const blueTori = torusMeshesWithColor(group, APPROVED_TEST_RING_COLOR)
    expect(blueTori.length, 'expected no blue torus for an approved requirement node').toBe(0)
  })

  it('type=defect, status=approved produces no approved-test coloured torus', async () => {
    await mountForceGraph3D([makeNode({ type: 'defect' })])
    const group = callBuildNodeObject(makeNode({ type: 'defect' }))
    const blueTori = torusMeshesWithColor(group, APPROVED_TEST_RING_COLOR)
    expect(blueTori.length).toBe(0)
  })
})
