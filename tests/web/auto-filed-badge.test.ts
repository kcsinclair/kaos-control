// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 7 — Auto-filed Badge Tests
 *
 * Verifies the auto-filed badge (Bot icon) in ArtifactListView:
 *   - Defect with 'auto-filed' label shows the badge
 *   - Badge carries correct tooltip / aria-label text
 *   - Defect WITHOUT 'auto-filed' label has no badge
 *   - Non-defect artifact with 'auto-filed' label has no badge
 *
 * The badge is rendered inline in ArtifactListView.vue:
 *   v-if="row.type === 'defect' && row.frontmatter?.labels?.includes('auto-filed')"
 *
 * Run with: pnpm --prefix tests/web test auto-filed-badge
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
  listArtifacts: vi.fn().mockResolvedValue({ items: [], total: 0 }),
  listLabels: vi.fn().mockResolvedValue({ labels: [] }),
  listPriorities: vi.fn().mockResolvedValue({ priorities: [] }),
  getArtifact: vi.fn().mockResolvedValue({ artifact: {}, body: '', body_html: '' }),
}))

vi.mock('@/api/ws', () => ({
  getProjectWs: vi.fn(() => ({
    onType: vi.fn(() => () => {}),
  })),
}))

vi.mock('@/api/releases', () => ({
  listReleases: vi.fn().mockResolvedValue([]),
}))

vi.mock('vue-router', async (importActual) => {
  const actual = await importActual<typeof import('vue-router')>()
  return {
    ...actual,
    useRoute: vi.fn(() => ({ params: { project: 'testproject' }, query: {} })),
    useRouter: vi.fn(() => ({ push: vi.fn(), replace: vi.fn() })),
  }
})

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeArtifactRow(overrides: Partial<ArtifactRow> & { labels?: string[] } = {}): ArtifactRow {
  const { labels, ...rest } = overrides
  return {
    path: 'lifecycle/defects/test-defect-7-defect.md',
    slug: 'test-defect',
    lineage: 'test-defect',
    index: 7,
    stage: 'defects',
    type: 'defect',
    status: 'draft',
    title: 'Test Defect',
    frontmatter: {
      title: 'Test Defect',
      type: 'defect',
      status: 'draft',
      lineage: 'test-defect',
      labels: labels ?? [],
    },
    mtime: new Date().toISOString(),
    created: new Date().toISOString(),
    agent_run_count: 0,
    ...rest,
  }
}

function createTestRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:project/artifacts', component: { template: '<div/>' } }],
  })
}

async function mountViewWithItems(items: ArtifactRow[]) {
  const { listArtifacts } = await import('@/api/artifacts')
  vi.mocked(listArtifacts).mockResolvedValue({ items, total: items.length })

  setActivePinia(createPinia())
  const router = createTestRouter()
  await router.push('/testproject/artifacts')

  const wrapper = mount(ArtifactListView, {
    global: {
      plugins: [router],
    },
  })
  await flushPromises()
  return wrapper
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('auto-filed badge visibility', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows the badge for a defect with the auto-filed label', async () => {
    const wrapper = await mountViewWithItems([
      makeArtifactRow({ labels: ['auto-filed'] }),
    ])

    const badge = wrapper.find('.auto-filed-badge')
    expect(badge.exists()).toBe(true)
  })

  it('badge has the correct title tooltip', async () => {
    const wrapper = await mountViewWithItems([
      makeArtifactRow({ labels: ['auto-filed'] }),
    ])

    const badge = wrapper.find('.auto-filed-badge')
    expect(badge.attributes('title')).toBe('Auto-filed by test-runner agent')
  })

  it('badge has the correct aria-label for accessibility', async () => {
    const wrapper = await mountViewWithItems([
      makeArtifactRow({ labels: ['auto-filed'] }),
    ])

    const badge = wrapper.find('.auto-filed-badge')
    expect(badge.attributes('aria-label')).toBe('Auto-filed by test-runner agent')
  })

  it('does not show the badge for a defect without the auto-filed label', async () => {
    const wrapper = await mountViewWithItems([
      makeArtifactRow({ labels: ['backend'] }), // has a label, but not auto-filed
    ])

    expect(wrapper.find('.auto-filed-badge').exists()).toBe(false)
  })

  it('does not show the badge for a defect with no labels', async () => {
    const wrapper = await mountViewWithItems([
      makeArtifactRow({ labels: [] }),
    ])

    expect(wrapper.find('.auto-filed-badge').exists()).toBe(false)
  })

  it('does not show the badge for a non-defect artifact with the auto-filed label', async () => {
    const wrapper = await mountViewWithItems([
      makeArtifactRow({
        type: 'test',
        stage: 'tests',
        path: 'lifecycle/tests/some-2-test.md',
        labels: ['auto-filed'],
      }),
    ])

    expect(wrapper.find('.auto-filed-badge').exists()).toBe(false)
  })

  it('shows badges for all auto-filed defects in the list', async () => {
    const wrapper = await mountViewWithItems([
      makeArtifactRow({
        path: 'lifecycle/defects/a-7-defect.md',
        labels: ['auto-filed'],
      }),
      makeArtifactRow({
        path: 'lifecycle/defects/b-8-defect.md',
        labels: ['auto-filed'],
      }),
      makeArtifactRow({
        path: 'lifecycle/defects/c-9-defect.md',
        labels: [], // no badge
      }),
    ])

    const badges = wrapper.findAll('.auto-filed-badge')
    expect(badges).toHaveLength(2)
  })
})
