// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect } from 'vitest'
import { isArchitectureZone } from '@/types/api'

// Frontend plan: lifecycle/frontend-plans/architecture-overview-view-4-fe.md
// — Milestone F5 (FR-9/FR-9a). Broadened from the old isCatalogMaterial,
// which excluded only catalog candidates + the archive — this predicate
// excludes the whole lifecycle/architecture/ zone by default, now that the
// overview view owns it.

// Minimal row shape — isArchitectureZone only reads path + frontmatter.
function row(path: string, labels?: string[]) {
  return { path, frontmatter: labels ? { labels } : {} } as Parameters<typeof isArchitectureZone>[0]
}

describe('isArchitectureZone', () => {
  it('flags candidate catalog files (carry the `catalog` label)', () => {
    expect(
      isArchitectureZone(row('lifecycle/architecture/architectures/cloud-native-microservices.md', ['architecture', 'catalog'])),
    ).toBe(true)
    expect(
      isArchitectureZone(row('lifecycle/architecture/tech-stacks/go-vue-sqlite.md', ['catalog'])),
    ).toBe(true)
  })

  it('flags superseded promoted choices under architecture/archive/', () => {
    expect(isArchitectureZone(row('lifecycle/architecture/archive/edge-hybrid.md'))).toBe(true)
  })

  it('flags the chosen architecture promoted to the root', () => {
    expect(isArchitectureZone(row('lifecycle/architecture/cloud-native-microservices.md', ['architecture']))).toBe(true)
  })

  it('flags ADRs and standards', () => {
    expect(isArchitectureZone(row('lifecycle/architecture/decisions/adr-0001-datastore.md', ['adr']))).toBe(true)
    expect(isArchitectureZone(row('lifecycle/architecture/standards/secrets-handling.md', ['standard']))).toBe(true)
  })

  it('flags the architecture summary', () => {
    expect(isArchitectureZone(row('lifecycle/architecture/architecture-summary.md'))).toBe(true)
  })

  it('does NOT flag ordinary lifecycle artifacts', () => {
    expect(isArchitectureZone(row('lifecycle/ideas/login.md', ['feature']))).toBe(false)
    expect(isArchitectureZone(row('lifecycle/requirements/login-2.md'))).toBe(false)
  })

  it('does not flag a path that merely contains "architecture" as a substring', () => {
    expect(isArchitectureZone(row('lifecycle/requirements/architecture-notes-2.md'))).toBe(false)
  })
})
