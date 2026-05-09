/**
 * Component tests for dashboard clickable filters feature.
 *
 * Covers:
 *   Milestone 3 — SummaryCountCard click-through behaviour
 *     - Lifecycle Total card navigates to artifacts list with no filter
 *     - Blocked card navigates to artifacts list filtered by status=blocked
 *     - In Progress card is not interactive (no cursor:pointer, no navigation)
 *     - Completed This Week card is not interactive
 *     - Keyboard activation (Enter / Space) triggers navigation on interactive cards
 *
 *   Milestone 4 — StatusDistributionWidget chart ARIA
 *     - Chart container has role="img" and aria-label mentioning clickability
 *     - Router.push is called with correct status when chart click event fires
 *
 *   Milestone 5 — Accessibility
 *     - Interactive cards carry role="link", tabindex="0", aria-label matching
 *       the pattern /view \d+ .* artifacts/i
 *     - Non-interactive cards carry role="figure" (not role="link")
 *     - Interactive cards have visible focus ring via .summary-card--interactive class
 *
 * Notes on scope:
 *   Back-button behaviour (M3-TC5) and CSS @media layout tests require a real
 *   browser; they are deferred to a Playwright E2E suite.
 *   Cursor-pointer style assertions (M3-TC3, M3-TC4, M4-TC3) rely on CSS
 *   evaluation which happy-dom does not provide; those are covered by the
 *   .summary-card--interactive class assertions instead.
 *   Velocity chart granularity toggle (M6-TC4) requires a real canvas renderer
 *   and is deferred to the Playwright suite.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

import SummaryCountCard from '../../web/src/components/dashboard/widgets/SummaryCountCard.vue'
import SummaryCountsWidget from '../../web/src/components/dashboard/widgets/SummaryCountsWidget.vue'
import StatusDistributionWidget from '../../web/src/components/dashboard/widgets/StatusDistributionWidget.vue'

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

// Mock vue-router at the top level so useRouter() returns a controllable push.
const mockPush = vi.fn()

vi.mock('vue-router', async (importActual) => {
  const actual = await importActual<typeof import('vue-router')>()
  return {
    ...actual,
    useRouter: vi.fn(() => ({ push: mockPush, replace: vi.fn() })),
    useRoute:  vi.fn(() => ({ params: { project: 'testproject' }, query: {} })),
  }
})

vi.mock('@/api/client', () => ({
  api: {
    get: vi.fn().mockResolvedValue({
      total_tickets: 0,
      in_progress: 0,
      blocked: 0,
      completed_this_week: 0,
    }),
  },
}))

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: vi.fn(),
}))

// Mock echarts so StatusDistributionWidget can mount without a real canvas.
// The mock returns a minimal chart object that captures the click handler so
// tests can fire it manually.
let capturedClickHandler: ((params: { name?: string }) => void) | null = null

vi.mock('echarts/core', () => ({
  use:  vi.fn(),
  init: vi.fn(() => ({
    setOption: vi.fn(),
    on: vi.fn((event: string, handler: (params: { name?: string }) => void) => {
      if (event === 'click') capturedClickHandler = handler
    }),
    resize:  vi.fn(),
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
// Milestone 3 — SummaryCountCard: interactive behaviour
// ===========================================================================

describe('SummaryCountCard — interactive (to prop provided)', () => {
  const interactiveTo = { name: 'artifacts', params: { project: 'testproject' }, query: { status: 'blocked' } }

  it('renders with role="link" when a to prop is provided', () => {
    const wrapper = mount(SummaryCountCard, {
      props: { label: 'Blocked', value: 3, to: interactiveTo },
    })
    expect(wrapper.attributes('role')).toBe('link')
  })

  it('has tabindex="0" when a to prop is provided (keyboard focusable)', () => {
    const wrapper = mount(SummaryCountCard, {
      props: { label: 'Blocked', value: 3, to: interactiveTo },
    })
    expect(wrapper.attributes('tabindex')).toBe('0')
  })

  it('applies summary-card--interactive class when a to prop is provided', () => {
    // The CSS class drives cursor:pointer. happy-dom does not evaluate CSS,
    // so we assert the class is present as a proxy for cursor behaviour.
    const wrapper = mount(SummaryCountCard, {
      props: { label: 'Blocked', value: 3, to: interactiveTo },
    })
    expect(wrapper.classes()).toContain('summary-card--interactive')
  })

  it('calls router.push with the to prop value when clicked', async () => {
    const wrapper = mount(SummaryCountCard, {
      props: { label: 'Blocked', value: 3, to: interactiveTo },
    })
    await wrapper.trigger('click')
    expect(mockPush).toHaveBeenCalledOnce()
    expect(mockPush).toHaveBeenCalledWith(interactiveTo)
  })

  it('calls router.push when Enter key is pressed', async () => {
    const wrapper = mount(SummaryCountCard, {
      props: { label: 'Blocked', value: 3, to: interactiveTo },
    })
    await wrapper.trigger('keydown.enter')
    expect(mockPush).toHaveBeenCalledOnce()
    expect(mockPush).toHaveBeenCalledWith(interactiveTo)
  })

  it('calls router.push when Space key is pressed', async () => {
    const wrapper = mount(SummaryCountCard, {
      props: { label: 'Blocked', value: 3, to: interactiveTo },
    })
    await wrapper.trigger('keydown.space')
    expect(mockPush).toHaveBeenCalledOnce()
    expect(mockPush).toHaveBeenCalledWith(interactiveTo)
  })

  it('sets aria-label to "View N label artifacts" pattern', () => {
    const wrapper = mount(SummaryCountCard, {
      props: { label: 'Blocked', value: 5, to: interactiveTo },
    })
    const label = wrapper.attributes('aria-label') ?? ''
    expect(label).toMatch(/view 5 blocked artifacts/i)
  })
})

// ===========================================================================
// Milestone 3 + Milestone 5 — SummaryCountCard: non-interactive behaviour
// ===========================================================================

describe('SummaryCountCard — non-interactive (to=null)', () => {
  it('renders with role="figure" when to is null', () => {
    const wrapper = mount(SummaryCountCard, {
      props: { label: 'In Progress', value: 2, to: null },
    })
    expect(wrapper.attributes('role')).toBe('figure')
  })

  it('has no tabindex when to is null', () => {
    const wrapper = mount(SummaryCountCard, {
      props: { label: 'In Progress', value: 2, to: null },
    })
    expect(wrapper.attributes('tabindex')).toBeUndefined()
  })

  it('does not apply summary-card--interactive class when to is null', () => {
    const wrapper = mount(SummaryCountCard, {
      props: { label: 'In Progress', value: 2, to: null },
    })
    expect(wrapper.classes()).not.toContain('summary-card--interactive')
  })

  it('does not navigate when clicked', async () => {
    const wrapper = mount(SummaryCountCard, {
      props: { label: 'In Progress', value: 2, to: null },
    })
    await wrapper.trigger('click')
    expect(mockPush).not.toHaveBeenCalled()
  })

  it('does not navigate when Enter is pressed', async () => {
    const wrapper = mount(SummaryCountCard, {
      props: { label: 'In Progress', value: 2, to: null },
    })
    await wrapper.trigger('keydown.enter')
    expect(mockPush).not.toHaveBeenCalled()
  })

  it('sets aria-label to "label: value" pattern (not "view N")', () => {
    const wrapper = mount(SummaryCountCard, {
      props: { label: 'In Progress', value: 4, to: null },
    })
    const label = wrapper.attributes('aria-label') ?? ''
    expect(label).toMatch(/in progress.*4/i)
    expect(label).not.toMatch(/view/i)
  })
})

// ===========================================================================
// Milestone 3 — SummaryCountsWidget: click-through configuration
// ===========================================================================

describe('SummaryCountsWidget — click-through routing configuration', () => {
  it('Lifecycle Total card has role="link" (to prop set)', async () => {
    const wrapper = mount(SummaryCountsWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    // The Lifecycle Total card is the first SummaryCountCard rendered.
    const cards = wrapper.findAllComponents(SummaryCountCard)
    expect(cards.length).toBeGreaterThanOrEqual(1)
    const totalCard = cards[0]
    expect(totalCard.attributes('role')).toBe('link')
  })

  it('Blocked card has role="link" (to prop set)', async () => {
    const wrapper = mount(SummaryCountsWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    // Card order: Lifecycle Total, In Progress, Blocked, Completed This Week
    const cards = wrapper.findAllComponents(SummaryCountCard)
    expect(cards.length).toBe(4)
    const blockedCard = cards[2]
    expect(blockedCard.attributes('role')).toBe('link')
  })

  it('Blocked card navigates to artifacts?status=blocked when clicked', async () => {
    const wrapper = mount(SummaryCountsWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    const cards = wrapper.findAllComponents(SummaryCountCard)
    const blockedCard = cards[2]
    await blockedCard.trigger('click')

    expect(mockPush).toHaveBeenCalledOnce()
    const call = mockPush.mock.calls[0][0]
    expect(call).toMatchObject({ query: { status: 'blocked' } })
  })

  it('Lifecycle Total card navigates to artifacts with no status filter when clicked', async () => {
    const wrapper = mount(SummaryCountsWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    const cards = wrapper.findAllComponents(SummaryCountCard)
    const totalCard = cards[0]
    await totalCard.trigger('click')

    expect(mockPush).toHaveBeenCalledOnce()
    const call = mockPush.mock.calls[0][0]
    // No status key in query (or empty query object)
    const query = (call as { query?: Record<string, string> }).query ?? {}
    expect(query.status).toBeUndefined()
  })

  it('In Progress card has role="figure" (not interactive — M3-TC3)', async () => {
    const wrapper = mount(SummaryCountsWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    const cards = wrapper.findAllComponents(SummaryCountCard)
    const inProgressCard = cards[1]
    expect(inProgressCard.attributes('role')).toBe('figure')
  })

  it('In Progress card does not navigate when clicked (M3-TC3)', async () => {
    const wrapper = mount(SummaryCountsWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    const cards = wrapper.findAllComponents(SummaryCountCard)
    const inProgressCard = cards[1]
    await inProgressCard.trigger('click')
    expect(mockPush).not.toHaveBeenCalled()
  })

  it('Completed This Week card has role="figure" (not interactive — M3-TC4)', async () => {
    const wrapper = mount(SummaryCountsWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    const cards = wrapper.findAllComponents(SummaryCountCard)
    const ctw = cards[3]
    expect(ctw.attributes('role')).toBe('figure')
  })

  it('Completed This Week card does not navigate when clicked (M3-TC4)', async () => {
    const wrapper = mount(SummaryCountsWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    const cards = wrapper.findAllComponents(SummaryCountCard)
    const ctw = cards[3]
    await ctw.trigger('click')
    expect(mockPush).not.toHaveBeenCalled()
  })
})

// ===========================================================================
// Milestone 4 — StatusDistributionWidget: ARIA and click navigation
// ===========================================================================

describe('StatusDistributionWidget — chart ARIA and click navigation', () => {
  it('chart container has role="img" (M4 + M5-TC5)', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({
      distribution: [{ status: 'draft', count: 2 }],
    })

    const wrapper = mount(StatusDistributionWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    const chart = wrapper.find('[role="img"]')
    expect(chart.exists()).toBe(true)
  })

  it('chart container aria-label mentions clickability (M5-TC5)', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({
      distribution: [{ status: 'draft', count: 2 }],
    })

    const wrapper = mount(StatusDistributionWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    const chart = wrapper.find('[role="img"]')
    const label = chart.attributes('aria-label') ?? ''
    expect(label).toMatch(/click/i)
  })

  it('clicking a pie segment navigates to artifacts?status=<status> (M4-TC1)', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({
      distribution: [
        { status: 'draft', count: 2 },
        { status: 'blocked', count: 1 },
      ],
    })

    mount(StatusDistributionWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    // Fire the echarts click handler captured during chart initialisation.
    expect(capturedClickHandler).not.toBeNull()
    capturedClickHandler!({ name: 'draft' })

    expect(mockPush).toHaveBeenCalledOnce()
    expect(mockPush).toHaveBeenCalledWith(
      expect.objectContaining({ query: { status: 'draft' } })
    )
  })

  it('clicking a different segment navigates to the correct status (M4-TC2)', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({
      distribution: [
        { status: 'draft', count: 2 },
        { status: 'blocked', count: 1 },
      ],
    })

    mount(StatusDistributionWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    expect(capturedClickHandler).not.toBeNull()
    capturedClickHandler!({ name: 'blocked' })

    expect(mockPush).toHaveBeenCalledOnce()
    expect(mockPush).toHaveBeenCalledWith(
      expect.objectContaining({ query: { status: 'blocked' } })
    )
  })

  it('status value in navigation URL exactly matches the segment name (no case mismatch)', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({
      distribution: [{ status: 'in-development', count: 3 }],
    })

    mount(StatusDistributionWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    expect(capturedClickHandler).not.toBeNull()
    capturedClickHandler!({ name: 'in-development' })

    expect(mockPush).toHaveBeenCalledOnce()
    const call = mockPush.mock.calls[0][0] as { query: { status: string } }
    // Must be exact — no display-label leakage (e.g. "In Development" or "IN-DEVELOPMENT")
    expect(call.query.status).toBe('in-development')
  })

  it('shows empty state when distribution is empty (no chart, no navigation target)', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({ distribution: [] })

    const wrapper = mount(StatusDistributionWidget, {
      props: { project: 'testproject' },
    })
    await flushPromises()

    expect(wrapper.find('[role="img"]').exists()).toBe(false)
    expect(wrapper.find('.widget-empty').exists()).toBe(true)
  })
})

// ===========================================================================
// Milestone 5 — Accessibility: ARIA attributes on interactive dashboard cards
// ===========================================================================

describe('Accessibility — interactive card ARIA attributes (M5)', () => {
  it('interactive card has aria-label matching /view \\d+ .* artifacts/i (M5-TC3)', () => {
    const to = { name: 'artifacts', params: { project: 'testproject' }, query: {} }
    const wrapper = mount(SummaryCountCard, {
      props: { label: 'Lifecycle Total', value: 42, to },
    })
    const label = wrapper.attributes('aria-label') ?? ''
    expect(label).toMatch(/view \d+ .* artifacts/i)
    expect(label).toContain('42')
  })

  it('non-interactive card does not have role="link" (M5-TC4 — In Progress)', () => {
    const wrapper = mount(SummaryCountCard, {
      props: { label: 'In Progress', value: 2, to: null },
    })
    expect(wrapper.attributes('role')).not.toBe('link')
  })

  it('non-interactive card does not have role="link" (M5-TC4 — Completed This Week)', () => {
    const wrapper = mount(SummaryCountCard, {
      props: { label: 'Completed This Week', value: 5, to: null },
    })
    expect(wrapper.attributes('role')).not.toBe('link')
  })

  it('Lifecycle Total keyboard activation: Enter navigates to artifacts (M5-TC1)', async () => {
    const to = { name: 'artifacts', params: { project: 'testproject' }, query: {} }
    const wrapper = mount(SummaryCountCard, {
      props: { label: 'Lifecycle Total', value: 10, to },
    })
    await wrapper.trigger('keydown.enter')
    expect(mockPush).toHaveBeenCalledOnce()
    expect(mockPush).toHaveBeenCalledWith(to)
  })

  it('Blocked card keyboard activation: Space navigates to filtered list (M5-TC2)', async () => {
    const to = { name: 'artifacts', params: { project: 'testproject' }, query: { status: 'blocked' } }
    const wrapper = mount(SummaryCountCard, {
      props: { label: 'Blocked', value: 3, to },
    })
    await wrapper.trigger('keydown.space')
    expect(mockPush).toHaveBeenCalledOnce()
    expect(mockPush).toHaveBeenCalledWith(to)
  })

  it('interactive card has focus-ring class (summary-card--interactive) for visual focus ring', () => {
    // The .summary-card--interactive:focus-visible selector provides the focus ring.
    // happy-dom does not evaluate CSS, so we assert the class is present as a proxy.
    const to = { name: 'artifacts', params: { project: 'testproject' }, query: {} }
    const wrapper = mount(SummaryCountCard, {
      props: { label: 'Lifecycle Total', value: 7, to },
    })
    expect(wrapper.classes()).toContain('summary-card--interactive')
  })
})

// ===========================================================================
// Milestone 6 — Regression: WebSocket-triggered stats refetch (M6-TC5)
// ===========================================================================

describe('SummaryCountsWidget — WebSocket-triggered stats refetch (M6-TC5)', () => {
  it('calls the stats API on mount', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValue({
      total_tickets: 0, in_progress: 0, blocked: 0, completed_this_week: 0,
    })

    mount(SummaryCountsWidget, { props: { project: 'testproject' } })
    await flushPromises()

    expect(api.get).toHaveBeenCalledWith('/p/testproject/dashboard/stats')
  })

  it('registers a WebSocket listener for artifact.indexed events', async () => {
    const { useWebSocket } = await import('@/composables/useWebSocket' as any)

    mount(SummaryCountsWidget, { props: { project: 'testproject' } })

    expect(vi.mocked(useWebSocket)).toHaveBeenCalledWith(
      'testproject',
      'artifact.indexed',
      expect.any(Function)
    )
  })

  it('refetches stats when the artifact.indexed WebSocket handler is invoked', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValue({
      total_tickets: 0, in_progress: 0, blocked: 0, completed_this_week: 0,
    })

    const { useWebSocket } = await import('@/composables/useWebSocket' as any)
    let capturedWsHandler: ((e: unknown) => void) | null = null
    vi.mocked(useWebSocket).mockImplementationOnce(
      (_project: string, _event: string, handler: (e: unknown) => void) => {
        capturedWsHandler = handler
      }
    )

    mount(SummaryCountsWidget, { props: { project: 'testproject' } })
    await flushPromises()

    const callCountAfterMount = vi.mocked(api.get).mock.calls.length

    // Simulate an artifact.indexed WebSocket event.
    expect(capturedWsHandler).not.toBeNull()
    capturedWsHandler!({})
    await flushPromises()

    expect(vi.mocked(api.get).mock.calls.length).toBeGreaterThan(callCountAfterMount)
  })
})
