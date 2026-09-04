<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed } from 'vue'
import { useProviderSwitchStore } from '@/stores/providerSwitch'

const props = defineProps<{
  project: string
}>()

const providerSwitchStore = useProviderSwitchStore()

// FR-9.1: only relevant once a secondary is configured for at least one
// agent — with no secondary anywhere there is nothing to fail back to.
const visible = computed(() => providerSwitchStore.hasSecondaryConfigured)

const label = computed(() => {
  switch (providerSwitchStore.currentSide) {
    case 'secondary':
      return 'Secondary Agents'
    case 'partial':
      return 'Partially Paused'
    default:
      return 'Primary Agents'
  }
})

const tooltip = computed(() => {
  switch (providerSwitchStore.currentSide) {
    case 'secondary':
      return `${providerSwitchStore.failoverCount} agent(s) on their secondary provider — click to review failback`
    case 'partial':
      return `${providerSwitchStore.partiallyPausedAgents.length} agent(s) paused with no secondary — click to review`
    default:
      return 'All agents on their primary provider'
  }
})

const failbackPath = computed(() => `/p/${encodeURIComponent(props.project)}/failback`)
</script>

<template>
  <RouterLink
    v-if="visible"
    :to="failbackPath"
    class="switchover-status-button"
    :class="`switchover-status-button--${providerSwitchStore.currentSide}`"
    :aria-label="tooltip"
    :title="tooltip"
  >
    {{ label }}
  </RouterLink>
</template>

<style scoped>
.switchover-status-button {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-3);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  text-decoration: none;
  cursor: pointer;
  font-family: inherit;
  transition: color 0.15s, border-color 0.15s, background 0.15s;
  border: 1px solid;
}
.switchover-status-button--primary {
  border-color: #22c55e;
  color: #22c55e;
  background: rgba(34, 197, 94, 0.12);
}
.switchover-status-button--primary:hover {
  color: #fff;
  border-color: #4ade80;
  background: rgba(34, 197, 94, 0.25);
}
.switchover-status-button--secondary {
  border-color: #f59e0b;
  color: #f59e0b;
  background: rgba(245, 158, 11, 0.12);
}
.switchover-status-button--secondary:hover {
  color: #fff;
  border-color: #fbbf24;
  background: rgba(245, 158, 11, 0.25);
}
.switchover-status-button--partial {
  border-color: #ef4444;
  color: #ef4444;
  background: rgba(239, 68, 68, 0.12);
}
.switchover-status-button--partial:hover {
  color: #fff;
  border-color: #f87171;
  background: rgba(239, 68, 68, 0.25);
}
</style>
