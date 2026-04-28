/**
 * Milestone 3 — Integration tests for ArtifactListView pagination
 *
 * Tests that ArtifactListView correctly paginates its artifact rows,
 * preserves state through navigation, and resets the page when filters change.
 *
 * Uses a real Vue Router so that usePagination can read and write URL query
 * params correctly. The artifacts API is mocked to avoid network calls.
 *
 * Timing note: ArtifactListView.onMounted calls store.fetchList() which is
 * async. We always call flushPromises() after mounting so the initial fetch
 * completes before patching the store with test fixtures.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import ArtifactListView from '../../web/src/views/project/ArtifactListView.vue'
import { useArtifactsStore } from '../../web/src/stores/artifacts'
import { listArtifacts as listArtifactsMock } from '../../web/src/api/artifacts'
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

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

function makeArtifact(i: number, overrides: Partial<ArtifactRow> = {}): ArtifactRow {
  return {
    path:        `lifecycle/ideas/item-${i}.md`,
    slug:        `item-${i}`,
    lineage:     `item-${i}`,
    index:       i,
    stage:       'ideas',
    type:        'idea',
    status:      'draft',
    title:       `Artifact ${i}`,
    frontmatter: { title: `Artifact ${i}`, type: 'idea', status: 'draft', lineage: `item-${i}` },
    mtime:       '2026-01-01T00:00:00Z',
    created:     '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

function makeItems(count: number, overrides: Partial<ArtifactRow> = {}): ArtifactRow[] {
  return Array.from({ length: count }, (_, i) => makeArtifact(i + 1, overrides))
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
      { path: '/p/:project', component: ArtifactListView },
      { path: '/:pathMatch(.*)*', component: { template: '<div/>' } },
    ],
  })
  await router.push(url)
  await router.isReady()

  const wrapper = mount(ArtifactListView, {
    global: { plugins: [pinia, router] },
  })

  return { wrapper, router }
}

/**
 * Mount the view, wait for onMounted fetch to complete, then populate the
 * store with `count` test items and wait for the DOM to update.
 */
async function mountWithItems(count: number, url = '/p/testproject') {
  const { wrapper, router } = await mountView(url)
  await flushPromises() // onMounted fetch completes (empty list from default mock)
  const store = useArtifactsStore()
  store.$patch({ items: makeItems(count), loading: false })
  await flushPromises()
  return { wrapper, router, store }
}

function getArtifactRows(wrapper: ReturnType<typeof mount>) {
  return wrapper.findAll('tbody tr.artifact-row')
}

// ---------------------------------------------------------------------------
// Setup / teardown
// ---------------------------------------------------------------------------

beforeEach(() => {
  setActivePinia(createPinia())
  vi.mocked(listArtifactsMock).mockResolvedValue({ items: [], total: 0 })
})

afterEach(() => {
  vi.clearAllMocks()
})

// ---------------------------------------------------------------------------
// Test 1 — Paginated rendering
// ---------------------------------------------------------------------------

describe('ArtifactListView — paginated rendering', () => {
  it('shows 25 rows on page 1 when 60 items are loaded', async () => {
    const { wrapper } = await mountWithItems(60)
    expect(getArtifactRows(wrapper).length).toBe(25)
  })

  it('shows 10 rows on page 3 of 60 items at size 25 (items 51–60)', async () => {
    const { wrapper, router } = await mountWithItems(60)

    await router.push('/p/testproject?page=3&size=25')
    await flushPromises()

    expect(getArtifactRows(wrapper).length).toBe(10)
  })

  it('first row on page 2 contains the correct artifact (item 26)', async () => {
    const { wrapper, router } = await mountWithItems(60)

    await router.push('/p/testproject?page=2&size=25')
    await flushPromises()

    const rows = getArtifactRows(wrapper)
    expect(rows.length).toBe(25)
    expect(rows[0].text()).toContain('Artifact 26')
  })

  it('TablePagination receives correct totalItems prop', async () => {
    const { wrapper } = await mountWithItems(42)

    const pagination = wrapper.findComponent({ name: 'TablePagination' })
    expect(pagination.exists()).toBe(true)
    expect(pagination.props('totalItems')).toBe(42)
  })
})

// ---------------------------------------------------------------------------
// Test 2 — Filter + pagination
// ---------------------------------------------------------------------------

describe('ArtifactListView — filter changes reset page', () => {
  it('changing stage filter resets page to 1', async () => {
    const { wrapper, router } = await mountWithItems(60, '/p/testproject?page=3&size=25')

    expect(router.currentRoute.value.query.page).toBe('3')

    // Mock API to return 15 filtered items for the next fetch
    vi.mocked(listArtifactsMock).mockResolvedValueOnce({ items: makeItems(15), total: 15 })

    // Change the stage filter dropdown (triggers applyFilters → setPage(1) + fetchList)
    const stageSelect = wrapper.findAll('.filter-bar select')[0]
    await stageSelect.setValue('ideas')
    await flushPromises()

    expect(router.currentRoute.value.query.page).toBe('1')
  })

  it('shows filtered item count after filter is applied', async () => {
    const { wrapper } = await mountWithItems(60)

    vi.mocked(listArtifactsMock).mockResolvedValueOnce({ items: makeItems(15), total: 15 })

    const stageSelect = wrapper.findAll('.filter-bar select')[0]
    await stageSelect.setValue('ideas')
    await flushPromises()

    expect(getArtifactRows(wrapper).length).toBe(15)
  })

  it('show-completed toggle resets page to 1', async () => {
    const { wrapper, router } = await mountView('/p/testproject?page=3&size=25')
    await flushPromises()

    const store = useArtifactsStore()
    const items = [
      ...makeItems(55, { status: 'draft' }),
      ...Array.from({ length: 5 }, (_, i) =>
        makeArtifact(100 + i, { status: 'done', slug: `done-${i}`, path: `lifecycle/ideas/done-${i}.md` })
      ),
    ]
    store.$patch({ items, loading: false })
    await flushPromises()

    // Toggle "Show completed" → watcher calls setPage(1)
    const checkbox = wrapper.find('input[type="checkbox"]')
    expect(checkbox.exists()).toBe(true)
    await checkbox.setValue(true)
    await flushPromises()

    expect(router.currentRoute.value.query.page).toBe('1')
  })
})

// ---------------------------------------------------------------------------
// Test 3 — State preservation via URL
// ---------------------------------------------------------------------------

describe('ArtifactListView — state preservation via URL', () => {
  it('clicking Next writes page=2 to the URL', async () => {
    const { wrapper, router } = await mountWithItems(60)

    const nextBtn = wrapper.find('button[aria-label="Next page"]')
    expect(nextBtn.exists()).toBe(true)
    await nextBtn.trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.query.page).toBe('2')
  })

  it('navigating away and back restores the page=2 URL', async () => {
    const { wrapper, router } = await mountWithItems(60)

    // Go to page 2
    await wrapper.find('button[aria-label="Next page"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query.page).toBe('2')

    // Navigate away
    await router.push('/p/testproject/artifacts/lifecycle/ideas/item-1.md')
    await flushPromises()

    // Go back
    await router.go(-1)
    await flushPromises()

    expect(router.currentRoute.value.query.page).toBe('2')
  })
})

// ---------------------------------------------------------------------------
// Test 4 — Page reset on filter change
// ---------------------------------------------------------------------------

describe('ArtifactListView — page reset on filter change', () => {
  it('reset filters button resets to page 1', async () => {
    const { wrapper, router } = await mountWithItems(60, '/p/testproject?page=3&size=25')

    vi.mocked(listArtifactsMock).mockResolvedValueOnce({ items: makeItems(60), total: 60 })

    const resetBtn = wrapper.find('button.btn-ghost')
    expect(resetBtn.exists()).toBe(true)
    await resetBtn.trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.query.page).toBe('1')
  })
})

// ---------------------------------------------------------------------------
// Test 5 — Row interaction targets the correct paginated artifact
// ---------------------------------------------------------------------------

describe('ArtifactListView — row click targets the correct paginated artifact', () => {
  it('clicking first row on page 2 navigates to item-26', async () => {
    const { wrapper, router } = await mountWithItems(60, '/p/testproject?page=2&size=25')

    const rows = getArtifactRows(wrapper)
    expect(rows.length).toBe(25)

    await rows[0].trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.fullPath).toContain('item-26')
  })

  it('clicking last row on page 2 navigates to item-50 (not item-75 from page 3)', async () => {
    const { wrapper, router } = await mountWithItems(60, '/p/testproject?page=2&size=25')

    const rows = getArtifactRows(wrapper)
    await rows[rows.length - 1].trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.fullPath).toContain('item-50')
    expect(router.currentRoute.value.fullPath).not.toContain('item-75')
  })
})

// ---------------------------------------------------------------------------
// Test 6 — URL deep link
// ---------------------------------------------------------------------------

describe('ArtifactListView — URL deep link', () => {
  it('mounting with ?page=2&size=10 renders 10 rows', async () => {
    const { wrapper } = await mountWithItems(60, '/p/testproject?page=2&size=10')
    expect(getArtifactRows(wrapper).length).toBe(10)
  })

  it('mounting with ?page=2&size=10 shows items 11–20', async () => {
    const { wrapper } = await mountWithItems(60, '/p/testproject?page=2&size=10')

    const rows = getArtifactRows(wrapper)
    expect(rows[0].text()).toContain('Artifact 11')
    expect(rows[9].text()).toContain('Artifact 20')
  })

  it('mounting with ?page=3&size=25 on 60 items shows final 10 items (51–60)', async () => {
    const { wrapper } = await mountWithItems(60, '/p/testproject?page=3&size=25')

    expect(getArtifactRows(wrapper).length).toBe(10)
    const rows = getArtifactRows(wrapper)
    expect(rows[0].text()).toContain('Artifact 51')
  })
})
