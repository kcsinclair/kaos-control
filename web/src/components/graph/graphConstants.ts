export const NODE_COLORS: Record<string, string> = {
  idea:            '#f59e0b',  // amber
  ticket:          '#3b82f6',  // blue
  epic:            '#1d4ed8',  // darker blue
  'plan-backend':  '#8b5cf6',  // violet
  'plan-frontend': '#a78bfa',  // lighter violet
  'plan-dev':      '#7c3aed',  // deep violet
  'plan-test':     '#c084fc',  // lavender
  test:            '#06b6d4',  // cyan
  prototype:       '#14b8a6',  // teal
  release:         '#ef4444',  // red
  sprint:          '#ec4899',  // pink
  defect:          '#f43f5e',  // rose
  label:           '#a855f7',  // purple — synthetic label nodes
}

export const PRIORITY_COLORS: Record<string, string> = {
  high:   '#ef4444',
  medium: '#f97316',
  normal: '#22c55e',
  low:    '#3b82f6',
}

export const EDGE_COLORS: Record<string, string> = {
  parent:     '#94a3b8',
  depends_on: '#f97316',
  blocks:     '#ef4444',
  related_to: '#64748b',
  label:      '#a855f7',
}
