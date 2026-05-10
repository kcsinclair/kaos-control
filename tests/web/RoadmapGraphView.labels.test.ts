// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 6 — Integration tests for RoadmapGraphView.vue label-toggle wiring.
 *
 * Verifies that RoadmapGraphView renders the two label-control checkboxes,
 * that they default to unchecked, and that clicking them updates the props
 * passed to the embedded ForceGraph3D component.
 *
 * Testing approach
 * ────────────────
 * - RoadmapGraphView is a component (not a view) that takes a project prop.
 * - It uses local refs (showNodeTitles, showNodeLineage) — not Pinia store state.
 * - The releases API (getRoadmapGraph) is mocked to return a small set of nodes
 *   so the ForceGraph3D branch (v-else) renders.
 * - ForceGraph3D is stubbed so Three.js is never invoked.
 * - WebSocket composable is mocked.
 *
 * Acceptance criteria (from test plan Milestone 6):
 *   - Two checkboxes appear in the roadmap 3D view.
 *   - Both default to unchecked.
 *   - Clicking a checkbox updates the prop passed to ForceGraph3D.
 *
 * Component: web/src/components/releases/RoadmapGraphView.vue
 */

import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

// ---------------------------------------------------------------------------
// Mock data factory — defined via vi.hoisted so it's accessible inside vi.mock
// ---------------------------------------------------------------------------

const { makeGraphData } = vi.hoisted(() => {
  function makeGraphData() {
    return {
      nodes: [
        {
          id: 'release:backlog',
          title: 'Backlog',
          type: 'release',
          status: 'draft',
          stage: 'releases',
          lineage: '',
          slug: 'backlog',
          index: 0,
          synthetic: true,
        },
        {
          id: 'idea-1',
          title: 'An idea artifact',
          type: 'idea',
          status: 'in-development',
          stage: 'ideas',
          lineage: 'feature-one',
          slug: 'feature-one',
          index: 1,
        },
      ],
      edges: [],
    }
  }
  return { makeGraphData }
})

// ---------------------------------------------------------------------------
// Module mocks — prevent real network I/O
// ---------------------------------------------------------------------------

vi.mock('@/api/releases', () => ({
  getRoadmapGraph: vi.fn().mockImplementation(() => Promise.resolve(makeGraphData())),
  listReleases: vi.fn().mockResolvedValue({ releases: [] }),
  getRelease: vi.fn().mockResolvedValue(null),
  createRelease: vi.fn().mockResolvedValue(null),
  updateRelease: vi.fn().mockResolvedValue(null),
  deleteRelease: vi.fn().mockResolvedValue(null),
  getRoadmapArtifacts: vi.fn().mockResolvedValue({ artifacts: [] }),
}))

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: vi.fn(),
}))

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

async function mountRoadmapGraphView() {
  const { default: RoadmapGraphView } = await import(
    '../../web/src/components/releases/RoadmapGraphView.vue'
  )

  const wrapper = mount(RoadmapGraphView, {
    props: { project: 'testproject' },
    global: {
      stubs: {
        ForceGraph3D: {
          name: 'ForceGraph3D',
          template: '<div data-testid="force-graph-3d" />',
          props: [
            'nodes', 'edges', 'dagMode',
            'showNodeTitles', 'showNodeLineage',
          ],
          emits: ['nodeClick'],
        },
        // Graph2DView is loaded via defineAsyncComponent — stub it
        Graph2DView: { template: '<div />' },
      },
    },
  })

  // Wait for onMounted → load() → API call to resolve
  await flushPromises()

  return wrapper
}

// ---------------------------------------------------------------------------
// Checkbox-finding helper
// ---------------------------------------------------------------------------

function findCheckboxByText(wrapper: ReturnType<typeof mount>, text: string) {
  return wrapper.findAll('input[type="checkbox"]').find((el) => {
    const label = el.element.closest('label')
    return label?.textContent?.includes(text)
  })
}

// ---------------------------------------------------------------------------
// Cleanup
// ---------------------------------------------------------------------------

afterEach(() => {
  document.body.innerHTML = ''
  vi.clearAllMocks()
})

// ===========================================================================
// Checkbox presence — both checkboxes render
// ===========================================================================

describe('RoadmapGraphView — label-toggle checkboxes present (M6)', () => {
  it('a checkbox labelled "Show node titles" is rendered', async () => {
    const wrapper = await mountRoadmapGraphView()
    const cb = findCheckboxByText(wrapper, 'Show node titles')
    expect(cb, 'expected a checkbox near "Show node titles"').toBeDefined()
    expect(cb!.element.tagName.toLowerCase()).toBe('input')
  })

  it('a checkbox labelled "Show node lineage" is rendered', async () => {
    const wrapper = await mountRoadmapGraphView()
    const cb = findCheckboxByText(wrapper, 'Show node lineage')
    expect(cb, 'expected a checkbox near "Show node lineage"').toBeDefined()
    expect(cb!.element.tagName.toLowerCase()).toBe('input')
  })

  it('checkboxes are type="checkbox"', async () => {
    const wrapper = await mountRoadmapGraphView()
    const titlesCb = findCheckboxByText(wrapper, 'Show node titles')!
    const lineageCb = findCheckboxByText(wrapper, 'Show node lineage')!
    expect((titlesCb.element as HTMLInputElement).type).toBe('checkbox')
    expect((lineageCb.element as HTMLInputElement).type).toBe('checkbox')
  })
})

// ===========================================================================
// Default state — both unchecked on mount
// ===========================================================================

describe('RoadmapGraphView — label-toggle default state (M6)', () => {
  it('"Show node titles" checkbox is unchecked by default', async () => {
    const wrapper = await mountRoadmapGraphView()
    const cb = findCheckboxByText(wrapper, 'Show node titles')!
    expect((cb.element as HTMLInputElement).checked).toBe(false)
  })

  it('"Show node lineage" checkbox is unchecked by default', async () => {
    const wrapper = await mountRoadmapGraphView()
    const cb = findCheckboxByText(wrapper, 'Show node lineage')!
    expect((cb.element as HTMLInputElement).checked).toBe(false)
  })

  it('both checkboxes are unchecked simultaneously on initial render', async () => {
    const wrapper = await mountRoadmapGraphView()
    const titlesCb = findCheckboxByText(wrapper, 'Show node titles')!
    const lineageCb = findCheckboxByText(wrapper, 'Show node lineage')!
    expect((titlesCb.element as HTMLInputElement).checked).toBe(false)
    expect((lineageCb.element as HTMLInputElement).checked).toBe(false)
  })
})

// ===========================================================================
// ForceGraph3D prop binding — props match local ref state
// ===========================================================================

describe('RoadmapGraphView — ForceGraph3D receives label props (M6)', () => {
  it('ForceGraph3D stub receives showNodeTitles=false on initial render', async () => {
    const wrapper = await mountRoadmapGraphView()
    const graphStub = wrapper.findComponent({ name: 'ForceGraph3D' })
    expect(graphStub.exists(), 'ForceGraph3D stub not found').toBe(true)
    expect(graphStub.props('showNodeTitles')).toBe(false)
  })

  it('ForceGraph3D stub receives showNodeLineage=false on initial render', async () => {
    const wrapper = await mountRoadmapGraphView()
    const graphStub = wrapper.findComponent({ name: 'ForceGraph3D' })
    expect(graphStub.props('showNodeLineage')).toBe(false)
  })
})

// ===========================================================================
// Checkbox interaction — clicking updates the prop on ForceGraph3D
// ===========================================================================

describe('RoadmapGraphView — clicking checkbox updates ForceGraph3D prop (M6)', () => {
  it('clicking "Show node titles" sets ForceGraph3D showNodeTitles to true', async () => {
    const wrapper = await mountRoadmapGraphView()
    const cb = findCheckboxByText(wrapper, 'Show node titles')!
    const graphStub = wrapper.findComponent({ name: 'ForceGraph3D' })

    expect(graphStub.props('showNodeTitles')).toBe(false)

    await cb.trigger('change')
    await flushPromises()

    expect(graphStub.props('showNodeTitles')).toBe(true)
  })

  it('clicking "Show node lineage" sets ForceGraph3D showNodeLineage to true', async () => {
    const wrapper = await mountRoadmapGraphView()
    const cb = findCheckboxByText(wrapper, 'Show node lineage')!
    const graphStub = wrapper.findComponent({ name: 'ForceGraph3D' })

    expect(graphStub.props('showNodeLineage')).toBe(false)

    await cb.trigger('change')
    await flushPromises()

    expect(graphStub.props('showNodeLineage')).toBe(true)
  })

  it('clicking each checkbox toggles independently', async () => {
    const wrapper = await mountRoadmapGraphView()
    const titlesCb = findCheckboxByText(wrapper, 'Show node titles')!
    const lineageCb = findCheckboxByText(wrapper, 'Show node lineage')!
    const graphStub = wrapper.findComponent({ name: 'ForceGraph3D' })

    await titlesCb.trigger('change')
    await flushPromises()

    expect(graphStub.props('showNodeTitles')).toBe(true)
    expect(graphStub.props('showNodeLineage')).toBe(false)

    await lineageCb.trigger('change')
    await flushPromises()

    expect(graphStub.props('showNodeTitles')).toBe(true)
    expect(graphStub.props('showNodeLineage')).toBe(true)
  })

  it('unchecking "Show node titles" sets ForceGraph3D showNodeTitles back to false', async () => {
    const wrapper = await mountRoadmapGraphView()
    const cb = findCheckboxByText(wrapper, 'Show node titles')!
    const graphStub = wrapper.findComponent({ name: 'ForceGraph3D' })

    // Enable
    await cb.trigger('change')
    await flushPromises()
    expect(graphStub.props('showNodeTitles')).toBe(true)

    // Disable
    await cb.trigger('change')
    await flushPromises()
    expect(graphStub.props('showNodeTitles')).toBe(false)
  })
})

// ===========================================================================
// Accessibility — checkboxes are wrapped in <label> elements
// ===========================================================================

describe('RoadmapGraphView — label-toggle accessibility (M6)', () => {
  it('"Show node titles" input is inside a <label> element', async () => {
    const wrapper = await mountRoadmapGraphView()
    const cb = findCheckboxByText(wrapper, 'Show node titles')!
    const parentLabel = (cb.element as HTMLInputElement).closest('label')
    expect(parentLabel, 'input must be inside a <label>').not.toBeNull()
  })

  it('"Show node lineage" input is inside a <label> element', async () => {
    const wrapper = await mountRoadmapGraphView()
    const cb = findCheckboxByText(wrapper, 'Show node lineage')!
    const parentLabel = (cb.element as HTMLInputElement).closest('label')
    expect(parentLabel, 'input must be inside a <label>').not.toBeNull()
  })
})
