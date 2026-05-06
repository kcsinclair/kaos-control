<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import DashboardGrid from '@/components/dashboard/DashboardGrid.vue'

const route = useRoute()
const project = computed(() => route.params.project as string)
</script>

<template>
  <div class="dashboard-view">
    <header class="dashboard-header">
      <h2 class="dashboard-title">Dashboard</h2>
    </header>
    <DashboardGrid :project="project" />
  </div>
</template>

<style scoped>
.dashboard-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: var(--space-6) var(--space-4);
  box-sizing: border-box;
  overflow-y: auto;
}

.dashboard-header {
  margin-bottom: var(--space-4);
}

.dashboard-title {
  font-size: var(--text-xl);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}
</style>
