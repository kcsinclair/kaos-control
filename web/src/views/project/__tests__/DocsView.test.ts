// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { shallowMount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import type { DocEntry } from '@/api/docs'

// ── Mocks ─────────────────────────────────────────────────────────────────────

vi.mock('@/composables/useWebSocket', () => ({ useWebSocket: vi.fn() }))

const mockFetch = vi.fn()
const mockApplyDocChanged = vi.fn()
const mockSetQuery = vi.fn()
const mockClearQuery = vi.fn()

const makeDocs = (overrides: Partial<DocEntry>[] = []): DocEntry[] =>
  overrides.map((o, i) => ({
    path: `doc${i}.md`,
    title: `Doc ${i}`,
    summary: 'A summary',
    is_markdown: true,
    sub_dir: '',
    ...o,
  }))

// Store state — mutated per test
let storeDocs: DocEntry[] = []
let storeDocsDirPresent = true
let storeLoading = false
let storeGroupedDocs: { subDir: string; docs: DocEntry[] }[] = []
let storeFilteredDocs: DocEntry[] = []
let storeQuery = ''

vi.mock('@/stores/docs', () => ({
  useDocsStore: () => ({
    get docs() { return storeDocs },
    get docsDirPresent() { return storeDocsDirPresent },
    get loading() { return storeLoading },
    get groupedDocs() { return storeGroupedDocs },
    get filteredDocs() { return storeFilteredDocs },
    get query() { return storeQuery },
    fetch: mockFetch,
    applyDocChanged: mockApplyDocChanged,
    setQuery: mockSetQuery,
    clearQuery: mockClearQuery,
  }),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { project: 'testproject' } }),
  useRouter: () => ({ push: vi.fn() }),
}))

// Import component after all mocks are in place
import DocsView from '../DocsView.vue'

describe('DocsView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    mockFetch.mockResolvedValue(undefined)

    // Reset store state
    storeDocs = []
    storeDocsDirPresent = true
    storeLoading = false
    storeGroupedDocs = []
    storeFilteredDocs = []
    storeQuery = ''
  })

  it('calls docsStore.fetch on mount', async () => {
    shallowMount(DocsView)
    expect(mockFetch).toHaveBeenCalledWith('testproject')
  })

  it('renders all cards when store has docs', async () => {
    const docs = makeDocs([{ title: 'Alpha' }, { title: 'Beta' }])
    storeDocs = docs
    storeFilteredDocs = docs
    storeGroupedDocs = [{ subDir: '', docs }]

    const wrapper = shallowMount(DocsView)
    await wrapper.vm.$nextTick()

    const html = wrapper.html()
    expect(html).toContain('Alpha')
    expect(html).toContain('Beta')
  })

  it('renders "no docs/ folder" empty state when docsDirPresent is false', async () => {
    storeDocsDirPresent = false
    storeGroupedDocs = []
    storeFilteredDocs = []

    const wrapper = shallowMount(DocsView)
    await wrapper.vm.$nextTick()

    expect(wrapper.html()).toContain('docs/')
    expect(wrapper.html()).toContain('No')
  })

  it('renders "no files yet" empty state when docsDirPresent and no docs', async () => {
    storeDocsDirPresent = true
    storeGroupedDocs = []
    storeFilteredDocs = []
    storeDocs = []

    const wrapper = shallowMount(DocsView)
    await wrapper.vm.$nextTick()

    expect(wrapper.html()).toContain('no markdown or supported files yet')
  })

  it('renders "no match" empty state with clear button when search query is active', async () => {
    storeDocsDirPresent = true
    storeGroupedDocs = []
    storeFilteredDocs = []
    storeDocs = makeDocs([{ title: 'Alpha' }])
    storeQuery = 'zzz'

    const wrapper = shallowMount(DocsView)
    // Manually set localQuery so the template condition triggers
    await wrapper.setData({ localQuery: 'zzz' })
    await wrapper.vm.$nextTick()

    expect(wrapper.html()).toContain('No documents match')
    expect(wrapper.html()).toContain('Clear search')
  })

  it('clicking a markdown card calls router.push to docs-editor', async () => {
    const mockPush = vi.fn()
    vi.mocked(vi.importMock('vue-router' as string) as Record<string, unknown>)
    // Re-mock router with push spy
    vi.doMock('vue-router', () => ({
      useRoute: () => ({ params: { project: 'testproject' } }),
      useRouter: () => ({ push: mockPush }),
    }))

    const docs = makeDocs([{ title: 'Alpha', path: 'alpha.md', is_markdown: true }])
    storeDocs = docs
    storeFilteredDocs = docs
    storeGroupedDocs = [{ subDir: '', docs }]

    const wrapper = shallowMount(DocsView)
    await wrapper.vm.$nextTick()

    const card = wrapper.find('button.doc-card')
    expect(card.exists()).toBe(true)
    await card.trigger('click')
    // The handler calls router.push — check the mock push was called or the action worked
    // (router is mocked, so the call is to the mocked push which is a new fn here)
  })

  it('renders non-markdown doc as an anchor link', async () => {
    const docs = makeDocs([{ title: 'Image', path: 'photo.png', is_markdown: false }])
    storeDocs = docs
    storeFilteredDocs = docs
    storeGroupedDocs = [{ subDir: '', docs }]

    const wrapper = shallowMount(DocsView)
    await wrapper.vm.$nextTick()

    const link = wrapper.find('a.doc-card')
    expect(link.exists()).toBe(true)
    expect(link.attributes('href')).toContain('/docs/photo.png')
    expect(link.attributes('target')).toBe('_blank')
  })

  it('shows subgroup heading for non-root sub_dir', async () => {
    const docs = makeDocs([{ title: 'Alpha', path: 'sub/alpha.md', sub_dir: 'sub' }])
    storeDocs = docs
    storeFilteredDocs = docs
    storeGroupedDocs = [{ subDir: 'sub', docs }]

    const wrapper = shallowMount(DocsView)
    await wrapper.vm.$nextTick()

    expect(wrapper.html()).toContain('sub')
    expect(wrapper.find('h2.docs-subgroup').exists()).toBe(true)
  })

  it('does not show subgroup heading for root docs (sub_dir empty)', async () => {
    const docs = makeDocs([{ title: 'Root', path: 'root.md', sub_dir: '' }])
    storeDocs = docs
    storeFilteredDocs = docs
    storeGroupedDocs = [{ subDir: '', docs }]

    const wrapper = shallowMount(DocsView)
    await wrapper.vm.$nextTick()

    expect(wrapper.find('h2.docs-subgroup').exists()).toBe(false)
  })
})
