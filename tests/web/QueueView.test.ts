// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Unit tests for the QueueView page and its queue-specific sub-components.
 *
 * Covers Suite 3.2 scenarios FV1–FV5:
 *   FV1 renders running + pending + recent sections
 *   FV2 empty running shows empty state
 *   FV3 pause banner only when paused
 *   FV4 Resume now visible only for product-owner / devops
 *   FV5 Remove on a pending row calls queueStore.cancel
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { ref, nextTick } from 'vue'
import type { QueueSnapshot, QueueJob } from '../../web/src/api/queue'

// ---------------------------------------------------------------------------
// Reactive store state
// ---------------------------------------------------------------------------

const _snapshotRef = ref<QueueSnapshot>({
  running: null,
  pending: [],
  recent: [],
  paused: false,
  paused_until: null,
  pause_reason: null,
})
const _loading = ref(false)
const _error = ref<string | null>(null)
const _cancelMock = vi.fn().mockResolvedValue(undefined)
const _resumeMock = vi.fn().mockResolvedValue(undefined)
const _fetchMock = vi.fn().mockResolvedValue(undefined)

let _authRoles: string[] = ['product-owner']

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

vi.mock('@/api/ws', () => ({
  getAppWs: vi.fn(() => ({
    on: vi.fn(() => () => {}),
  })),
}))

vi.mock('@/api/queue', () => ({
  listQueue: vi.fn().mockResolvedValue({
    running: null, pending: [], recent: [], paused: false, paused_until: null, pause_reason: null,
  }),
  enqueue: vi.fn(),
  cancelQueue: vi.fn().mockResolvedValue(undefined),
  pauseQueue: vi.fn(),
  resumeQueue: vi.fn().mockResolvedValue(undefined),
}))

vi.mock('@/stores/queue', () => ({
  useQueueStore: () => ({
    get snapshot() { return _snapshotRef.value },
    get loading() { return _loading.value },
    get error() { return _error.value },
    get isPaused() { return _snapshotRef.value.paused },
    get pausedUntilDate() {
      return _snapshotRef.value.paused_until ? new Date(_snapshotRef.value.paused_until) : null
    },
    get pendingCount() { return _snapshotRef.value.pending.length },
    fetch: _fetchMock,
    cancel: _cancelMock,
    resume: _resumeMock,
    enqueue: vi.fn(),
    pause: vi.fn(),
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    get me() {
      return {
        email: 'user@test.local',
        display_name: 'User',
        roles: { testproject: _authRoles },
      }
    },
    isAuthenticated: true,
    rolesForProject: (_p: string) => _authRoles,
    logout: vi.fn(),
  }),
}))

vi.mock('@/stores/ui', () => ({
  useUiStore: () => ({
    success: vi.fn(),
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

vi.mock('@/composables/useNow', () => ({
  useNow: () => ref(new Date()),
}))

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeJob(overrides: Partial<QueueJob> = {}): QueueJob {
  return {
    id: 'job-1',
    project: 'testproject',
    artifact_path: 'lifecycle/ideas/test.md',
    agent: 'requirements-analyst',
    state: 'pending',
    attempts: 1,
    enqueued_at: 1700000000,
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
      { path: '/:pathMatch(.*)*', component: { template: '<div/>' } },
    ],
  })
  router.push('/queue')
  return router
}

async function mountQueueView() {
  const { default: QueueView } = await import('../../web/src/views/QueueView.vue')
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = makeRouter()
  await router.isReady()

  const wrapper = mount(QueueView, {
    global: { plugins: [pinia, router] },
  })
  await flushPromises()
  return wrapper
}

async function mountPauseBanner() {
  const { default: QueuePauseBanner } = await import('../../web/src/components/queue/QueuePauseBanner.vue')
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = makeRouter()
  await router.isReady()

  const wrapper = mount(QueuePauseBanner, {
    global: { plugins: [pinia, router] },
  })
  await flushPromises()
  return wrapper
}

async function mountPendingTable() {
  const { default: QueuePendingTable } = await import('../../web/src/components/queue/QueuePendingTable.vue')
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = makeRouter()
  await router.isReady()

  const wrapper = mount(QueuePendingTable, {
    global: { plugins: [pinia, router] },
  })
  await flushPromises()
  return wrapper
}

async function mountRunningPanel() {
  const { default: QueueRunningPanel } = await import('../../web/src/components/queue/QueueRunningPanel.vue')
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = makeRouter()
  await router.isReady()

  const wrapper = mount(QueueRunningPanel, {
    global: { plugins: [pinia, router] },
  })
  await flushPromises()
  return wrapper
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
  _loading.value = false
  _error.value = null
  _authRoles = ['product-owner']
  _cancelMock.mockClear()
  _resumeMock.mockClear()
  _fetchMock.mockClear()
})

afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

// ---------------------------------------------------------------------------
// FV1 — renders running + pending + recent sections
// ---------------------------------------------------------------------------

describe('FV1: renders running + pending + recent sections', () => {
  it('renders all three section headings when fully populated', async () => {
    _snapshotRef.value = {
      running: makeJob({ id: 'run-1', state: 'running' }),
      pending: [makeJob({ id: 'pend-1', position: 1 })],
      recent: [makeJob({ id: 'rec-1', state: 'completed' })],
      paused: false,
      paused_until: null,
      pause_reason: null,
    }

    const wrapper = await mountQueueView()
    const text = wrapper.text()
    expect(text).toContain('Running')
    expect(text).toContain('Pending')
    expect(text).toContain('Recently finished')
  })

  it('running panel shows the agent name when a job is running', async () => {
    _snapshotRef.value.running = makeJob({ id: 'run-1', agent: 'requirements-analyst', state: 'running' })

    const wrapper = await mountRunningPanel()
    expect(wrapper.text()).toContain('requirements-analyst')
  })

  it('pending table shows job rows', async () => {
    _snapshotRef.value.pending = [
      makeJob({ id: 'p1', position: 1, artifact_path: 'lifecycle/ideas/a.md' }),
      makeJob({ id: 'p2', position: 2, artifact_path: 'lifecycle/ideas/b.md' }),
    ]

    const wrapper = await mountPendingTable()
    expect(wrapper.text()).toContain('lifecycle/ideas/a.md')
    expect(wrapper.text()).toContain('lifecycle/ideas/b.md')
  })
})

// ---------------------------------------------------------------------------
// FV2 — empty running shows empty state
// ---------------------------------------------------------------------------

describe('FV2: empty running shows empty state', () => {
  it('shows "Nothing running" when running is null', async () => {
    _snapshotRef.value.running = null

    const wrapper = await mountRunningPanel()
    expect(wrapper.find('.empty-state').exists()).toBe(true)
    expect(wrapper.find('.empty-state').text()).toContain('Nothing running')
  })

  it('hides empty state when a job is running', async () => {
    _snapshotRef.value.running = makeJob({ id: 'r1', state: 'running' })

    const wrapper = await mountRunningPanel()
    expect(wrapper.find('.empty-state').exists()).toBe(false)
    expect(wrapper.find('.running-row').exists()).toBe(true)
  })
})

// ---------------------------------------------------------------------------
// FV3 — pause banner only when paused
// ---------------------------------------------------------------------------

describe('FV3: pause banner only when paused', () => {
  it('pause banner is absent when paused=false', async () => {
    _snapshotRef.value.paused = false

    const wrapper = await mountQueueView()
    expect(wrapper.find('.pause-banner').exists()).toBe(false)
  })

  it('pause banner is visible when paused=true', async () => {
    _snapshotRef.value.paused = true
    _snapshotRef.value.paused_until = '2026-06-01T20:00:00+10:00'
    _snapshotRef.value.pause_reason = 'rate_limit'

    const wrapper = await mountQueueView()
    await nextTick()
    expect(wrapper.find('.pause-banner').exists()).toBe(true)
  })

  it('pause banner contains reset time when paused_until is set', async () => {
    _snapshotRef.value.paused = true
    _snapshotRef.value.paused_until = '2026-06-01T20:00:00+10:00'

    const wrapper = await mountPauseBanner()
    expect(wrapper.find('.pause-banner').exists()).toBe(true)
    // The banner renders "resumes <time>"
    expect(wrapper.text()).toMatch(/resumes/i)
  })
})

// ---------------------------------------------------------------------------
// FV4 — Resume now visible only for product-owner / devops
// ---------------------------------------------------------------------------

describe('FV4: Resume now visible only for product-owner / devops', () => {
  it('Resume now button is visible for product-owner role', async () => {
    _authRoles = ['product-owner']
    _snapshotRef.value.paused = true

    const wrapper = await mountPauseBanner()
    expect(wrapper.find('.btn-resume').exists()).toBe(true)
    expect(wrapper.find('.btn-resume').text()).toContain('Resume now')
  })

  it('Resume now button is visible for devops role', async () => {
    _authRoles = ['devops']
    _snapshotRef.value.paused = true

    const wrapper = await mountPauseBanner()
    expect(wrapper.find('.btn-resume').exists()).toBe(true)
  })

  it('Resume now button is hidden for qa role', async () => {
    _authRoles = ['qa']
    _snapshotRef.value.paused = true

    const wrapper = await mountPauseBanner()
    expect(wrapper.find('.btn-resume').exists()).toBe(false)
  })

  it('clicking Resume now calls queueStore.resume', async () => {
    _authRoles = ['product-owner']
    _snapshotRef.value.paused = true

    const wrapper = await mountPauseBanner()
    const btn = wrapper.find('.btn-resume')
    expect(btn.exists()).toBe(true)

    await btn.trigger('click')
    await flushPromises()

    expect(_resumeMock).toHaveBeenCalledOnce()
  })
})

// ---------------------------------------------------------------------------
// FV5 — Remove on a pending row calls queueStore.cancel
// ---------------------------------------------------------------------------

describe('FV5: Remove on a pending row calls queueStore.cancel', () => {
  it('calls cancel(id) when Remove button is clicked', async () => {
    _authRoles = ['product-owner']
    _snapshotRef.value.pending = [
      makeJob({ id: 'remove-me', enqueued_by: 'admin@test.local' }),
    ]

    const wrapper = await mountPendingTable()
    const btn = wrapper.find('.btn-remove')
    expect(btn.exists()).toBe(true)

    await btn.trigger('click')
    await flushPromises()

    expect(_cancelMock).toHaveBeenCalledOnce()
    expect(_cancelMock).toHaveBeenCalledWith('remove-me')
  })
})
