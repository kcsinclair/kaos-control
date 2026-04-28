/**
 * Milestone 5 — Cross-view consistency tests.
 *
 * Verifies that the "Show completed" toggle is independent across views:
 *  - Toggling in list view does not affect kanban
 *  - Toggling in kanban does not affect graph
 *  - Each view resets to its default hidden state independently
 *  - No extra API calls are triggered by toggling
 *
 * Implementation notes:
 *  - ArtifactListView and KanbanBoardView use local refs/composables so each
 *    "mount" creates a fresh, independent instance.
 *  - GraphView uses the Pinia graph store (a singleton). GraphView.onMounted
 *    resets store.hideTerminal = true on every mount to ensure navigation
 *    resets the state.
 *  - useKanbanBoard is a plain function returning independent reactive state.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { ref, computed, effectScope } from 'vue'
import { setActivePinia, createPinia } from 'pinia'
import { TERMINAL_STATUSES } from '@/types/api'

vi.mock('@/api/client', () => ({
  api: { get: vi.fn() },
}))
vi.mock('@/api/artifacts', () => ({
  listArtifacts: vi.fn(),
}))
vi.mock('@/api/graph', () => ({
  getGraph: vi.fn(),
}))

import { api } from '@/api/client'
import * as artifactsApi from '@/api/artifacts'
import * as graphApi from '@/api/graph'
import { useKanbanBoard } from '@/composables/useKanbanBoard'
import { useGraphStore } from '@/stores/graph'
import {
  makeArtifactsForAllStatuses,
  makeGraphNodesForAllStatuses,
  ACTIVE_STATUSES,
} from '../helpers/seed_artifacts'

const KANBAN_CONFIG = {
  columns: [
    { name: 'Backlog', statuses: ['draft', 'clarifying'] },
    { name: 'In Progress', statuses: ['planning', 'in-development', 'in-qa'] },
    { name: 'Done', statuses: ['done', 'abandoned', 'rejected'] },
  ],
  uncategorised: false,
}

function setupApiMocks() {
  vi.mocked(api.get).mockResolvedValue({ kanban: KANBAN_CONFIG })
  vi.mocked(artifactsApi.listArtifacts).mockResolvedValue({
    items: makeArtifactsForAllStatuses(),
    total: 8,
  })
  vi.mocked(graphApi.getGraph).mockResolvedValue({
    nodes: makeGraphNodesForAllStatuses(),
    edges: [],
  })
}

/** Simulate ArtifactListView's showCompleted ref + visibleItems computed. */
function makeListView(storeItems = makeArtifactsForAllStatuses()) {
  const items = ref(storeItems)
  const showCompleted = ref(false)
  const visibleItems = computed(() =>
    showCompleted.value
      ? items.value
      : items.value.filter((r) => !(TERMINAL_STATUSES as readonly string[]).includes(r.status)),
  )
  return { showCompleted, visibleItems }
}

let scopes: ReturnType<typeof effectScope>[] = []

function makeKanbanScope() {
  let board!: ReturnType<typeof useKanbanBoard>
  const scope = effectScope()
  scope.run(() => {
    board = useKanbanBoard('test-project')
  })
  scopes.push(scope)
  return board
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
  setupApiMocks()
  scopes = []
})

afterEach(() => {
  scopes.forEach((s) => s.stop())
})

// ---------------------------------------------------------------------------
// Toggling in list view does not affect kanban
// ---------------------------------------------------------------------------

describe('cross-view: list toggle does not affect kanban', () => {
  it('enabling showCompleted in list view leaves kanban hideTerminal unchanged', async () => {
    const listView = makeListView()
    const kanban = makeKanbanScope()
    await kanban.refresh()

    // Enable in list view
    listView.showCompleted.value = true
    expect(listView.visibleItems.value.length).toBe(8) // all shown

    // Kanban's hideTerminal is still true — Done column still hidden
    expect(kanban.hideTerminal.value).toBe(true)
    expect(kanban.columns.value.find((c) => c.name === 'Done')).toBeUndefined()
  })
})

// ---------------------------------------------------------------------------
// Toggling in kanban does not affect graph
// ---------------------------------------------------------------------------

describe('cross-view: kanban toggle does not affect graph', () => {
  it('disabling hideTerminal in kanban leaves graph store hideTerminal unchanged', async () => {
    const kanban = makeKanbanScope()
    const graphStore = useGraphStore()
    graphStore.rawNodes = makeGraphNodesForAllStatuses()

    await kanban.refresh()

    // Enable in kanban
    kanban.hideTerminal.value = false
    expect(kanban.columns.value.find((c) => c.name === 'Done')).toBeDefined()

    // Graph store's hideTerminal is still true
    expect(graphStore.hideTerminal).toBe(true)
    expect(graphStore.filteredNodes.length).toBe(ACTIVE_STATUSES.length)
  })
})

// ---------------------------------------------------------------------------
// Two kanban board instances are independent
// ---------------------------------------------------------------------------

describe('cross-view: two useKanbanBoard instances are independent', () => {
  it('toggling hideTerminal on one instance does not affect the other', async () => {
    const boardA = makeKanbanScope()
    const boardB = makeKanbanScope()

    await boardA.refresh()
    await boardB.refresh()

    boardA.hideTerminal.value = false

    // A shows Done column
    expect(boardA.columns.value.find((c) => c.name === 'Done')).toBeDefined()
    // B still hides Done column
    expect(boardB.hideTerminal.value).toBe(true)
    expect(boardB.columns.value.find((c) => c.name === 'Done')).toBeUndefined()
  })
})

// ---------------------------------------------------------------------------
// Each view resets independently
// ---------------------------------------------------------------------------

describe('cross-view: each view resets to default on "remount"', () => {
  it('list view showCompleted resets to false on a new component instance', () => {
    const first = makeListView()
    first.showCompleted.value = true
    expect(first.visibleItems.value.length).toBe(8)

    // Simulate navigation away and back: fresh instance
    const second = makeListView()
    expect(second.showCompleted.value).toBe(false)
    expect(second.visibleItems.value.length).toBe(ACTIVE_STATUSES.length)
  })

  it('kanban hideTerminal resets to true on a new composable instance', () => {
    const first = makeKanbanScope()
    first.hideTerminal.value = false

    // Simulate navigation away and back: fresh composable instance
    const second = makeKanbanScope()
    expect(second.hideTerminal.value).toBe(true)
  })

  it('graph store hideTerminal is restored to true by simulated onMounted reset', () => {
    const graphStore = useGraphStore()
    graphStore.rawNodes = makeGraphNodesForAllStatuses()

    // User checked "Show completed" in GraphView
    graphStore.hideTerminal = false
    expect(graphStore.filteredNodes.length).toBe(graphStore.rawNodes.length)

    // GraphView.onMounted resets it:
    graphStore.hideTerminal = true
    expect(graphStore.filteredNodes.length).toBe(ACTIVE_STATUSES.length)
  })
})

// ---------------------------------------------------------------------------
// No extra API calls on toggle (cross-view)
// ---------------------------------------------------------------------------

describe('cross-view: no extra API calls when toggling any view', () => {
  it('toggling list showCompleted triggers no API calls', () => {
    const listView = makeListView()
    const before = vi.mocked(artifactsApi.listArtifacts).mock.calls.length

    listView.showCompleted.value = true
    listView.showCompleted.value = false

    expect(vi.mocked(artifactsApi.listArtifacts).mock.calls.length).toBe(before)
  })

  it('toggling graph hideTerminal triggers no API calls', () => {
    const graphStore = useGraphStore()
    const before = vi.mocked(graphApi.getGraph).mock.calls.length

    graphStore.toggleHideTerminal()
    graphStore.toggleHideTerminal()

    expect(vi.mocked(graphApi.getGraph).mock.calls.length).toBe(before)
  })

  it('toggling kanban hideTerminal triggers no API calls', async () => {
    const kanban = makeKanbanScope()
    await kanban.refresh()

    const artifactCallsBefore = vi.mocked(artifactsApi.listArtifacts).mock.calls.length
    const configCallsBefore = vi.mocked(api.get).mock.calls.length

    kanban.hideTerminal.value = false
    kanban.hideTerminal.value = true

    expect(vi.mocked(artifactsApi.listArtifacts).mock.calls.length).toBe(artifactCallsBefore)
    expect(vi.mocked(api.get).mock.calls.length).toBe(configCallsBefore)
  })
})

// ---------------------------------------------------------------------------
// All three views start hidden by default (parallel)
// ---------------------------------------------------------------------------

describe('cross-view: all views default to hiding terminal items', () => {
  it('list, kanban, and graph all start with terminal items hidden', async () => {
    const listView = makeListView()
    const kanban = makeKanbanScope()
    const graphStore = useGraphStore()
    graphStore.rawNodes = makeGraphNodesForAllStatuses()

    await kanban.refresh()

    // List: only active items visible
    expect(listView.showCompleted.value).toBe(false)
    const listStatuses = listView.visibleItems.value.map((r) => r.status)
    expect(listStatuses).not.toContain('done')
    expect(listStatuses).not.toContain('rejected')
    expect(listStatuses).not.toContain('abandoned')

    // Kanban: Done column hidden
    expect(kanban.hideTerminal.value).toBe(true)
    expect(kanban.columns.value.find((c) => c.name === 'Done')).toBeUndefined()

    // Graph: terminal nodes not in filteredNodes
    expect(graphStore.hideTerminal).toBe(true)
    const graphStatuses = graphStore.filteredNodes.map((n) => n.status)
    expect(graphStatuses).not.toContain('done')
    expect(graphStatuses).not.toContain('rejected')
    expect(graphStatuses).not.toContain('abandoned')
  })
})
