<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAgentsStore } from '@/stores/agents'

const props = defineProps<{
  project: string
  targetPath: string
}>()

const emit = defineEmits<{ 'select-run': [runId: string] }>()

const store = useAgentsStore()

const runs = computed(() => store.artifactRuns)

// Relative time for recent runs (< 24 hours), absolute for older ones.
function formatTime(iso: string): string {
  const d = new Date(iso)
  const diffMs = Date.now() - d.getTime()
  const diffSecs = Math.round(diffMs / 1000)
  if (diffSecs < 60) return `${diffSecs}s ago`
  const diffMins = Math.floor(diffSecs / 60)
  if (diffMins < 60) return `${diffMins}m ago`
  const diffHours = Math.floor(diffMins / 60)
  if (diffHours < 24) return `${diffHours}h ago`
  return d.toLocaleString()
}

onMounted(() => {
  store.fetchRunsByTargetPath(props.project, props.targetPath)
})
</script>

<template>
  <details class="arh-details">
    <summary class="arh-summary">
      <span class="arh-label">Agent Runs</span>
      <span class="arh-count">{{ runs.length }}</span>
    </summary>

    <div class="arh-body">
      <div v-if="!runs.length" class="arh-empty">No agent runs for this artifact.</div>
      <ul v-else class="arh-list" role="list">
        <li
          v-for="run in runs"
          :key="run.run_id"
          class="arh-row"
          role="button"
          tabindex="0"
          @click="emit('select-run', run.run_id)"
          @keydown.enter="emit('select-run', run.run_id)"
          @keydown.space.prevent="emit('select-run', run.run_id)"
        >
          <span class="arh-run-id" aria-label="Run ID">{{ run.run_id.slice(0, 8) }}</span>
          <span class="arh-agent">{{ run.agent_name }}</span>
          <span class="arh-time cell-muted">{{ formatTime(run.started_at) }}</span>
          <span
            class="arh-status status-chip"
            :data-status="run.status"
            :aria-label="`Status: ${run.status}`"
          >{{ run.status }}</span>
        </li>
      </ul>
    </div>
  </details>
</template>

<style scoped>
.arh-details {
  border-top: 1px solid var(--color-border);
  padding: var(--space-3) var(--space-6);
}
.arh-summary {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  cursor: pointer;
  list-style: none;
  user-select: none;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--color-text-muted);
  padding: var(--space-1) 0;
}
.arh-summary::-webkit-details-marker { display: none; }
.arh-summary::before {
  content: '▶';
  font-size: 8px;
  transition: transform 0.15s;
}
details[open] .arh-summary::before { transform: rotate(90deg); }
.arh-label { flex: 1; }
.arh-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 99px;
  background: var(--color-border);
  color: var(--color-text-muted);
  font-size: 11px;
  font-weight: 600;
}
.arh-body {
  padding-top: var(--space-2);
}
.arh-empty {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  padding: var(--space-2) 0;
}
.arh-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.arh-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-2);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: var(--text-sm);
}
.arh-row:hover { background: var(--color-surface); }
.arh-row:focus-visible { outline: 2px solid var(--color-accent); outline-offset: 1px; }
.arh-run-id {
  font-family: monospace;
  font-size: 12px;
  color: var(--color-text-muted);
  flex-shrink: 0;
  width: 5.5em;
}
.arh-agent {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--color-text);
  font-size: var(--text-sm);
}
.arh-time {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--color-text-muted);
}
.arh-status { flex-shrink: 0; }

/* Status chip — matches AgentsRunsView */
.status-chip {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 500;
  background: var(--color-border);
  color: var(--color-text);
}
.status-chip[data-status="running"]        { background: var(--badge-approved-bg);     color: var(--badge-approved-text); }
.status-chip[data-status="done"]           { background: var(--badge-done-bg);          color: var(--badge-done-text); }
.status-chip[data-status="failed"]         { background: var(--badge-blocked-bg);       color: var(--badge-blocked-text); }
.status-chip[data-status="killed"]         { background: var(--badge-blocked-bg);       color: var(--badge-blocked-text); }
.status-chip[data-status="killed-timeout"] { background: var(--badge-in-progress-bg);  color: var(--badge-in-progress-text); }
</style>
