<script setup lang="ts">
import { useRoute } from 'vue-router'

const route = useRoute()
const section = (route.path.split('/').pop() ?? '').replace('-', ' ')
</script>

<template>
  <div class="placeholder-view">
    <p class="muted">{{ section }} — coming soon.</p>
  </div>
</template>

<style scoped>
.placeholder-view {
  padding: var(--space-8);
}
.muted {
  color: var(--color-text-muted);
  font-size: var(--text-sm);
  text-transform: capitalize;
}
</style>
