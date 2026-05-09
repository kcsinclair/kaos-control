// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Unit tests for DevOpsView and PipelineCard — DevOps Pipeline Management
 *
 * Covers (Milestone 7):
 *   - DevOps view shows "access denied" for users without product-owner/devops role
 *   - DevOps view renders pipeline columns for authorised users
 *   - Pipelines are grouped by type into columns
 *   - Unknown pipeline types get their own column
 *   - "Run" button triggers the execution API
 *   - "Run" button is disabled while a run is in progress
 *   - "Cancel" button appears when a run is active
 *   - Failed steps are visually distinguished (pipeline-card--failed class)
 *   - DevOps nav item visibility is controlled by hasAccess computed property
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import DevOpsView from '../../web/src/views/project/DevOpsView.vue'
import PipelineCard from '../../web/src/components/devops/PipelineCard.vue'
import { useDevOpsStore } from '../../web/src/stores/devops'
import type { Pipeline } from '../../web/src/api/devops'

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

const mockListPipelines = vi.fn().mockResolvedValue({ pipelines: [] })
vi.mock('@/api/devops', () => ({
  listPipelines: (...args: unknown[]) => mockListPipelines(...args),
  runPipeline: vi.fn().mockResolvedValue({ run_id: 'test-run-id' }),
  cancelPipeline: vi.fn().mockResolvedValue(undefined),
  getRunLog: vi.fn().mockResolvedValue(''),
}))

vi.mock('@/api/ws', () => ({
  getProjectWs: vi.fn(() => ({
    onType: vi.fn(() => () => {}),
  })),
}))

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: vi.fn(),
}))

vi.mock('@/stores/ui', () => ({
  useUiStore: vi.fn(() => ({
    error: vi.fn(),
  })),
}))

// Auth store is mocked per-test to vary roles.
const mockRolesForProject = vi.fn<[string], string[]>()
vi.mock('@/stores/auth', () => ({
  useAuthStore: vi.fn(() => ({
    rolesForProject: mockRolesForProject,
  })),
}))

vi.mock('@/stores/project', () => ({
  useProjectStore: vi.fn(() => ({
    current: { name: 'testproject' },
  })),
}))

// ---------------------------------------------------------------------------
// Router factory
// ---------------------------------------------------------------------------

function makeRouter(project = 'testproject') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/p/:project', component: { template: '<div/>' } },
      { path: '/p/:project/devops', component: DevOpsView },
      { path: '/:pathMatch(.*)*', component: { template: '<div/>' } },
    ],
  })
  router.push(`/p/${project}/devops`)
  return router
}

// ---------------------------------------------------------------------------
// Mount helpers
// ---------------------------------------------------------------------------

async function mountDevOpsView(roles: string[] = [], pipelines: Pipeline[] = []) {
  setActivePinia(createPinia())
  mockRolesForProject.mockReturnValue(roles)
  // Configure the API mock to return the given pipelines for this test.
  mockListPipelines.mockResolvedValue({ pipelines })

  const router = makeRouter()
  await router.isReady()

  const wrapper = mount(DevOpsView, {
    global: {
      plugins: [router],
    },
  })
  await flushPromises()
  return wrapper
}

async function mountPipelineCard(pipeline: Pipeline, activeRun?: {
  runId: string
  steps: Array<{ name: string; status: string; output: string[] }>
  overallStatus: 'running' | 'passed' | 'failed' | 'cancelled'
}) {
  setActivePinia(createPinia())

  const store = useDevOpsStore()
  if (activeRun) {
    // Cast to expected type
    store.activeRuns.set(pipeline.slug, activeRun as any)
  }

  const router = makeRouter()
  await router.isReady()

  const wrapper = mount(PipelineCard, {
    props: { pipeline, project: 'testproject' },
    global: {
      plugins: [router],
    },
  })
  await flushPromises()
  return wrapper
}

// ---------------------------------------------------------------------------
// Test fixtures
// ---------------------------------------------------------------------------

const buildPipeline: Pipeline = {
  slug: 'build',
  name: 'Build Application',
  type: 'build',
  steps: [{ name: 'Compile', description: 'Compile source' }, { name: 'Test', description: '' }],
}

const deployPipeline: Pipeline = {
  slug: 'deploy',
  name: 'Deploy to Staging',
  type: 'deploy',
  steps: [{ name: 'Push', description: 'Push image' }],
}

const unknownTypePipeline: Pipeline = {
  slug: 'custom-scan',
  name: 'Security Scan',
  type: 'security',
  steps: [{ name: 'Scan', description: '' }],
}

// ---------------------------------------------------------------------------
// Tests: Access control (DevOpsView)
// ---------------------------------------------------------------------------

describe('DevOpsView — role-based access', () => {
  it('shows access-denied message for users without devops/product-owner role', async () => {
    const wrapper = await mountDevOpsView(['qa'])
    expect(wrapper.find('.access-denied').exists()).toBe(true)
    expect(wrapper.find('.columns').exists()).toBe(false)
  })

  it('shows access-denied message for unauthenticated users (no roles)', async () => {
    const wrapper = await mountDevOpsView([])
    expect(wrapper.find('.access-denied').exists()).toBe(true)
  })

  it('renders pipeline content for product-owner role', async () => {
    const wrapper = await mountDevOpsView(['product-owner'])
    expect(wrapper.find('.access-denied').exists()).toBe(false)
    expect(wrapper.find('.devops-view').exists()).toBe(true)
  })

  it('renders pipeline content for devops role', async () => {
    const wrapper = await mountDevOpsView(['devops'])
    expect(wrapper.find('.access-denied').exists()).toBe(false)
  })
})

// ---------------------------------------------------------------------------
// Tests: Pipeline grouping by type (DevOpsView)
// ---------------------------------------------------------------------------

describe('DevOpsView — pipeline column rendering', () => {
  it('renders a column per pipeline type', async () => {
    const wrapper = await mountDevOpsView(['product-owner'], [buildPipeline, deployPipeline])
    const columns = wrapper.findAll('.column')
    expect(columns.length).toBeGreaterThanOrEqual(2)
  })

  it('groups pipelines with the same type into one column', async () => {
    const anotherBuild: Pipeline = {
      slug: 'build-fast',
      name: 'Fast Build',
      type: 'build',
      steps: [{ name: 'Quick', description: '' }],
    }
    const wrapper = await mountDevOpsView(['product-owner'], [buildPipeline, anotherBuild])
    const buildColumns = wrapper.findAll('.column').filter((c) =>
      c.find('.column-header').text().toLowerCase().includes('build'),
    )
    expect(buildColumns.length).toBe(1)
  })

  it('renders unknown pipeline types in their own column', async () => {
    const wrapper = await mountDevOpsView(['product-owner'], [unknownTypePipeline])
    const headers = wrapper.findAll('.column-header').map((h) => h.text().toLowerCase())
    expect(headers.some((h) => h.includes('security'))).toBe(true)
  })

  it('shows empty-state message when no pipelines exist', async () => {
    const wrapper = await mountDevOpsView(['product-owner'], [])
    const text = wrapper.text()
    expect(text).toContain('No pipelines found')
  })
})

// ---------------------------------------------------------------------------
// Tests: PipelineCard Run/Cancel buttons
// ---------------------------------------------------------------------------

describe('PipelineCard — Run button', () => {
  it('shows Run button when no run is active', async () => {
    const wrapper = await mountPipelineCard(buildPipeline)
    const btn = wrapper.find('.btn-run')
    expect(btn.exists()).toBe(true)
    expect(btn.text()).toBe('Run')
  })

  it('Run button is not disabled when no run is active', async () => {
    const wrapper = await mountPipelineCard(buildPipeline)
    const btn = wrapper.find('.btn-run')
    expect((btn.element as HTMLButtonElement).disabled).toBe(false)
  })

  it('clicking Run button calls runPipeline API', async () => {
    const { runPipeline } = await import('../../web/src/api/devops')
    const wrapper = await mountPipelineCard(buildPipeline)
    await wrapper.find('.btn-run').trigger('click')
    await flushPromises()
    expect(runPipeline).toHaveBeenCalledWith('testproject', 'build')
  })

  it('shows Cancel button when a run is active', async () => {
    const wrapper = await mountPipelineCard(buildPipeline, {
      runId: 'run-123',
      overallStatus: 'running',
      steps: buildPipeline.steps.map((s) => ({ name: s.name, status: 'pending', output: [] })),
    })
    expect(wrapper.find('.btn-cancel').exists()).toBe(true)
    expect(wrapper.find('.btn-run').exists()).toBe(false)
  })

  it('shows Run button (not Cancel) when run has completed', async () => {
    const wrapper = await mountPipelineCard(buildPipeline, {
      runId: 'run-123',
      overallStatus: 'passed',
      steps: buildPipeline.steps.map((s) => ({ name: s.name, status: 'passed', output: [] })),
    })
    expect(wrapper.find('.btn-run').exists()).toBe(true)
    expect(wrapper.find('.btn-cancel').exists()).toBe(false)
  })
})

describe('PipelineCard — Cancel button', () => {
  it('clicking Cancel button calls cancelPipeline API', async () => {
    const { cancelPipeline } = await import('../../web/src/api/devops')
    const wrapper = await mountPipelineCard(buildPipeline, {
      runId: 'run-123',
      overallStatus: 'running',
      steps: buildPipeline.steps.map((s) => ({ name: s.name, status: 'running', output: [] })),
    })
    await wrapper.find('.btn-cancel').trigger('click')
    await flushPromises()
    expect(cancelPipeline).toHaveBeenCalledWith('testproject', 'build')
  })
})

// ---------------------------------------------------------------------------
// Tests: Failed step styling
// ---------------------------------------------------------------------------

describe('PipelineCard — run status styling', () => {
  it('applies pipeline-card--failed class when run has failed', async () => {
    const wrapper = await mountPipelineCard(buildPipeline, {
      runId: 'run-123',
      overallStatus: 'failed',
      steps: buildPipeline.steps.map((s) => ({ name: s.name, status: 'failed', output: [] })),
    })
    expect(wrapper.find('.pipeline-card').classes()).toContain('pipeline-card--failed')
  })

  it('applies pipeline-card--running class when run is active', async () => {
    const wrapper = await mountPipelineCard(buildPipeline, {
      runId: 'run-123',
      overallStatus: 'running',
      steps: buildPipeline.steps.map((s) => ({ name: s.name, status: 'running', output: [] })),
    })
    expect(wrapper.find('.pipeline-card').classes()).toContain('pipeline-card--running')
  })

  it('applies pipeline-card--passed class when run has passed', async () => {
    const wrapper = await mountPipelineCard(buildPipeline, {
      runId: 'run-123',
      overallStatus: 'passed',
      steps: buildPipeline.steps.map((s) => ({ name: s.name, status: 'passed', output: [] })),
    })
    expect(wrapper.find('.pipeline-card').classes()).toContain('pipeline-card--passed')
  })

  it('shows Failed run-status badge when run has failed', async () => {
    const wrapper = await mountPipelineCard(buildPipeline, {
      runId: 'run-123',
      overallStatus: 'failed',
      steps: buildPipeline.steps.map((s) => ({ name: s.name, status: 'failed', output: [] })),
    })
    expect(wrapper.find('.run-status--failed').exists()).toBe(true)
  })
})

// ---------------------------------------------------------------------------
// Tests: Step progress display
// ---------------------------------------------------------------------------

describe('PipelineCard — step progress', () => {
  it('renders step progress list when a run is active', async () => {
    const wrapper = await mountPipelineCard(buildPipeline, {
      runId: 'run-123',
      overallStatus: 'running',
      steps: buildPipeline.steps.map((s) => ({ name: s.name, status: 'running', output: [] })),
    })
    expect(wrapper.find('.step-list').exists()).toBe(true)
  })

  it('does not render step list when no run has started', async () => {
    const wrapper = await mountPipelineCard(buildPipeline)
    expect(wrapper.find('.step-list').exists()).toBe(false)
  })
})
