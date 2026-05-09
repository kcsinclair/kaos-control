// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Fixtures for AgentLaunchModal tests.
 *
 * Provides factory functions for AgentSummary and ArtifactRow objects used by
 * the agent-launch test suites. Builds on makeArtifactRow from seed_artifacts.
 */

import type { AgentSummary, ArtifactRow } from '@/types/api'
import { makeArtifactRow } from './seed_artifacts'

/**
 * Build a minimal AgentSummary with configurable name, active_status, and roles.
 */
export function makeAgentSummary(
  name: string,
  activeStatus: string,
  roles: string[],
): AgentSummary {
  return {
    name,
    roles,
    driver: 'claude-code',
    active_status: activeStatus,
  }
}

/**
 * Produce one ArtifactRow per requested status, all sharing the given type.
 * Each artifact gets a unique slug derived from type + status.
 */
export function makeArtifactsByStatusAndType(type: string, statuses: string[]): ArtifactRow[] {
  return statuses.map((status, i) => {
    const slug = `${type}-${status}-${i + 1}`
    return makeArtifactRow({
      path: `lifecycle/${type}s/${slug}.md`,
      slug,
      lineage: slug,
      index: i + 1,
      stage: `${type}s`,
      type,
      status,
      title: `${type} (${status})`,
      frontmatter: {
        title: `${type} (${status})`,
        type,
        status,
        lineage: slug,
      },
    })
  })
}

/**
 * Build an ArtifactRow representing a defect assigned to one or more roles.
 * Status defaults to 'approved' since the modal only shows approved defects.
 */
export function makeDefect(
  assigneeRoles: string[],
  status = 'approved',
  overrides: Partial<ArtifactRow> = {},
): ArtifactRow {
  const slug = `defect-${assigneeRoles.join('-')}-${status}`
  return makeArtifactRow({
    path: `lifecycle/defects/${slug}.md`,
    slug,
    lineage: slug,
    index: 1,
    stage: 'defects',
    type: 'defect',
    status,
    title: `Defect (${status}, roles: ${assigneeRoles.join(', ')})`,
    frontmatter: {
      title: `Defect for ${assigneeRoles.join(', ')}`,
      type: 'defect',
      status,
      lineage: slug,
      assignees: assigneeRoles.map((role) => ({ role, who: 'agent' })),
    },
    ...overrides,
  })
}

/**
 * Maps agent names to the primary artifact type they expect as input.
 * Mirrors the agentInputTypeMap constant in AgentLaunchModal.vue.
 */
export const AGENT_INPUT_TYPE_MAP: Record<string, string> = {
  'requirements-analyst': 'idea',
  'planning-analyst': 'requirement',
  'backend-developer': 'plan-backend',
  'frontend-developer': 'plan-frontend',
  'test-developer': 'plan-test',
  qa: 'test',
}

/**
 * Reproduce the primary artifact filtering logic from AgentLaunchModal.fetchArtifacts.
 *
 * Given a flat list of candidate artifacts and a flat list of defect artifacts,
 * returns the combined filtered set: approved primary-type artifacts plus (for
 * developer agents) approved defects assigned to the agent's roles.
 */
export function applyAgentLaunchFilter(
  agentName: string,
  agentRoles: string[],
  candidateArtifacts: ArtifactRow[],
  allDefects: ArtifactRow[] = [],
): ArtifactRow[] {
  const inputType = AGENT_INPUT_TYPE_MAP[agentName]
  const results: ArtifactRow[] = []

  // Primary filter: status=approved + correct type (mirrors the API query the modal makes).
  results.push(
    ...candidateArtifacts.filter((a) => a.status === 'approved' && a.type === inputType),
  )

  // Defect inclusion: only for developer agents (plan-* input type).
  if (inputType?.startsWith('plan-')) {
    const roleSet = new Set(agentRoles)
    results.push(
      ...allDefects.filter(
        (a) =>
          a.status === 'approved' &&
          a.frontmatter.assignees?.some((assignee) => roleSet.has(assignee.role)),
      ),
    )
  }

  return results
}
