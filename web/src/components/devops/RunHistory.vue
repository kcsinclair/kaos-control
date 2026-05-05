<script setup lang="ts">
import { computed } from 'vue'
import { useDevOpsStore } from '@/stores/devops'
import type { RunHistoryEntry } from '@/stores/devops'

const props = defineProps<{
  pipelineSlug: string
}>()

const emit = defineEmits<{
  (e: 'view-log', entry: RunHistoryEntry): void
}>()

const devops = useDevOpsStore()

const history = computed(() =>
  devops.historyForPipeline(props.pipelineSlug).slice().reverse()
)

function formatTime(ts: number): string {
  return new Date(ts).toLocaleTimeString()
}

function formatDuration(entry: RunHistoryEntry): string | null {
  if (!entry.completedAt) return null
  const ms = entry.completedAt - entry.startedAt
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}
</script>

<template>
  <div v-if="history.length > 0" class="run-history">
    <div class="history-header">Recent runs</div>
    <div
      v-for="entry in history"
      :key="entry.runId"
      class="history-row"
    >
      <span class="history-status" :class="`history-status--${entry.overallStatus}`">
        {{ entry.overallStatus }}
      </span>
      <span class="history-time">{{ formatTime(entry.startedAt) }}</span>
      <span v-if="formatDuration(entry)" class="history-duration">{{ formatDuration(entry) }}</span>
      <button class="history-log-btn" @click="emit('view-log', entry)">log</button>
    </div>
  </div>
</template>

<style scoped>
.run-history {
  border-top: 1px solid var(--color-border);
  padding-top: var(--space-2);
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.history-header {
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--color-text-muted);
  margin-bottom: var(--space-1);
}
.history-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: 11px;
}
.history-status {
  font-weight: 600;
  font-size: 10px;
  padding: 1px 5px;
  border-radius: 99px;
}
.history-status--running {
  background: var(--badge-approved-bg);
  color: var(--badge-approved-text);
}
.history-status--passed {
  background: var(--badge-done-bg);
  color: var(--badge-done-text);
}
.history-status--failed {
  background: var(--badge-blocked-bg);
  color: var(--badge-blocked-text);
}
.history-status--cancelled {
  background: var(--color-border);
  color: var(--color-text-muted);
}
.history-time {
  color: var(--color-text-muted);
}
.history-duration {
  color: var(--color-text-muted);
}
.history-log-btn {
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: 10px;
  color: var(--color-accent);
  cursor: pointer;
  padding: 0 var(--space-1);
  margin-left: auto;
}
.history-log-btn:hover { background: var(--color-surface); }
</style>
