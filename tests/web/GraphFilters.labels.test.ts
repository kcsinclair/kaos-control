// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 2 — Component tests for MapFilters.vue label-toggle checkboxes.
 *
 * Verifies that the two new label-control checkboxes:
 *   - "Show node titles"   (id="toggle-show-node-titles")
 *   - "Show node lineage"  (id="toggle-show-node-lineage")
 *
 * …render correctly, reflect their props, emit the right events, and satisfy
 * basic accessibility requirements (each input is wrapped in a <label>).
 *
 * Acceptance criteria (from test plan Milestone 2):
 *   - A checkbox labelled "Show node titles" is rendered.
 *   - A checkbox labelled "Show node lineage" is rendered.
 *   - Both are unchecked when showNodeTitles=false and showNodeLineage=false.
 *   - Clicking "Show node titles" emits toggleShowNodeTitles.
 *   - Clicking "Show node lineage" emits toggleShowNodeLineage.
 *   - Both checkboxes have associated <label> elements (accessibility).
 *
 * Component: web/src/components/map/MapFilters.vue
 */

import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import type { GraphFilter } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Default props — all required fields with safe zero values
// ---------------------------------------------------------------------------

const defaultProps = {
  filter: {
    types: [],
    statuses: [],
    lineages: [],
    labels: [],
    priorities: [],
  } as GraphFilter,
  uniqueTypes: [] as string[],
  uniqueStatuses: [] as string[],
  uniqueLineages: [] as string[],
  uniqueLabels: [] as string[],
  uniquePriorities: [] as string[],
  nodeCount: 0,
  totalCount: 0,
  showLabelNodes: false,
  showReleases: false,
  hideTerminal: true,
  hideTests: true,
  showNodeTitles: false,
  showNodeLineage: false,
  searchText: '',
}

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

async function mountFilters(overrides: Partial<typeof defaultProps> = {}) {
  const { default: MapFilters } = await import(
    '../../web/src/components/map/MapFilters.vue'
  )
  return mount(MapFilters, {
    props: { ...defaultProps, ...overrides },
  })
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Find the <input type="checkbox"> whose enclosing .toggle-label contains text. */
function findCheckboxByLabel(wrapper: ReturnType<typeof mount>, text: string) {
  return wrapper.findAll('input[type="checkbox"]').find((el) =>
    el.element.closest('.toggle-label')?.textContent?.includes(text),
  )
}

// ===========================================================================
// Rendering — both checkboxes are present
// ===========================================================================

describe('MapFilters — label-toggle checkboxes rendered (M2)', () => {
  it('a .toggle-label containing "Show node titles" is present', async () => {
    const wrapper = await mountFilters()
    const labels = wrapper.findAll('.toggle-label')
    const match = labels.find((l) => l.text().includes('Show node titles'))
    expect(match, 'expected a .toggle-label with "Show node titles"').toBeDefined()
    expect(match!.find('input[type="checkbox"]').exists()).toBe(true)
  })

  it('a .toggle-label containing "Show node lineage" is present', async () => {
    const wrapper = await mountFilters()
    const labels = wrapper.findAll('.toggle-label')
    const match = labels.find((l) => l.text().includes('Show node lineage'))
    expect(match, 'expected a .toggle-label with "Show node lineage"').toBeDefined()
    expect(match!.find('input[type="checkbox"]').exists()).toBe(true)
  })
})

// ===========================================================================
// Default state — both unchecked when props are false
// ===========================================================================

describe('MapFilters — label-toggle default state (M2)', () => {
  it('"Show node titles" checkbox is unchecked when showNodeTitles=false', async () => {
    const wrapper = await mountFilters({ showNodeTitles: false })
    const input = findCheckboxByLabel(wrapper, 'Show node titles')
    expect(input, 'checkbox input not found').toBeDefined()
    expect((input!.element as HTMLInputElement).checked).toBe(false)
  })

  it('"Show node lineage" checkbox is unchecked when showNodeLineage=false', async () => {
    const wrapper = await mountFilters({ showNodeLineage: false })
    const input = findCheckboxByLabel(wrapper, 'Show node lineage')
    expect(input, 'checkbox input not found').toBeDefined()
    expect((input!.element as HTMLInputElement).checked).toBe(false)
  })

  it('both checkboxes are unchecked simultaneously when both props are false', async () => {
    const wrapper = await mountFilters({ showNodeTitles: false, showNodeLineage: false })
    const titlesInput = findCheckboxByLabel(wrapper, 'Show node titles')
    const lineageInput = findCheckboxByLabel(wrapper, 'Show node lineage')
    expect((titlesInput!.element as HTMLInputElement).checked).toBe(false)
    expect((lineageInput!.element as HTMLInputElement).checked).toBe(false)
  })
})

// ===========================================================================
// Checked state — checked when props are true
// ===========================================================================

describe('MapFilters — label-toggle checked state (M2)', () => {
  it('"Show node titles" checkbox is checked when showNodeTitles=true', async () => {
    const wrapper = await mountFilters({ showNodeTitles: true })
    const input = findCheckboxByLabel(wrapper, 'Show node titles')
    expect(input, 'checkbox input not found').toBeDefined()
    expect((input!.element as HTMLInputElement).checked).toBe(true)
  })

  it('"Show node lineage" checkbox is checked when showNodeLineage=true', async () => {
    const wrapper = await mountFilters({ showNodeLineage: true })
    const input = findCheckboxByLabel(wrapper, 'Show node lineage')
    expect(input, 'checkbox input not found').toBeDefined()
    expect((input!.element as HTMLInputElement).checked).toBe(true)
  })
})

// ===========================================================================
// Events — toggling emits the correct events
// ===========================================================================

describe('MapFilters — label-toggle emitted events (M2)', () => {
  it('clicking "Show node titles" checkbox emits toggleShowNodeTitles', async () => {
    const wrapper = await mountFilters()
    const input = findCheckboxByLabel(wrapper, 'Show node titles')!
    await input.trigger('change')
    expect(
      wrapper.emitted('toggleShowNodeTitles'),
      'toggleShowNodeTitles not emitted',
    ).toBeDefined()
    expect(wrapper.emitted('toggleShowNodeTitles')).toHaveLength(1)
  })

  it('clicking "Show node lineage" checkbox emits toggleShowNodeLineage', async () => {
    const wrapper = await mountFilters()
    const input = findCheckboxByLabel(wrapper, 'Show node lineage')!
    await input.trigger('change')
    expect(
      wrapper.emitted('toggleShowNodeLineage'),
      'toggleShowNodeLineage not emitted',
    ).toBeDefined()
    expect(wrapper.emitted('toggleShowNodeLineage')).toHaveLength(1)
  })

  it('each toggle emits independently (titles click does not emit lineage event)', async () => {
    const wrapper = await mountFilters()
    const titlesInput = findCheckboxByLabel(wrapper, 'Show node titles')!
    await titlesInput.trigger('change')
    expect(wrapper.emitted('toggleShowNodeTitles')).toHaveLength(1)
    expect(wrapper.emitted('toggleShowNodeLineage')).toBeUndefined()
  })

  it('each toggle emits independently (lineage click does not emit titles event)', async () => {
    const wrapper = await mountFilters()
    const lineageInput = findCheckboxByLabel(wrapper, 'Show node lineage')!
    await lineageInput.trigger('change')
    expect(wrapper.emitted('toggleShowNodeLineage')).toHaveLength(1)
    expect(wrapper.emitted('toggleShowNodeTitles')).toBeUndefined()
  })
})

// ===========================================================================
// Accessibility — inputs are wrapped in <label> elements
// ===========================================================================

describe('MapFilters — label-toggle accessibility (M2)', () => {
  it('"Show node titles" input is wrapped inside a <label> element', async () => {
    const wrapper = await mountFilters()
    const input = findCheckboxByLabel(wrapper, 'Show node titles')!
    const parentLabel = (input.element as HTMLInputElement).closest('label')
    expect(parentLabel, '"Show node titles" input must be inside a <label>').not.toBeNull()
  })

  it('"Show node lineage" input is wrapped inside a <label> element', async () => {
    const wrapper = await mountFilters()
    const input = findCheckboxByLabel(wrapper, 'Show node lineage')!
    const parentLabel = (input.element as HTMLInputElement).closest('label')
    expect(parentLabel, '"Show node lineage" input must be inside a <label>').not.toBeNull()
  })

  it('"Show node titles" input has an id attribute for <label for="..."> linkage', async () => {
    const wrapper = await mountFilters()
    const input = findCheckboxByLabel(wrapper, 'Show node titles')!
    expect(
      (input.element as HTMLInputElement).id,
      'expected non-empty id for explicit label association',
    ).toBeTruthy()
  })

  it('"Show node lineage" input has an id attribute for <label for="..."> linkage', async () => {
    const wrapper = await mountFilters()
    const input = findCheckboxByLabel(wrapper, 'Show node lineage')!
    expect(
      (input.element as HTMLInputElement).id,
      'expected non-empty id for explicit label association',
    ).toBeTruthy()
  })

  it('inputs are not disabled (keyboard-accessible by default)', async () => {
    const wrapper = await mountFilters()
    const titlesInput = findCheckboxByLabel(wrapper, 'Show node titles')!
    const lineageInput = findCheckboxByLabel(wrapper, 'Show node lineage')!
    expect((titlesInput.element as HTMLInputElement).disabled).toBe(false)
    expect((lineageInput.element as HTMLInputElement).disabled).toBe(false)
  })

  it('both label-toggle checkboxes are in the same .filter-group as other toggles', async () => {
    const wrapper = await mountFilters()
    const titlesInput = findCheckboxByLabel(wrapper, 'Show node titles')!
    const lineageInput = findCheckboxByLabel(wrapper, 'Show node lineage')!
    const titlesGroup = titlesInput.element.closest('.filter-group')
    const lineageGroup = lineageInput.element.closest('.filter-group')
    expect(titlesGroup).not.toBeNull()
    expect(lineageGroup).not.toBeNull()
    expect(titlesGroup).toBe(lineageGroup)
  })
})
