// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 3 — Component tests for ForceGraph3D.vue — node label sprites.
 *
 * Tests that buildNodeObject produces the correct Three.js Group structure
 * (sprite count, positions, and text content) under different combinations
 * of the showNodeTitles and showNodeLineage props.
 *
 * Testing approach
 * ────────────────
 * buildNodeObject is a private function inside ForceGraph3D.vue's <script setup>.
 * We access it indirectly by mocking '3d-force-graph' and capturing the callback
 * registered via .nodeThreeObject(...).  Calling that callback with a GraphNode
 * invokes the real buildNodeObject implementation.
 *
 * Three.js is mocked with lightweight stand-ins.  Because happy-dom does not
 * implement CanvasRenderingContext2D, we spy on document.createElement in
 * beforeEach to return a fake canvas with a fake context.  fillText() calls are
 * captured in a shared array so individual tests can assert on text content.
 *
 * Acceptance criteria (from test plan Milestone 3):
 *   - Both props false → non-label/non-release nodes produce no title/lineage sprites.
 *   - showNodeTitles=true → title sprite added; text is truncated to 15 chars + ….
 *   - showNodeLineage=true → lineage sprite added; text matches node.lineage.
 *   - Both true → two sprites at different y-offsets.
 *   - Release nodes (type='release') are unaffected by either prop.
 *   - Label nodes (type='label') are unaffected by either prop.
 *
 * Component: web/src/components/map/ForceGraph3D.vue
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import type { GraphNode, GraphEdge } from '../../web/src/types/api'

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
        activeStatusColors: {},
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
// Mock: THREE — lightweight stand-ins
// ---------------------------------------------------------------------------

let _createdSprites: MockSprite[] = []

class MockVector3 {
  x = 0; y = 0; z = 0
  set(x: number, y: number, z: number) { this.x = x; this.y = y; this.z = z; return this }
  setScalar = vi.fn()
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
  ) {}
}

class MockOctahedronGeometry {
  constructor(public radius: number) {}
}

class MockBoxGeometry {
  constructor(public w: number, public h: number, public d: number) {}
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
  constructor(public material: MockSpriteMaterial) {
    _createdSprites.push(this)
  }
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
  OctahedronGeometry: MockOctahedronGeometry,
  BoxGeometry: MockBoxGeometry,
  MeshLambertMaterial: MockMeshLambertMaterial,
  SpriteMaterial: MockSpriteMaterial,
  Sprite: MockSprite,
  CanvasTexture: MockCanvasTexture,
  Mesh: MockMesh,
}))

// ---------------------------------------------------------------------------
// Mock: 3d-force-graph — captures nodeThreeObject callback
// ---------------------------------------------------------------------------

const { getNodeThreeObjectCallback, mockGraphInstance } = vi.hoisted(() => {
  let _nodeThreeObjectCb: ((n: any) => any) | null = null

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
  instance.onEngineTick = fluent()
  instance.dagMode = fluent()
  instance.width = fluent()
  instance.height = fluent()
  instance.zoomToFit = fluent()
  instance._destructor = vi.fn()

  return {
    getNodeThreeObjectCallback: () => _nodeThreeObjectCb,
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
    id: 'node-1',
    title: 'My Feature Plan',   // exactly 15 chars
    type: 'plan-backend',
    status: 'in-development',
    stage: 'backend-plans',
    lineage: 'my-feature',
    slug: 'my-feature',
    index: 3,
    ...overrides,
  }
}

const MockResizeObserver = vi.fn().mockImplementation(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
}))

// ---------------------------------------------------------------------------
// Canvas mock — installed globally so textSprite() never hits null context
// ---------------------------------------------------------------------------

/** fillText() calls collected across the current test. */
let capturedTexts: string[] = []
let createElementSpy: ReturnType<typeof vi.spyOn> | null = null

function installCanvasMock() {
  const origCreateElement = document.createElement.bind(document)
  createElementSpy = vi.spyOn(document, 'createElement').mockImplementation((tag: string) => {
    if (tag !== 'canvas') return origCreateElement(tag)
    const fakeCtx = {
      font: '',
      fillStyle: '',
      textBaseline: '',
      measureText: (_t: string) => ({ width: 100 }),
      fillText: (text: string) => { capturedTexts.push(text) },
    }
    return {
      getContext: () => fakeCtx,
      width: 0,
      height: 0,
    } as unknown as HTMLCanvasElement
  })
}

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

async function mountForceGraph3D(
  nodes: GraphNode[] = [],
  edges: GraphEdge[] = [],
  extraProps: Record<string, unknown> = {},
) {
  vi.stubGlobal('ResizeObserver', MockResizeObserver)
  vi.useFakeTimers()

  const ForceGraph3DVue = (await import('../../web/src/components/map/ForceGraph3D.vue')).default
  const wrapper = mount(ForceGraph3DVue, {
    props: { nodes, edges, ...extraProps },
    attachTo: document.body,
  })
  await flushPromises()
  vi.advanceTimersByTime(1100)
  await flushPromises()
  vi.useRealTimers()

  return wrapper
}

/** Invoke buildNodeObject / buildReleaseObject via the captured nodeThreeObject callback. */
function callNodeThreeObject(node: GraphNode): MockGroup {
  const cb = getNodeThreeObjectCallback()
  if (!cb) throw new Error('nodeThreeObject callback not registered — is the component mounted?')
  return cb(node) as MockGroup
}

/** All sprite children in a group. */
function spritesIn(group: MockGroup): MockSprite[] {
  return group.children.filter((c): c is MockSprite => c instanceof MockSprite)
}

/** All torus mesh children in a group. */
function torusMeshesIn(group: MockGroup): MockMesh[] {
  return group.children.filter(
    (c): c is MockMesh => c instanceof MockMesh && c.geometry instanceof MockTorusGeometry,
  )
}

// ---------------------------------------------------------------------------
// Setup / teardown
// ---------------------------------------------------------------------------

beforeEach(() => {
  _createdSprites = []
  capturedTexts = []
  MockResizeObserver.mockClear()
  mockGraphInstance.nodeThreeObject.mockClear()
  // Install canvas mock before each test so textSprite() has a working context
  installCanvasMock()
})

afterEach(() => {
  document.body.innerHTML = ''
  createElementSpy?.mockRestore()
  createElementSpy = null
  vi.clearAllMocks()
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

// ===========================================================================
// Both props false — no title or lineage sprites on regular nodes
// ===========================================================================

describe('ForceGraph3D labels — both props false (M3)', () => {
  it('a regular node produces no sprites when both props are false', async () => {
    await mountForceGraph3D([], [], { showNodeTitles: false, showNodeLineage: false })
    const group = callNodeThreeObject(makeNode())
    expect(spritesIn(group)).toHaveLength(0)
  })

  it('a regular node with no priority/active-status ring has only non-sprite children', async () => {
    await mountForceGraph3D([], [], { showNodeTitles: false, showNodeLineage: false })
    const node = makeNode({ status: 'draft', priority: undefined })
    const group = callNodeThreeObject(node)
    expect(spritesIn(group)).toHaveLength(0)
    expect(torusMeshesIn(group)).toHaveLength(0)
  })
})

// ===========================================================================
// showNodeTitles=true — title sprite added, truncated to 15 chars
// ===========================================================================

describe('ForceGraph3D labels — showNodeTitles=true (M3)', () => {
  it('a regular node gets exactly one sprite when showNodeTitles=true (no lineage)', async () => {
    await mountForceGraph3D([], [], { showNodeTitles: true, showNodeLineage: false })
    capturedTexts = []
    const group = callNodeThreeObject(makeNode())
    expect(spritesIn(group)).toHaveLength(1)
  })

  it('title text is rendered unchanged when exactly 15 chars long', async () => {
    const title = 'My Feature Plan' // exactly 15 chars
    expect(title).toHaveLength(15)
    await mountForceGraph3D([], [], { showNodeTitles: true, showNodeLineage: false })
    capturedTexts = []
    callNodeThreeObject(makeNode({ title }))
    expect(capturedTexts).toContain(title)
  })

  it('title text is truncated to 15 chars + … when title is 16+ chars', async () => {
    const title = 'My Very Long Feature Plan Title'
    await mountForceGraph3D([], [], { showNodeTitles: true, showNodeLineage: false })
    capturedTexts = []
    callNodeThreeObject(makeNode({ title }))
    expect(capturedTexts).toContain('My Very Long Fe\u2026')
  })

  it('title uses the node slug as fallback when title is empty', async () => {
    await mountForceGraph3D([], [], { showNodeTitles: true, showNodeLineage: false })
    capturedTexts = []
    callNodeThreeObject(makeNode({ title: '', slug: 'my-slug' }))
    expect(capturedTexts).toContain('my-slug')
  })
})

// ===========================================================================
// showNodeLineage=true — lineage sprite added, full untruncated text
// ===========================================================================

describe('ForceGraph3D labels — showNodeLineage=true (M3)', () => {
  it('a regular node gets exactly one sprite when showNodeLineage=true (no titles)', async () => {
    await mountForceGraph3D([], [], { showNodeTitles: false, showNodeLineage: true })
    capturedTexts = []
    const group = callNodeThreeObject(makeNode({ lineage: 'feature-slug' }))
    expect(spritesIn(group)).toHaveLength(1)
  })

  it('lineage text is rendered in full (not truncated)', async () => {
    const lineage = 'this-is-a-long-lineage-slug-name'
    await mountForceGraph3D([], [], { showNodeTitles: false, showNodeLineage: true })
    capturedTexts = []
    callNodeThreeObject(makeNode({ lineage }))
    expect(capturedTexts).toContain(lineage)
  })

  it('node with empty lineage produces no lineage sprite', async () => {
    await mountForceGraph3D([], [], { showNodeTitles: false, showNodeLineage: true })
    capturedTexts = []
    const group = callNodeThreeObject(makeNode({ lineage: '' }))
    // buildNodeObject skips the lineage sprite when lineage is falsy
    expect(spritesIn(group)).toHaveLength(0)
  })
})

// ===========================================================================
// Both props true — two sprites at different y-offsets
// ===========================================================================

describe('ForceGraph3D labels — both props true (M3)', () => {
  it('a regular node gets two sprites when both props are true', async () => {
    await mountForceGraph3D([], [], { showNodeTitles: true, showNodeLineage: true })
    capturedTexts = []
    const group = callNodeThreeObject(makeNode({ lineage: 'my-feature' }))
    expect(spritesIn(group)).toHaveLength(2)
  })

  it('the two sprites are at different y-offsets (no overlap)', async () => {
    await mountForceGraph3D([], [], { showNodeTitles: true, showNodeLineage: true })
    capturedTexts = []
    const group = callNodeThreeObject(makeNode({ lineage: 'my-feature' }))
    const sprites = spritesIn(group)
    expect(sprites).toHaveLength(2)
    const [y0, y1] = sprites.map((s) => s.position.y)
    expect(y0, 'title and lineage sprites must be at different y-offsets').not.toBe(y1)
  })

  it('title sprite is at a higher y-offset than lineage sprite', async () => {
    // Per implementation: titleY=12 when both active, lineageY=5
    await mountForceGraph3D([], [], { showNodeTitles: true, showNodeLineage: true })
    capturedTexts = []
    const group = callNodeThreeObject(makeNode({ lineage: 'my-feature' }))
    const sprites = spritesIn(group)
    expect(sprites).toHaveLength(2)
    const [titleSprite, lineageSprite] = sprites
    expect(titleSprite.position.y).toBeGreaterThan(lineageSprite.position.y)
  })
})

// ===========================================================================
// Release nodes — buildReleaseObject is used (not buildNodeObject)
// ===========================================================================

describe('ForceGraph3D labels — release nodes unaffected (M3)', () => {
  it('a release node gets exactly one sprite regardless of props (its title label)', async () => {
    await mountForceGraph3D([], [], { showNodeTitles: false, showNodeLineage: false })
    capturedTexts = []
    const releaseNode = makeNode({ type: 'release', title: 'v1.0.0', slug: 'v1.0.0', status: 'draft' })
    const group = callNodeThreeObject(releaseNode)
    // buildReleaseObject always adds one title sprite; no extra sprites from showNodeTitles/showNodeLineage
    expect(spritesIn(group)).toHaveLength(1)
  })

  it('showNodeTitles=true does not add extra sprites to a release node', async () => {
    // Count sprites with props false
    await mountForceGraph3D([], [], { showNodeTitles: false, showNodeLineage: false })
    const releaseNode = makeNode({ type: 'release', title: 'v1.0.0', slug: 'v1.0.0' })
    capturedTexts = []
    const countOff = spritesIn(callNodeThreeObject(releaseNode)).length

    // Count sprites with showNodeTitles=true
    await mountForceGraph3D([], [], { showNodeTitles: true, showNodeLineage: false })
    capturedTexts = []
    const countOn = spritesIn(callNodeThreeObject(releaseNode)).length

    expect(countOn).toBe(countOff)
  })

  it('showNodeLineage=true does not add extra sprites to a release node', async () => {
    await mountForceGraph3D([], [], { showNodeTitles: false, showNodeLineage: false })
    const releaseNode = makeNode({ type: 'release', title: 'v1.0.0', lineage: 'releases' })
    capturedTexts = []
    const countOff = spritesIn(callNodeThreeObject(releaseNode)).length

    await mountForceGraph3D([], [], { showNodeTitles: false, showNodeLineage: true })
    capturedTexts = []
    const countOn = spritesIn(callNodeThreeObject(releaseNode)).length

    expect(countOn).toBe(countOff)
  })
})

// ===========================================================================
// Label nodes — always get their own text sprite, never get title/lineage overlays
// ===========================================================================

describe('ForceGraph3D labels — label-type nodes unaffected (M3)', () => {
  it('a label node gets exactly one sprite (its own label text) when both props false', async () => {
    await mountForceGraph3D([], [], { showNodeTitles: false, showNodeLineage: false })
    capturedTexts = []
    const labelNode = makeNode({ type: 'label', title: 'Backend', lineage: '' })
    const group = callNodeThreeObject(labelNode)
    expect(spritesIn(group)).toHaveLength(1)
  })

  it('a label node still gets exactly one sprite when showNodeTitles=true', async () => {
    await mountForceGraph3D([], [], { showNodeTitles: true, showNodeLineage: false })
    capturedTexts = []
    const labelNode = makeNode({ type: 'label', title: 'Backend', lineage: '' })
    const group = callNodeThreeObject(labelNode)
    expect(spritesIn(group)).toHaveLength(1)
  })

  it('a label node still gets exactly one sprite when showNodeLineage=true', async () => {
    await mountForceGraph3D([], [], { showNodeTitles: false, showNodeLineage: true })
    capturedTexts = []
    const labelNode = makeNode({ type: 'label', title: 'Backend', lineage: 'some-lineage' })
    const group = callNodeThreeObject(labelNode)
    expect(spritesIn(group)).toHaveLength(1)
  })

  it('a label node still gets exactly one sprite when both props are true', async () => {
    await mountForceGraph3D([], [], { showNodeTitles: true, showNodeLineage: true })
    capturedTexts = []
    const labelNode = makeNode({ type: 'label', title: 'Backend', lineage: 'some-lineage' })
    const group = callNodeThreeObject(labelNode)
    expect(spritesIn(group)).toHaveLength(1)
  })
})
