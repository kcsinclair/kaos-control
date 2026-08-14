// SPDX-License-Identifier: AGPL-3.0-or-later

import type { GraphEdge } from '@/types/api'

// ─── Decision-signal node encoding (FR-5) ──────────────────────────────────

export type ScaleBucket = 'low' | 'medium' | 'high' | 'neutral'

export const SCALE_COLOURS: Record<ScaleBucket, string> = {
  low: '#22c55e',      // green-500
  medium: '#eab308',   // yellow-500
  high: '#f97316',     // orange-500
  neutral: '#3b82f6',  // blue-500 — the fixed "otherwise" default
}

/**
 * Classifies a node's scale signal from its labels. Case-insensitive; the
 * first label (in array order) starting with low/medium/high wins. A node
 * with no matching label — including one with no labels at all — is
 * `neutral`, the fixed default (resolved question).
 */
export function scaleBucket(labels?: string[]): ScaleBucket {
  if (labels) {
    for (const label of labels) {
      const lower = label.toLowerCase()
      if (lower.startsWith('low')) return 'low'
      if (lower.startsWith('medium')) return 'medium'
      if (lower.startsWith('high')) return 'high'
    }
  }
  return 'neutral'
}

export function scaleColour(labels?: string[]): string {
  return SCALE_COLOURS[scaleBucket(labels)]
}

export const SCALE_BUCKET_LABELS: Record<ScaleBucket, string> = {
  low: 'Low scale',
  medium: 'Medium scale',
  high: 'High scale',
  neutral: 'Unspecified scale',
}

export interface SignalGlyph {
  key: 'offline' | 'mobile'
  icon: string
  label: string
}

/**
 * Offline-capable/mobile signals derived from labels (e.g. `offline*`,
 * `mobile*`), rendered as text glyphs — never colour alone (NFR-4). Neutral
 * default (no glyphs) when no relevant labels are present.
 */
export function signalGlyphs(labels?: string[]): SignalGlyph[] {
  if (!labels) return []
  const lower = labels.map((l) => l.toLowerCase())
  const glyphs: SignalGlyph[] = []
  if (lower.some((l) => l.startsWith('offline'))) {
    glyphs.push({ key: 'offline', icon: 'OFF', label: 'Offline-capable' })
  }
  if (lower.some((l) => l.startsWith('mobile'))) {
    glyphs.push({ key: 'mobile', icon: 'MOB', label: 'Mobile' })
  }
  return glyphs
}

export interface ArchNodeStyle {
  color: string
  glyphs: SignalGlyph[]
}

export function nodeStyle(node: { labels?: string[] }): ArchNodeStyle {
  return { color: scaleColour(node.labels), glyphs: signalGlyphs(node.labels) }
}

// ─── Typed relationship edge styling (FR-4) ────────────────────────────────

export interface ArchEdgeStyle {
  lineStyle: 'solid' | 'dashed'
  arrow: boolean
  width: number
  /** On-canvas label; empty for generic/undirected edges (2D only, FR-4). */
  label: string
}

const GENERIC_EDGE_STYLE: ArchEdgeStyle = { lineStyle: 'solid', arrow: false, width: 1, label: '' }

const EDGE_STYLES: Record<string, ArchEdgeStyle> = {
  evolves_into: { lineStyle: 'solid', arrow: true, width: 2.5, label: 'Evolves into' },
  alternative_to: { lineStyle: 'dashed', arrow: false, width: 1.5, label: 'Alternative to' },
  composed_with: { lineStyle: 'solid', arrow: false, width: 3.5, label: 'Composed with' },
  related_to: { ...GENERIC_EDGE_STYLE },
  related: { ...GENERIC_EDGE_STYLE },
}

/** Unknown/missing typed fields degrade to the generic style (FR-4/NFR-5). */
export function edgeStyle(kind: string): ArchEdgeStyle {
  return EDGE_STYLES[kind] ?? GENERIC_EDGE_STYLE
}

export function edgeStyleForEdge(edge: GraphEdge): ArchEdgeStyle {
  return edgeStyle(edge.kind)
}
