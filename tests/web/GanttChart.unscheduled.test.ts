// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Tests for GanttChart — Unscheduled Column and Release Bars
 *
 * Covers Milestone 2 (FR1.1, FR1.4, FR1.5, OQ3) and Milestone 3
 * (FR2.1–FR2.5) from the Roadmap Backlog Panel and Unscheduled Column
 * test plan.
 *
 * Testing approach:
 * ─────────────────
 * happy-dom does not compute layout or apply CSS variables, so:
 *   - "sticky" behaviour is verified by checking the CSS class
 *     col-header--unscheduled (which carries `position: sticky` in the
 *     component's <style scoped>), not by measuring computed styles.
 *   - Visual distinction of unscheduled bars is checked by asserting the
 *     release-bar--unscheduled class (which carries the hatched overlay
 *     background-image in production CSS).
 *   - The dashed left border of the Unscheduled column header is verified
 *     by asserting the col-header--unscheduled class.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import GanttChart from '../../web/src/components/releases/GanttChart.vue'
import type { Release, ReleaseDetail } from '../../web/src/types/release'

// ---------------------------------------------------------------------------
// Fixture factory helpers
// ---------------------------------------------------------------------------

function makeScheduledRelease(overrides: Partial<Release> = {}): Release {
  return {
    id: 1,
    name: 'v-scheduled',
    status: 'planned',
    start_date: '2026-01-01',
    end_date: '2026-03-31',
    created_at: '2025-01-01T00:00:00Z',
    updated_at: '2025-01-01T00:00:00Z',
    ...overrides,
  }
}

function makeUnscheduledRelease(overrides: Partial<Release> = {}): Release {
  return {
    id: 2,
    name: 'v-unscheduled',
    status: 'planned',
    start_date: null,
    end_date: null,
    created_at: '2025-01-01T00:00:00Z',
    updated_at: '2025-01-01T00:00:00Z',
    ...overrides,
  }
}

function mountGantt(releases: Release[], releaseDetails?: Map<number, ReleaseDetail>) {
  return mount(GanttChart, {
    props: {
      releases,
      granularity: 'month',
      project: 'test-project',
      releaseDetails,
    },
  })
}

// ---------------------------------------------------------------------------
// Milestone 2 — Unscheduled Column Tests
// ---------------------------------------------------------------------------

describe('GanttChart — Unscheduled Column (Milestone 2)', () => {
  it('FR1.1: renders Unscheduled column header when unscheduled releases exist', () => {
    const releases = [
      makeScheduledRelease({ id: 1 }),
      makeUnscheduledRelease({ id: 2 }),
    ]
    const wrapper = mountGantt(releases)

    const header = wrapper.find('.col-header--unscheduled')
    expect(header.exists()).toBe(true)
    expect(header.text()).toBe('Unscheduled')
  })

  it('FR1.5: Unscheduled column header is absent when all releases are scheduled', () => {
    const releases = [
      makeScheduledRelease({ id: 1, name: 'v1', start_date: '2026-01-01', end_date: '2026-03-31' }),
      makeScheduledRelease({ id: 2, name: 'v2', start_date: '2026-04-01', end_date: '2026-06-30' }),
    ]
    const wrapper = mountGantt(releases)

    const header = wrapper.find('.col-header--unscheduled')
    expect(header.exists()).toBe(false)
  })

  it('FR1.4: Unscheduled column header has the col-header--unscheduled class (carries dashed left border)', () => {
    const releases = [makeUnscheduledRelease({ id: 1 })]
    const wrapper = mountGantt(releases)

    const header = wrapper.find('.col-header--unscheduled')
    expect(header.exists()).toBe(true)
    // The col-header--unscheduled class carries `border-left: 2px dashed` in
    // the component's scoped CSS.  We assert the class is present; layout-level
    // border checks require a real browser.
    expect(header.classes()).toContain('col-header--unscheduled')
  })

  it('OQ3: Unscheduled column header has position:sticky class (col-header--unscheduled)', () => {
    const releases = [makeUnscheduledRelease({ id: 1 })]
    const wrapper = mountGantt(releases)

    // col-header--unscheduled carries `position: sticky; right: 0` in scoped CSS.
    const header = wrapper.find('.col-header--unscheduled')
    expect(header.exists()).toBe(true)
    expect(header.classes()).toContain('col-header--unscheduled')
  })

  it('FR1.1: Unscheduled column header has no date label', () => {
    const releases = [makeUnscheduledRelease({ id: 1 })]
    const wrapper = mountGantt(releases)

    const header = wrapper.find('.col-header--unscheduled')
    expect(header.exists()).toBe(true)
    // Header text should just be "Unscheduled" — no date string.
    expect(header.text()).toBe('Unscheduled')
    // Should not contain any digit sequences that would indicate a date.
    expect(/\d{4}/.test(header.text())).toBe(false)
  })

  it('renders empty state when releases array is empty (no columns)', () => {
    const wrapper = mountGantt([])
    expect(wrapper.find('.empty-state').exists()).toBe(true)
    expect(wrapper.find('.col-header--unscheduled').exists()).toBe(false)
  })
})

// ---------------------------------------------------------------------------
// Milestone 3 — Unscheduled Release Bar Tests
// ---------------------------------------------------------------------------

describe('GanttChart — Unscheduled Release Bars (Milestone 3)', () => {
  it('FR2.1: each unscheduled release renders as a row with a bar', () => {
    const releases = [
      makeUnscheduledRelease({ id: 1, name: 'v-unsched-a' }),
      makeUnscheduledRelease({ id: 2, name: 'v-unsched-b' }),
    ]
    const wrapper = mountGantt(releases)

    const bars = wrapper.findAll('.release-bar--unscheduled')
    expect(bars).toHaveLength(2)
  })

  it('FR2.2: unscheduled bars have release-bar--unscheduled class (visually distinct)', () => {
    const releases = [
      makeScheduledRelease({ id: 1, name: 'v-sched' }),
      makeUnscheduledRelease({ id: 2, name: 'v-unsched' }),
    ]
    const wrapper = mountGantt(releases)

    // Scheduled bar exists but does NOT have the --unscheduled modifier class.
    const allBars = wrapper.findAll('.release-bar')
    const scheduledBars = allBars.filter((b) => !b.classes().includes('release-bar--unscheduled'))
    const unscheduledBars = allBars.filter((b) => b.classes().includes('release-bar--unscheduled'))

    expect(scheduledBars.length).toBeGreaterThanOrEqual(1)
    expect(unscheduledBars).toHaveLength(1)
  })

  it('FR2.3: unscheduled bars are ordered alphabetically top-to-bottom', () => {
    const releases = [
      makeUnscheduledRelease({ id: 1, name: 'v-zzz' }),
      makeUnscheduledRelease({ id: 2, name: 'v-aaa' }),
      makeUnscheduledRelease({ id: 3, name: 'v-mmm' }),
    ]
    const wrapper = mountGantt(releases)

    const bars = wrapper.findAll('.release-bar--unscheduled')
    const names = bars.map((b) => b.find('.bar-name').text())

    expect(names).toEqual(['v-aaa', 'v-mmm', 'v-zzz'])
  })

  it('FR2.4: clicking an unscheduled bar emits clickRelease with the correct release id', async () => {
    const releases = [makeUnscheduledRelease({ id: 42, name: 'v-click-me' })]
    const wrapper = mountGantt(releases)

    const bar = wrapper.find('.release-bar--unscheduled')
    expect(bar.exists()).toBe(true)
    await bar.trigger('click')

    const emitted = wrapper.emitted('clickRelease')
    expect(emitted).toBeTruthy()
    expect(emitted![0]).toEqual([42])
  })

  it('FR2.5: summary badge appears on unscheduled bar when release has assigned artifacts', () => {
    const release = makeUnscheduledRelease({ id: 10, name: 'v-with-badge' })
    const releaseDetails = new Map<number, ReleaseDetail>([
      [10, { ...release, idea_count: 3, defect_count: 1 }],
    ])

    const wrapper = mountGantt([release], releaseDetails)

    const badge = wrapper.find('.release-bar--unscheduled .bar-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toContain('3 ideas')
    expect(badge.text()).toContain('1 defect')
  })

  it('FR2.5: no badge rendered when release has no assigned artifacts', () => {
    const release = makeUnscheduledRelease({ id: 11, name: 'v-no-badge' })
    const releaseDetails = new Map<number, ReleaseDetail>([
      [11, { ...release, idea_count: 0, defect_count: 0 }],
    ])

    const wrapper = mountGantt([release], releaseDetails)

    const badge = wrapper.find('.release-bar--unscheduled .bar-badge')
    expect(badge.exists()).toBe(false)
  })

  it('FR2.1: unscheduled bar renders in unscheduled-cell--bar container', () => {
    const release = makeUnscheduledRelease({ id: 20, name: 'v-cell' })
    const wrapper = mountGantt([release])

    const cell = wrapper.find('.unscheduled-cell--bar')
    expect(cell.exists()).toBe(true)
    expect(cell.find('.release-bar--unscheduled').exists()).toBe(true)
  })

  it('scheduled rows have unscheduled-cell placeholder when unscheduled column is visible', () => {
    const releases = [
      makeScheduledRelease({ id: 1, name: 'v-sched' }),
      makeUnscheduledRelease({ id: 2, name: 'v-unsched' }),
    ]
    const wrapper = mountGantt(releases)

    // The placeholder .unscheduled-cell (without --bar modifier) keeps grid
    // alignment in scheduled rows.
    const placeholder = wrapper.find('.gantt-row .unscheduled-cell:not(.unscheduled-cell--bar)')
    expect(placeholder.exists()).toBe(true)
  })
})

// ---------------------------------------------------------------------------
// Milestone 3 — Regression: Gantt badge counts driven by releaseDetails
// ---------------------------------------------------------------------------

describe('GanttChart — Badge Counts from releaseDetails (Milestone 3)', () => {
  it('Gantt bar badges display idea_count and defect_count from releaseDetails, independent of any modal filter', () => {
    // Provide a scheduled release so we get a regular bar (not --unscheduled).
    const release = makeScheduledRelease({ id: 5, name: 'v-badge-check' })
    const releaseDetails = new Map<number, ReleaseDetail>([
      [5, { ...release, idea_count: 3, defect_count: 2 }],
    ])

    const wrapper = mountGantt([release], releaseDetails)

    // The badge on the scheduled bar must show the counts from releaseDetails.
    const badge = wrapper.find('.release-bar .bar-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toContain('3 ideas')
    expect(badge.text()).toContain('2 defects')
  })

  it('scheduled bar badge uses releaseDetails counts regardless of artifact types that exist', () => {
    // Simulate a scenario where many non-idea/non-defect artifacts are assigned;
    // the badge must still read its numbers solely from releaseDetails counts.
    const release = makeScheduledRelease({ id: 6, name: 'v-mixed-arts' })
    const releaseDetails = new Map<number, ReleaseDetail>([
      [6, { ...release, idea_count: 7, defect_count: 0 }],
    ])

    const wrapper = mountGantt([release], releaseDetails)

    const badge = wrapper.find('.release-bar .bar-badge')
    expect(badge.exists()).toBe(true)
    // 7 ideas should appear; defects badge absent or shows 0.
    expect(badge.text()).toContain('7 ideas')
  })

  it('no badge rendered on scheduled bar when releaseDetails reports zero ideas and defects', () => {
    const release = makeScheduledRelease({ id: 7, name: 'v-zero-counts' })
    const releaseDetails = new Map<number, ReleaseDetail>([
      [7, { ...release, idea_count: 0, defect_count: 0 }],
    ])

    const wrapper = mountGantt([release], releaseDetails)

    const badge = wrapper.find('.release-bar .bar-badge')
    expect(badge.exists()).toBe(false)
  })
})
