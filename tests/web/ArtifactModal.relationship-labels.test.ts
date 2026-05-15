// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 2 — Component tests for ArtifactModal label rendering
 *
 * Covers:
 *   TC1: Outbound parent edge displays "CHILD OF"
 *   TC2: Inbound parent edge displays "PARENT OF"
 *   TC3: All six kinds render correct labels in the right section
 *   TC4: Unknown kind falls back to uppercase in rendered output
 *
 * Component: web/src/components/artifact/ArtifactModal.vue
 * Test plan: lifecycle/test-plans/artefact-relationship-labels-and-links-5-test.md §Milestone 2
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
  getArtifact:   vi.fn(() => new Promise(() => {})), // never resolves — keeps loading state
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

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

function mountModal(node: GraphNode, edges: GraphEdge[]) {
  return mount(ArtifactModal, {
    props: { node, project: 'testproject', edges },
    global: {
      plugins: [createPinia()],
      stubs: {
        Teleport:          true,
        MarkdownPreview:   true,
        TransitionDialog:  true,
        RunAgentDialog:    true,
        ArtifactRunHistory:true,
        RunDetailModal:    true,
        StatusCheckPanel:  true,
      },
    },
  })
}

// ---------------------------------------------------------------------------
// Helpers to read rendered labels
// ---------------------------------------------------------------------------

function outboundKindTexts(wrapper: ReturnType<typeof mountModal>): string[] {
  // Each edge renders .edge-kind inside an .edge-group that contains .edge-group-label "Outbound"
  const groups = wrapper.findAll('.edge-group')
  const outboundGroup = groups.find(g => g.find('.edge-group-label').text() === 'Outbound')
  if (!outboundGroup) return []
  return outboundGroup.findAll('.edge-kind').map(el => el.text())
}

function inboundKindTexts(wrapper: ReturnType<typeof mountModal>): string[] {
  const groups = wrapper.findAll('.edge-group')
  const inboundGroup = groups.find(g => g.find('.edge-group-label').text() === 'Inbound')
  if (!inboundGroup) return []
  return inboundGroup.findAll('.edge-kind').map(el => el.text())
}

// ===========================================================================
// TC1: Outbound parent edge displays "CHILD OF"
// ===========================================================================

describe('TC1: outbound parent edge label', () => {
  it('renders CHILD OF for a parent-kind outbound edge', () => {
    const node = makeNode()
    const edges: GraphEdge[] = [{ source: NODE_ID, target: 'lifecycle/requirements/parent.md', kind: 'parent' }]

    const wrapper = mountModal(node, edges)

    expect(outboundKindTexts(wrapper)).toContain('CHILD OF')
  })

  it('does not show raw kind value "parent" for an outbound parent edge', () => {
    const node = makeNode()
    const edges: GraphEdge[] = [{ source: NODE_ID, target: 'lifecycle/requirements/parent.md', kind: 'parent' }]

    const wrapper = mountModal(node, edges)

    // Raw kind "parent" must not appear as a label
    for (const text of outboundKindTexts(wrapper)) {
      expect(text).not.toBe('parent')
    }
  })
})

// ===========================================================================
// TC2: Inbound parent edge displays "PARENT OF"
// ===========================================================================

describe('TC2: inbound parent edge label', () => {
  it('renders PARENT OF for a parent-kind inbound edge', () => {
    const node = makeNode()
    const edges: GraphEdge[] = [{ source: 'lifecycle/ideas/child.md', target: NODE_ID, kind: 'parent' }]

    const wrapper = mountModal(node, edges)

    expect(inboundKindTexts(wrapper)).toContain('PARENT OF')
  })

  it('places the inbound edge in the Inbound section, not Outbound', () => {
    const node = makeNode()
    const edges: GraphEdge[] = [{ source: 'lifecycle/ideas/child.md', target: NODE_ID, kind: 'parent' }]

    const wrapper = mountModal(node, edges)

    expect(outboundKindTexts(wrapper)).not.toContain('PARENT OF')
    expect(inboundKindTexts(wrapper)).toContain('PARENT OF')
  })
})

// ===========================================================================
// TC3: All six kinds render correct labels in the right section
// ===========================================================================

describe('TC3: all six relationship kinds render correct labels', () => {
  // Define outbound + inbound expected labels per the FR-1 table
  const KIND_EXPECTATIONS: Array<{
    kind: string
    outboundLabel: string
    inboundLabel: string
  }> = [
    { kind: 'parent',     outboundLabel: 'CHILD OF',       inboundLabel: 'PARENT OF' },
    { kind: 'depends_on', outboundLabel: 'DEPENDS ON',     inboundLabel: 'DEPENDED ON BY' },
    { kind: 'blocks',     outboundLabel: 'BLOCKS',         inboundLabel: 'BLOCKED BY' },
    { kind: 'related_to', outboundLabel: 'RELATED TO',     inboundLabel: 'RELATED TO' },
    { kind: 'members',    outboundLabel: 'MEMBER OF',      inboundLabel: 'HAS MEMBER' },
    { kind: 'wiki',       outboundLabel: 'LINKS TO',       inboundLabel: 'LINKED FROM' },
  ]

  for (const { kind, outboundLabel, inboundLabel } of KIND_EXPECTATIONS) {
    it(`outbound ${kind} edge renders "${outboundLabel}"`, () => {
      const node = makeNode()
      const edges: GraphEdge[] = [{ source: NODE_ID, target: `lifecycle/ideas/other.md`, kind }]
      const wrapper = mountModal(node, edges)
      expect(outboundKindTexts(wrapper)).toContain(outboundLabel)
    })

    it(`inbound ${kind} edge renders "${inboundLabel}"`, () => {
      const node = makeNode()
      const edges: GraphEdge[] = [{ source: `lifecycle/ideas/other.md`, target: NODE_ID, kind }]
      const wrapper = mountModal(node, edges)
      expect(inboundKindTexts(wrapper)).toContain(inboundLabel)
    })
  }

  it('mounts with one edge of each kind in both directions without error', () => {
    const node = makeNode()
    const edges: GraphEdge[] = KIND_EXPECTATIONS.flatMap(({ kind }, i) => [
      { source: NODE_ID, target: `lifecycle/ideas/out-${kind}-${i}.md`, kind },
      { source: `lifecycle/ideas/in-${kind}-${i}.md`, target: NODE_ID, kind },
    ])

    const wrapper = mountModal(node, edges)

    // All 6 outbound labels present
    const outLabels = outboundKindTexts(wrapper)
    const inLabels = inboundKindTexts(wrapper)

    for (const { outboundLabel, inboundLabel } of KIND_EXPECTATIONS) {
      expect(outLabels).toContain(outboundLabel)
      expect(inLabels).toContain(inboundLabel)
    }
  })
})

// ===========================================================================
// TC4: Unknown kind falls back to uppercase in rendered output
// ===========================================================================

describe('TC4: unknown kind falls back to uppercase in rendered label', () => {
  it('renders CUSTOM_REL for an outbound edge with kind "custom_rel"', () => {
    const node = makeNode()
    const edges: GraphEdge[] = [{ source: NODE_ID, target: 'lifecycle/ideas/other.md', kind: 'custom_rel' }]

    const wrapper = mountModal(node, edges)

    expect(outboundKindTexts(wrapper)).toContain('CUSTOM_REL')
  })

  it('renders CUSTOM_REL for an inbound edge with kind "custom_rel"', () => {
    const node = makeNode()
    const edges: GraphEdge[] = [{ source: 'lifecycle/ideas/other.md', target: NODE_ID, kind: 'custom_rel' }]

    const wrapper = mountModal(node, edges)

    expect(inboundKindTexts(wrapper)).toContain('CUSTOM_REL')
  })
})

// ===========================================================================
// Structural: edges appear in correct section (inbound vs outbound)
// ===========================================================================

describe('edge section placement', () => {
  it('outbound edges (source == node.id) appear only in the Outbound section', () => {
    const node = makeNode()
    const edges: GraphEdge[] = [
      { source: NODE_ID, target: 'lifecycle/ideas/target.md', kind: 'parent' },
    ]
    const wrapper = mountModal(node, edges)

    expect(outboundKindTexts(wrapper).length).toBe(1)
    expect(inboundKindTexts(wrapper).length).toBe(0)
  })

  it('inbound edges (target == node.id) appear only in the Inbound section', () => {
    const node = makeNode()
    const edges: GraphEdge[] = [
      { source: 'lifecycle/ideas/source.md', target: NODE_ID, kind: 'parent' },
    ]
    const wrapper = mountModal(node, edges)

    expect(outboundKindTexts(wrapper).length).toBe(0)
    expect(inboundKindTexts(wrapper).length).toBe(1)
  })

  it('does not render the edge footer when there are no edges', () => {
    const node = makeNode()
    const wrapper = mountModal(node, [])

    expect(wrapper.find('.modal-footer').exists()).toBe(false)
  })
})
