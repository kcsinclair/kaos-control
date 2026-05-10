// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Tests for "Show tests" toggle — graph-show-tests-toggle
 *
 * Milestone 1: Store unit tests — hideTests state and filtering
 *   Verifies the hideTests ref, toggleHideTests action, and the
 *   filteredNodes / filteredEdges computed behaviour in the graph store.
 *
 * Milestone 2: Component tests — GraphFilters checkbox rendering
 *   Verifies the "Show tests" checkbox renders correctly, reflects the
 *   hideTests prop, and emits the toggle event.
 *
 * Milestone 3: Integration smoke tests — both graph views
 *   Verifies end-to-end that the store correctly shows/hides test nodes
 *   when the toggle is used, and that GraphView resets state on remount.
 *
 * Component: web/src/stores/graph.ts
 *            web/src/components/map/MapFilters.vue
 *            web/src/views/project/GraphView.vue
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { mount, flushPromises } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import { useGraphStore } from '../../web/src/stores/graph'
import type { GraphNode, GraphEdge, GraphFilter } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Module mocks — prevent real network I/O and WebSocket connections
// ---------------------------------------------------------------------------

vi.mock('@/api/graph', () => ({
  getGraph: vi.fn().mockResolvedValue({ nodes: [], edges: [] }),
}))

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: vi.fn(),
}))

// ---------------------------------------------------------------------------
// Shared fixtures
// ---------------------------------------------------------------------------

function makeNode(overrides: Partial<GraphNode> = {}): GraphNode {
  return {
    id: 'node-1',
    title: 'Test artifact',
    type: 'test',
    status: 'draft',
    stage: 'tests',
    lineage: 'some-feature',
    slug: 'some-feature',
    index: 5,
    ...overrides,
  }
}

function makeEdge(source: string, target: string, kind = 'parent'): GraphEdge {
  return { source, target, kind }
}

// ===========================================================================
// MILESTONE 1 — Store unit tests
// ===========================================================================

describe('GraphStore — hideTests state and filtering (Milestone 1)', () => {
  let store: ReturnType<typeof useGraphStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    store = useGraphStore()
  })

  it('1. hideTests is true after store initialisation', () => {
    expect(store.hideTests).toBe(true)
  })

  it('2. toggleHideTests flips hideTests from true to false and back', () => {
    expect(store.hideTests).toBe(true)
    store.toggleHideTests()
    expect(store.hideTests).toBe(false)
    store.toggleHideTests()
    expect(store.hideTests).toBe(true)
  })

  it('3. nodes with type === "test" are excluded from filteredNodes when hideTests is true and no type filter', () => {
    store.rawNodes = [
      makeNode({ id: 'test-1', type: 'test' }),
      makeNode({ id: 'idea-1', type: 'idea' }),
    ]
    const ids = store.filteredNodes.map((n) => n.id)
    expect(ids).not.toContain('test-1')
    expect(ids).toContain('idea-1')
  })

  it('4. test nodes are included in filteredNodes when hideTests is false', () => {
    store.rawNodes = [makeNode({ id: 'test-1', type: 'test' })]
    store.toggleHideTests()
    expect(store.filteredNodes.map((n) => n.id)).toContain('test-1')
  })

  it('5. test nodes appear in filteredNodes when hideTests is true but type filter includes "test"', () => {
    store.rawNodes = [makeNode({ id: 'test-1', type: 'test' })]
    store.setFilter({ types: ['test'] })
    // hideTests is still true, but type filter overrides the bypass
    expect(store.hideTests).toBe(true)
    expect(store.filteredNodes.map((n) => n.id)).toContain('test-1')
  })

  it('6. test nodes excluded when hideTests is true and type filter does not include "test"', () => {
    store.rawNodes = [
      makeNode({ id: 'test-1', type: 'test' }),
      makeNode({ id: 'idea-1', type: 'idea' }),
    ]
    store.setFilter({ types: ['idea'] })
    const ids = store.filteredNodes.map((n) => n.id)
    // test-1 excluded: type filter does not include 'test' AND hideTests bypass not triggered
    expect(ids).not.toContain('test-1')
    // idea-1 included: matches the type filter
    expect(ids).toContain('idea-1')
  })

  it('7. edges to/from hidden test nodes are excluded; edges between visible nodes remain', () => {
    store.rawNodes = [
      makeNode({ id: 'idea-1', type: 'idea' }),
      makeNode({ id: 'test-1', type: 'test' }),
      makeNode({ id: 'req-1', type: 'requirement' }),
    ]
    store.rawEdges = [
      makeEdge('idea-1', 'test-1'), // suppressed — test-1 hidden
      makeEdge('idea-1', 'req-1'), // retained — both visible
    ]
    const edgeKeys = store.filteredEdges.map((e) => `${e.source}→${e.target}`)
    expect(edgeKeys).not.toContain('idea-1→test-1')
    expect(edgeKeys).toContain('idea-1→req-1')
  })

  it('8. toggling hideTests does not affect hideTerminal, and vice versa', () => {
    // Toggling hideTests leaves hideTerminal unchanged
    expect(store.hideTerminal).toBe(true)
    store.toggleHideTests()
    expect(store.hideTerminal).toBe(true)

    // Toggling hideTerminal leaves hideTests unchanged
    store.toggleHideTerminal()
    expect(store.hideTerminal).toBe(false)
    expect(store.hideTests).toBe(false) // still flipped from the toggleHideTests above

    store.toggleHideTerminal()
    expect(store.hideTests).toBe(false) // still unchanged
  })
})

// ===========================================================================
// MILESTONE 2 — GraphFilters component tests
// ===========================================================================

describe('GraphFilters — Show tests checkbox (Milestone 2)', () => {
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
    showNodeTitles: true,
    showNodeLineage: false,
    searchText: '',
  }

  async function mountFilters(props: Partial<typeof defaultProps> = {}) {
    const { default: GraphFilters } = await import(
      '../../web/src/components/map/MapFilters.vue'
    )
    return mount(GraphFilters, {
      props: { ...defaultProps, ...props },
    })
  }

  /** Finds the input[type="checkbox"] whose enclosing label contains `text`. */
  function findCheckboxByLabel(wrapper: ReturnType<typeof mount>, text: string) {
    return wrapper.findAll('input[type="checkbox"]').find((el) =>
      el.element.closest('.toggle-label')?.textContent?.includes(text),
    )
  }

  it('1. a checkbox with label text "Show tests" is present', async () => {
    const wrapper = await mountFilters()
    const labels = wrapper.findAll('.toggle-label')
    const showTestsLabel = labels.find((l) => l.text().includes('Show tests'))
    expect(showTestsLabel, 'expected a .toggle-label element containing "Show tests"').toBeDefined()
    expect(showTestsLabel!.find('input[type="checkbox"]').exists()).toBe(true)
  })

  it('2. checkbox is unchecked when hideTests prop is true (checked = !hideTests)', async () => {
    const wrapper = await mountFilters({ hideTests: true })
    const input = findCheckboxByLabel(wrapper, 'Show tests')
    expect(input, 'expected a checkbox near "Show tests" label').toBeDefined()
    expect((input!.element as HTMLInputElement).checked).toBe(false)
  })

  it('3. checkbox is checked when hideTests prop is false', async () => {
    const wrapper = await mountFilters({ hideTests: false })
    const input = findCheckboxByLabel(wrapper, 'Show tests')
    expect(input, 'expected a checkbox near "Show tests" label').toBeDefined()
    expect((input!.element as HTMLInputElement).checked).toBe(true)
  })

  it('4. clicking the checkbox emits "toggleHideTests"', async () => {
    const wrapper = await mountFilters()
    const input = findCheckboxByLabel(wrapper, 'Show tests')!
    await input.trigger('change')
    expect(wrapper.emitted('toggleHideTests'), 'expected toggleHideTests to be emitted').toBeDefined()
    expect(wrapper.emitted('toggleHideTests')).toHaveLength(1)
  })

  it('5. checkbox is keyboard accessible (not disabled, responds to change event)', async () => {
    const wrapper = await mountFilters()
    const input = findCheckboxByLabel(wrapper, 'Show tests')!
    const el = input.element as HTMLInputElement

    // A native <input type="checkbox"> is keyboard-accessible by design when it
    // is not disabled.  Space/Enter fire a 'change' event, which is the same
    // event the component listens to — no extra keyboard handling is needed.
    expect(el.type).toBe('checkbox')
    expect(el.disabled).toBe(false)

    // Verify the change event (fired by Space/Enter in real browsers) emits
    // the toggleHideTests event from the component.
    await input.trigger('change')
    expect(wrapper.emitted('toggleHideTests')).toBeDefined()
  })

  it('6. the checkbox input is wrapped inside a <label> element', async () => {
    const wrapper = await mountFilters()
    const input = findCheckboxByLabel(wrapper, 'Show tests')!
    const parentLabel = (input.element as HTMLInputElement).closest('label')
    expect(parentLabel, 'input must be wrapped inside a <label> element').not.toBeNull()
  })

  it('visual consistency: "Show tests" and "Show completed" are in the same filter group with matching structure', async () => {
    const wrapper = await mountFilters()

    // Both toggles must be inside the same .filter-group container
    const showTestsInput = findCheckboxByLabel(wrapper, 'Show tests')!
    const showCompletedInput = findCheckboxByLabel(wrapper, 'Show completed')!

    const testsGroup = showTestsInput.element.closest('.filter-group')
    const completedGroup = showCompletedInput.element.closest('.filter-group')
    expect(testsGroup).not.toBeNull()
    expect(completedGroup).not.toBeNull()
    expect(testsGroup).toBe(completedGroup)

    // Both labels use the same CSS class structure
    const testsLabel = showTestsInput.element.closest('.toggle-label')
    const completedLabel = showCompletedInput.element.closest('.toggle-label')
    expect(testsLabel).not.toBeNull()
    expect(completedLabel).not.toBeNull()

    // Both inputs share the same toggle-input class
    expect(showTestsInput.element.classList.contains('toggle-input')).toBe(true)
    expect(showCompletedInput.element.classList.contains('toggle-input')).toBe(true)
  })
})

// ===========================================================================
// MILESTONE 3 — Integration smoke tests
// ===========================================================================

describe('MapView integration — Show tests toggle (Milestone 3)', () => {
  // Shared node/edge fixtures that include both test and non-test artifacts
  const nodes: GraphNode[] = [
    makeNode({ id: 'test-1', type: 'test', title: 'Test artifact 1' }),
    makeNode({ id: 'test-2', type: 'test', title: 'Test artifact 2' }),
    makeNode({ id: 'idea-1', type: 'idea', title: 'Idea 1' }),
    makeNode({ id: 'req-1', type: 'requirement', title: 'Requirement 1' }),
  ]
  const edges: GraphEdge[] = [
    makeEdge('idea-1', 'test-1'),
    makeEdge('idea-1', 'req-1'),
  ]

  let store: ReturnType<typeof useGraphStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    store = useGraphStore()
    store.rawNodes = nodes
    store.rawEdges = edges
  })

  it('1. 2D view: test nodes are hidden on load (store hideTests defaults to true)', () => {
    // Graph2DView receives store.augmentedNodes as its :nodes prop.
    // Verify the store's filtered set excludes test nodes by default.
    expect(store.hideTests).toBe(true)
    const visibleIds = store.filteredNodes.map((n) => n.id)
    expect(visibleIds).not.toContain('test-1')
    expect(visibleIds).not.toContain('test-2')
    expect(visibleIds).toContain('idea-1')
    expect(visibleIds).toContain('req-1')
  })

  it('2. 2D view: test nodes and their edges appear after toggle', () => {
    store.toggleHideTests()
    expect(store.hideTests).toBe(false)

    const visibleIds = store.filteredNodes.map((n) => n.id)
    expect(visibleIds).toContain('test-1')
    expect(visibleIds).toContain('test-2')

    const edgeKeys = store.filteredEdges.map((e) => `${e.source}→${e.target}`)
    expect(edgeKeys).toContain('idea-1→test-1')
  })

  it('3. 3D view: test nodes are absent with default hideTests=true', () => {
    // ForceGraph3D receives store.augmentedNodes as its :nodes prop.
    const visibleIds = store.augmentedNodes.map((n) => n.id)
    expect(visibleIds).not.toContain('test-1')
    expect(visibleIds).not.toContain('test-2')
    expect(visibleIds).toContain('idea-1')
  })

  it('4. 3D view: test nodes appear after toggle', () => {
    store.toggleHideTests()
    const visibleIds = store.augmentedNodes.map((n) => n.id)
    expect(visibleIds).toContain('test-1')
    expect(visibleIds).toContain('test-2')
  })

  it('5. toggle state resets to hidden when MapView is (re)mounted', async () => {
    // Create a controlled pinia so we can pre-set hideTests=false, then
    // verify the MapView onMounted hook resets it back to true.
    const pinia = createPinia()
    setActivePinia(pinia)
    const controlledStore = useGraphStore()
    controlledStore.hideTests = false // simulate user having toggled tests visible

    expect(controlledStore.hideTests).toBe(false)

    // Provide a router with the :project param that MapView needs.
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/p/:project/map',
          component: { template: '<div/>' },
        },
      ],
    })
    await router.push('/p/testproject/map')
    await router.isReady()

    const { default: MapView } = await import(
      '../../web/src/views/project/MapView.vue'
    )

    // Mount MapView. Its onMounted hook sets store.hideTests = true.
    // The mocked API returns { nodes: [], edges: [] } so no graph view components
    // are rendered — only the "No artifacts indexed yet." placeholder.
    mount(MapView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          MapFilters: true,
          GraphLegend: true,
          ArtifactModal: true,
          LabelModal: true,
          StatusCheckPanel: true,
          ForceGraph3D: true,
        },
      },
    })
    await flushPromises()

    // MapView.onMounted sets store.hideTests = true — verify the reset.
    expect(controlledStore.hideTests).toBe(true)
  })
})
