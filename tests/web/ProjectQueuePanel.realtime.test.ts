// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Real-time update tests for the ProjectQueuePanel component.
 *
 * Covers Milestone 3 requirements from the test plan:
 * - Simulating queue.added for current project adds job to panel
 * - Simulating queue.started moves job to running section when belongs to project
 * - Simulating queue.finished/cancelled removes job from waiting sections
 * - Events for different projects don't appear in the panel
 * - queue.paused/resumed toggles panel's paused affordance
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

// ProjectQueuePanel's onMounted calls projectStore.fetchProjects(); mock the project
// store so it resolves instead of leaking a real request to test.local (an
// unhandled rejection, which Vitest 4 treats as fatal).
vi.mock('@/stores/project', () => ({
  useProjectStore: () => ({
    fetchProjects: vi.fn().mockResolvedValue(undefined),
    projects: [],
    current: null,
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
    agent_name: 'requirements-analyst',
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
      { path: '/', component: { template: '<div/>' } },
      { path: '/queue', component: { template: '<div/>' } },
      { path: '/:pathMatch(.*)*', component: { template: '<div/>' } },
    ],
  })
  router.push('/')
  return router
}

async function mountProjectQueuePanel(project: string = 'testproject') {
  const { default: ProjectQueuePanel } = await import('../../web/src/components/agent/ProjectQueuePanel.vue')
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = makeRouter()
  await router.isReady()

  const wrapper = mount(ProjectQueuePanel, {
    props: { project },
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
// Test Cases
// ---------------------------------------------------------------------------

describe('ProjectQueuePanel real-time updates', () => {
  // Test FR-5: Real-time updates - queue.added for current project
  it('adds job to pending list when queue.added event for current project', async () => {
    const wrapper = await mountProjectQueuePanel()

    // Simulate a queue.added event for the current project
    const wsMock = vi.mocked(await import('@/api/ws')).getAppWs()
    const onCall = wsMock.on.mock.calls[0][0]

    // Call the handler with an event for the current project
    onCall({
      type: 'queue.added',
      payload: {
        id: 'new-job-1',
        project: 'testproject',
        artifact_path: 'lifecycle/ideas/new.md',
        agent_name: 'requirements-analyst',
        state: 'pending',
        position: 5,
        enqueued_at: 1700000005,
      }
    })

    await flushPromises()

    // Should now show the new job
    expect(wrapper.text()).toContain('new-job-1')
    expect(wrapper.text()).toContain('lifecycle/ideas/new.md')
  })

  // Test FR-5: Real-time updates - queue.started for current project
  it('moves job to running section when queue.started event for current project', async () => {
    _snapshotRef.value.pending = [
      makeJob({ id: 'p1', project: 'testproject', position: 1 }),
    ]

    const wrapper = await mountProjectQueuePanel()

    // Simulate a queue.started event for the current project
    const wsMock = vi.mocked(await import('@/api/ws')).getAppWs()
    const onCall = wsMock.on.mock.calls[0][0]

    onCall({
      type: 'queue.started',
      payload: {
        id: 'p1',
        started_at: 1700000005,
      }
    })

    await flushPromises()

    // Should now show the job in running section
    expect(wrapper.text()).toContain('p1')
    expect(wrapper.find('.running-row').exists()).toBe(true)
    expect(wrapper.find('.pending-section').exists()).toBe(false)
  })

  it('does not move job to running when queue.started event for different project', async () => {
    _snapshotRef.value.pending = [
      makeJob({ id: 'p1', project: 'otherproject', position: 1 }),
    ]

    const wrapper = await mountProjectQueuePanel()

    // Simulate a queue.started event for a different project
    const wsMock = vi.mocked(await import('@/api/ws')).getAppWs()
    const onCall = wsMock.on.mock.calls[0][0]

    onCall({
      type: 'queue.started',
      payload: {
        id: 'p1',
        started_at: 1700000005,
      }
    })

    await flushPromises()

    // Should not change - job should still be pending
    expect(wrapper.find('.pending-row').exists()).toBe(true)
    expect(wrapper.find('.running-row').exists()).toBe(false)
  })

  // Test FR-5: Real-time updates - queue.finished/cancelled removes job from waiting sections
  it('removes job from pending when queue.finished event for current project', async () => {
    _snapshotRef.value.pending = [
      makeJob({ id: 'p1', project: 'testproject' }),
    ]

    const wrapper = await mountProjectQueuePanel()

    // Simulate a queue.finished event for the current project
    const wsMock = vi.mocked(await import('@/api/ws')).getAppWs()
    const onCall = wsMock.on.mock.calls[0][0]

    onCall({
      type: 'queue.finished',
      payload: {
        id: 'p1',
        terminal_state: 'completed',
      }
    })

    await flushPromises()

    // Should no longer show the job in pending
    expect(wrapper.text()).not.toContain('p1')
  })

  it('removes job from pending when queue.cancelled event for current project', async () => {
    _snapshotRef.value.pending = [
      makeJob({ id: 'p1', project: 'testproject' }),
    ]

    const wrapper = await mountProjectQueuePanel()

    // Simulate a queue.cancelled event for the current project
    const wsMock = vi.mocked(await import('@/api/ws')).getAppWs()
    const onCall = wsMock.on.mock.calls[0][0]

    onCall({
      type: 'queue.cancelled',
      payload: {
        id: 'p1',
      }
    })

    await flushPromises()

    // Should no longer show the job in pending
    expect(wrapper.text()).not.toContain('p1')
  })

  // Test FR-5: Events for different projects don't appear in panel
  it('does not add job to panel when queue.added event for different project', async () => {
    const wrapper = await mountProjectQueuePanel()

    // Simulate a queue.added event for a different project
    const wsMock = vi.mocked(await import('@/api/ws')).getAppWs()
    const onCall = wsMock.on.mock.calls[0][0]

    onCall({
      type: 'queue.added',
      payload: {
        id: 'new-job-1',
        project: 'otherproject', // different project
        artifact_path: 'lifecycle/ideas/new.md',
        agent_name: 'requirements-analyst',
        state: 'pending',
        position: 5,
        enqueued_at: 1700000005,
      }
    })

    await flushPromises()

    // Should not show the job from other project
    expect(wrapper.text()).not.toContain('new-job-1')
  })

  // Test FR-5: queue.paused/resumed toggles panel's paused affordance
  it('shows paused state when queue.paused event is received', async () => {
    const wrapper = await mountProjectQueuePanel()

    // Simulate a queue.paused event
    const wsMock = vi.mocked(await import('@/api/ws')).getAppWs()
    const onCall = wsMock.on.mock.calls[0][0]

    onCall({
      type: 'queue.paused',
      payload: {
        paused_until: '2026-06-01T20:00:00+10:00',
        pause_reason: 'rate_limit',
      }
    })

    await flushPromises()

    // Should show the paused note
    expect(wrapper.find('.paused-note').exists()).toBe(true)
  })

  it('hides paused state when queue.resumed event is received', async () => {
    _snapshotRef.value.paused = true

    const wrapper = await mountProjectQueuePanel()

    // Simulate a queue.resumed event
    const wsMock = vi.mocked(await import('@/api/ws')).getAppWs()
    const onCall = wsMock.on.mock.calls[0][0]

    onCall({
      type: 'queue.resumed',
      payload: {}
    })

    await flushPromises()

    // Should no longer show the paused note
    expect(wrapper.find('.paused-note').exists()).toBe(false)
  })
})