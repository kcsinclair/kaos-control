<script setup lang="ts">
import { useProjectStore } from '@/stores/project'

const projectStore = useProjectStore()
</script>

<template>
  <div class="placeholder-view">
    <h2>{{ projectStore.current?.name ?? 'Project' }}</h2>
    <p class="muted">Artifact list coming in M2.</p>
  </div>
</template>

<style scoped>
.placeholder-view {
  padding: var(--space-8);
}
.placeholder-view h2 {
  font-size: var(--text-xl);
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 var(--space-2);
}
.muted {
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
</style>
