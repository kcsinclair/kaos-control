// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 5 — Unit tests for useFormatDate composable
 *
 * Tests the date-formatting helpers in isolation.
 * The functions must handle all `created` field format variants without errors.
 *
 * Source: web/src/composables/useFormatDate.ts
 * Exports: formatShortDate, formatFullDateTime, useFormatDate
 */

import { describe, it, expect } from 'vitest'
import { formatShortDate, formatFullDateTime } from '../../web/src/composables/useFormatDate'

// ---------------------------------------------------------------------------
// formatShortDate
// ---------------------------------------------------------------------------

describe('formatShortDate', () => {
  it('returns a non-empty string for a valid RFC3339 timestamp', () => {
    const result = formatShortDate('2026-04-27T10:00:00Z')
    expect(result).not.toBe('—')
    expect(result.length).toBeGreaterThan(0)
    // Must not be "Invalid Date"
    expect(result).not.toContain('Invalid')
  })

  it('returns a non-empty string for an RFC3339 timestamp with offset', () => {
    const result = formatShortDate('2025-12-01T08:30:00+10:00')
    expect(result).not.toBe('—')
    expect(result).not.toContain('Invalid')
  })

  it('returns a non-empty string for a plain-date string (YYYY-MM-DD)', () => {
    // Browsers parse plain dates as UTC midnight; the helper should still
    // produce a valid formatted date rather than "Invalid Date" or the fallback.
    const result = formatShortDate('2026-04-27')
    expect(result).not.toBe('—')
    expect(result).not.toContain('Invalid')
  })

  it('returns the fallback "—" for undefined input', () => {
    const result = formatShortDate(undefined)
    expect(result).toBe('—')
  })

  it('returns the fallback "—" for an empty string', () => {
    const result = formatShortDate('')
    expect(result).toBe('—')
  })

  it('returns the fallback "—" for a garbage string that is not a date', () => {
    const result = formatShortDate('not-a-date')
    expect(result).toBe('—')
  })

  it('returns the fallback "—" for a null-like empty value', () => {
    // Callers may pass the raw API field which can be an empty string.
    const result = formatShortDate('   ')
    // '   ' parses to NaN in Date — should return fallback.
    expect(result).toBe('—')
  })
})

// ---------------------------------------------------------------------------
// formatFullDateTime
// ---------------------------------------------------------------------------

describe('formatFullDateTime', () => {
  it('returns a non-empty string for a valid RFC3339 timestamp', () => {
    const result = formatFullDateTime('2026-04-27T10:00:00Z')
    expect(result).not.toBe('—')
    expect(result.length).toBeGreaterThan(0)
    expect(result).not.toContain('Invalid')
  })

  it('returns a non-empty string for an RFC3339 timestamp with offset', () => {
    const result = formatFullDateTime('2025-06-15T14:45:30+05:30')
    expect(result).not.toBe('—')
    expect(result).not.toContain('Invalid')
  })

  it('returns a non-empty string for a plain-date string (YYYY-MM-DD)', () => {
    const result = formatFullDateTime('2026-01-01')
    expect(result).not.toBe('—')
    expect(result).not.toContain('Invalid')
  })

  it('returns the fallback "—" for undefined input', () => {
    const result = formatFullDateTime(undefined)
    expect(result).toBe('—')
  })

  it('returns the fallback "—" for an empty string', () => {
    const result = formatFullDateTime('')
    expect(result).toBe('—')
  })

  it('returns the fallback "—" for a garbage string', () => {
    const result = formatFullDateTime('totally-invalid')
    expect(result).toBe('—')
  })
})

// ---------------------------------------------------------------------------
// useFormatDate composable
// ---------------------------------------------------------------------------

describe('useFormatDate', () => {
  it('returns the same formatShortDate and formatFullDateTime functions', async () => {
    const { useFormatDate } = await import('../../web/src/composables/useFormatDate')
    const { formatShortDate: short, formatFullDateTime: full } = useFormatDate()

    // Sanity check: the returned functions behave identically to the named exports.
    expect(short('2026-04-27T00:00:00Z')).toBe(formatShortDate('2026-04-27T00:00:00Z'))
    expect(full('2026-04-27T00:00:00Z')).toBe(formatFullDateTime('2026-04-27T00:00:00Z'))
    expect(short(undefined)).toBe('—')
    expect(full(undefined)).toBe('—')
  })
})
