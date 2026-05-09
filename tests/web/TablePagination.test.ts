// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 1 — Unit tests for the `TablePagination` component
 *
 * Tests the component in isolation: rendering, user interaction, boundary
 * conditions, and accessibility attributes. No router or store required.
 *
 * Component location: web/src/components/common/TablePagination.vue
 * Props: totalItems (required), currentPage (default 1), pageSize (default 25)
 * Emits: update:currentPage, update:pageSize
 */

import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import TablePagination from '../../web/src/components/common/TablePagination.vue'

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

function mountPagination(props: {
  totalItems: number
  currentPage?: number
  pageSize?: number
}) {
  return mount(TablePagination, { props })
}

// ---------------------------------------------------------------------------
// Test 1 — Renders with defaults
// ---------------------------------------------------------------------------

describe('TablePagination — default rendering', () => {
  it('renders "Showing 1–25 of 100" with totalItems=100 and default props', () => {
    const wrapper = mountPagination({ totalItems: 100 })
    expect(wrapper.text()).toContain('Showing 1–25 of 100')
  })

  it('renders page 1 of 4 for 100 items at size 25', () => {
    const wrapper = mountPagination({ totalItems: 100 })
    expect(wrapper.text()).toContain('of 4')
  })

  it('Previous button is disabled on page 1', () => {
    const wrapper = mountPagination({ totalItems: 100 })
    const prev = wrapper.find('button[aria-label="Previous page"]')
    expect(prev.attributes('disabled')).toBeDefined()
  })

  it('Next button is enabled on page 1 of 4', () => {
    const wrapper = mountPagination({ totalItems: 100 })
    const next = wrapper.find('button[aria-label="Next page"]')
    expect(next.attributes('disabled')).toBeUndefined()
  })
})

// ---------------------------------------------------------------------------
// Test 2 — Page-size dropdown
// ---------------------------------------------------------------------------

describe('TablePagination — page-size dropdown', () => {
  it('emits update:pageSize(50) and update:currentPage(1) when size changes to 50', async () => {
    const wrapper = mountPagination({ totalItems: 100 })
    const select = wrapper.find('#page-size-select')
    await select.setValue('50')

    const pageSizeEmissions = wrapper.emitted('update:pageSize')
    expect(pageSizeEmissions).toBeDefined()
    expect(pageSizeEmissions![pageSizeEmissions!.length - 1]).toEqual([50])

    const pageEmissions = wrapper.emitted('update:currentPage')
    expect(pageEmissions).toBeDefined()
    expect(pageEmissions![pageEmissions!.length - 1]).toEqual([1])
  })

  it('emits update:pageSize(10) when size changes to 10', async () => {
    const wrapper = mountPagination({ totalItems: 100 })
    await wrapper.find('#page-size-select').setValue('10')
    const emissions = wrapper.emitted('update:pageSize')!
    expect(emissions[emissions.length - 1]).toEqual([10])
  })
})

// ---------------------------------------------------------------------------
// Test 3 — Next / Previous navigation
// ---------------------------------------------------------------------------

describe('TablePagination — Next / Previous navigation', () => {
  it('clicking Next emits update:currentPage(2)', async () => {
    const wrapper = mountPagination({ totalItems: 100, currentPage: 1 })
    await wrapper.find('button[aria-label="Next page"]').trigger('click')
    expect(wrapper.emitted('update:currentPage')).toBeDefined()
    expect(wrapper.emitted('update:currentPage')!.at(-1)).toEqual([2])
  })

  it('Previous is disabled on page 1', () => {
    const wrapper = mountPagination({ totalItems: 100, currentPage: 1 })
    expect(wrapper.find('button[aria-label="Previous page"]').attributes('disabled')).toBeDefined()
  })

  it('clicking Previous emits update:currentPage(1) when on page 2', async () => {
    const wrapper = mountPagination({ totalItems: 100, currentPage: 2 })
    await wrapper.find('button[aria-label="Previous page"]').trigger('click')
    expect(wrapper.emitted('update:currentPage')!.at(-1)).toEqual([1])
  })

  it('Next is disabled on the last page', () => {
    const wrapper = mountPagination({ totalItems: 100, currentPage: 4, pageSize: 25 })
    expect(wrapper.find('button[aria-label="Next page"]').attributes('disabled')).toBeDefined()
  })

  it('Next is NOT disabled when not on the last page', () => {
    const wrapper = mountPagination({ totalItems: 100, currentPage: 3, pageSize: 25 })
    expect(wrapper.find('button[aria-label="Next page"]').attributes('disabled')).toBeUndefined()
  })
})

// ---------------------------------------------------------------------------
// Test 4 — Page-jump input
// ---------------------------------------------------------------------------

describe('TablePagination — page-jump input', () => {
  it('entering "3" and committing emits update:currentPage(3)', async () => {
    const wrapper = mountPagination({ totalItems: 100, currentPage: 1 })
    const input = wrapper.find('#page-jump-input')
    await input.setValue('3')
    await input.trigger('change')
    expect(wrapper.emitted('update:currentPage')!.at(-1)).toEqual([3])
  })

  it('entering "999" clamps to last page (4 for 100 items / size 25)', async () => {
    const wrapper = mountPagination({ totalItems: 100, currentPage: 1 })
    const input = wrapper.find('#page-jump-input')
    await input.setValue('999')
    await input.trigger('change')
    expect(wrapper.emitted('update:currentPage')!.at(-1)).toEqual([4])
  })

  it('entering "0" clamps to page 1', async () => {
    const wrapper = mountPagination({ totalItems: 100, currentPage: 2 })
    const input = wrapper.find('#page-jump-input')
    await input.setValue('0')
    await input.trigger('change')
    expect(wrapper.emitted('update:currentPage')!.at(-1)).toEqual([1])
  })

  it('entering non-numeric keeps current page (emits currentPage unchanged)', async () => {
    const wrapper = mountPagination({ totalItems: 100, currentPage: 2 })
    const input = wrapper.find('#page-jump-input')
    await input.setValue('abc')
    await input.trigger('change')
    // Non-numeric falls back to currentPage (2)
    expect(wrapper.emitted('update:currentPage')!.at(-1)).toEqual([2])
  })

  it('pressing Enter commits the jump', async () => {
    const wrapper = mountPagination({ totalItems: 100, currentPage: 1 })
    const input = wrapper.find('#page-jump-input')
    await input.setValue('2')
    await input.trigger('keydown', { key: 'Enter' })
    expect(wrapper.emitted('update:currentPage')!.at(-1)).toEqual([2])
  })
})

// ---------------------------------------------------------------------------
// Test 5 — Position summary accuracy
// ---------------------------------------------------------------------------

describe('TablePagination — position summary accuracy', () => {
  it('page 1 of 142 items size 25 shows "Showing 1–25 of 142"', () => {
    const wrapper = mountPagination({ totalItems: 142, currentPage: 1, pageSize: 25 })
    expect(wrapper.text()).toContain('Showing 1–25 of 142')
  })

  it('page 6 of 142 items size 25 shows "Showing 126–142 of 142"', () => {
    const wrapper = mountPagination({ totalItems: 142, currentPage: 6, pageSize: 25 })
    expect(wrapper.text()).toContain('Showing 126–142 of 142')
  })

  it('page 2 of 50 items size 25 shows "Showing 26–50 of 50"', () => {
    const wrapper = mountPagination({ totalItems: 50, currentPage: 2, pageSize: 25 })
    expect(wrapper.text()).toContain('Showing 26–50 of 50')
  })
})

// ---------------------------------------------------------------------------
// Test 6 — Single-page dataset
// ---------------------------------------------------------------------------

describe('TablePagination — single-page dataset', () => {
  it('renders without crashing for totalItems=10, pageSize=25', () => {
    const wrapper = mountPagination({ totalItems: 10, pageSize: 25 })
    expect(wrapper.exists()).toBe(true)
  })

  it('Previous is disabled when there is only one page', () => {
    const wrapper = mountPagination({ totalItems: 10, pageSize: 25 })
    expect(wrapper.find('button[aria-label="Previous page"]').attributes('disabled')).toBeDefined()
  })

  it('Next is disabled when there is only one page', () => {
    const wrapper = mountPagination({ totalItems: 10, pageSize: 25 })
    expect(wrapper.find('button[aria-label="Next page"]').attributes('disabled')).toBeDefined()
  })

  it('page-jump input is disabled when there is only one page', () => {
    const wrapper = mountPagination({ totalItems: 10, pageSize: 25 })
    expect(wrapper.find('#page-jump-input').attributes('disabled')).toBeDefined()
  })
})

// ---------------------------------------------------------------------------
// Test 7 — Empty dataset
// ---------------------------------------------------------------------------

describe('TablePagination — empty dataset', () => {
  it('renders without crashing for totalItems=0', () => {
    const wrapper = mountPagination({ totalItems: 0 })
    expect(wrapper.exists()).toBe(true)
  })

  it('shows "No items" for totalItems=0', () => {
    const wrapper = mountPagination({ totalItems: 0 })
    expect(wrapper.text()).toContain('No items')
  })

  it('Previous and Next are disabled for totalItems=0', () => {
    const wrapper = mountPagination({ totalItems: 0 })
    expect(wrapper.find('button[aria-label="Previous page"]').attributes('disabled')).toBeDefined()
    expect(wrapper.find('button[aria-label="Next page"]').attributes('disabled')).toBeDefined()
  })
})

// ---------------------------------------------------------------------------
// Test 8 — ARIA attributes
// ---------------------------------------------------------------------------

describe('TablePagination — ARIA attributes', () => {
  it('Previous button has aria-label="Previous page"', () => {
    const wrapper = mountPagination({ totalItems: 100 })
    const btn = wrapper.find('button[aria-label="Previous page"]')
    expect(btn.exists()).toBe(true)
  })

  it('Next button has aria-label="Next page"', () => {
    const wrapper = mountPagination({ totalItems: 100 })
    const btn = wrapper.find('button[aria-label="Next page"]')
    expect(btn.exists()).toBe(true)
  })

  it('page-jump input has aria-label="Jump to page"', () => {
    const wrapper = mountPagination({ totalItems: 100 })
    const input = wrapper.find('#page-jump-input')
    expect(input.attributes('aria-label')).toBe('Jump to page')
  })

  it('page-jump input has an associated <label> element', () => {
    const wrapper = mountPagination({ totalItems: 100 })
    const label = wrapper.find('label[for="page-jump-input"]')
    expect(label.exists()).toBe(true)
  })

  it('page-size select has aria-label="Rows per page"', () => {
    const wrapper = mountPagination({ totalItems: 100 })
    const select = wrapper.find('#page-size-select')
    expect(select.attributes('aria-label')).toBe('Rows per page')
  })

  it('pagination container has role="navigation"', () => {
    const wrapper = mountPagination({ totalItems: 100 })
    const nav = wrapper.find('[role="navigation"]')
    expect(nav.exists()).toBe(true)
  })
})

// ---------------------------------------------------------------------------
// Test 9 — Keyboard navigation (DOM order)
// ---------------------------------------------------------------------------

describe('TablePagination — keyboard navigation order', () => {
  it('focusable controls appear in logical DOM order: size-select, prev, jump-input, next', () => {
    const wrapper = mountPagination({ totalItems: 100, currentPage: 2 })
    const html = wrapper.html()

    const sizePos  = html.indexOf('id="page-size-select"')
    const prevPos  = html.indexOf('aria-label="Previous page"')
    const jumpPos  = html.indexOf('id="page-jump-input"')
    const nextPos  = html.indexOf('aria-label="Next page"')

    expect(sizePos).toBeGreaterThan(-1)
    expect(prevPos).toBeGreaterThan(-1)
    expect(jumpPos).toBeGreaterThan(-1)
    expect(nextPos).toBeGreaterThan(-1)

    expect(sizePos).toBeLessThan(prevPos)
    expect(prevPos).toBeLessThan(jumpPos)
    expect(jumpPos).toBeLessThan(nextPos)
  })
})
