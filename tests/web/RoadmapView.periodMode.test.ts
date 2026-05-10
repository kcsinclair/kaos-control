// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Tests for RoadmapView toolbar period-mode selector and roadmapSettings store.
 *
 * Covers Milestones 2 and 8 from the Roadmap Gantt Period Display Options test plan:
 *   - Milestone 2: Period-mode selector and fixed-period picker visibility
 *   - Milestone 8: Default-from-config initial state; mode/period independence
 *
 * Testing approach (happy-dom constraints):
 * ─────────────────────────────────────────
 * RoadmapView makes network calls (releases API, WebSocket) on mount.  We mock
 * those entirely so no real I/O occurs.  The roadmapSettings store is driven
 * directly (bypassing the async config fetch) to keep tests deterministic.
 *
 * The store's loadDefaultPeriodMode is unit-tested separately by calling it
 * directly with a mocked getConfig response.
 *
 * Viewport-level checks (overflow, sticky positioning) require a real browser;
 * we assert CSS class presence instead.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { useRoadmapSettingsStore } from '../../web/src/stores/roadmapSettings'

// ---------------------------------------------------------------------------
// Module mocks — prevent real network I/O and WebSocket connections
// ---------------------------------------------------------------------------

vi.mock('@/stores/releases', () => ({
  useReleasesStore: () => ({
    releases: [],
    loading: false,
    fetch: vi.fn().mockResolvedValue(undefined),
    connectWs: vi.fn(),
    disconnectWs: vi.fn(),
    byId: vi.fn().mockReturnValue(null),
    remove: vi.fn().mockResolvedValue(undefined),
  }),
}))

vi.mock('@/stores/artifacts', () => ({
  useArtifactsStore: () => ({
    items: [],
    fetchList: vi.fn().mockResolvedValue(undefined),
  }),
}))

vi.mock('@/api/ws', () => ({
  getProjectWs: vi.fn(() => ({
    onType: vi.fn(() => () => {}),
  })),
}))

vi.mock('@/api/releases', () => ({
  getRelease: vi.fn().mockResolvedValue({}),
  listReleaseArtifacts: vi.fn().mockResolvedValue([]),
}))

// Intercept config fetch — we override individual tests that need specific behaviour.
vi.mock('@/api/config', () => ({
  getConfig: vi.fn().mockResolvedValue({ raw: '' }),
  parseConfigYaml: vi.fn().mockReturnValue({}),
}))

// ---------------------------------------------------------------------------
// Router and pinia setup helpers
// ---------------------------------------------------------------------------

import RoadmapView from '../../web/src/views/project/RoadmapView.vue'

function buildRouter(project = 'testproject') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path: '/p/:project/roadmap',
        component: RoadmapView,
        name: 'roadmap',
      },
    ],
  })
  return router
}

async function mountRoadmap(project = 'testproject') {
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = buildRouter(project)
  await router.push(`/p/${project}/roadmap`)
  await router.isReady()

  const wrapper = mount(RoadmapView, {
    global: {
      plugins: [pinia, router],
    },
  })
  await flushPromises()
  return { wrapper, pinia }
}

// ---------------------------------------------------------------------------
// Milestone 2 — Period-mode selector UI tests
// ---------------------------------------------------------------------------

describe('RoadmapView — Period-mode selector (Milestone 2)', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('M2.1: Gantt view shows the period-mode selector (Autoscale / Fixed Period)', async () => {
    const { wrapper } = await mountRoadmap()

    // The period-mode segmented control group is labelled "Period mode".
    const periodModeGroup = wrapper.find('[aria-label="Period mode"]')
    expect(periodModeGroup.exists()).toBe(true)

    // Both buttons must be present.
    const buttons = periodModeGroup.findAll('button')
    const labels = buttons.map((b) => b.text())
    expect(labels).toContain('Autoscale')
    expect(labels).toContain('Fixed Period')
  })

  it('M2.2: clicking Fixed Period shows the secondary period picker', async () => {
    const { wrapper } = await mountRoadmap()

    // Period picker should not be visible initially (default = autoscale).
    expect(wrapper.find('[aria-label="Fixed period"]').exists()).toBe(false)

    // Click "Fixed Period".
    const periodModeGroup = wrapper.find('[aria-label="Period mode"]')
    const fixedBtn = periodModeGroup.findAll('button').find((b) => b.text() === 'Fixed Period')
    expect(fixedBtn).toBeTruthy()
    await fixedBtn!.trigger('click')
    await flushPromises()

    // Now the fixed-period picker should be visible.
    const picker = wrapper.find('[aria-label="Fixed period"]')
    expect(picker.exists()).toBe(true)
    const pickerLabels = picker.findAll('button').map((b) => b.text())
    expect(pickerLabels).toContain('Month')
    expect(pickerLabels).toContain('Quarter')
    expect(pickerLabels).toContain('Half-Year')
    expect(pickerLabels).toContain('Year')
  })

  it('M2.3: clicking Autoscale hides the secondary period picker', async () => {
    const { wrapper, pinia } = await mountRoadmap()
    const store = useRoadmapSettingsStore(pinia)

    // Start in Fixed Period mode.
    store.periodMode = 'fixed'
    await flushPromises()
    expect(wrapper.find('[aria-label="Fixed period"]').exists()).toBe(true)

    // Click Autoscale.
    const periodModeGroup = wrapper.find('[aria-label="Period mode"]')
    const autoscaleBtn = periodModeGroup.findAll('button').find((b) => b.text() === 'Autoscale')
    await autoscaleBtn!.trigger('click')
    await flushPromises()

    expect(wrapper.find('[aria-label="Fixed period"]').exists()).toBe(false)
  })

  it('M2.4: period-mode selector is hidden in Graph view', async () => {
    const { wrapper } = await mountRoadmap()

    // Switch to Graph view.
    const viewGroup = wrapper.find('[aria-label="View mode"]')
    const graphBtn = viewGroup.findAll('button').find((b) => b.text() === 'Graph')
    await graphBtn!.trigger('click')
    await flushPromises()

    // Period-mode group should no longer be in the DOM.
    expect(wrapper.find('[aria-label="Period mode"]').exists()).toBe(false)
  })

  it('M2.5: switching back to Gantt preserves the period-mode selection', async () => {
    const { wrapper, pinia } = await mountRoadmap()
    const store = useRoadmapSettingsStore(pinia)

    // Set Fixed Period mode.
    store.periodMode = 'fixed'
    store.fixedPeriod = 'quarter'
    await flushPromises()

    // Switch to Graph.
    const viewGroup = wrapper.find('[aria-label="View mode"]')
    await viewGroup.findAll('button').find((b) => b.text() === 'Graph')!.trigger('click')
    await flushPromises()

    // Switch back to Gantt.
    await viewGroup.findAll('button').find((b) => b.text() === 'Gantt')!.trigger('click')
    await flushPromises()

    // Store state should be preserved.
    expect(store.periodMode).toBe('fixed')
    expect(store.fixedPeriod).toBe('quarter')

    // The Fixed Period picker should be visible again.
    expect(wrapper.find('[aria-label="Fixed period"]').exists()).toBe(true)
  })

  it('M7.2: period-mode selector group has role="group" and aria-label="Period mode"', async () => {
    const { wrapper } = await mountRoadmap()
    const group = wrapper.find('[aria-label="Period mode"]')
    expect(group.exists()).toBe(true)
    expect(group.attributes('role')).toBe('group')
  })

  it('M7.2: fixed-period picker group has role="group" and aria-label="Fixed period"', async () => {
    const { wrapper, pinia } = await mountRoadmap()
    const store = useRoadmapSettingsStore(pinia)
    store.periodMode = 'fixed'
    await flushPromises()

    const group = wrapper.find('[aria-label="Fixed period"]')
    expect(group.exists()).toBe(true)
    expect(group.attributes('role')).toBe('group')
  })
})

// ---------------------------------------------------------------------------
// Milestone 8 — Default-from-config and no-extra-API-calls tests
// ---------------------------------------------------------------------------

import * as configApi from '../../web/src/api/config'

describe('roadmapSettings store — loadDefaultPeriodMode (Milestone 8)', () => {
  beforeEach(() => {
    const pinia = createPinia()
    setActivePinia(pinia)
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('M8.1: config default_period_mode "quarter" initialises store to Fixed Period > Quarter', async () => {
    vi.mocked(configApi.getConfig).mockResolvedValueOnce({ raw: 'roadmap:\n  default_period_mode: quarter\n' })
    vi.mocked(configApi.parseConfigYaml).mockReturnValueOnce({
      roadmap: { default_period_mode: 'quarter' },
    })

    const store = useRoadmapSettingsStore()
    await store.loadDefaultPeriodMode('testproject')

    expect(store.periodMode).toBe('fixed')
    expect(store.fixedPeriod).toBe('quarter')
  })

  it('M8.1: config default_period_mode "year" initialises store to Fixed Period > Year', async () => {
    vi.mocked(configApi.getConfig).mockResolvedValueOnce({ raw: 'roadmap:\n  default_period_mode: year\n' })
    vi.mocked(configApi.parseConfigYaml).mockReturnValueOnce({
      roadmap: { default_period_mode: 'year' },
    })

    const store = useRoadmapSettingsStore()
    await store.loadDefaultPeriodMode('testproject')

    expect(store.periodMode).toBe('fixed')
    expect(store.fixedPeriod).toBe('year')
  })

  it('M8.2: no roadmap config → store initialises in Autoscale mode', async () => {
    vi.mocked(configApi.getConfig).mockResolvedValueOnce({ raw: '' })
    vi.mocked(configApi.parseConfigYaml).mockReturnValueOnce({})

    const store = useRoadmapSettingsStore()
    await store.loadDefaultPeriodMode('testproject')

    expect(store.periodMode).toBe('autoscale')
  })

  it('M8.2: config with roadmap section but no default_period_mode → Autoscale', async () => {
    vi.mocked(configApi.getConfig).mockResolvedValueOnce({ raw: 'roadmap: {}\n' })
    vi.mocked(configApi.parseConfigYaml).mockReturnValueOnce({ roadmap: {} })

    const store = useRoadmapSettingsStore()
    await store.loadDefaultPeriodMode('testproject')

    expect(store.periodMode).toBe('autoscale')
  })

  it('M8.3: loadDefaultPeriodMode is idempotent — second call does not overwrite user selection', async () => {
    vi.mocked(configApi.getConfig).mockResolvedValue({ raw: 'roadmap:\n  default_period_mode: quarter\n' })
    vi.mocked(configApi.parseConfigYaml).mockReturnValue({
      roadmap: { default_period_mode: 'quarter' },
    })

    const store = useRoadmapSettingsStore()

    // First call loads the config default.
    await store.loadDefaultPeriodMode('testproject')
    expect(store.periodMode).toBe('fixed')
    expect(store.fixedPeriod).toBe('quarter')

    // User changes the selection.
    store.periodMode = 'autoscale'

    // Second call (e.g., re-mount) must not overwrite the user's selection.
    await store.loadDefaultPeriodMode('testproject')
    expect(store.periodMode).toBe('autoscale')

    // getConfig was only called once (the second call is a no-op).
    expect(configApi.getConfig).toHaveBeenCalledTimes(1)
  })

  it('M8.4: granularity and period mode operate independently — changing fixedPeriod does not reset periodMode', async () => {
    vi.mocked(configApi.getConfig).mockResolvedValueOnce({ raw: '' })
    vi.mocked(configApi.parseConfigYaml).mockReturnValueOnce({})

    const store = useRoadmapSettingsStore()
    await store.loadDefaultPeriodMode('testproject')

    store.periodMode = 'fixed'
    store.fixedPeriod = 'month'

    // Change fixed period — period mode must remain 'fixed'.
    store.fixedPeriod = 'year'
    expect(store.periodMode).toBe('fixed')

    // Change period mode — fixed period must remain 'year'.
    store.periodMode = 'autoscale'
    expect(store.fixedPeriod).toBe('year')
  })

  it('M8.1 edge: "month" config default → Fixed Period > Month', async () => {
    vi.mocked(configApi.getConfig).mockResolvedValueOnce({ raw: 'roadmap:\n  default_period_mode: month\n' })
    vi.mocked(configApi.parseConfigYaml).mockReturnValueOnce({
      roadmap: { default_period_mode: 'month' },
    })

    const store = useRoadmapSettingsStore()
    await store.loadDefaultPeriodMode('testproject')

    expect(store.periodMode).toBe('fixed')
    expect(store.fixedPeriod).toBe('month')
  })

  it('M8.1 edge: "half-year" config default → Fixed Period > Half-Year', async () => {
    vi.mocked(configApi.getConfig).mockResolvedValueOnce({ raw: 'roadmap:\n  default_period_mode: half-year\n' })
    vi.mocked(configApi.parseConfigYaml).mockReturnValueOnce({
      roadmap: { default_period_mode: 'half-year' },
    })

    const store = useRoadmapSettingsStore()
    await store.loadDefaultPeriodMode('testproject')

    expect(store.periodMode).toBe('fixed')
    expect(store.fixedPeriod).toBe('half-year')
  })

  it('M8.1 edge: "autoscale" config default → Autoscale mode (not Fixed)', async () => {
    vi.mocked(configApi.getConfig).mockResolvedValueOnce({ raw: 'roadmap:\n  default_period_mode: autoscale\n' })
    vi.mocked(configApi.parseConfigYaml).mockReturnValueOnce({
      roadmap: { default_period_mode: 'autoscale' },
    })

    const store = useRoadmapSettingsStore()
    await store.loadDefaultPeriodMode('testproject')

    // "autoscale" is not a valid FixedPeriod so store should be in autoscale mode.
    expect(store.periodMode).toBe('autoscale')
  })

  it('M8: config fetch error defaults gracefully to autoscale', async () => {
    vi.mocked(configApi.getConfig).mockRejectedValueOnce(new Error('network error'))

    const store = useRoadmapSettingsStore()
    await store.loadDefaultPeriodMode('testproject')

    // Non-fatal: should default to autoscale.
    expect(store.periodMode).toBe('autoscale')
    // defaultPeriodModeLoaded should still be true so we don't retry on every mount.
    expect(store.defaultPeriodModeLoaded).toBe(true)
  })
})
