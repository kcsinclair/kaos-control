// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * seed_projects.ts — factory functions for ProjectSummary test data.
 *
 * Usage:
 *   import { makeProjectSummary } from '@/tests/web/helpers/seed_projects'
 *
 *   const proj = makeProjectSummary({ name: 'my-proj', initialised: true })
 */

import type { ProjectSummary } from '@/types/api'

/**
 * Build a minimal ProjectSummary with sensible defaults.
 * Override any field by passing a partial object.
 */
export function makeProjectSummary(overrides: Partial<ProjectSummary> = {}): ProjectSummary {
  const name = overrides.name ?? 'test-project'
  return {
    name,
    description: `Test project: ${name}`,
    path: `/tmp/projects/${name}`,
    owner: 'team-a',
    initialised: false,
    ...overrides,
  }
}

/**
 * Build a list of ProjectSummary objects from a list of names.
 * All projects use sensible defaults; override by passing the full list with makeProjectSummary.
 */
export function makeProjectList(names: string[]): ProjectSummary[] {
  return names.map((name) => makeProjectSummary({ name }))
}

/**
 * Build a ProjectSummary that represents an initialised project.
 */
export function makeInitialisedProject(overrides: Partial<ProjectSummary> = {}): ProjectSummary {
  return makeProjectSummary({ initialised: true, ...overrides })
}

/**
 * Build a ProjectSummary that represents an uninitialised project.
 */
export function makeUninitialisedProject(overrides: Partial<ProjectSummary> = {}): ProjectSummary {
  return makeProjectSummary({ initialised: false, ...overrides })
}
