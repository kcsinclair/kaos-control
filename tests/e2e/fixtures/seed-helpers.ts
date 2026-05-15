/** Programmatic seeding utilities for kaos-control E2E tests. */

export interface SeedArtifact {
  path: string
  title: string
  type: string
  status: string
  lineage: string
}

/**
 * Return the expected tracked-type counts from the standard smoke fixture.
 * fixture has: 10 ideas + 3 requirements + 1 defect = 14 items.
 */
export function expectedTotalCount(): number {
  return 14
}

/** The artifact used in the edit-save flow test (flow 02). */
export const EDIT_TARGET: SeedArtifact = {
  path: 'lifecycle/requirements/smoke-req-01.md',
  title: 'Smoke Requirement Alpha',
  type: 'requirement',
  status: 'draft',
  lineage: 'smoke-req-01',
}

/** The artifact used in the transition flow test (flow 03). */
export const TRANSITION_TARGET: SeedArtifact = {
  path: 'lifecycle/requirements/smoke-req-02.md',
  title: 'Smoke Requirement Beta',
  type: 'requirement',
  status: 'planning',
  lineage: 'smoke-req-02',
}
