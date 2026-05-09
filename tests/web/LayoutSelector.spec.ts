// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 1 — Component unit tests for LayoutSelector.vue
 *
 * Covers:
 *   1. Renders a select/dropdown with all five layout options
 *      (fcose, breadthfirst, concentric, circle, dagre).
 *   2. Calls store.setLayout() when the user selects a different option.
 *   3. Reflects the current activeLayout from the store as the selected value.
 *   4. Directed toggle calls store.toggleDirected().
 *   5. The select control has an aria-label attribute.
 *   6. The select is keyboard-navigable (not disabled, focusable).
 *
 * Testing approach
 * ────────────────
 * The component imports the Pinia graph store directly, so each test creates a
 * fresh pinia via createPinia() / setActivePinia() and seeds initial state by
 * calling store actions or mutating refs directly.
 *
 * Component: web/src/components/graph/LayoutSelector.vue
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { useGraphStore } from '../../web/src/stores/graph'

// ---------------------------------------------------------------------------
// Module mocks — prevent real network / WS during store initialisation
// ---------------------------------------------------------------------------

vi.mock('@/api/graph', () => ({
  getGraph: vi.fn().mockResolvedValue({ nodes: [], edges: [] }),
}))

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: vi.fn(),
}))

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

async function mountSelector() {
  const { default: LayoutSelector } = await import(
    '../../web/src/components/graph/LayoutSelector.vue'
  )
  return mount(LayoutSelector)
}

// ---------------------------------------------------------------------------
// Setup — fresh store per test
// ---------------------------------------------------------------------------

let store: ReturnType<typeof useGraphStore>

beforeEach(() => {
  setActivePinia(createPinia())
  store = useGraphStore()
})

// ===========================================================================
// 1. All five layout options are rendered
// ===========================================================================

describe('LayoutSelector — option rendering (Milestone 1 AC1)', () => {
  it('renders a <select> element', async () => {
    const wrapper = await mountSelector()
    expect(wrapper.find('select').exists()).toBe(true)
  })

  it('renders exactly five <option> elements', async () => {
    const wrapper = await mountSelector()
    const options = wrapper.findAll('option')
    expect(options).toHaveLength(5)
  })

  it('includes an option with value "fcose"', async () => {
    const wrapper = await mountSelector()
    const values = wrapper.findAll('option').map((o) => (o.element as HTMLOptionElement).value)
    expect(values).toContain('fcose')
  })

  it('includes an option with value "breadthfirst"', async () => {
    const wrapper = await mountSelector()
    const values = wrapper.findAll('option').map((o) => (o.element as HTMLOptionElement).value)
    expect(values).toContain('breadthfirst')
  })

  it('includes an option with value "concentric"', async () => {
    const wrapper = await mountSelector()
    const values = wrapper.findAll('option').map((o) => (o.element as HTMLOptionElement).value)
    expect(values).toContain('concentric')
  })

  it('includes an option with value "circle"', async () => {
    const wrapper = await mountSelector()
    const values = wrapper.findAll('option').map((o) => (o.element as HTMLOptionElement).value)
    expect(values).toContain('circle')
  })

  it('includes an option with value "dagre"', async () => {
    const wrapper = await mountSelector()
    const values = wrapper.findAll('option').map((o) => (o.element as HTMLOptionElement).value)
    expect(values).toContain('dagre')
  })
})

// ===========================================================================
// 2. Changing the select calls store.setLayout()
// ===========================================================================

describe('LayoutSelector — layout change event (Milestone 1 AC2)', () => {
  it('calls store.setLayout() with the selected key on change', async () => {
    const wrapper = await mountSelector()
    const spy = vi.spyOn(store, 'setLayout')

    const select = wrapper.find('select')
    await select.setValue('breadthfirst')
    await select.trigger('change')

    expect(spy).toHaveBeenCalledWith('breadthfirst')
  })

  it('calls store.setLayout() with "concentric" when concentric is selected', async () => {
    const wrapper = await mountSelector()
    const spy = vi.spyOn(store, 'setLayout')

    const select = wrapper.find('select')
    await select.setValue('concentric')
    await select.trigger('change')

    expect(spy).toHaveBeenCalledWith('concentric')
  })
})

// ===========================================================================
// 3. Reflects activeLayout from the store as the selected value
// ===========================================================================

describe('LayoutSelector — reflects store.activeLayout (Milestone 1 AC3)', () => {
  it('select value matches store.activeLayout ("fcose" by default)', async () => {
    expect(store.activeLayout).toBe('fcose')
    const wrapper = await mountSelector()
    const select = wrapper.find('select').element as HTMLSelectElement
    expect(select.value).toBe('fcose')
  })

  it('select value updates when store.activeLayout changes to "circle"', async () => {
    store.setLayout('circle')
    const wrapper = await mountSelector()
    const select = wrapper.find('select').element as HTMLSelectElement
    expect(select.value).toBe('circle')
  })

  it('select value updates when store.activeLayout changes to "dagre"', async () => {
    store.setLayout('dagre')
    const wrapper = await mountSelector()
    const select = wrapper.find('select').element as HTMLSelectElement
    expect(select.value).toBe('dagre')
  })
})

// ===========================================================================
// 4. Directed toggle calls store.toggleDirected()
// ===========================================================================

describe('LayoutSelector — directed toggle (Milestone 1 AC4)', () => {
  it('clicking the Directed button calls store.toggleDirected()', async () => {
    const wrapper = await mountSelector()
    const spy = vi.spyOn(store, 'toggleDirected')

    const btn = wrapper.find('button')
    expect(btn.exists()).toBe(true)
    await btn.trigger('click')

    expect(spy).toHaveBeenCalledOnce()
  })

  it('Directed button reflects store.directed (false → not active)', async () => {
    expect(store.directed).toBe(false)
    const wrapper = await mountSelector()
    const btn = wrapper.find('button')
    expect(btn.classes()).not.toContain('active')
    expect((btn.element as HTMLButtonElement).getAttribute('aria-pressed')).toBe('false')
  })

  it('Directed button has active class when store.directed is true', async () => {
    store.toggleDirected()
    expect(store.directed).toBe(true)
    const wrapper = await mountSelector()
    const btn = wrapper.find('button')
    expect(btn.classes()).toContain('active')
    expect((btn.element as HTMLButtonElement).getAttribute('aria-pressed')).toBe('true')
  })
})

// ===========================================================================
// 5. Control has aria-label attribute
// ===========================================================================

describe('LayoutSelector — accessibility (Milestone 1 AC5)', () => {
  it('the <select> element has an aria-label attribute', async () => {
    const wrapper = await mountSelector()
    const select = wrapper.find('select')
    expect(select.attributes('aria-label')).toBeTruthy()
  })

  it('the Directed button has an aria-label attribute', async () => {
    const wrapper = await mountSelector()
    const btn = wrapper.find('button')
    expect(btn.attributes('aria-label')).toBeTruthy()
  })

  it('the Directed button has an aria-pressed attribute', async () => {
    const wrapper = await mountSelector()
    const btn = wrapper.find('button')
    expect(btn.attributes('aria-pressed')).toBeDefined()
  })
})

// ===========================================================================
// 6. Control is keyboard-navigable
// ===========================================================================

describe('LayoutSelector — keyboard navigation (Milestone 1 AC6)', () => {
  it('<select> is not disabled when layoutAnimating is false', async () => {
    expect(store.layoutAnimating).toBe(false)
    const wrapper = await mountSelector()
    const select = wrapper.find('select').element as HTMLSelectElement
    expect(select.disabled).toBe(false)
  })

  it('<select> is disabled when store.layoutAnimating is true', async () => {
    store.layoutAnimating = true
    const wrapper = await mountSelector()
    const select = wrapper.find('select').element as HTMLSelectElement
    expect(select.disabled).toBe(true)
  })

  it('Directed button is disabled when store.layoutAnimating is true', async () => {
    store.layoutAnimating = true
    const wrapper = await mountSelector()
    const btn = wrapper.find('button').element as HTMLButtonElement
    expect(btn.disabled).toBe(true)
  })

  it('<select> is a native element (keyboard-accessible by nature)', async () => {
    const wrapper = await mountSelector()
    const select = wrapper.find('select')
    // Native <select> elements are keyboard-accessible by browser default
    expect(select.element.tagName).toBe('SELECT')
  })
})
