// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 5b — Accessibility tests for `SortHeader` component
 *
 * Expected component location: web/src/components/SortHeader.vue
 *
 * Expected props:
 *   label:          string         — display text for the column
 *   column:         string         — column key (passed to onToggle)
 *   sortColumn:     string | null  — currently active sort column
 *   sortDirection:  'asc' | 'desc' | null — current direction
 *   sortable?:      boolean        — whether this column is sortable (default true)
 *
 * Expected emits / prop:
 *   onToggle / @toggle: (column: string) => void
 *
 * Accessibility requirements from the test plan:
 *   - role="button" or equivalent semantics (focusable th with keyboard support)
 *   - tabindex="0" on the interactive element
 *   - Activates on Enter key press
 *   - Activates on Space key press
 *   - aria-sort reflects current direction ("ascending", "descending", or "none")
 */

import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import SortHeader from '../../web/src/components/SortHeader.vue'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function mountSortHeader(overrides: Record<string, unknown> = {}) {
  return mount(SortHeader, {
    props: {
      label: 'Path',
      column: 'title',
      sortColumn: null,
      sortDirection: null,
      sortable: true,
      ...overrides,
    },
  })
}

// ---------------------------------------------------------------------------
// Focusability
// ---------------------------------------------------------------------------

describe('SortHeader — focusability', () => {
  it('sortable header is focusable via tabindex="0"', () => {
    const wrapper = mountSortHeader()

    // The interactive element (th or inner button) must have tabindex="0"
    const focusable = wrapper.find('[tabindex="0"]')
    expect(focusable.exists()).toBe(true)
  })

  it('non-sortable header has no tabindex or tabindex="-1"', () => {
    const wrapper = mountSortHeader({ sortable: false })

    const el = wrapper.find('[tabindex="0"]')
    expect(el.exists()).toBe(false)
  })
})

// ---------------------------------------------------------------------------
// aria-sort
// ---------------------------------------------------------------------------

describe('SortHeader — aria-sort attribute', () => {
  it('shows aria-sort="none" when sortable and not the active column', () => {
    const wrapper = mountSortHeader({
      sortColumn: null,
      sortDirection: null,
    })

    // Look for aria-sort on any element within the component
    const el = wrapper.find('[aria-sort]')
    expect(el.exists()).toBe(true)
    expect(el.attributes('aria-sort')).toBe('none')
  })

  it('shows aria-sort="ascending" when this column is sorted ascending', () => {
    const wrapper = mountSortHeader({
      sortColumn: 'title',
      sortDirection: 'asc',
    })

    const el = wrapper.find('[aria-sort]')
    expect(el.exists()).toBe(true)
    expect(el.attributes('aria-sort')).toBe('ascending')
  })

  it('shows aria-sort="descending" when this column is sorted descending', () => {
    const wrapper = mountSortHeader({
      sortColumn: 'title',
      sortDirection: 'desc',
    })

    const el = wrapper.find('[aria-sort]')
    expect(el.exists()).toBe(true)
    expect(el.attributes('aria-sort')).toBe('descending')
  })

  it('shows aria-sort="none" (or no aria-sort) for a different active column', () => {
    // 'title' is sortable but 'stage' is the active sort column
    const wrapper = mountSortHeader({
      column: 'title',
      sortColumn: 'stage',
      sortDirection: 'asc',
    })

    const el = wrapper.find('[aria-sort]')
    if (el.exists()) {
      // If aria-sort is present, it must say "none" (not ascending/descending)
      expect(el.attributes('aria-sort')).toBe('none')
    }
    // Alternatively the attribute may be omitted entirely — both are valid
  })

  it('non-sortable header does not expose aria-sort', () => {
    const wrapper = mountSortHeader({ sortable: false })

    const el = wrapper.find('[aria-sort]')
    expect(el.exists()).toBe(false)
  })
})

// ---------------------------------------------------------------------------
// Keyboard activation — Enter
// ---------------------------------------------------------------------------

describe('SortHeader — keyboard: Enter key', () => {
  it('emits toggle event with column key on Enter press', async () => {
    const onToggle = vi.fn()
    const wrapper = mountSortHeader({ onToggle })

    const focusable = wrapper.find('[tabindex="0"]')
    await focusable.trigger('keydown', { key: 'Enter' })

    expect(onToggle).toHaveBeenCalledOnce()
    expect(onToggle).toHaveBeenCalledWith('title')
  })

  it('does not emit on Enter when not sortable', async () => {
    const onToggle = vi.fn()
    const wrapper = mountSortHeader({ sortable: false, onToggle })

    // Non-sortable columns may not have a focusable element at all;
    // clicking/pressing should be a no-op.
    const th = wrapper.find('th')
    await th.trigger('keydown', { key: 'Enter' })

    expect(onToggle).not.toHaveBeenCalled()
  })
})

// ---------------------------------------------------------------------------
// Keyboard activation — Space
// ---------------------------------------------------------------------------

describe('SortHeader — keyboard: Space key', () => {
  it('emits toggle event with column key on Space press', async () => {
    const onToggle = vi.fn()
    const wrapper = mountSortHeader({ onToggle })

    const focusable = wrapper.find('[tabindex="0"]')
    await focusable.trigger('keydown', { key: ' ' })

    expect(onToggle).toHaveBeenCalledOnce()
    expect(onToggle).toHaveBeenCalledWith('title')
  })
})

// ---------------------------------------------------------------------------
// Click activation
// ---------------------------------------------------------------------------

describe('SortHeader — click activation', () => {
  it('emits toggle event on click', async () => {
    const onToggle = vi.fn()
    const wrapper = mountSortHeader({ onToggle })

    await wrapper.trigger('click')

    expect(onToggle).toHaveBeenCalledOnce()
    expect(onToggle).toHaveBeenCalledWith('title')
  })

  it('non-sortable header does not emit on click', async () => {
    const onToggle = vi.fn()
    const wrapper = mountSortHeader({ sortable: false, onToggle })

    await wrapper.trigger('click')

    expect(onToggle).not.toHaveBeenCalled()
  })
})

// ---------------------------------------------------------------------------
// Label rendering
// ---------------------------------------------------------------------------

describe('SortHeader — label text', () => {
  it('renders the label text', () => {
    const wrapper = mountSortHeader({ label: 'Created' })
    expect(wrapper.text()).toContain('Created')
  })

  it('renders as a <th> element', () => {
    const wrapper = mountSortHeader()
    expect(wrapper.element.tagName.toLowerCase()).toBe('th')
  })
})
