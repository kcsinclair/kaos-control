/**
 * Milestone 3 — Integration tests for AgentsRunsView sorting
 *
 * Tests sorting on the agent runs table. Verifies that sortable columns work
 * correctly and that the non-sortable actions column is inert.
 *
 * These are TDD tests: they describe the expected behaviour AFTER the
 * sortable-table-columns feature is integrated into AgentsRunsView. They
 * will fail until the implementation is complete.
 *
 * Implementation assumptions (from frontend plan milestone 4):
 *  - AgentsRunsView uses useSortableTable with store.runs.
 *  - Sortable columns: Run ID (run_id/string), Agent (agent_name/string),
 *    Target (target_path/string), Status (status/string),
 *    Started (started_at/date), Elapsed (computed/number).
 *  - The last column (actions/expand) has no sort indicator and no click handler.
 *  - Expanding a detail row still works correctly after sorting.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import AgentsRunsView from '../../web/src/views/project/AgentsRunsView.vue'
import { useAgentsStore } from '../../web/src/stores/agents'
import type { AgentRunRow } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

vi.mock('@/api/agents', () => ({
  listRuns:    vi.fn().mockResolvedValue({ runs: [] }),
  listAgents:  vi.fn().mockResolvedValue({ agents: [] }),
  startRun:    vi.fn().mockResolvedValue({ run_id: 'new-run' }),
  killRun:     vi.fn().mockResolvedValue({}),
  getRunLog:   vi.fn().mockResolvedValue(''),
}))

vi.mock('vue-router', async (importActual) => {
  const actual = await importActual<typeof import('vue-router')>()
  return {
    ...actual,
    useRoute:  vi.fn(() => ({ params: { project: 'testproject' }, query: {} })),
    useRouter: vi.fn(() => ({ push: vi.fn(), replace: vi.fn() })),
  }
})

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

function makeRun(overrides: Partial<AgentRunRow> = {}): AgentRunRow {
  return {
    run_id:             'aaaaaaaa-0000-0000-0000-000000000000',
    agent_name:         'frontend-developer',
    role:               'developer',
    target_path:        'lifecycle/requirements/test.md',
    started_at:         '2024-03-01T10:00:00Z',
    finished_at:        '2024-03-01T10:01:00Z',
    status:             'done',
    stderr_tail:        '',
    artifacts_produced: [],
    ...overrides,
  }
}

function makeFixtures(): AgentRunRow[] {
  return [
    makeRun({
      run_id:      'ccc00000-0000-0000-0000-000000000000',
      agent_name:  'qa',
      started_at:  '2024-06-01T10:00:00Z',
      finished_at: '2024-06-01T10:05:00Z',  // 5m elapsed
    }),
    makeRun({
      run_id:      'aaa00000-0000-0000-0000-000000000000',
      agent_name:  'requirements-analyst',
      started_at:  '2023-01-01T10:00:00Z',
      finished_at: '2023-01-01T10:00:30Z',  // 30s elapsed
    }),
    makeRun({
      run_id:      'bbb00000-0000-0000-0000-000000000000',
      agent_name:  'backend-developer',
      started_at:  '2024-03-01T10:00:00Z',
      finished_at: '2024-03-01T10:02:00Z',  // 2m elapsed
    }),
  ]
}

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

function mountView() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes:  [{ path: '/', component: { template: '<div/>' } }],
  })

  return mount(AgentsRunsView, {
    global: { plugins: [router] },
  })
}

async function clickSortHeader(wrapper: ReturnType<typeof mountView>, label: string) {
  const headers = wrapper.findAll('th')
  const target = headers.find(th => th.text().includes(label))
  expect(target, `Could not find column header "${label}"`).toBeDefined()
  await target!.trigger('click')
}

function getAgentNames(wrapper: ReturnType<typeof mountView>): string[] {
  // Agent name is in the second column (index 1) of each data row
  return wrapper
    .findAll('tbody tr.run-row')
    .map(row => row.findAll('td')[1]?.text().trim() ?? '')
    .filter(Boolean)
}

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  setActivePinia(createPinia())
})

// ---------------------------------------------------------------------------
// Agent column sort
// ---------------------------------------------------------------------------

describe('AgentsRunsView — Agent column sort', () => {
  it('clicking Agent header sorts runs alphabetically by agent name (ascending)', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({ runs: makeFixtures() })
    await flushPromises()

    await clickSortHeader(wrapper, 'Agent')

    const names = getAgentNames(wrapper)
    expect(names[0]).toBe('requirements-analyst')
    expect(names[1]).toBe('backend-developer')
    expect(names[2]).toBe('qa')
  })

  it('clicking Agent header again sorts descending', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({ runs: makeFixtures() })
    await flushPromises()

    await clickSortHeader(wrapper, 'Agent')
    await clickSortHeader(wrapper, 'Agent')

    const names = getAgentNames(wrapper)
    expect(names[0]).toBe('qa')
    expect(names[2]).toBe('requirements-analyst')
  })
})

// ---------------------------------------------------------------------------
// Started column sort
// ---------------------------------------------------------------------------

describe('AgentsRunsView — Started column sort', () => {
  it('clicking Started header sorts runs chronologically (ascending)', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({ runs: makeFixtures() })
    await flushPromises()

    await clickSortHeader(wrapper, 'Started')

    const names = getAgentNames(wrapper)
    // requirements-analyst has earliest started_at (2023-01-01)
    expect(names[0]).toBe('requirements-analyst')
    // qa has latest started_at (2024-06-01)
    expect(names[2]).toBe('qa')
  })
})

// ---------------------------------------------------------------------------
// Elapsed column sort (numeric)
// ---------------------------------------------------------------------------

describe('AgentsRunsView — Elapsed column sort', () => {
  it('clicking Elapsed header sorts runs by computed elapsed time numerically', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({ runs: makeFixtures() })
    await flushPromises()

    await clickSortHeader(wrapper, 'Elapsed')

    const names = getAgentNames(wrapper)
    // requirements-analyst: 30s (shortest)
    // backend-developer: 2m
    // qa: 5m (longest)
    expect(names[0]).toBe('requirements-analyst')
    expect(names[2]).toBe('qa')
  })
})

// ---------------------------------------------------------------------------
// Actions column — not sortable
// ---------------------------------------------------------------------------

describe('AgentsRunsView — actions column is not sortable', () => {
  it('the last (actions/expand) column header has no sort indicator', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({ runs: makeFixtures() })
    await flushPromises()

    const headers = wrapper.findAll('th')
    const lastHeader = headers[headers.length - 1]

    // The last header (actions) must NOT have aria-sort
    expect(lastHeader.attributes('aria-sort')).toBeUndefined()
    // And the element inside (if any) should not have aria-sort
    const innerSortEl = lastHeader.find('[aria-sort]')
    expect(innerSortEl.exists()).toBe(false)
  })

  it('clicking the actions column header does not trigger a sort', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useAgentsStore()
    const fixtures = makeFixtures()
    store.$patch({ runs: fixtures })
    await flushPromises()

    const originalNames = getAgentNames(wrapper)
    const headers = wrapper.findAll('th')
    const lastHeader = headers[headers.length - 1]

    await lastHeader.trigger('click')
    await flushPromises()

    const namesAfter = getAgentNames(wrapper)
    expect(namesAfter).toEqual(originalNames)

    // No sort indicator should appear
    const sortedHeaders = wrapper.findAll('[aria-sort="ascending"], [aria-sort="descending"]')
    expect(sortedHeaders.length).toBe(0)
  })
})

// ---------------------------------------------------------------------------
// Expand row still works after sort
// ---------------------------------------------------------------------------

describe('AgentsRunsView — expanding a run detail row works after sort', () => {
  it('clicking a run row expands its detail after the table has been sorted', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({ runs: makeFixtures() })
    await flushPromises()

    // Sort by Agent name
    await clickSortHeader(wrapper, 'Agent')

    // Click the first visible run row to expand it
    const runRows = wrapper.findAll('tbody tr.run-row')
    expect(runRows.length).toBeGreaterThan(0)
    await runRows[0].trigger('click')
    await flushPromises()

    // A detail row should now be present
    const detailRow = wrapper.find('tr.run-detail')
    expect(detailRow.exists()).toBe(true)
  })
})

// ---------------------------------------------------------------------------
// Sort indicators
// ---------------------------------------------------------------------------

describe('AgentsRunsView — sort indicators', () => {
  it('exactly one column shows an active sort indicator at a time', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({ runs: makeFixtures() })
    await flushPromises()

    await clickSortHeader(wrapper, 'Agent')

    const ascHeaders  = wrapper.findAll('[aria-sort="ascending"]')
    const descHeaders = wrapper.findAll('[aria-sort="descending"]')
    expect(ascHeaders.length + descHeaders.length).toBe(1)
  })

  it('switching columns moves the active indicator', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({ runs: makeFixtures() })
    await flushPromises()

    await clickSortHeader(wrapper, 'Agent')

    // Find which header has aria-sort=ascending
    const agentHeader = wrapper.findAll('th').find(th => th.text().includes('Agent'))
    expect(agentHeader!.find('[aria-sort="ascending"]').exists() ||
           agentHeader!.attributes('aria-sort') === 'ascending').toBe(true)

    await clickSortHeader(wrapper, 'Started')

    // Agent header should no longer be the active sort
    const activeHeaders = wrapper.findAll('[aria-sort="ascending"]')
    const activeText = activeHeaders.map(h => h.text()).join(' ')
    expect(activeText).not.toContain('Agent')
  })
})
