/**
 * Milestone 3 — KanbanBoardView toggle tests.
 *
 * Tests the `useKanbanBoard` composable's `hideTerminal` ref and the `columns`
 * computed derived from it. The API layer is mocked so no server is needed.
 *
 * Key behaviour under test:
 *  - hideTerminal defaults to true
 *  - When true, terminal-status cards (done/rejected/abandoned) are absent from columns
 *  - When true, columns whose statuses are all-terminal are suppressed entirely
 *  - When false, the Done column and its cards appear
 *  - Non-terminal columns are unaffected by the toggle
 *  - Setting hideTerminal does NOT trigger additional API requests
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { effectScope } from 'vue'
import { setActivePinia, createPinia } from 'pinia'

// Mock API modules before importing the composable
vi.mock('@/api/client', () => ({
  api: { get: vi.fn() },
}))
vi.mock('@/api/artifacts', () => ({
  listArtifacts: vi.fn(),
}))

import { api } from '@/api/client'
import * as artifactsApi from '@/api/artifacts'
import { useKanbanBoard } from '@/composables/useKanbanBoard'
import type { KanbanConfig } from '@/composables/useKanbanBoard'
import { makeArtifactRow, makeArtifactsForAllStatuses } from '../helpers/seed_artifacts'

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

/** Kanban config with a Done column (all-terminal statuses) and active columns. */
const KANBAN_CONFIG: KanbanConfig = {
  columns: [
    { name: 'Backlog', statuses: ['draft', 'clarifying'] },
    { name: 'In Progress', statuses: ['planning', 'in-development', 'in-qa'] },
    { name: 'Done', statuses: ['done', 'abandoned', 'rejected'] },
  ],
  uncategorised: false,
}

function setupApiMocks(artifacts = makeArtifactsForAllStatuses()) {
  vi.mocked(api.get).mockResolvedValue({ kanban: KANBAN_CONFIG })
  vi.mocked(artifactsApi.listArtifacts).mockResolvedValue({
    items: artifacts,
    total: artifacts.length,
  })
}

// ---------------------------------------------------------------------------
// Setup / teardown
// ---------------------------------------------------------------------------

let scope: ReturnType<typeof effectScope>
let board: ReturnType<typeof useKanbanBoard>

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
  setupApiMocks()
  scope = effectScope()
  scope.run(() => {
    board = useKanbanBoard('test-project')
  })
})

afterEach(() => {
  scope.stop()
})

// ---------------------------------------------------------------------------
// Default state — hideTerminal = true
// ---------------------------------------------------------------------------

describe('useKanbanBoard — default state (hideTerminal=true)', () => {
  it('hideTerminal defaults to true', () => {
    expect(board.hideTerminal.value).toBe(true)
  })

  it('Done column is absent from columns before refresh', () => {
    // Before refresh, allArtifacts is empty so no columns form yet
    const doneCol = board.columns.value.find((c) => c.name === 'Done')
    expect(doneCol).toBeUndefined()
  })

  it('Done column is hidden after refresh with hideTerminal=true', async () => {
    await board.refresh()

    const doneCol = board.columns.value.find((c) => c.name === 'Done')
    expect(doneCol).toBeUndefined()
  })

  it('terminal cards are absent from all columns', async () => {
    await board.refresh()

    for (const col of board.columns.value) {
      const terminalCards = col.cards.filter((c) =>
        ['done', 'rejected', 'abandoned'].includes(c.status),
      )
      expect(terminalCards).toHaveLength(0)
    }
  })

  it('active columns (Backlog, In Progress) are present', async () => {
    await board.refresh()

    const names = board.columns.value.map((c) => c.name)
    expect(names).toContain('Backlog')
    expect(names).toContain('In Progress')
  })
})

// ---------------------------------------------------------------------------
// Toggle reveals Done column
// ---------------------------------------------------------------------------

describe('useKanbanBoard — toggle reveals Done column', () => {
  it('Done column appears after setting hideTerminal=false', async () => {
    await board.refresh()
    expect(board.columns.value.find((c) => c.name === 'Done')).toBeUndefined()

    board.hideTerminal.value = false

    const doneCol = board.columns.value.find((c) => c.name === 'Done')
    expect(doneCol).toBeDefined()
  })

  it('Done column contains all terminal-status cards when revealed', async () => {
    await board.refresh()
    board.hideTerminal.value = false

    const doneCol = board.columns.value.find((c) => c.name === 'Done')
    expect(doneCol).toBeDefined()

    const doneStatuses = doneCol!.cards.map((c) => c.status)
    expect(doneStatuses).toContain('done')
    expect(doneStatuses).toContain('rejected')
    expect(doneStatuses).toContain('abandoned')
  })

  it('column card counts reflect the visible set', async () => {
    await board.refresh()

    // With hideTerminal=true: no Done column, Backlog has draft+clarifying cards
    const backlogHidden = board.columns.value.find((c) => c.name === 'Backlog')
    expect(backlogHidden).toBeDefined()
    const hiddenBacklogCount = backlogHidden!.cards.length

    board.hideTerminal.value = false
    const backlogShown = board.columns.value.find((c) => c.name === 'Backlog')
    const shownBacklogCount = backlogShown!.cards.length

    // Backlog count is unaffected by the toggle (no terminal items there)
    expect(shownBacklogCount).toBe(hiddenBacklogCount)

    // Done column should now have 3 cards (done, rejected, abandoned)
    const doneCol = board.columns.value.find((c) => c.name === 'Done')
    expect(doneCol!.cards.length).toBe(3)
  })
})

// ---------------------------------------------------------------------------
// Toggle hides Done column again
// ---------------------------------------------------------------------------

describe('useKanbanBoard — toggle re-hides Done column', () => {
  it('Done column disappears after toggling back to hideTerminal=true', async () => {
    await board.refresh()

    board.hideTerminal.value = false
    expect(board.columns.value.find((c) => c.name === 'Done')).toBeDefined()

    board.hideTerminal.value = true
    expect(board.columns.value.find((c) => c.name === 'Done')).toBeUndefined()
  })
})

// ---------------------------------------------------------------------------
// Other columns are unaffected by the toggle
// ---------------------------------------------------------------------------

describe('useKanbanBoard — other columns unaffected by toggle', () => {
  it('Backlog cards are the same regardless of hideTerminal', async () => {
    await board.refresh()

    const getBacklogCards = () =>
      board.columns.value.find((c) => c.name === 'Backlog')?.cards ?? []

    const before = getBacklogCards().map((c) => c.path).sort()

    board.hideTerminal.value = false
    const after = getBacklogCards().map((c) => c.path).sort()

    expect(after).toEqual(before)
  })

  it('In Progress cards are the same regardless of hideTerminal', async () => {
    await board.refresh()

    const getInProgressCards = () =>
      board.columns.value.find((c) => c.name === 'In Progress')?.cards ?? []

    const before = getInProgressCards().map((c) => c.path).sort()

    board.hideTerminal.value = false
    const after = getInProgressCards().map((c) => c.path).sort()

    expect(after).toEqual(before)
  })
})

// ---------------------------------------------------------------------------
// Toggling does NOT trigger additional API requests
// ---------------------------------------------------------------------------

describe('useKanbanBoard — no extra API calls on toggle', () => {
  it('toggling hideTerminal does not call listArtifacts again', async () => {
    await board.refresh()

    const callsBefore = vi.mocked(artifactsApi.listArtifacts).mock.calls.length

    board.hideTerminal.value = false
    board.hideTerminal.value = true

    expect(vi.mocked(artifactsApi.listArtifacts).mock.calls.length).toBe(callsBefore)
  })

  it('toggling hideTerminal does not call api.get again', async () => {
    await board.refresh()

    const callsBefore = vi.mocked(api.get).mock.calls.length

    board.hideTerminal.value = false
    board.hideTerminal.value = true

    expect(vi.mocked(api.get).mock.calls.length).toBe(callsBefore)
  })
})

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

describe('useKanbanBoard — edge cases', () => {
  it('Done column is hidden even when it would be empty with hideTerminal=false', async () => {
    // Seed only active artifacts — Done column will be empty when revealed
    const activeOnly = makeArtifactsForAllStatuses().filter(
      (a) => !['done', 'rejected', 'abandoned'].includes(a.status),
    )
    vi.mocked(artifactsApi.listArtifacts).mockResolvedValue({
      items: activeOnly,
      total: activeOnly.length,
    })
    await board.refresh()

    board.hideTerminal.value = false
    // Done column appears but is empty
    const doneCol = board.columns.value.find((c) => c.name === 'Done')
    expect(doneCol).toBeDefined()
    expect(doneCol!.cards).toHaveLength(0)
  })

  it('returns empty columns when no kanban config is present', async () => {
    vi.mocked(api.get).mockResolvedValue({ kanban: null })
    await board.refresh()

    expect(board.hasConfig.value).toBe(false)
    expect(board.columns.value).toHaveLength(0)
  })
})
