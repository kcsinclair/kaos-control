// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import RunSummaryCard from '../RunSummaryCard.vue'
import type { RunResult } from '@/types/api'

// Frontend plan: lifecycle/frontend-plans/gemini-cli-stream-json-4-fe.md
// Milestone 2 — verify the run summary card for a gemini-cli (agy) run.
// The backend (plan -3-be) fills the same RunResult struct agy runs use, so
// this is verification that no summary-card change is needed, per FR-6:
// CacheCreationInputTokens and TotalCostUSD are always zero for agy.

function agyResult(overrides: Partial<RunResult> = {}): RunResult {
  return {
    subtype: 'success',
    is_error: false,
    result: 'OK\n',
    total_cost_usd: 0,
    duration_ms: 4364,
    duration_api_ms: 4364,
    num_turns: 1,
    usage: {
      input_tokens: 19425,
      cache_creation_input_tokens: 0,
      cache_read_input_tokens: 0,
      output_tokens: 25,
    },
    permission_denials: [],
    session_id: 'conv-1',
    ...overrides,
  }
}

describe('RunSummaryCard — gemini-cli (agy) RunResult', () => {
  it('renders turns, duration, zero cost, and token usage for a successful run', () => {
    const wrapper = mount(RunSummaryCard, {
      props: { result: agyResult(), driverAvailable: true },
    })

    expect(wrapper.find('.rsc-subtype-badge').text()).toBe('success')
    expect(wrapper.find('.rsc-subtype-badge').classes()).not.toContain('rsc-badge-error')
    expect(wrapper.text()).toContain('$0.0000')
    expect(wrapper.text()).toContain('4s')
    const metrics = wrapper.findAll('.rsc-metric').map((m) => m.text())
    expect(metrics.some((m) => m.includes('Turns') && m.includes('1'))).toBe(true)
    expect(wrapper.text()).toContain('19,425')
    expect(wrapper.text()).toContain('25')
    expect(wrapper.find('.rsc-error-msg').exists()).toBe(false)
  })

  it('does not NaN the cache hit ratio when cache_creation is zero', () => {
    const wrapper = mount(RunSummaryCard, {
      props: { result: agyResult(), driverAvailable: true },
    })

    // input_tokens > 0 keeps the denominator nonzero even though both cache
    // fields are 0 for agy (FR-6): ratio is 0%, never NaN/N-A.
    expect(wrapper.find('.rsc-cache-value').text()).toBe('0.0%')
    expect(wrapper.text()).not.toContain('NaN')
  })

  it('shows the error outcome and failure reason for a non-success agy run', () => {
    const wrapper = mount(RunSummaryCard, {
      props: {
        result: agyResult({
          subtype: 'error',
          is_error: true,
          result: 'error',
        }),
        driverAvailable: true,
      },
    })

    expect(wrapper.find('.rsc-subtype-badge').text()).toBe('Error')
    expect(wrapper.find('.rsc-subtype-badge').classes()).toContain('rsc-badge-error')
    expect(wrapper.find('.rsc-error-msg').text()).toBe('error')
  })
})

describe('RunSummaryCard — openai-compatible / OpenRouter RunResult', () => {
  it('asserts full metric display when all usage/cost data is provided', () => {
    const wrapper = mount(RunSummaryCard, {
      props: {
        result: {
          subtype: 'success',
          is_error: false,
          result: 'Task completed',
          total_cost_usd: 0.0356,
          cost_reported: true,
          duration_ms: 12500,
          duration_api_ms: 9800,
          num_turns: 4,
          usage: {
            input_tokens: 1500,
            cache_creation_input_tokens: 0,
            cache_read_input_tokens: 500,
            output_tokens: 300,
          },
          permission_denials: [],
          session_id: 'openrouter-sess-1',
          usage_source: 'provider_stream',
        },
        driverAvailable: true,
      },
    })

    expect(wrapper.find('.rsc-subtype-badge').text()).toBe('success')
    expect(wrapper.text()).toContain('$0.0356')
    expect(wrapper.text()).toContain('12s (API: 9s)')
    const metrics = wrapper.findAll('.rsc-metric').map((m) => m.text())
    expect(metrics.some((m) => m.includes('Turns') && m.includes('4'))).toBe(true)

    // Token table rows
    const rows = wrapper.findAll('.rsc-table tbody tr')
    expect(rows).toHaveLength(4)
    expect(wrapper.text()).toContain('1,500')
    expect(wrapper.text()).toContain('500')
    expect(wrapper.text()).toContain('300')

    // Cache hit ratio: 500 / (500 + 0 + 1500) = 25.0% -> Fair / Poor
    expect(wrapper.find('.rsc-cache-value').text()).toBe('25.0%')
    expect(wrapper.find('.rsc-quality-badge').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('Token usage not reported by this provider')
    expect(wrapper.text()).not.toContain('Token metrics not available for this driver')
  })

  it('asserts the "Token usage not reported by this provider" fallback state when usage_source: "none" is passed', () => {
    const wrapper = mount(RunSummaryCard, {
      props: {
        result: {
          subtype: 'success',
          is_error: false,
          result: 'Clean finish without usage',
          total_cost_usd: 0,
          cost_reported: false,
          duration_ms: 5000,
          duration_api_ms: 4000,
          num_turns: 2,
          usage: {
            input_tokens: 0,
            cache_creation_input_tokens: 0,
            cache_read_input_tokens: 0,
            output_tokens: 0,
          },
          permission_denials: [],
          session_id: 'openrouter-sess-2',
          usage_source: 'none',
        },
        driverAvailable: true,
      },
    })

    expect(wrapper.find('.rsc-subtype-badge').text()).toBe('success')
    expect(wrapper.text()).toContain('—') // cost should be dash
    expect(wrapper.text()).toContain('5s (API: 4s)')
    const metrics = wrapper.findAll('.rsc-metric').map((m) => m.text())
    expect(metrics.some((m) => m.includes('Turns') && m.includes('2'))).toBe(true)

    // Fallback message shown, token table & cache row omitted
    expect(wrapper.find('.rsc-usage-unreported').text()).toBe('Token usage not reported by this provider')
    expect(wrapper.find('.rsc-table').exists()).toBe(false)
    expect(wrapper.find('.rsc-cache-row').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Token metrics not available for this driver')
  })

  it('asserts the cost renders as — when cost_reported: false', () => {
    const wrapper = mount(RunSummaryCard, {
      props: {
        result: {
          subtype: 'success',
          is_error: false,
          result: 'Free or unreported cost',
          total_cost_usd: 0,
          cost_reported: false,
          duration_ms: 2000,
          duration_api_ms: 1500,
          num_turns: 1,
          usage: {
            input_tokens: 100,
            cache_creation_input_tokens: 0,
            cache_read_input_tokens: 0,
            output_tokens: 50,
          },
          permission_denials: [],
          session_id: 'openrouter-sess-3',
          usage_source: 'provider_stream',
        },
        driverAvailable: true,
      },
    })

    const costMetric = wrapper.findAll('.rsc-metric').find((m) => m.text().includes('Cost'))
    expect(costMetric?.text()).toBe('Cost —')
    expect(wrapper.text()).not.toContain('$0.0000')

    // cache_creation_input_tokens row remains visible with value of 0
    expect(wrapper.find('.rsc-table').exists()).toBe(true)
    const rows = wrapper.findAll('.rsc-table tbody tr')
    expect(rows).toHaveLength(4)
    const cacheCreationRow = rows.find((r) => r.text().includes('Cache Creation'))
    expect(cacheCreationRow?.text()).toContain('0')
  })

  it('renders legacy not available message when driver genuinely not metrics capable and result is null', () => {
    const wrapper = mount(RunSummaryCard, {
      props: {
        result: null,
        driverAvailable: false,
      },
    })

    expect(wrapper.find('.rsc-unavailable').text()).toBe('Token metrics not available for this driver.')
  })
})
