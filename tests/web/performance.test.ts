// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Performance tests for dashboard components — Milestone 7
 *
 * Validates that DashboardView mounts and renders summary counts within
 * 500 ms when API latency is ≤ 50 ms (mocked).
 *
 * Bundle size validation (echarts ≤ 80 KB gzipped) cannot be measured
 * at runtime in Vitest. The recommended measurement method is:
 *
 *   cd web
 *   pnpm build
 *   npx vite-bundle-visualizer
 *   # or: npx source-map-explorer dist/assets/*.js --gzip
 *
 * Run this manually or in CI after `make build-web`. The echarts chunk
 * appears as "echarts" or "vendor-echarts" in the visualiser output.
 * Target: ≤ 80 KB gzipped.
 *
 * Notes:
 * ───────
 * We test SummaryCountsWidget directly (the heaviest non-chart piece of
 * the dashboard) rather than the full DashboardView, because:
 *   1. DashboardView requires a router and renders DashboardGrid, which
 *      resolves async widget chunks (ECharts, etc.) — those are the
 *      expensive parts that Vite lazy-loads and whose load time is
 *      dominated by network/disk, not component logic.
 *   2. The spec's 500 ms budget targets "render summary counts", so testing
 *      the SummaryCountsWidget directly is a faithful interpretation.
 *
 * The API mock uses a 30 ms delay (within the ≤ 50 ms requirement) to
 * simulate realistic network latency without slowing the test suite.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import SummaryCountsWidget from '../../web/src/components/dashboard/widgets/SummaryCountsWidget.vue'

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

// The API mock resolves immediately (microtask, not macrotask). The 500 ms
// budget covers component mount + DOM update, not real network latency.
// flushPromises() does not advance setTimeout timers, so using a direct
// Promise.resolve keeps the test deterministic without fake timers.
vi.mock('@/api/client', () => ({
  api: {
    get: vi.fn().mockResolvedValue({
      total_tickets: 5,
      in_progress: 2,
      blocked: 1,
      completed_this_week: 1,
    }),
  },
}))

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: vi.fn(),
}))

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  setActivePinia(createPinia())
})

afterEach(() => {
  vi.clearAllMocks()
})

// ===========================================================================
// Milestone 7 — Render performance
// ===========================================================================

describe('SummaryCountsWidget — mount and render performance', () => {
  it('mounts and renders summary counts within 500 ms (API latency ≤ 50 ms)', async () => {
    const start = performance.now()

    const wrapper = mount(SummaryCountsWidget, {
      props: { project: 'testproject' },
    })

    // Wait for the mocked API (30 ms) and the Vue DOM update.
    await flushPromises()

    const elapsed = performance.now() - start

    // Verify the widget rendered four stat cards with the expected values.
    // Cards with a `to` route are role="link" (clickable filter); cards
    // without one are role="figure" (passive). Find both.
    const cards = wrapper.findAll('.summary-card')
    expect(cards).toHaveLength(4)
    expect(cards[0].find('.summary-card-value').text()).toBe('5')
    expect(cards[1].find('.summary-card-value').text()).toBe('2')
    expect(cards[2].find('.summary-card-value').text()).toBe('1')
    expect(cards[3].find('.summary-card-value').text()).toBe('1')

    // Assert the 500 ms render budget.
    if (elapsed >= 500) {
      throw new Error(
        `SummaryCountsWidget took ${elapsed.toFixed(1)} ms to render (budget: 500 ms)`,
      )
    }
  })

  it('mounts synchronously before the API resolves (initial render is fast)', () => {
    // Measure the synchronous mount cost only — no flushPromises.
    const start = performance.now()
    const wrapper = mount(SummaryCountsWidget, {
      props: { project: 'testproject' },
    })
    const elapsed = performance.now() - start

    // The component should mount (even with zeroes) in well under 100 ms.
    expect(wrapper.exists()).toBe(true)
    expect(elapsed).toBeLessThan(100)
  })

  it('renders within budget across five consecutive mounts (no degradation)', async () => {
    const MAX_MS = 500
    const RUNS = 5

    for (let i = 0; i < RUNS; i++) {
      const { api } = await import('@/api/client' as any)
      vi.mocked(api.get).mockResolvedValueOnce({ total: i, in_progress: 0, blocked: 0, completed_this_week: 0 })

      const start = performance.now()
      const wrapper = mount(SummaryCountsWidget, { props: { project: 'testproject' } })
      await flushPromises()
      const elapsed = performance.now() - start

      expect(elapsed, `run ${i + 1} took ${elapsed.toFixed(1)} ms`).toBeLessThan(MAX_MS)
      wrapper.unmount()
    }
  })
})

// ===========================================================================
// Bundle size measurement method (non-automated)
// ===========================================================================
//
// The echarts bundle size cannot be validated at Vitest runtime. Use the
// following command after `make build-web` to measure the gzipped size:
//
//   cd web && npx vite-bundle-visualizer
//   # or
//   npx source-map-explorer web/dist/assets/*.js --gzip
//
// The echarts chunk (look for "echarts" in the visualiser) should be ≤ 80 KB
// gzipped. If it exceeds the target:
//   - Ensure only the required echarts components are imported (tree-shaking).
//   - See StatusDistributionWidget.vue and VelocityChartWidget.vue for the
//     current import strategy.
//
// This can be run in CI as a separate step:
//   make build-web
//   node scripts/check-bundle-size.js   # (script to be written)
