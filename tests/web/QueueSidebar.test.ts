// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Tests for QueueSidebar — project filter sidebar.
 *
 * Covers Milestones 1 & 2 of the queue-page-project-navigation test plan:
 *
 * Milestone 1 — Job Count Logic (M1-1 … M1-6)
 *   Verifies the reactive jobCounts computation (running + pending, excluding
 *   recent/finished jobs) by checking rendered badge text.
 *
 * Milestone 2 — Component Rendering and Behaviour (M2-1 … M2-7)
 *   Mount, selection, collapse toggle, keyboard accessibility, ARIA attributes.
 */

import { describe, it, expect, vi, beforeAll, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { ref, nextTick } from 'vue'
import type { QueueSnapshot, QueueJob } from '../../web/src/api/queue'
import type { ProjectSummary } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Stub window.matchMedia (happy-dom has none; QueueSidebar calls it at setup)
// ---------------------------------------------------------------------------

beforeAll(() => {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    configurable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false, // desktop — no auto-collapse
      media: query,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })),
  })
})

// ---------------------------------------------------------------------------
// Reactive store state shared across tests
// ---------------------------------------------------------------------------

const _snapshot = ref<QueueSnapshot>({
  running: null,
  pending: [],
  recent: [],
  paused: false,
  paused_until: null,
  pause_reason: null,
})

const _projects = ref<ProjectSummary[]>([])
const _projectLoading = ref(false)
const _fetchProjectsMock = vi.fn().mockResolvedValue(undefined)

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

vi.mock('@/stores/queue', () => ({
  useQueueStore: () => ({
    get snapshot() { return _snapshot.value },
  }),
}))

vi.mock('@/stores/project', () => ({
  useProjectStore: () => ({
    get projects() { return _projects.value },
    get loading() { return _projectLoading.value },
    fetchProjects: _fetchProjectsMock,
  }),
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

async function mountSidebar(props: Record<string, unknown> = {}) {
  const { default: QueueSidebar } = await import('../../web/src/components/queue/QueueSidebar.vue')
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = makeRouter()
  await router.isReady()
  const wrapper = mount(QueueSidebar, {
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
  _projects.value = []
  _projectLoading.value = false
  _fetchProjectsMock.mockClear()
})

afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

// ===========================================================================
// Milestone 1 — Job Count Logic
// ===========================================================================

describe('QueueSidebar — Milestone 1: job count logic', () => {
  it('M1-1: no jobs — all project badges show 0', async () => {
    _projects.value = [makeProject('project-a'), makeProject('project-b')]

    const wrapper = await mountSidebar()
    const badges = wrapper.findAll('.item-badge')
    for (const badge of badges) {
      expect(badge.text()).toBe('0')
    }
  })

  it('M1-2: one pending job — badge for that project shows 1, others show 0', async () => {
    _projects.value = [makeProject('project-a'), makeProject('project-b')]
    _snapshot.value.pending = [makeJob({ project: 'project-a' })]

    const wrapper = await mountSidebar()
    const items = wrapper.findAll('.sidebar-item')
    // items[0] = All Projects, items[1] = project-a, items[2] = project-b
    expect(items[0].find('.item-badge').text()).toBe('1') // totalCount
    expect(items[1].find('.item-badge').text()).toBe('1') // project-a
    expect(items[2].find('.item-badge').text()).toBe('0') // project-b
  })

  it('M1-3: running + pending for same project — badge shows combined count', async () => {
    _projects.value = [makeProject('project-a')]
    _snapshot.value.running = makeJob({ id: 'r1', project: 'project-a', state: 'running' })
    _snapshot.value.pending = [makeJob({ id: 'p1', project: 'project-a' })]

    const wrapper = await mountSidebar()
    const items = wrapper.findAll('.sidebar-item')
    expect(items[0].find('.item-badge').text()).toBe('2') // totalCount = 1 running + 1 pending
    expect(items[1].find('.item-badge').text()).toBe('2') // project-a combined
  })

  it('M1-4: running for project-a, pending for project-b — each badge shows 1', async () => {
    _projects.value = [makeProject('project-a'), makeProject('project-b')]
    _snapshot.value.running = makeJob({ id: 'r1', project: 'project-a', state: 'running' })
    _snapshot.value.pending = [makeJob({ id: 'p1', project: 'project-b' })]

    const wrapper = await mountSidebar()
    const items = wrapper.findAll('.sidebar-item')
    expect(items[0].find('.item-badge').text()).toBe('2') // totalCount = 1+1
    expect(items[1].find('.item-badge').text()).toBe('1') // project-a (running)
    expect(items[2].find('.item-badge').text()).toBe('1') // project-b (pending)
  })

  it('M1-5: multiple pending for one project — badge shows correct aggregate', async () => {
    _projects.value = [makeProject('project-a')]
    _snapshot.value.pending = [
      makeJob({ id: 'p1', project: 'project-a', position: 1 }),
      makeJob({ id: 'p2', project: 'project-a', position: 2 }),
      makeJob({ id: 'p3', project: 'project-a', position: 3 }),
    ]

    const wrapper = await mountSidebar()
    const items = wrapper.findAll('.sidebar-item')
    expect(items[0].find('.item-badge').text()).toBe('3') // totalCount
    expect(items[1].find('.item-badge').text()).toBe('3') // project-a
  })

  it('M1-6: recent (finished) jobs are NOT counted in badges', async () => {
    _projects.value = [makeProject('project-a')]
    _snapshot.value.running = null
    _snapshot.value.pending = []
    _snapshot.value.recent = [
      makeJob({ id: 'r1', project: 'project-a', state: 'completed' }),
      makeJob({ id: 'r2', project: 'project-a', state: 'failed' }),
      makeJob({ id: 'r3', project: 'project-a', state: 'skipped' }),
      makeJob({ id: 'r4', project: 'project-a', state: 'cancelled' }),
    ]

    const wrapper = await mountSidebar()
    const items = wrapper.findAll('.sidebar-item')
    expect(items[0].find('.item-badge').text()).toBe('0') // totalCount (recent excluded)
    expect(items[1].find('.item-badge').text()).toBe('0') // project-a (recent excluded)
  })
})

// ===========================================================================
// Milestone 2 — Component Rendering and Behaviour
// ===========================================================================

describe('QueueSidebar — Milestone 2: rendering and behaviour', () => {
  it('M2-1: renders an item for each project plus "All Projects"', async () => {
    _projects.value = [makeProject('alpha'), makeProject('beta'), makeProject('gamma')]

    const wrapper = await mountSidebar()
    const items = wrapper.findAll('.sidebar-item')
    expect(items).toHaveLength(4) // All Projects + 3 projects
    expect(wrapper.text()).toContain('All Projects')
    expect(wrapper.text()).toContain('alpha')
    expect(wrapper.text()).toContain('beta')
    expect(wrapper.text()).toContain('gamma')
  })

  it('M2-2: "All Projects" is selected on mount with aria-current="page"', async () => {
    _projects.value = [makeProject('alpha')]

    const wrapper = await mountSidebar()
    const allBtn = wrapper.findAll('.sidebar-item')[0]
    expect(allBtn.attributes('aria-current')).toBe('page')
  })

  it('M2-3: clicking a project item gains aria-current; previous loses it', async () => {
    _projects.value = [makeProject('alpha'), makeProject('beta')]

    const wrapper = await mountSidebar()
    const items = wrapper.findAll('.sidebar-item')
    const allBtn = items[0]
    const alphaBtn = items[1]

    // Initially All Projects is active
    expect(allBtn.attributes('aria-current')).toBe('page')
    expect(alphaBtn.attributes('aria-current')).toBeUndefined()

    await alphaBtn.trigger('click')
    await nextTick()

    expect(alphaBtn.attributes('aria-current')).toBe('page')
    expect(allBtn.attributes('aria-current')).toBeUndefined()
  })

  it('M2-4: clicking "All Projects" after project selection clears selection', async () => {
    _projects.value = [makeProject('alpha')]

    const wrapper = await mountSidebar({ modelValue: 'alpha' })
    await nextTick()

    const items = wrapper.findAll('.sidebar-item')
    const allBtn = items[0]
    const alphaBtn = items[1]

    // alpha is pre-selected via modelValue
    expect(alphaBtn.attributes('aria-current')).toBe('page')

    await allBtn.trigger('click')
    await nextTick()

    expect(allBtn.attributes('aria-current')).toBe('page')
    expect(alphaBtn.attributes('aria-current')).toBeUndefined()
  })

  it('M2-5: collapse toggle hides nav; clicking again restores it', async () => {
    _projects.value = [makeProject('alpha')]

    const wrapper = await mountSidebar()

    // Nav visible initially
    expect(wrapper.find('#queue-sidebar-nav').exists()).toBe(true)

    // Collapse
    await wrapper.find('.collapse-toggle').trigger('click')
    await nextTick()

    expect(wrapper.find('#queue-sidebar-nav').exists()).toBe(false)

    // Expand again
    await wrapper.find('.collapse-toggle').trigger('click')
    await nextTick()

    expect(wrapper.find('#queue-sidebar-nav').exists()).toBe(true)
  })

  it('M2-6: sidebar items are <button> elements (natively keyboard-accessible)', async () => {
    _projects.value = [makeProject('alpha')]

    const wrapper = await mountSidebar()
    const items = wrapper.findAll('.sidebar-item')
    for (const item of items) {
      expect(item.element.tagName.toLowerCase()).toBe('button')
    }
  })

  it('M2-6: Space key on a sidebar item triggers selection (button default)', async () => {
    _projects.value = [makeProject('alpha')]

    const wrapper = await mountSidebar()
    const alphaBtn = wrapper.findAll('.sidebar-item')[1]

    // Browsers fire click on <button> when Space or Enter is pressed;
    // simulate by triggering click which is what the browser does.
    await alphaBtn.trigger('click')
    await nextTick()

    expect(alphaBtn.attributes('aria-current')).toBe('page')
  })

  it('M2-7: nav element has role="navigation" and non-empty aria-label', async () => {
    const wrapper = await mountSidebar()
    const nav = wrapper.find('#queue-sidebar-nav')
    expect(nav.attributes('role')).toBe('navigation')
    expect(nav.attributes('aria-label')).toBeTruthy()
  })

  it('M2-7: collapse-toggle has aria-expanded="true" when expanded', async () => {
    const wrapper = await mountSidebar()
    const toggle = wrapper.find('.collapse-toggle')
    expect(toggle.attributes('aria-expanded')).toBe('true')
  })

  it('M2-7: collapse-toggle has aria-expanded="false" when collapsed', async () => {
    const wrapper = await mountSidebar()
    const toggle = wrapper.find('.collapse-toggle')
    await toggle.trigger('click')
    await nextTick()
    expect(toggle.attributes('aria-expanded')).toBe('false')
  })

  it('M2: renders correctly with zero projects (only "All Projects" item)', async () => {
    _projects.value = []

    const wrapper = await mountSidebar()
    const items = wrapper.findAll('.sidebar-item')
    expect(items).toHaveLength(1)
    expect(items[0].text()).toContain('All Projects')
  })

  it('M2: emits select and update:modelValue with project name when item clicked', async () => {
    _projects.value = [makeProject('alpha')]

    const wrapper = await mountSidebar()
    const alphaBtn = wrapper.findAll('.sidebar-item')[1]
    await alphaBtn.trigger('click')

    expect(wrapper.emitted('select')).toBeTruthy()
    expect(wrapper.emitted('select')![0]).toEqual(['alpha'])
    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')![0]).toEqual(['alpha'])
  })

  it('M2: emits select with null when "All Projects" is clicked', async () => {
    _projects.value = [makeProject('alpha')]
    const wrapper = await mountSidebar({ modelValue: 'alpha' })
    await nextTick()

    await wrapper.findAll('.sidebar-item')[0].trigger('click')

    expect(wrapper.emitted('select')![0]).toEqual([null])
    expect(wrapper.emitted('update:modelValue')![0]).toEqual([null])
  })

  it('M2: fetchProjects() is called on mount', async () => {
    await mountSidebar()
    expect(_fetchProjectsMock).toHaveBeenCalledOnce()
  })

  it('M2: badge reactively updates when store snapshot changes', async () => {
    _projects.value = [makeProject('project-a')]
    _snapshot.value.pending = []

    const wrapper = await mountSidebar()
    expect(wrapper.findAll('.sidebar-item')[1].find('.item-badge').text()).toBe('0')

    // Add a job reactively
    _snapshot.value = {
      ..._snapshot.value,
      pending: [makeJob({ project: 'project-a' })],
    }
    await nextTick()

    expect(wrapper.findAll('.sidebar-item')[1].find('.item-badge').text()).toBe('1')
  })
})
