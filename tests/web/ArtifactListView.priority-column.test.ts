/**
 * Milestone 2 — Priority column display tests
 *
 * Verifies that the Priority column renders priority pills with correct text
 * and CSS classes, shows a dash for missing priority, and sits between the
 * Status and Release columns.
 *
 * Component reference: web/src/views/project/ArtifactListView.vue
 *  Priority cell: <td class="cell-priority">
 *    <span class="priority-pill priority-{value}">…</span>   (when set)
 *    <span class="muted">—</span>                            (when absent)
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

describe('ArtifactListView — Priority column display', () => {
  it('TC1: renders a priority pill with text and class for priority: high', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({
      items: [
        makeArtifact({
          path: 'lifecycle/ideas/high.md',
          lineage: 'high',
          frontmatter: { title: 'High', type: 'idea', status: 'draft', lineage: 'high', priority: 'high' },
        }),
      ],
      total: 1,
    })
    await flushPromises()

    const pill = wrapper.find('.priority-pill')
    expect(pill.exists(), 'priority pill element must be present').toBe(true)
    expect(pill.text()).toBe('high')
    expect(pill.classes()).toContain('priority-high')
    // The fallback dash must not be present
    expect(wrapper.find('.cell-priority .muted').exists()).toBe(false)
  })

  it('TC2: renders a dash when the artifact has no priority', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({
      items: [
        makeArtifact({
          path: 'lifecycle/ideas/noprio.md',
          lineage: 'noprio',
          frontmatter: { title: 'No Prio', type: 'idea', status: 'draft', lineage: 'noprio' },
        }),
      ],
      total: 1,
    })
    await flushPromises()

    const cell = wrapper.find('.cell-priority')
    expect(cell.exists(), '.cell-priority cell must be present').toBe(true)
    const dash = cell.find('.muted')
    expect(dash.exists(), 'dash element must be present for no-priority artifact').toBe(true)
    expect(dash.text()).toBe('—')
    expect(cell.find('.priority-pill').exists(), 'no pill when priority absent').toBe(false)
  })

  it('TC3: renders correct colour class for each priority value', async () => {
    const allPriorities = ['critical', 'high', 'normal', 'low'] as const

    for (const priority of allPriorities) {
      setActivePinia(createPinia())
      const wrapper = mountView()
      await flushPromises()

      const store = useArtifactsStore()
      store.$patch({
        items: [
          makeArtifact({
            path: `lifecycle/ideas/${priority}.md`,
            lineage: priority,
            frontmatter: { title: priority, type: 'idea', status: 'draft', lineage: priority, priority },
          }),
        ],
        total: 1,
      })
      await flushPromises()

      const pill = wrapper.find('.priority-pill')
      expect(pill.exists(), `priority "${priority}": pill must render`).toBe(true)
      expect(pill.classes(), `priority "${priority}": must have class priority-${priority}`).toContain(`priority-${priority}`)
      expect(pill.text(), `priority "${priority}": pill text must match`).toBe(priority)
    }
  })

  it('TC4: Priority column header appears after Status and before Release', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useArtifactsStore()
    store.$patch({ items: [makeArtifact()], total: 1 })
    await flushPromises()

    const headers = wrapper.findAll('th')
    const labels = headers.map(h => h.text().trim())

    const statusIdx   = labels.findIndex(l => l.includes('Status'))
    const priorityIdx = labels.findIndex(l => l.includes('Priority'))
    const releaseIdx  = labels.findIndex(l => l.includes('Release'))

    expect(statusIdx,   'Status column must exist').toBeGreaterThanOrEqual(0)
    expect(priorityIdx, 'Priority column must exist').toBeGreaterThanOrEqual(0)
    expect(releaseIdx,  'Release column must exist').toBeGreaterThanOrEqual(0)

    expect(priorityIdx).toBeGreaterThan(statusIdx)
    expect(releaseIdx).toBeGreaterThan(priorityIdx)
  })
})
