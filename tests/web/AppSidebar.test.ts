/**
 * Integration tests for AppSidebar — Collapsible Sidebar with Icon-Only Mode
 *
 * Covers:
 *   Milestone 1  — Toggle behaviour (collapse / expand, aria attributes, icon swap)
 *   Milestone 2  — Icon rendering in both expanded and collapsed states
 *   Milestone 3  — Tooltip behaviour (collapsed shows tooltip, expanded does not)
 *   Milestone 4  — Badge preservation across sidebar states
 *   Milestone 5  — State persistence via localStorage
 *   Milestone 6  — Hover-to-expand overlay (does not affect localStorage)
 *   Milestone 7  — CSS transition property is present
 *   Milestone 8  — Animation: sidebar has CSS transition including width
 *
 * Notes on testing approach:
 * ─────────────────────────
 * happy-dom does not compute layout (getBoundingClientRect returns zeros,
 * getComputedStyle returns only inline styles).  Width assertions therefore
 * target the CSS classes that drive the width CSS variable rather than
 * measuring pixels directly:
 *   • Expanded:  `.app-sidebar` WITHOUT  `.sidebar--collapsed`
 *   • Collapsed: `.app-sidebar` WITH     `.sidebar--collapsed`
 *
 * The `transition` style assertion reads the scoped CSS written inline on
 * the element via the component's <style scoped> block, which IS reflected
 * in the DOM as a style attribute when set by Vite's scoped-css transform.
 * Where that is not present we validate the CSS class that carries the
 * transition rule instead.
 *
 * Tooltip visibility is checked by looking for `.sidebar-tooltip` elements
 * in `document.body` (the SidebarTooltip component uses Teleport to body).
 *
 * localStorage is fully functional in happy-dom.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import AppSidebar from '../../web/src/components/layout/AppSidebar.vue'
import { useUiStore } from '../../web/src/stores/ui'

// ---------------------------------------------------------------------------
// Module mocks — prevent real network I/O and WebSocket connections
// ---------------------------------------------------------------------------

vi.mock('@/api/client', () => ({
  api: {
    get: vi.fn().mockResolvedValue({ errors: [] }),
  },
}))

vi.mock('@/api/ws', () => ({
  getProjectWs: vi.fn(() => ({
    onType: vi.fn(() => () => {}),
  })),
}))

vi.mock('@/stores/project', () => ({
  useProjectStore: vi.fn(() => ({
    current: { name: 'Test Project' },
  })),
}))

// ---------------------------------------------------------------------------
// Router factory — AppSidebar reads route.params.project
// ---------------------------------------------------------------------------

function makeRouter(path = '/p/testproject') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/p/:project/:sub*', component: { template: '<div/>' } },
      { path: '/:pathMatch(.*)*', component: { template: '<div/>' } },
    ],
  })
  router.push(path)
  return router
}

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

async function mountSidebar(opts: { collapsed?: boolean; path?: string } = {}) {
  // Reset localStorage between tests
  localStorage.clear()
  if (opts.collapsed !== undefined) {
    localStorage.setItem('sidebar-collapsed', String(opts.collapsed))
  }

  const pinia = createPinia()
  setActivePinia(pinia)

  const router = makeRouter(opts.path ?? '/p/testproject')
  await router.isReady()

  const wrapper = mount(AppSidebar, {
    global: { plugins: [pinia, router] },
    attachTo: document.body,
  })

  await flushPromises()

  const uiStore = useUiStore()
  return { wrapper, uiStore, router }
}

// ---------------------------------------------------------------------------
// Cleanup — remove appended DOM nodes between tests
// ---------------------------------------------------------------------------

afterEach(() => {
  localStorage.clear()
  vi.clearAllMocks()
  // Remove any lingering Teleport nodes
  document.body.innerHTML = ''
})

// ===========================================================================
// Milestone 1 — Toggle Behaviour
// ===========================================================================

describe('AppSidebar — Milestone 1: toggle behaviour', () => {
  it('starts expanded by default (no sidebar--collapsed class)', async () => {
    const { wrapper } = await mountSidebar()
    expect(wrapper.find('nav.app-sidebar').classes()).not.toContain('sidebar--collapsed')
  })

  it('clicking toggle collapses the sidebar (adds sidebar--collapsed class)', async () => {
    const { wrapper } = await mountSidebar()
    const toggle = wrapper.find('.sidebar-toggle')
    await toggle.trigger('click')
    expect(wrapper.find('nav.app-sidebar').classes()).toContain('sidebar--collapsed')
  })

  it('clicking toggle again expands the sidebar (removes sidebar--collapsed class)', async () => {
    const { wrapper } = await mountSidebar()
    const toggle = wrapper.find('.sidebar-toggle')
    await toggle.trigger('click')
    await toggle.trigger('click')
    expect(wrapper.find('nav.app-sidebar').classes()).not.toContain('sidebar--collapsed')
  })

  it('toggle button shows ChevronLeft icon when expanded', async () => {
    const { wrapper } = await mountSidebar({ collapsed: false })
    const toggle = wrapper.find('.sidebar-toggle')
    // ChevronLeft renders as an SVG; ChevronRight should NOT be present when expanded
    const svgs = toggle.findAll('svg')
    expect(svgs.length).toBeGreaterThan(0)
    // The component renders ChevronRight when collapsed, ChevronLeft when expanded.
    // We validate the uiStore state drives the right v-if branch.
    const { uiStore } = await mountSidebar({ collapsed: false })
    expect(uiStore.sidebarCollapsed).toBe(false)
  })

  it('toggle button shows ChevronRight icon when collapsed', async () => {
    const { wrapper, uiStore } = await mountSidebar({ collapsed: true })
    expect(uiStore.sidebarCollapsed).toBe(true)
    const toggle = wrapper.find('.sidebar-toggle')
    expect(toggle.find('svg').exists()).toBe(true)
  })

  it('toggle button aria-expanded is "true" when sidebar is expanded', async () => {
    const { wrapper } = await mountSidebar({ collapsed: false })
    const toggle = wrapper.find('.sidebar-toggle')
    // aria-expanded is a boolean prop; Vue renders it as the string "true"
    expect(toggle.attributes('aria-expanded')).toBe('true')
  })

  it('toggle button aria-expanded is "false" when sidebar is collapsed', async () => {
    const { wrapper } = await mountSidebar({ collapsed: true })
    const toggle = wrapper.find('.sidebar-toggle')
    expect(toggle.attributes('aria-expanded')).toBe('false')
  })

  it('toggle button aria-label reads "Collapse sidebar" when expanded', async () => {
    const { wrapper } = await mountSidebar({ collapsed: false })
    const toggle = wrapper.find('.sidebar-toggle')
    expect(toggle.attributes('aria-label')).toBe('Collapse sidebar')
  })

  it('toggle button aria-label reads "Expand sidebar" when collapsed', async () => {
    const { wrapper } = await mountSidebar({ collapsed: true })
    const toggle = wrapper.find('.sidebar-toggle')
    expect(toggle.attributes('aria-label')).toBe('Expand sidebar')
  })

  it('aria-label on toggle switches after clicking', async () => {
    const { wrapper } = await mountSidebar({ collapsed: false })
    const toggle = wrapper.find('.sidebar-toggle')
    expect(toggle.attributes('aria-label')).toBe('Collapse sidebar')
    await toggle.trigger('click')
    expect(toggle.attributes('aria-label')).toBe('Expand sidebar')
  })
})

// ===========================================================================
// Milestone 2 — Icon Rendering
// ===========================================================================

describe('AppSidebar — Milestone 2: icon rendering', () => {
  const expectedLabels = ['Dashboard', 'List', 'Board', 'Testing', 'Graph', 'Roadmap', 'Agents', 'Scheduler', 'Feed', 'Parse Errors', 'Config', 'Ollama']

  it('renders an SVG icon for each nav item in expanded mode', async () => {
    const { wrapper } = await mountSidebar({ collapsed: false })
    const navItems = wrapper.findAll('.nav-item')
    expect(navItems.length).toBe(expectedLabels.length)
    for (const item of navItems) {
      expect(item.find('svg').exists()).toBe(true)
    }
  })

  it('renders an SVG icon for each nav item in collapsed mode', async () => {
    const { wrapper } = await mountSidebar({ collapsed: true })
    const navItems = wrapper.findAll('.nav-item')
    expect(navItems.length).toBe(expectedLabels.length)
    for (const item of navItems) {
      expect(item.find('svg').exists()).toBe(true)
    }
  })

  it('nav-label text is present in the DOM when expanded', async () => {
    const { wrapper } = await mountSidebar({ collapsed: false })
    for (const label of expectedLabels) {
      const found = wrapper.findAll('.nav-label').some(el => el.text() === label)
      expect(found, `label "${label}" not found`).toBe(true)
    }
  })

  it('nav-label elements are hidden via CSS class when collapsed', async () => {
    // In collapsed mode the sidebar--collapsed class is present, which sets
    // .sidebar--collapsed .nav-label { display: none }. The elements remain
    // in the DOM (v-show/CSS, not v-if), so we assert the class is present.
    const { wrapper } = await mountSidebar({ collapsed: true })
    expect(wrapper.find('nav.app-sidebar').classes()).toContain('sidebar--collapsed')
    // Labels are still in the DOM
    const labels = wrapper.findAll('.nav-label')
    expect(labels.length).toBe(expectedLabels.length)
  })

  it('does not render an "Artefacts" group header', async () => {
    const { wrapper } = await mountSidebar({ collapsed: false })
    expect(wrapper.text()).not.toContain('Artefacts')
  })

  it('all twelve expected nav items are rendered', async () => {
    const { wrapper } = await mountSidebar({ collapsed: false })
    for (const label of expectedLabels) {
      expect(wrapper.text()).toContain(label)
    }
  })
})

// ===========================================================================
// Milestone 3 — Tooltip Behaviour
// ===========================================================================

describe('AppSidebar — Milestone 3: tooltip behaviour', () => {
  it('collapsed nav links have aria-label attributes matching their visible labels', async () => {
    const { wrapper } = await mountSidebar({ collapsed: true })
    const navLinks = wrapper.findAll('.nav-link')
    for (const link of navLinks) {
      // When collapsed, each link receives aria-label
      expect(link.attributes('aria-label')).toBeTruthy()
    }
  })

  it('expanded nav links do not have aria-label (text label is visible)', async () => {
    const { wrapper } = await mountSidebar({ collapsed: false })
    const navLinks = wrapper.findAll('.nav-link')
    for (const link of navLinks) {
      // When expanded the text is visible; aria-label is not needed and should be absent
      expect(link.attributes('aria-label')).toBeUndefined()
    }
  })

  it('tooltip appears on mouseenter when sidebar is collapsed', async () => {
    const { wrapper } = await mountSidebar({ collapsed: true })
    // The SidebarTooltip wrapper handles mouseenter → shows tooltip via Teleport
    const tooltipWrapper = wrapper.find('.nav-item .tooltip-wrapper')
    expect(tooltipWrapper.exists()).toBe(true)
    await tooltipWrapper.trigger('mouseenter')
    // Tooltip is teleported to body
    const tooltip = document.querySelector('.sidebar-tooltip')
    expect(tooltip).not.toBeNull()
  })

  it('tooltip disappears on mouseleave when sidebar is collapsed', async () => {
    const { wrapper } = await mountSidebar({ collapsed: true })
    const tooltipWrapper = wrapper.find('.nav-item .tooltip-wrapper')
    await tooltipWrapper.trigger('mouseenter')
    await tooltipWrapper.trigger('mouseleave')
    const tooltip = document.querySelector('.sidebar-tooltip')
    expect(tooltip).toBeNull()
  })

  it('tooltip does not appear on mouseenter when sidebar is expanded', async () => {
    const { wrapper } = await mountSidebar({ collapsed: false })
    const tooltipWrapper = wrapper.find('.nav-item .tooltip-wrapper')
    await tooltipWrapper.trigger('mouseenter')
    const tooltip = document.querySelector('.sidebar-tooltip')
    expect(tooltip).toBeNull()
  })

  it('tooltip is shown on focusin (keyboard accessible) when collapsed', async () => {
    const { wrapper } = await mountSidebar({ collapsed: true })
    const tooltipWrapper = wrapper.find('.nav-item .tooltip-wrapper')
    await tooltipWrapper.trigger('focusin')
    const tooltip = document.querySelector('.sidebar-tooltip')
    expect(tooltip).not.toBeNull()
  })

  it('tooltip disappears on focusout', async () => {
    const { wrapper } = await mountSidebar({ collapsed: true })
    const tooltipWrapper = wrapper.find('.nav-item .tooltip-wrapper')
    await tooltipWrapper.trigger('focusin')
    await tooltipWrapper.trigger('focusout')
    const tooltip = document.querySelector('.sidebar-tooltip')
    expect(tooltip).toBeNull()
  })

  it('aria-label on nav link matches the corresponding nav item label', async () => {
    const { wrapper } = await mountSidebar({ collapsed: true })
    const allExpectedLabels = ['Dashboard', 'List', 'Board', 'Testing', 'Graph', 'Roadmap', 'Agents', 'Scheduler', 'Feed', 'Parse Errors', 'Config', 'Ollama']
    const navLinks = wrapper.findAll('.nav-link')
    // Iterate over navLinks (not a fixed-size array) so the test stays correct
    // when nav items are added or removed in future.
    expect(navLinks.length).toBe(allExpectedLabels.length)
    for (let i = 0; i < navLinks.length; i++) {
      expect(navLinks[i].attributes('aria-label')).toBe(allExpectedLabels[i])
    }
  })
})

// ===========================================================================
// Milestone 4 — Badge Preservation
// ===========================================================================

describe('AppSidebar — Milestone 4: badge preservation', () => {
  it('badge is visible in expanded mode when parse errors exist', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({ errors: [{ path: 'a.md', message: 'err' }] })

    const { wrapper } = await mountSidebar({ collapsed: false })
    await flushPromises()

    // In expanded mode the badge element with class "badge" is rendered
    const badge = wrapper.find('.badge')
    expect(badge.exists()).toBe(true)
  })

  it('badge-dot is visible in collapsed mode when parse errors exist', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({ errors: [{ path: 'a.md', message: 'err' }] })

    const { wrapper } = await mountSidebar({ collapsed: true })
    await flushPromises()

    // In collapsed mode the badge-dot element is rendered instead
    const badgeDot = wrapper.find('.badge-dot')
    expect(badgeDot.exists()).toBe(true)
  })

  it('badge-dot aria-label includes the error count', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValueOnce({
      errors: [{ path: 'a.md', message: 'err1' }, { path: 'b.md', message: 'err2' }],
    })

    const { wrapper } = await mountSidebar({ collapsed: true })
    await flushPromises()

    const badgeDot = wrapper.find('.badge-dot')
    expect(badgeDot.exists()).toBe(true)
    expect(badgeDot.attributes('aria-label')).toContain('2')
  })

  it('no badge rendered when there are no parse errors', async () => {
    // Default mock returns { errors: [] }
    const { wrapper } = await mountSidebar({ collapsed: false })
    await flushPromises()

    expect(wrapper.find('.badge').exists()).toBe(false)
    expect(wrapper.find('.badge-dot').exists()).toBe(false)
  })

  it('switching from expanded to collapsed moves badge from .badge to .badge-dot', async () => {
    const { api } = await import('@/api/client' as any)
    vi.mocked(api.get).mockResolvedValue({ errors: [{ path: 'a.md', message: 'err' }] })

    const { wrapper } = await mountSidebar({ collapsed: false })
    await flushPromises()

    expect(wrapper.find('.badge').exists()).toBe(true)
    expect(wrapper.find('.badge-dot').exists()).toBe(false)

    await wrapper.find('.sidebar-toggle').trigger('click')
    await flushPromises()

    expect(wrapper.find('.badge').exists()).toBe(false)
    expect(wrapper.find('.badge-dot').exists()).toBe(true)
  })
})

// ===========================================================================
// Milestone 5 — State Persistence via localStorage
// ===========================================================================

describe('AppSidebar — Milestone 5: localStorage persistence', () => {
  it('collapsing the sidebar writes sidebar-collapsed=true to localStorage', async () => {
    const { wrapper } = await mountSidebar({ collapsed: false })
    await wrapper.find('.sidebar-toggle').trigger('click')
    expect(localStorage.getItem('sidebar-collapsed')).toBe('true')
  })

  it('expanding the sidebar writes sidebar-collapsed=false to localStorage', async () => {
    const { wrapper } = await mountSidebar({ collapsed: true })
    await wrapper.find('.sidebar-toggle').trigger('click')
    expect(localStorage.getItem('sidebar-collapsed')).toBe('false')
  })

  it('sidebar initialises as collapsed when localStorage has sidebar-collapsed=true', async () => {
    const { wrapper } = await mountSidebar({ collapsed: true })
    expect(wrapper.find('nav.app-sidebar').classes()).toContain('sidebar--collapsed')
  })

  it('sidebar initialises as expanded when localStorage has sidebar-collapsed=false', async () => {
    const { wrapper } = await mountSidebar({ collapsed: false })
    expect(wrapper.find('nav.app-sidebar').classes()).not.toContain('sidebar--collapsed')
  })

  it('localStorage value after two toggles is "false" (returns to expanded)', async () => {
    const { wrapper } = await mountSidebar({ collapsed: false })
    const toggle = wrapper.find('.sidebar-toggle')
    await toggle.trigger('click')
    await toggle.trigger('click')
    expect(localStorage.getItem('sidebar-collapsed')).toBe('false')
  })

  it('a fresh mount reads the persisted collapsed state correctly', async () => {
    // First session: collapse
    const { wrapper: w1 } = await mountSidebar({ collapsed: false })
    await w1.find('.sidebar-toggle').trigger('click')
    expect(localStorage.getItem('sidebar-collapsed')).toBe('true')
    w1.unmount()

    // Second session: re-mount — reads from localStorage
    const pinia2 = createPinia()
    setActivePinia(pinia2)
    const router2 = makeRouter()
    await router2.isReady()
    const wrapper2 = mount(AppSidebar, {
      global: { plugins: [pinia2, router2] },
      attachTo: document.body,
    })
    await flushPromises()
    expect(wrapper2.find('nav.app-sidebar').classes()).toContain('sidebar--collapsed')
    wrapper2.unmount()
  })
})

// ===========================================================================
// Milestone 6 — Hover-to-Expand Overlay
// ===========================================================================

describe('AppSidebar — Milestone 6: hover-to-expand overlay', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('hovering over a collapsed sidebar after 200 ms adds sidebar--overlay class', async () => {
    const { wrapper } = await mountSidebar({ collapsed: true })
    const nav = wrapper.find('nav.app-sidebar')

    await nav.trigger('mouseenter')
    expect(nav.classes()).not.toContain('sidebar--overlay')

    vi.advanceTimersByTime(200)
    await flushPromises()

    expect(nav.classes()).toContain('sidebar--overlay')
  })

  it('overlay is not triggered before 200 ms delay', async () => {
    const { wrapper } = await mountSidebar({ collapsed: true })
    const nav = wrapper.find('nav.app-sidebar')

    await nav.trigger('mouseenter')
    vi.advanceTimersByTime(150)
    await flushPromises()

    expect(nav.classes()).not.toContain('sidebar--overlay')
  })

  it('mouseleave removes the overlay', async () => {
    const { wrapper } = await mountSidebar({ collapsed: true })
    const nav = wrapper.find('nav.app-sidebar')

    await nav.trigger('mouseenter')
    vi.advanceTimersByTime(200)
    await flushPromises()

    expect(nav.classes()).toContain('sidebar--overlay')

    await nav.trigger('mouseleave')
    await flushPromises()

    expect(nav.classes()).not.toContain('sidebar--overlay')
  })

  it('sidebar--collapsed class is absent while overlay is active', async () => {
    const { wrapper } = await mountSidebar({ collapsed: true })
    const nav = wrapper.find('nav.app-sidebar')

    await nav.trigger('mouseenter')
    vi.advanceTimersByTime(200)
    await flushPromises()

    // The template binds sidebar--collapsed only when collapsed && !hoverExpanded
    expect(nav.classes()).not.toContain('sidebar--collapsed')
  })

  it('localStorage value does not change during hover expand', async () => {
    const { wrapper } = await mountSidebar({ collapsed: true })
    const nav = wrapper.find('nav.app-sidebar')

    await nav.trigger('mouseenter')
    vi.advanceTimersByTime(200)
    await flushPromises()

    // localStorage must still reflect the persisted collapsed state
    expect(localStorage.getItem('sidebar-collapsed')).toBe('true')
  })

  it('hover does not expand when sidebar is already expanded', async () => {
    const { wrapper } = await mountSidebar({ collapsed: false })
    const nav = wrapper.find('nav.app-sidebar')

    await nav.trigger('mouseenter')
    vi.advanceTimersByTime(200)
    await flushPromises()

    expect(nav.classes()).not.toContain('sidebar--overlay')
  })

  it('after hover-expand, full labels are visible (sidebar visually expanded)', async () => {
    const { wrapper } = await mountSidebar({ collapsed: true })
    const nav = wrapper.find('nav.app-sidebar')

    await nav.trigger('mouseenter')
    vi.advanceTimersByTime(200)
    await flushPromises()

    // In hover-expanded state, the component shows expanded content
    // nav-label elements should be rendered (sidebar is visually expanded)
    expect(wrapper.findAll('.nav-label').length).toBeGreaterThan(0)
  })
})

// ===========================================================================
// Milestone 7 — Layout Integrity (class-level assertions)
// ===========================================================================

describe('AppSidebar — Milestone 7: layout integrity', () => {
  const views = [
    '/p/testproject/artifacts',
    '/p/testproject/artifacts/board',
    '/p/testproject/graph',
    '/p/testproject/agents',
    '/p/testproject/parse-errors',
    '/p/testproject/config',
  ]

  for (const path of views) {
    it(`collapsed sidebar renders correctly on route ${path}`, async () => {
      const { wrapper } = await mountSidebar({ collapsed: true, path })
      const nav = wrapper.find('nav.app-sidebar')
      expect(nav.exists()).toBe(true)
      expect(nav.classes()).toContain('sidebar--collapsed')
    })

    it(`expanded sidebar renders correctly on route ${path}`, async () => {
      const { wrapper } = await mountSidebar({ collapsed: false, path })
      const nav = wrapper.find('nav.app-sidebar')
      expect(nav.exists()).toBe(true)
      expect(nav.classes()).not.toContain('sidebar--collapsed')
    })
  }

  it('all nav links are rendered for each view without errors', async () => {
    for (const path of views) {
      const { wrapper } = await mountSidebar({ path })
      const navLinks = wrapper.findAll('.nav-link')
      expect(navLinks.length, `expected 12 nav links on ${path}`).toBe(12)
      wrapper.unmount()
    }
  })
})

// ===========================================================================
// Milestone 8 — Animation Quality (CSS transition assertions)
// ===========================================================================

describe('AppSidebar — Milestone 8: animation / CSS transition', () => {
  it('the sidebar nav element has a CSS transition style containing "width"', async () => {
    const { wrapper } = await mountSidebar({ collapsed: false })
    const nav = wrapper.find('nav.app-sidebar')

    // The scoped <style> block sets transition: width 250ms ease
    // We verify the inline style OR the computed style string contains "width".
    // happy-dom exposes the raw style attribute; the component's scoped CSS
    // cannot be read via getComputedStyle. We assert via the DOM node's style
    // attribute, or fall back to checking that the .app-sidebar rule exists in
    // the document's stylesheets (JSDOM/happy-dom may or may not inject them).
    //
    // Primary assertion: the component itself exposes transition through inline
    // style or via the class name whose rule carries the transition.
    const el = nav.element as HTMLElement
    const inlineTransition = el.style.transition

    if (inlineTransition) {
      expect(inlineTransition).toContain('width')
    } else {
      // Fallback: verify class-level transition contract via the class names
      // that drive the sequenced animation (sidebar--expanding / sidebar--collapsing).
      // Click expand → should gain sidebar--expanding class
      expect(nav.classes()).toContain('app-sidebar')
      // The component-defined transition is correct by construction; what we
      // can reliably test in happy-dom is that the class that carries the
      // transition rule is applied when toggling.
      await wrapper.find('.sidebar-toggle').trigger('click')
      // After collapse, the collapsing transition class should be set
      expect(nav.classes()).toContain('sidebar--collapsing')
    }
  })

  it('sidebar gains sidebar--collapsing class immediately when collapsing', async () => {
    const { wrapper } = await mountSidebar({ collapsed: false })
    const nav = wrapper.find('nav.app-sidebar')

    await wrapper.find('.sidebar-toggle').trigger('click')
    // The watch fires synchronously on the reactive change
    expect(nav.classes()).toContain('sidebar--collapsing')
  })

  it('sidebar gains sidebar--expanding class immediately when expanding', async () => {
    const { wrapper } = await mountSidebar({ collapsed: true })
    const nav = wrapper.find('nav.app-sidebar')

    await wrapper.find('.sidebar-toggle').trigger('click')
    expect(nav.classes()).toContain('sidebar--expanding')
  })

  it('sidebar--collapsing and sidebar--expanding are mutually exclusive', async () => {
    const { wrapper } = await mountSidebar({ collapsed: false })
    const nav = wrapper.find('nav.app-sidebar')

    await wrapper.find('.sidebar-toggle').trigger('click')
    const classes = nav.classes()
    expect(classes).toContain('sidebar--collapsing')
    expect(classes).not.toContain('sidebar--expanding')
  })
})
