// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 5 — Integration tests for MapView.vue label-toggle wiring.
 *
 * Verifies that MapView correctly wires the Pinia graph store's label-toggle
 * state to the GraphFilters (MapFilters.vue) and ForceGraph3D component props,
 * and that toggle events from GraphFilters call the corresponding store actions.
 *
 * Testing approach
 * ────────────────
 * - MapView is mounted with a real Pinia store.
 * - All heavy child components are stubbed.
 * - The API mock returns a visible idea node so the graph template branch
 *   (v-else) renders ForceGraph3D rather than the "No artifacts" placeholder.
 * - Components are located via findComponent(importedDefinition) which works
 *   reliably even when a component is stubbed.
 * - Store action spies are installed BEFORE mounting so the template captures
 *   the spy function reference.
 * - Event wiring is validated by (a) calling $emit on the stub and checking the
 *   spy, and (b) observing the resulting store state mutation.
 *
 * Acceptance criteria (from test plan Milestone 5):
 *   - GraphFilters receives showNodeTitles and showNodeLineage from store state.
 *   - ForceGraph3D receives showNodeTitles and showNodeLineage from store state.
 *   - Emitting toggleShowNodeTitles from GraphFilters calls store.toggleShowNodeTitles().
 *   - Emitting toggleShowNodeLineage from GraphFilters calls store.toggleShowNodeLineage().
 *
 * View: web/src/views/project/MapView.vue
 * Store: web/src/stores/graph.ts
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { useGraphStore } from '../../web/src/stores/graph'
import type { GraphNode } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Module mocks — prevent real network I/O and WebSocket connections.
// The API returns a visible idea node so the graph template branch renders.
// ---------------------------------------------------------------------------

vi.mock('@/api/graph', () => ({
  getGraph: vi.fn().mockResolvedValue({
    nodes: [{
      id: 'idea-1',
      title: 'A test idea',
      type: 'idea',
      status: 'in-development', // not terminal, not test → passes filteredNodes
      stage: 'ideas',
      lineage: 'feature-one',
      slug: 'feature-one',
      index: 1,
    }],
    edges: [],
  }),
}))

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: vi.fn(),
}))

// ---------------------------------------------------------------------------
// Component stubs — defined outside beforeEach so we can use them as keys
// in findComponent() calls
// ---------------------------------------------------------------------------

const ForceGraph3DStub = {
  name: 'ForceGraph3D',
  template: '<div data-testid="force-graph-3d" />',
  props: ['nodes', 'edges', 'matchedNodeIds', 'showNodeTitles', 'showNodeLineage', 'dagMode'],
  emits: ['nodeClick'],
}

const GraphFiltersStub = {
  name: 'GraphFilters',
  template: '<div data-testid="map-filters" />',
  props: [
    'filter', 'uniqueTypes', 'uniqueStatuses', 'uniqueLineages',
    'uniqueLabels', 'uniquePriorities', 'nodeCount', 'totalCount',
    'showLabelNodes', 'showReleases', 'hideTerminal', 'hideTests',
    'showNodeTitles', 'showNodeLineage', 'searchText',
  ],
  emits: [
    'toggle', 'reset', 'toggleLabelNodes', 'toggleShowReleases',
    'toggleHideTerminal', 'toggleHideTests',
    'toggleShowNodeTitles', 'toggleShowNodeLineage',
    'update:searchText',
  ],
}

// ---------------------------------------------------------------------------
// Mount helper — spyOn before mount so template captures spy references
// ---------------------------------------------------------------------------

async function mountMapView(
  storeSetup?: (store: ReturnType<typeof useGraphStore>) => void,
) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const store = useGraphStore()

  if (storeSetup) storeSetup(store)

  // Install spies before mounting — Vue captures handler refs at render time
  const titleSpy = vi.spyOn(store, 'toggleShowNodeTitles')
  const lineageSpy = vi.spyOn(store, 'toggleShowNodeLineage')

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

  const { default: MapView } = await import('../../web/src/views/project/MapView.vue')

  const wrapper = mount(MapView, {
    global: {
      plugins: [pinia, router],
      stubs: {
        // Use the ForceGraph3D and GraphFilters stub objects defined above
        ForceGraph3D: ForceGraph3DStub,
        // MapFilters.vue is imported as 'GraphFilters' in MapView
        GraphFilters: GraphFiltersStub,
        // Stub remaining child components
        Graph2DView: { template: '<div />' },
        GraphLegend: { template: '<div />' },
        LayoutSelector: { template: '<div />' },
        ArtifactModal: { template: '<div />' },
        LabelModal: { template: '<div />' },
        StatusCheckPanel: { template: '<div />' },
        MapLegend: { template: '<div />' },
      },
    },
  })

  // Wait for onMounted → fetchGraph → API resolves → store.rawNodes populated
  await flushPromises()

  return { wrapper, store, titleSpy, lineageSpy }
}

// ---------------------------------------------------------------------------
// Cleanup
// ---------------------------------------------------------------------------

afterEach(() => {
  document.body.innerHTML = ''
  vi.clearAllMocks()
})

// ===========================================================================
// Prop binding — GraphFilters receives the store's label-toggle state
// ===========================================================================

describe('MapView — GraphFilters label prop bindings (M5)', () => {
  beforeEach(() => { setActivePinia(createPinia()) })

  it('GraphFilters receives showNodeTitles=false (store default)', async () => {
    const { wrapper } = await mountMapView()
    const filtersStub = wrapper.findComponent(GraphFiltersStub)
    expect(filtersStub.exists(), 'GraphFilters stub not found').toBe(true)
    expect(filtersStub.props('showNodeTitles')).toBe(false)
  })

  it('GraphFilters receives showNodeLineage=false (store default)', async () => {
    const { wrapper } = await mountMapView()
    const filtersStub = wrapper.findComponent(GraphFiltersStub)
    expect(filtersStub.props('showNodeLineage')).toBe(false)
  })

  it('GraphFilters receives showNodeTitles=true when store has it set', async () => {
    const { wrapper } = await mountMapView((store) => { store.showNodeTitles = true })
    const filtersStub = wrapper.findComponent(GraphFiltersStub)
    expect(filtersStub.props('showNodeTitles')).toBe(true)
  })

  it('GraphFilters receives showNodeLineage=true when store has it set', async () => {
    const { wrapper } = await mountMapView((store) => { store.showNodeLineage = true })
    const filtersStub = wrapper.findComponent(GraphFiltersStub)
    expect(filtersStub.props('showNodeLineage')).toBe(true)
  })
})

// ===========================================================================
// Prop binding — ForceGraph3D receives the store's label-toggle state
// ===========================================================================

describe('MapView — ForceGraph3D label prop bindings (M5)', () => {
  beforeEach(() => { setActivePinia(createPinia()) })

  it('ForceGraph3D receives showNodeTitles=false (store default)', async () => {
    const { wrapper } = await mountMapView()
    const graphStub = wrapper.findComponent(ForceGraph3DStub)
    expect(graphStub.exists(), 'ForceGraph3D stub not found — check rawNodes/loading state').toBe(true)
    expect(graphStub.props('showNodeTitles')).toBe(false)
  })

  it('ForceGraph3D receives showNodeLineage=false (store default)', async () => {
    const { wrapper } = await mountMapView()
    const graphStub = wrapper.findComponent(ForceGraph3DStub)
    expect(graphStub.props('showNodeLineage')).toBe(false)
  })

  it('ForceGraph3D receives showNodeTitles=true when store has it set', async () => {
    const { wrapper } = await mountMapView((store) => { store.showNodeTitles = true })
    const graphStub = wrapper.findComponent(ForceGraph3DStub)
    expect(graphStub.props('showNodeTitles')).toBe(true)
  })

  it('ForceGraph3D receives showNodeLineage=true when store has it set', async () => {
    const { wrapper } = await mountMapView((store) => { store.showNodeLineage = true })
    const graphStub = wrapper.findComponent(ForceGraph3DStub)
    expect(graphStub.props('showNodeLineage')).toBe(true)
  })
})

// ===========================================================================
// Prop reactivity — ForceGraph3D prop updates when store state changes
// ===========================================================================

describe('MapView — ForceGraph3D label props update reactively (M5)', () => {
  beforeEach(() => { setActivePinia(createPinia()) })

  it('showNodeTitles prop updates when store.toggleShowNodeTitles() is called', async () => {
    const { wrapper, store } = await mountMapView()
    const graphStub = wrapper.findComponent(ForceGraph3DStub)
    expect(graphStub.exists(), 'ForceGraph3D stub not rendered').toBe(true)
    expect(graphStub.props('showNodeTitles')).toBe(false)

    store.toggleShowNodeTitles()
    await flushPromises()

    expect(graphStub.props('showNodeTitles')).toBe(true)
  })

  it('showNodeLineage prop updates when store.toggleShowNodeLineage() is called', async () => {
    const { wrapper, store } = await mountMapView()
    const graphStub = wrapper.findComponent(ForceGraph3DStub)
    expect(graphStub.exists(), 'ForceGraph3D stub not rendered').toBe(true)
    expect(graphStub.props('showNodeLineage')).toBe(false)

    store.toggleShowNodeLineage()
    await flushPromises()

    expect(graphStub.props('showNodeLineage')).toBe(true)
  })
})

// ===========================================================================
// Event wiring — GraphFilters events call store toggle methods
// ===========================================================================

describe('MapView — GraphFilters toggle events wire to store actions (M5)', () => {
  beforeEach(() => { setActivePinia(createPinia()) })

  it('toggleShowNodeTitles event from GraphFilters calls store.toggleShowNodeTitles()', async () => {
    const { wrapper, titleSpy } = await mountMapView()
    const filtersStub = wrapper.findComponent(GraphFiltersStub)
    expect(filtersStub.exists()).toBe(true)

    await filtersStub.vm.$emit('toggleShowNodeTitles')
    await flushPromises()

    expect(titleSpy, 'store.toggleShowNodeTitles was not called').toHaveBeenCalledOnce()
  })

  it('toggleShowNodeLineage event from GraphFilters calls store.toggleShowNodeLineage()', async () => {
    const { wrapper, lineageSpy } = await mountMapView()
    const filtersStub = wrapper.findComponent(GraphFiltersStub)
    expect(filtersStub.exists()).toBe(true)

    await filtersStub.vm.$emit('toggleShowNodeLineage')
    await flushPromises()

    expect(lineageSpy, 'store.toggleShowNodeLineage was not called').toHaveBeenCalledOnce()
  })

  it('toggleShowNodeTitles event mutates store.showNodeTitles to true', async () => {
    const { wrapper, store } = await mountMapView()
    expect(store.showNodeTitles).toBe(false)

    const filtersStub = wrapper.findComponent(GraphFiltersStub)
    await filtersStub.vm.$emit('toggleShowNodeTitles')
    await flushPromises()

    expect(store.showNodeTitles).toBe(true)
  })

  it('toggleShowNodeLineage event mutates store.showNodeLineage to true', async () => {
    const { wrapper, store } = await mountMapView()
    expect(store.showNodeLineage).toBe(false)

    const filtersStub = wrapper.findComponent(GraphFiltersStub)
    await filtersStub.vm.$emit('toggleShowNodeLineage')
    await flushPromises()

    expect(store.showNodeLineage).toBe(true)
  })
})
