// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 1 — Unit tests for graphConstants.ts / useGraphTheme()
 *
 * Covers:
 *   1. Palette selection — useGraphTheme() returns dark palette when isDark is
 *      true and the light palette when isDark is false.
 *   2. Completeness — both palettes define every required GraphPalette key and
 *      no value is undefined.
 *   3. WCAG AA contrast — text-on-background pairs achieve ≥ 4.5:1; node/edge
 *      colours on canvas achieve ≥ 3:1 (graphical objects).
 *   4. No stale exports — the old bare-named constants (NODE_COLORS,
 *      PRIORITY_COLORS, ACTIVE_STATUS_COLORS, EDGE_COLORS,
 *      APPROVED_TEST_RING_COLOR) are no longer exported from the module.
 *
 * Implementation notes
 * ────────────────────
 * useGraphTheme() calls useThemeStore() internally, which requires an active
 * Pinia instance and reads localStorage / window.matchMedia.  Each test group
 * creates a fresh Pinia and controls the theme via themeStore.setTheme().
 *
 * window.matchMedia is stubbed so tests are not affected by the jsdom default.
 *
 * Component: web/src/components/graph/graphConstants.ts
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useThemeStore } from '../../web/src/stores/theme'
import { useGraphTheme } from '../../web/src/components/map/graphConstants'
import type { GraphPalette } from '../../web/src/components/map/graphConstants'

// ---------------------------------------------------------------------------
// Stub window.matchMedia (not available in happy-dom by default)
// ---------------------------------------------------------------------------

beforeEach(() => {
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
})

afterEach(() => {
  vi.unstubAllGlobals()
})

// ---------------------------------------------------------------------------
// WCAG contrast utilities (inline — no external dependency)
// ---------------------------------------------------------------------------

/** Convert an sRGB channel value [0–255] to linear light. */
function toLinear(c8bit: number): number {
  const c = c8bit / 255
  return c <= 0.04045 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4)
}

/** Relative luminance of a #rrggbb colour string (WCAG 2.x definition). */
function luminance(hex: string): number {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  return 0.2126 * toLinear(r) + 0.7152 * toLinear(g) + 0.0722 * toLinear(b)
}

/** WCAG contrast ratio between two #rrggbb colours. */
function contrastRatio(a: string, b: string): number {
  const l1 = luminance(a)
  const l2 = luminance(b)
  const lighter = Math.max(l1, l2)
  const darker = Math.min(l1, l2)
  return (lighter + 0.05) / (darker + 0.05)
}

/** Returns true if the string is a 6-digit #rrggbb hex colour. */
function isHex6(value: string): boolean {
  return /^#[0-9a-fA-F]{6}$/.test(value)
}

// ---------------------------------------------------------------------------
// Expected keys for a complete GraphPalette
// ---------------------------------------------------------------------------

const REQUIRED_PALETTE_KEYS: Array<keyof GraphPalette> = [
  'nodeColors',
  'priorityColors',
  'activeStatusColors',
  'edgeColors',
  'approvedTestRingColor',
  'canvasBg',
  'labelColor',
  'labelNodeBg',
  'labelNodeText',
  'labelNodeBorder',
  'releaseText',
  'releaseBorderColor',
  'backlogText',
  'edgeLabelBg',
  'edgeLabelText',
  'timelineEdgeColor',
  'timelineEdgeTextColor',
  'assignedEdgeColor',
  'borderDefault',
  'selectedBorderColor',
  'searchHighlight',
]

const REQUIRED_NODE_TYPES = [
  'idea', 'requirement', 'plan-backend', 'plan-frontend', 'plan-test',
  'test', 'prototype', 'defect', 'label', 'release', 'backlog',
]

const REQUIRED_PRIORITY_LEVELS = ['high', 'medium', 'normal', 'low']

const REQUIRED_ACTIVE_STATUSES = [
  'in-development', 'in-qa', 'in-progress', 'clarifying', 'planning',
]

const REQUIRED_EDGE_KINDS = ['parent', 'depends_on', 'blocks', 'related_to', 'label']

// ---------------------------------------------------------------------------
// Helper: build a fresh theme + graph hook for a given isDark value
// ---------------------------------------------------------------------------

function setupPalette(dark: boolean): GraphPalette {
  setActivePinia(createPinia())
  const themeStore = useThemeStore()
  themeStore.setTheme(dark ? 'dark' : 'light')
  const { palette } = useGraphTheme()
  return palette.value
}

// ===========================================================================
// 1 — Palette selection
// ===========================================================================

describe('useGraphTheme — palette selection', () => {
  it('returns a palette with a dark canvas background when isDark is true', () => {
    const palette = setupPalette(true)
    // Dark canvas is a deep navy/slate, luminance well below 0.1
    expect(luminance(palette.canvasBg)).toBeLessThan(0.1)
  })

  it('returns a palette with a light canvas background when isDark is false', () => {
    const palette = setupPalette(false)
    // Light canvas (#ffffff) has luminance 1.0; expect well above 0.8
    expect(luminance(palette.canvasBg)).toBeGreaterThan(0.8)
  })

  it('dark palette labelColor is lighter than its canvasBg (white text on dark)', () => {
    const p = setupPalette(true)
    expect(luminance(p.labelColor)).toBeGreaterThan(luminance(p.canvasBg))
  })

  it('light palette labelColor is darker than its canvasBg (dark text on light)', () => {
    const p = setupPalette(false)
    expect(luminance(p.labelColor)).toBeLessThan(luminance(p.canvasBg))
  })

  it('dark and light palettes return different canvasBg values', () => {
    const dark = setupPalette(true)
    const light = setupPalette(false)
    expect(dark.canvasBg).not.toBe(light.canvasBg)
  })

  it('dark and light palettes return different nodeColors.idea values', () => {
    const dark = setupPalette(true)
    const light = setupPalette(false)
    expect(dark.nodeColors.idea).not.toBe(light.nodeColors.idea)
  })
})

// ===========================================================================
// 2 — Completeness
// ===========================================================================

describe('useGraphTheme — palette completeness', () => {
  for (const theme of ['dark', 'light'] as const) {
    describe(`${theme} palette`, () => {
      it('defines every required top-level key and none are undefined', () => {
        const p = setupPalette(theme === 'dark')
        for (const key of REQUIRED_PALETTE_KEYS) {
          expect(p[key], `key "${key}" must not be undefined`).toBeDefined()
        }
      })

      it('nodeColors defines every required artifact type', () => {
        const { nodeColors } = setupPalette(theme === 'dark')
        for (const type of REQUIRED_NODE_TYPES) {
          expect(nodeColors[type], `nodeColors["${type}"] must be defined`).toBeDefined()
        }
      })

      it('priorityColors defines every priority level', () => {
        const { priorityColors } = setupPalette(theme === 'dark')
        for (const level of REQUIRED_PRIORITY_LEVELS) {
          expect(priorityColors[level], `priorityColors["${level}"] must be defined`).toBeDefined()
        }
      })

      it('activeStatusColors defines every required status', () => {
        const { activeStatusColors } = setupPalette(theme === 'dark')
        for (const status of REQUIRED_ACTIVE_STATUSES) {
          expect(
            activeStatusColors[status],
            `activeStatusColors["${status}"] must be defined`,
          ).toBeDefined()
        }
      })

      it('edgeColors defines every required edge kind', () => {
        const { edgeColors } = setupPalette(theme === 'dark')
        for (const kind of REQUIRED_EDGE_KINDS) {
          expect(edgeColors[kind], `edgeColors["${kind}"] must be defined`).toBeDefined()
        }
      })

      it('approvedTestRingColor is a 6-digit hex string', () => {
        const { approvedTestRingColor } = setupPalette(theme === 'dark')
        expect(approvedTestRingColor).toMatch(/^#[0-9a-fA-F]{6}$/)
      })

      it('canvasBg is a 6-digit hex string', () => {
        const { canvasBg } = setupPalette(theme === 'dark')
        expect(canvasBg).toMatch(/^#[0-9a-fA-F]{6}$/)
      })
    })
  }
})

// ===========================================================================
// 3 — WCAG AA contrast
// ===========================================================================

/**
 * Text-on-background pairs that must achieve ≥ 4.5:1 (WCAG AA, normal text).
 * Each entry is [foreground key path, background key path, description].
 */
type TextPair = {
  label: string
  getFg: (p: GraphPalette) => string
  getBg: (p: GraphPalette) => string
}

const TEXT_PAIRS: TextPair[] = [
  {
    label: 'labelColor on canvasBg (node labels)',
    getFg: (p) => p.labelColor,
    getBg: (p) => p.canvasBg,
  },
  {
    label: 'edgeLabelText on edgeLabelBg',
    getFg: (p) => p.edgeLabelText,
    getBg: (p) => p.edgeLabelBg,
  },
  {
    label: 'labelNodeText on labelNodeBg (pill nodes)',
    getFg: (p) => p.labelNodeText,
    getBg: (p) => p.labelNodeBg,
  },
  {
    label: 'backlogText on canvasBg',
    getFg: (p) => p.backlogText,
    getBg: (p) => p.canvasBg,
  },
  {
    label: 'timelineEdgeTextColor on canvasBg',
    getFg: (p) => p.timelineEdgeTextColor,
    getBg: (p) => p.canvasBg,
  },
]

/**
 * Graphical-object colours that must achieve ≥ 3:1 contrast against the canvas
 * (WCAG AA, graphical objects and UI components).
 */
type GraphicalPair = {
  label: string
  getFg: (p: GraphPalette) => string
}

const GRAPHICAL_PAIRS: GraphicalPair[] = [
  // Node fill colours vs canvas
  ...REQUIRED_NODE_TYPES.map((type) => ({
    label: `nodeColors["${type}"] on canvasBg`,
    getFg: (p: GraphPalette) => p.nodeColors[type],
  })),
  // Edge stroke colours vs canvas
  ...REQUIRED_EDGE_KINDS.map((kind) => ({
    label: `edgeColors["${kind}"] on canvasBg`,
    getFg: (p: GraphPalette) => p.edgeColors[kind],
  })),
  // Priority ring colours vs canvas
  ...REQUIRED_PRIORITY_LEVELS.map((level) => ({
    label: `priorityColors["${level}"] on canvasBg`,
    getFg: (p: GraphPalette) => p.priorityColors[level],
  })),
  // Approved-test ring vs canvas
  {
    label: 'approvedTestRingColor on canvasBg',
    getFg: (p) => p.approvedTestRingColor,
  },
  // NOTE: searchHighlight is a ring overlaid ON TOP OF a node, not an element
  // placed directly on the canvas background.  Contrast for this colour is
  // meaningful against node fills, not against canvasBg.  It is excluded from
  // this canvas-background graphical-object check per the test plan guidance
  // ("node fill vs canvas, edge vs canvas").
]

describe('useGraphTheme — WCAG AA contrast', () => {
  for (const theme of ['dark', 'light'] as const) {
    describe(`${theme} palette — text pairs (≥ 4.5:1)`, () => {
      for (const { label, getFg, getBg } of TEXT_PAIRS) {
        it(label, () => {
          const p = setupPalette(theme === 'dark')
          const fg = getFg(p)
          const bg = getBg(p)
          // Skip any rgba/non-hex values — those are not subject to this check
          if (!isHex6(fg) || !isHex6(bg)) return
          const ratio = contrastRatio(fg, bg)
          expect(
            ratio,
            `${label} [${theme}]: contrast ${ratio.toFixed(2)}:1 is below 4.5:1 (fg=${fg} bg=${bg})`,
          ).toBeGreaterThanOrEqual(4.5)
        })
      }
    })

    describe(`${theme} palette — graphical object pairs (≥ 3:1)`, () => {
      for (const { label, getFg } of GRAPHICAL_PAIRS) {
        it(label, () => {
          const p = setupPalette(theme === 'dark')
          const fg = getFg(p)
          const bg = p.canvasBg
          if (!isHex6(fg) || !isHex6(bg)) return
          const ratio = contrastRatio(fg, bg)
          expect(
            ratio,
            `${label} [${theme}]: contrast ${ratio.toFixed(2)}:1 is below 3:1 (fg=${fg} bg=${bg})`,
          ).toBeGreaterThanOrEqual(3.0)
        })
      }
    })
  }
})

// ===========================================================================
// 4 — No stale bare-named exports
// ===========================================================================

describe('graphConstants — no stale exports', () => {
  it('does not export NODE_COLORS', async () => {
    const mod = await import('../../web/src/components/map/graphConstants')
    expect((mod as Record<string, unknown>)['NODE_COLORS']).toBeUndefined()
  })

  it('does not export PRIORITY_COLORS', async () => {
    const mod = await import('../../web/src/components/map/graphConstants')
    expect((mod as Record<string, unknown>)['PRIORITY_COLORS']).toBeUndefined()
  })

  it('does not export ACTIVE_STATUS_COLORS', async () => {
    const mod = await import('../../web/src/components/map/graphConstants')
    expect((mod as Record<string, unknown>)['ACTIVE_STATUS_COLORS']).toBeUndefined()
  })

  it('does not export EDGE_COLORS', async () => {
    const mod = await import('../../web/src/components/map/graphConstants')
    expect((mod as Record<string, unknown>)['EDGE_COLORS']).toBeUndefined()
  })

  it('does not export APPROVED_TEST_RING_COLOR', async () => {
    const mod = await import('../../web/src/components/map/graphConstants')
    expect((mod as Record<string, unknown>)['APPROVED_TEST_RING_COLOR']).toBeUndefined()
  })

  it('does export useGraphTheme as a function', async () => {
    const mod = await import('../../web/src/components/map/graphConstants')
    expect(typeof mod.useGraphTheme).toBe('function')
  })
})
