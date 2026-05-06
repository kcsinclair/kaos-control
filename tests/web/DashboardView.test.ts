/**
 * Component tests for DashboardGrid and SummaryCountsWidget — Milestone 5
 *
 * Covers:
 *   - DashboardGrid renders registered widgets in correct slot positions
 *   - Summary counts display correctly after API response resolves
 *   - Summary counts remain zero when the API call fails
 *
 * Viewport layout assertions (two-column at ≥1024 px, single-column at
 * <1024 px) require Playwright because happy-dom does not evaluate CSS
 * @media rules. This was agreed in Q4 resolution (option b). Those tests
 * are deferred to a Playwright suite set up separately.
 *
 * Notes on testing approach:
 * ──────────────────────────
 * DashboardGrid reads widgetList, a module-level reactive singleton.
 * Each test clears it in beforeEach (widgetList.splice(0)) to prevent
 * cross-test pollution.
 *
 * SummaryCountsWidget is tested in isolation (not via DashboardView) so
 * that the API mock is straightforward and the test does not depend on
 * async component loading of chart widgets.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { defineComponent, markRaw } from 'vue'

import { widgetList, registerWidget } from '../../web/src/components/dashboard/widgetRegistry'
import DashboardGrid from '../../web/src/components/dashboard/DashboardGrid.vue'
import SummaryCountsWidget from '../../web/src/components/dashboard/widgets/SummaryCountsWidget.vue'

// ---------------------------------------------------------------------------
// Module mocks — prevent real network I/O and WebSocket connections
// ---------------------------------------------------------------------------

vi.mock('@/api/client', () => ({
  api: {
    get: vi.fn().mockResolvedValue({
      total: 0,
      in_progress: 0,
      blocked: 0,
      completed_this_week: 0,
    }),
  },
}))

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: vi.fn(),
}))

// ---------------------------------------------------------------------------
// Router helper — DashboardGrid accepts a :project prop; SummaryCountsWidget
// receives project as a prop directly.
// ---------------------------------------------------------------------------

function makeRouter(path = '/p/testproject/dashboard') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/p/:project/dashboard', component: { template: '<div />' } },
      { path: '/:pathMatch(.*)*', component: { template: '<div />' } },
    ],
  })
  router.push(path)
  return router
}

// Minimal stub widgets used as stand-ins so we don't load real async chunks.
// markRaw prevents Vue from wrapping these in a reactive proxy when they are
// stored inside widgetList (a reactive array), which would cause the
// "Component made reactive" warning and break identity comparisons.
const StubSummary = markRaw(defineComponent({ name: 'StubSummary', template: '<div class="stub-summary" />' }))
const StubChart   = markRaw(defineComponent({ name: 'StubChart',   template: '<div class="stub-chart" />' }))
const StubPanel   = markRaw(defineComponent({ name: 'StubPanel',   template: '<div class="stub-panel" />' }))
const StubChartB  = markRaw(defineComponent({ name: 'StubChartB',  template: '<div class="stub-chart-b" />' }))

// ---------------------------------------------------------------------------
// Setup / teardown
// ---------------------------------------------------------------------------

beforeEach(() => {
  setActivePinia(createPinia())
  // Reset the reactive singleton so each test starts with an empty registry.
  widgetList.splice(0)
})

afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

// ===========================================================================
// Milestone 5 — DashboardGrid slot rendering
// ===========================================================================

describe('DashboardGrid — slot rendering', () => {
  it('renders a widget registered to the summary slot', async () => {
    registerWidget('test-summary', StubSummary, { slot: 'summary', order: 0 })

    const wrapper = mount(DashboardGrid, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    const section = wrapper.find('section[aria-label="Summary statistics"]')
    expect(section.exists()).toBe(true)
    expect(section.find('.stub-summary').exists()).toBe(true)
  })

  it('renders a widget registered to the chart slot', async () => {
    registerWidget('test-chart', StubChart, { slot: 'chart', order: 0 })

    const wrapper = mount(DashboardGrid, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    const section = wrapper.find('section[aria-label="Charts"]')
    expect(section.exists()).toBe(true)
    expect(section.find('.stub-chart').exists()).toBe(true)
  })

  it('renders a widget registered to the panel slot', async () => {
    registerWidget('test-panel', StubPanel, { slot: 'panel', order: 0 })

    const wrapper = mount(DashboardGrid, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    const section = wrapper.find('section[aria-label="Panels"]')
    expect(section.exists()).toBe(true)
    expect(section.find('.stub-panel').exists()).toBe(true)
  })

  it('omits the summary section when no summary widgets are registered', async () => {
    registerWidget('chart-only', StubChart, { slot: 'chart', order: 0 })

    const wrapper = mount(DashboardGrid, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    // v-if="summaryWidgets.length" means the section is absent from the DOM.
    expect(wrapper.find('section[aria-label="Summary statistics"]').exists()).toBe(false)
  })

  it('omits the panel section when no panel widgets are registered', async () => {
    registerWidget('chart-only', StubChart, { slot: 'chart', order: 0 })

    const wrapper = mount(DashboardGrid, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    expect(wrapper.find('section[aria-label="Panels"]').exists()).toBe(false)
  })

  it('renders widgets in ascending order within the chart slot', async () => {
    // Register in reverse order; registerWidget sorts by order.
    registerWidget('second', StubChartB, { slot: 'chart', order: 1 })
    registerWidget('first', StubChart,   { slot: 'chart', order: 0 })

    const wrapper = mount(DashboardGrid, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    const chartsSection = wrapper.find('section[aria-label="Charts"]')
    // The first child should be the order-0 stub, the second the order-1 stub.
    expect(chartsSection.find('.stub-chart').exists()).toBe(true)
    expect(chartsSection.find('.stub-chart-b').exists()).toBe(true)

    const html = chartsSection.html()
    expect(html.indexOf('stub-chart')).toBeLessThan(html.indexOf('stub-chart-b'))
  })

  it('passes the project prop down to each widget', async () => {
    // Use a stub with a named prop that echoes the value into the DOM so we
    // can assert it without relying on closure capture (which breaks when the
    // component is stored in a reactive array and setup() fires at an
    // unexpected time).
    const PropEchoStub = markRaw(defineComponent({
      props: ['project'],
      template: '<div class="prop-echo" :data-project="project" />',
    }))
    registerWidget('prop-echo', PropEchoStub, { slot: 'summary', order: 0 })

    const wrapper = mount(DashboardGrid, {
      props: { project: 'my-special-project' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    const echo = wrapper.find('.prop-echo')
    expect(echo.exists()).toBe(true)
    expect(echo.attributes('data-project')).toBe('my-special-project')
  })

  it('renders widgets across all three slots simultaneously', async () => {
    registerWidget('s', StubSummary, { slot: 'summary', order: 0 })
    registerWidget('c', StubChart,   { slot: 'chart',   order: 0 })
    registerWidget('p', StubPanel,   { slot: 'panel',   order: 0 })

    const wrapper = mount(DashboardGrid, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    expect(wrapper.find('.stub-summary').exists()).toBe(true)
    expect(wrapper.find('.stub-chart').exists()).toBe(true)
    expect(wrapper.find('.stub-panel').exists()).toBe(true)
  })
})

// ===========================================================================
// Milestone 5 — SummaryCountsWidget: display after API response
// ===========================================================================

describe('SummaryCountsWidget — summary counts after API response', () => {
  it('renders four stat cards on mount', async () => {
    const wrapper = mount(SummaryCountsWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    const cards = wrapper.findAll('[role="figure"]')
    expect(cards).toHaveLength(4)
  })

  it('shows zero counts while waiting for the API (initial state)', () => {
    // Do not flush promises — check the synchronous initial render.
    const wrapper = mount(SummaryCountsWidget, {
      props: { project: 'testproject' },
    })

    const cards = wrapper.findAll('[role="figure"]')
    const values = cards.map(c => c.find('.summary-card-value').text())
    expect(values).toEqual(['0', '0', '0', '0'])
  })

  it('displays counts returned by the API after the response resolves', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({
      total: 12,
      in_progress: 3,
      blocked: 1,
      completed_this_week: 5,
    })

    const wrapper = mount(SummaryCountsWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    const cards = wrapper.findAll('[role="figure"]')
    // Card order: Total Tickets, In Progress, Blocked, Completed This Week
    expect(cards[0].find('.summary-card-value').text()).toBe('12')
    expect(cards[1].find('.summary-card-value').text()).toBe('3')
    expect(cards[2].find('.summary-card-value').text()).toBe('1')
    expect(cards[3].find('.summary-card-value').text()).toBe('5')
  })

  it('keeps zero counts when the API call fails (graceful degradation)', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockRejectedValueOnce(new Error('network error'))

    const wrapper = mount(SummaryCountsWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    const values = wrapper.findAll('[role="figure"]').map(c =>
      c.find('.summary-card-value').text()
    )
    expect(values).toEqual(['0', '0', '0', '0'])
  })

  it('calls the API with the correct project-scoped URL', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValue({
      total: 0, in_progress: 0, blocked: 0, completed_this_week: 0,
    })

    mount(SummaryCountsWidget, { props: { project: 'alpha-project' } })
    await flushPromises()

    expect(api.get).toHaveBeenCalledWith('/p/alpha-project/dashboard/stats')
  })

  it('displays correct card labels', async () => {
    const wrapper = mount(SummaryCountsWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    const labels = wrapper.findAll('.summary-card-label').map(el => el.text())
    expect(labels).toContain('Total Tickets')
    expect(labels).toContain('In Progress')
    expect(labels).toContain('Blocked')
    expect(labels).toContain('Completed This Week')
  })
})

// ===========================================================================
// NOTE: Viewport layout tests (two-column grid at ≥1024 px, single-column at
// <1024 px) require Playwright because happy-dom does not evaluate CSS
// @media rules. This was the resolution to Q4 in the test plan (option b).
// Those tests should be added to a tests/e2e/ Playwright suite.
// ===========================================================================
