/**
 * Milestone 5 — Integration tests for ParseErrorsView pagination
 *
 * Tests that ParseErrorsView correctly paginates its error rows, that the
 * Reload button preserves pagination state, and that an empty error list
 * renders without crashing.
 *
 * Uses a real Vue Router so that usePagination({ queryPrefix: 'pe' }) can
 * read and write URL query params correctly.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import ParseErrorsView from '../../web/src/views/project/ParseErrorsView.vue'
import type { ParseErrorRow } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Module mocks — prevent real network I/O
// ---------------------------------------------------------------------------

// vi.hoisted ensures mockGet is initialized before vi.mock factory runs
const mockGet = vi.hoisted(() => vi.fn().mockResolvedValue({ errors: [] }))

vi.mock('@/api/client', () => ({
  api: { get: mockGet },
}))

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

function makeError(i: number): ParseErrorRow {
  return {
    path:    `lifecycle/ideas/error-${i}.md`,
    message: `Parse error message ${i}`,
  }
}

function makeErrors(count: number): ParseErrorRow[] {
  return Array.from({ length: count }, (_, i) => makeError(i + 1))
}

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

async function mountView(url = '/p/testproject', errors: ParseErrorRow[] = []) {
  const pinia = createPinia()
  setActivePinia(pinia)

  mockGet.mockResolvedValue({ errors })

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/p/:project', component: ParseErrorsView },
      { path: '/:pathMatch(.*)*', component: { template: '<div/>' } },
    ],
  })
  await router.push(url)
  await router.isReady()

  const wrapper = mount(ParseErrorsView, {
    global: { plugins: [pinia, router] },
  })

  return { wrapper, router }
}

function getErrorRows(wrapper: ReturnType<typeof mount>) {
  return wrapper.findAll('tbody tr.error-row')
}

// ---------------------------------------------------------------------------
// Setup / teardown
// ---------------------------------------------------------------------------

beforeEach(() => {
  setActivePinia(createPinia())
  mockGet.mockResolvedValue({ errors: [] })
})

afterEach(() => {
  vi.clearAllMocks()
})

// ---------------------------------------------------------------------------
// Test 1 — Paginated rendering
// ---------------------------------------------------------------------------

describe('ParseErrorsView — paginated rendering', () => {
  it('shows 25 error rows on page 1 when 30 errors are loaded', async () => {
    const { wrapper } = await mountView('/p/testproject', makeErrors(30))
    await flushPromises()

    expect(getErrorRows(wrapper).length).toBe(25)
  })

  it('shows 5 error rows on page 2 when 30 errors exist at default size 25', async () => {
    const { wrapper, router } = await mountView('/p/testproject', makeErrors(30))
    await flushPromises()

    await router.push('/p/testproject?pe_page=2&pe_size=25')
    await flushPromises()

    expect(getErrorRows(wrapper).length).toBe(5)
  })

  it('TablePagination renders with correct total errors count', async () => {
    const { wrapper } = await mountView('/p/testproject', makeErrors(30))
    await flushPromises()

    const pagination = wrapper.findComponent({ name: 'TablePagination' })
    expect(pagination.exists()).toBe(true)
    expect(pagination.props('totalItems')).toBe(30)
  })

  it('does not render TablePagination when errors list is empty', async () => {
    const { wrapper } = await mountView('/p/testproject', [])
    await flushPromises()

    const pagination = wrapper.findComponent({ name: 'TablePagination' })
    expect(pagination.exists()).toBe(false)
  })

  it('mounting with URL deep link ?pe_page=2&pe_size=10 renders 10 rows', async () => {
    const { wrapper } = await mountView('/p/testproject?pe_page=2&pe_size=10', makeErrors(30))
    await flushPromises()

    expect(getErrorRows(wrapper).length).toBe(10)
  })
})

// ---------------------------------------------------------------------------
// Test 2 — Reload preserves pagination
// ---------------------------------------------------------------------------

describe('ParseErrorsView — Reload preserves pagination', () => {
  it('clicking Reload does not reset the current page', async () => {
    const { wrapper, router } = await mountView('/p/testproject?pe_page=2&pe_size=25', makeErrors(30))
    await flushPromises()

    // We're on page 2
    expect(router.currentRoute.value.query.pe_page).toBe('2')

    // Click Reload — it re-fetches but should not call setPage
    mockGet.mockResolvedValueOnce({ errors: makeErrors(30) })
    const reloadBtn = wrapper.find('button[aria-label="Reload parse errors"]')
    expect(reloadBtn.exists()).toBe(true)
    await reloadBtn.trigger('click')
    await flushPromises()

    // Page should still be 2 in the URL
    expect(router.currentRoute.value.query.pe_page).toBe('2')
  })

  it('Reload re-fetches data while preserving page state', async () => {
    const { wrapper, router } = await mountView('/p/testproject?pe_page=2&pe_size=25', makeErrors(30))
    await flushPromises()

    // Still 5 rows on page 2
    expect(getErrorRows(wrapper).length).toBe(5)

    // Simulate data refresh: now there are 35 errors
    mockGet.mockResolvedValueOnce({ errors: makeErrors(35) })
    await wrapper.find('button[aria-label="Reload parse errors"]').trigger('click')
    await flushPromises()

    // Page 2 with 35 items → 10 rows (items 26–35)
    expect(getErrorRows(wrapper).length).toBe(10)
    // Still on page 2
    expect(router.currentRoute.value.query.pe_page).toBe('2')
  })

  it('Reload button is disabled while loading', async () => {
    // Use a never-resolving promise to keep loading=true
    mockGet.mockReturnValueOnce(new Promise(() => {}))

    const { wrapper } = await mountView('/p/testproject')
    // Don't flush — still loading

    const reloadBtn = wrapper.find('button[aria-label="Reload parse errors"]')
    expect(reloadBtn.exists()).toBe(true)
    // While initial load is in progress, button is disabled
    expect(reloadBtn.attributes('disabled')).toBeDefined()
  })
})

// ---------------------------------------------------------------------------
// Test 3 — Empty state
// ---------------------------------------------------------------------------

describe('ParseErrorsView — empty state', () => {
  it('shows "No parse errors" message when errors list is empty', async () => {
    const { wrapper } = await mountView('/p/testproject', [])
    await flushPromises()

    expect(wrapper.text()).toContain('No parse errors')
  })

  it('renders without crashing for 0 errors', async () => {
    const { wrapper } = await mountView('/p/testproject', [])
    await flushPromises()

    expect(wrapper.exists()).toBe(true)
    expect(wrapper.find('table').exists()).toBe(false)
  })

  it('does not show the table when errors is empty', async () => {
    const { wrapper } = await mountView('/p/testproject', [])
    await flushPromises()

    expect(wrapper.find('table.errors-table').exists()).toBe(false)
  })

  it('Reload button is always present even with no errors', async () => {
    const { wrapper } = await mountView('/p/testproject', [])
    await flushPromises()

    expect(wrapper.find('button[aria-label="Reload parse errors"]').exists()).toBe(true)
  })
})

// ---------------------------------------------------------------------------
// Pagination navigation with pe_ prefix
// ---------------------------------------------------------------------------

describe('ParseErrorsView — pagination navigation', () => {
  it('clicking Next updates pe_page to 2 in the URL', async () => {
    const { wrapper, router } = await mountView('/p/testproject', makeErrors(30))
    await flushPromises()

    const nextBtn = wrapper.find('button[aria-label="Next page"]')
    expect(nextBtn.exists()).toBe(true)
    await nextBtn.trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.query.pe_page).toBe('2')
  })

  it('clicking Previous from page 2 updates pe_page to 1 in the URL', async () => {
    const { wrapper, router } = await mountView('/p/testproject?pe_page=2&pe_size=25', makeErrors(30))
    await flushPromises()

    const prevBtn = wrapper.find('button[aria-label="Previous page"]')
    expect(prevBtn.exists()).toBe(true)
    await prevBtn.trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.query.pe_page).toBe('1')
  })

  it('pe_page and runs_page query params are independent', async () => {
    // Adding a runs_page param should not affect pe pagination
    const { wrapper } = await mountView('/p/testproject?pe_page=2&pe_size=25&runs_page=5', makeErrors(30))
    await flushPromises()

    expect(getErrorRows(wrapper).length).toBe(5)
  })
})
