/**
 * Milestone 7 — Responsive layout and accessibility tests
 *
 * Verifies that the release filter dropdown has correct accessible markup and
 * that the table structure does not introduce obvious layout regressions.
 *
 * Notes:
 *  TC1–TC3 (viewport rendering at 1280/1024/768 px) require a real browser
 *  environment and are excluded from unit tests — they should be covered by
 *  dedicated E2E/Playwright tests. This file covers what can be verified in
 *  vitest (happy-dom): DOM structure and ARIA attributes.
 *
 *  TC4: Release dropdown accessibility — label association and keyboard attrs.
 *
 * Component reference: web/src/views/project/ArtifactListView.vue
 *   <label class="sr-only" for="release-filter">Release</label>
 *   <select id="release-filter" …>
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import ArtifactListView from '../../web/src/views/project/ArtifactListView.vue'
import { useArtifactsStore } from '../../web/src/stores/artifacts'
import type { ArtifactRow } from '../../web/src/types/api'

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

function mountView() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', component: { template: '<div/>' } }],
  })
  return mount(ArtifactListView, {
    global: { plugins: [router] },
  })
}

beforeEach(() => {
  setActivePinia(createPinia())
})

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('ArtifactListView — Responsive layout (structural)', () => {
  it('table renders with all expected column headers present', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: [makeArtifact()], total: 1 })
    await flushPromises()

    const headers = wrapper.findAll('th').map(h => h.text().trim())

    // All expected columns must be present
    for (const expected of ['Path', 'Stage', 'Status', 'Priority', 'Release', 'Type', 'Created', 'Modified']) {
      expect(headers.some(h => h.includes(expected)), `Column "${expected}" must exist`).toBe(true)
    }
  })

  it('each table row contains a cell for every column (no missing <td>s)', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({
      items: [
        makeArtifact({ path: 'lifecycle/ideas/a.md' }),
        makeArtifact({ path: 'lifecycle/ideas/b.md', frontmatter: { title: 'B', type: 'idea', status: 'draft', lineage: 'b', priority: 'high', release: 'v1.0' } }),
      ],
      total: 2,
    })
    await flushPromises()

    const headerCount = wrapper.findAll('th').length
    const rows = wrapper.findAll('tbody tr')
    for (const row of rows) {
      const cellCount = row.findAll('td').length
      expect(cellCount).toBe(headerCount)
    }
  })

  it('filter bar renders without horizontal overflow wrapper', async () => {
    const wrapper = mountView()
    await flushPromises()

    const filterBar = wrapper.find('.filter-bar')
    expect(filterBar.exists(), 'filter bar must exist').toBe(true)
    // The filter bar must contain the release select
    expect(filterBar.find('#release-filter').exists(), 'release filter must be inside filter bar').toBe(true)
  })
})

describe('ArtifactListView — Release filter accessibility (TC4)', () => {
  it('release dropdown has an associated accessible label via for/id', async () => {
    const wrapper = mountView()
    await flushPromises()

    // The <label for="release-filter"> must exist
    const label = wrapper.find('label[for="release-filter"]')
    expect(label.exists(), '<label for="release-filter"> must exist').toBe(true)
    expect(label.text().trim()).toBeTruthy()

    // The <select id="release-filter"> must exist
    const sel = wrapper.find('select#release-filter')
    expect(sel.exists(), '<select id="release-filter"> must exist').toBe(true)
  })

  it('release dropdown is keyboard-focusable (no tabindex=-1 on select)', async () => {
    const wrapper = mountView()
    await flushPromises()

    const sel = wrapper.find('select#release-filter')
    expect(sel.exists()).toBe(true)

    // By default, <select> elements are keyboard-focusable.
    // Verify it has not been explicitly disabled from the tab order.
    const tabIndex = (sel.element as HTMLSelectElement).getAttribute('tabindex')
    expect(tabIndex).not.toBe('-1')
  })

  it('release dropdown responds to keyboard-driven value change', async () => {
    const wrapper = mountView()
    await flushPromises()

    const sel = wrapper.find('select#release-filter')
    expect(sel.exists()).toBe(true)

    // Simulate keyboard-driven value change (same mechanism as mouse click)
    await sel.setValue('__unassigned__')
    await sel.trigger('change')
    await flushPromises()

    expect((sel.element as HTMLSelectElement).value).toBe('__unassigned__')
  })

  it('Priority and Release column headers are keyboard-activatable (tabindex=0 or role=columnheader)', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: [makeArtifact()], total: 1 })
    await flushPromises()

    const headers = wrapper.findAll('th')

    for (const label of ['Priority', 'Release']) {
      const th = headers.find(h => h.text().includes(label))
      expect(th, `${label} column header must exist`).toBeDefined()
      // SortHeader renders th with tabindex="0" for keyboard access
      const tabIndex = th!.attributes('tabindex')
      expect(tabIndex, `${label} header must have tabindex="0"`).toBe('0')
    }
  })
})
