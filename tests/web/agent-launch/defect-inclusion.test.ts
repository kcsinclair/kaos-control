// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 3 — Defect Inclusion Tests
 *
 * Validates that developer agents (backend-developer, frontend-developer,
 * test-developer) see approved defects assigned to their role alongside their
 * primary plan artifacts, and that non-developer agents (analyst-*, qa) do not
 * see defects under any circumstances.
 */

import { describe, it, expect } from 'vitest'
import {
  makeAgentSummary,
  makeArtifactsByStatusAndType,
  makeDefect,
  applyAgentLaunchFilter,
} from '../helpers/agent_launch_fixtures'

// ---------------------------------------------------------------------------
// backend-developer sees approved defects assigned to backend-developer
// ---------------------------------------------------------------------------
describe('backend-developer defect inclusion', () => {
  it('sees an approved defect assigned to backend-developer alongside plan-backend', () => {
    const agent = makeAgentSummary('backend-developer', 'in-development', ['backend-developer'])
    const plans = makeArtifactsByStatusAndType('plan-backend', ['approved'])
    const defects = [makeDefect(['backend-developer'])]

    const result = applyAgentLaunchFilter(agent.name, agent.roles, plans, defects)

    expect(result).toHaveLength(2)
    expect(result.some((a) => a.type === 'plan-backend')).toBe(true)
    expect(result.some((a) => a.type === 'defect')).toBe(true)
  })

  it('sees approved defect even when no approved plan-backend exists', () => {
    const agent = makeAgentSummary('backend-developer', 'in-development', ['backend-developer'])
    const plans = makeArtifactsByStatusAndType('plan-backend', ['draft'])
    const defects = [makeDefect(['backend-developer'])]

    const result = applyAgentLaunchFilter(agent.name, agent.roles, plans, defects)

    expect(result).toHaveLength(1)
    expect(result[0].type).toBe('defect')
  })
})

// ---------------------------------------------------------------------------
// frontend-developer sees approved defects assigned to frontend-developer
// ---------------------------------------------------------------------------
describe('frontend-developer defect inclusion', () => {
  it('sees an approved defect assigned to frontend-developer alongside plan-frontend', () => {
    const agent = makeAgentSummary('frontend-developer', 'in-development', ['frontend-developer'])
    const plans = makeArtifactsByStatusAndType('plan-frontend', ['approved'])
    const defects = [makeDefect(['frontend-developer'])]

    const result = applyAgentLaunchFilter(agent.name, agent.roles, plans, defects)

    expect(result).toHaveLength(2)
    expect(result.some((a) => a.type === 'plan-frontend')).toBe(true)
    expect(result.some((a) => a.type === 'defect')).toBe(true)
  })
})

// ---------------------------------------------------------------------------
// test-developer sees approved defects assigned to test-developer
// ---------------------------------------------------------------------------
describe('test-developer defect inclusion', () => {
  it('sees an approved defect assigned to test-developer alongside plan-test', () => {
    const agent = makeAgentSummary('test-developer', 'in-development', ['test-developer'])
    const plans = makeArtifactsByStatusAndType('plan-test', ['approved'])
    const defects = [makeDefect(['test-developer'])]

    const result = applyAgentLaunchFilter(agent.name, agent.roles, plans, defects)

    expect(result).toHaveLength(2)
    expect(result.some((a) => a.type === 'plan-test')).toBe(true)
    expect(result.some((a) => a.type === 'defect')).toBe(true)
  })
})

// ---------------------------------------------------------------------------
// Developer agent does not see unapproved defects
// ---------------------------------------------------------------------------
describe('unapproved defects are excluded', () => {
  it('backend-developer excludes a draft defect assigned to its role', () => {
    const agent = makeAgentSummary('backend-developer', 'in-development', ['backend-developer'])
    const plans = makeArtifactsByStatusAndType('plan-backend', ['approved'])
    const defects = [makeDefect(['backend-developer'], 'draft')]

    const result = applyAgentLaunchFilter(agent.name, agent.roles, plans, defects)

    expect(result.every((a) => a.type !== 'defect')).toBe(true)
  })

  it('frontend-developer excludes a rejected defect assigned to its role', () => {
    const agent = makeAgentSummary('frontend-developer', 'in-development', ['frontend-developer'])
    const plans = makeArtifactsByStatusAndType('plan-frontend', ['approved'])
    const defects = [makeDefect(['frontend-developer'], 'rejected')]

    const result = applyAgentLaunchFilter(agent.name, agent.roles, plans, defects)

    expect(result.every((a) => a.type !== 'defect')).toBe(true)
  })

  it('test-developer excludes an in-development defect assigned to its role', () => {
    const agent = makeAgentSummary('test-developer', 'in-development', ['test-developer'])
    const plans = makeArtifactsByStatusAndType('plan-test', ['approved'])
    const defects = [makeDefect(['test-developer'], 'in-development')]

    const result = applyAgentLaunchFilter(agent.name, agent.roles, plans, defects)

    expect(result.every((a) => a.type !== 'defect')).toBe(true)
  })
})

// ---------------------------------------------------------------------------
// Developer agent does not see defects assigned to a different role
// ---------------------------------------------------------------------------
describe('defects assigned to other roles are excluded', () => {
  it('backend-developer excludes an approved defect assigned to frontend-developer', () => {
    const agent = makeAgentSummary('backend-developer', 'in-development', ['backend-developer'])
    const plans = makeArtifactsByStatusAndType('plan-backend', ['approved'])
    const defects = [makeDefect(['frontend-developer'])]

    const result = applyAgentLaunchFilter(agent.name, agent.roles, plans, defects)

    expect(result.every((a) => a.type !== 'defect')).toBe(true)
  })

  it('frontend-developer excludes an approved defect assigned to backend-developer', () => {
    const agent = makeAgentSummary('frontend-developer', 'in-development', ['frontend-developer'])
    const plans = makeArtifactsByStatusAndType('plan-frontend', ['approved'])
    const defects = [makeDefect(['backend-developer'])]

    const result = applyAgentLaunchFilter(agent.name, agent.roles, plans, defects)

    expect(result.every((a) => a.type !== 'defect')).toBe(true)
  })

  it('test-developer excludes an approved defect assigned to backend-developer', () => {
    const agent = makeAgentSummary('test-developer', 'in-development', ['test-developer'])
    const plans = makeArtifactsByStatusAndType('plan-test', ['approved'])
    const defects = [makeDefect(['backend-developer'])]

    const result = applyAgentLaunchFilter(agent.name, agent.roles, plans, defects)

    expect(result.every((a) => a.type !== 'defect')).toBe(true)
  })

  it('developer agent sees defect when it is assigned to multiple roles including its own', () => {
    const agent = makeAgentSummary('backend-developer', 'in-development', ['backend-developer'])
    const plans = makeArtifactsByStatusAndType('plan-backend', [])
    const defects = [makeDefect(['frontend-developer', 'backend-developer'])]

    const result = applyAgentLaunchFilter(agent.name, agent.roles, plans, defects)

    expect(result).toHaveLength(1)
    expect(result[0].type).toBe('defect')
  })
})

// ---------------------------------------------------------------------------
// Non-developer agents (requirements-analyst) do not see defects
// ---------------------------------------------------------------------------
describe('requirements-analyst does not see defects', () => {
  it('approved defect assigned to analyst role is excluded', () => {
    const agent = makeAgentSummary('requirements-analyst', 'approved', ['analyst'])
    const ideas = makeArtifactsByStatusAndType('idea', ['approved'])
    const defects = [makeDefect(['analyst'])]

    const result = applyAgentLaunchFilter(agent.name, agent.roles, ideas, defects)

    expect(result.every((a) => a.type !== 'defect')).toBe(true)
    expect(result.every((a) => a.type === 'idea')).toBe(true)
  })

  it('has no defects even when defect list contains approved entries for every role', () => {
    const agent = makeAgentSummary('requirements-analyst', 'approved', ['analyst'])
    const ideas = makeArtifactsByStatusAndType('idea', ['approved'])
    const defects = [
      makeDefect(['analyst']),
      makeDefect(['backend-developer']),
      makeDefect(['frontend-developer']),
    ]

    const result = applyAgentLaunchFilter(agent.name, agent.roles, ideas, defects)

    expect(result.filter((a) => a.type === 'defect')).toHaveLength(0)
  })
})

// ---------------------------------------------------------------------------
// qa does not see defects
// ---------------------------------------------------------------------------
describe('qa does not see defects', () => {
  it('approved defect assigned to qa role is excluded', () => {
    const agent = makeAgentSummary('qa', 'in-qa', ['qa'])
    const tests = makeArtifactsByStatusAndType('test', ['approved'])
    const defects = [makeDefect(['qa'])]

    const result = applyAgentLaunchFilter(agent.name, agent.roles, tests, defects)

    expect(result.every((a) => a.type !== 'defect')).toBe(true)
    expect(result.every((a) => a.type === 'test')).toBe(true)
  })
})
