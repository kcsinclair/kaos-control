// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Component tests for RecentIdeasDefectsWidget — Milestone 4
 *
 * Covers:
 *   TC1 — Renders items: 4 items → widget displays 4 entries with correct
 *          titles, type badges, and timestamps
 *   TC2 — Empty state: 0 items → "No recent ideas or defects" message
 *   TC3 — Navigation: router-link points to /p/{project}/artifacts/{path}
 *   TC4 — Type badges: correct badge text and CSS class for idea vs defect
 *   TC5 — Live update: artifact.indexed WebSocket event triggers re-fetch
 *   TC6 — Accessibility: items are focusable; type badges have aria-label
 *
 * Design notes:
 *   - The component uses listArtifacts() from @/api/artifacts, so we mock
 *     that module rather than @/api/client directly.
 *   - useWebSocket is mocked to capture the registered callback so TC5 can
 *     fire it manually and assert that a second fetch is triggered.
 *   - router-link navigation (TC3) is verified by inspecting the rendered
 *     <a> href attribute after mounting with a real memory-history router.
 *   - Keyboard focusability (TC6) relies on <a> elements being natively
 *     focusable — no additional tabindex check needed beyond the link existing.
 *   - CSS @media assertions (responsive collapse) require Playwright and are
 *     deferred to an E2E suite (happy-dom does not evaluate CSS rules).
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'

import RecentIdeasDefectsWidget from '../../web/src/components/dashboard/widgets/RecentIdeasDefectsWidget.vue'

// ---------------------------------------------------------------------------
// Module mocks — must be at top level (vi.mock calls are hoisted)
// ---------------------------------------------------------------------------

// Capture the WebSocket callback so TC5 can fire it.
let capturedWsCallback: ((e: any) => void) | null = null

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: vi.fn((_project: string, _event: string, cb: (e: any) => void) => {
    capturedWsCallback = cb
  }),
}))

// Default mock: empty list. Individual tests override as needed.
vi.mock('@/api/artifacts', () => ({
  listArtifacts: vi.fn().mockResolvedValue({ items: [], total: 0 }),
}))

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeRouter(path = '/p/testproject/dashboard') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/p/:project/dashboard', component: { template: '<div />' } },
      { path: '/p/:project/artifacts/:pathMatch(.*)*', component: { template: '<div />' } },
      { path: '/:pathMatch(.*)*', component: { template: '<div />' } },
    ],
  })
  router.push(path)
  return router
}

function makeItem(overrides: Partial<{
  path: string
  title: string
  type: string
  created: string
  slug: string
  lineage: string
  status: string
}> = {}) {
  return {
    path: 'lifecycle/ideas/test-idea.md',
    title: 'Test Idea',
    type: 'idea',
    slug: 'test-idea',
    lineage: 'test-idea',
    index: 0,
    stage: 'ideas',
    status: 'draft',
    created: new Date().toISOString(),
    mtime: new Date().toISOString(),
    frontmatter: {},
    ...overrides,
  }
}

// ---------------------------------------------------------------------------
// Setup / teardown
// ---------------------------------------------------------------------------

beforeEach(() => {
  setActivePinia(createPinia())
  capturedWsCallback = null
})

afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

// ===========================================================================
// TC1 — Renders items with correct titles, type badges, and timestamps
// ===========================================================================

describe('RecentIdeasDefectsWidget — TC1: renders items', () => {
  it('displays the correct number of items returned by the API', async () => {
    const { listArtifacts } = await import('@/api/artifacts' as any)
    vi.mocked(listArtifacts).mockResolvedValueOnce({
      items: [
        makeItem({ title: 'Alpha Idea',   type: 'idea',   path: 'lifecycle/ideas/alpha.md' }),
        makeItem({ title: 'Beta Defect',  type: 'defect', path: 'lifecycle/defects/beta.md' }),
        makeItem({ title: 'Gamma Idea',   type: 'idea',   path: 'lifecycle/ideas/gamma.md' }),
        makeItem({ title: 'Delta Defect', type: 'defect', path: 'lifecycle/defects/delta.md' }),
      ],
      total: 4,
    })

    const wrapper = mount(RecentIdeasDefectsWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    const listItems = wrapper.findAll('[role="listitem"]')
    expect(listItems).toHaveLength(4)
  })

  it('renders each item title in the DOM', async () => {
    const { listArtifacts } = await import('@/api/artifacts' as any)
    vi.mocked(listArtifacts).mockResolvedValueOnce({
      items: [
        makeItem({ title: 'First Idea',  type: 'idea' }),
        makeItem({ title: 'Second Defect', type: 'defect', path: 'lifecycle/defects/second.md' }),
      ],
      total: 2,
    })

    const wrapper = mount(RecentIdeasDefectsWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('First Idea')
    expect(wrapper.text()).toContain('Second Defect')
  })

  it('renders a relative timestamp for each item', async () => {
    const recentDate = new Date(Date.now() - 60_000).toISOString() // 1 minute ago
    const { listArtifacts } = await import('@/api/artifacts' as any)
    vi.mocked(listArtifacts).mockResolvedValueOnce({
      items: [makeItem({ title: 'Timed Item', created: recentDate })],
      total: 1,
    })

    const wrapper = mount(RecentIdeasDefectsWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    // The item-time element must be non-empty (relativeTime produces a string).
    const timers = wrapper.findAll('.item-time')
    expect(timers).toHaveLength(1)
    expect(timers[0].text()).not.toBe('')
  })
})

// ===========================================================================
// TC2 — Empty state when API returns 0 items
// ===========================================================================

describe('RecentIdeasDefectsWidget — TC2: empty state', () => {
  it('shows "No recent ideas or defects" when API returns empty list', async () => {
    // Default mock already returns { items: [], total: 0 }.
    const wrapper = mount(RecentIdeasDefectsWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    expect(wrapper.find('.empty-state').exists()).toBe(true)
    expect(wrapper.find('.empty-state').text()).toBe('No recent ideas or defects')
  })

  it('does not render the item list when empty', async () => {
    const wrapper = mount(RecentIdeasDefectsWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    expect(wrapper.find('[role="list"]').exists()).toBe(false)
  })

  it('shows empty state when API rejects (error handling)', async () => {
    const { listArtifacts } = await import('@/api/artifacts' as any)
    vi.mocked(listArtifacts).mockRejectedValueOnce(new Error('network error'))

    const wrapper = mount(RecentIdeasDefectsWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    expect(wrapper.find('.empty-state').exists()).toBe(true)
  })
})

// ===========================================================================
// TC3 — Navigation: router-link points to the correct artifact path
// ===========================================================================

describe('RecentIdeasDefectsWidget — TC3: navigation links', () => {
  it('item link href resolves to /p/{project}/artifacts/{path}', async () => {
    const { listArtifacts } = await import('@/api/artifacts' as any)
    vi.mocked(listArtifacts).mockResolvedValueOnce({
      items: [makeItem({
        title: 'Nav Item',
        path: 'lifecycle/ideas/nav-item.md',
      })],
      total: 1,
    })

    const router = makeRouter()
    const wrapper = mount(RecentIdeasDefectsWidget, {
      props: { project: 'testproject' },
      global: { plugins: [router] },
    })
    await flushPromises()
    await router.isReady()

    const link = wrapper.find('a.item-link')
    expect(link.exists()).toBe(true)
    // router-link resolves href based on the to prop.
    const href = link.attributes('href')
    expect(href).toBe('/p/testproject/artifacts/lifecycle/ideas/nav-item.md')
  })

  it('each item generates a unique link based on its path', async () => {
    const { listArtifacts } = await import('@/api/artifacts' as any)
    vi.mocked(listArtifacts).mockResolvedValueOnce({
      items: [
        makeItem({ path: 'lifecycle/ideas/first.md',   title: 'First' }),
        makeItem({ path: 'lifecycle/ideas/second.md',  title: 'Second', lineage: 'second', slug: 'second' }),
      ],
      total: 2,
    })

    const router = makeRouter()
    const wrapper = mount(RecentIdeasDefectsWidget, {
      props: { project: 'testproject' },
      global: { plugins: [router] },
    })
    await flushPromises()
    await router.isReady()

    const links = wrapper.findAll('a.item-link')
    expect(links).toHaveLength(2)
    expect(links[0].attributes('href')).toContain('first.md')
    expect(links[1].attributes('href')).toContain('second.md')
  })
})

// ===========================================================================
// TC4 — Type badges: correct text and CSS class
// ===========================================================================

describe('RecentIdeasDefectsWidget — TC4: type badges', () => {
  it('idea item renders a badge with text "idea" and class type-badge--idea', async () => {
    const { listArtifacts } = await import('@/api/artifacts' as any)
    vi.mocked(listArtifacts).mockResolvedValueOnce({
      items: [makeItem({ type: 'idea', title: 'Badge Idea' })],
      total: 1,
    })

    const wrapper = mount(RecentIdeasDefectsWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    const badge = wrapper.find('.type-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toBe('idea')
    expect(badge.classes()).toContain('type-badge--idea')
  })

  it('defect item renders a badge with text "defect" and class type-badge--defect', async () => {
    const { listArtifacts } = await import('@/api/artifacts' as any)
    vi.mocked(listArtifacts).mockResolvedValueOnce({
      items: [makeItem({ type: 'defect', title: 'Badge Defect', path: 'lifecycle/defects/badge.md' })],
      total: 1,
    })

    const wrapper = mount(RecentIdeasDefectsWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    const badge = wrapper.find('.type-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toBe('defect')
    expect(badge.classes()).toContain('type-badge--defect')
  })

  it('mixed list shows each badge with the correct class for its type', async () => {
    const { listArtifacts } = await import('@/api/artifacts' as any)
    vi.mocked(listArtifacts).mockResolvedValueOnce({
      items: [
        makeItem({ type: 'idea',   title: 'An Idea',   path: 'lifecycle/ideas/mix-idea.md' }),
        makeItem({ type: 'defect', title: 'A Defect',  path: 'lifecycle/defects/mix-defect.md', lineage: 'mix-defect', slug: 'mix-defect' }),
      ],
      total: 2,
    })

    const wrapper = mount(RecentIdeasDefectsWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    const badges = wrapper.findAll('.type-badge')
    expect(badges).toHaveLength(2)

    const ideaBadge   = badges.find((b) => b.text() === 'idea')
    const defectBadge = badges.find((b) => b.text() === 'defect')
    expect(ideaBadge).toBeDefined()
    expect(defectBadge).toBeDefined()
    expect(ideaBadge!.classes()).toContain('type-badge--idea')
    expect(defectBadge!.classes()).toContain('type-badge--defect')
  })
})

// ===========================================================================
// TC5 — Live update: artifact.indexed WebSocket event triggers re-fetch
// ===========================================================================

describe('RecentIdeasDefectsWidget — TC5: live update via WebSocket', () => {
  it('registers a handler for the artifact.indexed event on mount', async () => {
    const { useWebSocket } = await import('@/composables/useWebSocket' as any)

    mount(RecentIdeasDefectsWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    // useWebSocket must be called with the correct project and event name.
    expect(vi.mocked(useWebSocket)).toHaveBeenCalledWith(
      'testproject',
      'artifact.indexed',
      expect.any(Function),
    )
  })

  it('re-fetches data when the artifact.indexed WebSocket callback is invoked', async () => {
    const { listArtifacts } = await import('@/api/artifacts' as any)
    vi.mocked(listArtifacts).mockResolvedValue({ items: [], total: 0 })

    mount(RecentIdeasDefectsWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    const callsAfterMount = vi.mocked(listArtifacts).mock.calls.length
    expect(callsAfterMount).toBeGreaterThanOrEqual(1)

    // Simulate the WebSocket event firing.
    expect(capturedWsCallback).not.toBeNull()
    capturedWsCallback!({ type: 'artifact.indexed' })
    await flushPromises()

    // listArtifacts must have been called at least once more.
    expect(vi.mocked(listArtifacts).mock.calls.length).toBeGreaterThan(callsAfterMount)
  })

  it('fetches with the same query parameters on refresh as on initial mount', async () => {
    const { listArtifacts } = await import('@/api/artifacts' as any)
    vi.mocked(listArtifacts).mockResolvedValue({ items: [], total: 0 })

    mount(RecentIdeasDefectsWidget, {
      props: { project: 'myproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    capturedWsCallback!({})
    await flushPromises()

    // Both calls should use the same project and filter params.
    const calls = vi.mocked(listArtifacts).mock.calls
    expect(calls.length).toBeGreaterThanOrEqual(2)
    const [firstProject, firstFilter] = calls[0] as [string, Record<string, unknown>]
    const [lastProject,  lastFilter]  = calls[calls.length - 1] as [string, Record<string, unknown>]
    expect(firstProject).toBe(lastProject)
    expect(firstFilter).toEqual(lastFilter)
  })
})

// ===========================================================================
// TC6 — Accessibility: focusable items and aria-label on type badges
// ===========================================================================

describe('RecentIdeasDefectsWidget — TC6: accessibility', () => {
  it('each item renders as an <a> element (natively focusable)', async () => {
    const { listArtifacts } = await import('@/api/artifacts' as any)
    vi.mocked(listArtifacts).mockResolvedValueOnce({
      items: [
        makeItem({ title: 'Focus Item 1', path: 'lifecycle/ideas/focus1.md' }),
        makeItem({ title: 'Focus Item 2', path: 'lifecycle/ideas/focus2.md', lineage: 'focus2', slug: 'focus2' }),
      ],
      total: 2,
    })

    const wrapper = mount(RecentIdeasDefectsWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    // Each item-link is an <a> element — natively keyboard focusable.
    const links = wrapper.findAll('a.item-link')
    expect(links).toHaveLength(2)
  })

  it('type badges carry aria-label describing the artifact type', async () => {
    const { listArtifacts } = await import('@/api/artifacts' as any)
    vi.mocked(listArtifacts).mockResolvedValueOnce({
      items: [
        makeItem({ type: 'idea',   title: 'Aria Idea',   path: 'lifecycle/ideas/aria-idea.md' }),
        makeItem({ type: 'defect', title: 'Aria Defect', path: 'lifecycle/defects/aria-defect.md', lineage: 'aria-defect', slug: 'aria-defect' }),
      ],
      total: 2,
    })

    const wrapper = mount(RecentIdeasDefectsWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    const badges = wrapper.findAll('.type-badge')
    for (const badge of badges) {
      const ariaLabel = badge.attributes('aria-label')
      expect(ariaLabel).toBeTruthy()
      expect(ariaLabel).toMatch(/type:/i)
    }
  })

  it('idea badge aria-label contains "idea"', async () => {
    const { listArtifacts } = await import('@/api/artifacts' as any)
    vi.mocked(listArtifacts).mockResolvedValueOnce({
      items: [makeItem({ type: 'idea', title: 'Aria Idea Badge' })],
      total: 1,
    })

    const wrapper = mount(RecentIdeasDefectsWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    const badge = wrapper.find('.type-badge')
    expect(badge.attributes('aria-label')).toMatch(/idea/i)
  })

  it('defect badge aria-label contains "defect"', async () => {
    const { listArtifacts } = await import('@/api/artifacts' as any)
    vi.mocked(listArtifacts).mockResolvedValueOnce({
      items: [makeItem({ type: 'defect', title: 'Aria Defect Badge', path: 'lifecycle/defects/aria-def.md' })],
      total: 1,
    })

    const wrapper = mount(RecentIdeasDefectsWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    const badge = wrapper.find('.type-badge')
    expect(badge.attributes('aria-label')).toMatch(/defect/i)
  })
})

// ===========================================================================
// General: widget title and API query parameters
// ===========================================================================

describe('RecentIdeasDefectsWidget — general', () => {
  it('renders the widget title "Recent Ideas & Defects"', async () => {
    const wrapper = mount(RecentIdeasDefectsWidget, {
      props: { project: 'testproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    expect(wrapper.find('.widget-title').text()).toContain('Recent Ideas')
  })

  it('calls listArtifacts with type=idea,defect, sort=created:desc, limit=7', async () => {
    const { listArtifacts } = await import('@/api/artifacts' as any)
    vi.mocked(listArtifacts).mockResolvedValue({ items: [], total: 0 })

    mount(RecentIdeasDefectsWidget, {
      props: { project: 'myproject' },
      global: { plugins: [makeRouter()] },
    })
    await flushPromises()

    expect(vi.mocked(listArtifacts)).toHaveBeenCalledWith(
      'myproject',
      expect.objectContaining({
        type: 'idea,defect',
        sort: 'created:desc',
        limit: 7,
      }),
    )
  })
})
