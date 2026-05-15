// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 1 — Unit tests for the edge label map
 *
 * Covers:
 *   - edgeLabel() returns correct directional labels for all 6 relationship kinds
 *   - EDGE_LABEL_MAP contains all expected kinds with both directions
 *   - Unknown kind falls back to uppercased kind string
 *
 * Component: web/src/components/map/graphConstants.ts
 * Test plan: lifecycle/test-plans/artefact-relationship-labels-and-links-5-test.md §Milestone 1
 */

import { describe, it, expect } from 'vitest'
import { edgeLabel, EDGE_LABEL_MAP } from '../../web/src/components/map/graphConstants'

// ===========================================================================
// edgeLabel() — all 6 known kinds × 2 directions
// ===========================================================================

describe('edgeLabel() — known relationship kinds', () => {
  // parent
  it('returns CHILD OF for parent outbound', () => {
    expect(edgeLabel('parent', 'outbound')).toBe('CHILD OF')
  })

  it('returns PARENT OF for parent inbound', () => {
    expect(edgeLabel('parent', 'inbound')).toBe('PARENT OF')
  })

  // depends_on
  it('returns DEPENDS ON for depends_on outbound', () => {
    expect(edgeLabel('depends_on', 'outbound')).toBe('DEPENDS ON')
  })

  it('returns DEPENDED ON BY for depends_on inbound', () => {
    expect(edgeLabel('depends_on', 'inbound')).toBe('DEPENDED ON BY')
  })

  // blocks
  it('returns BLOCKS for blocks outbound', () => {
    expect(edgeLabel('blocks', 'outbound')).toBe('BLOCKS')
  })

  it('returns BLOCKED BY for blocks inbound', () => {
    expect(edgeLabel('blocks', 'inbound')).toBe('BLOCKED BY')
  })

  // related_to
  it('returns RELATED TO for related_to outbound', () => {
    expect(edgeLabel('related_to', 'outbound')).toBe('RELATED TO')
  })

  it('returns RELATED TO for related_to inbound', () => {
    expect(edgeLabel('related_to', 'inbound')).toBe('RELATED TO')
  })

  // members
  it('returns MEMBER OF for members outbound', () => {
    expect(edgeLabel('members', 'outbound')).toBe('MEMBER OF')
  })

  it('returns HAS MEMBER for members inbound', () => {
    expect(edgeLabel('members', 'inbound')).toBe('HAS MEMBER')
  })

  // wiki
  it('returns LINKS TO for wiki outbound', () => {
    expect(edgeLabel('wiki', 'outbound')).toBe('LINKS TO')
  })

  it('returns LINKED FROM for wiki inbound', () => {
    expect(edgeLabel('wiki', 'inbound')).toBe('LINKED FROM')
  })
})

// ===========================================================================
// edgeLabel() — unknown kind fallback
// ===========================================================================

describe('edgeLabel() — unknown kind fallback', () => {
  it('falls back to uppercased kind for unknown kinds', () => {
    expect(edgeLabel('custom_rel', 'outbound')).toBe('CUSTOM_REL')
    expect(edgeLabel('custom_rel', 'inbound')).toBe('CUSTOM_REL')
  })

  it('falls back to uppercase for an empty string kind', () => {
    expect(edgeLabel('', 'outbound')).toBe('')
  })

  it('falls back to uppercase for a mixed-case unknown kind', () => {
    expect(edgeLabel('SomeKind', 'outbound')).toBe('SOMEKIND')
  })
})

// ===========================================================================
// EDGE_LABEL_MAP — structural completeness
// ===========================================================================

describe('EDGE_LABEL_MAP — structural completeness', () => {
  const EXPECTED_KINDS = ['parent', 'depends_on', 'blocks', 'related_to', 'members', 'wiki']

  it('contains all 6 expected relationship kinds', () => {
    for (const kind of EXPECTED_KINDS) {
      expect(EDGE_LABEL_MAP[kind], `EDGE_LABEL_MAP must contain "${kind}"`).toBeDefined()
    }
  })

  it('every entry has a non-empty outbound label', () => {
    for (const kind of EXPECTED_KINDS) {
      const entry = EDGE_LABEL_MAP[kind]
      expect(
        entry.outbound,
        `EDGE_LABEL_MAP["${kind}"].outbound must be a non-empty string`,
      ).toBeTruthy()
    }
  })

  it('every entry has a non-empty inbound label', () => {
    for (const kind of EXPECTED_KINDS) {
      const entry = EDGE_LABEL_MAP[kind]
      expect(
        entry.inbound,
        `EDGE_LABEL_MAP["${kind}"].inbound must be a non-empty string`,
      ).toBeTruthy()
    }
  })

  it('edgeLabel() result matches EDGE_LABEL_MAP values for known kinds', () => {
    for (const kind of EXPECTED_KINDS) {
      expect(edgeLabel(kind, 'outbound')).toBe(EDGE_LABEL_MAP[kind].outbound)
      expect(edgeLabel(kind, 'inbound')).toBe(EDGE_LABEL_MAP[kind].inbound)
    }
  })
})
