// SPDX-License-Identifier: AGPL-3.0-or-later

// Mirrors internal/artifact/rice.go — keep the formula, constraints, and
// validation messages in lockstep with the Go implementation (single source
// of truth per requirement §22).

export interface RiceComponents {
  rice_reach?: number | null
  rice_impact?: number | null
  rice_confidence?: number | null
  rice_effort?: number | null
}

export type RiceField = 'rice_reach' | 'rice_impact' | 'rice_confidence' | 'rice_effort'

/** Editor pre-fill only, applied when opening the editor on an item with no RICE fields set. */
export const RICE_DEFAULTS = {
  reach: 100,
  impact: 0.25,
  confidence: 25,
  effort: 1,
} as const

export const IMPACT_TIERS = [0.25, 0.5, 1, 2, 3] as const

/**
 * riceScore derives the RICE score from c's four components:
 * (reach × impact × (confidence/100)) / effort.
 * Returns null ("N/A") unless all four components are present and each
 * satisfies its constraint — mirrors internal/artifact/rice.go RiceScore.
 */
export function riceScore(c: RiceComponents | null | undefined): number | null {
  if (!c) return null
  const { rice_reach: reach, rice_impact: impact, rice_confidence: confidence, rice_effort: effort } = c
  if (reach == null || impact == null || confidence == null || effort == null) return null
  if ([reach, impact, confidence, effort].some((v) => Number.isNaN(v))) return null
  if (reach < 0 || impact < 0) return null
  if (confidence < 0 || confidence > 100) return null
  if (effort <= 0) return null
  return (reach * impact * (confidence / 100)) / effort
}

export function formatRice(v: number | null | undefined): string {
  return v == null ? 'N/A' : v.toFixed(2)
}

/**
 * validateRiceComponent mirrors internal/artifact/rice.go ValidateRiceComponent.
 * A null/undefined value is always valid (the unset state) — it returns a
 * field-level message only when a present value violates its constraint.
 */
export function validateRiceComponent(field: RiceField, value: number | null | undefined): string | null {
  if (value == null) return null
  if (Number.isNaN(value)) return 'must be a number'
  switch (field) {
    case 'rice_reach':
    case 'rice_impact':
      return value < 0 ? 'must be >= 0' : null
    case 'rice_confidence':
      return value < 0 || value > 100 ? 'must be between 0 and 100' : null
    case 'rice_effort':
      return value <= 0 ? 'must be > 0' : null
  }
}
