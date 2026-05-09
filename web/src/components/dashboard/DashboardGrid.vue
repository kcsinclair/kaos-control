<script setup lang="ts">
import { computed } from 'vue'
import { widgetList } from './widgetRegistry'

defineProps<{ project: string }>()

const summaryWidgets = computed(() => widgetList.filter((w) => w.slot === 'summary'))
const chartWidgets = computed(() => widgetList.filter((w) => w.slot === 'chart'))
const panelWidgets = computed(() => widgetList.filter((w) => w.slot === 'panel'))

const topRowWidgets = computed(() => chartWidgets.value.slice(0, 3))
const bottomChartWidgets = computed(() => chartWidgets.value.slice(3))
</script>

<template>
  <div class="dashboard-grid">
    <!-- Summary row: four stat cards -->
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

    <!-- Charts column + Panel column -->
    <div class="dashboard-main">
      <section class="dashboard-charts" aria-label="Charts">
        <!-- Top row: first 3 chart widgets in equal-width columns -->
        <div v-if="topRowWidgets.length" class="dashboard-charts-top">
          <component
            :is="widget.component"
            v-for="widget in topRowWidgets"
            :key="widget.id"
            :project="project"
          />
        </div>

        <!-- Remaining chart widgets (velocity-chart etc.) -->
        <div v-if="bottomChartWidgets.length" class="dashboard-charts-bottom">
          <component
            :is="widget.component"
            v-for="widget in bottomChartWidgets"
            :key="widget.id"
            :project="project"
            class="velocity-widget"
          />
        </div>
      </section>

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

/* Summary row: auto-fit cards, min 150 px each */
.dashboard-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: var(--space-3);
}

/* Main area: single column by default (mobile) */
.dashboard-main {
  display: grid;
  grid-template-columns: 1fr;
  gap: var(--space-4);
  min-width: 0;
}

/* Two-column at ≥ 1024 px: charts 2/3, panel 1/3 */
@media (min-width: 1024px) {
  .dashboard-main {
    grid-template-columns: 2fr 1fr;
    align-items: start;
  }
}

.dashboard-charts {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  min-width: 0;
}

/* Top row: 3-column grid for the first three chart widgets */
.dashboard-charts-top {
  display: grid;
  grid-template-columns: 1fr;
  gap: var(--space-4);
}

@media (min-width: 1024px) {
  .dashboard-charts-top {
    grid-template-columns: repeat(3, 1fr);
  }
}

/* Bottom row: velocity spans two-thirds, using a 3-column sub-grid */
.dashboard-charts-bottom {
  display: grid;
  grid-template-columns: 1fr;
  gap: var(--space-4);
}

@media (min-width: 1024px) {
  .dashboard-charts-bottom {
    grid-template-columns: repeat(3, 1fr);
  }

  .dashboard-charts-bottom .velocity-widget {
    grid-column: span 2;
  }
}

.dashboard-panels {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  min-width: 0;
}
</style>
