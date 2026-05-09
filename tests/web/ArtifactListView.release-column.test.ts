// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 3 — Release column display tests
 *
 * Verifies that the Release column renders release values as plain text and
 * shows a dash when absent. Also asserts column position relative to Priority.
 *
 * Component reference: web/src/views/project/ArtifactListView.vue
 *  Release cell: <td class="cell-release muted">{{ row.frontmatter?.release || '—' }}</td>
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

describe('ArtifactListView — Release column display', () => {
  it('TC1: renders the release value as plain text in the Release column', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({
      items: [
        makeArtifact({
          path: 'lifecycle/ideas/with-release.md',
          lineage: 'with-release',
          frontmatter: { title: 'With Release', type: 'idea', status: 'draft', lineage: 'with-release', release: 'v1.0' },
        }),
      ],
      total: 1,
    })
    await flushPromises()

    const cell = wrapper.find('.cell-release')
    expect(cell.exists(), '.cell-release cell must be present').toBe(true)
    expect(cell.text()).toBe('v1.0')
  })

  it('TC2: renders a dash when the artifact has no release field', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({
      items: [
        makeArtifact({
          path: 'lifecycle/ideas/no-release.md',
          lineage: 'no-release',
          frontmatter: { title: 'No Release', type: 'idea', status: 'draft', lineage: 'no-release' },
        }),
      ],
      total: 1,
    })
    await flushPromises()

    const cell = wrapper.find('.cell-release')
    expect(cell.exists(), '.cell-release cell must be present').toBe(true)
    expect(cell.text()).toBe('—')
  })

  it('TC3: Release column header appears after Priority column', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: [makeArtifact()], total: 1 })
    await flushPromises()

    const headers = wrapper.findAll('th')
    const labels = headers.map(h => h.text().trim())

    const priorityIdx = labels.findIndex(l => l.includes('Priority'))
    const releaseIdx  = labels.findIndex(l => l.includes('Release'))

    expect(priorityIdx, 'Priority column must exist').toBeGreaterThanOrEqual(0)
    expect(releaseIdx,  'Release column must exist').toBeGreaterThanOrEqual(0)
    expect(releaseIdx).toBeGreaterThan(priorityIdx)
  })
})
