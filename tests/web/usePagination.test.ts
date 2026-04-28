/**
 * Milestone 2 — Unit tests for the `usePagination` composable
 *
 * Tests URL sync, computed slice indices, default values, invalid params,
 * and multi-instance prefix isolation.
 *
 * Uses vi.hoisted() + per-test mockReturnValue so the mock factory has
 * access to the mock functions before module-level `let` declarations
 * are initialized (avoiding temporal dead zone issues with vi.mock hoisting).
 *
 * Composable location: web/src/composables/usePagination.ts
 * API: usePagination({ defaultSize?, queryPrefix? })
 *      => { currentPage, pageSize, sliceStart, sliceEnd, setPage, setPageSize }
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { defineComponent, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { usePagination, type UsePaginationOptions } from '../../web/src/composables/usePagination'

// ---------------------------------------------------------------------------
// Hoisted mocks — accessible inside vi.mock factory without TDZ issues
// ---------------------------------------------------------------------------

const mocks = vi.hoisted(() => ({
  useRoute:  vi.fn(() => ({ query: {} as Record<string, string>, params: {} })),
  replace:   vi.fn(),
  useRouter: vi.fn(),
}))

// Configure useRouter once (its return value doesn't change between tests)
mocks.useRouter.mockReturnValue({ replace: mocks.replace })

vi.mock('vue-router', async (importActual) => {
  const actual = await importActual<typeof import('vue-router')>()
  return {
    ...actual,
    useRoute:  mocks.useRoute,
    useRouter: mocks.useRouter,
  }
})

// ---------------------------------------------------------------------------
// Mount helper — creates a thin wrapper component so composable runs in setup
// ---------------------------------------------------------------------------

function setupComposable(options: UsePaginationOptions = {}) {
  const pinia = createPinia()
  setActivePinia(pinia)

  let result!: ReturnType<typeof usePagination>

  mount(
    defineComponent({
      setup() {
        result = usePagination(options)
        return {}
      },
      template: '<div/>',
    }),
    { global: { plugins: [pinia] } },
  )

  return result
}

/** Set route.query before calling setupComposable() for a test. */
function withQuery(query: Record<string, string>) {
  mocks.useRoute.mockReturnValue({ query, params: {} })
}

// ---------------------------------------------------------------------------
// Setup / teardown
// ---------------------------------------------------------------------------

beforeEach(() => {
  mocks.useRoute.mockReturnValue({ query: {}, params: {} })
  mocks.replace.mockClear()
  mocks.useRouter.mockReturnValue({ replace: mocks.replace })
  setActivePinia(createPinia())
})

afterEach(() => {
  vi.clearAllMocks()
})

// ---------------------------------------------------------------------------
// Test 1 — Default values
// ---------------------------------------------------------------------------

describe('usePagination — default values', () => {
  it('currentPage defaults to 1 when no query params present', () => {
    const { currentPage } = setupComposable()
    expect(currentPage.value).toBe(1)
  })

  it('pageSize defaults to 25 when no query params present', () => {
    const { pageSize } = setupComposable()
    expect(pageSize.value).toBe(25)
  })

  it('sliceStart defaults to 0 (page 1, size 25)', () => {
    const { sliceStart } = setupComposable()
    expect(sliceStart.value).toBe(0)
  })

  it('sliceEnd defaults to 25 (page 1, size 25)', () => {
    const { sliceEnd } = setupComposable()
    expect(sliceEnd.value).toBe(25)
  })

  it('respects custom defaultSize option', () => {
    const { pageSize, sliceEnd } = setupComposable({ defaultSize: 10 })
    expect(pageSize.value).toBe(10)
    expect(sliceEnd.value).toBe(10)
  })
})

// ---------------------------------------------------------------------------
// Test 2 — Read from URL
// ---------------------------------------------------------------------------

describe('usePagination — reads initial state from URL query params', () => {
  it('reads currentPage from ?page query param', () => {
    withQuery({ page: '3', size: '50' })
    const { currentPage } = setupComposable()
    expect(currentPage.value).toBe(3)
  })

  it('reads pageSize from ?size query param', () => {
    withQuery({ page: '3', size: '50' })
    const { pageSize } = setupComposable()
    expect(pageSize.value).toBe(50)
  })

  it('computes sliceStart correctly for page=3, size=50 (expect 100)', () => {
    withQuery({ page: '3', size: '50' })
    const { sliceStart } = setupComposable()
    expect(sliceStart.value).toBe(100)
  })

  it('computes sliceEnd correctly for page=3, size=50 (expect 150)', () => {
    withQuery({ page: '3', size: '50' })
    const { sliceEnd } = setupComposable()
    expect(sliceEnd.value).toBe(150)
  })
})

// ---------------------------------------------------------------------------
// Test 3 — setPage updates URL
// ---------------------------------------------------------------------------

describe('usePagination — setPage updates URL via router.replace', () => {
  it('calling setPage(2) calls router.replace with page=2', () => {
    const { setPage } = setupComposable()
    setPage(2)
    expect(mocks.replace).toHaveBeenCalledOnce()
    const callArg = mocks.replace.mock.calls[0][0] as { query: Record<string, string> }
    expect(callArg.query.page).toBe('2')
  })

  it('calling setPage(2) preserves current size in the URL', () => {
    const { setPage } = setupComposable()
    setPage(2)
    const callArg = mocks.replace.mock.calls[0][0] as { query: Record<string, string> }
    expect(callArg.query.size).toBe('25')
  })

  it('calling setPage(2) updates currentPage ref to 2', async () => {
    const { setPage, currentPage } = setupComposable()
    setPage(2)
    await nextTick()
    expect(currentPage.value).toBe(2)
  })

  it('calling setPage with the same value does not call router.replace', () => {
    const { setPage } = setupComposable()
    setPage(1) // already on page 1
    expect(mocks.replace).not.toHaveBeenCalled()
  })
})

// ---------------------------------------------------------------------------
// Test 4 — setPageSize resets page to 1
// ---------------------------------------------------------------------------

describe('usePagination — setPageSize resets page to 1', () => {
  it('calling setPageSize(50) sets pageSize to 50', async () => {
    withQuery({ page: '3', size: '25' })
    const { setPageSize, pageSize } = setupComposable()
    setPageSize(50)
    await nextTick()
    expect(pageSize.value).toBe(50)
  })

  it('calling setPageSize(50) resets currentPage to 1', async () => {
    withQuery({ page: '3', size: '25' })
    const { setPageSize, currentPage } = setupComposable()
    setPageSize(50)
    await nextTick()
    expect(currentPage.value).toBe(1)
  })

  it('calling setPageSize(50) calls router.replace with page=1&size=50', () => {
    withQuery({ page: '3', size: '25' })
    const { setPageSize } = setupComposable()
    setPageSize(50)
    const callArg = mocks.replace.mock.calls[0][0] as { query: Record<string, string> }
    expect(callArg.query.page).toBe('1')
    expect(callArg.query.size).toBe('50')
  })
})

// ---------------------------------------------------------------------------
// Test 5 — Invalid query params fall back to defaults
// ---------------------------------------------------------------------------

describe('usePagination — invalid query params fall back to defaults', () => {
  it('non-numeric ?page falls back to page 1', () => {
    withQuery({ page: 'abc', size: '25' })
    const { currentPage } = setupComposable()
    expect(currentPage.value).toBe(1)
  })

  it('negative ?size falls back to defaultSize (25)', () => {
    withQuery({ page: '1', size: '-1' })
    const { pageSize } = setupComposable()
    expect(pageSize.value).toBe(25)
  })

  it('?page=0 falls back to page 1', () => {
    withQuery({ page: '0', size: '25' })
    const { currentPage } = setupComposable()
    expect(currentPage.value).toBe(1)
  })

  it('?size=0 falls back to defaultSize (25)', () => {
    withQuery({ page: '1', size: '0' })
    const { pageSize } = setupComposable()
    expect(pageSize.value).toBe(25)
  })
})

// ---------------------------------------------------------------------------
// Test 6 — Prefix isolation
// ---------------------------------------------------------------------------

describe('usePagination — queryPrefix isolation', () => {
  it('reads a_page/a_size when queryPrefix="a"', () => {
    withQuery({ a_page: '2', a_size: '10', b_page: '5', b_size: '50' })
    const { currentPage, pageSize } = setupComposable({ queryPrefix: 'a' })
    expect(currentPage.value).toBe(2)
    expect(pageSize.value).toBe(10)
  })

  it('reads b_page/b_size when queryPrefix="b"', () => {
    withQuery({ a_page: '2', a_size: '10', b_page: '5', b_size: '50' })
    const { currentPage, pageSize } = setupComposable({ queryPrefix: 'b' })
    expect(currentPage.value).toBe(5)
    expect(pageSize.value).toBe(50)
  })

  it('setPage on prefix-a writes a_page to router.replace (not b_page or plain page)', () => {
    withQuery({ a_page: '1', a_size: '25' })
    const { setPage } = setupComposable({ queryPrefix: 'a' })
    setPage(4)
    const callArg = mocks.replace.mock.calls[0][0] as { query: Record<string, string> }
    expect(callArg.query.a_page).toBe('4')
    expect(callArg.query).not.toHaveProperty('b_page')
    expect(callArg.query).not.toHaveProperty('page')
  })

  it('prefix-less instance uses plain page/size keys', () => {
    withQuery({ page: '7', size: '10' })
    const { currentPage, pageSize } = setupComposable()
    expect(currentPage.value).toBe(7)
    expect(pageSize.value).toBe(10)
  })

  it('setPage on prefix-less instance writes page (not a_page) to router.replace', () => {
    const { setPage } = setupComposable()
    setPage(3)
    const callArg = mocks.replace.mock.calls[0][0] as { query: Record<string, string> }
    expect(callArg.query.page).toBe('3')
    expect(callArg.query).not.toHaveProperty('a_page')
  })

  it('runs_page prefix is used for AgentsRunsView-style pagination', () => {
    withQuery({ runs_page: '2', runs_size: '25' })
    const { currentPage, pageSize } = setupComposable({ queryPrefix: 'runs' })
    expect(currentPage.value).toBe(2)
    expect(pageSize.value).toBe(25)
  })

  it('pe_page prefix is used for ParseErrorsView-style pagination', () => {
    withQuery({ pe_page: '3', pe_size: '50' })
    const { currentPage, pageSize } = setupComposable({ queryPrefix: 'pe' })
    expect(currentPage.value).toBe(3)
    expect(pageSize.value).toBe(50)
  })
})

// ---------------------------------------------------------------------------
// Slice index correctness
// ---------------------------------------------------------------------------

describe('usePagination — slice index correctness', () => {
  it('page=2, size=25: sliceStart=25, sliceEnd=50', () => {
    withQuery({ page: '2', size: '25' })
    const { sliceStart, sliceEnd } = setupComposable()
    expect(sliceStart.value).toBe(25)
    expect(sliceEnd.value).toBe(50)
  })

  it('page=1, size=10: sliceStart=0, sliceEnd=10', () => {
    withQuery({ page: '1', size: '10' })
    const { sliceStart, sliceEnd } = setupComposable()
    expect(sliceStart.value).toBe(0)
    expect(sliceEnd.value).toBe(10)
  })

  it('sliceStart and sliceEnd update reactively after setPage', async () => {
    const { sliceStart, sliceEnd, setPage } = setupComposable()
    expect(sliceStart.value).toBe(0)
    expect(sliceEnd.value).toBe(25)

    setPage(3)
    await nextTick()

    expect(sliceStart.value).toBe(50)
    expect(sliceEnd.value).toBe(75)
  })
})
