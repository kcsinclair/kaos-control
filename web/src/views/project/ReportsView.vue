<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useReportsStore } from '@/stores/reports'

const route = useRoute()
const reportsStore = useReportsStore()
const project = route.params.project as string

onMounted(() => {
  if (!reportsStore.report) {
    void reportsStore.fetch(project)
  }
})
</script>

<template>
  <div class="reports-view">
    <h1 class="reports-title">Reports</h1>
    <p v-if="reportsStore.loading" class="state-msg">Loading…</p>
    <p v-else-if="reportsStore.error" class="state-msg state-msg--error">{{ reportsStore.error }}</p>
  </div>
</template>

<style scoped>
.reports-view {
  padding: var(--space-6);
  max-width: 1400px;
  margin: 0 auto;
}
.reports-title {
  font-size: var(--text-xl);
  font-weight: 700;
  color: var(--color-text);
  margin: 0 0 var(--space-4);
}
.state-msg {
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.state-msg--error {
  color: var(--color-error);
}
</style>
