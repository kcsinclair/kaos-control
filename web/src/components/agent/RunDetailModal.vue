<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import * as agentsApi from '@/api/agents'
import type { AgentRunRow, RunResult } from '@/types/api'
import RunSummaryCard from './RunSummaryCard.vue'
import RawLogModal from './RawLogModal.vue'

const props = defineProps<{
  project: string
  runId: string
}>()

const emit = defineEmits<{ close: [] }>()

const run = ref<AgentRunRow | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)

const runResult = ref<RunResult | null>(null)
const resultLoading = ref(false)

const TERMINAL_RUN_STATUSES = new Set(['done', 'failed', 'killed', 'killed-timeout'])
const showRawLog = ref(false)

// Focus management: save element that had focus before the modal opened.
let previousFocus: HTMLElement | null = null

onMounted(async () => {
  previousFocus = document.activeElement as HTMLElement | null
  try {
    const data = await agentsApi.getRun(props.project, props.runId)
    run.value = data.run
    if (data.run && TERMINAL_RUN_STATUSES.has(data.run.status)) {
      resultLoading.value = true
      const { result } = await agentsApi.getRunResult(props.project, props.runId)
      runResult.value = result
      resultLoading.value = false
    }
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to load run'
  } finally {
    loading.value = false
  }
})

onBeforeUnmount(() => {
  previousFocus?.focus()
})

function formatDatetime(iso: string | undefined): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleString()
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    emit('close')
    return
  }
  // Focus trap: keep Tab/Shift+Tab inside the modal panel.
  if (e.key === 'Tab') {
    const panel = (e.currentTarget as HTMLElement).querySelector<HTMLElement>('.rdm-panel')
    if (!panel) return
    const focusable = Array.from(
      panel.querySelectorAll<HTMLElement>(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
      ),
    ).filter((el) => !el.hasAttribute('disabled'))
    if (!focusable.length) { e.preventDefault(); return }
    const first = focusable[0]
    const last = focusable[focusable.length - 1]
    if (e.shiftKey) {
      if (document.activeElement === first) { e.preventDefault(); last.focus() }
    } else {
      if (document.activeElement === last) { e.preventDefault(); first.focus() }
    }
  }
}

function handleOverlayClick(e: MouseEvent) {
  if ((e.target as HTMLElement).classList.contains('rdm-overlay')) emit('close')
}
</script>

<template>
  <Teleport to="body">
    <div
      class="rdm-overlay"
      role="dialog"
      aria-modal="true"
      aria-label="Run details"
      tabindex="-1"
      @click="handleOverlayClick"
      @keydown="handleKeydown"
    >
      <div class="rdm-panel">
        <div class="rdm-header">
          <h3 class="rdm-title">Run Details</h3>
          <button class="rdm-close" aria-label="Close" @click="emit('close')">✕</button>
        </div>

        <div v-if="loading" class="rdm-state">Loading…</div>
        <div v-else-if="error" class="rdm-state rdm-state--error">{{ error }}</div>

        <div v-else-if="run" class="rdm-body">
          <!-- Run ID -->
          <div class="rdm-field">
            <div class="rdm-field-label">Run ID</div>
            <div class="rdm-field-value rdm-mono">{{ run.run_id }}</div>
          </div>

          <!-- Agent / Role -->
          <div class="rdm-row">
            <div class="rdm-field">
              <div class="rdm-field-label">Agent</div>
              <div class="rdm-field-value">{{ run.agent_name }}</div>
            </div>
            <div class="rdm-field">
              <div class="rdm-field-label">Role</div>
              <div class="rdm-field-value">{{ run.role || '—' }}</div>
            </div>
          </div>

          <!-- Target path -->
          <div class="rdm-field">
            <div class="rdm-field-label">Target path</div>
            <div class="rdm-field-value rdm-mono">{{ run.target_path }}</div>
          </div>

          <!-- Timestamps -->
          <div class="rdm-row">
            <div class="rdm-field">
              <div class="rdm-field-label">Started at</div>
              <div class="rdm-field-value">{{ formatDatetime(run.started_at) }}</div>
            </div>
            <div class="rdm-field">
              <div class="rdm-field-label">Finished at</div>
              <div class="rdm-field-value">{{ formatDatetime(run.finished_at) }}</div>
            </div>
          </div>

          <!-- Status / Exit code -->
          <div class="rdm-row">
            <div class="rdm-field">
              <div class="rdm-field-label">Status</div>
              <div class="rdm-field-value">
                <span class="status-chip" :data-status="run.status" :aria-label="`Status: ${run.status}`">
                  {{ run.status }}
                </span>
              </div>
            </div>
            <div class="rdm-field">
              <div class="rdm-field-label">Exit code</div>
              <div class="rdm-field-value">{{ run.exit_code != null ? run.exit_code : '—' }}</div>
            </div>
          </div>

          <!-- Run summary card (terminal runs only) -->
          <div v-if="TERMINAL_RUN_STATUSES.has(run.status)">
            <div v-if="resultLoading" class="rdm-state">Loading summary…</div>
            <RunSummaryCard
              v-else
              :result="runResult"
              :driver-available="true"
            />
          </div>

          <!-- Stderr tail -->
          <div class="rdm-field" v-if="run.stderr_tail">
            <div class="rdm-field-label">Stderr tail</div>
            <pre class="rdm-log rdm-log--err">{{ run.stderr_tail }}</pre>
          </div>

          <!-- Artifacts produced -->
          <div class="rdm-field" v-if="run.artifacts_produced?.length">
            <div class="rdm-field-label">Artifacts produced</div>
            <ul class="rdm-artifacts">
              <li v-for="p in run.artifacts_produced" :key="p" class="rdm-artifact-path rdm-mono">{{ p }}</li>
            </ul>
          </div>

          <div
            v-if="!run.stderr_tail && !run.artifacts_produced?.length"
            class="rdm-state"
          >No output recorded.</div>

          <!-- View Full Log button -->
          <div class="rdm-log-action">
            <button class="rdm-btn-log" @click="showRawLog = true">View Full Log</button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>

  <!-- Raw log modal — uses its own Teleport internally -->
  <RawLogModal
    v-if="showRawLog"
    :project="project"
    :run-id="runId"
    @close="showRawLog = false"
  />
</template>

<style scoped>
.rdm-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 300;
  padding: var(--space-6);
}
.rdm-panel {
  position: relative;
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  width: 100%;
  max-width: 600px;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.rdm-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-5) var(--space-6) var(--space-3);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.rdm-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}
.rdm-close {
  background: none;
  border: none;
  font-size: var(--text-lg);
  color: var(--color-text-muted);
  cursor: pointer;
  line-height: 1;
  padding: var(--space-1);
}
.rdm-close:hover { color: var(--color-text); }
.rdm-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-5) var(--space-6);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
.rdm-state {
  padding: var(--space-6);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.rdm-state--error { color: #dc2626; }
.rdm-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-4);
}
.rdm-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.rdm-field-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
}
.rdm-field-value {
  font-size: var(--text-sm);
  color: var(--color-text);
}
.rdm-mono {
  font-family: monospace;
  font-size: 12px;
  word-break: break-all;
}
.rdm-log {
  font-family: monospace;
  font-size: 12px;
  background: #0f172a;
  color: #e2e8f0;
  padding: var(--space-3);
  border-radius: var(--radius-sm);
  overflow-x: auto;
  max-height: 200px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
}
.rdm-log--err { color: #fca5a5; }
.rdm-artifacts {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.rdm-artifact-path {
  font-size: 12px;
  color: var(--color-text);
}

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

.rdm-log-action {
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-2);
}
.rdm-btn-log {
  font-size: var(--text-sm);
  font-weight: 500;
  padding: var(--space-1) var(--space-3);
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border);
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
}
.rdm-btn-log:hover {
  background: var(--color-border);
}
</style>
