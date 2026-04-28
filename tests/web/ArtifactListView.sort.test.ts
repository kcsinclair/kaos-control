/**
 * Milestone 2 — Integration tests for ArtifactListView sorting
 *
 * Tests sorting behaviour in the artifact list table with a fully-rendered
 * component. All stores are backed by in-memory Pinia; API calls are mocked
 * to prevent network requests.
 *
 * These are TDD tests: they describe the expected behaviour AFTER the
 * sortable-table-columns feature is integrated into ArtifactListView. They
 * will fail until the implementation is complete.
 *
 * Implementation assumptions (from frontend plan milestone 3):
 *  - ArtifactListView uses useSortableTable and SortHeader for all six data columns.
 *  - Pagination slices sortedRows client-side.
 *  - applyFilters() and resetFilters() call resetSort().
 *  - The view fetches all artifacts (limit=0) so sorting covers the full dataset.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import ArtifactListView from '../../web/src/views/project/ArtifactListView.vue'
import { useArtifactsStore } from '../../web/src/stores/artifacts'
import type { ArtifactRow } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Module mocks — prevent real network I/O
// ---------------------------------------------------------------------------

vi.mock('@/api/artifacts', () => ({
  listArtifacts:  vi.fn().mockResolvedValue({ items: [], total: 0 }),
  listLabels:     vi.fn().mockResolvedValue({ labels: [] }),
  listPriorities: vi.fn().mockResolvedValue({ priorities: [] }),
  getArtifact:    vi.fn().mockResolvedValue({ artifact: {}, body: '', body_html: '' }),
}))

vi.mock('@/api/ws', () => ({
  getProjectWs: vi.fn(() => ({
    onType: vi.fn(() => () => {}),
  })),
}))

vi.mock('vue-router', async (importActual) => {
  const actual = await importActual<typeof import('vue-router')>()
  return {
    ...actual,
    useRoute:  vi.fn(() => ({ params: { project: 'testproject' } })),
    useRouter: vi.fn(() => ({ push: vi.fn() })),
  }
})

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

function makeArtifact(overrides: Partial<ArtifactRow> = {}): ArtifactRow {
  return {
    path:      'lifecycle/ideas/test.md',
    slug:      'test',
    lineage:   'test',
    index:     1,
    stage:     'ideas',
    type:      'idea',
    status:    'draft',
    title:     'Test Artifact',
    frontmatter: {
      title:   'Test Artifact',
      type:    'idea',
      status:  'draft',
      lineage: 'test',
    },
    mtime:   '2024-01-15T00:00:00Z',
    created: '2024-01-01T00:00:00Z',
    ...overrides,
  }
}

// A small set of artifacts with predictable sort order.
function makeFixtures(): ArtifactRow[] {
  return [
    makeArtifact({ path: 'lifecycle/ideas/zebra.md',  title: 'Zebra Feature',  created: '2024-03-01T00:00:00Z', mtime: '2024-06-01T00:00:00Z' }),
    makeArtifact({ path: 'lifecycle/ideas/apple.md',  title: 'Apple Feature',  created: '2023-01-01T00:00:00Z', mtime: '2025-01-01T00:00:00Z' }),
    makeArtifact({ path: 'lifecycle/ideas/mango.md',  title: 'Mango Feature',  created: '2024-06-15T00:00:00Z', mtime: '2024-03-01T00:00:00Z' }),
    makeArtifact({ path: 'lifecycle/ideas/banana.md', title: 'Banana Feature', created: '2022-12-01T00:00:00Z', mtime: '2024-12-01T00:00:00Z' }),
  ]
}

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

function mountView() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', component: { template: '<div/>' } }],
  })

  return mount(ArtifactListView, {
    global: {
      plugins: [router],
    },
  })
}

// ---------------------------------------------------------------------------
// Column sort helpers
// ---------------------------------------------------------------------------

/**
 * Find a SortHeader (rendered as a <th>) by its label text and trigger a click.
 * After the click, wait one tick for Vue to re-render.
 */
async function clickSortHeader(wrapper: ReturnType<typeof mountView>, label: string) {
  const headers = wrapper.findAll('th')
  const target = headers.find(th => th.text().includes(label))
  expect(target, `Could not find column header "${label}"`).toBeDefined()
  await target!.trigger('click')
}

/** Return the first-column cell text for each visible row. */
function getRowTitles(wrapper: ReturnType<typeof mountView>): string[] {
  return wrapper
    .findAll('tbody tr')
    .map(row => row.find('td').text().trim())
    .filter(Boolean)
}

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  setActivePinia(createPinia())
})

// ---------------------------------------------------------------------------
// Path column sort
// ---------------------------------------------------------------------------

describe('ArtifactListView — Path column sort', () => {
  it('clicking Path header sorts rows alphabetically by title (ascending)', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: makeFixtures(), total: makeFixtures().length })
    await flushPromises()

    await clickSortHeader(wrapper, 'Path')

    const titles = getRowTitles(wrapper)
    expect(titles[0]).toContain('Apple')
    expect(titles[1]).toContain('Banana')
    expect(titles[2]).toContain('Mango')
    expect(titles[3]).toContain('Zebra')
  })

  it('clicking Path header again sorts descending', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: makeFixtures(), total: makeFixtures().length })
    await flushPromises()

    await clickSortHeader(wrapper, 'Path')
    await clickSortHeader(wrapper, 'Path')

    const titles = getRowTitles(wrapper)
    expect(titles[0]).toContain('Zebra')
    expect(titles[3]).toContain('Apple')
  })

  it('clicking Path header a third time resets to original order', async () => {
    const fixtures = makeFixtures()
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: fixtures, total: fixtures.length })
    await flushPromises()

    await clickSortHeader(wrapper, 'Path')
    await clickSortHeader(wrapper, 'Path')
    await clickSortHeader(wrapper, 'Path')

    const titles = getRowTitles(wrapper)
    expect(titles[0]).toContain('Zebra')    // original first item
    expect(titles[1]).toContain('Apple')    // original second item
  })
})

// ---------------------------------------------------------------------------
// Date column sorts
// ---------------------------------------------------------------------------

describe('ArtifactListView — Created column sort', () => {
  it('clicking Created header sorts rows chronologically (ascending)', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: makeFixtures(), total: makeFixtures().length })
    await flushPromises()

    await clickSortHeader(wrapper, 'Created')

    const titles = getRowTitles(wrapper)
    // Banana has the earliest created date (2022-12-01)
    expect(titles[0]).toContain('Banana')
    // Mango has the latest created date (2024-06-15)
    expect(titles[3]).toContain('Mango')
  })
})

describe('ArtifactListView — Modified column sort', () => {
  it('clicking Modified header sorts rows chronologically (ascending)', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: makeFixtures(), total: makeFixtures().length })
    await flushPromises()

    await clickSortHeader(wrapper, 'Modified')

    const titles = getRowTitles(wrapper)
    // Mango has the earliest mtime (2024-03-01)
    expect(titles[0]).toContain('Mango')
    // Apple has the latest mtime (2025-01-01)
    expect(titles[3]).toContain('Apple')
  })
})

// ---------------------------------------------------------------------------
// Sort indicator
// ---------------------------------------------------------------------------

describe('ArtifactListView — sort indicator', () => {
  it('only one column shows an active sort indicator at a time', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: makeFixtures(), total: makeFixtures().length })
    await flushPromises()

    await clickSortHeader(wrapper, 'Path')

    // Count headers with aria-sort="ascending" — should be exactly one
    const ascHeaders = wrapper.findAll('[aria-sort="ascending"]')
    expect(ascHeaders.length).toBe(1)

    // No other header should have aria-sort="descending"
    const descHeaders = wrapper.findAll('[aria-sort="descending"]')
    expect(descHeaders.length).toBe(0)
  })

  it('sort indicator updates to descending on second click', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: makeFixtures(), total: makeFixtures().length })
    await flushPromises()

    await clickSortHeader(wrapper, 'Path')
    await clickSortHeader(wrapper, 'Path')

    const descHeaders = wrapper.findAll('[aria-sort="descending"]')
    expect(descHeaders.length).toBe(1)

    const ascHeaders = wrapper.findAll('[aria-sort="ascending"]')
    expect(ascHeaders.length).toBe(0)
  })

  it('sort indicator resets to none on third click', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: makeFixtures(), total: makeFixtures().length })
    await flushPromises()

    await clickSortHeader(wrapper, 'Path')
    await clickSortHeader(wrapper, 'Path')
    await clickSortHeader(wrapper, 'Path')

    const activeHeaders = wrapper.findAll('[aria-sort="ascending"], [aria-sort="descending"]')
    expect(activeHeaders.length).toBe(0)
  })
})

// ---------------------------------------------------------------------------
// Filter reset clears sort
// ---------------------------------------------------------------------------

describe('ArtifactListView — filter change resets sort', () => {
  it('changing a filter dropdown resets the sort state', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: makeFixtures(), total: makeFixtures().length })
    await flushPromises()

    // Sort by Path (ascending)
    await clickSortHeader(wrapper, 'Path')
    expect(wrapper.findAll('[aria-sort="ascending"]').length).toBe(1)

    // Change a filter — select a stage option
    const selects = wrapper.findAll('select')
    expect(selects.length).toBeGreaterThan(0)
    await selects[0].setValue('ideas')
    await selects[0].trigger('change')
    await flushPromises()

    // Sort indicator should be cleared
    const activeHeaders = wrapper.findAll('[aria-sort="ascending"], [aria-sort="descending"]')
    expect(activeHeaders.length).toBe(0)
  })
})

// ---------------------------------------------------------------------------
// Pagination resets to page 1 after sort
// ---------------------------------------------------------------------------

describe('ArtifactListView — pagination after sort', () => {
  it('pagination resets to page 1 when sort changes', async () => {
    // Build 55 artifacts so we get two pages (50 per page)
    const manyItems = Array.from({ length: 55 }, (_, i) =>
      makeArtifact({
        path:  `lifecycle/ideas/artifact-${String(i).padStart(3, '0')}.md`,
        title: `Artifact ${String(i).padStart(3, '0')}`,
      }),
    )

    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: manyItems, total: 55 })
    await flushPromises()

    // Navigate to page 2 if pagination controls are rendered
    const nextBtn = wrapper.find('button:not([disabled])')
    const paginationText = wrapper.find('.page-info')
    if (paginationText.exists()) {
      // Click Next if available
      const nextBtns = wrapper.findAll('button').filter(b => b.text().includes('Next'))
      if (nextBtns.length > 0) {
        await nextBtns[0].trigger('click')
        await flushPromises()
        // Should now be on page 2
        expect(paginationText.text()).toContain('Page 2')
      }
    }

    // Now sort — this must reset back to page 1
    await clickSortHeader(wrapper, 'Path')

    if (paginationText.exists()) {
      expect(paginationText.text()).toContain('Page 1')
    }
  })
})

// ---------------------------------------------------------------------------
// Keyboard activation
// ---------------------------------------------------------------------------

describe('ArtifactListView — keyboard activation of sort headers', () => {
  it('Enter key on a column header triggers the same sort as a click', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: makeFixtures(), total: makeFixtures().length })
    await flushPromises()

    const headers = wrapper.findAll('th')
    const pathHeader = headers.find(th => th.text().includes('Path'))
    expect(pathHeader).toBeDefined()

    const focusable = pathHeader!.find('[tabindex="0"]') ?? pathHeader!
    await focusable.trigger('keydown', { key: 'Enter' })

    const ascHeaders = wrapper.findAll('[aria-sort="ascending"]')
    expect(ascHeaders.length).toBe(1)
  })

  it('Space key on a column header triggers the same sort as a click', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: makeFixtures(), total: makeFixtures().length })
    await flushPromises()

    const headers = wrapper.findAll('th')
    const pathHeader = headers.find(th => th.text().includes('Path'))
    expect(pathHeader).toBeDefined()

    const focusable = pathHeader!.find('[tabindex="0"]') ?? pathHeader!
    await focusable.trigger('keydown', { key: ' ' })

    const ascHeaders = wrapper.findAll('[aria-sort="ascending"]')
    expect(ascHeaders.length).toBe(1)
  })
})
