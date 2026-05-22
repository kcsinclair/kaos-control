// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { shallowMount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import type { ArtifactRow } from '@/types/api'
import { TERMINAL_STATUSES } from '@/types/api'

// Test plan: lifecycle/test-plans/raw-artefact-status-5-test.md §Milestone 3, Scenario 3
//
// Verifies that 'raw' artefacts are visible in the default list view
// (showCompleted=false) because 'raw' is not a terminal status.
// 'done' artefacts are hidden by default; all artefacts are visible when
// showCompleted is toggled on.

// ── Mocks ─────────────────────────────────────────────────────────────────────

vi.mock('@/composables/useWebSocket', () => ({ useWebSocket: vi.fn() }))
vi.mock('@/composables/useTextFilterShortcut', () => ({ useTextFilterShortcut: vi.fn() }))
vi.mock('@/stores/ui', () => ({
  useUiStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))
vi.mock('@/stores/releases', () => ({
  useReleasesStore: () => ({ releases: [], fetch: vi.fn() }),
}))

const mockFetchList = vi.fn()
const mockFetchLabels = vi.fn()
const mockFetchPriorities = vi.fn()

const mockItems: ArtifactRow[] = [
  {
    path: 'lifecycle/ideas/raw-idea.md',
    slug: 'raw-idea',
    lineage: 'raw-idea',
    index: 0,
    stage: 'ideas',
    type: 'idea',
    status: 'raw',
    title: 'Raw Idea',
    frontmatter: { title: 'Raw Idea', type: 'idea', status: 'raw', lineage: 'raw-idea' },
    mtime: '2026-01-01T00:00:00Z',
    created: '2026-01-01T00:00:00Z',
    agent_run_count: 0,
  },
  {
    path: 'lifecycle/ideas/draft-idea.md',
    slug: 'draft-idea',
    lineage: 'draft-idea',
    index: 0,
    stage: 'ideas',
    type: 'idea',
    status: 'draft',
    title: 'Draft Idea',
    frontmatter: { title: 'Draft Idea', type: 'idea', status: 'draft', lineage: 'draft-idea' },
    mtime: '2026-01-01T00:00:00Z',
    created: '2026-01-01T00:00:00Z',
    agent_run_count: 0,
  },
  {
    path: 'lifecycle/ideas/done-idea.md',
    slug: 'done-idea',
    lineage: 'done-idea',
    index: 0,
    stage: 'ideas',
    type: 'idea',
    status: 'done',
    title: 'Done Idea',
    frontmatter: { title: 'Done Idea', type: 'idea', status: 'done', lineage: 'done-idea' },
    mtime: '2026-01-01T00:00:00Z',
    created: '2026-01-01T00:00:00Z',
    agent_run_count: 0,
  },
]

vi.mock('@/stores/artifacts', () => ({
  useArtifactsStore: () => ({
    items: mockItems,
    loading: false,
    filter: {},
    labels: [],
    priorities: [],
    fetchList: mockFetchList,
    fetchLabels: mockFetchLabels,
    fetchPriorities: mockFetchPriorities,
    invalidate: vi.fn(),
  }),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({
    params: { project: 'testproject' },
    query: {},
  }),
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
  }),
}))

// Import component after all mocks are in place.
import ArtifactListView from '../ArtifactListView.vue'

describe('ArtifactListView — raw artefact visibility', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    // fetchList returns resolved promise by default so onMounted doesn't throw.
    mockFetchList.mockResolvedValue(undefined)
    mockFetchLabels.mockResolvedValue(undefined)
    mockFetchPriorities.mockResolvedValue(undefined)
  })

  // TC1: 'raw' is not a terminal status.
  it('TERMINAL_STATUSES does not include "raw"', () => {
    expect((TERMINAL_STATUSES as readonly string[]).includes('raw')).toBe(false)
  })

  // TC2: 'done' IS a terminal status.
  it('TERMINAL_STATUSES includes "done"', () => {
    expect((TERMINAL_STATUSES as readonly string[]).includes('done')).toBe(true)
  })

  // TC3: Default view (showCompleted=false) shows raw and draft but not done.
  it('shows raw and draft rows but hides done rows by default', async () => {
    const wrapper = shallowMount(ArtifactListView, {
      global: { stubs: { RouterLink: true } },
    })

    // Allow onMounted to complete.
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    const html = wrapper.html()

    // The raw and draft artefact titles should be present in the rendered HTML.
    expect(html).toContain('Raw Idea')
    expect(html).toContain('Draft Idea')

    // The done artefact should be absent (filtered out as terminal).
    expect(html).not.toContain('Done Idea')
  })

  // TC4: Toggling showCompleted makes done artefacts visible.
  it('shows done rows after toggling showCompleted on', async () => {
    const wrapper = shallowMount(ArtifactListView, {
      global: { stubs: { RouterLink: true } },
    })

    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    // Toggle the "Show completed" checkbox.
    const toggle = wrapper.find('input[type="checkbox"]')
    expect(toggle.exists()).toBe(true)
    await toggle.setValue(true)
    await wrapper.vm.$nextTick()

    const html = wrapper.html()
    expect(html).toContain('Raw Idea')
    expect(html).toContain('Draft Idea')
    expect(html).toContain('Done Idea')
  })
})
