/**
 * Milestone 4 — Integration tests for ParseErrorsView sorting
 *
 * Tests sorting on the parse errors table. Both columns (File, Error) are
 * sortable; neither is active by default.
 *
 * These are TDD tests: they describe the expected behaviour AFTER the
 * sortable-table-columns feature is integrated into ParseErrorsView. They
 * will fail until the implementation is complete.
 *
 * Implementation assumptions (from frontend plan milestone 5):
 *  - ParseErrorsView uses useSortableTable with the local `errors` ref.
 *  - Sortable columns: File (path/string), Error (message/string).
 *  - The Reload button still works after sorting.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import ParseErrorsView from '../../web/src/views/project/ParseErrorsView.vue'
import type { ParseErrorRow } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

// vi.mock factories are hoisted to the top of the file by Vitest at transform
// time — before any top-level `const` declarations run. Using vi.hoisted()
// ensures the mock function is created inside the hoist boundary so it is
// available when the factory executes.
const mockApiGet = vi.hoisted(() => vi.fn().mockResolvedValue({ errors: [] }))

vi.mock('@/api/client', () => ({
  api: { get: mockApiGet },
}))

vi.mock('vue-router', async (importActual) => {
  const actual = await importActual<typeof import('vue-router')>()
  return {
    ...actual,
    useRoute: vi.fn(() => ({ params: { project: 'testproject' }, query: {} })),
  }
})

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

function makeErrors(): ParseErrorRow[] {
  return [
    { path: 'lifecycle/requirements/zebra.md',  message: 'Missing required field: type' },
    { path: 'lifecycle/requirements/apple.md',  message: 'Unknown status value: invalid' },
    { path: 'lifecycle/requirements/mango.md',  message: 'Frontmatter parse error: bad YAML' },
    { path: 'lifecycle/requirements/banana.md', message: 'Duplicate lineage detected' },
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

  return mount(ParseErrorsView, {
    global: { plugins: [router] },
  })
}

function injectErrors(errors: ParseErrorRow[]) {
  mockApiGet.mockResolvedValue({ errors })
}

async function clickSortHeader(wrapper: ReturnType<typeof mountView>, label: string) {
  const headers = wrapper.findAll('th')
  const target = headers.find(th => th.text().includes(label))
  expect(target, `Could not find column header "${label}"`).toBeDefined()
  await target!.trigger('click')
}

function getFilePaths(wrapper: ReturnType<typeof mountView>): string[] {
  return wrapper
    .findAll('tbody tr')
    .map(row => row.find('td.cell-path')?.text().trim() ?? '')
    .filter(Boolean)
}

function getMessages(wrapper: ReturnType<typeof mountView>): string[] {
  return wrapper
    .findAll('tbody tr')
    .map(row => row.find('td.cell-msg')?.text().trim() ?? '')
    .filter(Boolean)
}

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  setActivePinia(createPinia())
})

afterEach(() => {
  vi.clearAllMocks()
})

// ---------------------------------------------------------------------------
// File column sort
// ---------------------------------------------------------------------------

describe('ParseErrorsView — File column sort', () => {
  it('clicking File header sorts errors alphabetically by path (ascending)', async () => {
    injectErrors(makeErrors())
    const wrapper = mountView()
    await flushPromises()

    await clickSortHeader(wrapper, 'File')

    const paths = getFilePaths(wrapper)
    expect(paths[0]).toContain('apple')
    expect(paths[1]).toContain('banana')
    expect(paths[2]).toContain('mango')
    expect(paths[3]).toContain('zebra')
  })

  it('clicking File header again sorts descending', async () => {
    injectErrors(makeErrors())
    const wrapper = mountView()
    await flushPromises()

    await clickSortHeader(wrapper, 'File')
    await clickSortHeader(wrapper, 'File')

    const paths = getFilePaths(wrapper)
    expect(paths[0]).toContain('zebra')
    expect(paths[3]).toContain('apple')
  })

  it('clicking File header a third time resets to original order', async () => {
    const fixtures = makeErrors()
    injectErrors(fixtures)
    const wrapper = mountView()
    await flushPromises()

    await clickSortHeader(wrapper, 'File')
    await clickSortHeader(wrapper, 'File')
    await clickSortHeader(wrapper, 'File')

    const paths = getFilePaths(wrapper)
    // Original order: zebra, apple, mango, banana
    expect(paths[0]).toContain('zebra')
    expect(paths[1]).toContain('apple')
  })
})

// ---------------------------------------------------------------------------
// Error column sort
// ---------------------------------------------------------------------------

describe('ParseErrorsView — Error column sort', () => {
  it('clicking Error header sorts errors alphabetically by message (ascending)', async () => {
    injectErrors(makeErrors())
    const wrapper = mountView()
    await flushPromises()

    await clickSortHeader(wrapper, 'Error')

    const messages = getMessages(wrapper)
    // Alphabetical by message:
    // "Duplicate lineage detected"
    // "Frontmatter parse error: bad YAML"
    // "Missing required field: type"
    // "Unknown status value: invalid"
    expect(messages[0]).toContain('Duplicate')
    expect(messages[1]).toContain('Frontmatter')
    expect(messages[2]).toContain('Missing')
    expect(messages[3]).toContain('Unknown')
  })

  it('clicking Error header again sorts descending', async () => {
    injectErrors(makeErrors())
    const wrapper = mountView()
    await flushPromises()

    await clickSortHeader(wrapper, 'Error')
    await clickSortHeader(wrapper, 'Error')

    const messages = getMessages(wrapper)
    expect(messages[0]).toContain('Unknown')
    expect(messages[3]).toContain('Duplicate')
  })
})

// ---------------------------------------------------------------------------
// Three-state toggle cycle
// ---------------------------------------------------------------------------

describe('ParseErrorsView — three-state toggle cycle', () => {
  it('goes through asc → desc → unsorted on the same column', async () => {
    injectErrors(makeErrors())
    const wrapper = mountView()
    await flushPromises()

    // asc
    await clickSortHeader(wrapper, 'File')
    expect(wrapper.findAll('[aria-sort="ascending"]').length).toBe(1)

    // desc
    await clickSortHeader(wrapper, 'File')
    expect(wrapper.findAll('[aria-sort="descending"]').length).toBe(1)

    // reset
    await clickSortHeader(wrapper, 'File')
    const activeHeaders = wrapper.findAll('[aria-sort="ascending"], [aria-sort="descending"]')
    expect(activeHeaders.length).toBe(0)
  })
})

// ---------------------------------------------------------------------------
// Sort indicators
// ---------------------------------------------------------------------------

describe('ParseErrorsView — sort indicators', () => {
  it('shows ascending indicator after first click', async () => {
    injectErrors(makeErrors())
    const wrapper = mountView()
    await flushPromises()

    await clickSortHeader(wrapper, 'File')

    const ascHeaders = wrapper.findAll('[aria-sort="ascending"]')
    expect(ascHeaders.length).toBe(1)
  })

  it('shows descending indicator after second click', async () => {
    injectErrors(makeErrors())
    const wrapper = mountView()
    await flushPromises()

    await clickSortHeader(wrapper, 'File')
    await clickSortHeader(wrapper, 'File')

    const descHeaders = wrapper.findAll('[aria-sort="descending"]')
    expect(descHeaders.length).toBe(1)
    expect(wrapper.findAll('[aria-sort="ascending"]').length).toBe(0)
  })

  it('only one column has an active indicator at a time', async () => {
    injectErrors(makeErrors())
    const wrapper = mountView()
    await flushPromises()

    await clickSortHeader(wrapper, 'File')

    const allActive = wrapper.findAll('[aria-sort="ascending"], [aria-sort="descending"]')
    expect(allActive.length).toBe(1)

    await clickSortHeader(wrapper, 'Error')

    const allActiveAfter = wrapper.findAll('[aria-sort="ascending"], [aria-sort="descending"]')
    expect(allActiveAfter.length).toBe(1)
  })
})

// ---------------------------------------------------------------------------
// Reload button works after sorting
// ---------------------------------------------------------------------------

describe('ParseErrorsView — reload after sort', () => {
  it('Reload button re-fetches data and the sort state is preserved or reset', async () => {
    mockApiGet.mockResolvedValue({ errors: makeErrors() })

    const wrapper = mountView()
    await flushPromises()

    // Sort by File
    await clickSortHeader(wrapper, 'File')
    expect(wrapper.findAll('[aria-sort="ascending"]').length).toBe(1)

    // Click reload
    const reloadBtn = wrapper.find('button[aria-label="Reload parse errors"]')
    expect(reloadBtn.exists()).toBe(true)
    await reloadBtn.trigger('click')
    await flushPromises()

    // Table should still show data (reload did not crash)
    const rows = wrapper.findAll('tbody tr')
    expect(rows.length).toBe(4)
  })
})
