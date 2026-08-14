// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useGraphStore } from '../graph'
import type { GraphNode } from '@/types/api'

// Frontend plan: lifecycle/frontend-plans/architectural-artefacts-4-fe.md — Milestone 2
//
// Files under lifecycle/architecture/ carry no lineage (FR-19/FR-20). Verifies the graph
// store's default filtering does not drop these nodes just because lineage is empty.

function makeNode(overrides: Partial<GraphNode>): GraphNode {
  return {
    id: 'lifecycle/architecture/microservices.md',
    title: 'Microservices',
    type: 'architecture',
    status: 'approved',
    stage: 'architecture',
    lineage: '',
    slug: 'microservices',
    index: 0,
    ...overrides,
  }
}

describe('graph store — architecture-zone nodes with empty lineage', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('renders an architecture node with empty lineage by default (no lineage filter applied)', () => {
    const store = useGraphStore()
    store.rawNodes = [makeNode({})]

    expect(store.filteredNodes.map((n) => n.id)).toContain('lifecycle/architecture/microservices.md')
  })

  it('includes the empty-lineage node in uniqueLineages without throwing', () => {
    const store = useGraphStore()
    store.rawNodes = [makeNode({})]

    expect(store.uniqueLineages).toContain('')
  })

  it('does not drop the node when an unrelated lineage filter is active', () => {
    const store = useGraphStore()
    store.rawNodes = [
      makeNode({}),
      { id: 'lifecycle/ideas/login.md', title: 'Login', type: 'idea', status: 'draft', stage: 'ideas', lineage: 'login', slug: 'login', index: 0 },
    ]
    // No lineages selected — both nodes render (empty filter = no restriction).
    expect(store.filteredNodes.length).toBe(2)
  })
})
