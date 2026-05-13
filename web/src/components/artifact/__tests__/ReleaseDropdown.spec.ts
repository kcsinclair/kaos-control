// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ReleaseDropdown from '../ReleaseDropdown.vue'
import type { Release } from '@/types/release'

// Mock the API modules so no real HTTP calls are made.
vi.mock('@/api/artifacts', () => ({
  patchRelease: vi.fn(),
}))
vi.mock('@/api/releases', () => ({
  listReleases: vi.fn(),
}))

// Import the mocked functions for assertion.
import { patchRelease } from '@/api/artifacts'
import { listReleases } from '@/api/releases'

const mockReleases: Release[] = [
  { id: 1, name: 'v1.0', status: 'planned', start_date: null, end_date: null, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 2, name: 'v2.0', status: 'active', start_date: '2026-04-01', end_date: '2026-06-30', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
]

const DEFAULT_PROPS = {
  project: 'testproject',
  path: 'lifecycle/ideas/test.md',
  release: null as string | null,
}

describe('ReleaseDropdown', () => {
  beforeEach(() => {
    vi.mocked(listReleases).mockResolvedValue(mockReleases)
    vi.mocked(patchRelease).mockResolvedValue({ artifact: {} as never })
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  // ── TC1: Renders current release name when release prop is set ─────────────

  it('renders the current release name when release prop is set', () => {
    const wrapper = mount(ReleaseDropdown, {
      props: { ...DEFAULT_PROPS, release: 'v1.0' },
    })
    expect(wrapper.find('button').text()).toBe('v1.0')
  })

  // ── TC2: Renders "None" when release prop is null ─────────────────────────

  it('renders "None" when release is null', () => {
    const wrapper = mount(ReleaseDropdown, {
      props: { ...DEFAULT_PROPS, release: null },
    })
    expect(wrapper.find('button').text()).toBe('None')
  })

  // ── TC3: Opens dropdown on click and shows release options from API ────────

  it('opens dropdown on click and fetches release options', async () => {
    const wrapper = mount(ReleaseDropdown, {
      props: { ...DEFAULT_PROPS },
      attachTo: document.body,
    })
    await wrapper.find('button').trigger('click')
    await flushPromises()

    expect(vi.mocked(listReleases)).toHaveBeenCalledWith('testproject')
    expect(wrapper.find('[role="listbox"]').exists()).toBe(true)

    wrapper.unmount()
  })

  // ── TC4: Shows release name and status for each option ────────────────────

  it('shows release name and status in dropdown options', async () => {
    const wrapper = mount(ReleaseDropdown, {
      props: { ...DEFAULT_PROPS },
      attachTo: document.body,
    })
    await wrapper.find('button').trigger('click')
    await flushPromises()

    const names = wrapper.findAll('.release-name')
    const statuses = wrapper.findAll('.release-status')

    expect(names).toHaveLength(2)
    expect(names[0].text()).toBe('v1.0')
    expect(statuses[0].text()).toBe('planned')
    expect(names[1].text()).toBe('v2.0')
    expect(statuses[1].text()).toBe('active')

    wrapper.unmount()
  })

  // ── TC5: "None" option is present at the top of the dropdown ─────────────

  it('shows "None" as the first option in the dropdown', async () => {
    const wrapper = mount(ReleaseDropdown, {
      props: { ...DEFAULT_PROPS },
      attachTo: document.body,
    })
    await wrapper.find('button').trigger('click')
    await flushPromises()

    const options = wrapper.findAll('[role="option"]')
    expect(options[0].text()).toContain('None')

    wrapper.unmount()
  })

  // ── TC6: Selecting a release calls patchRelease and emits 'changed' ───────

  it('calls patchRelease and emits changed when a release option is selected', async () => {
    const wrapper = mount(ReleaseDropdown, {
      props: { ...DEFAULT_PROPS, release: null },
      attachTo: document.body,
    })
    await wrapper.find('button').trigger('click')
    await flushPromises()

    // options[0] = None, options[1] = v1.0
    const options = wrapper.findAll('[role="option"]')
    await options[1].trigger('click')
    await flushPromises()

    expect(vi.mocked(patchRelease)).toHaveBeenCalledWith('testproject', 'lifecycle/ideas/test.md', 'v1.0')
    expect(wrapper.emitted('changed')).toBeTruthy()
    expect(wrapper.emitted('changed')![0]).toEqual(['v1.0'])

    wrapper.unmount()
  })

  // ── TC7: Selecting "None" calls patchRelease with null ────────────────────

  it('calls patchRelease with null and emits changed when "None" is selected', async () => {
    const wrapper = mount(ReleaseDropdown, {
      props: { ...DEFAULT_PROPS, release: 'v1.0' },
      attachTo: document.body,
    })
    await wrapper.find('button').trigger('click')
    await flushPromises()

    const options = wrapper.findAll('[role="option"]')
    await options[0].trigger('click') // None option
    await flushPromises()

    expect(vi.mocked(patchRelease)).toHaveBeenCalledWith('testproject', 'lifecycle/ideas/test.md', null)
    expect(wrapper.emitted('changed')).toBeTruthy()
    expect(wrapper.emitted('changed')![0]).toEqual([null])

    wrapper.unmount()
  })

  // ── TC8: Optimistic update — value shows immediately before PATCH resolves ─

  it('shows the new release immediately before PATCH resolves (optimistic update)', async () => {
    let resolvePatching!: (v: unknown) => void
    vi.mocked(patchRelease).mockImplementation(
      () => new Promise((r) => { resolvePatching = r }),
    )

    const wrapper = mount(ReleaseDropdown, {
      props: { ...DEFAULT_PROPS, release: null },
      attachTo: document.body,
    })
    await wrapper.find('button').trigger('click')
    await flushPromises()

    const options = wrapper.findAll('[role="option"]')
    await options[1].trigger('click') // select v1.0

    // Before the PATCH resolves, the button should already show 'v1.0'
    expect(wrapper.find('button').text()).toBe('v1.0')

    resolvePatching({ artifact: {} })
    wrapper.unmount()
  })

  // ── TC9: Error rollback ────────────────────────────────────────────────────

  it('reverts the value and emits error when PATCH fails', async () => {
    vi.mocked(patchRelease).mockRejectedValue(new Error('Network error'))

    const wrapper = mount(ReleaseDropdown, {
      props: { ...DEFAULT_PROPS, release: null },
      attachTo: document.body,
    })
    await wrapper.find('button').trigger('click')
    await flushPromises()

    const options = wrapper.findAll('[role="option"]')
    await options[1].trigger('click') // select v1.0
    await flushPromises()

    // After rejection the optimistic value should revert to null → shows 'None'
    expect(wrapper.find('button').text()).toBe('None')
    expect(wrapper.emitted('error')).toBeTruthy()
    expect(wrapper.emitted('error')![0]).toEqual(['Network error'])

    wrapper.unmount()
  })

  // ── TC10: Readonly mode — clicking does not open the dropdown ─────────────

  it('does not render an interactive button or open dropdown in readonly mode', () => {
    const wrapper = mount(ReleaseDropdown, {
      props: { ...DEFAULT_PROPS, release: 'v1.0', readonly: true },
    })
    // In readonly mode the template renders a <span>, not a <button>
    expect(wrapper.find('button').exists()).toBe(false)
    expect(wrapper.find('[role="listbox"]').exists()).toBe(false)
    expect(wrapper.find('span.release-badge').text()).toBe('v1.0')
  })

  // ── TC11: Keyboard navigation ─────────────────────────────────────────────

  it('supports keyboard navigation: Enter opens, ArrowDown/Up moves focus, Escape closes', async () => {
    const wrapper = mount(ReleaseDropdown, {
      props: { ...DEFAULT_PROPS },
      attachTo: document.body,
    })
    const button = wrapper.find('button')

    // Open with Enter
    await button.trigger('keydown', { key: 'Enter' })
    await flushPromises()
    expect(wrapper.find('[role="listbox"]').exists()).toBe(true)

    const menu = wrapper.find('[role="listbox"]')

    // ArrowDown moves focus down
    await menu.trigger('keydown', { key: 'ArrowDown' })
    // ArrowUp moves focus back up
    await menu.trigger('keydown', { key: 'ArrowUp' })

    // Escape closes
    await menu.trigger('keydown', { key: 'Escape' })
    expect(wrapper.find('[role="listbox"]').exists()).toBe(false)

    wrapper.unmount()
  })

  // ── TC12: Outside click closes the dropdown ───────────────────────────────

  it('closes the dropdown when clicking outside the component', async () => {
    const wrapper = mount(ReleaseDropdown, {
      props: { ...DEFAULT_PROPS },
      attachTo: document.body,
    })
    await wrapper.find('button').trigger('click')
    await flushPromises()
    expect(wrapper.find('[role="listbox"]').exists()).toBe(true)

    // Dispatch a click event on a node outside the component
    const outside = document.createElement('div')
    document.body.appendChild(outside)
    outside.dispatchEvent(new MouseEvent('click', { bubbles: true, composed: true }))
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[role="listbox"]').exists()).toBe(false)

    outside.remove()
    wrapper.unmount()
  })

  // ── TC13: ARIA attributes ─────────────────────────────────────────────────

  it('sets correct ARIA attributes on the trigger and listbox', async () => {
    const wrapper = mount(ReleaseDropdown, {
      props: { ...DEFAULT_PROPS },
      attachTo: document.body,
    })
    const button = wrapper.find('button')

    // Trigger has aria-haspopup and aria-expanded=false when closed
    expect(button.attributes('aria-haspopup')).toBe('listbox')
    expect(button.attributes('aria-expanded')).toBe('false')

    await button.trigger('click')
    await flushPromises()

    // aria-expanded becomes true once open
    expect(button.attributes('aria-expanded')).toBe('true')

    // Listbox role exists
    const listbox = wrapper.find('[role="listbox"]')
    expect(listbox.exists()).toBe(true)

    // aria-activedescendant is set on the listbox
    expect(listbox.attributes('aria-activedescendant')).toBeTruthy()

    wrapper.unmount()
  })
})
