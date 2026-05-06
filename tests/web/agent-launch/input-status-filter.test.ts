/**
 * Milestone 2 — Input Status Filtering Tests
 *
 * Validates that the artifact filtering logic in AgentLaunchModal always uses
 * status: 'approved', regardless of agent type or active_status value. The
 * predecessorMap was removed; these tests confirm 'approved' is always the
 * filter status for all agent types.
 *
 * Uses Vue ref/computed to reproduce the filtering inline — same pattern as
 * tests/web/hide-done-items/artifact-list-toggle.test.ts.
 */

import { describe, it, expect } from 'vitest'
import { ref, computed } from 'vue'
import type { ArtifactRow } from '@/types/api'
import {
  makeAgentSummary,
  makeArtifactsByStatusAndType,
  applyAgentLaunchFilter,
  AGENT_INPUT_TYPE_MAP,
} from '../helpers/agent_launch_fixtures'

// ---------------------------------------------------------------------------
// Helper — reproduce fetchArtifacts filtering as a reactive computed.
// ---------------------------------------------------------------------------
function makeFilteredItems(
  agentName: string,
  agentRoles: string[],
  candidates: ArtifactRow[],
  defects: ArtifactRow[] = [],
) {
  const allCandidates = ref(candidates)
  const allDefects = ref(defects)

  const filteredItems = computed(() =>
    applyAgentLaunchFilter(agentName, agentRoles, allCandidates.value, allDefects.value),
  )

  return { allCandidates, allDefects, filteredItems }
}

// Mixed statuses used across tests.
const MIXED_STATUSES = ['draft', 'clarifying', 'approved', 'rejected', 'in-development', 'done']

// ---------------------------------------------------------------------------
// requirements-analyst: sees only approved ideas
// ---------------------------------------------------------------------------
describe('requirements-analyst input filtering', () => {
  it('sees only the approved idea from a mixed-status set', () => {
    const agent = makeAgentSummary('requirements-analyst', 'approved', ['analyst'])
    const ideas = makeArtifactsByStatusAndType('idea', MIXED_STATUSES)
    const { filteredItems } = makeFilteredItems(agent.name, agent.roles, ideas)

    expect(filteredItems.value).toHaveLength(1)
    expect(filteredItems.value[0].status).toBe('approved')
    expect(filteredItems.value[0].type).toBe('idea')
  })

  it('excludes draft, clarifying, rejected, in-development, done ideas', () => {
    const agent = makeAgentSummary('requirements-analyst', 'approved', ['analyst'])
    const ideas = makeArtifactsByStatusAndType('idea', MIXED_STATUSES)
    const { filteredItems } = makeFilteredItems(agent.name, agent.roles, ideas)

    const statuses = filteredItems.value.map((a) => a.status)
    for (const s of ['draft', 'clarifying', 'rejected', 'in-development', 'done']) {
      expect(statuses).not.toContain(s)
    }
  })
})

// ---------------------------------------------------------------------------
// planning-analyst: sees only approved requirements
// ---------------------------------------------------------------------------
describe('planning-analyst input filtering', () => {
  it('sees only the approved requirement from a mixed-status set', () => {
    const agent = makeAgentSummary('planning-analyst', 'approved', ['analyst'])
    const reqs = makeArtifactsByStatusAndType('requirement', MIXED_STATUSES)
    const { filteredItems } = makeFilteredItems(agent.name, agent.roles, reqs)

    expect(filteredItems.value).toHaveLength(1)
    expect(filteredItems.value[0].status).toBe('approved')
    expect(filteredItems.value[0].type).toBe('requirement')
  })

  it('excludes non-approved requirements', () => {
    const agent = makeAgentSummary('planning-analyst', 'approved', ['analyst'])
    const reqs = makeArtifactsByStatusAndType('requirement', ['draft', 'rejected'])
    const { filteredItems } = makeFilteredItems(agent.name, agent.roles, reqs)

    expect(filteredItems.value).toHaveLength(0)
  })
})

// ---------------------------------------------------------------------------
// backend-developer: sees only approved plan-backend
// ---------------------------------------------------------------------------
describe('backend-developer input filtering', () => {
  it('sees only approved plan-backend artifacts', () => {
    const agent = makeAgentSummary('backend-developer', 'in-development', ['backend-developer'])
    const plans = makeArtifactsByStatusAndType('plan-backend', MIXED_STATUSES)
    const { filteredItems } = makeFilteredItems(agent.name, agent.roles, plans)

    expect(filteredItems.value).toHaveLength(1)
    expect(filteredItems.value[0].status).toBe('approved')
    expect(filteredItems.value[0].type).toBe('plan-backend')
  })
})

// ---------------------------------------------------------------------------
// frontend-developer: sees only approved plan-frontend
// ---------------------------------------------------------------------------
describe('frontend-developer input filtering', () => {
  it('sees only approved plan-frontend artifacts', () => {
    const agent = makeAgentSummary('frontend-developer', 'in-development', ['frontend-developer'])
    const plans = makeArtifactsByStatusAndType('plan-frontend', MIXED_STATUSES)
    const { filteredItems } = makeFilteredItems(agent.name, agent.roles, plans)

    expect(filteredItems.value).toHaveLength(1)
    expect(filteredItems.value[0].status).toBe('approved')
    expect(filteredItems.value[0].type).toBe('plan-frontend')
  })
})

// ---------------------------------------------------------------------------
// test-developer: sees only approved plan-test
// ---------------------------------------------------------------------------
describe('test-developer input filtering', () => {
  it('sees only approved plan-test artifacts', () => {
    const agent = makeAgentSummary('test-developer', 'in-development', ['test-developer'])
    const plans = makeArtifactsByStatusAndType('plan-test', MIXED_STATUSES)
    const { filteredItems } = makeFilteredItems(agent.name, agent.roles, plans)

    expect(filteredItems.value).toHaveLength(1)
    expect(filteredItems.value[0].status).toBe('approved')
    expect(filteredItems.value[0].type).toBe('plan-test')
  })
})

// ---------------------------------------------------------------------------
// qa: sees only approved tests
// ---------------------------------------------------------------------------
describe('qa input filtering', () => {
  it('sees only approved test artifacts', () => {
    const agent = makeAgentSummary('qa', 'in-qa', ['qa'])
    const tests = makeArtifactsByStatusAndType('test', MIXED_STATUSES)
    const { filteredItems } = makeFilteredItems(agent.name, agent.roles, tests)

    expect(filteredItems.value).toHaveLength(1)
    expect(filteredItems.value[0].status).toBe('approved')
    expect(filteredItems.value[0].type).toBe('test')
  })
})

// ---------------------------------------------------------------------------
// predecessorMap no longer influences filtering
// ---------------------------------------------------------------------------
describe('predecessorMap is not used — filter status is always approved', () => {
  it('filter status is approved regardless of agent active_status = draft', () => {
    // An agent whose active_status is 'draft' (as if predecessorMap implied 'draft')
    // should still only see 'approved' artifacts.
    const agent = makeAgentSummary('requirements-analyst', 'draft', ['analyst'])
    const ideas = makeArtifactsByStatusAndType('idea', ['draft', 'approved'])
    const { filteredItems } = makeFilteredItems(agent.name, agent.roles, ideas)

    expect(filteredItems.value).toHaveLength(1)
    expect(filteredItems.value[0].status).toBe('approved')
  })

  it('filter status is approved regardless of agent active_status = in-development', () => {
    const agent = makeAgentSummary('backend-developer', 'in-development', ['backend-developer'])
    const plans = makeArtifactsByStatusAndType('plan-backend', ['in-development', 'approved'])
    const { filteredItems } = makeFilteredItems(agent.name, agent.roles, plans)

    expect(filteredItems.value).toHaveLength(1)
    expect(filteredItems.value[0].status).toBe('approved')
  })

  it('all agents in AGENT_INPUT_TYPE_MAP yield empty list when no approved artifacts exist', () => {
    const nonApprovedStatuses = ['draft', 'clarifying', 'rejected', 'in-development', 'done']

    for (const [agentName, inputType] of Object.entries(AGENT_INPUT_TYPE_MAP)) {
      const roles = [agentName]
      const artifacts = makeArtifactsByStatusAndType(inputType, nonApprovedStatuses)
      const { filteredItems } = makeFilteredItems(agentName, roles, artifacts)

      expect(filteredItems.value, `${agentName} should have empty list with no approved`).toHaveLength(0)
    }
  })

  it('all agents in AGENT_INPUT_TYPE_MAP yield exactly the approved artifact', () => {
    for (const [agentName, inputType] of Object.entries(AGENT_INPUT_TYPE_MAP)) {
      const roles = [agentName]
      const artifacts = makeArtifactsByStatusAndType(inputType, ['draft', 'approved', 'rejected'])
      const { filteredItems } = makeFilteredItems(agentName, roles, artifacts)

      expect(filteredItems.value, `${agentName} should see exactly 1 approved artifact`).toHaveLength(1)
      expect(filteredItems.value[0].status).toBe('approved')
    }
  })
})
