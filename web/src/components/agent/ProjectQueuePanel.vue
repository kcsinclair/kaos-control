<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { CirclePlay, CircleDashed, CirclePause, X } from 'lucide-vue-next'
import { useQueueStore } from '@/stores/queue'
import { useUiStore } from '@/stores/ui'
import { useNow } from '@/composables/useNow'
import { formatRelativeTime, formatDurationMs } from '@/composables/useRunFormatters'

const props = defineProps<{
  project: string
}>()

const queueStore = useQueueStore()
const ui = useUiStore()
const now = useNow()

const running = computed(() => {
  const r = queueStore.snapshot.running
  return r && r.project === props.project ? r : null
})

const pending = computed(() =>
  queueStore.snapshot.pending
    .filter((j) => j.project === props.project)
    .sort((a, b) => a.position - b.position),
)

const pendingCount = computed(() => pending.value.length)

const elapsedLabel = computed(() => {
  const startedAt = running.value?.started_at
  if (!startedAt) return '—'
  const startMs = new Date(startedAt).getTime()
  if (isNaN(startMs)) return '—'
  return formatDurationMs(now.value.getTime() - startMs)
})

function relativeLabel(iso: string): string {
  return formatRelativeTime(iso, now.value)
}

async function handleCancel(id: string) {
  try {
    await queueStore.cancel(id)
    ui.success('Removed from queue')
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to remove from queue')
  }
}
</script>

<template>
  <aside class="project-queue-panel" aria-label="Project queue">
    <div class="panel-header">
      <h3 class="panel-title">Project Queue</h3>
      <span
        class="pending-badge"
        :aria-label="`${pendingCount} jobs pending for this project`"
      >{{ pendingCount }} pending</span>
    </div>

    <div v-if="queueStore.isPaused" class="paused-note" role="status">
      <CirclePause :size="14" class="paused-icon" aria-hidden="true" />
      <span>Queue is paused.</span>
      <RouterLink class="paused-link" to="/queue">See the global queue for details</RouterLink>
    </div>

    <section class="running-section">
      <div v-if="running" class="running-row">
        <CirclePlay :size="14" class="status-icon status-icon--running" aria-hidden="true" />
        <div class="running-info">
          <RouterLink
            class="artifact-link"
            :to="`/p/${encodeURIComponent(project)}/artifacts/${running.artifact_path}`"
          >{{ running.artifact_path }}</RouterLink>
          <span class="running-meta">
            {{ running.agent_name }}
            <template v-if="running.started_at"> · started {{ relativeLabel(running.started_at) }} · {{ elapsedLabel }} elapsed</template>
          </span>
        </div>
      </div>
      <div v-else class="idle-row">
        <CircleDashed :size="14" class="status-icon status-icon--idle" aria-hidden="true" />
        <span>Nothing running for this project</span>
      </div>
    </section>

    <section v-if="pending.length" class="pending-section">
      <h4 class="section-label">Pending</h4>
      <ul class="pending-list">
        <li v-for="job in pending" :key="job.id" class="pending-row">
          <div class="pending-info">
            <RouterLink
              class="artifact-link"
              :to="`/p/${encodeURIComponent(project)}/artifacts/${job.artifact_path}`"
            >{{ job.artifact_path }}</RouterLink>
            <span class="pending-meta">{{ job.agent_name }} · enqueued {{ relativeLabel(job.enqueued_at) }}</span>
          </div>
          <button
            class="btn-cancel"
            aria-label="cancel queued job"
            @click="handleCancel(job.id)"
          ><X :size="12" aria-hidden="true" /></button>
        </li>
      </ul>
    </section>
    <div v-else-if="!running" class="empty-state">No queued work for this project</div>

    <RouterLink class="global-link" to="/queue">View global queue</RouterLink>
  </aside>
</template>

<style scoped>
.project-queue-panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  font-size: var(--text-sm);
}
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
}
.panel-title {
  font-size: var(--text-base);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}
.pending-badge {
  font-size: 11px;
  font-weight: 600;
  padding: 1px 8px;
  border-radius: 99px;
  background: var(--color-border);
  color: var(--color-text-muted);
  white-space: nowrap;
}
.section-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
  margin: 0 0 var(--space-2);
}
.paused-note {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-error, #dc2626);
  border-radius: var(--radius-sm);
  background: var(--badge-blocked-bg);
  color: var(--badge-blocked-text);
  font-size: var(--text-sm);
}
.paused-icon { flex-shrink: 0; }
.paused-link {
  color: inherit;
  text-decoration: underline;
}
.running-row,
.idle-row {
  display: flex;
  align-items: flex-start;
  gap: var(--space-2);
}
.idle-row {
  align-items: center;
  color: var(--color-text-muted);
}
.status-icon { flex-shrink: 0; margin-top: 2px; }
.status-icon--running { color: #22c55e; }
.status-icon--idle { color: var(--color-text-muted); }
.running-info,
.pending-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.running-meta,
.pending-meta {
  font-size: 12px;
  color: var(--color-text-muted);
}
.artifact-link {
  font-family: monospace;
  font-size: 12px;
  color: var(--color-accent);
  text-decoration: none;
  word-break: break-all;
}
.artifact-link:hover { text-decoration: underline; }
.pending-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.pending-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-2);
}
.btn-cancel {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 20px;
  height: 20px;
  padding: 0;
  margin-top: 1px;
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  color: var(--color-text-muted);
  cursor: pointer;
}
.btn-cancel:hover { border-color: #ef4444; color: #ef4444; }
.empty-state {
  padding: var(--space-3);
  color: var(--color-text-muted);
  font-size: var(--text-sm);
  border: 1px dashed var(--color-border);
  border-radius: var(--radius-md);
  text-align: center;
}
.global-link {
  align-self: flex-start;
  font-size: 12px;
  color: var(--color-accent);
  text-decoration: none;
}
.global-link:hover { text-decoration: underline; }
</style>
