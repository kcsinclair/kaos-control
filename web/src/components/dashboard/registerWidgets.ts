// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Central widget registration.
 * Import this file once (in main.ts) to register all dashboard widgets.
 * To add a new widget: call registerWidget() here — no other files need editing.
 *
 * Dashboard layout:
 *   Row 1 (summary): SummaryCounts
 *   Row 2 (chart, 3 equal cols): StagesDistribution | StatusDistribution | RecentIdeasDefects
 *   Row 3 (chart, 3-col grid): VelocityChart [span 2] | ActivityFeed [span 1]
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

// Row 2 — three equal columns
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

// Row 3 — velocity spans cols 1–2, activity feed in col 3
registerWidget(
  'velocity-chart',
  defineAsyncComponent(() => import('./widgets/VelocityChartWidget.vue')),
  { slot: 'chart', order: 3, span: 2 },
)

registerWidget(
  'activity-feed',
  defineAsyncComponent(() => import('./widgets/ActivityFeedWidget.vue')),
  { slot: 'chart', order: 4 },
)
