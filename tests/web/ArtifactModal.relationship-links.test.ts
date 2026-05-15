// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 3 — Component tests for ArtifactModal clickable navigation
 *
 * Covers:
 *   TC5: Clicking an outbound edge emits navigate-artifact with the target path
 *   TC6: Clicking an inbound edge emits navigate-artifact with the source path
 *   TC7: Edge path elements are <a> tags with cursor-pointer class
 *   TC8: Hover state applies highlight styling (class or attribute check)
 *
 * Component: web/src/components/artifact/ArtifactModal.vue
 * Test plan: lifecycle/test-plans/artefact-relationship-labels-and-links-5-test.md §Milestone 3
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ArtifactModal from '../../web/src/components/artifact/ArtifactModal.vue'
import type { GraphNode, GraphEdge } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Module-level mocks
// ---------------------------------------------------------------------------

vi.mock('vue-router', () => ({
  useRouter: vi.fn(() => ({ push: vi.fn() })),
}))

vi.mock('@/api/artifacts', () => ({
  getArtifact:   vi.fn(() => new Promise(() => {})),
  patchPriority: vi.fn().mockResolvedValue({}),
  listArtifacts: vi.fn().mockResolvedValue({ items: [], total: 0 }),
  listLabels:    vi.fn().mockResolvedValue({ labels: [] }),
  listPriorities:vi.fn().mockResolvedValue({ priorities: [] }),
  listLineages:  vi.fn().mockResolvedValue({ lineages: [] }),
}))

vi.mock('@/api/agents', () => ({
  listRunsByTargetPath: vi.fn().mockResolvedValue([]),
  listRuns:             vi.fn().mockResolvedValue({ runs: [] }),
  listAgents:           vi.fn().mockResolvedValue({ agents: [] }),
  startRun:             vi.fn().mockResolvedValue({ run_id: 'mock-run' }),
  killRun:              vi.fn().mockResolvedValue({}),
  getRun:               vi.fn().mockResolvedValue({ run: null }),
  getRunLog:            vi.fn().mockResolvedValue(''),
}))

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  setActivePinia(createPinia())
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
})

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

const NODE_ID = 'lifecycle/ideas/current.md'

function makeNode(overrides: Partial<GraphNode> = {}): GraphNode {
  return {
    id:      NODE_ID,
    title:   'Current Artifact',
    type:    'idea',
    status:  'draft',
    stage:   'ideas',
    lineage: 'current',
    slug:    'current',
    index:   1,
    labels:  [],
    ...overrides,
  }
}

function mountModal(node: GraphNode, edges: GraphEdge[]) {
  return mount(ArtifactModal, {
    props: { node, project: 'testproject', edges },
    global: {
      plugins: [createPinia()],
      stubs: {
        Teleport:           true,
        MarkdownPreview:    true,
        TransitionDialog:   true,
        RunAgentDialog:     true,
        ArtifactRunHistory: true,
        RunDetailModal:     true,
        StatusCheckPanel:   true,
      },
    },
  })
}

// ===========================================================================
// TC5: Clicking an outbound edge emits navigate-artifact with target path
// ===========================================================================

describe('TC5: click outbound edge → emits navigate-artifact with target', () => {
  it('emits navigate-artifact with the target path when an outbound link is clicked', async () => {
    const node = makeNode()
    const targetPath = 'lifecycle/requirements/linked.md'
    const edges: GraphEdge[] = [{ source: NODE_ID, target: targetPath, kind: 'parent' }]

    const wrapper = mountModal(node, edges)

    const link = wrapper.find('.edge-path-link')
    expect(link.exists()).toBe(true)
    await link.trigger('click')

    const emitted = wrapper.emitted('navigate-artifact')
    expect(emitted).toBeDefined()
    expect(emitted![0]).toEqual([targetPath])
  })

  it('emits the correct target path even when multiple outbound edges are present', async () => {
    const node = makeNode()
    const target1 = 'lifecycle/requirements/linked-1.md'
    const target2 = 'lifecycle/requirements/linked-2.md'
    const edges: GraphEdge[] = [
      { source: NODE_ID, target: target1, kind: 'depends_on' },
      { source: NODE_ID, target: target2, kind: 'blocks' },
    ]

    const wrapper = mountModal(node, edges)

    const links = wrapper.findAll('.edge-path-link')
    expect(links.length).toBeGreaterThanOrEqual(2)

    // Click the second link
    await links[1].trigger('click')

    const emitted = wrapper.emitted('navigate-artifact')
    expect(emitted).toBeDefined()
    expect(emitted![0]).toEqual([target2])
  })
})

// ===========================================================================
// TC6: Clicking an inbound edge emits navigate-artifact with source path
// ===========================================================================

describe('TC6: click inbound edge → emits navigate-artifact with source', () => {
  it('emits navigate-artifact with the source path when an inbound link is clicked', async () => {
    const node = makeNode()
    const sourcePath = 'lifecycle/ideas/originator.md'
    const edges: GraphEdge[] = [{ source: sourcePath, target: NODE_ID, kind: 'parent' }]

    const wrapper = mountModal(node, edges)

    const link = wrapper.find('.edge-path-link')
    expect(link.exists()).toBe(true)
    await link.trigger('click')

    const emitted = wrapper.emitted('navigate-artifact')
    expect(emitted).toBeDefined()
    expect(emitted![0]).toEqual([sourcePath])
  })

  it('emits the source path for a mixed inbound+outbound scenario', async () => {
    const node = makeNode()
    const outboundTarget = 'lifecycle/requirements/target.md'
    const inboundSource  = 'lifecycle/ideas/source.md'
    const edges: GraphEdge[] = [
      { source: NODE_ID,      target: outboundTarget, kind: 'parent' },
      { source: inboundSource, target: NODE_ID,        kind: 'depends_on' },
    ]

    const wrapper = mountModal(node, edges)

    const links = wrapper.findAll('.edge-path-link')
    // Outbound section comes first → links[0] is outbound, links[1] is inbound
    await links[1].trigger('click')

    const emitted = wrapper.emitted('navigate-artifact')
    expect(emitted).toBeDefined()
    expect(emitted![0]).toEqual([inboundSource])
  })
})

// ===========================================================================
// TC7: Links are <a> elements — semantic HTML and cursor-pointer class
// ===========================================================================

describe('TC7: edge links are <a> elements', () => {
  it('renders edge paths as <a> tags', () => {
    const node = makeNode()
    const edges: GraphEdge[] = [
      { source: NODE_ID, target: 'lifecycle/requirements/target.md', kind: 'parent' },
    ]
    const wrapper = mountModal(node, edges)

    const link = wrapper.find('.edge-path-link')
    expect(link.exists()).toBe(true)
    expect(link.element.tagName.toLowerCase()).toBe('a')
  })

  it('edge link has the edge-path-link class (cursor: pointer applied via CSS)', () => {
    const node = makeNode()
    const edges: GraphEdge[] = [
      { source: NODE_ID, target: 'lifecycle/requirements/target.md', kind: 'parent' },
    ]
    const wrapper = mountModal(node, edges)

    const link = wrapper.find('.edge-path-link')
    expect(link.classes()).toContain('edge-path-link')
  })

  it('all edge path links are <a> elements for multiple edges', () => {
    const node = makeNode()
    const edges: GraphEdge[] = [
      { source: NODE_ID, target: 'lifecycle/requirements/a.md', kind: 'parent' },
      { source: NODE_ID, target: 'lifecycle/requirements/b.md', kind: 'blocks' },
      { source: 'lifecycle/ideas/c.md', target: NODE_ID, kind: 'depends_on' },
    ]
    const wrapper = mountModal(node, edges)

    for (const link of wrapper.findAll('.edge-path-link')) {
      expect(link.element.tagName.toLowerCase()).toBe('a')
    }
  })

  it('clicking the link does not cause a full-page navigation (href="#" with .prevent)', async () => {
    const node = makeNode()
    const edges: GraphEdge[] = [
      { source: NODE_ID, target: 'lifecycle/requirements/target.md', kind: 'parent' },
    ]
    const wrapper = mountModal(node, edges)

    // The link uses @click.prevent — it should emit an event, not follow href
    const link = wrapper.find('.edge-path-link')
    await link.trigger('click')

    // If navigate-artifact was emitted, the Vue click handler ran (not native navigation)
    const emitted = wrapper.emitted('navigate-artifact')
    expect(emitted).toBeDefined()
  })
})

// ===========================================================================
// TC8: Hover state applies highlight class/styling
// ===========================================================================

describe('TC8: hover state on edge links', () => {
  it('edge-path-link element exists and has the edge-path class for hover styling', () => {
    const node = makeNode()
    const edges: GraphEdge[] = [
      { source: NODE_ID, target: 'lifecycle/requirements/target.md', kind: 'parent' },
    ]
    const wrapper = mountModal(node, edges)

    // Both .edge-path and .edge-path-link classes must be present — the CSS
    // defines :hover on .edge-path-link which changes colour (cursor pointer +
    // colour via var(--link-highlight)).
    const link = wrapper.find('.edge-path-link')
    expect(link.classes()).toContain('edge-path')
    expect(link.classes()).toContain('edge-path-link')
  })

  it('mouseenter does not throw and the link remains in the DOM', async () => {
    const node = makeNode()
    const edges: GraphEdge[] = [
      { source: NODE_ID, target: 'lifecycle/requirements/target.md', kind: 'parent' },
    ]
    const wrapper = mountModal(node, edges)

    const link = wrapper.find('.edge-path-link')
    // Trigger mouseenter — the component uses CSS :hover, not JS. We just
    // verify the element survives the event without throwing.
    await link.trigger('mouseenter')
    expect(link.exists()).toBe(true)
  })

  it('mouseleave does not remove the link element from the DOM', async () => {
    const node = makeNode()
    const edges: GraphEdge[] = [
      { source: NODE_ID, target: 'lifecycle/requirements/target.md', kind: 'parent' },
    ]
    const wrapper = mountModal(node, edges)

    const link = wrapper.find('.edge-path-link')
    await link.trigger('mouseenter')
    await link.trigger('mouseleave')
    expect(link.exists()).toBe(true)
  })
})
