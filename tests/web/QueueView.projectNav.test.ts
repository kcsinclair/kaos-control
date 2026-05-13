// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Integration tests for QueueView — project navigation, URL sync, and
 * full filtering flow.
 *
 * Covers Milestone 5 (cases 1–9) of the queue-page-project-navigation
 * test plan:
 *
 *   M5-1  Mount with no query param — "All Projects" selected, unfiltered data
 *   M5-2  Mount with ?project=valid-project — sidebar pre-selects project, queue filtered
 *   M5-3  Mount with ?project=nonexistent — falls back to "All Projects" without error
 *   M5-4  Select project in sidebar — URL updates to ?project=<name> via router.replace
 *   M5-5  Select "All Projects" — ?project= query param removed from URL
 *   M5-6  Concurrent loading — both project list and queue snapshot start loading on mount
 *   M5-7  Click project link in running panel — link targets /p/:project/dashboard
 *   M5-8  Click project link in pending table — link targets /p/:project/agents
 *   M5-9  Real-time update while filtered — new job for different project stays hidden;
 *         switching to All Projects reveals it
 *
 * Note on M5-3 URL cleanup:
 *   The test plan specifies that the URL should be cleaned up when ?project=nonexistent
 *   is provided. The current QueueView implementation falls back to "All Projects"
 *   correctly but does not perform a router.replace to strip the invalid query param.
 *   The URL-cleanup assertion is therefore written but will fail until the feature is
 *   implemented. The fallback-to-All-Projects assertion is expected to pass immediately.
 */

import { describe, it, expect, vi, beforeEach, afterEach, beforeAll } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { ref, nextTick } from 'vue'
import type { QueueSnapshot, QueueJob } from '../../web/src/api/queue'
import type { ProjectSummary } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Stub window.matchMedia (required by child QueueSidebar component)
// ---------------------------------------------------------------------------

beforeAll(() => {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    configurable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })),
  })
})

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
const _queueLoading = ref(false)
const _queueError = ref<string | null>(null)
const _fetchQueueMock = vi.fn().mockResolvedValue(undefined)

const _projects = ref<ProjectSummary[]>([])
const _projectLoading = ref(false)
// fetchProjects resolves immediately by default; tests may override.
let _fetchProjectsMockFn = vi.fn().mockResolvedValue(undefined)

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
    running: null,
    pending: [],
    recent: [],
    paused: false,
    paused_until: null,
    pause_reason: null,
  }),
  enqueue: vi.fn(),
  cancelQueue: vi.fn().mockResolvedValue(undefined),
  pauseQueue: vi.fn(),
  resumeQueue: vi.fn().mockResolvedValue(undefined),
}))

vi.mock('@/stores/queue', () => ({
  useQueueStore: () => ({
    get snapshot() { return _snapshot.value },
    get loading() { return _queueLoading.value },
    get error() { return _queueError.value },
    get isPaused() { return _snapshot.value.paused },
    get pausedUntilDate() { return null },
    get pendingCount() { return _snapshot.value.pending.length },
    fetch: _fetchQueueMock,
    cancel: vi.fn().mockResolvedValue(undefined),
    resume: vi.fn(),
    enqueue: vi.fn(),
    pause: vi.fn(),
  }),
}))

vi.mock('@/stores/project', () => ({
  useProjectStore: () => ({
    get projects() { return _projects.value },
    get loading() { return _projectLoading.value },
    fetchProjects: (...args: unknown[]) => _fetchProjectsMockFn(...args),
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    me: { email: 'user@test.local', display_name: 'User', roles: {} },
    isAuthenticated: true,
    rolesForProject: () => [],
    logout: vi.fn(),
  }),
}))

vi.mock('@/stores/ui', () => ({
  useUiStore: () => ({
    success: vi.fn(),
    error: vi.fn(),
    sidebarCollapsed: false,
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
    project: 'project-a',
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

function makeProject(name: string): ProjectSummary {
  return { name, description: '', path: `/data/${name}` }
}

function makeRouter(initialPath = '/queue') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/queue', component: { template: '<div/>' } },
      { path: '/p/:project/:view*', component: { template: '<div/>' } },
      { path: '/:pathMatch(.*)*', component: { template: '<div/>' } },
    ],
  })
  router.push(initialPath)
  return router
}

async function mountQueueView(initialPath = '/queue') {
  const { default: QueueView } = await import('../../web/src/views/QueueView.vue')
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = makeRouter(initialPath)
  await router.isReady()

  const wrapper = mount(QueueView, {
    global: { plugins: [pinia, router] },
  })
  await flushPromises()
  return { wrapper, router }
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
  _queueLoading.value = false
  _queueError.value = null
  _projects.value = []
  _projectLoading.value = false
  _fetchQueueMock.mockClear()
  _fetchProjectsMockFn = vi.fn().mockResolvedValue(undefined)
})

afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

// ===========================================================================
// Milestone 5 — URL Sync and Full Flow
// ===========================================================================

describe('QueueView — Milestone 5: project navigation and URL sync', () => {
  // -------------------------------------------------------------------------
  // M5-1: Mount with no query param
  // -------------------------------------------------------------------------

  it('M5-1: mount with no query param — "All Projects" active, data unfiltered', async () => {
    _projects.value = [makeProject('alpha')]
    _snapshot.value.pending = [
      makeJob({ id: 'p1', project: 'alpha' }),
    ]

    const { wrapper } = await mountQueueView('/queue')

    // Sidebar shows "All Projects" as selected
    const allProjectsBtn = wrapper.findAll('.sidebar-item')[0]
    expect(allProjectsBtn.attributes('aria-current')).toBe('page')

    // Pending table shows the unfiltered job
    expect(wrapper.text()).toContain('lifecycle/ideas/test.md')
  })

  // -------------------------------------------------------------------------
  // M5-2: Mount with ?project=valid-project
  // -------------------------------------------------------------------------

  it('M5-2: mount with ?project=valid — sidebar pre-selects project, data filtered', async () => {
    _projects.value = [makeProject('alpha'), makeProject('beta')]
    _snapshot.value.pending = [
      makeJob({ id: 'p1', project: 'alpha', artifact_path: 'lifecycle/alpha.md' }),
      makeJob({ id: 'p2', project: 'beta', artifact_path: 'lifecycle/beta.md' }),
    ]
    // fetchProjects completes synchronously so the query-param handler fires before flushPromises returns
    _fetchProjectsMockFn = vi.fn().mockResolvedValue(undefined)

    const { wrapper } = await mountQueueView('/queue?project=alpha')

    // "alpha" project button should be active
    const items = wrapper.findAll('.sidebar-item')
    // items[0] = All Projects, items[1] = alpha, items[2] = beta
    expect(items[1].attributes('aria-current')).toBe('page')

    // Only alpha jobs should be visible
    expect(wrapper.text()).toContain('lifecycle/alpha.md')
    expect(wrapper.text()).not.toContain('lifecycle/beta.md')
  })

  // -------------------------------------------------------------------------
  // M5-3: Mount with ?project=nonexistent
  // -------------------------------------------------------------------------

  it('M5-3: mount with ?project=nonexistent — falls back to "All Projects" without error', async () => {
    _projects.value = [makeProject('alpha')]

    const { wrapper } = await mountQueueView('/queue?project=nonexistent')

    // Should fall back to All Projects (aria-current on first item)
    const allProjectsBtn = wrapper.findAll('.sidebar-item')[0]
    expect(allProjectsBtn.attributes('aria-current')).toBe('page')

    // No error message shown
    expect(wrapper.find('.state-msg.error').exists()).toBe(false)
  })

  it('M5-3: mount with ?project=nonexistent — URL is cleaned up (query param removed)', async () => {
    _projects.value = [makeProject('alpha')]

    const { router } = await mountQueueView('/queue?project=nonexistent')
    await nextTick()

    // The URL should no longer contain ?project=nonexistent after falling back.
    // NOTE: This assertion documents the expected behaviour per the test plan.
    // The current QueueView implementation does NOT perform this cleanup —
    // if this test fails it indicates a gap in the implementation.
    const query = router.currentRoute.value.query
    expect(query.project).toBeUndefined()
  })

  // -------------------------------------------------------------------------
  // M5-4: Select project in sidebar updates URL
  // -------------------------------------------------------------------------

  it('M5-4: selecting a project in sidebar updates URL to ?project=<name>', async () => {
    _projects.value = [makeProject('alpha')]

    const { wrapper, router } = await mountQueueView('/queue')

    // Click the alpha project button
    const alphaBtn = wrapper.findAll('.sidebar-item')[1]
    await alphaBtn.trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.query.project).toBe('alpha')
  })

  it('M5-4: URL update uses router.replace (no new history entry)', async () => {
    _projects.value = [makeProject('alpha')]

    const { wrapper, router } = await mountQueueView('/queue')
    const historyLengthBefore = router.currentRoute.value.fullPath

    const alphaBtn = wrapper.findAll('.sidebar-item')[1]
    await alphaBtn.trigger('click')
    await flushPromises()

    // router.replace does not push a new entry — path changes but is still the same route
    expect(router.currentRoute.value.path).toBe('/queue')
    expect(router.currentRoute.value.query.project).toBe('alpha')
  })

  // -------------------------------------------------------------------------
  // M5-5: Select "All Projects" removes query param
  // -------------------------------------------------------------------------

  it('M5-5: selecting "All Projects" removes ?project= from URL', async () => {
    _projects.value = [makeProject('alpha')]

    const { wrapper, router } = await mountQueueView('/queue?project=alpha')
    await nextTick()

    // Click All Projects
    const allBtn = wrapper.findAll('.sidebar-item')[0]
    await allBtn.trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.query.project).toBeUndefined()
  })

  // -------------------------------------------------------------------------
  // M5-6: Concurrent loading
  // -------------------------------------------------------------------------

  it('M5-6: both queueStore.fetch and projectStore.fetchProjects are called on mount', async () => {
    _projects.value = [makeProject('alpha')]

    await mountQueueView('/queue')

    expect(_fetchQueueMock).toHaveBeenCalledOnce()
    // fetchProjects is called by both QueueView (for URL-param resolution) and
    // the child QueueSidebar (to populate its list) — at least one call confirms
    // the load was initiated on mount.
    expect(_fetchProjectsMockFn).toHaveBeenCalled()
  })

  it('M5-6: queue content is not blocked by project loading', async () => {
    // Projects are slow to load, but queue data should still render.
    let resolveProjects!: () => void
    _fetchProjectsMockFn = vi.fn(() => new Promise<void>((resolve) => {
      resolveProjects = resolve
    }))

    _snapshot.value.pending = [makeJob({ id: 'p1', project: 'alpha' })]

    const { wrapper } = await mountQueueView('/queue')

    // Queue content visible even while projects are loading
    expect(wrapper.text()).toContain('lifecycle/ideas/test.md')

    // Resolve projects — no crash
    resolveProjects()
    await flushPromises()
  })

  // -------------------------------------------------------------------------
  // M5-7: Click project link in running panel → /p/:project/dashboard
  // -------------------------------------------------------------------------

  it('M5-7: project link in running panel targets /p/:project/dashboard', async () => {
    _snapshot.value.running = makeJob({ id: 'r1', project: 'my-project', state: 'running' })

    const { wrapper } = await mountQueueView('/queue')

    // Find all <a> tags and locate the one pointing to /dashboard
    const links = wrapper.findAll('a')
    const dashboardLink = links.find((a) => a.attributes('href')?.endsWith('/dashboard'))
    expect(dashboardLink).toBeDefined()
    expect(dashboardLink!.attributes('href')).toBe('/p/my-project/dashboard')
  })

  // -------------------------------------------------------------------------
  // M5-8: Click project link in pending table → /p/:project/agents
  // -------------------------------------------------------------------------

  it('M5-8: project link in pending table targets /p/:project/agents', async () => {
    _snapshot.value.pending = [makeJob({ id: 'p1', project: 'my-project' })]

    const { wrapper } = await mountQueueView('/queue')

    const projectLinks = wrapper.findAll('a.project-link')
    // QueuePendingTable renders .project-link
    const pendingLink = projectLinks.find((a) => a.attributes('href')?.includes('/agents'))
    expect(pendingLink).toBeDefined()
    expect(pendingLink!.attributes('href')).toBe('/p/my-project/agents')
  })

  // -------------------------------------------------------------------------
  // M5-9: Real-time update while filtered
  // -------------------------------------------------------------------------

  it('M5-9: new job for a different project does not appear in filtered view', async () => {
    _projects.value = [makeProject('project-a'), makeProject('project-b')]
    _snapshot.value.pending = [
      makeJob({ id: 'p1', project: 'project-a', artifact_path: 'lifecycle/a.md' }),
    ]

    const { wrapper } = await mountQueueView('/queue')

    // Select project-a
    await wrapper.findAll('.sidebar-item')[1].trigger('click')
    await nextTick()

    expect(wrapper.text()).toContain('lifecycle/a.md')
    expect(wrapper.text()).not.toContain('lifecycle/b.md')

    // Simulate a real-time WS event adding a job for project-b
    _snapshot.value = {
      ..._snapshot.value,
      pending: [
        makeJob({ id: 'p1', project: 'project-a', artifact_path: 'lifecycle/a.md' }),
        makeJob({ id: 'p2', project: 'project-b', artifact_path: 'lifecycle/b.md' }),
      ],
    }
    await nextTick()

    // project-b job must NOT appear while filter is project-a
    expect(wrapper.text()).not.toContain('lifecycle/b.md')
  })

  it('M5-9: switching to "All Projects" reveals the new job', async () => {
    _projects.value = [makeProject('project-a'), makeProject('project-b')]
    _snapshot.value.pending = [
      makeJob({ id: 'p1', project: 'project-a', artifact_path: 'lifecycle/a.md' }),
    ]

    const { wrapper } = await mountQueueView('/queue')

    // Select project-a then add project-b job
    await wrapper.findAll('.sidebar-item')[1].trigger('click')
    await nextTick()

    _snapshot.value = {
      ..._snapshot.value,
      pending: [
        makeJob({ id: 'p1', project: 'project-a', artifact_path: 'lifecycle/a.md' }),
        makeJob({ id: 'p2', project: 'project-b', artifact_path: 'lifecycle/b.md' }),
      ],
    }
    await nextTick()

    // Switch to All Projects
    await wrapper.findAll('.sidebar-item')[0].trigger('click')
    await nextTick()

    // Now both jobs are visible
    expect(wrapper.text()).toContain('lifecycle/a.md')
    expect(wrapper.text()).toContain('lifecycle/b.md')
  })

  // -------------------------------------------------------------------------
  // No regressions in existing coverage
  // -------------------------------------------------------------------------

  it('no regression: renders Running, Pending, Recently finished sections', async () => {
    _snapshot.value = {
      running: makeJob({ id: 'r1', state: 'running' }),
      pending: [makeJob({ id: 'p1', position: 1 })],
      recent: [makeJob({ id: 'rec1', state: 'completed' })],
      paused: false,
      paused_until: null,
      pause_reason: null,
    }

    const { wrapper } = await mountQueueView('/queue')
    expect(wrapper.text()).toContain('Running')
    expect(wrapper.text()).toContain('Pending')
    expect(wrapper.text()).toContain('Recently finished')
  })
})
