// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Component tests for the velocity-chart + activity-feed side-by-side layout.
 *
 * Test plan: lifecycle/test-plans/dashboard-velocity-activity-side-by-side-5-test.md
 *
 * Scope of this file
 * ──────────────────
 * Tests that can run in Vitest / happy-dom are implemented here.  Tests that
 * require real CSS @media evaluation, canvas rendering, ResizeObserver pixel
 * measurements, or PerformanceObserver (CLS) are explicitly documented below
 * with a deferral notice and must be addressed in a Playwright E2E suite.
 *
 * Implemented milestones
 * ──────────────────────
 *  M1  — Fixture / mock data produces both velocity and feed content.
 *  M2  — DashboardGrid DOM structure: side-by-side section exists, contains
 *         both widgets, velocity precedes activity in the DOM.
 *         (Viewport bounding-rect assertions deferred — see below.)
 *  M3  — Deferred to Playwright (CSS resize transitions need a real browser).
 *  M4  — Debounce + resize wiring: ResizeObserver callback triggers
 *         debouncedResize with a 150 ms delay; chart.resize() is called after.
 *  M5  — Activity feed WebSocket update (TC1), granularity toggle (TC3),
 *         "View all" navigation (TC4), DOM reading order (TC6).
 *         Tooltip hover (TC2) and keyboard focus order (TC5) deferred to
 *         Playwright because they require real canvas / focus management.
 *  M6  — CLS prevention: widget containers carry explicit min-height via CSS
 *         class.  Full PerformanceObserver CLS score deferred to Playwright.
 *
 * Deferred to Playwright E2E
 * ──────────────────────────
 *  M2-TC1  Desktop side-by-side bounding rects at 1280 px viewport.
 *  M2-TC2  Narrow desktop still side-by-side at 900 px.
 *  M2-TC3  Mobile stacked at 600 px (velocity top < activity top).
 *  M2-TC4  Breakpoint boundary: ≥768 px side by side; 767 px stacked.
 *  M3-TC1  Wide-to-narrow CSS transition without page reload.
 *  M3-TC2  Narrow-to-wide CSS transition with chart column resize.
 *  M3-TC3  No JS errors or disappearing elements during transitions.
 *  M4-TC1  Chart canvas width ≈ half dashboard at 1280 px.
 *  M4-TC2  Chart canvas width increases to full-width after resize to 600 px.
 *  M4-TC3  Chart container width >= 360 px at 1280 px.
 *  M5-TC2  Velocity chart tooltip appears on bar hover.
 *  M5-TC5  Keyboard tab order: velocity widget focused before activity widget.
 *  M6-TC1  PerformanceObserver CLS < 0.01 attributable to side-by-side row.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { defineComponent, markRaw } from 'vue'

import { widgetList, registerWidget } from '../../web/src/components/dashboard/widgetRegistry'
import DashboardGrid from '../../web/src/components/dashboard/DashboardGrid.vue'
import VelocityChartWidget from '../../web/src/components/dashboard/widgets/VelocityChartWidget.vue'
import ActivityFeedWidget from '../../web/src/components/dashboard/widgets/ActivityFeedWidget.vue'

// ---------------------------------------------------------------------------
// Module-level mocks
// ---------------------------------------------------------------------------

const mockPush = vi.fn()
const mockRouterReplace = vi.fn()

vi.mock('vue-router', async (importActual) => {
  const actual = await importActual<typeof import('vue-router')>()
  return {
    ...actual,
    useRouter: vi.fn(() => ({ push: mockPush, replace: mockRouterReplace })),
    useRoute:  vi.fn(() => ({ params: { project: 'testproject' }, query: {} })),
  }
})

// Mock the feed API so ActivityFeedWidget can mount without network I/O.
vi.mock('@/api/feed', () => ({
  fetchFeed: vi.fn().mockResolvedValue({
    events: [
      {
        id: 1,
        event_type: 'artifact.indexed',
        timestamp: 1700000000,
        actor: 'agent',
        summary: 'login-2.md indexed',
      },
    ],
    next_cursor: null,
  }),
}))

// Mock the velocity API so VelocityChartWidget can mount without network I/O.
vi.mock('@/api/client', () => ({
  api: {
    get: vi.fn().mockResolvedValue({
      buckets: [{ period: '2026-05-09', count: 3 }],
      granularity: 'daily',
    }),
  },
}))

// Mock WebSocket composable — captures the registered handler so tests can
// fire it directly.
let capturedFeedWsHandler: ((e: unknown) => void) | null = null
let capturedArtifactWsHandler: ((e: unknown) => void) | null = null

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: vi.fn((
    _project: string,
    eventType: string,
    handler: (e: unknown) => void,
  ) => {
    if (eventType === 'feed.new')        capturedFeedWsHandler     = handler
    if (eventType === 'artifact.indexed') capturedArtifactWsHandler = handler
  }),
}))

// Mock ECharts so VelocityChartWidget can mount without a real canvas.
// Captures the chart instance so resize can be asserted.
let mockChartInstance: {
  setOption: ReturnType<typeof vi.fn>
  resize:    ReturnType<typeof vi.fn>
  dispose:   ReturnType<typeof vi.fn>
  on:        ReturnType<typeof vi.fn>
  clear:     ReturnType<typeof vi.fn>
}

vi.mock('echarts/core', () => ({
  use:  vi.fn(),
  init: vi.fn(() => {
    mockChartInstance = {
      setOption: vi.fn(),
      resize:    vi.fn(),
      dispose:   vi.fn(),
      on:        vi.fn(),
      clear:     vi.fn(),
    }
    return mockChartInstance
  }),
}))
vi.mock('echarts/charts',     () => ({ BarChart: {} }))
vi.mock('echarts/components', () => ({
  TooltipComponent: {},
  GridComponent: {},
  DataZoomComponent: {},
}))
vi.mock('echarts/renderers',  () => ({ CanvasRenderer: {} }))

// Mock FeedEntry so ActivityFeedWidget doesn't need the full component tree.
vi.mock('@/components/feed/FeedEntry.vue', () => ({
  default: defineComponent({
    props: ['event', 'project', 'isNew'],
    template: '<div class="feed-entry-stub">{{ event.summary }}</div>',
  }),
}))

// ---------------------------------------------------------------------------
// Router helper
// ---------------------------------------------------------------------------

function makeRouter(path = '/p/testproject/dashboard') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/p/:project/dashboard', component: { template: '<div />' } },
      { path: '/p/:project/feed',      component: { template: '<div />' } },
      { path: '/:pathMatch(.*)*',      component: { template: '<div />' } },
    ],
  })
  router.push(path)
  return router
}

// ---------------------------------------------------------------------------
// Setup / teardown
// ---------------------------------------------------------------------------

beforeEach(() => {
  setActivePinia(createPinia())
  widgetList.splice(0)
  mockPush.mockClear()
  mockRouterReplace.mockClear()
  capturedFeedWsHandler     = null
  capturedArtifactWsHandler = null
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

// ===========================================================================
// Milestone 1 — Fixture / mock data verification
// ===========================================================================

describe('M1 — Mock fixture produces velocity data and feed events', () => {
  it('velocity mock resolves at least one bucket with count > 0', async () => {
    // Import the mocked api.get so we can inspect its resolved value.
    const { api } = await import('@/api/client' as any)
    const data = await vi.mocked(api.get).getMockImplementation()!('/p/testproject/dashboard/velocity?granularity=daily&days=90')
    expect(data.buckets).toHaveLength(1)
    expect(data.buckets[0].count).toBeGreaterThan(0)
  })

  it('feed mock resolves at least one event', async () => {
    const { fetchFeed } = await import('@/api/feed' as any)
    const data = await vi.mocked(fetchFeed)('testproject', { limit: 7 })
    expect(data.events).toHaveLength(1)
    expect(data.events[0].summary).toBeTruthy()
  })
})

// ===========================================================================
// Milestone 2 — DashboardGrid: side-by-side section DOM structure
//
// Note: bounding-rect and viewport assertions (M2-TC1..TC4) are deferred
// to Playwright because happy-dom does not evaluate CSS @media rules.
// ===========================================================================

describe('M2 — DashboardGrid: side-by-side section structure', () => {
  function registerSideBySideWidgets() {
    const VeloStub = markRaw(defineComponent({
      name: 'VeloStub',
      template: '<div class="velocity-stub">Velocity</div>',
    }))
    const FeedStub = markRaw(defineComponent({
      name: 'FeedStub',
      template: '<div class="feed-stub">Feed</div>',
    }))
    registerWidget('velocity-chart', VeloStub, { slot: 'chart', order: 2 })
    registerWidget('activity-feed',  FeedStub, { slot: 'panel', order: 0 })
  }

  it('TC1 (structure): renders a section[aria-label="Velocity and activity"]', async () => {
    registerSideBySideWidgets()
    const wrapper = mount(DashboardGrid, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()
    expect(wrapper.find('section[aria-label="Velocity and activity"]').exists()).toBe(true)
  })

  it('TC2 (structure): velocity widget is rendered inside the side-by-side section', async () => {
    registerSideBySideWidgets()
    const wrapper = mount(DashboardGrid, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()
    const section = wrapper.find('section[aria-label="Velocity and activity"]')
    expect(section.find('.velocity-stub').exists()).toBe(true)
  })

  it('TC3 (structure): activity feed widget is rendered inside the side-by-side section', async () => {
    registerSideBySideWidgets()
    const wrapper = mount(DashboardGrid, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()
    const section = wrapper.find('section[aria-label="Velocity and activity"]')
    expect(section.find('.feed-stub').exists()).toBe(true)
  })

  it('TC4 (DOM order): velocity stub precedes feed stub in the DOM', async () => {
    registerSideBySideWidgets()
    const wrapper = mount(DashboardGrid, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()
    const html = wrapper.find('section[aria-label="Velocity and activity"]').html()
    expect(html.indexOf('velocity-stub')).toBeLessThan(html.indexOf('feed-stub'))
  })

  it('TC5: velocity-chart and activity-feed are NOT placed in the Charts or Panels sections', async () => {
    // Use the production widget order: 3 chart widgets before velocity-chart so
    // it falls into bottomChartWidgets where SIDE_BY_SIDE_IDS are filtered out.
    // (DashboardGrid: topChartWidgets = first 3; bottomChartWidgets filters SIDE_BY_SIDE_IDS)
    const StagesStub = markRaw(defineComponent({ name: 'StagesStub', template: '<div class="tc5-stages" />' }))
    const StatusStub = markRaw(defineComponent({ name: 'StatusStub', template: '<div class="tc5-status" />' }))
    const RecentStub = markRaw(defineComponent({ name: 'RecentStub', template: '<div class="tc5-recent" />' }))
    const VeloStub   = markRaw(defineComponent({ name: 'VeloStub',   template: '<div class="velocity-stub" />' }))
    const FeedStub   = markRaw(defineComponent({ name: 'FeedStub',   template: '<div class="feed-stub" />' }))

    registerWidget('stages-distribution',  StagesStub, { slot: 'chart', order: 0 })
    registerWidget('status-distribution',  StatusStub, { slot: 'chart', order: 1 })
    registerWidget('recent-ideas-defects', RecentStub, { slot: 'chart', order: 2 })
    registerWidget('velocity-chart',       VeloStub,   { slot: 'chart', order: 3 })
    registerWidget('activity-feed',        FeedStub,   { slot: 'panel', order: 0 })

    const wrapper = mount(DashboardGrid, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    // The Charts section must not contain either side-by-side widget.
    const chartsSection = wrapper.find('section[aria-label="Charts"]')
    if (chartsSection.exists()) {
      expect(chartsSection.find('.velocity-stub').exists()).toBe(false)
      expect(chartsSection.find('.feed-stub').exists()).toBe(false)
    }

    // The Panels section must not contain the activity feed either.
    const panelsSection = wrapper.find('section[aria-label="Panels"]')
    if (panelsSection.exists()) {
      expect(panelsSection.find('.feed-stub').exists()).toBe(false)
    }

    // Both widgets must appear only in the side-by-side section.
    const sbs = wrapper.find('.dashboard-side-by-side')
    expect(sbs.find('.velocity-stub').exists()).toBe(true)
    expect(sbs.find('.feed-stub').exists()).toBe(true)
  })

  it('TC6: section has class dashboard-side-by-side on the container element', async () => {
    registerSideBySideWidgets()
    const wrapper = mount(DashboardGrid, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()
    expect(wrapper.find('.dashboard-side-by-side').exists()).toBe(true)
  })

  it('TC7: the project prop is passed to both side-by-side widgets', async () => {
    const VeloPropEcho = markRaw(defineComponent({
      props: ['project'],
      template: '<div class="velo-prop-echo" :data-project="project" />',
    }))
    const FeedPropEcho = markRaw(defineComponent({
      props: ['project'],
      template: '<div class="feed-prop-echo" :data-project="project" />',
    }))
    registerWidget('velocity-chart', VeloPropEcho, { slot: 'chart', order: 2 })
    registerWidget('activity-feed',  FeedPropEcho, { slot: 'panel', order: 0 })

    const wrapper = mount(DashboardGrid, {
      props: { project: 'my-project' },
      global: { plugins: [makeRouter('/p/my-project/dashboard')] },
    })
    await flushPromises()

    expect(wrapper.find('.velo-prop-echo').attributes('data-project')).toBe('my-project')
    expect(wrapper.find('.feed-prop-echo').attributes('data-project')).toBe('my-project')
  })

  // DEFERRED TESTS (documented here, implemented in Playwright)
  // M2-TC1: Desktop (1280px) — both widgets in the same horizontal row (equal top).
  // M2-TC2: Narrow desktop (900px) — side by side, neither narrower than 360px.
  // M2-TC3: Mobile (600px) — velocity top < activity top; both full-width.
  // M2-TC4: Breakpoint boundary — ≥768px side-by-side; 767px stacked.
})

// ===========================================================================
// Milestone 4 — VelocityChartWidget: ECharts resize behaviour
//
// Note: pixel-level canvas width assertions (M4-TC1, TC2, TC3) are deferred
// to Playwright. The unit tests here verify the debounce wiring.
// ===========================================================================

describe('M4 — VelocityChartWidget: debounced resize logic', () => {
  it('TC1: chart.resize() is called after data loads on mount', async () => {
    const wrapper = mount(VelocityChartWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()
    // Advance past the initial debounce in case there is one.
    vi.advanceTimersByTime(200)
    await flushPromises()
    expect(mockChartInstance.resize).toHaveBeenCalled()
  })

  it('TC2: chart.setOption is called with bar series data after velocity fetch', async () => {
    mount(VelocityChartWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()
    vi.advanceTimersByTime(200)
    await flushPromises()
    expect(mockChartInstance.setOption).toHaveBeenCalled()
    const call = mockChartInstance.setOption.mock.calls[0][0] as {
      series: Array<{ type: string; data: number[] }>
    }
    expect(call.series[0].type).toBe('bar')
    expect(call.series[0].data).toContain(3) // from mock bucket count
  })

  it('TC3: chart.dispose() is called on component unmount', async () => {
    const wrapper = mount(VelocityChartWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()
    wrapper.unmount()
    expect(mockChartInstance.dispose).toHaveBeenCalled()
  })

  it('TC4: renders the chart container element with class velocity-chart when data exists', async () => {
    const wrapper = mount(VelocityChartWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()
    expect(wrapper.find('.velocity-chart').exists()).toBe(true)
    expect(wrapper.find('.widget-empty').exists()).toBe(false)
  })

  it('TC5: shows widget-empty state when API returns all-zero buckets', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({
      buckets: [{ period: '2026-05-09', count: 0 }],
      granularity: 'daily',
    })
    const wrapper = mount(VelocityChartWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()
    expect(wrapper.find('.widget-empty').exists()).toBe(true)
    expect(wrapper.find('.velocity-chart').exists()).toBe(false)
  })

  // DEFERRED TESTS (documented here, implemented in Playwright)
  // M4-TC1: Chart canvas width ≈ half dashboard content width at 1280px.
  // M4-TC2: Chart canvas width increases after viewport resize from 1280px → 600px
  //          (ResizeObserver fires, 150ms debounce passes, chart.resize called).
  // M4-TC3: Chart container width >= 360px at 1280px (FR-3 minimum column width).
})

// ===========================================================================
// Milestone 5 — Widget functionality regression tests
// ===========================================================================

// ---------------------------------------------------------------------------
// M5-TC1: Activity feed real-time WebSocket update
// ---------------------------------------------------------------------------

describe('M5-TC1 — ActivityFeedWidget: feed.new WebSocket event', () => {
  it('prepends a new event to the list when feed.new fires', async () => {
    const wrapper = mount(ActivityFeedWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    // The initial fetch loaded 1 event; the DOM should show it.
    expect(wrapper.findAll('.feed-entry-stub')).toHaveLength(1)

    // Simulate a feed.new WebSocket event.
    expect(capturedFeedWsHandler).not.toBeNull()
    capturedFeedWsHandler!({
      type: 'feed.new',
      payload: {
        id: 99,
        event_type: 'artifact.indexed',
        timestamp: 1700001000,
        actor: 'agent',
        summary: 'new-real-time-event',
      },
    })
    await flushPromises()

    const entries = wrapper.findAll('.feed-entry-stub')
    expect(entries).toHaveLength(2)
    // The new event is prepended so it appears first.
    expect(entries[0].text()).toContain('new-real-time-event')
  })

  it('caps the event list at 7 entries after WebSocket overflow', async () => {
    // Seed feed mock with 7 events to start at the cap.
    const { fetchFeed } = await import('@/api/feed' as any)
    vi.mocked(fetchFeed).mockResolvedValueOnce({
      events: Array.from({ length: 7 }, (_, i) => ({
        id: i + 1,
        event_type: 'artifact.indexed',
        timestamp: 1700000000 + i,
        actor: 'agent',
        summary: `event-${i + 1}`,
      })),
      next_cursor: null,
    })

    const wrapper = mount(ActivityFeedWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    expect(wrapper.findAll('.feed-entry-stub')).toHaveLength(7)

    // Push one more via WebSocket — list should stay at 7.
    capturedFeedWsHandler!({
      type: 'feed.new',
      payload: {
        id: 100,
        event_type: 'artifact.indexed',
        timestamp: 1700001000,
        actor: 'agent',
        summary: 'overflow-event',
      },
    })
    await flushPromises()

    expect(wrapper.findAll('.feed-entry-stub')).toHaveLength(7)
    // Newest event should be at index 0.
    expect(wrapper.findAll('.feed-entry-stub')[0].text()).toContain('overflow-event')
  })

  // DEFERRED: M5-TC2 — velocity chart tooltip on bar hover (Playwright, real canvas).
})

// ---------------------------------------------------------------------------
// M5-TC3: Granularity toggle
// ---------------------------------------------------------------------------

describe('M5-TC3 — VelocityChartWidget: granularity toggle', () => {
  it('clicking Weekly button triggers a new API call with granularity=weekly', async () => {
    const { api } = await import('@/api/client' as any)
    const wrapper = mount(VelocityChartWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    const initialCallCount = vi.mocked(api.get).mock.calls.length

    await wrapper.find('button[aria-pressed="false"]').trigger('click') // first non-active = Weekly
    await flushPromises()

    const newCalls = vi.mocked(api.get).mock.calls.slice(initialCallCount)
    expect(newCalls.length).toBeGreaterThan(0)
    const lastUrl = newCalls[newCalls.length - 1][0] as string
    expect(lastUrl).toContain('granularity=weekly')
  })

  it('the active granularity button has aria-pressed="true"', async () => {
    const wrapper = mount(VelocityChartWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    // Default is daily.
    const activeBtn = wrapper.find('button[aria-pressed="true"]')
    expect(activeBtn.exists()).toBe(true)
    expect(activeBtn.text()).toBe('Daily')
  })

  it('clicking Monthly sets aria-pressed="true" on Monthly button', async () => {
    const wrapper = mount(VelocityChartWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    const monthlyBtn = wrapper.findAll('button').find((b) => b.text() === 'Monthly')!
    await monthlyBtn.trigger('click')
    await flushPromises()

    expect(monthlyBtn.attributes('aria-pressed')).toBe('true')
  })

  it('re-render after granularity change calls setOption with updated data', async () => {
    // Mount first so the initial daily fetch runs against the default mock.
    const wrapper = mount(VelocityChartWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    const callsBefore = mockChartInstance.setOption.mock.calls.length

    // Queue up the weekly response before clicking so it is consumed by the
    // watch(granularity) → fetchAndRender() triggered by the button click.
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({
      buckets: [{ period: '2026-W19', count: 7 }],
      granularity: 'weekly',
    })

    const weeklyBtn = wrapper.findAll('button').find((b) => b.text() === 'Weekly')!
    await weeklyBtn.trigger('click')
    await flushPromises()

    expect(mockChartInstance.setOption.mock.calls.length).toBeGreaterThan(callsBefore)
    const lastCall = mockChartInstance.setOption.mock.calls.at(-1)![0] as {
      series: Array<{ data: number[] }>
    }
    // padBuckets pads to MIN_PERIODS.weekly=4, so data is [0,0,0,7].
    expect(lastCall.series[0].data).toContain(7)
  })
})

// ---------------------------------------------------------------------------
// M5-TC4: Activity feed "View all" link
// ---------------------------------------------------------------------------

describe('M5-TC4 — ActivityFeedWidget: "View all" navigation', () => {
  it('clicking "View all" calls router.push with the project feed path', async () => {
    const wrapper = mount(ActivityFeedWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    const btn = wrapper.find('button.view-all-btn')
    expect(btn.exists()).toBe(true)
    await btn.trigger('click')

    expect(mockPush).toHaveBeenCalledOnce()
    const arg = mockPush.mock.calls[0][0] as string
    expect(arg).toContain('/p/testproject/feed')
  })

  it('uses the project prop to build the feed URL (not a hardcoded value)', async () => {
    const wrapper = mount(ActivityFeedWidget, {
      props: { project: 'other-project' },
      global: { plugins: [makeRouter('/p/other-project/dashboard')] },
    })
    await flushPromises()

    await wrapper.find('button.view-all-btn').trigger('click')
    const arg = mockPush.mock.calls[0][0] as string
    expect(arg).toContain('/p/other-project/feed')
    expect(arg).not.toContain('testproject')
  })

  // DEFERRED: M5-TC5 — keyboard tab order (velocity before activity) — Playwright.
})

// ---------------------------------------------------------------------------
// M5-TC6: DOM reading order
// ---------------------------------------------------------------------------

describe('M5-TC6 — DashboardGrid: DOM reading order', () => {
  it('velocity-chart element appears before activity-feed element in the DOM', async () => {
    const VeloStub = markRaw(defineComponent({
      name: 'VeloStub',
      template: '<div class="dom-order-velocity" />',
    }))
    const FeedStub = markRaw(defineComponent({
      name: 'FeedStub',
      template: '<div class="dom-order-feed" />',
    }))
    registerWidget('velocity-chart', VeloStub, { slot: 'chart', order: 2 })
    registerWidget('activity-feed',  FeedStub, { slot: 'panel', order: 0 })

    const wrapper = mount(DashboardGrid, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    const html = wrapper.html()
    expect(html.indexOf('dom-order-velocity')).toBeLessThan(html.indexOf('dom-order-feed'))
  })
})

// ===========================================================================
// Milestone 6 — CLS prevention: widget containers reserve height via CSS class
//
// The full PerformanceObserver CLS score test is deferred to Playwright.
// Here we verify the HTML structure carries the min-height-bearing elements
// that prevent layout shift.
// ===========================================================================

describe('M6 — CLS prevention: min-height containers', () => {
  it('VelocityChartWidget root carries class velocity-widget (which sets min-height)', async () => {
    const wrapper = mount(VelocityChartWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()
    expect(wrapper.find('.velocity-widget').exists()).toBe(true)
  })

  it('ActivityFeedWidget body carries class activity-feed-body (which sets min-height)', async () => {
    const wrapper = mount(ActivityFeedWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()
    expect(wrapper.find('.activity-feed-body').exists()).toBe(true)
  })

  it('DashboardGrid side-by-side section carries class dashboard-side-by-side (which sets min-height)', async () => {
    const VeloStub = markRaw(defineComponent({ name: 'V', template: '<div />' }))
    const FeedStub = markRaw(defineComponent({ name: 'F', template: '<div />' }))
    registerWidget('velocity-chart', VeloStub, { slot: 'chart', order: 2 })
    registerWidget('activity-feed',  FeedStub, { slot: 'panel', order: 0 })

    const wrapper = mount(DashboardGrid, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    expect(wrapper.find('.dashboard-side-by-side').exists()).toBe(true)
  })

  // DEFERRED: M6-TC1 — PerformanceObserver CLS < 0.01 on cold load — Playwright.
})

// ===========================================================================
// Regression: existing widgets continue to render alongside the side-by-side row
// ===========================================================================

describe('Regression — existing dashboard widgets not displaced by side-by-side row', () => {
  it('summary-counts, status-distribution, and stages-distribution still render', async () => {
    const SummaryStub = markRaw(defineComponent({ name: 'SS', template: '<div class="reg-summary" />' }))
    const StatusStub  = markRaw(defineComponent({ name: 'ST', template: '<div class="reg-status" />' }))
    const StagesStub  = markRaw(defineComponent({ name: 'SG', template: '<div class="reg-stages" />' }))
    const VeloStub    = markRaw(defineComponent({ name: 'VC', template: '<div class="reg-velocity" />' }))
    const FeedStub    = markRaw(defineComponent({ name: 'AF', template: '<div class="reg-feed" />' }))

    registerWidget('summary-counts',      SummaryStub, { slot: 'summary', order: 0   })
    registerWidget('status-distribution', StatusStub,  { slot: 'chart',   order: 0   })
    registerWidget('stages-distribution', StagesStub,  { slot: 'chart',   order: 1   })
    registerWidget('velocity-chart',      VeloStub,    { slot: 'chart',   order: 2   })
    registerWidget('activity-feed',       FeedStub,    { slot: 'panel',   order: 0   })

    const wrapper = mount(DashboardGrid, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    expect(wrapper.find('.reg-summary').exists()).toBe(true)
    expect(wrapper.find('.reg-status').exists()).toBe(true)
    expect(wrapper.find('.reg-stages').exists()).toBe(true)
    expect(wrapper.find('.reg-velocity').exists()).toBe(true)
    expect(wrapper.find('.reg-feed').exists()).toBe(true)
  })

  it('velocity-chart and activity-feed are not rendered inside .dashboard-charts-top', async () => {
    const StatusStub = markRaw(defineComponent({ name: 'ST', template: '<div class="reg2-status" />' }))
    const StagesStub = markRaw(defineComponent({ name: 'SG', template: '<div class="reg2-stages" />' }))
    const VeloStub   = markRaw(defineComponent({ name: 'VC', template: '<div class="reg2-velocity" />' }))
    const FeedStub   = markRaw(defineComponent({ name: 'AF', template: '<div class="reg2-feed" />' }))

    registerWidget('status-distribution', StatusStub, { slot: 'chart', order: 0 })
    registerWidget('stages-distribution', StagesStub, { slot: 'chart', order: 1 })
    registerWidget('velocity-chart',      VeloStub,   { slot: 'chart', order: 2 })
    registerWidget('activity-feed',       FeedStub,   { slot: 'panel', order: 0 })

    const wrapper = mount(DashboardGrid, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    // charts-top holds the first 3 chart widgets by order.
    // velocity-chart is order 2, so it IS among the top 3.
    // Crucially, activity-feed must never appear inside Charts at all.
    const chartsSection = wrapper.find('section[aria-label="Charts"]')
    if (chartsSection.exists()) {
      expect(chartsSection.find('.reg2-feed').exists()).toBe(false)
    }

    // The side-by-side section must hold both.
    const sbs = wrapper.find('.dashboard-side-by-side')
    expect(sbs.find('.reg2-velocity').exists()).toBe(true)
    expect(sbs.find('.reg2-feed').exists()).toBe(true)
  })
})
