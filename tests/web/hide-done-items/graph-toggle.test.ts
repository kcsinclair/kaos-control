/**
 * Milestone 4 — GraphView toggle tests.
 *
 * Tests the Pinia graph store's `hideTerminal` ref and the computed
 * `filteredNodes` / `filteredEdges` that depend on it.
 *
 * Key behaviour under test:
 *  - hideTerminal defaults to true
 *  - When true, nodes with terminal status are absent from filteredNodes
 *  - When true, edges connecting to hidden nodes are pruned from filteredEdges
 *  - When false, all nodes and their edges reappear
 *  - When a user-defined status filter is active, hideTerminal is overridden
 *    (i.e. the user's explicit status selection takes precedence)
 *  - toggleHideTerminal() flips the flag
 *  - Changing hideTerminal does not call any graph API
 *  - Node count (filteredNodes.length) reflects the filtered set
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useGraphStore } from '@/stores/graph'
import {
  makeGraphNode,
  makeGraphEdge,
  makeGraphNodesForAllStatuses,
  TERMINAL_STATUSES,
  ACTIVE_STATUSES,
} from '../helpers/seed_artifacts'

// The graph store calls graphApi.getGraph; mock it so no fetch is needed.
vi.mock('@/api/graph', () => ({
  getGraph: vi.fn(),
}))

import * as graphApi from '@/api/graph'

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

// ---------------------------------------------------------------------------
// Default state
// ---------------------------------------------------------------------------

describe('useGraphStore — default state (hideTerminal=true)', () => {
  it('hideTerminal defaults to true', () => {
    const store = useGraphStore()
    expect(store.hideTerminal).toBe(true)
  })

  it('filteredNodes excludes terminal-status nodes', () => {
    const store = useGraphStore()
    store.rawNodes = makeGraphNodesForAllStatuses()

    const statuses = store.filteredNodes.map((n) => n.status)
    for (const t of TERMINAL_STATUSES) {
      expect(statuses).not.toContain(t)
    }
  })

  it('filteredNodes includes all active-status nodes', () => {
    const store = useGraphStore()
    store.rawNodes = makeGraphNodesForAllStatuses()

    const statuses = store.filteredNodes.map((n) => n.status)
    for (const a of ACTIVE_STATUSES) {
      expect(statuses).toContain(a)
    }
  })

  it('filteredNodes count equals active artifact count', () => {
    const store = useGraphStore()
    store.rawNodes = makeGraphNodesForAllStatuses()

    expect(store.filteredNodes.length).toBe(ACTIVE_STATUSES.length)
  })

  it('returns all nodes when none are terminal', () => {
    const store = useGraphStore()
    store.rawNodes = ACTIVE_STATUSES.map((s) => makeGraphNode({ status: s }))

    expect(store.filteredNodes.length).toBe(ACTIVE_STATUSES.length)
  })

  it('returns no nodes when all are terminal', () => {
    const store = useGraphStore()
    store.rawNodes = TERMINAL_STATUSES.map((s) => makeGraphNode({ status: s }))

    expect(store.filteredNodes.length).toBe(0)
  })
})

// ---------------------------------------------------------------------------
// Edges to hidden nodes are removed
// ---------------------------------------------------------------------------

describe('useGraphStore — edges to hidden terminal nodes are pruned', () => {
  it('edge from active to terminal is absent from filteredEdges', () => {
    const store = useGraphStore()
    const active = makeGraphNode({ id: 'a.md', status: 'draft' })
    const terminal = makeGraphNode({ id: 'b.md', status: 'done' })

    store.rawNodes = [active, terminal]
    store.rawEdges = [makeGraphEdge('a.md', 'b.md')]

    expect(store.filteredEdges).toHaveLength(0)
  })

  it('edge between two active nodes is retained', () => {
    const store = useGraphStore()
    const n1 = makeGraphNode({ id: 'n1.md', status: 'draft' })
    const n2 = makeGraphNode({ id: 'n2.md', status: 'in-development' })

    store.rawNodes = [n1, n2]
    store.rawEdges = [makeGraphEdge('n1.md', 'n2.md')]

    expect(store.filteredEdges).toHaveLength(1)
    expect(store.filteredEdges[0].source).toBe('n1.md')
    expect(store.filteredEdges[0].target).toBe('n2.md')
  })

  it('edges between two terminal nodes are both pruned', () => {
    const store = useGraphStore()
    const d = makeGraphNode({ id: 'd.md', status: 'done' })
    const r = makeGraphNode({ id: 'r.md', status: 'rejected' })

    store.rawNodes = [d, r]
    store.rawEdges = [makeGraphEdge('d.md', 'r.md')]

    expect(store.filteredEdges).toHaveLength(0)
  })

  it('mixed graph: only active↔active edges survive', () => {
    const store = useGraphStore()
    const a1 = makeGraphNode({ id: 'a1.md', status: 'draft' })
    const a2 = makeGraphNode({ id: 'a2.md', status: 'in-development' })
    const t1 = makeGraphNode({ id: 't1.md', status: 'done' })

    store.rawNodes = [a1, a2, t1]
    store.rawEdges = [
      makeGraphEdge('a1.md', 'a2.md'), // survives
      makeGraphEdge('a2.md', 't1.md'), // pruned (target is terminal)
      makeGraphEdge('t1.md', 'a1.md'), // pruned (source is terminal)
    ]

    expect(store.filteredEdges).toHaveLength(1)
    expect(store.filteredEdges[0]).toMatchObject({ source: 'a1.md', target: 'a2.md' })
  })
})

// ---------------------------------------------------------------------------
// Toggle reveals terminal nodes
// ---------------------------------------------------------------------------

describe('useGraphStore — toggle reveals terminal nodes', () => {
  it('filteredNodes includes terminal nodes when hideTerminal=false', () => {
    const store = useGraphStore()
    store.rawNodes = makeGraphNodesForAllStatuses()

    store.hideTerminal = false

    const statuses = store.filteredNodes.map((n) => n.status)
    for (const t of TERMINAL_STATUSES) {
      expect(statuses).toContain(t)
    }
  })

  it('filteredNodes count equals rawNodes count when hideTerminal=false', () => {
    const store = useGraphStore()
    store.rawNodes = makeGraphNodesForAllStatuses()

    store.hideTerminal = false

    expect(store.filteredNodes.length).toBe(store.rawNodes.length)
  })

  it('edges to terminal nodes reappear when hideTerminal=false', () => {
    const store = useGraphStore()
    const active = makeGraphNode({ id: 'a.md', status: 'draft' })
    const terminal = makeGraphNode({ id: 'b.md', status: 'done' })

    store.rawNodes = [active, terminal]
    store.rawEdges = [makeGraphEdge('a.md', 'b.md')]

    // initially hidden
    expect(store.filteredEdges).toHaveLength(0)

    store.hideTerminal = false
    expect(store.filteredEdges).toHaveLength(1)
  })
})

// ---------------------------------------------------------------------------
// toggleHideTerminal action
// ---------------------------------------------------------------------------

describe('useGraphStore — toggleHideTerminal action', () => {
  it('flips hideTerminal from true to false', () => {
    const store = useGraphStore()
    expect(store.hideTerminal).toBe(true)

    store.toggleHideTerminal()

    expect(store.hideTerminal).toBe(false)
  })

  it('flips hideTerminal from false to true', () => {
    const store = useGraphStore()
    store.hideTerminal = false

    store.toggleHideTerminal()

    expect(store.hideTerminal).toBe(true)
  })
})

// ---------------------------------------------------------------------------
// Explicit status filter overrides hideTerminal
// ---------------------------------------------------------------------------

describe('useGraphStore — explicit status filter overrides hideTerminal', () => {
  it('terminal nodes appear when user has filtered by that terminal status', () => {
    const store = useGraphStore()
    store.rawNodes = makeGraphNodesForAllStatuses()

    // Explicit status filter: show only "done"
    store.setFilter({ statuses: ['done'] })

    const statuses = store.filteredNodes.map((n) => n.status)
    expect(statuses).toContain('done')
    // Other non-done statuses should be excluded because of the status filter
    expect(statuses.every((s) => s === 'done')).toBe(true)
  })

  it('hideTerminal has no effect when user status filter is set', () => {
    const store = useGraphStore()
    store.rawNodes = makeGraphNodesForAllStatuses()

    store.setFilter({ statuses: ['done'] })

    // hideTerminal is true (default) but since filter.statuses is non-empty,
    // the hideTerminal guard is bypassed
    expect(store.hideTerminal).toBe(true)
    const statuses = store.filteredNodes.map((n) => n.status)
    expect(statuses).toContain('done')
  })

  it('clearing status filter re-activates hideTerminal', () => {
    const store = useGraphStore()
    store.rawNodes = makeGraphNodesForAllStatuses()

    store.setFilter({ statuses: ['done'] })
    expect(store.filteredNodes.map((n) => n.status)).toContain('done')

    store.setFilter({ statuses: [] })
    expect(store.filteredNodes.map((n) => n.status)).not.toContain('done')
  })
})

// ---------------------------------------------------------------------------
// Node count display (N / M nodes)
// ---------------------------------------------------------------------------

describe('useGraphStore — node count reflects filtered set', () => {
  it('filteredNodes.length < rawNodes.length when hideTerminal=true', () => {
    const store = useGraphStore()
    store.rawNodes = makeGraphNodesForAllStatuses()

    expect(store.filteredNodes.length).toBeLessThan(store.rawNodes.length)
  })

  it('filteredNodes.length === rawNodes.length when hideTerminal=false', () => {
    const store = useGraphStore()
    store.rawNodes = makeGraphNodesForAllStatuses()

    store.hideTerminal = false

    expect(store.filteredNodes.length).toBe(store.rawNodes.length)
  })
})

// ---------------------------------------------------------------------------
// Toggling does NOT trigger API calls
// ---------------------------------------------------------------------------

describe('useGraphStore — no extra API calls on toggle', () => {
  it('toggling hideTerminal does not call getGraph', async () => {
    const store = useGraphStore()
    // fetch once to set baseline
    vi.mocked(graphApi.getGraph).mockResolvedValue({ nodes: [], edges: [] })
    await store.fetchGraph('test-project')

    const callsBefore = vi.mocked(graphApi.getGraph).mock.calls.length

    store.toggleHideTerminal()
    store.toggleHideTerminal()

    expect(vi.mocked(graphApi.getGraph).mock.calls.length).toBe(callsBefore)
  })
})

// ---------------------------------------------------------------------------
// Reset behaviour (simulating GraphView onMounted)
// ---------------------------------------------------------------------------

describe('useGraphStore — reset on navigation (simulating onMounted)', () => {
  it('setting hideTerminal=true restores default hidden state', () => {
    const store = useGraphStore()
    store.rawNodes = makeGraphNodesForAllStatuses()

    // User checked "Show completed"
    store.hideTerminal = false
    expect(store.filteredNodes.length).toBe(store.rawNodes.length)

    // GraphView.onMounted resets
    store.hideTerminal = true
    expect(store.filteredNodes.length).toBe(ACTIVE_STATUSES.length)
  })
})
