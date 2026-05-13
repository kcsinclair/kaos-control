// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Unit tests for the AppHeader queue badge.
 *
 * Covers Suite 3.4 scenarios FH1–FH3:
 *   FH1 renders pending count
 *   FH2 paused state shows pause icon
 *   FH3 click navigates to /queue
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { ref, nextTick } from 'vue'
import AppHeader from '../../web/src/components/layout/AppHeader.vue'
import type { QueueSnapshot } from '../../web/src/api/queue'

// ---------------------------------------------------------------------------
// Reactive queue store state
// ---------------------------------------------------------------------------

const _snapshotRef = ref<QueueSnapshot>({
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

vi.mock('@/api/client', () => ({
  api: { get: vi.fn().mockResolvedValue({}) },
  ApiError: class ApiError extends Error {},
}))

vi.mock('@/api/ws', () => ({
  getAppWs: vi.fn(() => ({
    on: vi.fn(() => () => {}),
  })),
  getProjectWs: vi.fn(() => ({
    onType: vi.fn(() => () => {}),
  })),
}))

vi.mock('@/api/queue', () => ({
  listQueue: vi.fn().mockResolvedValue({
    running: null, pending: [], recent: [], paused: false, paused_until: null, pause_reason: null,
  }),
  enqueue: vi.fn(),
  cancelQueue: vi.fn(),
  pauseQueue: vi.fn(),
  resumeQueue: vi.fn(),
}))

vi.mock('@/stores/queue', () => ({
  useQueueStore: () => ({
    get snapshot() { return _snapshotRef.value },
    get isPaused() { return _snapshotRef.value.paused },
    get pendingCount() { return _snapshotRef.value.pending.length },
    get pausedUntilDate() {
      return _snapshotRef.value.paused_until ? new Date(_snapshotRef.value.paused_until) : null
    },
    fetch: vi.fn().mockResolvedValue(undefined),
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    me: { email: 'admin@test.local', display_name: 'Admin', roles: {} },
    isAuthenticated: true,
    logout: vi.fn(),
  }),
}))

vi.mock('@/stores/ui', () => ({
  useUiStore: () => ({
    error: vi.fn(),
  }),
}))

vi.mock('@/stores/theme', () => ({
  useThemeStore: () => ({
    isDark: false,
    toggle: vi.fn(),
  }),
}))

vi.mock('@/stores/agents', () => ({
  useAgentsStore: () => ({
    activeRuns: [],
    agents: [],
  }),
}))

// ---------------------------------------------------------------------------
// Router / mount helper
// ---------------------------------------------------------------------------

function makeRouter(initialPath = '/projects') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/projects', component: { template: '<div/>' } },
      { path: '/queue', component: { template: '<div/>' } },
      { path: '/:pathMatch(.*)*', component: { template: '<div/>' } },
    ],
  })
  router.push(initialPath)
  return router
}

async function mountHeader(initialPath = '/projects') {
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = makeRouter(initialPath)
  await router.isReady()

  const wrapper = mount(AppHeader, {
    global: { plugins: [pinia, router] },
    attachTo: document.body,
  })
  await flushPromises()
  return { wrapper, router }
}

// ---------------------------------------------------------------------------
// Teardown
// ---------------------------------------------------------------------------

beforeEach(() => {
  _snapshotRef.value = {
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

// ---------------------------------------------------------------------------
// FH1 — renders pending count
// ---------------------------------------------------------------------------

describe('FH1: renders pending count', () => {
  it('shows the pending count badge when there are pending jobs', async () => {
    _snapshotRef.value.pending = [
      { id: 'j1', project: 'p', artifact_path: 'a.md', agent: 'a', state: 'pending', attempts: 1, enqueued_at: 0, position: 1, enqueued_by: 'u' },
      { id: 'j2', project: 'p', artifact_path: 'b.md', agent: 'a', state: 'pending', attempts: 1, enqueued_at: 0, position: 2, enqueued_by: 'u' },
      { id: 'j3', project: 'p', artifact_path: 'c.md', agent: 'a', state: 'pending', attempts: 1, enqueued_at: 0, position: 3, enqueued_by: 'u' },
    ]

    const { wrapper } = await mountHeader()
    const badge = wrapper.find('.header-queue-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toContain('3')
  })

  it('badge is visible in idle state (0 pending, not paused) so /queue stays reachable', async () => {
    _snapshotRef.value.pending = []
    _snapshotRef.value.paused = false

    const { wrapper } = await mountHeader()
    const badge = wrapper.find('.header-queue-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.classes()).toContain('header-queue-badge--idle')
    expect(badge.text()).toContain('0')
  })

  it('count updates reactively when a job is added', async () => {
    _snapshotRef.value.pending = []
    const { wrapper } = await mountHeader()
    // Idle state is rendered but tagged with the --idle modifier.
    let badge = wrapper.find('.header-queue-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.classes()).toContain('header-queue-badge--idle')

    _snapshotRef.value.pending = [
      { id: 'j1', project: 'p', artifact_path: 'a.md', agent: 'a', state: 'pending', attempts: 1, enqueued_at: 0, position: 1, enqueued_by: 'u' },
    ]
    await nextTick()
    badge = wrapper.find('.header-queue-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.classes()).not.toContain('header-queue-badge--idle')
    expect(badge.text()).toContain('1')
  })
})

// ---------------------------------------------------------------------------
// FH2 — paused state shows pause icon
// ---------------------------------------------------------------------------

describe('FH2: paused state shows pause icon', () => {
  it('shows pause icon (⏸) when queue is paused', async () => {
    _snapshotRef.value.paused = true
    _snapshotRef.value.paused_until = '2026-06-01T20:00:00+10:00'

    const { wrapper } = await mountHeader()
    const badge = wrapper.find('.header-queue-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.find('.queue-pause-icon').exists()).toBe(true)
    expect(badge.text()).toContain('⏸')
  })

  it('shows count (not pause icon) when queue is not paused but has pending jobs', async () => {
    _snapshotRef.value.paused = false
    _snapshotRef.value.pending = [
      { id: 'j1', project: 'p', artifact_path: 'a.md', agent: 'a', state: 'pending', attempts: 1, enqueued_at: 0, position: 1, enqueued_by: 'u' },
    ]

    const { wrapper } = await mountHeader()
    const badge = wrapper.find('.header-queue-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.find('.queue-count').exists()).toBe(true)
    expect(badge.find('.queue-pause-icon').exists()).toBe(false)
  })

  it('badge has paused CSS class when queue is paused', async () => {
    _snapshotRef.value.paused = true

    const { wrapper } = await mountHeader()
    const badge = wrapper.find('.header-queue-badge')
    expect(badge.classes()).toContain('header-queue-badge--paused')
  })
})

// ---------------------------------------------------------------------------
// FH3 — click navigates to /queue
// ---------------------------------------------------------------------------

describe('FH3: click navigates to /queue', () => {
  it('badge renders as a link to /queue', async () => {
    _snapshotRef.value.pending = [
      { id: 'j1', project: 'p', artifact_path: 'a.md', agent: 'a', state: 'pending', attempts: 1, enqueued_at: 0, position: 1, enqueued_by: 'u' },
    ]

    const { wrapper } = await mountHeader()
    const badge = wrapper.find('.header-queue-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.attributes('href')).toBe('/queue')
  })

  it('clicking the badge navigates to /queue', async () => {
    _snapshotRef.value.pending = [
      { id: 'j1', project: 'p', artifact_path: 'a.md', agent: 'a', state: 'pending', attempts: 1, enqueued_at: 0, position: 1, enqueued_by: 'u' },
    ]

    const { wrapper, router } = await mountHeader('/projects')
    const badge = wrapper.find('.header-queue-badge')
    expect(badge.exists()).toBe(true)

    await badge.trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.fullPath).toBe('/queue')
  })

  it('badge link to /queue is also present when queue is paused (even with 0 pending)', async () => {
    _snapshotRef.value.paused = true
    _snapshotRef.value.pending = []

    const { wrapper } = await mountHeader()
    const badge = wrapper.find('.header-queue-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.attributes('href')).toBe('/queue')
  })
})
