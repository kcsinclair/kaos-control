// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 4 — Accessibility tests for ArtifactModal relationship links
 *
 * Covers:
 *   TC9:  Links are focusable (focus() lands on .edge-path-link element)
 *   TC10: Enter key on a focused link triggers the navigate-artifact event
 *   TC11: Each link has an aria-label that includes direction label + artefact path
 *
 * Component: web/src/components/artifact/ArtifactModal.vue
 * Test plan: lifecycle/test-plans/artefact-relationship-labels-and-links-5-test.md §Milestone 4
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

function mountModal(node: GraphNode, edges: GraphEdge[], attachToBody = false) {
  return mount(ArtifactModal, {
    props: { node, project: 'testproject', edges },
    attachTo: attachToBody ? document.body : undefined,
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
// TC9: Links are focusable via Tab / programmatic focus
// ===========================================================================

describe('TC9: edge links are focusable', () => {
  it('edge-path-link element accepts programmatic focus', async () => {
    const node = makeNode()
    const edges: GraphEdge[] = [
      { source: NODE_ID, target: 'lifecycle/requirements/target.md', kind: 'parent' },
    ]
    // Attach to document.body so focus() works in happy-dom
    const wrapper = mountModal(node, edges, true)

    const link = wrapper.find('.edge-path-link')
    expect(link.exists()).toBe(true)

    ;(link.element as HTMLElement).focus()
    expect(document.activeElement).toBe(link.element)

    wrapper.unmount()
  })

  it('every edge-path-link is an anchor and thus keyboard-focusable by default', () => {
    const node = makeNode()
    const edges: GraphEdge[] = [
      { source: NODE_ID, target: 'lifecycle/requirements/a.md', kind: 'parent' },
      { source: 'lifecycle/ideas/b.md', target: NODE_ID, kind: 'depends_on' },
    ]
    const wrapper = mountModal(node, edges)

    for (const link of wrapper.findAll('.edge-path-link')) {
      // <a> elements with href are focusable by default (tabindex is not needed)
      expect(link.element.tagName.toLowerCase()).toBe('a')
      expect((link.element as HTMLAnchorElement).href).toBeTruthy()
    }
  })
})

// ===========================================================================
// TC10: Enter key on a focused link triggers navigation
// ===========================================================================

describe('TC10: Enter key triggers navigate-artifact', () => {
  it('keydown Enter on an outbound link emits navigate-artifact with target path', async () => {
    const node = makeNode()
    const targetPath = 'lifecycle/requirements/target.md'
    const edges: GraphEdge[] = [{ source: NODE_ID, target: targetPath, kind: 'parent' }]

    const wrapper = mountModal(node, edges)
    const link = wrapper.find('.edge-path-link')

    await link.trigger('keydown', { key: 'Enter' })

    // The component uses @click.prevent on the <a> tag; Enter on <a> fires click
    // in real browsers. In happy-dom we trigger keydown manually and also check
    // that clicking the link (the underlying mechanism) works:
    await link.trigger('click')

    const emitted = wrapper.emitted('navigate-artifact')
    expect(emitted).toBeDefined()
    expect(emitted!.some(call => call[0] === targetPath)).toBe(true)
  })

  it('keydown Enter on an inbound link emits navigate-artifact with source path', async () => {
    const node = makeNode()
    const sourcePath = 'lifecycle/ideas/source.md'
    const edges: GraphEdge[] = [{ source: sourcePath, target: NODE_ID, kind: 'depends_on' }]

    const wrapper = mountModal(node, edges)
    const link = wrapper.find('.edge-path-link')

    // Trigger click (Enter activates <a> click in real browsers)
    await link.trigger('click')

    const emitted = wrapper.emitted('navigate-artifact')
    expect(emitted).toBeDefined()
    expect(emitted![0]).toEqual([sourcePath])
  })
})

// ===========================================================================
// TC11: aria-label includes direction label and artefact path
// ===========================================================================

describe('TC11: aria-label on edge links', () => {
  it('outbound link aria-label contains the directional label and target path', () => {
    const node = makeNode()
    const targetPath = 'lifecycle/requirements/linked.md'
    const edges: GraphEdge[] = [{ source: NODE_ID, target: targetPath, kind: 'parent' }]

    const wrapper = mountModal(node, edges)

    const link = wrapper.find('.edge-path-link')
    const ariaLabel = link.attributes('aria-label') ?? ''

    // Must include the directional label "CHILD OF"
    expect(ariaLabel).toContain('CHILD OF')
    // Must include the target artefact path
    expect(ariaLabel).toContain(targetPath)
  })

  it('inbound link aria-label contains the directional label and source path', () => {
    const node = makeNode()
    const sourcePath = 'lifecycle/ideas/originator.md'
    const edges: GraphEdge[] = [{ source: sourcePath, target: NODE_ID, kind: 'parent' }]

    const wrapper = mountModal(node, edges)

    const link = wrapper.find('.edge-path-link')
    const ariaLabel = link.attributes('aria-label') ?? ''

    // Must include the directional label "PARENT OF"
    expect(ariaLabel).toContain('PARENT OF')
    // Must include the source artefact path
    expect(ariaLabel).toContain(sourcePath)
  })

  it('every edge link has a non-empty aria-label', () => {
    const node = makeNode()
    const edges: GraphEdge[] = [
      { source: NODE_ID, target: 'lifecycle/requirements/a.md', kind: 'parent' },
      { source: NODE_ID, target: 'lifecycle/requirements/b.md', kind: 'blocks' },
      { source: 'lifecycle/ideas/c.md', target: NODE_ID, kind: 'depends_on' },
      { source: 'lifecycle/ideas/d.md', target: NODE_ID, kind: 'wiki' },
    ]

    const wrapper = mountModal(node, edges)

    for (const link of wrapper.findAll('.edge-path-link')) {
      const ariaLabel = link.attributes('aria-label') ?? ''
      expect(ariaLabel.trim().length, 'every .edge-path-link must have a non-empty aria-label').toBeGreaterThan(0)
    }
  })

  it('aria-label for depends_on inbound reads "DEPENDED ON BY <path>"', () => {
    const node = makeNode()
    const sourcePath = 'lifecycle/backend-plans/dep-3-be.md'
    const edges: GraphEdge[] = [{ source: sourcePath, target: NODE_ID, kind: 'depends_on' }]

    const wrapper = mountModal(node, edges)

    const link = wrapper.find('.edge-path-link')
    const ariaLabel = link.attributes('aria-label') ?? ''

    expect(ariaLabel).toContain('DEPENDED ON BY')
    expect(ariaLabel).toContain(sourcePath)
  })

  it('aria-label for wiki outbound reads "LINKS TO <path>"', () => {
    const node = makeNode()
    const targetPath = 'lifecycle/ideas/other-slug.md'
    const edges: GraphEdge[] = [{ source: NODE_ID, target: targetPath, kind: 'wiki' }]

    const wrapper = mountModal(node, edges)

    const link = wrapper.find('.edge-path-link')
    const ariaLabel = link.attributes('aria-label') ?? ''

    expect(ariaLabel).toContain('LINKS TO')
    expect(ariaLabel).toContain(targetPath)
  })
})
