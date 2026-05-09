<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed } from 'vue'
import { widgetList } from './widgetRegistry'

defineProps<{ project: string }>()

const summaryWidgets = computed(() => widgetList.filter((w) => w.slot === 'summary'))
const topChartWidgets = computed(() =>
  widgetList.filter((w) => w.slot === 'chart' && w.order < 2),
)
const bottomChartWidgets = computed(() =>
  widgetList.filter((w) => w.slot === 'chart' && w.order >= 2),
)
const panelWidgets = computed(() => widgetList.filter((w) => w.slot === 'panel'))
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

    <!--
      Row 2: 3-column grid — orders 0, 1, 1.5
        Stages Distribution | Status Distribution | Recent Ideas & Defects
    -->
    <section
      v-if="topChartWidgets.length"
      class="dashboard-charts-top"
      aria-label="Charts top"
    >
      <div
        v-for="widget in topChartWidgets"
        :key="widget.id"
        class="chart-cell"
        :style="widget.span && widget.span > 1 ? { gridColumn: `span ${widget.span}` } : {}"
      >
        <component :is="widget.component" :project="project" />
      </div>
    </section>

    <!--
      Row 3: 3-column grid — orders ≥ 2
        Completion Velocity (span 2)
    -->
    <section
      v-if="bottomChartWidgets.length"
      class="dashboard-charts-bottom"
      aria-label="Charts bottom"
    >
      <div
        v-for="widget in bottomChartWidgets"
        :key="widget.id"
        class="chart-cell"
        :style="widget.span && widget.span > 1 ? { gridColumn: `span ${widget.span}` } : {}"
      >
        <component :is="widget.component" :project="project" />
      </div>
    </section>

    <!-- Panel row: full-width widgets (e.g. activity feed) -->
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

/* Rows 2 & 3: 3-column grid (stacked on mobile) */
.dashboard-charts-top,
.dashboard-charts-bottom {
  display: grid;
  grid-template-columns: 1fr;
  gap: var(--space-4);
  min-width: 0;
}

@media (min-width: 1024px) {
  .dashboard-charts-top,
  .dashboard-charts-bottom {
    grid-template-columns: repeat(3, 1fr);
  }
}

.chart-cell {
  min-width: 0;
}

/* Panel row: stacks full-width below the chart grid */
.dashboard-panels {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  min-width: 0;
}
</style>
