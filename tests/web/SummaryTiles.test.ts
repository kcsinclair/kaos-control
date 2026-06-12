// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

import SummaryTiles from '../../web/src/components/reports/SummaryTiles.vue'
import type { AgentUsageSummary } from '../../web/src/types/api'

function makeSummary(overrides?: Partial<AgentUsageSummary['overall']>): AgentUsageSummary {
  return {
    overall: {
      run_count: 10,
      success_count: 8,
      failure_count: 2,
      metrics_unavailable_count: 0,
      total_cost_usd: 1.50,
      total_input_cost_usd: 0.80,
      total_output_cost_usd: 0.70,
      total_duration_ms: 10000,
      total_input_tokens: 1000,
      total_cache_creation_tokens: 50,
      total_cache_read_tokens: 200,
      total_output_tokens: 500,
      mean_duration_ms: 1000,
      median_duration_ms: 900,
      p95_duration_ms: 1800,
      mean_cost_usd: 0.15,
      mean_output_tokens_per_second: 50.0,
      mean_ttft_ms: 120.0,
      p95_ttft_ms: 300.0,
      cache_hit_ratio: 0.40,
      ...overrides,
    },
    per_model: [],
    per_agent: [],
  }
}

beforeEach(() => {
  setActivePinia(createPinia())
})

describe('SummaryTiles', () => {
  it('renders 6 .dash-tile elements with correct label names', async () => {
    const wrapper = mount(SummaryTiles, {
      props: { summary: makeSummary() },
    })
    await flushPromises()

    const tiles = wrapper.findAll('.dash-tile')
    expect(tiles).toHaveLength(6)

    // Check expected label texts are present somewhere in the component.
    const text = wrapper.text()
    expect(text).toContain('Total runs')
    expect(text).toContain('Success rate')
    expect(text).toContain('Total cost')
    expect(text).toContain('Mean output tokens/s')
    expect(text).toContain('Mean TTFT')
    expect(text).toContain('Cache hit ratio')
  })

  it('success rate tile shows "—" when run_count is 0', async () => {
    const wrapper = mount(SummaryTiles, {
      props: {
        summary: makeSummary({
          run_count: 0,
          success_count: 0,
          failure_count: 0,
        }),
      },
    })
    await flushPromises()

    // The success rate tile should display '—' when there are no runs.
    const tiles = wrapper.findAll('.dash-tile')
    const successTile = tiles.find((t) => t.text().includes('Success rate'))
    expect(successTile).toBeDefined()
    expect(successTile!.find('.tile-value').text()).toBe('—')
  })

  it('success rate is formatted as percentage (8/10 runs → "80.0%")', async () => {
    const wrapper = mount(SummaryTiles, {
      props: {
        summary: makeSummary({
          run_count: 10,
          success_count: 8,
        }),
      },
    })
    await flushPromises()

    const tiles = wrapper.findAll('.dash-tile')
    const successTile = tiles.find((t) => t.text().includes('Success rate'))
    expect(successTile).toBeDefined()
    expect(successTile!.find('.tile-value').text()).toBe('80.0%')
  })
})
