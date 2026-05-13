<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useQueueStore } from '@/stores/queue'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'

const props = defineProps<{
  projectFilter?: string | null
}>()

const queueStore = useQueueStore()
const authStore = useAuthStore()
const ui = useUiStore()

const jobs = computed(() => {
  const all = queueStore.snapshot.pending
  if (!props.projectFilter) return all
  return all.filter((j) => j.project === props.projectFilter)
})

const emptyMessage = computed(() =>
  props.projectFilter ? `No pending jobs for ${props.projectFilter}` : 'Queue is empty',
)

function formatTime(iso: string): string {
  const d = new Date(iso)
  return isNaN(d.getTime()) ? '—' : d.toLocaleString()
}

function canRemove(enqueuedBy: string, project: string): boolean {
  // product-owner can remove any; otherwise must have a role for the agent
  // (simplified client-side check: if the user enqueued it, they can cancel it)
  const roles = Object.values(authStore.me?.roles ?? {}).flat()
  if (roles.includes('product-owner')) return true
  return authStore.me?.email === enqueuedBy ||
    (authStore.rolesForProject(project).length > 0)
}

async function handleRemove(id: string) {
  try {
    await queueStore.cancel(id)
    ui.success('Removed from queue')
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to remove')
  }
}
</script>

<template>
  <section class="pending-section">
    <h3 class="panel-title">Pending</h3>
    <div v-if="!jobs.length" class="empty-state">{{ emptyMessage }}</div>
    <table v-else class="queue-table">
      <thead>
        <tr>
          <th>#</th>
          <th>Project</th>
          <th>Artifact</th>
          <th>Agent</th>
          <th>Enqueued at</th>
          <th>Enqueued by</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="job in jobs" :key="job.id">
          <td class="pos">{{ job.position }}</td>
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
          <td class="mono">{{ formatTime(job.enqueued_at) }}</td>
          <td class="mono">{{ job.enqueued_by }}</td>
          <td>
            <button
              v-if="canRemove(job.enqueued_by, job.project)"
              class="btn-remove"
              @click="handleRemove(job.id)"
            >Remove</button>
          </td>
        </tr>
      </tbody>
    </table>
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
.pos {
  font-weight: 600;
  width: 2rem;
  text-align: center;
  color: var(--color-text-muted);
}
.mono { font-family: monospace; font-size: 12px; }
.agent-name { font-family: monospace; }
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
.btn-remove {
  padding: 2px var(--space-2);
  background: none;
  border: 1px solid #ef4444;
  border-radius: var(--radius-sm);
  font-size: 11px;
  color: #ef4444;
  cursor: pointer;
}
.btn-remove:hover { background: rgba(239, 68, 68, 0.1); }
</style>
