// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Component tests for RunHistory.vue and PipelineCard.vue (Milestone 5 — F4, F5, F7)
 *
 * Covers:
 *   F4 — Run history panel: rows, timestamps, duration, status colour/icon
 *   F5 — Inline log expansion: single-expand, log lines, error state
 *   F7 — Latest-run summary badge on PipelineCard (reflects pipelineHistory[slug][0])
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import RunHistory from '../../web/src/components/devops/RunHistory.vue'
import PipelineCard from '../../web/src/components/devops/PipelineCard.vue'
import { useDevOpsStore } from '../../web/src/stores/devops'
import type { RunHistoryRow, Pipeline } from '../../web/src/api/devops'

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

const mockListPipelineRuns = vi.fn()
const mockGetPipelineRunLog = vi.fn()
const mockParseRunLog = vi.fn()
const mockRunPipeline = vi.fn()
const mockCancelPipeline = vi.fn()
const mockListPipelines = vi.fn().mockResolvedValue({ pipelines: [] })

vi.mock('@/api/devops', () => ({
  listPipelines: (...args: unknown[]) => mockListPipelines(...args),
  listPipelineRuns: (...args: unknown[]) => mockListPipelineRuns(...args),
  getPipelineRunLog: (...args: unknown[]) => mockGetPipelineRunLog(...args),
  parseRunLog: (...args: unknown[]) => mockParseRunLog(...args),
  runPipeline: (...args: unknown[]) => mockRunPipeline(...args),
  cancelPipeline: (...args: unknown[]) => mockCancelPipeline(...args),
  getRunLog: vi.fn().mockResolvedValue(''),
  createPipeline: vi.fn(),
  updatePipeline: vi.fn(),
  getPipelineDefinition: vi.fn(),
}))

vi.mock('@/api/ws', () => ({
  getProjectWs: vi.fn(() => ({ onType: vi.fn(() => () => {}) })),
}))

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: vi.fn(),
}))

vi.mock('@/composables/useNow', () => ({
  useNow: vi.fn(() => ({ value: Date.now() })),
}))

vi.mock('@/composables/useRunFormatters', () => ({
  formatRelativeTime: vi.fn((iso: string) => `~${iso}`),
  formatDurationMs: vi.fn((ms: number) => `${ms}ms`),
}))

vi.mock('@/stores/ui', () => ({
  useUiStore: vi.fn(() => ({ error: vi.fn() })),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: vi.fn(() => ({
    rolesForProject: vi.fn(() => ['product-owner']),
  })),
}))

vi.mock('@/stores/project', () => ({
  useProjectStore: vi.fn(() => ({ current: { name: 'testproject' } })),
}))

// Stub child components used by PipelineCard so we don't need their deps.
vi.mock('@/components/devops/StepProgress.vue', () => ({
  default: { template: '<div class="stub-step-progress" />' },
}))
vi.mock('@/components/devops/StepOutput.vue', () => ({
  default: { template: '<div class="stub-step-output" />' },
}))

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

const PASSED_ROW: RunHistoryRow = {
  run_id: 'aabbccdd11223344',
  status: 'passed',
  started_at: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
  ended_at: new Date(Date.now() - 4 * 60 * 1000).toISOString(),
  duration_ms: 60000,
}

const FAILED_ROW: RunHistoryRow = {
  run_id: 'ddccbbaa44332211',
  status: 'failed',
  started_at: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
  ended_at: new Date(Date.now() - 29 * 60 * 1000).toISOString(),
  duration_ms: 58000,
}

const TEST_PIPELINE: Pipeline = {
  slug: 'test-pipe',
  name: 'Test Pipeline',
  type: 'build',
  steps: [{ name: 'Build', description: 'Build step' }],
}

// ---------------------------------------------------------------------------
// Router (required by components that use useRoute)
// ---------------------------------------------------------------------------

function makeRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div />' } }],
  })
}

// ---------------------------------------------------------------------------
// RunHistory component tests (F4, F5)
// ---------------------------------------------------------------------------

describe('RunHistory.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    mockListPipelineRuns.mockResolvedValue({ runs: [] })
    mockParseRunLog.mockReturnValue([])
  })

  // F4 — renders rows newest-first with timestamp, duration, and status icon
  it('renders history rows with status, timestamp, and duration', async () => {
    mockListPipelineRuns.mockResolvedValue({ runs: [PASSED_ROW, FAILED_ROW] })

    const wrapper = mount(RunHistory, {
      props: { pipelineSlug: 'test-pipe', project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })

    await flushPromises()
    // Expand the history panel (default collapsed)
    await wrapper.find('.history-toggle').trigger('click')
    await flushPromises()

    const rows = wrapper.findAll('.history-row')
    expect(rows).toHaveLength(2)

    // First row = newest (passed)
    const firstRow = rows[0]
    expect(firstRow.find('.history-status--passed').exists()).toBe(true)
    expect(firstRow.find('.history-duration').text()).toContain('60000ms')
  })

  // F4 — failure row has the red icon/class
  it('marks failure rows with the failed status class', async () => {
    mockListPipelineRuns.mockResolvedValue({ runs: [PASSED_ROW, FAILED_ROW] })

    const wrapper = mount(RunHistory, {
      props: { pipelineSlug: 'test-pipe', project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()
    await wrapper.find('.history-toggle').trigger('click')
    await flushPromises()

    const rows = wrapper.findAll('.history-row')
    expect(rows[1].find('.history-status--failed').exists()).toBe(true)
  })

  // F4 — empty history shows "No runs yet"
  it('shows "No runs yet" when there are no history rows', async () => {
    mockListPipelineRuns.mockResolvedValue({ runs: [] })

    const wrapper = mount(RunHistory, {
      props: { pipelineSlug: 'test-pipe', project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()
    await wrapper.find('.history-toggle').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('No runs yet')
  })

  // F4 — panel collapses and expands
  it('toggles collapse/expand via the history-toggle button', async () => {
    mockListPipelineRuns.mockResolvedValue({ runs: [PASSED_ROW] })

    const wrapper = mount(RunHistory, {
      props: { pipelineSlug: 'test-pipe', project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    // Initially collapsed — rows not visible
    expect(wrapper.find('.history-row').exists()).toBe(false)

    // Expand
    await wrapper.find('.history-toggle').trigger('click')
    await flushPromises()
    expect(wrapper.find('.history-row').exists()).toBe(true)

    // Collapse again
    await wrapper.find('.history-toggle').trigger('click')
    await flushPromises()
    expect(wrapper.find('.history-row').exists()).toBe(false)
  })

  // F5 — expanding a row calls getPipelineRunLog and renders parsed log lines
  it('fetches and displays log lines when a row is expanded', async () => {
    mockListPipelineRuns.mockResolvedValue({ runs: [PASSED_ROW] })
    mockGetPipelineRunLog.mockResolvedValue('{"type":"pipeline.run.started"}\n')
    mockParseRunLog.mockReturnValue([
      { kind: 'run-start', timestamp: Date.now(), text: 'Run started' },
    ])

    const wrapper = mount(RunHistory, {
      props: { pipelineSlug: 'test-pipe', project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()
    await wrapper.find('.history-toggle').trigger('click')
    await flushPromises()

    // Click the expand button on the first row
    await wrapper.find('.history-expand-btn').trigger('click')
    await flushPromises()

    expect(mockGetPipelineRunLog).toHaveBeenCalledWith(
      'testproject',
      'test-pipe',
      PASSED_ROW.run_id,
    )
    // Log pane should be visible with a log row
    expect(wrapper.find('.history-log-pane').exists()).toBe(true)
    expect(wrapper.find('.log-scroll').exists()).toBe(true)
  })

  // F5 — a second expand collapses the first (single-expand behaviour)
  it('collapses expanded row when the same row is expanded again', async () => {
    mockListPipelineRuns.mockResolvedValue({ runs: [PASSED_ROW, FAILED_ROW] })
    mockGetPipelineRunLog.mockResolvedValue('')
    mockParseRunLog.mockReturnValue([])

    const wrapper = mount(RunHistory, {
      props: { pipelineSlug: 'test-pipe', project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()
    await wrapper.find('.history-toggle').trigger('click')
    await flushPromises()

    const expandBtns = wrapper.findAll('.history-expand-btn')

    // Expand first row
    await expandBtns[0].trigger('click')
    await flushPromises()
    expect(wrapper.findAll('.history-log-pane')).toHaveLength(1)

    // Expand second row — first must collapse
    await expandBtns[1].trigger('click')
    await flushPromises()
    expect(wrapper.findAll('.history-log-pane')).toHaveLength(1)
    // Log pane follows the second row, not the first
    const rows = wrapper.findAll('.history-row')
    // The expanded row should be the second one
    expect(rows[1].classes()).toContain('history-row--expanded')
    expect(rows[0].classes()).not.toContain('history-row--expanded')
  })

  // F5 — rejected getPipelineRunLog shows inline error state
  it('shows an inline error state when log fetch fails', async () => {
    mockListPipelineRuns.mockResolvedValue({ runs: [PASSED_ROW] })
    mockGetPipelineRunLog.mockRejectedValue(new Error('Network error'))

    const wrapper = mount(RunHistory, {
      props: { pipelineSlug: 'test-pipe', project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()
    await wrapper.find('.history-toggle').trigger('click')
    await flushPromises()

    await wrapper.find('.history-expand-btn').trigger('click')
    await flushPromises()

    const logPane = wrapper.find('.history-log-pane')
    expect(logPane.exists()).toBe(true)
    // Error state must be visible (not blank)
    expect(logPane.find('.log-state--error').exists()).toBe(true)
    expect(logPane.find('.log-state--error').text()).toContain('Network error')
  })
})

// ---------------------------------------------------------------------------
// PipelineCard / F7 — latest-run summary badge
// ---------------------------------------------------------------------------

describe('PipelineCard.vue (F7 — latest-run summary badge)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    mockListPipelineRuns.mockResolvedValue({ runs: [PASSED_ROW] })
    mockListPipelines.mockResolvedValue({ pipelines: [TEST_PIPELINE] })
  })

  it('shows the latest-run badge when pipelineHistory has entries', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const devops = useDevOpsStore()

    // Pre-populate the store with history so the badge appears.
    devops.pipelineHistory.set('test-pipe', [PASSED_ROW, FAILED_ROW])

    const wrapper = mount(PipelineCard, {
      props: { pipeline: TEST_PIPELINE, project: 'testproject' },
      global: { plugins: [makeRouter(), pinia] },
    })
    await flushPromises()

    // The badge must reflect pipelineHistory["test-pipe"][0] = PASSED_ROW.
    const badge = wrapper.find('.latest-run-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.classes()).toContain('latest-run-badge--passed')
  })

  it('shows failed badge class when the latest run failed', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    // Latest run is FAILED_ROW (index 0 = newest) — drive via the mock so
    // RunHistory.vue's onMounted fetchPipelineHistory populates the store.
    mockListPipelineRuns.mockResolvedValue({ runs: [FAILED_ROW, PASSED_ROW] })

    const wrapper = mount(PipelineCard, {
      props: { pipeline: TEST_PIPELINE, project: 'testproject' },
      global: { plugins: [makeRouter(), pinia] },
    })
    await flushPromises()

    const badge = wrapper.find('.latest-run-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.classes()).toContain('latest-run-badge--failed')
  })

  it('hides the latest-run badge when there is no history', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    // Empty list → fetchPipelineHistory populates history with [] → no badge.
    mockListPipelineRuns.mockResolvedValue({ runs: [] })

    const wrapper = mount(PipelineCard, {
      props: { pipeline: TEST_PIPELINE, project: 'testproject' },
      global: { plugins: [makeRouter(), pinia] },
    })
    await flushPromises()

    expect(wrapper.find('.latest-run-badge').exists()).toBe(false)
  })
})
