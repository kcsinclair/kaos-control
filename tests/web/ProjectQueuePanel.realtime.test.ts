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
 *
 * ProjectQueuePanel itself never calls getAppWs() or queueStore.fetch() — in the
 * running app, an ancestor view (e.g. AgentsRunsView.vue) calls fetch() on mount,
 * which subscribes the *store* to the app WS. So this suite exercises the real
 * `@/stores/queue` store (only the network boundary — `@/api/ws` and
 * `@/api/queue` — is mocked) and drives it the same way: mount the panel, then
 * call queueStore.fetch() to trigger the WS subscription, then invoke the
 * captured handler to simulate a server event.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { ref } from 'vue'
import { useQueueStore } from '../../web/src/stores/queue'
import type { QueueSnapshot, QueueJob } from '../../web/src/api/queue'

// ---------------------------------------------------------------------------
// Hoisted mocks
// ---------------------------------------------------------------------------

// getAppWs() must return the *same* mock object (and `.on` function) across
// calls so that the queue store's registered handler and the handler this
// test inspects via `_wsOnMock.mock.calls` are the same reference — see
// project-queue-view-7-defect.md.
const { _wsOnMock, _listQueueMock } = vi.hoisted(() => ({
  _wsOnMock: vi.fn(() => () => {}),
  _listQueueMock: vi.fn(),
}))

let _authRoles: string[] = ['product-owner']

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

vi.mock('@/api/ws', () => ({
  getAppWs: vi.fn(() => ({
    on: _wsOnMock,
  })),
}))

vi.mock('@/api/queue', () => ({
  listQueue: _listQueueMock,
  enqueue: vi.fn(),
  cancelQueue: vi.fn().mockResolvedValue(undefined),
  pauseQueue: vi.fn(),
  resumeQueue: vi.fn().mockResolvedValue(undefined),
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

// ProjectQueuePanel's ancestor view calls projectStore.fetchProjects(); mock the
// project store so it resolves instead of leaking a real request to test.local
// (an unhandled rejection, which Vitest 4 treats as fatal).
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

function emptySnapshot(): QueueSnapshot {
  return {
    running: null,
    pending: [],
    recent: [],
    paused: false,
    paused_until: null,
    pause_reason: null,
  }
}

function makeJob(overrides: Partial<QueueJob> = {}): QueueJob {
  return {
    id: 'job-1',
    project: 'testproject',
    artifact_path: 'lifecycle/ideas/test.md',
    agent_name: 'requirements-analyst',
    state: 'pending',
    attempts: 1,
    enqueued_at: '2023-11-14T22:13:20Z',
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

// Mounts the panel and then drives the real queue store's fetch()/subscribe
// flow the same way an ancestor view does, so `getAppWs().on` is actually
// registered and the returned handler can be exercised.
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

  await useQueueStore().fetch()
  await flushPromises()
  return wrapper
}

// ---------------------------------------------------------------------------
// Teardown
// ---------------------------------------------------------------------------

beforeEach(() => {
  _authRoles = ['product-owner']
  _listQueueMock.mockReset()
  _listQueueMock.mockResolvedValue(emptySnapshot())
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
    const onCall = _wsOnMock.mock.calls[0][0]

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
        enqueued_at: '2023-11-14T22:13:25Z',
      }
    })

    await flushPromises()

    // Should now show the new job. ProjectQueuePanel never renders a job's
    // `id` in the DOM, so assert on the rendered artifact_path/agent_name
    // instead (see project-queue-view-7-defect.md for background on why this
    // suite previously couldn't reach real DOM assertions at all).
    expect(wrapper.findAll('.pending-row').length).toBe(1)
    expect(wrapper.text()).toContain('lifecycle/ideas/new.md')
    expect(wrapper.text()).toContain('requirements-analyst')
  })

  // Test FR-5: Real-time updates - queue.started for current project
  it('moves job to running section when queue.started event for current project', async () => {
    _listQueueMock.mockResolvedValue({
      ...emptySnapshot(),
      pending: [makeJob({ id: 'p1', project: 'testproject', position: 1 })],
    })

    const wrapper = await mountProjectQueuePanel()

    // Simulate a queue.started event for the current project
    const onCall = _wsOnMock.mock.calls[0][0]

    onCall({
      type: 'queue.started',
      payload: {
        id: 'p1',
        started_at: 1700000005,
      }
    })

    await flushPromises()

    // Should now show the job in running section. ProjectQueuePanel never
    // renders a job's `id` in the DOM, so assert on the running-row content
    // instead (see project-queue-view-7-defect.md for background).
    expect(wrapper.find('.running-row').exists()).toBe(true)
    expect(wrapper.find('.running-row').text()).toContain('lifecycle/ideas/test.md')
    expect(wrapper.find('.pending-section').exists()).toBe(false)
  })

  it('does not move job to running when queue.started event for a different project\'s job', async () => {
    // The store's queue.started handler matches purely on job id — it has no
    // notion of "project" itself. Project scoping happens in the panel's
    // computed `running`/`pending`, which filter by props.project. So the
    // meaningful scenario is: this project's job (p1) stays untouched while a
    // *different* job (belonging to another project) is the one that starts.
    _listQueueMock.mockResolvedValue({
      ...emptySnapshot(),
      pending: [
        makeJob({ id: 'p1', project: 'testproject', position: 1, artifact_path: 'lifecycle/ideas/mine.md' }),
        makeJob({ id: 'p2', project: 'otherproject', position: 2, artifact_path: 'lifecycle/ideas/other.md' }),
      ],
    })

    const wrapper = await mountProjectQueuePanel()

    // Simulate a queue.started event for the other project's job
    const onCall = _wsOnMock.mock.calls[0][0]

    onCall({
      type: 'queue.started',
      payload: {
        id: 'p2',
        started_at: 1700000005,
      }
    })

    await flushPromises()

    // testproject's job should still be pending; the running section should
    // stay empty since the job that started belongs to another project
    expect(wrapper.find('.pending-row').exists()).toBe(true)
    expect(wrapper.text()).toContain('lifecycle/ideas/mine.md')
    expect(wrapper.find('.running-row').exists()).toBe(false)
  })

  // Test FR-5: Real-time updates - queue.finished/cancelled removes job from waiting sections
  it('removes job from pending when queue.finished event for current project', async () => {
    _listQueueMock.mockResolvedValue({
      ...emptySnapshot(),
      pending: [makeJob({ id: 'p1', project: 'testproject' })],
    })

    const wrapper = await mountProjectQueuePanel()

    // Simulate a queue.finished event for the current project
    const onCall = _wsOnMock.mock.calls[0][0]

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
    _listQueueMock.mockResolvedValue({
      ...emptySnapshot(),
      pending: [makeJob({ id: 'p1', project: 'testproject' })],
    })

    const wrapper = await mountProjectQueuePanel()

    // Simulate a queue.cancelled event for the current project
    const onCall = _wsOnMock.mock.calls[0][0]

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
    const onCall = _wsOnMock.mock.calls[0][0]

    onCall({
      type: 'queue.added',
      payload: {
        id: 'new-job-1',
        project: 'otherproject', // different project
        artifact_path: 'lifecycle/ideas/new.md',
        agent_name: 'requirements-analyst',
        state: 'pending',
        position: 5,
        enqueued_at: '2023-11-14T22:13:25Z',
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
    const onCall = _wsOnMock.mock.calls[0][0]

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
    _listQueueMock.mockResolvedValue({
      ...emptySnapshot(),
      paused: true,
    })

    const wrapper = await mountProjectQueuePanel()

    // Simulate a queue.resumed event
    const onCall = _wsOnMock.mock.calls[0][0]

    onCall({
      type: 'queue.resumed',
      payload: {}
    })

    await flushPromises()

    // Should no longer show the paused note
    expect(wrapper.find('.paused-note').exists()).toBe(false)
  })
})
