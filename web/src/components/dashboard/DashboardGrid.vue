<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed } from 'vue'
import { widgetList } from './widgetRegistry'

defineProps<{ project: string }>()

// Chart slot is split into two rows:
//   • top:    first 3 chart widgets, rendered as a 3-column grid.
//   • bottom: the remainder (excluding velocity-chart), rendered full-width per widget.
// velocity-chart and activity-feed are rendered in a dedicated side-by-side row.
const TOP_CHART_COUNT = 3

// IDs that belong in the side-by-side row rather than their slot's default section.
const SIDE_BY_SIDE_IDS = new Set(['velocity-chart', 'activity-feed'])

const summaryWidgets = computed(() => widgetList.filter((w) => w.slot === 'summary'))

const sortedChartWidgets = computed(() =>
  widgetList.filter((w) => w.slot === 'chart').sort((a, b) => a.order - b.order),
)
const topChartWidgets = computed(() => sortedChartWidgets.value.slice(0, TOP_CHART_COUNT))
const bottomChartWidgets = computed(() =>
  sortedChartWidgets.value.slice(TOP_CHART_COUNT).filter((w) => !SIDE_BY_SIDE_IDS.has(w.id)),
)

const panelWidgets = computed(() =>
  widgetList.filter((w) => w.slot === 'panel' && !SIDE_BY_SIDE_IDS.has(w.id)),
)

// Side-by-side row: velocity chart (left/top) then activity feed (right/bottom).
const velocityWidget = computed(() => widgetList.find((w) => w.id === 'velocity-chart'))
const activityFeedWidget = computed(() => widgetList.find((w) => w.id === 'activity-feed'))
</script>

<template>
  <div class="dashboard-grid">
    <!-- Row 1: Summary stat cards -->
    <section
      v-if="summaryWidgets.length"
      class="dashboard-summary"
      aria-label="Summary statistics"
    >
      <component
        :is="widget.component"
        v-for="widget in summaryWidgets"
        :key="widget.id"
        :project="project"
      />
    </section>

    <!-- Charts section: two stacked sub-rows.
         Top:    first 3 chart widgets in a 3-column grid.
         Bottom: remaining chart widgets (excl. velocity), full-width per widget. -->
    <section
      v-if="topChartWidgets.length || bottomChartWidgets.length"
      class="dashboard-charts"
      aria-label="Charts"
    >
      <div v-if="topChartWidgets.length" class="dashboard-charts-top">
        <div
          v-for="widget in topChartWidgets"
          :key="widget.id"
          class="chart-cell"
          :style="widget.span && widget.span > 1 ? { gridColumn: `span ${widget.span}` } : {}"
        >
          <component :is="widget.component" :project="project" />
        </div>
      </div>

      <div v-if="bottomChartWidgets.length" class="dashboard-charts-bottom">
        <div
          v-for="widget in bottomChartWidgets"
          :key="widget.id"
          class="chart-cell"
        >
          <component :is="widget.component" :project="project" />
        </div>
      </div>
    </section>

    <!-- Side-by-side row: Completion Velocity (left) + Recent Activity (right).
         Two equal columns on viewports >= 768 px, stacked below that. -->
    <section
      v-if="velocityWidget || activityFeedWidget"
      class="dashboard-side-by-side"
      aria-label="Velocity and activity"
    >
      <component
        :is="velocityWidget!.component"
        v-if="velocityWidget"
        :project="project"
      />
      <component
        :is="activityFeedWidget!.component"
        v-if="activityFeedWidget"
        :project="project"
      />
    </section>

    <!-- Panel row: full-width widgets below all charts (excl. activity-feed) -->
    <section
      v-if="panelWidgets.length"
      class="dashboard-panels"
      aria-label="Panels"
    >
      <component
        :is="widget.component"
        v-for="widget in panelWidgets"
        :key="widget.id"
        :project="project"
      />
    </section>
  </div>
</template>

<style scoped>
/* Responsive breakpoint for the side-by-side row */
:root {
  --dashboard-side-by-side-bp: 768px;
}

.dashboard-grid {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  min-width: 0;
  box-sizing: border-box;
}

/* Row 1: auto-fit summary cards, min 150 px each */
.dashboard-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: var(--space-3);
}

/* Charts section: stacks the top and bottom sub-rows */
.dashboard-charts {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  min-width: 0;
}

/* Top sub-row: 3-column at desktop, single column on mobile */
.dashboard-charts-top {
  display: grid;
  grid-template-columns: 1fr;
  gap: var(--space-4);
  min-width: 0;
}

@media (min-width: 1024px) {
  .dashboard-charts-top {
    grid-template-columns: repeat(3, 1fr);
  }
}

/* Bottom sub-row: full-width per widget */
.dashboard-charts-bottom {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  min-width: 0;
}

.chart-cell {
  min-width: 0;
}

/* Side-by-side row: velocity (left/top) + activity feed (right/bottom).
   Two equal columns on desktop, single stacked column on mobile.
   align-items: start keeps widgets top-aligned. */
.dashboard-side-by-side {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-4);
  align-items: start;
  min-width: 0;
}

@media (max-width: 767px) {
  .dashboard-side-by-side {
    grid-template-columns: 1fr;
  }
}

/* Panels stack full-width below the chart rows */
.dashboard-panels {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  min-width: 0;
}
</style>
