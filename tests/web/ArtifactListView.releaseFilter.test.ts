/**
 * Milestone 6 — Release filter interaction tests
 *
 * Verifies that the release filter dropdown in ArtifactListView:
 *  - is populated with distinct release values from the releases store
 *  - triggers a re-fetch with the correct filter parameters on selection
 *  - composes correctly with status filters and text search
 *  - resets sort state on selection
 *  - returns to "All releases" when reset button is clicked
 *  - shows an empty-state message when no artifacts match after filtering
 *
 * Component reference: web/src/views/project/ArtifactListView.vue
 *   <label class="sr-only" for="release-filter">Release</label>
 *   <select id="release-filter" v-model="selectedRelease" @change="applyFilters">
 *     <option value="">All releases</option>
 *     <option v-for="r in releasesStore.releases" …>{{ r.name }}</option>
 *     <option value="__unassigned__">Unassigned</option>
 *   </select>
 *   <button class="btn-ghost" @click="resetFilters">Reset</button>
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import ArtifactListView from '../../web/src/views/project/ArtifactListView.vue'
import TextFilter from '../../web/src/components/TextFilter.vue'
import { useArtifactsStore } from '../../web/src/stores/artifacts'
import { useReleasesStore } from '../../web/src/stores/releases'
import type { ArtifactRow } from '../../web/src/types/api'
import type { Release } from '../../web/src/types/release'
import * as artifactsApi from '@/api/artifacts'

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

vi.mock('@/api/artifacts', () => ({
  listArtifacts:  vi.fn().mockResolvedValue({ items: [], total: 0 }),
  listLabels:     vi.fn().mockResolvedValue({ labels: [] }),
  listPriorities: vi.fn().mockResolvedValue({ priorities: [] }),
  getArtifact:    vi.fn().mockResolvedValue({ artifact: {}, body: '', body_html: '' }),
}))

vi.mock('@/api/releases', () => ({
  listReleases: vi.fn().mockResolvedValue([]),
}))

vi.mock('@/api/ws', () => ({
  getProjectWs: vi.fn(() => ({
    onType: vi.fn(() => () => {}),
    on:     vi.fn(() => () => {}),
  })),
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
// Helpers
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

function makeRelease(id: number, name: string): Release {
  return {
    id,
    name,
    status: 'planned',
    start_date: null,
    end_date: null,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  }
}

function mountView() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', component: { template: '<div/>' } }],
  })
  return mount(ArtifactListView, {
    global: { plugins: [router] },
  })
}

/** Returns the release filter <select> element. */
function getReleaseSelect(wrapper: ReturnType<typeof mountView>) {
  return wrapper.find('#release-filter')
}

/** Returns all <option> text values from the release dropdown. */
function getReleaseOptions(wrapper: ReturnType<typeof mountView>): string[] {
  return wrapper.findAll('#release-filter option').map(o => o.text().trim())
}

async function clickSortHeader(wrapper: ReturnType<typeof mountView>, label: string) {
  const headers = wrapper.findAll('th')
  const target = headers.find(th => th.text().includes(label))
  expect(target, `Could not find column header "${label}"`).toBeDefined()
  await target!.trigger('click')
}

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  setActivePinia(createPinia())
  vi.mocked(artifactsApi.listArtifacts).mockResolvedValue({ items: [], total: 0 })
})

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('ArtifactListView — Release filter dropdown', () => {
  it('TC1: filter dropdown contains All releases, release names from store, and Unassigned', async () => {
    const wrapper = mountView()
    await flushPromises()

    // Patch releases store after mount (simulating data load)
    const releasesStore = useReleasesStore()
    releasesStore.$patch({
      releases: [makeRelease(1, 'v1.0'), makeRelease(2, 'v2.0'), makeRelease(3, 'alpha')],
    })
    await flushPromises()

    const options = getReleaseOptions(wrapper)
    expect(options).toContain('All releases')
    expect(options).toContain('v1.0')
    expect(options).toContain('v2.0')
    expect(options).toContain('alpha')
    expect(options).toContain('Unassigned')
  })

  it('TC2: selecting a release triggers fetchList with release filter', async () => {
    const wrapper = mountView()
    await flushPromises()

    const releasesStore = useReleasesStore()
    releasesStore.$patch({
      releases: [makeRelease(1, 'v1.0'), makeRelease(2, 'v2.0')],
    })
    await flushPromises()

    vi.mocked(artifactsApi.listArtifacts).mockClear()

    const sel = getReleaseSelect(wrapper)
    await sel.setValue('v1.0')
    await sel.trigger('change')
    await flushPromises()

    expect(vi.mocked(artifactsApi.listArtifacts)).toHaveBeenCalled()
    const lastCall = vi.mocked(artifactsApi.listArtifacts).mock.calls.at(-1)!
    const filter = lastCall[1] as Record<string, unknown>
    expect(filter.release).toBe('v1.0')
  })

  it('TC3: selecting Unassigned passes release=__unassigned__ to fetchList', async () => {
    const wrapper = mountView()
    await flushPromises()

    vi.mocked(artifactsApi.listArtifacts).mockClear()

    const sel = getReleaseSelect(wrapper)
    await sel.setValue('__unassigned__')
    await sel.trigger('change')
    await flushPromises()

    const lastCall = vi.mocked(artifactsApi.listArtifacts).mock.calls.at(-1)!
    const filter = lastCall[1] as Record<string, unknown>
    expect(filter.release).toBe('__unassigned__')
  })

  it('TC4: selecting All releases passes no release filter to fetchList', async () => {
    const wrapper = mountView()
    await flushPromises()

    const releasesStore = useReleasesStore()
    releasesStore.$patch({ releases: [makeRelease(1, 'v1.0')] })
    await flushPromises()

    // First select a specific release
    const sel = getReleaseSelect(wrapper)
    await sel.setValue('v1.0')
    await sel.trigger('change')
    await flushPromises()

    vi.mocked(artifactsApi.listArtifacts).mockClear()

    // Now reset to "All releases"
    await sel.setValue('')
    await sel.trigger('change')
    await flushPromises()

    const lastCall = vi.mocked(artifactsApi.listArtifacts).mock.calls.at(-1)!
    const filter = lastCall[1] as Record<string, unknown>
    // release=undefined means the parameter is omitted from the query
    expect(filter.release == null || filter.release === '').toBe(true)
  })

  it('TC5: composing status=draft and release=v1.0 filters passes both to fetchList', async () => {
    const wrapper = mountView()
    await flushPromises()

    const releasesStore = useReleasesStore()
    releasesStore.$patch({ releases: [makeRelease(1, 'v1.0')] })
    await flushPromises()

    vi.mocked(artifactsApi.listArtifacts).mockClear()

    // Set status filter (first select in the filter bar is stage, second is status)
    const selects = wrapper.findAll('select')
    // status select is the second one (index 1: stage=0, status=1)
    const statusSel = selects[1]
    await statusSel.setValue('draft')
    await statusSel.trigger('change')
    await flushPromises()

    // Set release filter
    const relSel = getReleaseSelect(wrapper)
    await relSel.setValue('v1.0')
    await relSel.trigger('change')
    await flushPromises()

    const lastCall = vi.mocked(artifactsApi.listArtifacts).mock.calls.at(-1)!
    const filter = lastCall[1] as Record<string, unknown>
    expect(filter.release).toBe('v1.0')
    expect(filter.status).toBe('draft')
  })

  it('TC6: composing release filter with text search passes both to fetchList', async () => {
    const wrapper = mountView()
    await flushPromises()

    const releasesStore = useReleasesStore()
    releasesStore.$patch({ releases: [makeRelease(1, 'v1.0')] })
    await flushPromises()

    // Set release filter
    const relSel = getReleaseSelect(wrapper)
    await relSel.setValue('v1.0')
    await relSel.trigger('change')
    await flushPromises()

    vi.mocked(artifactsApi.listArtifacts).mockClear()

    // Emit update:modelValue directly on the TextFilter sub-component to bypass
    // its 200 ms debounce (flushPromises only drains microtasks, not setTimeout).
    const textFilterWrapper = wrapper.findComponent(TextFilter)
    expect(textFilterWrapper.exists(), 'TextFilter component must be present').toBe(true)
    textFilterWrapper.vm.$emit('update:modelValue', 'login')
    await flushPromises()

    const lastCall = vi.mocked(artifactsApi.listArtifacts).mock.calls.at(-1)!
    const filter = lastCall[1] as Record<string, unknown>
    expect(filter.release).toBe('v1.0')
    expect(filter.q).toBe('login')
  })

  it('TC7: changing the release filter resets the active sort state', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({
      items: [
        makeArtifact({ path: 'lifecycle/ideas/a.md', title: 'Apple' }),
        makeArtifact({ path: 'lifecycle/ideas/b.md', title: 'Banana' }),
      ],
      total: 2,
    })
    await flushPromises()

    // Sort by Path ascending
    await clickSortHeader(wrapper, 'Path')
    expect(wrapper.findAll('[aria-sort="ascending"]').length).toBe(1)

    // Change the release filter
    const sel = getReleaseSelect(wrapper)
    await sel.setValue('__unassigned__')
    await sel.trigger('change')
    await flushPromises()

    // Sort indicator must be gone
    const active = wrapper.findAll('[aria-sort="ascending"], [aria-sort="descending"]')
    expect(active.length).toBe(0)
  })

  it('TC8: clicking Reset returns release dropdown to All releases', async () => {
    const wrapper = mountView()
    await flushPromises()

    const releasesStore = useReleasesStore()
    releasesStore.$patch({ releases: [makeRelease(1, 'v1.0')] })
    await flushPromises()

    // Select a specific release
    const sel = getReleaseSelect(wrapper)
    await sel.setValue('v1.0')
    await sel.trigger('change')
    await flushPromises()
    expect((sel.element as HTMLSelectElement).value).toBe('v1.0')

    // Click the Reset button
    const resetBtn = wrapper.find('.btn-ghost')
    expect(resetBtn.exists(), 'Reset button must be present').toBe(true)
    await resetBtn.trigger('click')
    await flushPromises()

    // Release dropdown must be back to "All releases" (empty value)
    expect((sel.element as HTMLSelectElement).value).toBe('')
  })

  it('TC9: empty-state message appears when no artifacts match the applied filter', async () => {
    // Mock listArtifacts to return empty set
    vi.mocked(artifactsApi.listArtifacts).mockResolvedValue({ items: [], total: 0 })

    const wrapper = mountView()
    await flushPromises()

    // The store starts empty — no items → empty state
    const store = useArtifactsStore()
    expect(store.items.length).toBe(0)

    // The empty-state message should be visible
    const emptyMsg = wrapper.find('.state-msg')
    expect(emptyMsg.exists(), 'empty-state message must be present').toBe(true)
    expect(emptyMsg.text()).toContain('No artifacts found')
  })
})
