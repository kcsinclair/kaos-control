/**
 * Seed helpers — factory functions that produce typed fake objects for use
 * in Vitest unit tests. No network calls are made; all data is in-memory.
 */

import type { ArtifactRow, GraphNode, GraphEdge } from '@/types/api'

/** All workflow statuses defined by the spec, including the three terminal ones. */
export const ALL_STATUSES = [
  'draft',
  'clarifying',
  'planning',
  'in-development',
  'in-qa',
  'done',
  'rejected',
  'abandoned',
] as const

export type ArtifactStatus = (typeof ALL_STATUSES)[number]

/** Terminal statuses as a plain array for use in test assertions. */
export const TERMINAL_STATUSES: ArtifactStatus[] = ['done', 'rejected', 'abandoned']

/** Non-terminal statuses. */
export const ACTIVE_STATUSES: ArtifactStatus[] = [
  'draft',
  'clarifying',
  'planning',
  'in-development',
  'in-qa',
]

/** Build a minimal ArtifactRow with sensible defaults, overriding any fields. */
export function makeArtifactRow(overrides: Partial<ArtifactRow> = {}): ArtifactRow {
  const status = overrides.status ?? 'draft'
  const slug = overrides.slug ?? `test-${status}`
  return {
    path: `lifecycle/ideas/${slug}.md`,
    slug,
    lineage: slug,
    index: 1,
    stage: 'ideas',
    type: 'idea',
    status,
    title: `Test Artifact (${status})`,
    frontmatter: {
      title: `Test Artifact (${status})`,
      type: 'idea',
      status,
      lineage: slug,
    },
    mtime: '2026-01-01T00:00:00Z',
    created: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

/**
 * Create exactly one ArtifactRow per status in ALL_STATUSES.
 * Returns 8 artifacts: 5 active + 3 terminal.
 */
export function makeArtifactsForAllStatuses(): ArtifactRow[] {
  return ALL_STATUSES.map((status, i) =>
    makeArtifactRow({
      path: `lifecycle/ideas/status-${status}-${i + 1}.md`,
      slug: `status-${status}`,
      lineage: `status-${status}`,
      index: i + 1,
      status,
      title: `Artifact ${status}`,
    }),
  )
}

/** Build a minimal GraphNode with sensible defaults. */
export function makeGraphNode(overrides: Partial<GraphNode> = {}): GraphNode {
  const status = overrides.status ?? 'draft'
  const slug = overrides.slug ?? `node-${status}`
  return {
    id: overrides.id ?? `lifecycle/ideas/${slug}.md`,
    title: `Graph Node (${status})`,
    type: 'idea',
    status,
    stage: 'ideas',
    lineage: slug,
    slug,
    index: 1,
    labels: [],
    ...overrides,
  }
}

/** Build a GraphEdge linking source → target. */
export function makeGraphEdge(source: string, target: string, kind = 'parent'): GraphEdge {
  return { source, target, kind }
}

/**
 * Create one GraphNode per status in ALL_STATUSES.
 * Returns 8 nodes so tests can assert exactly which subset is visible.
 */
export function makeGraphNodesForAllStatuses(): GraphNode[] {
  return ALL_STATUSES.map((status, i) =>
    makeGraphNode({
      id: `lifecycle/ideas/status-${status}-${i + 1}.md`,
      slug: `status-${status}`,
      lineage: `status-${status}`,
      index: i + 1,
      status,
      title: `Node ${status}`,
    }),
  )
}
