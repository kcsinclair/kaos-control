export const NODE_COLORS: Record<string, string> = {
  idea:            '#f59e0b',  // amber
  requirement:     '#3b82f6',  // blue
  'plan-backend':  '#8b5cf6',  // violet
  'plan-frontend': '#a78bfa',  // lighter violet
  'plan-test':     '#c084fc',  // lavender
  test:            '#06b6d4',  // cyan
  prototype:       '#14b8a6',  // teal
  defect:          '#f43f5e',  // rose
  label:           '#a855f7',  // purple — synthetic label nodes
  release:         '#93c5fd',  // light blue — scheduled/unscheduled release nodes
  backlog:         '#6b7280',  // gray — synthetic Backlog root node
}

export const PRIORITY_COLORS: Record<string, string> = {
  high:   '#ef4444',
  medium: '#f97316',
  normal: '#22c55e',
  low:    '#3b82f6',
}

export const ACTIVE_STATUS_COLORS: Record<string, string> = {
  'in-development': '#4ade80',  // green
  'in-qa':          '#fbbf24',  // amber
  'in-progress':    '#4ade80',  // green
  'clarifying':     '#60a5fa',  // blue
  'planning':       '#a78bfa',  // violet
}

export const APPROVED_TEST_RING_COLOR = '#2563eb'  // blue-600: approved test ring

export const EDGE_COLORS: Record<string, string> = {
  parent:     '#94a3b8',
  depends_on: '#f97316',
  blocks:     '#ef4444',
  related_to: '#64748b',
  label:      '#a855f7',
}
