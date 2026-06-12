// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { flushPromises } from '@vue/test-utils'

// Mock the reports API module.
vi.mock('@/api/reports', () => ({
  getAgentUsageReport: vi.fn(),
}))

import { useReportsStore } from '../../web/src/stores/reports'
import { getAgentUsageReport } from '@/api/reports'

const mockGetReport = vi.mocked(getAgentUsageReport)

const makeReport = () => ({
  summary: {
    overall: {
      run_count: 1,
      success_count: 1,
      failure_count: 0,
      metrics_unavailable_count: 0,
      total_cost_usd: 0.01,
      total_input_cost_usd: 0,
      total_output_cost_usd: 0,
      total_duration_ms: 1000,
      total_input_tokens: 100,
      total_cache_creation_tokens: 0,
      total_cache_read_tokens: 0,
      total_output_tokens: 50,
      mean_duration_ms: 1000,
      median_duration_ms: 1000,
      p95_duration_ms: 1000,
      mean_cost_usd: 0.01,
      mean_output_tokens_per_second: null,
      mean_ttft_ms: null,
      p95_ttft_ms: null,
      cache_hit_ratio: null,
    },
    per_model: [],
    per_agent: [],
  },
  series: [],
  series_by_model: {},
})

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
  vi.useRealTimers()
})

describe('useReportsStore', () => {
  it('fetch sets loading then clears it', async () => {
    let resolveReport!: (v: any) => void
    mockGetReport.mockReturnValueOnce(
      new Promise((res) => { resolveReport = res }),
    )

    const store = useReportsStore()
    expect(store.loading).toBe(false)

    const fetchPromise = store.fetch('testproject')
    expect(store.loading).toBe(true)

    resolveReport(makeReport())
    await fetchPromise
    await flushPromises()

    expect(store.loading).toBe(false)
    expect(store.report).not.toBeNull()
  })

  it('fetch stores error on failure', async () => {
    mockGetReport.mockRejectedValueOnce(new Error('network error'))

    const store = useReportsStore()
    await store.fetch('testproject')
    await flushPromises()

    expect(store.loading).toBe(false)
    expect(store.error).toBeTruthy()
    expect(store.error).toContain('network error')
  })

  it('setFilter debounces multiple calls', async () => {
    vi.useFakeTimers()
    mockGetReport.mockResolvedValue(makeReport())

    const store = useReportsStore()

    // Three rapid setFilter calls.
    store.setFilter({ bucket: 'hour' }, 'testproject')
    store.setFilter({ bucket: 'day' }, 'testproject')
    store.setFilter({ bucket: 'week' }, 'testproject')

    // Not yet called.
    expect(mockGetReport).not.toHaveBeenCalled()

    // Advance past the 300ms debounce.
    vi.advanceTimersByTime(350)
    await flushPromises()

    // Should be called exactly once (debounced).
    expect(mockGetReport).toHaveBeenCalledOnce()

    vi.useRealTimers()
  })

  it('reset returns defaults', async () => {
    mockGetReport.mockResolvedValue(makeReport())

    const store = useReportsStore()
    // Mutate the filter.
    store.filter.bucket = 'hour'
    store.filter.agent = ['qa', 'backend-developer']

    store.reset('testproject')
    await flushPromises()

    expect(store.filter.bucket).toBe('day')
    expect(store.filter.agent).toEqual([])
  })
})
