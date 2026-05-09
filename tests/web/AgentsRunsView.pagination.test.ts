// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 4 — Integration tests for AgentsRunsView pagination
 *
 * Tests that AgentsRunsView correctly paginates its runs table, that
 * expanded row details work within paginated results, and that URL
 * deep-linking via the `runs_page` / `runs_size` prefixed query params works.
 *
 * Uses a real Vue Router so that usePagination({ queryPrefix: 'runs' }) can
 * read and write URL query params correctly.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import AgentsRunsView from '../../web/src/views/project/AgentsRunsView.vue'
import { useAgentsStore } from '../../web/src/stores/agents'
import type { AgentRunRow } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Module mocks — prevent real network I/O
// ---------------------------------------------------------------------------

vi.mock('@/api/agents', () => ({
  listRuns:   vi.fn().mockResolvedValue({ runs: [] }),
  listAgents: vi.fn().mockResolvedValue({ agents: [] }),
  startRun:   vi.fn().mockResolvedValue({ run_id: 'new-run' }),
  killRun:    vi.fn().mockResolvedValue({}),
  getRunLog:  vi.fn().mockResolvedValue(''),
}))

// AgentsRunsView mounts useProjectConfigStore() which calls getRoles();
// without this mock the call leaks a real fetch (ECONNREFUSED in tests).
vi.mock('@/api/config', () => ({
  getRoles: vi.fn().mockResolvedValue({ roles: [] }),
}))

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

function makeRun(i: number, overrides: Partial<AgentRunRow> = {}): AgentRunRow {
  return {
    run_id:             `run-${String(i).padStart(4, '0')}-0000-0000-0000-000000000000`,
    agent_name:         'frontend-developer',
    role:               'developer',
    target_path:        `lifecycle/requirements/item-${i}.md`,
    started_at:         `2024-03-01T${String(10 + (i % 14)).padStart(2, '0')}:00:00Z`,
    finished_at:        `2024-03-01T${String(10 + (i % 14)).padStart(2, '0')}:01:00Z`,
    status:             'done',
    stderr_tail:        '',
    artifacts_produced: [],
    ...overrides,
  }
}

function makeRuns(count: number): AgentRunRow[] {
  return Array.from({ length: count }, (_, i) => makeRun(i + 1))
}

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

async function mountView(url = '/p/testproject') {
  const pinia = createPinia()
  setActivePinia(pinia)

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/p/:project', component: AgentsRunsView },
      { path: '/:pathMatch(.*)*', component: { template: '<div/>' } },
    ],
  })
  await router.push(url)
  await router.isReady()

  const wrapper = mount(AgentsRunsView, {
    global: { plugins: [pinia, router] },
  })

  return { wrapper, router }
}

function getRunRows(wrapper: ReturnType<typeof mount>) {
  return wrapper.findAll('tbody tr.run-row')
}

// ---------------------------------------------------------------------------
// Setup / teardown
// ---------------------------------------------------------------------------

beforeEach(() => {
  setActivePinia(createPinia())
})

afterEach(() => {
  vi.clearAllMocks()
})

// ---------------------------------------------------------------------------
// Test 1 — Paginated rendering
// ---------------------------------------------------------------------------

describe('AgentsRunsView — paginated rendering', () => {
  it('shows 25 run rows on page 1 when 30 runs are loaded', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({ runs: makeRuns(30), loading: false })
    await flushPromises()

    expect(getRunRows(wrapper).length).toBe(25)
  })

  it('shows 5 run rows on page 2 when 30 runs exist at default size 25', async () => {
    const { wrapper, router } = await mountView()
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({ runs: makeRuns(30), loading: false })
    await flushPromises()

    await router.push('/p/testproject?runs_page=2&runs_size=25')
    await flushPromises()

    expect(getRunRows(wrapper).length).toBe(5)
  })

  it('TablePagination renders with correct total runs count', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({ runs: makeRuns(30), loading: false })
    await flushPromises()

    const pagination = wrapper.findComponent({ name: 'TablePagination' })
    expect(pagination.exists()).toBe(true)
    expect(pagination.props('totalItems')).toBe(30)
  })

  it('does not render TablePagination when there are no runs', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({ runs: [], loading: false })
    await flushPromises()

    const pagination = wrapper.findComponent({ name: 'TablePagination' })
    expect(pagination.exists()).toBe(false)
  })
})

// ---------------------------------------------------------------------------
// Test 2 — Expand row on page 2
// ---------------------------------------------------------------------------

describe('AgentsRunsView — expand row on page 2', () => {
  it('expanding a run row on page 2 shows a detail panel', async () => {
    const { wrapper, router } = await mountView()
    await flushPromises()

    const store = useAgentsStore()
    const runs = makeRuns(30)
    // Give run 26 (first on page 2) some output to show
    runs[25] = { ...runs[25], stderr_tail: 'Error output for run 26' }
    store.$patch({ runs, loading: false })
    await flushPromises()

    // Navigate to page 2
    await router.push('/p/testproject?runs_page=2&runs_size=25')
    await flushPromises()

    const runRows = getRunRows(wrapper)
    expect(runRows.length).toBe(5)

    // Click first run row on page 2 to expand it
    await runRows[0].trigger('click')
    await flushPromises()

    const detailRow = wrapper.find('tr.run-detail')
    expect(detailRow.exists()).toBe(true)
  })

  it('expanded run detail on page 2 shows the correct run data', async () => {
    const { wrapper, router } = await mountView()
    await flushPromises()

    const store = useAgentsStore()
    const runs = makeRuns(30)
    runs[25] = {
      ...runs[25],
      stderr_tail: 'Unique error for page-2 run',
      artifacts_produced: ['lifecycle/tests/my-test.md'],
    }
    store.$patch({ runs, loading: false })
    await flushPromises()

    await router.push('/p/testproject?runs_page=2&runs_size=25')
    await flushPromises()

    const runRows = getRunRows(wrapper)
    await runRows[0].trigger('click')
    await flushPromises()

    const detailRow = wrapper.find('tr.run-detail')
    expect(detailRow.text()).toContain('Unique error for page-2 run')
  })

  it('expanding a row on page 2 does not affect rows on page 1', async () => {
    const { wrapper, router } = await mountView()
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({ runs: makeRuns(30), loading: false })
    await flushPromises()

    // Navigate to page 2, expand a row
    await router.push('/p/testproject?runs_page=2&runs_size=25')
    await flushPromises()

    await getRunRows(wrapper)[0].trigger('click')
    await flushPromises()

    expect(wrapper.find('tr.run-detail').exists()).toBe(true)

    // Navigate back to page 1 — no detail row should be visible (expandedRun is different)
    await router.push('/p/testproject?runs_page=1&runs_size=25')
    await flushPromises()

    const runRowsPage1 = getRunRows(wrapper)
    expect(runRowsPage1.length).toBe(25)
    // The expanded run from page 2 is not visible on page 1
    expect(wrapper.find('tr.run-detail').exists()).toBe(false)
  })
})

// ---------------------------------------------------------------------------
// Test 3 — URL deep link with runs_ prefix
// ---------------------------------------------------------------------------

describe('AgentsRunsView — URL deep link', () => {
  it('mounting with ?runs_page=2&runs_size=10 renders 10 rows', async () => {
    const { wrapper } = await mountView('/p/testproject?runs_page=2&runs_size=10')
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({ runs: makeRuns(30), loading: false })
    await flushPromises()

    expect(getRunRows(wrapper).length).toBe(10)
  })

  it('mounting with ?runs_page=2&runs_size=10 shows runs 11–20', async () => {
    const { wrapper } = await mountView('/p/testproject?runs_page=2&runs_size=10')
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({ runs: makeRuns(30), loading: false })
    await flushPromises()

    const rows = getRunRows(wrapper)
    // First row on page 2 with size 10 = run 11 (run-0011-…)
    expect(rows[0].text()).toContain('run-0011')
  })

  it('runs_page and page query params are independent', async () => {
    // Using runs_page should not conflict with a plain page param
    const { wrapper } = await mountView('/p/testproject?runs_page=2&runs_size=25&page=5')
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({ runs: makeRuns(30), loading: false })
    await flushPromises()

    // The runs view uses runs_page prefix, so page=5 is irrelevant here
    // runs_page=2 with runs_size=25 → 5 rows (runs 26–30)
    expect(getRunRows(wrapper).length).toBe(5)
  })
})

// ---------------------------------------------------------------------------
// Pagination navigation
// ---------------------------------------------------------------------------

describe('AgentsRunsView — pagination navigation', () => {
  it('clicking Next on page 1 updates runs_page to 2 in the URL', async () => {
    const { wrapper, router } = await mountView()
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({ runs: makeRuns(30), loading: false })
    await flushPromises()

    const nextBtn = wrapper.find('button[aria-label="Next page"]')
    expect(nextBtn.exists()).toBe(true)
    await nextBtn.trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.query.runs_page).toBe('2')
  })

  it('clicking Previous from page 2 updates runs_page to 1 in the URL', async () => {
    const { wrapper, router } = await mountView('/p/testproject?runs_page=2&runs_size=25')
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({ runs: makeRuns(30), loading: false })
    await flushPromises()

    const prevBtn = wrapper.find('button[aria-label="Previous page"]')
    expect(prevBtn.exists()).toBe(true)
    await prevBtn.trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.query.runs_page).toBe('1')
  })
})
