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
    <!-- Summary row -->
    <section v-if="summaryWidgets.length" class="dashboard-summary" aria-label="Summary statistics">
      <component
        :is="widget.component"
        v-for="widget in summaryWidgets"
        :key="widget.id"
        :project="project"
      />
    </section>

    <!-- Charts + Panel row -->
    <div class="dashboard-main">
      <section class="dashboard-charts" aria-label="Charts">
        <component
          :is="widget.component"
          v-for="widget in chartWidgets"
          :key="widget.id"
          :project="project"
        />
      </section>

      <section v-if="panelWidgets.length" class="dashboard-panels" aria-label="Panels">
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
}

/* Summary row: auto-fit cards */
.dashboard-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: var(--space-3);
}

/* Main two-column layout at ≥ 1024 px */
.dashboard-main {
  display: grid;
  grid-template-columns: 1fr;
  gap: var(--space-4);
}

@media (min-width: 1024px) {
  .dashboard-main {
    grid-template-columns: 1fr 360px;
  }
}

.dashboard-charts {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  min-width: 0;
}

.dashboard-panels {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  min-width: 0;
}
</style>
