// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect } from 'vitest'
import { isCatalogMaterial } from '@/types/api'

// Minimal row shape — isCatalogMaterial only reads path + frontmatter.
function row(path: string, labels?: string[]) {
  return { path, frontmatter: labels ? { labels } : {} } as Parameters<typeof isCatalogMaterial>[0]
}

describe('isCatalogMaterial', () => {
  it('flags candidate catalog files (carry the `catalog` label)', () => {
    expect(
      isCatalogMaterial(row('lifecycle/architecture/architectures/cloud-native-microservices.md', ['architecture', 'catalog'])),
    ).toBe(true)
    expect(
      isCatalogMaterial(row('lifecycle/architecture/tech-stacks/go-vue-sqlite.md', ['catalog'])),
    ).toBe(true)
  })

  it('flags superseded promoted choices under architecture/archive/', () => {
    expect(isCatalogMaterial(row('lifecycle/architecture/archive/edge-hybrid.md'))).toBe(true)
  })

  it('does NOT flag the chosen architecture promoted to the root', () => {
    // No `catalog` label, lives at the architecture root, not in a subdir/archive.
    expect(isCatalogMaterial(row('lifecycle/architecture/cloud-native-microservices.md', ['architecture']))).toBe(false)
  })

  it('does NOT flag ADRs or standards', () => {
    expect(isCatalogMaterial(row('lifecycle/architecture/decisions/adr-0001-datastore.md', ['adr']))).toBe(false)
    expect(isCatalogMaterial(row('lifecycle/architecture/standards/secrets-handling.md', ['standard']))).toBe(false)
  })

  it('does NOT flag ordinary lifecycle artifacts', () => {
    expect(isCatalogMaterial(row('lifecycle/ideas/login.md', ['feature']))).toBe(false)
    expect(isCatalogMaterial(row('lifecycle/requirements/login-2.md'))).toBe(false)
  })
})
