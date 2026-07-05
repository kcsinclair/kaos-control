// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 6 (idea-archiving-5-test.md) — rel_path row display
 *
 * Verifies the artifact list's path chip shows rel_path (root-relative,
 * shorter) for nested artifacts and the bare filename for flat ones, and
 * that sorting/pagination are unaffected by the presence of rel_path.
 *
 * Component reference: web/src/views/project/ArtifactListView.vue
 *  Path cell: <td class="cell-path">…<span class="artifact-path">{{ row.rel_path || row.path }}</span>
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
    rel_path:  'test.md',
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
  } as ArtifactRow
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

describe('ArtifactListView — rel_path row display', () => {
  it('shows the bare filename for a flat artifact', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({
      items: [makeArtifact({ path: 'lifecycle/ideas/flat.md', rel_path: 'flat.md', lineage: 'flat' })],
      total: 1,
    })
    await flushPromises()

    const chips = wrapper.findAll('.artifact-path').map(n => n.text())
    expect(chips).toContain('flat.md')
    expect(chips).not.toContain('lifecycle/ideas/flat.md')
  })

  it('shows the rel_path (not the full repo path) for a nested artifact', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({
      items: [
        makeArtifact({
          path:     'lifecycle/ideas/archive/nested.md',
          rel_path: 'archive/nested.md',
          lineage:  'nested',
        }),
      ],
      total: 1,
    })
    await flushPromises()

    const chips = wrapper.findAll('.artifact-path').map(n => n.text())
    expect(chips).toContain('archive/nested.md')
    expect(chips).not.toContain('lifecycle/ideas/archive/nested.md')
  })

  it('shows a deeply-nested rel_path unabridged', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({
      items: [
        makeArtifact({
          path:     'lifecycle/ideas/2026/q3/release-x.md',
          rel_path: '2026/q3/release-x.md',
          lineage:  'release-x',
        }),
      ],
      total: 1,
    })
    await flushPromises()

    const chips = wrapper.findAll('.artifact-path').map(n => n.text())
    expect(chips).toContain('2026/q3/release-x.md')
  })

  it('falls back to the full path when rel_path is absent (stale index row)', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({
      items: [
        makeArtifact({
          path:     'lifecycle/ideas/stale.md',
          rel_path: '',
          lineage:  'stale',
        }),
      ],
      total: 1,
    })
    await flushPromises()

    const chips = wrapper.findAll('.artifact-path').map(n => n.text())
    expect(chips).toContain('lifecycle/ideas/stale.md')
  })

  it('sorting and pagination are unaffected by mixed flat and nested rel_path rows', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({
      items: [
        makeArtifact({ path: 'lifecycle/ideas/b-flat.md', rel_path: 'b-flat.md', lineage: 'b-flat', title: 'B Flat' }),
        makeArtifact({ path: 'lifecycle/ideas/a/nested.md', rel_path: 'a/nested.md', lineage: 'a-nested', title: 'A Nested' }),
      ],
      total: 2,
    })
    await flushPromises()

    const rows = wrapper.findAll('tbody tr')
    expect(rows.length).toBe(2)
    const chips = wrapper.findAll('.artifact-path').map(n => n.text())
    expect(chips).toEqual(['b-flat.md', 'a/nested.md'])
  })
})
