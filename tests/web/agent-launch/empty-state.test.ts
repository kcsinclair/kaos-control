// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 4 — Empty State Tests
 *
 * Validates the empty-list conditions for the AgentLaunchModal. The modal
 * shows "No eligible artifacts for this agent." when the combined filtered
 * result (primary type + approved status, plus defects for developer agents)
 * is empty.
 *
 * Uses Vue ref/computed to make the filtering reactive so that the
 * "clears when approved artifact is added" test can verify reactivity.
 */

import { describe, it, expect } from 'vitest'
import { ref, computed } from 'vue'
import type { ArtifactRow } from '@/types/api'
import {
  makeAgentSummary,
  makeArtifactsByStatusAndType,
  makeDefect,
  applyAgentLaunchFilter,
  AGENT_INPUT_TYPE_MAP,
} from '../helpers/agent_launch_fixtures'
import { makeArtifactRow } from '../helpers/seed_artifacts'

// ---------------------------------------------------------------------------
// Helper — reactive wrapper so we can test "adds an artifact → list changes".
// ---------------------------------------------------------------------------
function makeReactiveFilter(
  agentName: string,
  agentRoles: string[],
  initialCandidates: ArtifactRow[] = [],
  initialDefects: ArtifactRow[] = [],
) {
  const candidates = ref<ArtifactRow[]>(initialCandidates)
  const defects = ref<ArtifactRow[]>(initialDefects)

  const filteredItems = computed(() =>
    applyAgentLaunchFilter(agentName, agentRoles, candidates.value, defects.value),
  )

  return { candidates, defects, filteredItems }
}

// ---------------------------------------------------------------------------
// Empty state: no approved artifacts of the correct type
// ---------------------------------------------------------------------------
describe('empty state — no approved artifacts of correct type', () => {
  it('requirements-analyst sees empty list when all ideas are draft', () => {
    const agent = makeAgentSummary('requirements-analyst', 'approved', ['analyst'])
    const ideas = makeArtifactsByStatusAndType('idea', ['draft', 'clarifying', 'rejected'])

    const result = applyAgentLaunchFilter(agent.name, agent.roles, ideas)

    expect(result).toHaveLength(0)
  })

  it('backend-developer sees empty list when all plan-backend are draft', () => {
    const agent = makeAgentSummary('backend-developer', 'in-development', ['backend-developer'])
    const plans = makeArtifactsByStatusAndType('plan-backend', ['draft', 'in-development'])

    const result = applyAgentLaunchFilter(agent.name, agent.roles, plans)

    expect(result).toHaveLength(0)
  })

  it('qa sees empty list when all tests are draft', () => {
    const agent = makeAgentSummary('qa', 'in-qa', ['qa'])
    const tests = makeArtifactsByStatusAndType('test', ['draft', 'rejected'])

    const result = applyAgentLaunchFilter(agent.name, agent.roles, tests)

    expect(result).toHaveLength(0)
  })
})

// ---------------------------------------------------------------------------
// Empty state: artifacts exist but of the wrong type
// ---------------------------------------------------------------------------
describe('empty state — approved artifacts exist but wrong type', () => {
  it('requirements-analyst sees empty list when only approved requirements exist (not ideas)', () => {
    const agent = makeAgentSummary('requirements-analyst', 'approved', ['analyst'])
    // Approved requirements — wrong type for requirements-analyst (which wants ideas).
    const requirements = makeArtifactsByStatusAndType('requirement', ['approved'])

    const result = applyAgentLaunchFilter(agent.name, agent.roles, requirements)

    expect(result).toHaveLength(0)
  })

  it('backend-developer sees empty list when only approved plan-frontend exist', () => {
    const agent = makeAgentSummary('backend-developer', 'in-development', ['backend-developer'])
    const plans = makeArtifactsByStatusAndType('plan-frontend', ['approved'])

    const result = applyAgentLaunchFilter(agent.name, agent.roles, plans)

    expect(result).toHaveLength(0)
  })

  it('qa sees empty list when only approved ideas and requirements exist', () => {
    const agent = makeAgentSummary('qa', 'in-qa', ['qa'])
    const mixed = [
      ...makeArtifactsByStatusAndType('idea', ['approved']),
      ...makeArtifactsByStatusAndType('requirement', ['approved']),
    ]

    const result = applyAgentLaunchFilter(agent.name, agent.roles, mixed)

    expect(result).toHaveLength(0)
  })
})

// ---------------------------------------------------------------------------
// Empty state clears when an approved artifact of the correct type is added
// ---------------------------------------------------------------------------
describe('empty state clears when approved artifact is added', () => {
  it('requirements-analyst list becomes non-empty when an approved idea is added', () => {
    const agent = makeAgentSummary('requirements-analyst', 'approved', ['analyst'])
    const { candidates, filteredItems } = makeReactiveFilter(agent.name, agent.roles, [
      ...makeArtifactsByStatusAndType('idea', ['draft']),
    ])

    // Starts empty.
    expect(filteredItems.value).toHaveLength(0)

    // Add an approved idea.
    const approvedIdea = makeArtifactRow({
      slug: 'new-approved-idea',
      lineage: 'new-approved-idea',
      type: 'idea',
      status: 'approved',
      title: 'New Approved Idea',
      frontmatter: {
        title: 'New Approved Idea',
        type: 'idea',
        status: 'approved',
        lineage: 'new-approved-idea',
      },
    })
    candidates.value = [...candidates.value, approvedIdea]

    expect(filteredItems.value).toHaveLength(1)
    expect(filteredItems.value[0].status).toBe('approved')
  })

  it('backend-developer list becomes non-empty when an approved plan-backend is added', () => {
    const agent = makeAgentSummary('backend-developer', 'in-development', ['backend-developer'])
    const { candidates, filteredItems } = makeReactiveFilter(agent.name, agent.roles, [
      ...makeArtifactsByStatusAndType('plan-backend', ['draft']),
    ])

    expect(filteredItems.value).toHaveLength(0)

    const approvedPlan = makeArtifactRow({
      slug: 'new-approved-plan-backend',
      lineage: 'new-approved-plan-backend',
      type: 'plan-backend',
      status: 'approved',
      title: 'New Approved Plan',
      frontmatter: {
        title: 'New Approved Plan',
        type: 'plan-backend',
        status: 'approved',
        lineage: 'new-approved-plan-backend',
      },
    })
    candidates.value = [...candidates.value, approvedPlan]

    expect(filteredItems.value).toHaveLength(1)
    expect(filteredItems.value[0].status).toBe('approved')
  })
})

// ---------------------------------------------------------------------------
// Developer agent empty state: no approved plans and no approved defects for role
// ---------------------------------------------------------------------------
describe('developer agent empty state includes defect check', () => {
  it('backend-developer sees empty list when no approved plans and no approved defects for its role', () => {
    const agent = makeAgentSummary('backend-developer', 'in-development', ['backend-developer'])
    const plans = makeArtifactsByStatusAndType('plan-backend', ['draft', 'in-development'])
    const defects = [
      // Unapproved defect for the right role.
      makeDefect(['backend-developer'], 'draft'),
      // Approved defect for the wrong role.
      makeDefect(['frontend-developer'], 'approved'),
    ]

    const result = applyAgentLaunchFilter(agent.name, agent.roles, plans, defects)

    expect(result).toHaveLength(0)
  })

  it('frontend-developer sees empty list with no approved plans and no matching approved defects', () => {
    const agent = makeAgentSummary('frontend-developer', 'in-development', ['frontend-developer'])
    const plans = makeArtifactsByStatusAndType('plan-frontend', ['draft'])
    const defects = [makeDefect(['backend-developer'], 'approved')]

    const result = applyAgentLaunchFilter(agent.name, agent.roles, plans, defects)

    expect(result).toHaveLength(0)
  })

  it('test-developer sees empty list with no approved plans and no matching approved defects', () => {
    const agent = makeAgentSummary('test-developer', 'in-development', ['test-developer'])
    const plans = makeArtifactsByStatusAndType('plan-test', ['draft'])
    const defects = [makeDefect(['backend-developer'], 'approved')]

    const result = applyAgentLaunchFilter(agent.name, agent.roles, plans, defects)

    expect(result).toHaveLength(0)
  })

  it('developer agent empty state clears when an approved defect for its role is added', () => {
    const agent = makeAgentSummary('backend-developer', 'in-development', ['backend-developer'])
    const { defects, filteredItems } = makeReactiveFilter(
      agent.name,
      agent.roles,
      makeArtifactsByStatusAndType('plan-backend', ['draft']),
      [],
    )

    expect(filteredItems.value).toHaveLength(0)

    defects.value = [makeDefect(['backend-developer'], 'approved')]

    expect(filteredItems.value).toHaveLength(1)
    expect(filteredItems.value[0].type).toBe('defect')
  })
})

// ---------------------------------------------------------------------------
// All agent types: empty list when no candidates at all
// ---------------------------------------------------------------------------
describe('all agent types return empty list when candidates array is empty', () => {
  it.each(Object.keys(AGENT_INPUT_TYPE_MAP))(
    '%s returns empty list with no candidates',
    (agentName) => {
      const roles = [agentName]
      const result = applyAgentLaunchFilter(agentName, roles, [])
      expect(result).toHaveLength(0)
    },
  )
})
