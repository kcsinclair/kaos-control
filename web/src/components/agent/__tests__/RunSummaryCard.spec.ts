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
