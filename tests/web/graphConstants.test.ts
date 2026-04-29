/**
 * Milestone 1 — Unit tests for graphConstants.ts
 *
 * Covers:
 *   - APPROVED_TEST_RING_COLOR exists and is a valid hex colour
 *   - APPROVED_TEST_RING_COLOR is visually distinct from adjacent colours
 *     (test node fill, low-priority ring, clarifying active-status colour)
 *
 * These tests guard against accidental removal or modification of the constant.
 *
 * Component: web/src/components/graph/graphConstants.ts
 */

import { describe, it, expect } from 'vitest'
import {
  APPROVED_TEST_RING_COLOR,
  NODE_COLORS,
  PRIORITY_COLORS,
  ACTIVE_STATUS_COLORS,
} from '../../web/src/components/graph/graphConstants'

describe('graphConstants — APPROVED_TEST_RING_COLOR', () => {
  it('exports APPROVED_TEST_RING_COLOR as a string', () => {
    expect(typeof APPROVED_TEST_RING_COLOR).toBe('string')
  })

  it('is a valid 6-digit hex colour string', () => {
    expect(APPROVED_TEST_RING_COLOR).toMatch(/^#[0-9a-fA-F]{6}$/)
  })

  it('differs from the test node fill colour (cyan)', () => {
    expect(APPROVED_TEST_RING_COLOR).not.toBe(NODE_COLORS.test)
  })

  it('differs from the low-priority ring colour (blue)', () => {
    expect(APPROVED_TEST_RING_COLOR).not.toBe(PRIORITY_COLORS.low)
  })

  it('differs from the clarifying active-status colour (light blue)', () => {
    expect(APPROVED_TEST_RING_COLOR).not.toBe(ACTIVE_STATUS_COLORS.clarifying)
  })
})
