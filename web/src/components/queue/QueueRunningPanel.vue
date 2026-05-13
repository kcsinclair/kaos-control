<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useQueueStore } from '@/stores/queue'
import { useNow } from '@/composables/useNow'

const props = defineProps<{
  projectFilter?: string | null
}>()

const queueStore = useQueueStore()
const now = useNow()

const job = computed(() => {
  const running = queueStore.snapshot.running
  if (!running) return null
  if (props.projectFilter && running.project !== props.projectFilter) return null
  return running
})

const elapsedLabel = computed(() => {
  if (!job.value?.started_at) return '…'
  const startMs = new Date(job.value.started_at).getTime()
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

const startedAtLabel = computed(() => {
  if (!job.value?.started_at) return '—'
  const d = new Date(job.value.started_at)
  return isNaN(d.getTime()) ? '—' : d.toLocaleString()
})
</script>

<template>
  <section class="running-panel">
    <h3 class="panel-title">Running</h3>
    <div v-if="!job" class="empty-state">Nothing running</div>
    <div v-else class="running-row">
      <div class="running-field">
        <span class="field-label">Agent</span>
        <span class="field-value agent-name">{{ job.agent_name }}</span>
      </div>
      <div class="running-field">
        <span class="field-label">Project</span>
        <RouterLink
          class="field-link"
          :to="`/p/${encodeURIComponent(job.project)}/dashboard`"
          :aria-label="`Go to project ${job.project}`"
        >{{ job.project }}</RouterLink>
      </div>
      <div class="running-field">
        <span class="field-label">Artifact</span>
        <RouterLink
          class="field-link"
          :to="`/p/${encodeURIComponent(job.project)}/artifacts/${job.artifact_path}`"
        >{{ job.artifact_path }}</RouterLink>
      </div>
      <div class="running-field">
        <span class="field-label">Started</span>
        <span class="field-value">{{ startedAtLabel }}</span>
      </div>
      <div class="running-field">
        <span class="field-label">Elapsed</span>
        <span class="field-value elapsed">{{ elapsedLabel }}</span>
      </div>
    </div>
  </section>
</template>

<style scoped>
.panel-title {
  font-size: var(--text-base);
  font-weight: 600;
  margin: 0 0 var(--space-3);
  color: var(--color-text);
}
.empty-state {
  padding: var(--space-4) var(--space-3);
  color: var(--color-text-muted);
  font-size: var(--text-sm);
  border: 1px dashed var(--color-border);
  border-radius: var(--radius-md);
  text-align: center;
}
.running-row {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-4);
  padding: var(--space-4);
  border: 1px solid #22c55e;
  border-radius: var(--radius-md);
  background: rgba(34, 197, 94, 0.06);
}
.running-field {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 120px;
}
.field-label {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
}
.field-value {
  font-size: var(--text-sm);
  color: var(--color-text);
}
.agent-name {
  font-family: monospace;
  font-weight: 500;
}
.elapsed {
  font-family: monospace;
  font-weight: 600;
  color: #22c55e;
}
.field-link {
  font-size: var(--text-sm);
  font-family: monospace;
  color: var(--color-accent);
  text-decoration: none;
  word-break: break-all;
}
.field-link:hover { text-decoration: underline; }
</style>
