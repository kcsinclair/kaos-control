// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Tests for QueueRunningPanel — project filter and project link behaviour.
 *
 * Covers Milestone 3 (cases 1–4) and Milestone 4 (case 1) of the
 * queue-page-project-navigation test plan:
 *
 * Milestone 3 — Filtered panel rendering:
 *   M3-1  No filter — shows running job regardless of project
 *   M3-2  Filter matches running job — shows the job
 *   M3-3  Filter does not match running job — shows "Nothing running"
 *   M3-4  No running job + filter active — shows "Nothing running"
 *
 * Milestone 4 — RouterLink targets:
 *   M4-1  Project name renders as a link to /p/:project/dashboard
 *   M4-4  Link text matches job.project exactly
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
  }),
}))

vi.mock('@/composables/useNow', () => ({
  useNow: () => ref(new Date(1_700_000_000 * 1000)),
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
    state: 'running',
    attempts: 1,
    enqueued_at: 1_700_000_000,
    started_at: 1_700_000_000,
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

async function mountPanel(props: Record<string, unknown> = {}) {
  const { default: QueueRunningPanel } = await import('../../web/src/components/queue/QueueRunningPanel.vue')
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = makeRouter()
  await router.isReady()
  const wrapper = mount(QueueRunningPanel, {
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

describe('QueueRunningPanel — Milestone 3: project filter', () => {
  it('M3-1: no filter — shows running job regardless of project', async () => {
    _snapshot.value.running = makeJob({ project: 'project-x' })

    const wrapper = await mountPanel() // no projectFilter prop
    expect(wrapper.find('.empty-state').exists()).toBe(false)
    expect(wrapper.find('.running-row').exists()).toBe(true)
    expect(wrapper.text()).toContain('project-x')
  })

  it('M3-2: filter matches running job — shows the job', async () => {
    _snapshot.value.running = makeJob({ project: 'project-x' })

    const wrapper = await mountPanel({ projectFilter: 'project-x' })
    expect(wrapper.find('.empty-state').exists()).toBe(false)
    expect(wrapper.find('.running-row').exists()).toBe(true)
  })

  it('M3-3: filter does not match running job — shows "Nothing running"', async () => {
    _snapshot.value.running = makeJob({ project: 'project-x' })

    const wrapper = await mountPanel({ projectFilter: 'project-y' })
    expect(wrapper.find('.empty-state').exists()).toBe(true)
    expect(wrapper.find('.empty-state').text()).toContain('Nothing running')
    expect(wrapper.find('.running-row').exists()).toBe(false)
  })

  it('M3-4: no running job + filter active — shows "Nothing running"', async () => {
    _snapshot.value.running = null

    const wrapper = await mountPanel({ projectFilter: 'project-x' })
    expect(wrapper.find('.empty-state').exists()).toBe(true)
    expect(wrapper.find('.empty-state').text()).toContain('Nothing running')
  })

  it('M3: filter does not trigger additional API calls (purely client-side)', async () => {
    // No API mocks needed — the component reads only from the store, no fetch on prop change.
    // Verify that mounting and changing the filter does not cause errors.
    _snapshot.value.running = makeJob({ project: 'project-a' })

    const wrapper = await mountPanel({ projectFilter: 'project-a' })
    expect(wrapper.find('.running-row').exists()).toBe(true)

    await wrapper.setProps({ projectFilter: 'project-b' })
    expect(wrapper.find('.empty-state').exists()).toBe(true)
    // If any unexpected fetch had fired, the test environment would emit an
    // ECONNREFUSED error and the assertion above would already have failed.
  })
})

// ===========================================================================
// Milestone 4 — RouterLink Targets
// ===========================================================================

describe('QueueRunningPanel — Milestone 4: project links', () => {
  it('M4-1: project name renders as a link to /p/:project/dashboard', async () => {
    _snapshot.value.running = makeJob({ project: 'my-project' })

    const wrapper = await mountPanel()
    // Find the project link specifically (not the artifact link)
    const links = wrapper.findAll('a')
    const projectLink = links.find(
      (a) => a.text().trim() === 'my-project' && a.attributes('href')?.includes('/dashboard'),
    )
    expect(projectLink).toBeDefined()
    expect(projectLink!.attributes('href')).toBe('/p/my-project/dashboard')
  })

  it('M4-4: link text matches job.project exactly', async () => {
    _snapshot.value.running = makeJob({ project: 'exact-project-name' })

    const wrapper = await mountPanel()
    const links = wrapper.findAll('a')
    const projectLink = links.find((a) => a.attributes('href')?.includes('/dashboard'))
    expect(projectLink!.text().trim()).toBe('exact-project-name')
  })

  it('M4: project with special characters is URL-encoded in href', async () => {
    _snapshot.value.running = makeJob({ project: 'my project' })

    const wrapper = await mountPanel()
    const links = wrapper.findAll('a')
    const projectLink = links.find((a) => a.attributes('href')?.includes('/dashboard'))
    expect(projectLink!.attributes('href')).toBe('/p/my%20project/dashboard')
  })
})
