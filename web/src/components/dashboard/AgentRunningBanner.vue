<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useQueueStore } from '@/stores/queue'
import { useNow } from '@/composables/useNow'

const props = defineProps<{ project: string }>()

const queueStore = useQueueStore()
const now = useNow()

const runningJob = computed(() => {
  const running = queueStore.snapshot.running
  if (!running || running.project !== props.project) return null
  return running
})

const pendingCount = computed(
  () => queueStore.snapshot.pending.filter((j) => j.project === props.project).length,
)

const elapsedLabel = computed(() => {
  if (!runningJob.value?.started_at) return '…'
  const startMs = new Date(runningJob.value.started_at).getTime()
  if (isNaN(startMs)) return '—'
  const diffSec = Math.floor((now.value.getTime() - startMs) / 1000)
  if (diffSec < 0) return '0s'
  if (diffSec < 60) return `${diffSec}s`
  const mins = Math.floor(diffSec / 60)
  const secs = diffSec % 60
  if (mins < 60) return `${mins}m ${secs}s`
  const hrs = Math.floor(mins / 60)
  const rem = mins % 60
  return rem > 0 ? `${hrs}h ${rem}m` : `${hrs}h`
})
</script>

<template>
  <div v-if="runningJob" class="agent-running-banner" role="status" aria-live="polite">
    <span class="agent-dot" aria-hidden="true"></span>
    <span class="banner-text">
      <strong class="agent-name">{{ runningJob.agent_name }}</strong>
      running on
      <RouterLink
        class="artifact-link"
        :to="`/p/${encodeURIComponent(runningJob.project)}/artifacts/${runningJob.artifact_path}`"
      >{{ runningJob.artifact_path }}</RouterLink>
      <span class="elapsed">({{ elapsedLabel }})</span>
    </span>
    <span v-if="pendingCount > 0" class="queue-badge" :aria-label="`${pendingCount} more jobs queued`">
      +{{ pendingCount }} queued
    </span>
  </div>
</template>

<style scoped>
.agent-running-banner {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-4);
  background: rgba(34, 197, 94, 0.08);
  border: 1px solid #22c55e;
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
}

@media (prefers-color-scheme: dark) {
  .agent-running-banner {
    background: rgba(34, 197, 94, 0.12);
    border-color: #16a34a;
  }
}

/* Pulsing green dot */
.agent-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #22c55e;
  flex-shrink: 0;
  animation: pulse 1.8s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50%       { opacity: 0.5; transform: scale(0.75); }
}

.banner-text {
  color: var(--color-text);
  display: flex;
  align-items: center;
  gap: var(--space-1);
  flex-wrap: wrap;
  min-width: 0;
}

.agent-name {
  font-family: monospace;
  font-weight: 600;
  color: #16a34a;
}

@media (prefers-color-scheme: dark) {
  .agent-name { color: #4ade80; }
}

.artifact-link {
  font-family: monospace;
  color: var(--color-accent);
  text-decoration: none;
  word-break: break-all;
}
.artifact-link:hover { text-decoration: underline; }

.elapsed {
  font-family: monospace;
  color: var(--color-text-muted);
}

.queue-badge {
  margin-left: auto;
  flex-shrink: 0;
  font-size: var(--text-xs);
  font-weight: 500;
  color: var(--color-text-muted);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  padding: 2px var(--space-2);
  white-space: nowrap;
}
</style>
