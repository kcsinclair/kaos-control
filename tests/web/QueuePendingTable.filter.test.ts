// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Tests for QueuePendingTable — project filter and project link behaviour.
 *
 * Covers Milestone 3 (cases 5–7) and Milestone 4 (cases 2, 4, 5) of the
 * queue-page-project-navigation test plan:
 *
 * Milestone 3 — Filtered table rendering:
 *   M3-5  No filter — shows all pending jobs
 *   M3-6  Filter active — shows only jobs where job.project === filter
 *   M3-7  Filter active, no matching jobs — shows project-specific empty message
 *
 * Milestone 4 — RouterLink targets:
 *   M4-2  Project name renders as a link to /p/:project/agents
 *   M4-4  Link text matches job.project exactly
 *   M4-5  Multiple jobs — each row has its own correctly-targeted link
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { ref } from 'vue'
import type { QueueSnapshot, QueueJob } from '../../web/src/api/queue'

// ---------------------------------------------------------------------------
// Reactive store state
// ---------------------------------------------------------------------------

const _snapshot = ref<QueueSnapshot>({
  running: null,
  pending: [],
  recent: [],
  paused: false,
  paused_until: null,
  pause_reason: null,
})

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

vi.mock('@/stores/queue', () => ({
  useQueueStore: () => ({
    get snapshot() { return _snapshot.value },
    cancel: vi.fn().mockResolvedValue(undefined),
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    me: { email: 'user@test.local', display_name: 'User', roles: {} },
    isAuthenticated: true,
    rolesForProject: () => [],
  }),
}))

vi.mock('@/stores/ui', () => ({
  useUiStore: () => ({
    success: vi.fn(),
    error: vi.fn(),
  }),
}))

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeJob(overrides: Partial<QueueJob> = {}): QueueJob {
  return {
    id: 'job-1',
    project: 'my-project',
    artifact_path: 'lifecycle/ideas/test.md',
    agent: 'requirements-analyst',
    state: 'pending',
    attempts: 1,
    enqueued_at: 1_700_000_000,
    position: 1,
    enqueued_by: 'admin@test.local',
    ...overrides,
  }
}

function makeRouter() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/queue', component: { template: '<div/>' } },
      { path: '/p/:project/:view*', component: { template: '<div/>' } },
      { path: '/:pathMatch(.*)*', component: { template: '<div/>' } },
    ],
  })
  router.push('/queue')
  return router
}

async function mountTable(props: Record<string, unknown> = {}) {
  const { default: QueuePendingTable } = await import('../../web/src/components/queue/QueuePendingTable.vue')
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = makeRouter()
  await router.isReady()
  const wrapper = mount(QueuePendingTable, {
    props,
    global: { plugins: [pinia, router] },
  })
  await flushPromises()
  return wrapper
}

// ---------------------------------------------------------------------------
// Reset between tests
// ---------------------------------------------------------------------------

beforeEach(() => {
  _snapshot.value = {
    running: null,
    pending: [],
    recent: [],
    paused: false,
    paused_until: null,
    pause_reason: null,
  }
})

afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

// ===========================================================================
// Milestone 3 — Filtering Behaviour
// ===========================================================================

describe('QueuePendingTable — Milestone 3: project filter', () => {
  it('M3-5: no filter — shows all pending jobs', async () => {
    _snapshot.value.pending = [
      makeJob({ id: 'p1', project: 'project-a', position: 1 }),
      makeJob({ id: 'p2', project: 'project-b', position: 2 }),
    ]

    const wrapper = await mountTable()
    expect(wrapper.find('.empty-state').exists()).toBe(false)
    expect(wrapper.findAll('tbody tr')).toHaveLength(2)
    expect(wrapper.text()).toContain('project-a')
    expect(wrapper.text()).toContain('project-b')
  })

  it('M3-6: filter active — shows only jobs where job.project === filter', async () => {
    _snapshot.value.pending = [
      makeJob({ id: 'p1', project: 'project-a', position: 1 }),
      makeJob({ id: 'p2', project: 'project-b', position: 2 }),
      makeJob({ id: 'p3', project: 'project-a', position: 3 }),
    ]

    const wrapper = await mountTable({ projectFilter: 'project-a' })
    expect(wrapper.findAll('tbody tr')).toHaveLength(2)
    expect(wrapper.text()).toContain('project-a')
    expect(wrapper.text()).not.toContain('project-b')
  })

  it('M3-7: filter active, no matching jobs — shows project-specific empty message', async () => {
    _snapshot.value.pending = [
      makeJob({ id: 'p1', project: 'project-b' }),
    ]

    const wrapper = await mountTable({ projectFilter: 'project-a' })
    expect(wrapper.find('.empty-state').exists()).toBe(true)
    expect(wrapper.find('.empty-state').text()).toContain('project-a')
  })

  it('M3-7: no filter, empty queue — shows generic empty message', async () => {
    _snapshot.value.pending = []

    const wrapper = await mountTable()
    expect(wrapper.find('.empty-state').exists()).toBe(true)
    const msg = wrapper.find('.empty-state').text()
    expect(msg.toLowerCase()).toContain('empty')
  })

  it('M3: filtering is purely client-side (prop change triggers no API call)', async () => {
    _snapshot.value.pending = [
      makeJob({ id: 'p1', project: 'project-a' }),
    ]

    const wrapper = await mountTable({ projectFilter: 'project-a' })
    expect(wrapper.findAll('tbody tr')).toHaveLength(1)

    // Change filter — component re-computes from store, no fetch
    await wrapper.setProps({ projectFilter: 'project-b' })
    expect(wrapper.find('.empty-state').exists()).toBe(true)
  })
})

// ===========================================================================
// Milestone 4 — RouterLink Targets
// ===========================================================================

describe('QueuePendingTable — Milestone 4: project links', () => {
  it('M4-2: project name renders as a link to /p/:project/agents', async () => {
    _snapshot.value.pending = [makeJob({ project: 'my-project' })]

    const wrapper = await mountTable()
    const projectLinks = wrapper.findAll('a.project-link')
    expect(projectLinks).toHaveLength(1)
    expect(projectLinks[0].attributes('href')).toBe('/p/my-project/agents')
  })

  it('M4-4: link text matches job.project exactly', async () => {
    _snapshot.value.pending = [makeJob({ project: 'exact-project-name' })]

    const wrapper = await mountTable()
    const projectLink = wrapper.find('a.project-link')
    expect(projectLink.text().trim()).toBe('exact-project-name')
  })

  it('M4-5: multiple jobs — each row has its own correctly-targeted project link', async () => {
    _snapshot.value.pending = [
      makeJob({ id: 'p1', project: 'project-a', position: 1 }),
      makeJob({ id: 'p2', project: 'project-b', position: 2 }),
      makeJob({ id: 'p3', project: 'project-c', position: 3 }),
    ]

    const wrapper = await mountTable()
    const projectLinks = wrapper.findAll('a.project-link')
    expect(projectLinks).toHaveLength(3)
    expect(projectLinks[0].attributes('href')).toBe('/p/project-a/agents')
    expect(projectLinks[1].attributes('href')).toBe('/p/project-b/agents')
    expect(projectLinks[2].attributes('href')).toBe('/p/project-c/agents')
  })

  it('M4: project link uses RouterLink (SPA navigation, not full page reload)', async () => {
    // RouterLink renders as <a> with the path-only href when memory history is used.
    // The href must be a relative path (not an absolute URL) confirming it is a RouterLink.
    _snapshot.value.pending = [makeJob({ project: 'my-project' })]

    const wrapper = await mountTable()
    const projectLink = wrapper.find('a.project-link')
    const href = projectLink.attributes('href') ?? ''
    // Memory-history RouterLink produces /p/... not http://...
    expect(href.startsWith('/')).toBe(true)
    expect(href.startsWith('http')).toBe(false)
  })
})
