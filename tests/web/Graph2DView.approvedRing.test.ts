/**
 * Milestone 2 — Component tests for Graph2DView.vue — approved-test blue ring
 *
 * Covers:
 *   1. Cytoscape style array contains the approved-test selector with the correct
 *      border-color (APPROVED_TEST_RING_COLOR) and border-width (4).
 *   2. No other Cytoscape selector applies APPROVED_TEST_RING_COLOR — ensuring
 *      type: 'test' / status: 'draft' nodes are not affected.
 *   3. No selector applies APPROVED_TEST_RING_COLOR to non-test approved nodes
 *      (e.g. type: 'requirement', status: 'approved').
 *   4. The setInterval pulse loop skips approved test nodes — their border is
 *      not overridden even when their status would otherwise match an active-status
 *      colour.
 *
 * Testing approach
 * ───────────────
 * Cytoscape is dynamically imported inside Graph2DView.vue, so we intercept both
 * imports with vi.mock.  The mock Cytoscape constructor captures the full options
 * object passed to it, letting us inspect the style array directly.
 *
 * graphConstants is also mocked so that:
 *   - APPROVED_TEST_RING_COLOR is a known sentinel value we can assert on.
 *   - ACTIVE_STATUS_COLORS includes 'approved' — making the pulse-loop guard
 *     meaningful (without this entry, the guard is unreachable in the current code
 *     because ACTIVE_STATUS_COLORS['approved'] would be undefined).
 *
 * Component: web/src/components/graph/Graph2DView.vue
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import type { GraphNode, GraphEdge } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Sentinel value — used in all assertions so tests are independent of the real
// graphConstants value (which is verified separately in graphConstants.test.ts)
// ---------------------------------------------------------------------------

const APPROVED_TEST_RING_COLOR = '#2563eb'

// ---------------------------------------------------------------------------
// Hoisted mock state — must be declared before vi.mock calls
// ---------------------------------------------------------------------------

const { mockCyConstructor, mockCyInstance, setCyNodes } = vi.hoisted(() => {
  // Mutable array of mock cy nodes — tests swap this per-scenario
  let _cyNodes: any[] = []

  const mockCyInstance = {
    on: vi.fn(),
    elements: vi.fn().mockReturnValue({ remove: vi.fn() }),
    add: vi.fn(),
    layout: vi.fn().mockReturnValue({ run: vi.fn() }),
    nodes: vi.fn(() => ({
      forEach: (cb: (n: any) => void) => _cyNodes.forEach(cb),
    })),
    destroy: vi.fn(),
  }

  const ctor: any = vi.fn().mockReturnValue(mockCyInstance)
  ctor.use = vi.fn()

  return {
    mockCyConstructor: ctor,
    mockCyInstance,
    setCyNodes: (nodes: any[]) => { _cyNodes = nodes },
  }
})

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

vi.mock('cytoscape', () => ({ default: mockCyConstructor }))
vi.mock('cytoscape-fcose', () => ({ default: {} }))

// Mock graphConstants so:
//   - APPROVED_TEST_RING_COLOR has a known value
//   - ACTIVE_STATUS_COLORS includes 'approved', making the pulse-loop guard testable
vi.mock('@/components/graph/graphConstants', () => ({
  NODE_COLORS: {
    idea: '#f59e0b',
    requirement: '#3b82f6',
    'plan-backend': '#8b5cf6',
    'plan-frontend': '#a78bfa',
    'plan-test': '#c084fc',
    test: '#06b6d4',
    prototype: '#14b8a6',
    defect: '#f43f5e',
    label: '#a855f7',
  },
  PRIORITY_COLORS: {
    high: '#ef4444',
    medium: '#f97316',
    normal: '#22c55e',
    low: '#3b82f6',
  },
  ACTIVE_STATUS_COLORS: {
    'in-development': '#4ade80',
    'in-qa': '#fbbf24',
    'in-progress': '#4ade80',
    clarifying: '#60a5fa',
    planning: '#a78bfa',
    // Intentionally included so the pulse-loop guard is exercised in test 4
    approved: '#00ff00',
  },
  EDGE_COLORS: {
    parent: '#94a3b8',
    depends_on: '#f97316',
    blocks: '#ef4444',
    related_to: '#64748b',
    label: '#a855f7',
  },
  APPROVED_TEST_RING_COLOR,
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
    index: 3,
    ...overrides,
  }
}

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

async function mountGraph2D(nodes: GraphNode[] = [], edges: GraphEdge[] = []) {
  const Graph2DView = (await import('../../web/src/components/graph/Graph2DView.vue')).default
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
// Helpers
// ---------------------------------------------------------------------------

/** Returns the style array from the most recent Cytoscape constructor call. */
function capturedStyleArray(): Array<{ selector: string; style: Record<string, unknown> }> {
  const calls = mockCyConstructor.mock.calls
  if (!calls.length) throw new Error('Cytoscape constructor was not called')
  return calls[calls.length - 1][0].style
}

/** Finds the first style rule whose selector matches the given string. */
function findStyleRule(
  selector: string,
): { selector: string; style: Record<string, unknown> } | undefined {
  return capturedStyleArray().find((r) => r.selector === selector)
}

// ---------------------------------------------------------------------------
// Cleanup
// ---------------------------------------------------------------------------

beforeEach(() => {
  mockCyConstructor.mockClear()
  mockCyInstance.on.mockClear()
  mockCyInstance.nodes.mockClear()
  setCyNodes([])
})

afterEach(() => {
  document.body.innerHTML = ''
  vi.clearAllTimers()
})

// ===========================================================================
// Test 1 — Approved test node gets blue ring style
// ===========================================================================

describe('Graph2DView — approved-test Cytoscape style rule', () => {
  it('style array includes the approved-test selector', async () => {
    await mountGraph2D([makeNode()])
    const rule = findStyleRule('node[type="test"][status="approved"]')
    expect(rule, 'expected a style rule for node[type="test"][status="approved"]').toBeDefined()
  })

  it('approved-test selector has border-color equal to APPROVED_TEST_RING_COLOR', async () => {
    await mountGraph2D([makeNode()])
    const rule = findStyleRule('node[type="test"][status="approved"]')!
    expect(rule.style['border-color']).toBe(APPROVED_TEST_RING_COLOR)
  })

  it('approved-test selector has border-width of 4', async () => {
    await mountGraph2D([makeNode()])
    const rule = findStyleRule('node[type="test"][status="approved"]')!
    expect(rule.style['border-width']).toBe(4)
  })
})

// ===========================================================================
// Test 2 — Non-approved test node does not get the blue ring
// ===========================================================================

describe('Graph2DView — non-approved test node is not styled with blue ring', () => {
  it('APPROVED_TEST_RING_COLOR does not appear in any other style rule', async () => {
    await mountGraph2D([makeNode({ status: 'draft' })])
    const styles = capturedStyleArray()
    const rulesWithApprovedColor = styles.filter(
      (r) =>
        r.selector !== 'node[type="test"][status="approved"]' &&
        r.style['border-color'] === APPROVED_TEST_RING_COLOR,
    )
    expect(
      rulesWithApprovedColor,
      `unexpected style rules use APPROVED_TEST_RING_COLOR: ${JSON.stringify(rulesWithApprovedColor.map((r) => r.selector))}`,
    ).toHaveLength(0)
  })

  it('no style rule targets type=test without also requiring status=approved', async () => {
    await mountGraph2D([makeNode({ status: 'draft' })])
    const styles = capturedStyleArray()
    // A selector like `node[type="test"]` (no status constraint) would be too broad
    const tooBoard = styles.filter(
      (r) =>
        r.selector === 'node[type="test"]' &&
        r.style['border-color'] === APPROVED_TEST_RING_COLOR,
    )
    expect(tooBoard).toHaveLength(0)
  })
})

// ===========================================================================
// Test 3 — Non-test approved node does not get the blue ring
// ===========================================================================

describe('Graph2DView — non-test approved node is not styled with blue ring', () => {
  it('APPROVED_TEST_RING_COLOR does not appear in a selector without the type=test constraint', async () => {
    await mountGraph2D([makeNode({ type: 'requirement', status: 'approved' })])
    const styles = capturedStyleArray()
    // A selector like `node[status="approved"]` (no type constraint) would be too broad
    const tooBroadStatus = styles.filter(
      (r) =>
        r.selector === 'node[status="approved"]' &&
        r.style['border-color'] === APPROVED_TEST_RING_COLOR,
    )
    expect(tooBroadStatus).toHaveLength(0)
  })

  it('only the exact combined selector uses APPROVED_TEST_RING_COLOR', async () => {
    await mountGraph2D([makeNode({ type: 'requirement', status: 'approved' })])
    const styles = capturedStyleArray()
    // Every rule using APPROVED_TEST_RING_COLOR must have exactly the combined selector
    const allApprovedColorRules = styles.filter(
      (r) => r.style['border-color'] === APPROVED_TEST_RING_COLOR,
    )
    for (const rule of allApprovedColorRules) {
      expect(rule.selector).toBe('node[type="test"][status="approved"]')
    }
  })
})

// ===========================================================================
// Test 4 — Pulse loop skips approved test nodes
// ===========================================================================

describe('Graph2DView — pulse loop guard for approved test nodes', () => {
  it('does not call .style() on a test/approved node during the pulse tick', async () => {
    vi.useFakeTimers()

    // Build a mock cy node with type=test, status=approved
    // Its status 'approved' is in our mocked ACTIVE_STATUS_COLORS, so without the
    // guard the pulse loop would attempt to override its border.
    const mockStyle = vi.fn()
    const mockData = vi.fn((key: string) => {
      if (key === 'type') return 'test'
      if (key === 'status') return 'approved'
      return null
    })
    const approvedTestCyNode = { data: mockData, style: mockStyle }
    setCyNodes([approvedTestCyNode])

    await mountGraph2D([makeNode()])

    // Advance past the 700 ms pulse interval
    vi.advanceTimersByTime(700)
    await flushPromises()

    expect(
      mockStyle,
      'pulse loop must not call .style() on a test/approved node',
    ).not.toHaveBeenCalled()

    vi.useRealTimers()
  })

  it('does call .style() on an in-qa test node (guard does not block active statuses)', async () => {
    vi.useFakeTimers()

    const mockStyle = vi.fn()
    const mockData = vi.fn((key: string) => {
      if (key === 'type') return 'test'
      if (key === 'status') return 'in-qa'
      return null
    })
    const inQaCyNode = { data: mockData, style: mockStyle }
    setCyNodes([inQaCyNode])

    await mountGraph2D([makeNode({ status: 'in-qa' })])

    vi.advanceTimersByTime(700)
    await flushPromises()

    // 'in-qa' is in ACTIVE_STATUS_COLORS and is not an approved test node,
    // so the pulse loop should style it.
    expect(mockStyle).toHaveBeenCalled()

    vi.useRealTimers()
  })
})
