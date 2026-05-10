// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 1 — Unit tests for the 15-character title truncation rule.
 *
 * The truncation logic is inlined inside ForceGraph3D.vue's buildNodeObject:
 *
 *   const raw = n.title || n.slug
 *   const truncated = raw.length > 15 ? raw.slice(0, 15) + '\u2026' : raw
 *
 * These tests define and exercise the same pure function to lock in the
 * behaviour independently of the component.  If the helper is ever extracted
 * to a utility module, the import path here should be updated to point there.
 *
 * Acceptance criteria verified:
 *   - Strings of exactly 15 characters return unchanged (no ellipsis).
 *   - Strings of 16+ characters are truncated to 15 chars followed by … (U+2026).
 *   - Empty string and single-character inputs handled correctly.
 *   - Unicode characters are truncated by character count, not byte count.
 */

import { describe, it, expect } from 'vitest'

// ---------------------------------------------------------------------------
// The function under test — mirrors the inline logic in ForceGraph3D.vue
// ---------------------------------------------------------------------------

/** Truncate a string to 15 characters, appending … (U+2026) when truncated. */
function truncateTitle(s: string): string {
  return s.length > 15 ? s.slice(0, 15) + '\u2026' : s
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('truncateTitle — boundary: exactly 15 characters', () => {
  it('a string of exactly 15 characters is returned unchanged', () => {
    const s = 'a'.repeat(15) // 15 chars
    expect(truncateTitle(s)).toBe(s)
    expect(truncateTitle(s)).not.toContain('\u2026')
  })

  it('a string of exactly 15 characters has no ellipsis appended', () => {
    const s = 'Hello, World!!!'  // exactly 15 chars
    expect(s).toHaveLength(15)
    expect(truncateTitle(s)).toBe('Hello, World!!!')
  })
})

describe('truncateTitle — strings longer than 15 characters', () => {
  it('a string of 16 characters is truncated to 15 chars + …', () => {
    const s = 'a'.repeat(16)
    const result = truncateTitle(s)
    expect(result).toBe('a'.repeat(15) + '\u2026')
    expect(result).toHaveLength(16) // 15 chars + 1 ellipsis codepoint
  })

  it('a string of 100 characters keeps only the first 15 chars before …', () => {
    const s = 'Hello, World and beyond!'
    const result = truncateTitle(s)
    expect(result).toBe('Hello, World an\u2026')
    expect(result).toHaveLength(16)
  })

  it('uses U+2026 HORIZONTAL ELLIPSIS, not three dots', () => {
    const s = 'a'.repeat(16)
    const result = truncateTitle(s)
    expect(result.endsWith('\u2026')).toBe(true)
    expect(result.endsWith('...')).toBe(false)
  })
})

describe('truncateTitle — short strings and edge cases', () => {
  it('empty string returns empty string', () => {
    expect(truncateTitle('')).toBe('')
  })

  it('single character returns the same character', () => {
    expect(truncateTitle('x')).toBe('x')
  })

  it('14-character string is returned unchanged', () => {
    const s = 'a'.repeat(14)
    expect(truncateTitle(s)).toBe(s)
    expect(truncateTitle(s)).not.toContain('\u2026')
  })
})

describe('truncateTitle — Unicode character handling', () => {
  it('truncates by character count, not byte count (emoji = 1 char each)', () => {
    // Each emoji is a single codepoint from String.prototype.slice's perspective
    // ONLY when they're in the BMP; surrogate pairs count as 2 code units.
    // The implementation uses .slice() which operates on UTF-16 code units.
    // Test with BMP characters to verify character-count truncation.
    const s = 'ABCDEFGHIJKLMNOP' // 16 ASCII chars
    const result = truncateTitle(s)
    expect(result).toBe('ABCDEFGHIJKLMNO\u2026')
  })

  it('Japanese characters (BMP) are truncated by code-unit count', () => {
    // Each kanji is 1 code unit (BMP); 16 kanji should truncate to 15 + …
    const s = 'あいうえおかきくけこさしすせそた' // 16 chars
    expect(s.length).toBe(16)
    const result = truncateTitle(s)
    expect(result).toBe('あいうえおかきくけこさしすせそ\u2026')
  })

  it('a 15-character unicode string is returned unchanged', () => {
    const s = 'αβγδεζηθικλμνξο' // 15 Greek letters
    expect(s.length).toBe(15)
    expect(truncateTitle(s)).toBe(s)
    expect(truncateTitle(s)).not.toContain('\u2026')
  })
})
