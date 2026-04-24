<script setup lang="ts">
import { useAgentsStore } from '@/stores/agents'
import { useRouter } from 'vue-router'

const props = defineProps<{ project: string }>()
const store = useAgentsStore()
const router = useRouter()
</script>

<template>
  <Teleport to="body">
    <button
      v-if="store.activeRuns.length"
      class="run-chip"
      :title="`${store.activeRuns.length} agent run(s) in progress`"
      @click="router.push(`/p/${project}/agents`)"
    >
      <span class="run-dot" />
      {{ store.activeRuns.length }} running
    </button>
  </Teleport>
</template>

<style scoped>
.run-chip {
  position: fixed;
  bottom: var(--space-6);
  right: var(--space-6);
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-4);
  background: #1e293b;
  color: #f1f5f9;
  border: 1px solid #334155;
  border-radius: 99px;
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
  box-shadow: 0 4px 12px rgba(0,0,0,0.3);
  z-index: 100;
  font-family: inherit;
}
.run-chip:hover { background: #0f172a; }
.run-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #22c55e;
  animation: pulse 1.5s ease-in-out infinite;
}
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
</style>
