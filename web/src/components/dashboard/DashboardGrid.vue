<script setup lang="ts">
import { computed } from 'vue'
import { widgetList } from './widgetRegistry'

defineProps<{ project: string }>()

const summaryWidgets = computed(() => widgetList.filter((w) => w.slot === 'summary'))
const chartWidgets = computed(() => widgetList.filter((w) => w.slot === 'chart'))
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
      Rows 2 & 3: unified 3-column grid.
        Row 2: Stages Distribution | Status Distribution | Recent Ideas & Defects
        Row 3: Completion Velocity (span 2)          | Recent Activity
    -->
    <section
      v-if="chartWidgets.length"
      class="dashboard-charts"
      aria-label="Charts"
    >
      <div
        v-for="widget in chartWidgets"
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
.dashboard-charts {
  display: grid;
  grid-template-columns: 1fr;
  gap: var(--space-4);
  min-width: 0;
}

@media (min-width: 1024px) {
  .dashboard-charts {
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
