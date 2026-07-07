// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Unit tests for the ProjectQueuePanel component.
 *
 * Covers Milestone 2 requirements from the test plan:
 * - Panel shows project-scoped data correctly
 * - Running section shows only jobs belonging to current project
 * - Pending count indicator is accurate and accessible
 * - Artifact navigation works correctly
 * - Cancel action works properly
 * - Empty state is displayed when appropriate
 * - Paused state is handled correctly
 * - Link to global queue is present
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

describe('ProjectQueuePanel component', () => {
  // Test FR-2: Project-scoped data source
  it('shows only jobs belonging to the current project', async () => {
    _snapshotRef.value = {
      running: null,
      pending: [
        makeJob({ id: 'p1', project: 'project-a', position: 1, artifact_path: 'lifecycle/ideas/other-a.md' }),
        makeJob({ id: 'p2', project: 'project-b', position: 2, artifact_path: 'lifecycle/ideas/other-b.md' }),
        makeJob({ id: 'p3', project: 'testproject', position: 3, artifact_path: 'lifecycle/ideas/mine.md' }),
      ],
      recent: [],
      paused: false,
      paused_until: null,
      pause_reason: null,
    }

    const wrapper = await mountProjectQueuePanel('testproject')

    // Should only show jobs for testproject
    expect(wrapper.text()).toContain('lifecycle/ideas/mine.md') // job from current project
    expect(wrapper.text()).not.toContain('lifecycle/ideas/other-a.md') // job from other project
    expect(wrapper.text()).not.toContain('lifecycle/ideas/other-b.md') // job from other project
  })

  // Test FR-3: Sections shown
  it('shows running section when job belongs to project', async () => {
    _snapshotRef.value.running = makeJob({
      id: 'run-1',
      project: 'testproject',
      state: 'running',
      artifact_path: 'lifecycle/ideas/running-job.md',
    })

    const wrapper = await mountProjectQueuePanel()

    expect(wrapper.text()).toContain('lifecycle/ideas/running-job.md')
    expect(wrapper.text()).toContain('requirements-analyst')
  })

  it('shows idle row when no running job belongs to project', async () => {
    _snapshotRef.value.running = makeJob({
      id: 'run-1',
      project: 'otherproject',
      state: 'running'
    })

    const wrapper = await mountProjectQueuePanel()

    expect(wrapper.text()).toContain('Nothing running for this project')
  })

  it('shows pending jobs in position order', async () => {
    _snapshotRef.value.pending = [
      makeJob({ id: 'p2', project: 'testproject', position: 2, artifact_path: 'lifecycle/ideas/second.md' }),
      makeJob({ id: 'p1', project: 'testproject', position: 1, artifact_path: 'lifecycle/ideas/first.md' }),
      makeJob({ id: 'p3', project: 'testproject', position: 3, artifact_path: 'lifecycle/ideas/third.md' }),
    ]

    const wrapper = await mountProjectQueuePanel()

    // Should be sorted by position
    const pendingRows = wrapper.findAll('.pending-row')
    expect(pendingRows.length).toBe(3)
    expect(pendingRows[0].text()).toContain('lifecycle/ideas/first.md') // first in position order
    expect(pendingRows[1].text()).toContain('lifecycle/ideas/second.md') // second in position order
    expect(pendingRows[2].text()).toContain('lifecycle/ideas/third.md') // third in position order
  })

  it('shows pending count indicator with aria-label', async () => {
    _snapshotRef.value.pending = [
      makeJob({ id: 'p1', project: 'testproject' }),
      makeJob({ id: 'p2', project: 'testproject' }),
    ]

    const wrapper = await mountProjectQueuePanel()

    const badge = wrapper.find('.pending-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toContain('2 pending')
    expect(badge.attributes('aria-label')).toBe('2 jobs pending for this project')
  })

  // Test FR-4: Cancel a pending job
  it('cancel button calls store.cancel with correct id', async () => {
    _snapshotRef.value.pending = [
      makeJob({ id: 'cancel-me', project: 'testproject' }),
    ]

    const wrapper = await mountProjectQueuePanel()
    const btn = wrapper.find('.btn-cancel')
    expect(btn.exists()).toBe(true)

    await btn.trigger('click')
    await flushPromises()

    expect(_cancelMock).toHaveBeenCalledOnce()
    expect(_cancelMock).toHaveBeenCalledWith('cancel-me')
  })

  // Test FR-6: Link to global queue
  it('includes a link to the global queue', async () => {
    const wrapper = await mountProjectQueuePanel()

    const globalLink = wrapper.find('.global-link')
    expect(globalLink.exists()).toBe(true)
    expect(globalLink.attributes('href')).toBe('/queue')
  })

  // Test FR-7: Empty state
  it('shows empty state when no running, pending or recent jobs', async () => {
    _snapshotRef.value.running = null
    _snapshotRef.value.pending = []
    _snapshotRef.value.recent = []

    const wrapper = await mountProjectQueuePanel()

    expect(wrapper.text()).toContain('No queued work for this project')
  })

  // Test NFR-4: Accessibility
  it('running/idle distinction is not based on colour alone', async () => {
    // Test idle state
    _snapshotRef.value.running = null

    const wrapper = await mountProjectQueuePanel()

    expect(wrapper.find('.idle-row').exists()).toBe(true)
    expect(wrapper.find('.running-row').exists()).toBe(false)
  })

  it('shows artifact navigation for pending jobs', async () => {
    _snapshotRef.value.pending = [
      makeJob({
        id: 'p1',
        project: 'testproject',
        artifact_path: 'lifecycle/ideas/test.md'
      }),
    ]

    const wrapper = await mountProjectQueuePanel()

    const artifactLink = wrapper.find('.artifact-link')
    expect(artifactLink.exists()).toBe(true)
    expect(artifactLink.attributes('href')).toContain('/p/testproject/artifacts/lifecycle/ideas/test.md')
  })

  // Test Q3: Paused state
  it('shows paused state and link when queue is paused', async () => {
    _snapshotRef.value.paused = true

    const wrapper = await mountProjectQueuePanel()

    expect(wrapper.find('.paused-note').exists()).toBe(true)
    expect(wrapper.find('.paused-link').exists()).toBe(true)
  })

  // Test that the panel properly filters by project
  it('filters out jobs from other projects', async () => {
    _snapshotRef.value.pending = [
      makeJob({ id: 'p1', project: 'testproject', artifact_path: 'lifecycle/ideas/mine-1.md' }),
      makeJob({ id: 'p2', project: 'otherproject', artifact_path: 'lifecycle/ideas/other.md' }),
      makeJob({ id: 'p3', project: 'testproject', artifact_path: 'lifecycle/ideas/mine-2.md' }),
    ]

    const wrapper = await mountProjectQueuePanel()

    // Should only show jobs from testproject
    expect(wrapper.text()).toContain('lifecycle/ideas/mine-1.md')
    expect(wrapper.text()).toContain('lifecycle/ideas/mine-2.md')
    expect(wrapper.text()).not.toContain('lifecycle/ideas/other.md')
  })
})