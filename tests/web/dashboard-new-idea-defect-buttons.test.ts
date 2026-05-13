// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Integration tests for Dashboard New Idea & Defect Buttons feature.
 *
 * Covers:
 *   Milestone 1 — Dashboard button presence and layout
 *     - .btn-new-idea present in .dashboard-header
 *     - .btn-new-defect present in .dashboard-header
 *     - .btn-new-defect precedes .btn-new-idea in DOM order (FR-4)
 *     - Buttons are right-aligned via .header-actions (margin-left:auto container)
 *
 *   Milestone 2 — Dashboard modal integration: idea flow
 *     - Clicking "New Idea" renders BrainDumpModal with artifactType="idea"
 *     - After created event fires, router.push is called with artifact path
 *     - Success toast is triggered after created event
 *
 *   Milestone 3 — Dashboard modal integration: defect flow
 *     - Clicking "New Defect" renders BrainDumpModal with artifactType="defect"
 *     - After created event fires for defect, router.push is called
 *
 *   Milestone 4 — Modal dismiss and focus return
 *     - After opening via "New Idea" and emitting close, modal is removed
 *     - focus returns to the "New Idea" button after close
 *     - After opening via "New Defect" and emitting close, focus returns to "New Defect" button
 *
 *   Milestone 5 — Artifacts page button reordering
 *     - On ArtifactListView, .btn-new-idea precedes .btn-new-defect in DOM order
 *     - Both buttons function (clicking each opens the modal in the correct mode)
 *     - .btn-check-status is unaffected
 *
 * Notes:
 *   BrainDumpModal is stubbed to avoid Teleport/async complexity. The stub
 *   renders a sentinel element with a data-artifact-type attribute so we can
 *   assert the prop value without mounting the full modal.
 *
 *   Focus-return assertions (M4) require attachTo: document.body so that
 *   happy-dom tracks document.activeElement correctly.
 *
 *   CSS layout assertions (right-alignment, flex display) cannot be evaluated
 *   in happy-dom because it does not process scoped <style> blocks. We assert
 *   the structural proxy instead: the buttons must be children of
 *   .header-actions, which has margin-left:auto defined in source CSS.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { nextTick } from 'vue'

import DashboardView from '../../web/src/views/project/DashboardView.vue'
import ArtifactListView from '../../web/src/views/project/ArtifactListView.vue'

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

const mockPush = vi.fn()
const mockSuccess = vi.fn()

vi.mock('vue-router', async (importActual) => {
  const actual = await importActual<typeof import('vue-router')>()
  return {
    ...actual,
    useRouter: vi.fn(() => ({ push: mockPush, replace: vi.fn() })),
    useRoute: vi.fn(() => ({
      params: { project: 'testproject' },
      query: {},
    })),
  }
})

vi.mock('@/stores/ui', () => ({
  useUiStore: vi.fn(() => ({ success: mockSuccess })),
}))

// Stub BrainDumpModal — renders a sentinel div with the artifactType prop as
// a data attribute so tests can verify which mode was requested.
vi.mock('@/components/idea/BrainDumpModal.vue', () => ({
  default: {
    name: 'BrainDumpModalStub',
    props: ['project', 'artifactType'],
    emits: ['close', 'created'],
    template: `
      <div
        class="bdm-stub"
        :data-artifact-type="artifactType"
        data-testid="brain-dump-modal"
      />
    `,
  },
}))

// Stub DashboardGrid so DashboardView mounts without pulling in widget deps.
vi.mock('@/components/dashboard/DashboardGrid.vue', () => ({
  default: {
    name: 'DashboardGridStub',
    props: ['project'],
    template: '<div class="dashboard-grid-stub" />',
  },
}))

// Stub the brainDump store — DashboardView calls brainDumpStore.reset()
vi.mock('@/stores/brainDump', () => ({
  useBrainDumpStore: vi.fn(() => ({ reset: vi.fn() })),
}))

// ArtifactListView deps
vi.mock('@/api/artifacts', () => ({
  listArtifacts:  vi.fn().mockResolvedValue({ items: [], total: 0 }),
  listLabels:     vi.fn().mockResolvedValue({ labels: [] }),
  listPriorities: vi.fn().mockResolvedValue({ priorities: [] }),
  getArtifact:    vi.fn().mockResolvedValue({ artifact: {}, body: '', body_html: '' }),
}))

vi.mock('@/api/releases', () => ({
  listReleases: vi.fn().mockResolvedValue([]),
}))

vi.mock('@/api/ws', () => ({
  getProjectWs: vi.fn(() => ({ onType: vi.fn(() => () => {}) })),
}))

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  api: {
    get: vi.fn().mockResolvedValue({}),
    post: vi.fn().mockResolvedValue({}),
  },
}))

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeRouter(path = '/p/testproject/dashboard') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/p/:project/dashboard', component: { template: '<div />' } },
      { path: '/p/:project',           component: { template: '<div />' } },
      { path: '/:pathMatch(.*)*',      component: { template: '<div />' } },
    ],
  })
  router.push(path)
  return router
}

async function mountDashboard(opts: { attachTo?: HTMLElement } = {}) {
  const pinia = createPinia()
  setActivePinia(pinia)

  const router = makeRouter()
  await router.isReady()

  const wrapper = mount(DashboardView, {
    global: { plugins: [pinia, router] },
    ...opts,
  })
  await flushPromises()
  return wrapper
}

async function mountArtifactList() {
  const pinia = createPinia()
  setActivePinia(pinia)

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/p/:project', component: ArtifactListView },
      { path: '/:pathMatch(.*)*', component: { template: '<div/>' } },
    ],
  })
  await router.push('/p/testproject')
  await router.isReady()

  const wrapper = mount(ArtifactListView, {
    global: { plugins: [pinia, router] },
  })
  await flushPromises()
  return wrapper
}

// ---------------------------------------------------------------------------
// Setup / teardown
// ---------------------------------------------------------------------------

beforeEach(() => {
  mockPush.mockClear()
  mockSuccess.mockClear()
})

afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

// ===========================================================================
// Milestone 1 — Dashboard button presence and layout
// ===========================================================================

describe('DashboardView — button presence and layout (M1)', () => {
  it('M1-TC1: .btn-new-idea button is present inside .dashboard-header', async () => {
    const wrapper = await mountDashboard()
    const header = wrapper.find('.dashboard-header')
    expect(header.exists()).toBe(true)
    expect(header.find('.btn-new-idea').exists()).toBe(true)
  })

  it('M1-TC2: .btn-new-defect button is present inside .dashboard-header', async () => {
    const wrapper = await mountDashboard()
    const header = wrapper.find('.dashboard-header')
    expect(header.find('.btn-new-defect').exists()).toBe(true)
  })

  it('M1-TC3: .btn-new-defect precedes .btn-new-idea in DOM order (FR-4: Defect left, Idea right)', async () => {
    const wrapper = await mountDashboard()
    const html = wrapper.find('.dashboard-header').html()
    expect(html.indexOf('btn-new-defect')).toBeLessThan(html.indexOf('btn-new-idea'))
  })

  it('M1-TC4: both buttons are children of .header-actions (right-aligned container)', async () => {
    const wrapper = await mountDashboard()
    const actions = wrapper.find('.header-actions')
    expect(actions.exists()).toBe(true)
    expect(actions.find('.btn-new-idea').exists()).toBe(true)
    expect(actions.find('.btn-new-defect').exists()).toBe(true)
  })

  it('M1-TC5: .header-actions is inside .dashboard-header (structural assertion for right-alignment)', async () => {
    const wrapper = await mountDashboard()
    const header = wrapper.find('.dashboard-header')
    expect(header.find('.header-actions').exists()).toBe(true)
  })
})

// ===========================================================================
// Milestone 2 — Dashboard modal integration: idea flow
// ===========================================================================

describe('DashboardView — New Idea modal flow (M2)', () => {
  it('M2-TC1: clicking "New Idea" makes BrainDumpModal appear in the DOM', async () => {
    const wrapper = await mountDashboard()
    expect(wrapper.find('[data-testid="brain-dump-modal"]').exists()).toBe(false)

    await wrapper.find('.btn-new-idea').trigger('click')
    await nextTick()

    expect(wrapper.find('[data-testid="brain-dump-modal"]').exists()).toBe(true)
  })

  it('M2-TC2: BrainDumpModal receives artifactType="idea" when opened via New Idea button', async () => {
    const wrapper = await mountDashboard()
    await wrapper.find('.btn-new-idea').trigger('click')
    await nextTick()

    const modal = wrapper.find('[data-testid="brain-dump-modal"]')
    expect(modal.attributes('data-artifact-type')).toBe('idea')
  })

  it('M2-TC3: after the created event fires, router navigates to the artifact path', async () => {
    const wrapper = await mountDashboard()
    await wrapper.find('.btn-new-idea').trigger('click')
    await nextTick()

    const modal = wrapper.findComponent({ name: 'BrainDumpModalStub' })
    modal.vm.$emit('created', 'lifecycle/ideas/my-idea.md')
    await nextTick()

    expect(mockPush).toHaveBeenCalledWith(
      '/p/testproject/artifacts/lifecycle/ideas/my-idea.md',
    )
  })

  it('M2-TC4: a success toast is shown after the created event fires', async () => {
    const wrapper = await mountDashboard()
    await wrapper.find('.btn-new-idea').trigger('click')
    await nextTick()

    const modal = wrapper.findComponent({ name: 'BrainDumpModalStub' })
    modal.vm.$emit('created', 'lifecycle/ideas/my-idea.md')
    await nextTick()

    expect(mockSuccess).toHaveBeenCalledWith('Artifact created!')
  })

  it('M2-TC5: modal is removed from DOM after the created event fires', async () => {
    const wrapper = await mountDashboard()
    await wrapper.find('.btn-new-idea').trigger('click')
    await nextTick()

    const modal = wrapper.findComponent({ name: 'BrainDumpModalStub' })
    modal.vm.$emit('created', 'lifecycle/ideas/my-idea.md')
    await nextTick()

    expect(wrapper.find('[data-testid="brain-dump-modal"]').exists()).toBe(false)
  })
})

// ===========================================================================
// Milestone 3 — Dashboard modal integration: defect flow
// ===========================================================================

describe('DashboardView — New Defect modal flow (M3)', () => {
  it('M3-TC1: clicking "New Defect" makes BrainDumpModal appear in the DOM', async () => {
    const wrapper = await mountDashboard()
    expect(wrapper.find('[data-testid="brain-dump-modal"]').exists()).toBe(false)

    await wrapper.find('.btn-new-defect').trigger('click')
    await nextTick()

    expect(wrapper.find('[data-testid="brain-dump-modal"]').exists()).toBe(true)
  })

  it('M3-TC2: BrainDumpModal receives artifactType="defect" when opened via New Defect button', async () => {
    const wrapper = await mountDashboard()
    await wrapper.find('.btn-new-defect').trigger('click')
    await nextTick()

    const modal = wrapper.find('[data-testid="brain-dump-modal"]')
    expect(modal.attributes('data-artifact-type')).toBe('defect')
  })

  it('M3-TC3: after defect created event fires, router navigates to the defect artifact path', async () => {
    const wrapper = await mountDashboard()
    await wrapper.find('.btn-new-defect').trigger('click')
    await nextTick()

    const modal = wrapper.findComponent({ name: 'BrainDumpModalStub' })
    modal.vm.$emit('created', 'lifecycle/defects/my-defect.md')
    await nextTick()

    expect(mockPush).toHaveBeenCalledWith(
      '/p/testproject/artifacts/lifecycle/defects/my-defect.md',
    )
  })

  it('M3-TC4: only one modal is rendered at a time (defect replaces idea)', async () => {
    const wrapper = await mountDashboard()

    await wrapper.find('.btn-new-idea').trigger('click')
    await nextTick()
    expect(wrapper.findAll('[data-testid="brain-dump-modal"]')).toHaveLength(1)
    expect(wrapper.find('[data-testid="brain-dump-modal"]').attributes('data-artifact-type')).toBe('idea')

    // Dismiss and reopen as defect
    const modal1 = wrapper.findComponent({ name: 'BrainDumpModalStub' })
    modal1.vm.$emit('close')
    await nextTick()

    await wrapper.find('.btn-new-defect').trigger('click')
    await nextTick()
    expect(wrapper.findAll('[data-testid="brain-dump-modal"]')).toHaveLength(1)
    expect(wrapper.find('[data-testid="brain-dump-modal"]').attributes('data-artifact-type')).toBe('defect')
  })
})

// ===========================================================================
// Milestone 4 — Modal dismiss and focus return
// ===========================================================================

describe('DashboardView — modal dismiss and focus return (M4)', () => {
  // Focus assertions require the component to be attached to a real DOM node.
  function makeAttachTarget() {
    const div = document.createElement('div')
    document.body.appendChild(div)
    return div
  }

  it('M4-TC1: after opening via New Idea and emitting close, modal is removed from DOM', async () => {
    const attachTo = makeAttachTarget()
    const wrapper = await mountDashboard({ attachTo })

    await wrapper.find('.btn-new-idea').trigger('click')
    await nextTick()
    expect(wrapper.find('[data-testid="brain-dump-modal"]').exists()).toBe(true)

    const modal = wrapper.findComponent({ name: 'BrainDumpModalStub' })
    modal.vm.$emit('close')
    await nextTick()

    expect(wrapper.find('[data-testid="brain-dump-modal"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('M4-TC2: focus returns to .btn-new-idea after dismissing via close event', async () => {
    const attachTo = makeAttachTarget()
    const wrapper = await mountDashboard({ attachTo })

    const ideaBtn = wrapper.find('.btn-new-idea').element as HTMLButtonElement
    await wrapper.find('.btn-new-idea').trigger('click')
    await nextTick()

    const modal = wrapper.findComponent({ name: 'BrainDumpModalStub' })
    modal.vm.$emit('close')
    // nextTick twice: once for v-if to remove modal, once for focus() in nextTick callback
    await nextTick()
    await nextTick()

    expect(document.activeElement).toBe(ideaBtn)
    wrapper.unmount()
  })

  it('M4-TC3: focus returns to .btn-new-defect after dismissing via close event', async () => {
    const attachTo = makeAttachTarget()
    const wrapper = await mountDashboard({ attachTo })

    const defectBtn = wrapper.find('.btn-new-defect').element as HTMLButtonElement
    await wrapper.find('.btn-new-defect').trigger('click')
    await nextTick()

    const modal = wrapper.findComponent({ name: 'BrainDumpModalStub' })
    modal.vm.$emit('close')
    await nextTick()
    await nextTick()

    expect(document.activeElement).toBe(defectBtn)
    wrapper.unmount()
  })
})

// ===========================================================================
// Milestone 5 — ArtifactListView button reordering
// ===========================================================================

describe('ArtifactListView — button reordering (M5)', () => {
  it('M5-TC1: .btn-new-idea precedes .btn-new-defect in DOM order', async () => {
    const wrapper = await mountArtifactList()
    const html = wrapper.find('.list-header').html()
    expect(wrapper.find('.btn-new-idea').exists()).toBe(true)
    expect(wrapper.find('.btn-new-defect').exists()).toBe(true)
    expect(html.indexOf('btn-new-idea')).toBeLessThan(html.indexOf('btn-new-defect'))
  })

  it('M5-TC2: clicking .btn-new-idea opens modal in idea mode', async () => {
    const wrapper = await mountArtifactList()
    expect(wrapper.find('[data-testid="brain-dump-modal"]').exists()).toBe(false)

    await wrapper.find('.btn-new-idea').trigger('click')
    await nextTick()

    const modal = wrapper.find('[data-testid="brain-dump-modal"]')
    expect(modal.exists()).toBe(true)
    expect(modal.attributes('data-artifact-type')).toBe('idea')
  })

  it('M5-TC3: clicking .btn-new-defect opens modal in defect mode', async () => {
    const wrapper = await mountArtifactList()
    await wrapper.find('.btn-new-defect').trigger('click')
    await nextTick()

    const modal = wrapper.find('[data-testid="brain-dump-modal"]')
    expect(modal.exists()).toBe(true)
    expect(modal.attributes('data-artifact-type')).toBe('defect')
  })

  it('M5-TC4: .btn-check-status is still present and unaffected', async () => {
    const wrapper = await mountArtifactList()
    expect(wrapper.find('.btn-check-status').exists()).toBe(true)
  })

  it('M5-TC5: .btn-check-status precedes both new-idea and new-defect buttons in DOM order', async () => {
    const wrapper = await mountArtifactList()
    const html = wrapper.find('.list-header').html()
    const checkStatusIdx = html.indexOf('btn-check-status')
    const newIdeaIdx     = html.indexOf('btn-new-idea')
    const newDefectIdx   = html.indexOf('btn-new-defect')
    expect(checkStatusIdx).toBeLessThan(newIdeaIdx)
    expect(checkStatusIdx).toBeLessThan(newDefectIdx)
  })
})
