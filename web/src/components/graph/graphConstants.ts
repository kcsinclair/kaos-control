// SPDX-License-Identifier: AGPL-3.0-or-later

import { computed } from 'vue'
import { useThemeStore } from '@/stores/theme'

export interface GraphPalette {
  nodeColors: Record<string, string>
  priorityColors: Record<string, string>
  activeStatusColors: Record<string, string>
  edgeColors: Record<string, string>
  approvedTestRingColor: string
  canvasBg: string
  /** Text colour on regular nodes */
  labelColor: string
  /** Background of 'label'-type pill nodes */
  labelNodeBg: string
  /** Text colour inside 'label'-type pill nodes */
  labelNodeText: string
  /** Border colour of 'label'-type pill nodes */
  labelNodeBorder: string
  /** Text colour on release box nodes */
  releaseText: string
  /** Border colour of release box nodes */
  releaseBorderColor: string
  /** Text colour on synthetic backlog nodes */
  backlogText: string
  /** Edge label background */
  edgeLabelBg: string
  /** Edge label text colour */
  edgeLabelText: string
  /** Stroke colour of timeline edges */
  timelineEdgeColor: string
  /** Text colour of timeline edge duration labels */
  timelineEdgeTextColor: string
  /** Stroke colour of assigned (artifact→release) edges */
  assignedEdgeColor: string
  /** Default node border colour */
  borderDefault: string
  /** Node border colour when selected */
  selectedBorderColor: string
  /** Search-match highlight ring colour */
  searchHighlight: string
  /** Dim-blend colour for unmatched nodes: matches the canvas background at reduced opacity */
  dimBlend: string
}

// ─── Dark palette ─────────────────────────────────────────────────────────────

const DARK_PALETTE: GraphPalette = {
  nodeColors: {
    idea:            '#f59e0b',  // amber-400
    requirement:     '#3b82f6',  // blue-500
    'plan-backend':  '#8b5cf6',  // violet-500
    'plan-frontend': '#a78bfa',  // violet-400
    'plan-test':     '#c084fc',  // purple-400
    test:            '#06b6d4',  // cyan-500
    prototype:       '#14b8a6',  // teal-500
    defect:          '#f43f5e',  // rose-500
    label:           '#a855f7',  // purple-500
    release:         '#93c5fd',  // blue-300
    backlog:         '#6b7280',  // gray-500
  },
  priorityColors: {
    high:   '#ef4444',  // red-500
    medium: '#f97316',  // orange-500
    normal: '#22c55e',  // green-500
    low:    '#3b82f6',  // blue-500
  },
  activeStatusColors: {
    'in-development': '#4ade80',  // green-400
    'in-qa':          '#fbbf24',  // amber-400
    'in-progress':    '#4ade80',  // green-400
    'clarifying':     '#60a5fa',  // blue-400
    'planning':       '#a78bfa',  // violet-400
  },
  edgeColors: {
    parent:     '#94a3b8',  // slate-400
    depends_on: '#f97316',  // orange-500
    blocks:     '#ef4444',  // red-500
    related_to: '#64748b',  // slate-500
    label:      '#a855f7',  // purple-500
    timeline:   '#3b82f6',  // blue-500 — matches timelineEdgeColor
    assigned:   '#334155',  // slate-700 — matches assignedEdgeColor
  },
  approvedTestRingColor: '#2563eb',  // blue-600
  canvasBg:             '#0f172a',
  labelColor:           '#f1f5f9',
  labelNodeBg:          '#2e1a4a',
  labelNodeText:        '#d8b4fe',
  labelNodeBorder:      '#a855f7',
  releaseText:          '#1e3a5f',
  releaseBorderColor:   '#60a5fa',   // blue-400
  backlogText:          '#d1d5db',
  edgeLabelBg:          '#1e293b',
  edgeLabelText:        '#94a3b8',
  timelineEdgeColor:    '#3b82f6',   // blue-500
  timelineEdgeTextColor:'#93c5fd',   // blue-300
  assignedEdgeColor:    '#334155',   // slate-700
  borderDefault:        'rgba(255,255,255,0.25)',
  selectedBorderColor:  '#ffffff',
  searchHighlight:      '#facc15',   // yellow-400
  dimBlend:             '#1e2535',
}

// ─── Light palette ────────────────────────────────────────────────────────────

const LIGHT_PALETTE: GraphPalette = {
  nodeColors: {
    idea:            '#d97706',  // amber-600   — darker for contrast on white
    requirement:     '#2563eb',  // blue-600
    'plan-backend':  '#7c3aed',  // violet-600
    'plan-frontend': '#8b5cf6',  // violet-500
    'plan-test':     '#9333ea',  // purple-600
    test:            '#0891b2',  // cyan-600
    prototype:       '#0d9488',  // teal-600
    defect:          '#e11d48',  // rose-600
    label:           '#9333ea',  // purple-600
    release:         '#3b82f6',  // blue-500
    backlog:         '#4b5563',  // gray-600
  },
  priorityColors: {
    high:   '#dc2626',  // red-600
    medium: '#ea580c',  // orange-600
    normal: '#16a34a',  // green-600
    low:    '#2563eb',  // blue-600
  },
  activeStatusColors: {
    'in-development': '#16a34a',  // green-600
    'in-qa':          '#d97706',  // amber-600
    'in-progress':    '#16a34a',  // green-600
    'clarifying':     '#2563eb',  // blue-600
    'planning':       '#7c3aed',  // violet-600
  },
  edgeColors: {
    parent:     '#64748b',  // slate-500
    depends_on: '#ea580c',  // orange-600
    blocks:     '#dc2626',  // red-600
    related_to: '#475569',  // slate-600
    label:      '#7c3aed',  // violet-600
    timeline:   '#2563eb',  // blue-600 — matches timelineEdgeColor
    assigned:   '#94a3b8',  // slate-400 — matches assignedEdgeColor
  },
  approvedTestRingColor: '#1d4ed8',  // blue-700
  canvasBg:             '#ffffff',
  labelColor:           '#0f172a',   // slate-900 — dark text on light canvas
  labelNodeBg:          '#f3e8ff',   // purple-100
  labelNodeText:        '#6b21a8',   // purple-800
  labelNodeBorder:      '#a855f7',   // purple-500
  releaseText:          '#1e3a5f',   // dark blue (same as dark)
  releaseBorderColor:   '#2563eb',   // blue-600
  backlogText:          '#374151',   // gray-700
  edgeLabelBg:          '#f1f5f9',   // slate-100
  edgeLabelText:        '#334155',   // slate-700
  timelineEdgeColor:    '#2563eb',   // blue-600
  timelineEdgeTextColor:'#1d4ed8',   // blue-700
  assignedEdgeColor:    '#94a3b8',   // slate-400
  borderDefault:        'rgba(0,0,0,0.15)',
  selectedBorderColor:  '#000000',
  searchHighlight:      '#eab308',   // yellow-500 — darker for light bg
  dimBlend:             '#d1d5db',
}

// ─── Composable ───────────────────────────────────────────────────────────────

export function useGraphTheme() {
  const themeStore = useThemeStore()
  // Wrap in a new computed so callers get a ComputedRef<boolean> suitable for watch()
  const isDark = computed<boolean>(() => themeStore.isDark)
  const palette = computed<GraphPalette>(() => (isDark.value ? DARK_PALETTE : LIGHT_PALETTE))
  return { palette, isDark }
}

