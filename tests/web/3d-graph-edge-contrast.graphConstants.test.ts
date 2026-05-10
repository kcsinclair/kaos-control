// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * 3d-graph-edge-contrast — Milestones 1 & 2
 *
 * Milestone 1 — Contrast ratio verification
 *   Asserts that every edge colour in both palettes achieves ≥ 3:1 contrast
 *   against the palette's canvas background (WCAG graphical-object minimum).
 *   Specifically validates the updated `timeline` and `assigned` entries which
 *   were the colours that motivated this feature.
 *
 * Milestone 2 — Palette consistency
 *   Asserts that the named shorthand properties (`timelineEdgeColor`,
 *   `assignedEdgeColor`) exactly match the corresponding `edgeColors` map
 *   entries within each palette, ensuring no silent divergence.
 *
 * Component: web/src/components/map/graphConstants.ts
 * Test plan: lifecycle/test-plans/3d-graph-edge-contrast-5-test.md
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
// WCAG contrast utilities
// ---------------------------------------------------------------------------

function toLinear(c8bit: number): number {
  const c = c8bit / 255
  return c <= 0.04045 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4)
}

function luminance(hex: string): number {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  return 0.2126 * toLinear(r) + 0.7152 * toLinear(g) + 0.0722 * toLinear(b)
}

function contrastRatio(fg: string, bg: string): number {
  const l1 = luminance(fg)
  const l2 = luminance(bg)
  const lighter = Math.max(l1, l2)
  const darker = Math.min(l1, l2)
  return (lighter + 0.05) / (darker + 0.05)
}

// ---------------------------------------------------------------------------
// Helper: resolve palette for a given theme
// ---------------------------------------------------------------------------

function setupPalette(dark: boolean): GraphPalette {
  setActivePinia(createPinia())
  const themeStore = useThemeStore()
  themeStore.setTheme(dark ? 'dark' : 'light')
  const { palette } = useGraphTheme()
  return palette.value
}

// ---------------------------------------------------------------------------
// All edge kinds that must be present (including the new ones from this
// feature: timeline and assigned).
// ---------------------------------------------------------------------------

const ALL_EDGE_KINDS = [
  'parent',
  'depends_on',
  'blocks',
  'related_to',
  'label',
  'timeline',   // added by 3d-graph-edge-contrast
  'assigned',   // added by 3d-graph-edge-contrast
]

// ===========================================================================
// Milestone 1 — Edge colour contrast >= 3:1 against canvas background
// ===========================================================================

describe('3d-graph-edge-contrast — Milestone 1: edge colour contrast', () => {
  for (const theme of ['dark', 'light'] as const) {
    describe(`${theme} palette — all edge kinds >= 3:1 on canvasBg`, () => {
      for (const kind of ALL_EDGE_KINDS) {
        it(`edgeColors["${kind}"] achieves >= 3:1 contrast on canvasBg`, () => {
          const p = setupPalette(theme === 'dark')
          const fg = p.edgeColors[kind]
          const bg = p.canvasBg
          expect(fg, `edgeColors["${kind}"] must be defined in ${theme} palette`).toBeDefined()
          const ratio = contrastRatio(fg, bg)
          expect(
            ratio,
            `${theme} edgeColors["${kind}"] contrast ${ratio.toFixed(2)}:1 < 3:1 (fg=${fg} bg=${bg})`,
          ).toBeGreaterThanOrEqual(3.0)
        })
      }
    })
  }

  // Explicit tests for the two colours that were the motivation for this feature

  it('dark palette: assigned edge (#475569) achieves >= 3:1 on dark canvas (#0f172a)', () => {
    const p = setupPalette(true)
    expect(p.canvasBg).toBe('#0f172a')
    const ratio = contrastRatio(p.edgeColors['assigned'], p.canvasBg)
    expect(
      ratio,
      `assigned edge contrast ${ratio.toFixed(2)}:1 is below 3:1`,
    ).toBeGreaterThanOrEqual(3.0)
  })

  it('light palette: assigned edge (#64748b) achieves >= 3:1 on light canvas (#ffffff)', () => {
    const p = setupPalette(false)
    expect(p.canvasBg).toBe('#ffffff')
    const ratio = contrastRatio(p.edgeColors['assigned'], p.canvasBg)
    expect(
      ratio,
      `assigned edge contrast ${ratio.toFixed(2)}:1 is below 3:1`,
    ).toBeGreaterThanOrEqual(3.0)
  })

  it('dark palette: timeline edge achieves >= 3:1 on dark canvas', () => {
    const p = setupPalette(true)
    const ratio = contrastRatio(p.edgeColors['timeline'], p.canvasBg)
    expect(ratio, `timeline contrast ${ratio.toFixed(2)}:1 is below 3:1`).toBeGreaterThanOrEqual(3.0)
  })

  it('light palette: timeline edge achieves >= 3:1 on light canvas', () => {
    const p = setupPalette(false)
    const ratio = contrastRatio(p.edgeColors['timeline'], p.canvasBg)
    expect(ratio, `timeline contrast ${ratio.toFixed(2)}:1 is below 3:1`).toBeGreaterThanOrEqual(3.0)
  })
})

// ===========================================================================
// Milestone 2 — Named shorthand properties must be in sync with edgeColors map
// ===========================================================================

describe('3d-graph-edge-contrast — Milestone 2: palette consistency', () => {
  for (const theme of ['dark', 'light'] as const) {
    it(`${theme} palette: timelineEdgeColor === edgeColors["timeline"]`, () => {
      const p = setupPalette(theme === 'dark')
      expect(p.timelineEdgeColor).toBe(p.edgeColors['timeline'])
    })

    it(`${theme} palette: assignedEdgeColor === edgeColors["assigned"]`, () => {
      const p = setupPalette(theme === 'dark')
      expect(p.assignedEdgeColor).toBe(p.edgeColors['assigned'])
    })
  }

  it('both palettes define timeline and assigned in edgeColors', () => {
    for (const dark of [true, false]) {
      const p = setupPalette(dark)
      const theme = dark ? 'dark' : 'light'
      expect(p.edgeColors['timeline'], `${theme} edgeColors.timeline`).toBeDefined()
      expect(p.edgeColors['assigned'], `${theme} edgeColors.assigned`).toBeDefined()
      expect(p.timelineEdgeColor, `${theme} timelineEdgeColor`).toBeDefined()
      expect(p.assignedEdgeColor, `${theme} assignedEdgeColor`).toBeDefined()
    }
  })
})
