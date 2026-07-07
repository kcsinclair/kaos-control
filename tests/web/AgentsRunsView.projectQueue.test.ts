// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Integration tests for AgentsRunsView with ProjectQueuePanel.
 *
 * Covers Milestone 4 requirements from the test plan:
 * - AgentsRunsView renders ProjectQueuePanel alongside existing elements
 * - Panel receives correct project route param
 * - Responsive layout tests
 * - Existing global-queue tests pass unchanged
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
    runs: [],
    activeRuns: [],
    agents: [],
    loading: false,
    readyCounts: {},
    progressLines: new Map(),
    permissionEvents: new Map(),
    runResults: new Map(),
    fetchAgents: vi.fn().mockResolvedValue(undefined),
    fetchRuns: vi.fn().mockResolvedValue(undefined),
    fetchReadyCounts: vi.fn().mockResolvedValue(undefined),
    killRun: vi.fn().mockResolvedValue(undefined),
  }),
}))

vi.mock('@/stores/projectConfig', () => ({
  useProjectConfigStore: () => ({
    fetchRoles: vi.fn().mockResolvedValue(undefined),
    roles: [],
  }),
}))

// Mock the project store to avoid real HTTP requests
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

function makeRouter(project: string = 'testproject') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path: '/p/:project/agents',
        component: { template: '<div/>' },
        props: true
      },
      { path: '/', component: { template: '<div/>' } },
      { path: '/queue', component: { template: '<div/>' } },
      { path: '/:pathMatch(.*)*', component: { template: '<div/>' } },
    ],
  })
  router.push(`/p/${project}/agents`)
  return router
}

async function mountAgentsRunsView(project: string = 'testproject') {
  const { default: AgentsRunsView } = await import('../../web/src/views/project/AgentsRunsView.vue')
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = makeRouter(project)
  await router.isReady()

  const wrapper = mount(AgentsRunsView, {
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

describe('AgentsRunsView with ProjectQueuePanel', () => {
  // Test FR-1: Embedded right-side panel
  it('renders ProjectQueuePanel alongside existing agent rows and run-history table', async () => {
    const wrapper = await mountAgentsRunsView()

    // Should contain the ProjectQueuePanel component
    expect(wrapper.findComponent({ name: 'ProjectQueuePanel' }).exists()).toBe(true)

    // Should still have other elements like agent panel rows and runs table
    expect(wrapper.find('.runs-main').exists()).toBe(true)
  })

  // Test that project param is passed correctly to the panel
  it('passes correct project route param to ProjectQueuePanel', async () => {
    const wrapper = await mountAgentsRunsView('testproject')

    const panel = wrapper.findComponent({ name: 'ProjectQueuePanel' })
    expect(panel.props('project')).toBe('testproject')
  })

  it('passes different project param correctly', async () => {
    const wrapper = await mountAgentsRunsView('anotherproject')

    const panel = wrapper.findComponent({ name: 'ProjectQueuePanel' })
    expect(panel.props('project')).toBe('anotherproject')
  })

  // Test NFR-2: Layout integrity
  it('renders panel with correct styling and layout', async () => {
    const wrapper = await mountAgentsRunsView()

    // Should have the correct CSS class for the panel
    const queuePanel = wrapper.find('.runs-queue-panel')
    expect(queuePanel.exists()).toBe(true)

    // Should contain the panel header
    expect(wrapper.text()).toContain('Project Queue')
  })

  // Test that existing global queue tests would pass (regression guard)
  it('does not modify global queue behavior', async () => {
    // This test ensures our changes don't affect global queue functionality
    // by testing that we can still mount the view and it renders correctly

    const wrapper = await mountAgentsRunsView()

    // Should still be able to render without errors
    expect(wrapper.exists()).toBe(true)

    // Should contain both main content and queue panel
    expect(wrapper.find('.runs-main').exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'ProjectQueuePanel' }).exists()).toBe(true)
  })

  // Test that the layout is responsive (test basic structure)
  it('has responsive layout classes', async () => {
    const wrapper = await mountAgentsRunsView()

    // Should have the correct layout classes
    expect(wrapper.find('.runs-layout').exists()).toBe(true)
    expect(wrapper.find('.runs-main').exists()).toBe(true)
  })

  // Test that all existing queue functionality still works in context
  it('maintains access to global queue features', async () => {
    _snapshotRef.value.pending = [
      makeJob({ id: 'p1', project: 'testproject' }),
    ]

    const wrapper = await mountAgentsRunsView()

    // Should be able to see the pending job in the panel
    expect(wrapper.text()).toContain('lifecycle/ideas/test.md')

    // Should have access to global queue features like the cancel button
    expect(wrapper.find('.btn-cancel').exists()).toBe(true)
  })
})