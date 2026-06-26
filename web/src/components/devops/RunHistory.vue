<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed } from 'vue'
import { onMounted } from 'vue'
import { ChevronDown, ChevronRight, CheckCircle, XCircle, MinusCircle, Loader } from 'lucide-vue-next'
import { useDevOpsStore } from '@/stores/devops'
import { useNow } from '@/composables/useNow'
import { formatRelativeTime, formatDurationMs } from '@/composables/useRunFormatters'
import type { RunHistoryRow } from '@/api/devops'

const props = defineProps<{
  pipelineSlug: string
  project: string
}>()

const devops = useDevOpsStore()
const now = useNow()
const collapsed = ref(true)

const rows = computed((): RunHistoryRow[] => devops.pipelineHistory.get(props.pipelineSlug) ?? [])
const isLoading = computed(() => devops.historyLoading.get(props.pipelineSlug) ?? false)
const loadError = computed(() => devops.historyError.get(props.pipelineSlug) ?? null)

onMounted(() => {
  devops.fetchPipelineHistory(props.project, props.pipelineSlug)
})

function toggle() {
  collapsed.value = !collapsed.value
}

function absoluteTime(iso: string): string {
  return new Date(iso).toLocaleString()
}
</script>

<template>
  <div class="run-history">
    <button class="history-toggle" @click="toggle" aria-label="Toggle run history">
      <component :is="collapsed ? ChevronRight : ChevronDown" :size="12" class="toggle-icon" />
      <span class="history-header-label">Run history</span>
      <span v-if="isLoading" class="history-loading"><Loader :size="10" class="spin" /></span>
    </button>

    <template v-if="!collapsed">
      <div v-if="loadError" class="history-error">{{ loadError }}</div>
      <div v-else-if="rows.length === 0 && !isLoading" class="history-empty">No runs yet</div>
      <div
        v-for="row in rows"
        :key="row.run_id"
        class="history-row"
      >
        <span class="history-status" :class="`history-status--${row.status}`" :title="row.status">
          <CheckCircle v-if="row.status === 'passed'" :size="13" />
          <XCircle v-else-if="row.status === 'failed'" :size="13" />
          <MinusCircle v-else-if="row.status === 'cancelled'" :size="13" />
          <Loader v-else :size="13" class="spin" />
        </span>
        <span class="history-time" :title="absoluteTime(row.started_at)">
          {{ formatRelativeTime(row.started_at, now) }}
        </span>
        <span class="history-duration">{{ formatDurationMs(row.duration_ms) }}</span>
      </div>
    </template>
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
.history-toggle {
  display: flex;
  align-items: center;
  gap: 4px;
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
  color: var(--color-text-muted);
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  width: 100%;
  text-align: left;
  margin-bottom: var(--space-1);
}
.history-toggle:hover {
  color: var(--color-text);
}
.toggle-icon {
  flex-shrink: 0;
}
.history-header-label {
  flex: 1;
}
.history-loading {
  margin-left: auto;
  color: var(--color-text-muted);
  display: flex;
  align-items: center;
}
.history-error {
  font-size: 11px;
  color: var(--color-error);
  padding: 2px 0;
}
.history-empty {
  font-size: 11px;
  color: var(--color-text-muted);
  font-style: italic;
  padding: 2px 0;
}
.history-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: 11px;
  padding: 1px 0;
}
.history-status {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}
.history-status--passed {
  color: #22c55e;
}
.history-status--failed {
  color: var(--color-error);
}
.history-status--cancelled {
  color: var(--color-text-muted);
}
.history-status--running {
  color: var(--color-accent);
}
.history-time {
  color: var(--color-text-muted);
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.history-duration {
  color: var(--color-text-muted);
  flex-shrink: 0;
  font-variant-numeric: tabular-nums;
}
@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
.spin {
  animation: spin 1s linear infinite;
}
</style>
