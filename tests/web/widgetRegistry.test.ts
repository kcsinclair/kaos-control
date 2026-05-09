/**
 * Unit tests for the dashboard widget registry — Milestone 4
 *
 * Tests the extensibility contract:
 *   - registerWidget() adds widgets to the reactive list
 *   - Widgets are sorted by order within each slot
 *   - Duplicate IDs are silently skipped (first registration wins)
 *   - All three slot types ('summary', 'chart', 'panel') are supported
 *
 * Note: widgetList is a module-level reactive singleton. Each test
 * resets it via widgetList.splice(0) in beforeEach to guarantee isolation.
 */

import { describe, it, expect, beforeEach } from 'vitest'
import { defineComponent, markRaw } from 'vue'
import { widgetList, registerWidget } from '../../web/src/components/dashboard/widgetRegistry'
import type { WidgetSlot } from '../../web/src/components/dashboard/widgetRegistry'

// Minimal stub components used as stand-ins for real widgets.
// markRaw prevents Vue from wrapping these in a reactive proxy when they are
// stored inside widgetList (a reactive array). Without markRaw, `component`
// in widgetList[n] is a Proxy, not the original reference, making `toBe`
// identity checks fail and causing Vue's "Component made reactive" warning.
const StubA = markRaw(defineComponent({ template: '<div class="stub-a" />' }))
const StubB = markRaw(defineComponent({ template: '<div class="stub-b" />' }))
const StubC = markRaw(defineComponent({ template: '<div class="stub-c" />' }))

beforeEach(() => {
  // Reset the reactive singleton between tests.
  widgetList.splice(0)
})

// ===========================================================================
// Milestone 4 — registerWidget() adds widgets
// ===========================================================================

describe('widgetRegistry — registerWidget adds widgets', () => {
  it('adds a widget to the reactive list', () => {
    registerWidget('w1', StubA, { slot: 'summary', order: 0 })
    expect(widgetList).toHaveLength(1)
    expect(widgetList[0].id).toBe('w1')
  })

  it('stores the component reference unchanged', () => {
    registerWidget('w1', StubA, { slot: 'summary', order: 0 })
    expect(widgetList[0].component).toBe(StubA)
  })

  it('stores the slot correctly', () => {
    registerWidget('w1', StubA, { slot: 'chart', order: 0 })
    expect(widgetList[0].slot).toBe('chart')
  })

  it('stores the order correctly', () => {
    registerWidget('w1', StubA, { slot: 'panel', order: 7 })
    expect(widgetList[0].order).toBe(7)
  })

  it('adds multiple widgets to the list', () => {
    registerWidget('w1', StubA, { slot: 'summary', order: 0 })
    registerWidget('w2', StubB, { slot: 'chart', order: 0 })
    registerWidget('w3', StubC, { slot: 'panel', order: 0 })
    expect(widgetList).toHaveLength(3)
  })

  it('list starts empty after reset', () => {
    expect(widgetList).toHaveLength(0)
  })
})

// ===========================================================================
// Milestone 4 — Sorting by order within slot
// ===========================================================================

describe('widgetRegistry — sorting by order within slot', () => {
  it('widgets registered out of order are sorted ascending by order', () => {
    registerWidget('high', StubA, { slot: 'chart', order: 10 })
    registerWidget('low', StubA, { slot: 'chart', order: 0 })
    registerWidget('mid', StubA, { slot: 'chart', order: 5 })

    const ids = widgetList.filter(w => w.slot === 'chart').map(w => w.id)
    expect(ids).toEqual(['low', 'mid', 'high'])
  })

  it('widgets are sorted by slot (alphabetical) then by order', () => {
    registerWidget('panel-0', StubA, { slot: 'panel', order: 0 })
    registerWidget('chart-1', StubA, { slot: 'chart', order: 1 })
    registerWidget('chart-0', StubA, { slot: 'chart', order: 0 })
    registerWidget('summary-0', StubA, { slot: 'summary', order: 0 })

    // Alphabetical slot ordering: chart < panel < summary
    expect(widgetList[0].id).toBe('chart-0')
    expect(widgetList[1].id).toBe('chart-1')
    expect(widgetList[2].id).toBe('panel-0')
    expect(widgetList[3].id).toBe('summary-0')
  })

  it('widgets in different slots do not interfere with each other\'s ordering', () => {
    registerWidget('s2', StubA, { slot: 'summary', order: 2 })
    registerWidget('s0', StubA, { slot: 'summary', order: 0 })
    registerWidget('c1', StubA, { slot: 'chart', order: 1 })
    registerWidget('c0', StubA, { slot: 'chart', order: 0 })

    const summaryIds = widgetList.filter(w => w.slot === 'summary').map(w => w.id)
    expect(summaryIds).toEqual(['s0', 's2'])

    const chartIds = widgetList.filter(w => w.slot === 'chart').map(w => w.id)
    expect(chartIds).toEqual(['c0', 'c1'])
  })

  it('two widgets with the same slot and order are both present', () => {
    registerWidget('first', StubA, { slot: 'summary', order: 0 })
    registerWidget('second', StubB, { slot: 'summary', order: 0 })

    const ids = widgetList.filter(w => w.slot === 'summary').map(w => w.id)
    expect(ids).toContain('first')
    expect(ids).toContain('second')
  })
})

// ===========================================================================
// Milestone 4 — Duplicate ID handling
// ===========================================================================

describe('widgetRegistry — duplicate ID handling', () => {
  it('registering a duplicate ID does not add a second entry (silently skipped)', () => {
    registerWidget('dup', StubA, { slot: 'summary', order: 0 })
    registerWidget('dup', StubB, { slot: 'chart', order: 99 })

    expect(widgetList.filter(w => w.id === 'dup')).toHaveLength(1)
  })

  it('first registration is preserved on duplicate (not overwritten)', () => {
    registerWidget('dup', StubA, { slot: 'summary', order: 0 })
    registerWidget('dup', StubB, { slot: 'chart', order: 99 })

    expect(widgetList[0].slot).toBe('summary')
    expect(widgetList[0].order).toBe(0)
    expect(widgetList[0].component).toBe(StubA)
  })

  it('registering the same ID three times still yields one entry', () => {
    registerWidget('x', StubA, { slot: 'panel', order: 0 })
    registerWidget('x', StubA, { slot: 'panel', order: 1 })
    registerWidget('x', StubA, { slot: 'panel', order: 2 })

    expect(widgetList).toHaveLength(1)
  })

  it('duplicate registration is idempotent (re-registration after hot reload)', () => {
    registerWidget('hmr-widget', StubA, { slot: 'summary', order: 0 })
    registerWidget('hmr-widget', StubA, { slot: 'summary', order: 0 })

    expect(widgetList).toHaveLength(1)
    expect(widgetList[0].id).toBe('hmr-widget')
  })

  it('non-duplicate IDs are all registered independently', () => {
    registerWidget('a', StubA, { slot: 'summary', order: 0 })
    registerWidget('b', StubB, { slot: 'summary', order: 1 })
    registerWidget('c', StubC, { slot: 'summary', order: 2 })

    expect(widgetList).toHaveLength(3)
  })
})

// ===========================================================================
// Milestone 4 — All three slot types supported
// ===========================================================================

describe('widgetRegistry — all three slot types', () => {
  const slots: WidgetSlot[] = ['summary', 'chart', 'panel']

  for (const slot of slots) {
    it(`supports the "${slot}" slot`, () => {
      registerWidget(`test-${slot}`, StubA, { slot, order: 0 })
      expect(widgetList.some(w => w.slot === slot)).toBe(true)
    })
  }

  it('all three slots can coexist in the same list', () => {
    registerWidget('s', StubA, { slot: 'summary', order: 0 })
    registerWidget('c', StubB, { slot: 'chart', order: 0 })
    registerWidget('p', StubC, { slot: 'panel', order: 0 })

    const presentSlots = new Set(widgetList.map(w => w.slot))
    expect(presentSlots.has('summary')).toBe(true)
    expect(presentSlots.has('chart')).toBe(true)
    expect(presentSlots.has('panel')).toBe(true)
  })

  it('filtering by slot returns only widgets in that slot', () => {
    registerWidget('s1', StubA, { slot: 'summary', order: 0 })
    registerWidget('c1', StubB, { slot: 'chart', order: 0 })
    registerWidget('p1', StubC, { slot: 'panel', order: 0 })

    expect(widgetList.filter(w => w.slot === 'summary')).toHaveLength(1)
    expect(widgetList.filter(w => w.slot === 'chart')).toHaveLength(1)
    expect(widgetList.filter(w => w.slot === 'panel')).toHaveLength(1)
  })
})

// ===========================================================================
// Milestone 4 — stages-distribution widget registration contract
//
// These tests verify the expected registration arguments that registerWidgets.ts
// uses for the stages-distribution widget. They call registerWidget directly
// (mirroring what registerWidgets.ts does) in a clean widgetList so each test
// is self-contained and does not depend on module load order.
// ===========================================================================

describe('widgetRegistry — stages-distribution registration (Milestone 4)', () => {
  it('TC1: stages-distribution is registered in the chart slot with order 1', () => {
    registerWidget('stages-distribution', StubA, { slot: 'chart', order: 1 })

    const entry = widgetList.find(w => w.id === 'stages-distribution')
    expect(entry).toBeDefined()
    expect(entry?.slot).toBe('chart')
    expect(entry?.order).toBe(1)
  })

  it('TC2: chart-slot widgets are ordered status-distribution(0) → stages-distribution(1) → velocity-chart(2)', () => {
    // Register all three chart-slot widgets in arbitrary order.
    registerWidget('velocity-chart',      StubC, { slot: 'chart', order: 2 })
    registerWidget('stages-distribution', StubB, { slot: 'chart', order: 1 })
    registerWidget('status-distribution', StubA, { slot: 'chart', order: 0 })

    const chartIds = widgetList
      .filter(w => w.slot === 'chart')
      .map(w => w.id)

    expect(chartIds).toEqual(['status-distribution', 'stages-distribution', 'velocity-chart'])
  })

  it('TC3: registering stages-distribution twice does not create a duplicate entry', () => {
    registerWidget('stages-distribution', StubA, { slot: 'chart', order: 1 })
    registerWidget('stages-distribution', StubB, { slot: 'chart', order: 99 })

    expect(widgetList.filter(w => w.id === 'stages-distribution')).toHaveLength(1)
  })

  it('TC3b: first registration is preserved on duplicate (not overwritten)', () => {
    registerWidget('stages-distribution', StubA, { slot: 'chart', order: 1 })
    registerWidget('stages-distribution', StubB, { slot: 'panel', order: 99 })

    const entry = widgetList.find(w => w.id === 'stages-distribution')
    expect(entry?.slot).toBe('chart')
    expect(entry?.order).toBe(1)
    expect(entry?.component).toBe(StubA)
  })

  it('status-distribution retains order 0 (unchanged by stages-distribution addition)', () => {
    registerWidget('status-distribution', StubA, { slot: 'chart', order: 0 })
    registerWidget('stages-distribution', StubB, { slot: 'chart', order: 1 })

    const statusEntry = widgetList.find(w => w.id === 'status-distribution')
    expect(statusEntry?.order).toBe(0)
  })

  it('velocity-chart retains order 2 (updated slot position with stages-distribution at 1)', () => {
    registerWidget('status-distribution', StubA, { slot: 'chart', order: 0 })
    registerWidget('stages-distribution', StubB, { slot: 'chart', order: 1 })
    registerWidget('velocity-chart',      StubC, { slot: 'chart', order: 2 })

    const velocityEntry = widgetList.find(w => w.id === 'velocity-chart')
    expect(velocityEntry?.order).toBe(2)
  })
})
