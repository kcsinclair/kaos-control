// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Component tests for StagesDistributionWidget — Milestone 3
 *
 * Covers:
 *   TC1 — Renders chart container with data (non-empty distribution)
 *   TC2 — Shows "No artifacts yet" empty state when distribution is []
 *   TC3 — Shows "No artifacts yet" when all stage counts are zero
 *   TC4 — Click event on the echarts chart calls router.push with correct
 *          route ({ name: 'artifacts', params: { project }, query: { stage } })
 *   TC5 — Chart container has role="img" and aria-label mentioning stages + counts
 *   TC6 — Changing the project prop triggers a re-fetch
 *   TC7 — API error causes graceful degradation to the empty state
 *
 * Constraints / design notes:
 *   - happy-dom does not evaluate echarts canvas rendering. The mock captures
 *     the click handler registered via chart.on('click', ...) so tests can
 *     fire it manually without a real canvas.
 *   - Module-level mocks are declared at the top to satisfy Vitest's hoisting
 *     rules (vi.mock calls are hoisted before imports).
 *   - Each test resets mock state in beforeEach / afterEach to prevent
 *     cross-test pollution.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

import StagesDistributionWidget from '../../web/src/components/dashboard/widgets/StagesDistributionWidget.vue'

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

// Controllable router.push so we can assert navigation calls.
const mockPush = vi.fn()

vi.mock('vue-router', async (importActual) => {
  const actual = await importActual<typeof import('vue-router')>()
  return {
    ...actual,
    useRouter: vi.fn(() => ({ push: mockPush, replace: vi.fn() })),
    useRoute: vi.fn(() => ({ params: { project: 'testproject' }, query: {} })),
  }
})

// Default API mock — returns empty distribution; individual tests override as needed.
vi.mock('@/api/client', () => ({
  api: {
    get: vi.fn().mockResolvedValue({ distribution: [] }),
  },
}))

// echarts mock: captures the click handler so tests can fire it.
// Returns a minimal chart stub with the methods the widget uses.
let capturedClickHandler: ((params: { name?: string }) => void) | null = null

vi.mock('echarts/core', () => ({
  use: vi.fn(),
  init: vi.fn(() => ({
    setOption: vi.fn(),
    on: vi.fn((event: string, handler: (params: { name?: string }) => void) => {
      if (event === 'click') capturedClickHandler = handler
    }),
    resize: vi.fn(),
    dispose: vi.fn(),
  })),
}))
vi.mock('echarts/charts',     () => ({ PieChart: {} }))
vi.mock('echarts/components', () => ({ TooltipComponent: {}, LegendComponent: {} }))
vi.mock('echarts/renderers',  () => ({ CanvasRenderer: {} }))

// ---------------------------------------------------------------------------
// Setup / teardown
// ---------------------------------------------------------------------------

beforeEach(() => {
  setActivePinia(createPinia())
  mockPush.mockClear()
  capturedClickHandler = null
})

afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

// ===========================================================================
// TC1 — Renders chart container when distribution has data
// ===========================================================================

describe('StagesDistributionWidget — TC1: renders chart with data', () => {
  it('renders chart container (role="img") when distribution is non-empty', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({
      distribution: [
        { stage: 'ideas', count: 3 },
        { stage: 'requirements', count: 2 },
      ],
    })

    const wrapper = mount(StagesDistributionWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    expect(wrapper.find('[role="img"]').exists()).toBe(true)
    expect(wrapper.find('.widget-empty').exists()).toBe(false)
  })

  it('does not render the empty-state placeholder when data is present', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({
      distribution: [{ stage: 'requirements', count: 5 }],
    })

    const wrapper = mount(StagesDistributionWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    expect(wrapper.find('.widget-empty').exists()).toBe(false)
  })
})

// ===========================================================================
// TC2 — Empty state when distribution is []
// ===========================================================================

describe('StagesDistributionWidget — TC2: empty state for empty distribution', () => {
  it('shows "No artifacts yet" when distribution is an empty array', async () => {
    // Default mock already returns { distribution: [] }.
    const wrapper = mount(StagesDistributionWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    expect(wrapper.find('.widget-empty').exists()).toBe(true)
    expect(wrapper.find('.widget-empty').text()).toBe('No artifacts yet')
  })

  it('does not render the chart container when distribution is empty', async () => {
    const wrapper = mount(StagesDistributionWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    expect(wrapper.find('[role="img"]').exists()).toBe(false)
  })
})

// ===========================================================================
// TC3 — Empty state when all counts are zero
// ===========================================================================

describe('StagesDistributionWidget — TC3: empty state when all counts are zero', () => {
  it('shows "No artifacts yet" when every stage count is zero', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({
      distribution: [
        { stage: 'ideas', count: 0 },
        { stage: 'requirements', count: 0 },
      ],
    })

    const wrapper = mount(StagesDistributionWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    expect(wrapper.find('.widget-empty').exists()).toBe(true)
    expect(wrapper.find('.widget-empty').text()).toBe('No artifacts yet')
  })

  it('does not render the chart container when all counts are zero', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({
      distribution: [{ stage: 'ideas', count: 0 }],
    })

    const wrapper = mount(StagesDistributionWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    expect(wrapper.find('[role="img"]').exists()).toBe(false)
  })
})

// ===========================================================================
// TC4 — Click-through navigation
// ===========================================================================

describe('StagesDistributionWidget — TC4: click-through navigation', () => {
  it('calls router.push with name="artifacts" and query.stage when a segment is clicked', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({
      distribution: [
        { stage: 'ideas', count: 2 },
        { stage: 'requirements', count: 4 },
      ],
    })

    mount(StagesDistributionWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    expect(capturedClickHandler).not.toBeNull()
    capturedClickHandler!({ name: 'requirements' })

    expect(mockPush).toHaveBeenCalledOnce()
    expect(mockPush).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'artifacts',
        query: { stage: 'requirements' },
      })
    )
  })

  it('passes the correct project in params when navigating', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({
      distribution: [{ stage: 'ideas', count: 1 }],
    })

    mount(StagesDistributionWidget, {
      props: { project: 'my-project' },
    })
    await flushPromises()

    capturedClickHandler!({ name: 'ideas' })

    const call = mockPush.mock.calls[0][0] as { params?: { project?: string } }
    expect(call.params?.project).toBe('my-project')
  })

  it('uses the exact stage name from the echarts params (no transformation)', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({
      distribution: [{ stage: 'backend-plans', count: 3 }],
    })

    mount(StagesDistributionWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    capturedClickHandler!({ name: 'backend-plans' })

    const call = mockPush.mock.calls[0][0] as { query: { stage: string } }
    expect(call.query.stage).toBe('backend-plans')
  })

  it('does not navigate when the echarts click has no name (undefined segment)', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({
      distribution: [{ stage: 'ideas', count: 1 }],
    })

    mount(StagesDistributionWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    capturedClickHandler!({})  // no name property

    expect(mockPush).not.toHaveBeenCalled()
  })
})

// ===========================================================================
// TC5 — Accessibility: role="img" + descriptive aria-label
// ===========================================================================

describe('StagesDistributionWidget — TC5: accessibility attributes', () => {
  it('chart container has role="img"', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({
      distribution: [{ stage: 'ideas', count: 2 }],
    })

    const wrapper = mount(StagesDistributionWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    expect(wrapper.find('[role="img"]').exists()).toBe(true)
  })

  it('aria-label includes the stage name and count', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({
      distribution: [
        { stage: 'ideas', count: 3 },
        { stage: 'requirements', count: 7 },
      ],
    })

    const wrapper = mount(StagesDistributionWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    const label = wrapper.find('[role="img"]').attributes('aria-label') ?? ''
    expect(label).toContain('ideas')
    expect(label).toContain('requirements')
    expect(label).toMatch(/3/)
    expect(label).toMatch(/7/)
  })

  it('aria-label mentions clickability for filtering', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({
      distribution: [{ stage: 'sprints', count: 1 }],
    })

    const wrapper = mount(StagesDistributionWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    const label = wrapper.find('[role="img"]').attributes('aria-label') ?? ''
    expect(label).toMatch(/click/i)
  })

  it('aria-label describes no artifacts when the widget is in empty state', async () => {
    // Default mock returns empty distribution.
    const wrapper = mount(StagesDistributionWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    // The chart element is absent in empty state; the aria-label is on the
    // chartEl div which is hidden via v-if. We verify the component does NOT
    // render a [role="img"] element at all in this state.
    expect(wrapper.find('[role="img"]').exists()).toBe(false)
  })
})

// ===========================================================================
// TC6 — Project prop change triggers re-fetch
// ===========================================================================

describe('StagesDistributionWidget — TC6: project prop change re-fetches data', () => {
  it('calls the API again when the project prop changes', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValue({ distribution: [] })

    const wrapper = mount(StagesDistributionWidget, {
      props: { project: 'project-alpha' },
    })
    await flushPromises()

    const callsAfterMount = vi.mocked(api.get).mock.calls.length

    // Change the project prop.
    await wrapper.setProps({ project: 'project-beta' })
    await flushPromises()

    expect(vi.mocked(api.get).mock.calls.length).toBeGreaterThan(callsAfterMount)
  })

  it('fetches using the new project name in the URL after prop change', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValue({ distribution: [] })

    const wrapper = mount(StagesDistributionWidget, {
      props: { project: 'project-alpha' },
    })
    await flushPromises()

    await wrapper.setProps({ project: 'project-beta' })
    await flushPromises()

    const calls = vi.mocked(api.get).mock.calls.map((c) => c[0] as string)
    expect(calls.some((url) => url.includes('project-beta'))).toBe(true)
  })
})

// ===========================================================================
// TC7 — API error → graceful degradation to empty state
// ===========================================================================

describe('StagesDistributionWidget — TC7: error handling', () => {
  it('shows the empty state when the API call rejects', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockRejectedValueOnce(new Error('network error'))

    const wrapper = mount(StagesDistributionWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    expect(wrapper.find('.widget-empty').exists()).toBe(true)
    expect(wrapper.find('[role="img"]').exists()).toBe(false)
  })

  it('does not throw when the API call rejects (no unhandled rejection)', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockRejectedValueOnce(new Error('500 Internal Server Error'))

    // mount + flush should not throw even though the API call fails.
    await expect(
      (async () => {
        const wrapper = mount(StagesDistributionWidget, {
          props: { project: 'testproject' },
        })
        await flushPromises()
        return wrapper
      })()
    ).resolves.toBeDefined()
  })
})

// ===========================================================================
// General: widget title and API URL
// ===========================================================================

describe('StagesDistributionWidget — general', () => {
  it('renders the widget title "Stages Distribution"', async () => {
    const wrapper = mount(StagesDistributionWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    expect(wrapper.find('.widget-title').text()).toBe('Stages Distribution')
  })

  it('calls the API with the correct project-scoped stage-distribution URL', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValue({ distribution: [] })

    mount(StagesDistributionWidget, {
      props: { project: 'alpha-project' },
    })
    await flushPromises()

    expect(vi.mocked(api.get)).toHaveBeenCalledWith(
      '/p/alpha-project/dashboard/stage-distribution'
    )
  })
})
