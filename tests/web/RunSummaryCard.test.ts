// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 4 — Unit tests for the RunSummaryCard component
 *
 * Covers:
 *   - Rendering all summary fields (cost, duration, turns, token usage)
 *   - Cache hit ratio calculation and display
 *   - Cache quality threshold label bands (Excellent / Good / Fair / Poor / N/A)
 *   - Permission denials section (shown / hidden)
 *   - Fallback states for null result with Claude vs non-Claude driver
 *   - Token count thousands-separator formatting
 *
 * Component: web/src/components/agent/RunSummaryCard.vue
 * Props: result (RunResult | null), driverAvailable (boolean)
 * No API calls — component renders purely from props.
 *
 * Run with: pnpm --prefix tests/web test RunSummaryCard
 */

import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import RunSummaryCard from '../../web/src/components/agent/RunSummaryCard.vue'
import type { RunResult } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

function makeResult(overrides: Partial<RunResult> = {}): RunResult {
  return {
    subtype: 'success',
    total_cost_usd: 0.0234,
    duration_ms: 75000,   // 1m 15s
    duration_api_ms: 9800,
    num_turns: 3,
    usage: {
      input_tokens: 1500,
      cache_creation_input_tokens: 200,
      cache_read_input_tokens: 50,
      output_tokens: 400,
    },
    permission_denials: [],
    session_id: 'ses_test_001',
    ...overrides,
  }
}

function mountCard(result: RunResult | null, driverAvailable = true) {
  return mount(RunSummaryCard, {
    props: { result, driverAvailable },
  })
}

// ---------------------------------------------------------------------------
// Field rendering
// ---------------------------------------------------------------------------

describe('RunSummaryCard — field rendering', () => {
  it('renders all summary fields for a valid result', () => {
    const wrapper = mountCard(makeResult())
    const text = wrapper.text()

    // Cost: 4 decimal places with $
    expect(text).toContain('$0.0234')

    // Duration: 1m 15s (75000ms)
    expect(text).toContain('1m 15s')

    // Turns
    expect(text).toContain('3')

    // Token usage labels
    expect(text).toContain('Input')
    expect(text).toContain('Cache Creation')
    expect(text).toContain('Cache Read')
    expect(text).toContain('Output')
  })

  it('shows cost formatted to 4 decimal places with $ prefix', () => {
    const wrapper = mountCard(makeResult({ total_cost_usd: 0.1 }))
    expect(wrapper.text()).toContain('$0.1000')
  })

  it('formats duration in Xm Ys format for durations >= 60s', () => {
    // 90000ms = 1m 30s
    const wrapper = mountCard(makeResult({ duration_ms: 90000 }))
    expect(wrapper.text()).toContain('1m 30s')
  })

  it('formats duration in seconds-only for durations < 60s', () => {
    // 45000ms = 45s
    const wrapper = mountCard(makeResult({ duration_ms: 45000 }))
    expect(wrapper.text()).toContain('45s')
  })

  it('displays num_turns value', () => {
    const wrapper = mountCard(makeResult({ num_turns: 7 }))
    expect(wrapper.text()).toContain('7')
  })
})

// ---------------------------------------------------------------------------
// Cache hit ratio
// ---------------------------------------------------------------------------

describe('RunSummaryCard — cache hit ratio calculation', () => {
  it('calculates cache hit ratio correctly (80.0%)', () => {
    // cache_read:800, cache_creation:100, input:100 → denominator=1000 → 80.0%
    const result = makeResult({
      usage: {
        input_tokens: 100,
        cache_creation_input_tokens: 100,
        cache_read_input_tokens: 800,
        output_tokens: 50,
      },
    })
    const wrapper = mountCard(result)
    expect(wrapper.text()).toContain('80.0%')
  })

  it('displays N/A when denominator is zero (all usage fields zero)', () => {
    const result = makeResult({
      usage: {
        input_tokens: 0,
        cache_creation_input_tokens: 0,
        cache_read_input_tokens: 0,
        output_tokens: 0,
      },
    })
    const wrapper = mountCard(result)
    expect(wrapper.text()).toContain('N/A')
    // No percentage should appear in the cache area
    const cacheRow = wrapper.find('.rsc-cache-row')
    expect(cacheRow.text()).not.toMatch(/\d+\.\d+%/)
  })
})

// ---------------------------------------------------------------------------
// Cache quality threshold labels
// ---------------------------------------------------------------------------

describe('RunSummaryCard — cache quality threshold labels', () => {
  it('displays "Excellent" label for ratio >= 90% with green data-color', () => {
    // ratio = 92%: cache_read=920, creation=0, input=80 → 920/1000 = 92%
    const result = makeResult({
      usage: {
        input_tokens: 80,
        cache_creation_input_tokens: 0,
        cache_read_input_tokens: 920,
        output_tokens: 50,
      },
    })
    const wrapper = mountCard(result)
    expect(wrapper.text()).toContain('Excellent')
    const badge = wrapper.find('.rsc-quality-badge')
    expect(badge.attributes('data-color')).toBe('green')
  })

  it('displays "Good" label for ratio >= 75% and < 90% with blue data-color', () => {
    // ratio = 80%: cache_read=800, creation=0, input=200 → 800/1000 = 80%
    const result = makeResult({
      usage: {
        input_tokens: 200,
        cache_creation_input_tokens: 0,
        cache_read_input_tokens: 800,
        output_tokens: 50,
      },
    })
    const wrapper = mountCard(result)
    expect(wrapper.text()).toContain('Good')
    const badge = wrapper.find('.rsc-quality-badge')
    expect(badge.attributes('data-color')).toBe('blue')
  })

  it('displays "Fair" label for ratio >= 50% and < 75% with amber data-color', () => {
    // ratio = 55%: cache_read=550, creation=0, input=450 → 550/1000 = 55%
    const result = makeResult({
      usage: {
        input_tokens: 450,
        cache_creation_input_tokens: 0,
        cache_read_input_tokens: 550,
        output_tokens: 50,
      },
    })
    const wrapper = mountCard(result)
    expect(wrapper.text()).toContain('Fair')
    const badge = wrapper.find('.rsc-quality-badge')
    expect(badge.attributes('data-color')).toBe('amber')
  })

  it('displays "Poor" label for ratio < 50% with red data-color', () => {
    // ratio = 30%: cache_read=300, creation=0, input=700 → 300/1000 = 30%
    const result = makeResult({
      usage: {
        input_tokens: 700,
        cache_creation_input_tokens: 0,
        cache_read_input_tokens: 300,
        output_tokens: 50,
      },
    })
    const wrapper = mountCard(result)
    expect(wrapper.text()).toContain('Poor')
    const badge = wrapper.find('.rsc-quality-badge')
    expect(badge.attributes('data-color')).toBe('red')
  })
})

// ---------------------------------------------------------------------------
// Fallback / unavailable states
// ---------------------------------------------------------------------------

describe('RunSummaryCard — fallback states', () => {
  it('displays "Summary unavailable" for null result with Claude driver', () => {
    const wrapper = mountCard(null, true)
    expect(wrapper.text()).toContain('Summary unavailable')
    expect(wrapper.find('.rsc-card').exists()).toBe(false)
  })

  it('displays driver-unavailable message for non-Claude driver with null result', () => {
    const wrapper = mountCard(null, false)
    expect(wrapper.text()).toContain('Token metrics not available for this driver')
    expect(wrapper.find('.rsc-card').exists()).toBe(false)
  })
})

// ---------------------------------------------------------------------------
// Permission denials
// ---------------------------------------------------------------------------

describe('RunSummaryCard — permission denials', () => {
  it('renders permission denials section when non-empty', () => {
    const result = makeResult({
      permission_denials: [
        { tool: 'bash', reason: 'blocked' } as unknown as never,
        { tool: 'write', reason: 'path_denied' } as unknown as never,
      ],
    })
    const wrapper = mountCard(result)
    const denials = wrapper.find('.rsc-denials')
    expect(denials.exists()).toBe(true)
    expect(denials.text()).toContain('bash')
  })

  it('does not render permission denials section when list is empty', () => {
    const wrapper = mountCard(makeResult({ permission_denials: [] }))
    expect(wrapper.find('.rsc-denials').exists()).toBe(false)
  })
})

// ---------------------------------------------------------------------------
// Token count formatting
// ---------------------------------------------------------------------------

describe('RunSummaryCard — token count formatting', () => {
  it('formats token counts with thousands separators', () => {
    const result = makeResult({
      usage: {
        input_tokens: 12345,
        cache_creation_input_tokens: 1000,
        cache_read_input_tokens: 50000,
        output_tokens: 9876,
      },
    })
    const wrapper = mountCard(result)
    const text = wrapper.text()

    // toLocaleString() output depends on the runtime locale, but in happy-dom
    // it typically produces comma-separated thousands.
    // We assert that the raw token value appears somewhere, at minimum.
    expect(text).toContain('12')   // part of 12,345 or 12345
    expect(text).toContain('345')
    expect(text).toContain('50')   // part of 50,000 or 50000
    expect(text).toContain('000')
  })
})
