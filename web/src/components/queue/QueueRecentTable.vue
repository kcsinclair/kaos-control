<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useQueueStore } from '@/stores/queue'
import type { QueueJob } from '@/api/queue'

const props = defineProps<{
  projectFilter?: string | null
}>()

const queueStore = useQueueStore()

const jobs = computed(() => {
  const all = queueStore.snapshot.recent
  if (!props.projectFilter) return all
  return all.filter((j) => j.project === props.projectFilter)
})

const emptyMessage = computed(() =>
  props.projectFilter ? `No recent jobs for ${props.projectFilter}` : 'No recent jobs',
)

function formatTime(iso?: string): string {
  if (!iso) return '—'
  const d = new Date(iso)
  return isNaN(d.getTime()) ? '—' : d.toLocaleString()
}

function stateClass(state: QueueJob['state']): string {
  if (state === 'completed') return 'state--completed'
  if (state === 'failed') return 'state--failed'
  if (state === 'skipped') return 'state--skipped'
  if (state === 'cancelled') return 'state--cancelled'
  return ''
}
</script>

<template>
  <section class="recent-section">
    <h3 class="panel-title">Recently finished</h3>
    <div v-if="!jobs.length" class="empty-state">{{ emptyMessage }}</div>
    <div v-else class="table-scroll">
    <table class="queue-table">
      <thead>
        <tr>
          <th>State</th>
          <th>Project</th>
          <th>Artifact</th>
          <th>Agent</th>
          <th>Finished at</th>
          <th>Reason</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="job in jobs" :key="job.id">
          <td>
            <span class="state-badge" :class="stateClass(job.state)">{{ job.state }}</span>
          </td>
          <td>
            <RouterLink
              class="project-link"
              :to="`/p/${encodeURIComponent(job.project)}/agents`"
              :aria-label="`Go to project ${job.project}`"
            >{{ job.project }}</RouterLink>
          </td>
          <td>
            <RouterLink
              class="artifact-link"
              :to="`/p/${encodeURIComponent(job.project)}/artifacts/${job.artifact_path}`"
            >{{ job.artifact_path }}</RouterLink>
          </td>
          <td class="agent-name">{{ job.agent_name }}</td>
          <td class="mono">{{ formatTime(job.finished_at) }}</td>
          <td class="reason">{{ job.reason ?? '—' }}</td>
        </tr>
      </tbody>
    </table>
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
.queue-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-sm);
}
.queue-table th {
  text-align: left;
  padding: var(--space-2) var(--space-3);
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
  border-bottom: 1px solid var(--color-border);
}
.queue-table td {
  padding: var(--space-2) var(--space-3);
  border-bottom: 1px solid var(--color-border);
  color: var(--color-text);
  vertical-align: middle;
}
.queue-table tr:last-child td { border-bottom: none; }
.mono { font-family: monospace; font-size: 12px; }
.agent-name { font-family: monospace; }
.reason { font-size: 12px; color: var(--color-text-muted); font-family: monospace; }
.project-link {
  font-size: var(--text-sm);
  color: var(--color-accent);
  text-decoration: none;
}
.project-link:hover { text-decoration: underline; }
.artifact-link {
  font-family: monospace;
  font-size: 12px;
  color: var(--color-accent);
  text-decoration: none;
  word-break: break-all;
}
.artifact-link:hover { text-decoration: underline; }
.state-badge {
  display: inline-block;
  padding: 1px var(--space-2);
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
}
.state--completed { background: rgba(34, 197, 94, 0.15); color: #166534; }
.state--failed { background: rgba(239, 68, 68, 0.15); color: #991b1b; }
.state--skipped { background: rgba(156, 163, 175, 0.2); color: var(--color-text-muted); }
.state--cancelled { background: rgba(245, 158, 11, 0.15); color: #92400e; }
@media (prefers-color-scheme: dark) {
  .state--completed { background: rgba(34, 197, 94, 0.2); color: #4ade80; }
  .state--failed { background: rgba(239, 68, 68, 0.2); color: #f87171; }
  .state--skipped { background: rgba(156, 163, 175, 0.15); color: #9ca3af; }
  .state--cancelled { background: rgba(245, 158, 11, 0.2); color: #fbbf24; }
}
</style>
