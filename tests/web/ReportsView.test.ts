// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

vi.mock('@/api/reports', () => ({
  getAgentUsageReport: vi.fn(),
}))

vi.mock('@/api/agents', () => ({
  listRuns: vi.fn().mockResolvedValue({ runs: [] }),
  getRunResult: vi.fn().mockResolvedValue({ result: null }),
}))

vi.mock('@/stores/agents', () => ({
  useAgentsStore: vi.fn(() => ({
    agents: [],
    fetchAgents: vi.fn().mockResolvedValue(undefined),
  })),
}))

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: vi.fn(),
}))

// Mock echarts chart components so happy-dom doesn't need to render canvas.
vi.mock('@/components/reports/charts/RunsOverTimeChart.vue', () => ({
  default: { name: 'RunsOverTimeChart', template: '<div class="runs-over-time-chart" />' },
}))
vi.mock('@/components/reports/charts/OutputTokensPerSecChart.vue', () => ({
  default: { name: 'OutputTokensPerSecChart', template: '<div class="output-tokens-per-sec-chart" />' },
}))
vi.mock('@/components/reports/charts/TtftChart.vue', () => ({
  default: { name: 'TtftChart', template: '<div class="ttft-chart" />' },
}))
vi.mock('@/components/reports/charts/CostPerRunChart.vue', () => ({
  default: { name: 'CostPerRunChart', template: '<div class="cost-per-run-chart" />' },
}))
vi.mock('@/components/reports/charts/CostDurationScatter.vue', () => ({
  default: {
    name: 'CostDurationScatter',
    emits: ['select'],
    template: '<div class="cost-duration-scatter" />',
  },
}))

import { getAgentUsageReport } from '@/api/reports'
import ReportsView from '../../web/src/views/project/ReportsView.vue'

const mockGetReport = vi.mocked(getAgentUsageReport)

type AgentUsageGroupSummary = {
  run_count: number
  success_count: number
  failure_count: number
  metrics_unavailable_count: number
  total_cost_usd: number
  total_input_cost_usd: number
  total_output_cost_usd: number
  total_duration_ms: number
  total_input_tokens: number
  total_cache_creation_tokens: number
  total_cache_read_tokens: number
  total_output_tokens: number
  mean_duration_ms: number | null
  median_duration_ms: number | null
  p95_duration_ms: number | null
  mean_cost_usd: number | null
  mean_output_tokens_per_second: number | null
  mean_ttft_ms: number | null
  p95_ttft_ms: number | null
  cache_hit_ratio: number | null
}

function makeOverall(overrides?: Partial<AgentUsageGroupSummary>): AgentUsageGroupSummary {
  return {
    run_count: 5,
    success_count: 4,
    failure_count: 1,
    metrics_unavailable_count: 0,
    total_cost_usd: 0.50,
    total_input_cost_usd: 0.30,
    total_output_cost_usd: 0.20,
    total_duration_ms: 5000,
    total_input_tokens: 500,
    total_cache_creation_tokens: 10,
    total_cache_read_tokens: 50,
    total_output_tokens: 250,
    mean_duration_ms: 1000,
    median_duration_ms: 900,
    p95_duration_ms: 1800,
    mean_cost_usd: 0.10,
    mean_output_tokens_per_second: 50.0,
    mean_ttft_ms: 120.0,
    p95_ttft_ms: 300.0,
    cache_hit_ratio: 0.4,
    ...overrides,
  }
}

function makeReport(overrides?: {
  runCount?: number
  perAgent?: { agent_name: string }[]
  perModel?: { model: string }[]
}) {
  const runCount = overrides?.runCount ?? 5
  return {
    summary: {
      overall: makeOverall({ run_count: runCount, success_count: runCount }),
      per_model: (overrides?.perModel ?? [{ model: 'claude-opus' }]).map((m) => ({
        ...makeOverall(),
        ...m,
      })),
      per_agent: (overrides?.perAgent ?? [{ agent_name: 'qa' }]).map((a) => ({
        ...makeOverall(),
        ...a,
      })),
    },
    series: [
      {
        bucket_start: new Date().toISOString(),
        run_count: runCount,
        success_count: runCount,
        failure_count: 0,
        mean_duration_ms: null,
        mean_cost_usd: null,
        mean_output_tokens_per_second: null,
        mean_ttft_ms: null,
        cache_hit_ratio: null,
      },
    ],
    series_by_model: {},
  }
}

function makeRouter(project = 'testproject') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/p/:project/reports', component: ReportsView },
      { path: '/p/:project/agents', component: { template: '<div />' } },
      { path: '/:pathMatch(.*)*', component: { template: '<div />' } },
    ],
  })
  router.push(`/p/${project}/reports`)
  return router
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
  vi.useRealTimers()
})

afterEach(() => {
  document.body.innerHTML = ''
})

describe('ReportsView', () => {
  it('renders empty state when run_count is 0', async () => {
    mockGetReport.mockResolvedValueOnce(makeReport({ runCount: 0 }) as any)

    const router = makeRouter()
    const wrapper = mount(ReportsView, {
      global: { plugins: [router] },
    })
    await router.isReady()
    await flushPromises()

    expect(wrapper.text()).toContain('No agent runs in this window')
  })

  it('renders single-agent dataset — SummaryTiles present', async () => {
    mockGetReport.mockResolvedValue(
      makeReport({ perAgent: [{ agent_name: 'qa' }] }) as any,
    )

    const router = makeRouter()
    const wrapper = mount(ReportsView, {
      global: { plugins: [router] },
    })
    await router.isReady()
    await flushPromises()

    expect(wrapper.find('.summary-tiles').exists()).toBe(true)
  })

  it('renders multi-agent dataset — both .summary-tiles and .per-model-table present', async () => {
    mockGetReport.mockResolvedValue(
      makeReport({
        perAgent: [
          { agent_name: 'qa' },
          { agent_name: 'backend-developer' },
          { agent_name: 'frontend-developer' },
        ],
        perModel: [
          { model: 'claude-opus' },
          { model: 'claude-sonnet' },
        ],
      }) as any,
    )

    const router = makeRouter()
    const wrapper = mount(ReportsView, {
      global: { plugins: [router] },
    })
    await router.isReady()
    await flushPromises()

    expect(wrapper.find('.summary-tiles').exists()).toBe(true)
    expect(wrapper.find('.per-model-table').exists()).toBe(true)
  })

  it('renders error state on API failure — .error-banner with Retry button', async () => {
    mockGetReport.mockRejectedValueOnce(new Error('fetch failed'))

    const router = makeRouter()
    const wrapper = mount(ReportsView, {
      global: { plugins: [router] },
    })
    await router.isReady()
    await flushPromises()

    expect(wrapper.find('.error-banner').exists()).toBe(true)
    const retryBtn = wrapper.find('button.btn-retry')
    expect(retryBtn.exists()).toBe(true)
  })

  it('Retry triggers another fetch', async () => {
    mockGetReport
      .mockRejectedValueOnce(new Error('first failure'))
      .mockResolvedValueOnce(makeReport() as any)

    const router = makeRouter()
    const wrapper = mount(ReportsView, {
      global: { plugins: [router] },
    })
    await router.isReady()
    await flushPromises()

    expect(mockGetReport).toHaveBeenCalledOnce()
    expect(wrapper.find('.error-banner').exists()).toBe(true)

    // Click Retry.
    await wrapper.find('button.btn-retry').trigger('click')
    await flushPromises()

    expect(mockGetReport).toHaveBeenCalledTimes(2)
  })

  it('scatter select navigates to AgentsRunsView with run param', async () => {
    mockGetReport.mockResolvedValue(makeReport() as any)

    const router = makeRouter()

    const wrapper = mount(ReportsView, {
      global: { plugins: [router] },
    })
    // The router install-time push to '/' (memory history default) cancels the
    // makeRouter push. Re-push after isReady() so params.project is correct.
    await router.isReady()
    await router.push('/p/testproject/reports')
    await flushPromises()

    // Spy set up AFTER navigation completes so it only captures the scatter push.
    const pushSpy = vi.spyOn(router, 'push')

    // The CostDurationScatter stub is rendered inside the template v-else-if
    // block when the report is non-null and non-empty.
    const scatter = wrapper.findComponent({ name: 'CostDurationScatter' })
    expect(scatter.exists()).toBe(true)

    // Emit select event from the stub component.
    await scatter.vm.$emit('select', 'run-abc-123')
    await flushPromises()

    expect(pushSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        path: '/p/testproject/agents',
        query: { run: 'run-abc-123' },
      }),
    )
  })

  it('changing filter triggers refetch after debounce', async () => {
    vi.useFakeTimers()
    mockGetReport.mockResolvedValue(makeReport() as any)

    const router = makeRouter()
    const wrapper = mount(ReportsView, {
      global: { plugins: [router] },
    })
    await router.isReady()
    vi.advanceTimersByTime(0)
    await flushPromises()

    // Initial load call.
    const initialCalls = mockGetReport.mock.calls.length

    // Trigger setFilter via the reportsStore directly.
    const { useReportsStore } = await import('../../web/src/stores/reports')
    const store = useReportsStore()
    store.setFilter({ bucket: 'hour' }, 'testproject')

    // Before debounce — should not have been called again.
    expect(mockGetReport).toHaveBeenCalledTimes(initialCalls)

    // Advance past the 300ms debounce.
    vi.advanceTimersByTime(350)
    await flushPromises()

    expect(mockGetReport).toHaveBeenCalledTimes(initialCalls + 1)
    const lastCall = mockGetReport.mock.calls[mockGetReport.mock.calls.length - 1]
    const filter = lastCall[1] as { bucket?: string }
    expect(filter.bucket).toBe('hour')

    vi.useRealTimers()
  })
})
