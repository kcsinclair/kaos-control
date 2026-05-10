// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Central widget registration.
 * Import this file once (in main.ts) to register all dashboard widgets.
 * To add a new widget: call registerWidget() here — no other files need editing.
 *
 * Dashboard layout (rendered by DashboardGrid):
 *   Row 1 (summary slot):                 SummaryCounts (4 stat cards, auto-fit)
 *   Row 2 (chart slot, top — 3 cols):     StagesDistribution | StatusDistribution | RecentIdeasDefects
 *   Row 3 (chart slot, bottom — full-w):  VelocityChart
 *   Row 4 (panel slot, full-w per widget): ActivityFeed
 *
 * The chart slot is split by DashboardGrid: the first 3 chart widgets
 * (sorted by order) render in the top row's 3-column grid; the rest render
 * full-width in the bottom row. So time-series widgets like VelocityChart
 * get the horizontal space they need to be readable.
 */
import { defineAsyncComponent } from 'vue'
import { registerWidget } from './widgetRegistry'

// Async imports so each widget chunk (including heavy echarts vendor) is
// loaded lazily — only when the dashboard route is first visited.
registerWidget(
  'summary-counts',
  defineAsyncComponent(() => import('./widgets/SummaryCountsWidget.vue')),
  { slot: 'summary', order: 0 },
)

// Chart top row — three equal columns
registerWidget(
  'stages-distribution',
  defineAsyncComponent(() => import('./widgets/StagesDistributionWidget.vue')),
  { slot: 'chart', order: 0 },
)

registerWidget(
  'status-distribution',
  defineAsyncComponent(() => import('./widgets/StatusDistributionWidget.vue')),
  { slot: 'chart', order: 1 },
)

registerWidget(
  'recent-ideas-defects',
  defineAsyncComponent(() => import('./widgets/RecentIdeasDefectsWidget.vue')),
  { slot: 'chart', order: 2 },
)

// Chart bottom row — full-width
registerWidget(
  'velocity-chart',
  defineAsyncComponent(() => import('./widgets/VelocityChartWidget.vue')),
  { slot: 'chart', order: 3 },
)

// Panel row — full-width below the charts
registerWidget(
  'activity-feed',
  defineAsyncComponent(() => import('./widgets/ActivityFeedWidget.vue')),
  { slot: 'panel', order: 0 },
)
