<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed } from 'vue'
import { onMounted } from 'vue'
import { ChevronDown, ChevronRight, CheckCircle, XCircle, MinusCircle, Loader, ChevronDown as Expand, ChevronRight as Collapse, RefreshCw } from 'lucide-vue-next'
import { useDevOpsStore } from '@/stores/devops'
import * as devopsApi from '@/api/devops'
import { useNow } from '@/composables/useNow'
import { formatRelativeTime, formatDurationMs } from '@/composables/useRunFormatters'
import type { RunHistoryRow } from '@/api/devops'
import type { LogLine } from '@/stores/devops'

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

// ── Inline log expansion ──────────────────────────────────────────────────────

const expandedRunId = ref<string | null>(null)
const logLines = ref<LogLine[]>([])
const logLoading = ref(false)
const logError = ref<string | null>(null)

async function fetchLog(runId: string): Promise<void> {
  logLoading.value = true
  logError.value = null
  logLines.value = []
  try {
    const raw = await devopsApi.getPipelineRunLog(props.project, props.pipelineSlug, runId)
    logLines.value = devopsApi.parseRunLog(raw)
  } catch (e: unknown) {
    logError.value = e instanceof Error ? e.message : 'Failed to load log'
  } finally {
    logLoading.value = false
  }
}

async function toggleExpand(runId: string): Promise<void> {
  if (expandedRunId.value === runId) {
    expandedRunId.value = null
    logLines.value = []
    logError.value = null
    return
  }
  expandedRunId.value = runId
  await fetchLog(runId)
}

async function retryLog(): Promise<void> {
  if (expandedRunId.value) {
    await fetchLog(expandedRunId.value)
  }
}

onMounted(() => {
  devops.fetchPipelineHistory(props.project, props.pipelineSlug)
})

function toggle() {
  collapsed.value = !collapsed.value
}

function absoluteTime(iso: string): string {
  return new Date(iso).toLocaleString()
}

function formatTs(ts: number): string {
  return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function formatLogDuration(ms: number | undefined): string {
  if (ms == null) return ''
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(2)}s`
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
      <template v-for="row in rows" :key="row.run_id">
        <div class="history-row" :class="{ 'history-row--expanded': expandedRunId === row.run_id }">
          <button
            class="history-expand-btn"
            :aria-label="expandedRunId === row.run_id ? 'Collapse log' : 'Expand log'"
            @click="toggleExpand(row.run_id)"
          >
            <component :is="expandedRunId === row.run_id ? Expand : Collapse" :size="10" />
          </button>
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

        <!-- Inline log pane -->
        <div v-if="expandedRunId === row.run_id" class="history-log-pane">
          <div v-if="logLoading" class="log-state log-state--loading">
            <Loader :size="12" class="spin" /> Loading log…
          </div>
          <div v-else-if="logError" class="log-state log-state--error">
            {{ logError }}
            <button class="log-retry-btn" @click="retryLog">
              <RefreshCw :size="11" /> Retry
            </button>
          </div>
          <div v-else class="log-scroll">
            <div
              v-for="(line, i) in logLines"
              :key="i"
              class="log-row"
              :class="`log-row--${line.kind}`"
            >
              <template v-if="line.kind === 'step-start'">
                <span class="log-row__step-label">{{ line.text }}</span>
                <span class="log-row__ts">{{ formatTs(line.timestamp) }}</span>
              </template>
              <template v-else-if="line.kind === 'step-end'">
                <span class="log-row__step-label">
                  {{ line.text }} —
                  <span :class="line.status === 'passed' ? 'log-row__ok' : 'log-row__fail'">{{ line.status }}</span>
                  <span v-if="line.durationMs != null" class="log-row__dur"> {{ formatLogDuration(line.durationMs) }}</span>
                </span>
              </template>
              <template v-else-if="line.kind === 'run-end'">
                <span class="log-row__terminal">
                  Run
                  <span :class="line.status === 'passed' ? 'log-row__ok' : 'log-row__fail'">{{ line.status }}</span>
                  <span v-if="line.durationMs != null" class="log-row__dur"> {{ formatLogDuration(line.durationMs) }}</span>
                </span>
              </template>
              <template v-else>
                <span class="log-row__text">{{ line.text }}</span>
              </template>
            </div>
          </div>
        </div>
      </template>
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
  border-radius: var(--radius-sm);
}
.history-row--expanded {
  background: var(--color-surface);
}
.history-expand-btn {
  display: flex;
  align-items: center;
  background: none;
  border: none;
  padding: 0 2px;
  cursor: pointer;
  color: var(--color-text-muted);
  flex-shrink: 0;
  line-height: 1;
}
.history-expand-btn:hover {
  color: var(--color-text);
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

/* ── Inline log pane ─────────────────────────────────────────────────────────── */
.history-log-pane {
  background: #0f172a;
  border-radius: var(--radius-sm);
  overflow: hidden;
  margin: 2px 0;
}
.log-state {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  font-size: 11px;
  font-family: monospace;
}
.log-state--loading {
  color: #94a3b8;
}
.log-state--error {
  color: #fca5a5;
}
.log-retry-btn {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  background: #1e293b;
  border: 1px solid #334155;
  color: #93c5fd;
  border-radius: 3px;
  font-size: 10px;
  padding: 2px 6px;
  cursor: pointer;
  margin-left: 6px;
}
.log-retry-btn:hover {
  background: #1d4ed8;
  color: #fff;
}
.log-scroll {
  max-height: 280px;
  overflow-y: auto;
  overflow-x: auto;
}
.log-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 10px;
  min-height: 18px;
  line-height: 18px;
  font-family: monospace;
  font-size: 11px;
  white-space: pre;
  box-sizing: border-box;
  color: #e2e8f0;
}
.log-row--output {
  color: #e2e8f0;
}
.log-row--step-start {
  background: #1e3a5f;
  color: #93c5fd;
  font-weight: 600;
  border-top: 1px solid #1d4ed8;
}
.log-row--step-end {
  background: #16213a;
  color: #7dd3fc;
  border-bottom: 1px solid #1d4ed8;
}
.log-row--run-start {
  background: #0c1a2e;
  color: #475569;
  font-style: italic;
}
.log-row--run-end {
  background: #1a1a1a;
  border-top: 1px solid #334155;
  font-weight: 700;
}
.log-row__step-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.log-row__text {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: pre;
}
.log-row__ts {
  font-size: 10px;
  color: #475569;
  flex-shrink: 0;
}
.log-row__dur {
  font-size: 10px;
  color: #64748b;
  margin-left: 4px;
}
.log-row__ok {
  color: #86efac;
}
.log-row__fail {
  color: #fca5a5;
}
.log-row__terminal {
  color: #fde68a;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
.spin {
  animation: spin 1s linear infinite;
}
</style>
