/**
 * Central widget registration.
 * Import this file once (in main.ts) to register all dashboard widgets.
 * To add a new widget: call registerWidget() here — no other files need editing.
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

registerWidget(
  'status-distribution',
  defineAsyncComponent(() => import('./widgets/StatusDistributionWidget.vue')),
  { slot: 'chart', order: 0 },
)

registerWidget(
  'stages-distribution',
  defineAsyncComponent(() => import('./widgets/StagesDistributionWidget.vue')),
  { slot: 'chart', order: 1 },
)

registerWidget(
  'recent-ideas-defects',
  defineAsyncComponent(() => import('./widgets/RecentIdeasDefectsWidget.vue')),
  { slot: 'chart', order: 1.5 },
)

registerWidget(
  'velocity-chart',
  defineAsyncComponent(() => import('./widgets/VelocityChartWidget.vue')),
  { slot: 'chart', order: 2 },
)

registerWidget(
  'activity-feed',
  defineAsyncComponent(() => import('./widgets/ActivityFeedWidget.vue')),
  { slot: 'panel', order: 0 },
)
