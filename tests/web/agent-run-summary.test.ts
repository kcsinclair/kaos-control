// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 7 — Test Runner Run Summary Tests
 *
 * Verifies the TestRunSummaryCard component renders the test-runner run
 * summary correctly, including:
 *   - Per-suite statistics table (Go, Vitest, Playwright)
 *   - Defects created / duplicates found / orphaned failures counts
 *   - Collapsible coverage gaps section
 *   - Duration formatting
 *   - Empty suites state
 *
 * Component: web/src/components/agent/TestRunSummaryCard.vue
 * Props: summary (RunSummary)
 * Run with: pnpm --prefix tests/web test agent-run-summary
 */

import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import TestRunSummaryCard from '../../web/src/components/agent/TestRunSummaryCard.vue'
import type { RunSummary, RunSuiteSummary } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

function makeSuite(overrides: Partial<RunSuiteSummary> = {}): RunSuiteSummary {
  return {
    name: 'go',
    total: 10,
    passed: 8,
    failed: 1,
    skipped: 1,
    elapsed: 1500,
    ...overrides,
  }
}

function makeSummary(overrides: Partial<RunSummary> = {}): RunSummary {
  return {
    suites: [makeSuite()],
    defectsCreated: 1,
    duplicatesFound: 0,
    orphanedFailures: 0,
    coverageGaps: [],
    elapsed: 3200,
    ...overrides,
  }
}

function mountCard(summary: RunSummary) {
  return mount(TestRunSummaryCard, { props: { summary } })
}

// ---------------------------------------------------------------------------
// Per-suite statistics table
// ---------------------------------------------------------------------------

describe('TestRunSummaryCard — suite statistics table', () => {
  it('renders a row for each suite', () => {
    const summary = makeSummary({
      suites: [
        makeSuite({ name: 'go', total: 5, passed: 4, failed: 1, skipped: 0, elapsed: 900 }),
        makeSuite({ name: 'vitest', total: 3, passed: 3, failed: 0, skipped: 0, elapsed: 400 }),
      ],
    })
    const wrapper = mountCard(summary)
    const rows = wrapper.findAll('tbody tr')
    expect(rows).toHaveLength(2)
  })

  it('shows suite name, total, passed, failed, and skipped counts', () => {
    const summary = makeSummary({
      suites: [makeSuite({ name: 'go', total: 10, passed: 8, failed: 1, skipped: 1 })],
    })
    const wrapper = mountCard(summary)
    const row = wrapper.find('tbody tr')

    expect(row.text()).toContain('go')
    expect(row.text()).toContain('10')
    expect(row.text()).toContain('8')
    expect(row.text()).toContain('1')
  })

  it('renders empty-state message when suites array is empty', () => {
    const wrapper = mountCard(makeSummary({ suites: [] }))

    expect(wrapper.find('table').exists()).toBe(false)
    expect(wrapper.find('.trs-empty').exists()).toBe(true)
  })
})

// ---------------------------------------------------------------------------
// Summary line: defect / duplicate / orphan counts
// ---------------------------------------------------------------------------

describe('TestRunSummaryCard — summary line counts', () => {
  it('displays defects created count', () => {
    const wrapper = mountCard(makeSummary({ defectsCreated: 3 }))
    expect(wrapper.find('.trs-summary-line').text()).toContain('3')
    expect(wrapper.find('.trs-summary-line').text()).toMatch(/defect/i)
  })

  it('displays duplicates found count', () => {
    const wrapper = mountCard(makeSummary({ duplicatesFound: 2 }))
    expect(wrapper.find('.trs-summary-line').text()).toContain('2')
    expect(wrapper.find('.trs-summary-line').text()).toMatch(/duplicate/i)
  })

  it('displays orphaned failures count', () => {
    const wrapper = mountCard(makeSummary({ orphanedFailures: 4 }))
    expect(wrapper.find('.trs-summary-line').text()).toContain('4')
    expect(wrapper.find('.trs-summary-line').text()).toMatch(/orphaned/i)
  })

  it('applies warn style when defects are created', () => {
    const wrapper = mountCard(makeSummary({ defectsCreated: 2 }))
    const stats = wrapper.findAll('.trs-stat--warn')
    expect(stats.length).toBeGreaterThan(0)
  })

  it('applies warn style when orphaned failures exist', () => {
    const wrapper = mountCard(makeSummary({ orphanedFailures: 1 }))
    const stats = wrapper.findAll('.trs-stat--warn')
    expect(stats.length).toBeGreaterThan(0)
  })

  it('uses singular "defect" for exactly one defect', () => {
    const wrapper = mountCard(makeSummary({ defectsCreated: 1 }))
    expect(wrapper.find('.trs-summary-line').text()).toContain('1 defect ')
  })

  it('uses plural "defects" for multiple defects', () => {
    const wrapper = mountCard(makeSummary({ defectsCreated: 3 }))
    expect(wrapper.find('.trs-summary-line').text()).toContain('3 defects')
  })
})

// ---------------------------------------------------------------------------
// Duration formatting
// ---------------------------------------------------------------------------

describe('TestRunSummaryCard — duration formatting', () => {
  it('formats milliseconds as ms when under 1 second', () => {
    const wrapper = mountCard(
      makeSummary({ suites: [makeSuite({ elapsed: 250 })], elapsed: 250 }),
    )
    expect(wrapper.text()).toContain('250ms')
  })

  it('formats seconds as X.Xs when between 1s and 60s', () => {
    const wrapper = mountCard(
      makeSummary({ suites: [makeSuite({ elapsed: 4500 })], elapsed: 4500 }),
    )
    expect(wrapper.text()).toContain('4.5s')
  })

  it('formats minutes as Xm Ys for durations over 60s', () => {
    const wrapper = mountCard(
      makeSummary({ elapsed: 90000 }),
    )
    expect(wrapper.text()).toMatch(/1m \d+s/)
  })
})

// ---------------------------------------------------------------------------
// Coverage gaps (collapsible section)
// ---------------------------------------------------------------------------

describe('TestRunSummaryCard — coverage gaps', () => {
  it('does not render the gaps section when coverageGaps is empty', () => {
    const wrapper = mountCard(makeSummary({ coverageGaps: [] }))
    expect(wrapper.find('.trs-gaps').exists()).toBe(false)
  })

  it('renders the gaps toggle button when gaps exist', () => {
    const wrapper = mountCard(
      makeSummary({ coverageGaps: ['lifecycle/tests/login-2-test.md'] }),
    )
    expect(wrapper.find('.trs-gaps-toggle').exists()).toBe(true)
  })

  it('shows gap count in the toggle label', () => {
    const wrapper = mountCard(
      makeSummary({
        coverageGaps: ['lifecycle/tests/login-2-test.md', 'lifecycle/tests/signup-2-test.md'],
      }),
    )
    expect(wrapper.find('.trs-gaps-toggle').text()).toContain('2')
  })

  it('gap list is hidden initially (collapsed)', () => {
    const wrapper = mountCard(
      makeSummary({ coverageGaps: ['lifecycle/tests/login-2-test.md'] }),
    )
    expect(wrapper.find('.trs-gaps-list').exists()).toBe(false)
  })

  it('gap list is shown after clicking the toggle', async () => {
    const wrapper = mountCard(
      makeSummary({ coverageGaps: ['lifecycle/tests/login-2-test.md'] }),
    )
    await wrapper.find('.trs-gaps-toggle').trigger('click')
    expect(wrapper.find('.trs-gaps-list').exists()).toBe(true)
    expect(wrapper.find('.trs-gaps-list').text()).toContain('lifecycle/tests/login-2-test.md')
  })

  it('gap list is hidden again after second click (toggle off)', async () => {
    const wrapper = mountCard(
      makeSummary({ coverageGaps: ['lifecycle/tests/login-2-test.md'] }),
    )
    await wrapper.find('.trs-gaps-toggle').trigger('click')
    await wrapper.find('.trs-gaps-toggle').trigger('click')
    expect(wrapper.find('.trs-gaps-list').exists()).toBe(false)
  })
})
