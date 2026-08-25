<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, watch, onMounted, onBeforeUnmount } from 'vue'
import * as agentsApi from '@/api/agents'
import type { AgentRunRow, RunResult } from '@/types/api'
import RunSummaryCard from './RunSummaryCard.vue'
import RunDenialSummary from './RunDenialSummary.vue'
import RunFailureBanner from './RunFailureBanner.vue'
import RawLogModal from './RawLogModal.vue'
import TestRunSummaryCard from './TestRunSummaryCard.vue'
import { useAgentsStore } from '@/stores/agents'
import { parseLogTurns } from '@/lib/logParser'
import type { RunTurn } from '@/types/api'

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

const parsedTurns = ref<RunTurn[]>([])
const expandedTurns = ref<Set<number>>(new Set())
const expandedToolArgs = ref<Set<string>>(new Set())
const expandedToolResults = ref<Set<string>>(new Set())

function toggleTurn(turnNum: number) {
  if (expandedTurns.value.has(turnNum)) {
    expandedTurns.value.delete(turnNum)
  } else {
    expandedTurns.value.add(turnNum)
  }
}

function toggleToolArg(id: string) {
  if (expandedToolArgs.value.has(id)) {
    expandedToolArgs.value.delete(id)
  } else {
    expandedToolArgs.value.add(id)
  }
}

function toggleToolRes(id: string) {
  if (expandedToolResults.value.has(id)) {
    expandedToolResults.value.delete(id)
  } else {
    expandedToolResults.value.add(id)
  }
}

const TERMINAL_RUN_STATUSES = new Set(['done', 'failed', 'killed', 'killed-timeout'])
const showRawLog = ref(false)
const agentsStore = useAgentsStore()

// Files under lifecycle/ are artifacts and link to the editor; other produced
// files (code, config) are shown as plain paths.
function isLifecycleArtifact(path: string): boolean {
  return path.startsWith('lifecycle/') && path.endsWith('.md')
}
function artifactLink(path: string): string {
  return `/p/${encodeURIComponent(props.project)}/artifacts/${path}`
}

// When a run finishes while the modal is open, pick up the result from the store.
watch(
  () => agentsStore.runResults.get(props.runId),
  (newResult) => {
    if (newResult && !runResult.value) {
      runResult.value = newResult
      resultLoading.value = false
    }
  },
)

// Focus management: save element that had focus before the modal opened.
let previousFocus: HTMLElement | null = null

onMounted(async () => {
  previousFocus = document.activeElement as HTMLElement | null
  try {
    const data = await agentsApi.getRun(props.project, props.runId)
    run.value = data.run
    if (data.run && TERMINAL_RUN_STATUSES.has(data.run.status)) {
      // Check store cache first (populated by WS events) before calling API.
      const cached = agentsStore.getRunResult(props.runId)
      if (cached) {
        runResult.value = cached
      } else {
        resultLoading.value = true
        const { result } = await agentsApi.getRunResult(props.project, props.runId)
        runResult.value = result
        resultLoading.value = false
      }
    }

    try {
      const log = await agentsApi.getRunLog(props.project, props.runId)
      if (log) {
        parsedTurns.value = parseLogTurns(log)
        // Auto-expand last 3 turns
        parsedTurns.value.slice(-3).forEach((t) => expandedTurns.value.add(t.turn_number))
      }
    } catch {
      // Non-fatal if log file is not yet created or unreadable
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

function agentHasTokenMetrics(agentName: string): boolean {
  const driver = agentsStore.agents.find((agent) => agent.name === agentName)?.driver
  if (!driver) return true
  return driver === 'claude-code-cli' || driver === 'claude-mediated'
}

function providerNameFor(agentName: string): string | undefined {
  return agentsStore.agents.find((agent) => agent.name === agentName)?.provider
}

// Human labels for the known error_details keys (ClassifyRunError /
// recordPreflightFailure); anything else falls back to its raw key so a
// future backend field still renders instead of being silently dropped.
const DETAIL_LABELS: Record<string, string> = {
  message: 'Error message',
  stderr_excerpt: 'Stderr excerpt',
  observed_permission_mode: 'Observed permission mode',
  provider: 'Provider',
  model: 'Model',
  base_url: 'Endpoint',
  status_code: 'HTTP status',
}

function detailLabel(key: string): string {
  return DETAIL_LABELS[key] ?? key
}

function diagnosticEntries(details: Record<string, unknown> | null | undefined): Array<[string, string]> {
  if (!details) return []
  return Object.entries(details)
    .filter(([, v]) => v !== null && v !== undefined && v !== '')
    .map(([k, v]) => [detailLabel(k), typeof v === 'string' ? v : JSON.stringify(v)])
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

</script>

<template>
  <Teleport to="body">
    <div
      class="rdm-overlay"
      role="dialog"
      aria-modal="true"
      aria-label="Run details"
      tabindex="-1"
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

          <!-- Timestamps & TTFT -->
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

          <!-- Status / Exit code / TTFT -->
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

          <div v-if="run.ttft_ms != null" class="rdm-field">
            <div class="rdm-field-label">Time To First Token (TTFT)</div>
            <div class="rdm-field-value">{{ run.ttft_ms }} ms</div>
          </div>

          <!-- Failure banner with structured diagnostics and remediation -->
          <RunFailureBanner
            v-if="run.status === 'failed' && run.failure_reason"
            :failure-reason="run.failure_reason"
            :observed-mode="run.observed_permission_mode"
            :remediation="run.remediation"
            :error-details="run.error_details"
            :provider-name="providerNameFor(run.agent_name)"
          />
          <!-- Collapsible diagnostic info: whatever error_details the backend sent -->
          <details v-if="diagnosticEntries(run.error_details).length" class="rdm-field rdm-diagnostics">
            <summary class="rdm-diagnostics-summary">Diagnostic Info</summary>
            <dl class="rdm-diagnostics-list">
              <template v-for="[label, value] in diagnosticEntries(run.error_details)" :key="label">
                <dt class="rdm-diagnostics-key">{{ label }}</dt>
                <dd class="rdm-diagnostics-value">{{ value }}</dd>
              </template>
            </dl>
          </details>
          <!-- Denial notice for done runs with denials (on_denial: continue) -->
          <RunFailureBanner
            v-if="run.status === 'done' && run.denied_tool_calls?.length"
            :denial-count="run.denied_tool_calls.length"
          />
          <!-- Denied-calls summary -->
          <RunDenialSummary
            v-if="run.denied_tool_calls?.length"
            :denials="run.denied_tool_calls"
            :observe-only="agentsStore.agents.find(a => a.name === run.agent_name)?.observe_only"
          />

          <!-- Run summary card (terminal runs only) -->
          <div v-if="TERMINAL_RUN_STATUSES.has(run.status)">
            <div v-if="resultLoading" class="rdm-state">Loading summary…</div>
            <RunSummaryCard
              v-else
              :result="runResult"
              :driver-available="agentHasTokenMetrics(run.agent_name)"
            />
          </div>

          <!-- Test-runner run summary -->
          <TestRunSummaryCard
            v-if="run.run_summary"
            :summary="run.run_summary"
          />

          <!-- Multi-turn timeline -->
          <div v-if="parsedTurns.length" class="rdm-field">
            <div class="rdm-field-label">Turn Timeline ({{ parsedTurns.length }} turn{{ parsedTurns.length === 1 ? '' : 's' }})</div>
            <div class="rdm-turns-timeline">
              <div
                v-for="turn in parsedTurns"
                :key="turn.turn_number"
                class="rdm-turn-card"
                :class="{ 'rdm-turn-card--recovered': turn.is_recovered }"
              >
                <div class="rdm-turn-header" @click="toggleTurn(turn.turn_number)">
                  <span class="rdm-turn-badge" :data-role="turn.role">
                    {{ turn.role === 'system' ? 'System Prompt' : turn.role === 'user' ? 'User Prompt' : `Turn ${turn.turn_number}` }}
                  </span>
                  <span v-if="turn.is_recovered" class="rdm-recovery-badge" title="Recovered native tool-call (FR-5a)">
                    ⚡ Recovered Call
                  </span>
                  <span v-if="turn.tool_calls?.length" class="rdm-turn-tool-count">
                    {{ turn.tool_calls.length }} tool call{{ turn.tool_calls.length === 1 ? '' : 's' }}
                  </span>
                  <span class="rdm-turn-toggle">{{ expandedTurns.has(turn.turn_number) ? '▲' : '▼' }}</span>
                </div>

                <div v-if="expandedTurns.has(turn.turn_number)" class="rdm-turn-body">
                  <!-- Assistant reasoning or text -->
                  <div v-if="turn.content" class="rdm-turn-content">
                    <pre class="rdm-turn-text">{{ turn.content }}</pre>
                  </div>

                  <!-- Tool calls in this turn -->
                  <div v-if="turn.tool_calls?.length" class="rdm-tool-calls-list">
                    <div
                      v-for="tc in turn.tool_calls"
                      :key="tc.id"
                      class="rdm-tool-card"
                    >
                      <div class="rdm-tool-header">
                        <span class="rdm-tool-name">{{ tc.name }}</span>
                        <span class="rdm-tool-id">{{ tc.id }}</span>
                        <span v-if="tc.is_recovered" class="rdm-tool-rec-tag" title="Recovered native format (FR-5a)">FR-5a</span>
                      </div>

                      <!-- Arguments -->
                      <div v-if="tc.arguments" class="rdm-tool-block">
                        <div class="rdm-tool-block-header" @click="toggleToolArg(tc.id)">
                          <span>Arguments</span>
                          <span class="rdm-tool-toggle">{{ expandedToolArgs.has(tc.id) ? '▲' : '▼' }}</span>
                        </div>
                        <pre v-if="expandedToolArgs.has(tc.id)" class="rdm-code-block">{{ tc.arguments }}</pre>
                      </div>

                      <!-- Result -->
                      <div v-if="tc.result" class="rdm-tool-block">
                        <div class="rdm-tool-block-header" @click="toggleToolRes(tc.id)">
                          <span>Output</span>
                          <span class="rdm-tool-toggle">{{ expandedToolResults.has(tc.id) ? '▲' : '▼' }}</span>
                        </div>
                        <pre v-if="expandedToolResults.has(tc.id)" class="rdm-code-block rdm-code-block--res">{{ tc.result }}</pre>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Permission events -->
          <div v-if="agentsStore.permissionEvents.get(props.runId)?.length" class="rdm-field">
            <div class="rdm-field-label">Permission Events</div>
            <div class="rdm-perm-list">
              <div
                v-for="(ev, idx) in agentsStore.permissionEvents.get(props.runId)"
                :key="idx"
                class="rdm-perm-row"
              >
                <span class="rdm-perm-chip" :data-decision="ev.decision">{{ ev.decision }}</span>
                <span class="rdm-perm-tool">{{ ev.tool_name }}</span>
                <span class="rdm-perm-target">{{ ev.target_path ?? ev.command ?? '' }}</span>
                <span class="rdm-perm-reason">{{ ev.reason }}</span>
                <span class="rdm-perm-time">{{ new Date(ev.timestamp).toLocaleTimeString() }}</span>
              </div>
            </div>
          </div>

          <!-- Stderr tail -->
          <div class="rdm-field" v-if="run.stderr_tail">
            <div class="rdm-field-label">Stderr tail</div>
            <pre class="rdm-log rdm-log--err">{{ run.stderr_tail }}</pre>
          </div>

          <!-- Files created / modified -->
          <div class="rdm-field" v-if="run.artifacts_produced?.length">
            <div class="rdm-field-label">Files created / modified</div>
            <ul class="rdm-artifacts">
              <li v-for="p in run.artifacts_produced" :key="p" class="rdm-artifact-path rdm-mono">
                <router-link
                  v-if="isLifecycleArtifact(p)"
                  :to="artifactLink(p)"
                  class="rdm-artifact-link"
                  @click="emit('close')"
                >{{ p }}</router-link>
                <span v-else>{{ p }}</span>
              </li>
            </ul>
          </div>

          <div
            v-if="!run.stderr_tail && !run.artifacts_produced?.length"
            class="rdm-state"
          >No output recorded.</div>

          <!-- View Full Log button -->
          <div class="rdm-log-action">
            <button
              class="rdm-btn-log"
              :disabled="run.status === 'running'"
              :title="run.status === 'running' ? 'Log not yet available while run is in progress' : 'View the full raw log'"
              @click="showRawLog = true"
            >View Full Log</button>
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
.rdm-artifact-link {
  color: var(--color-accent);
  text-decoration: none;
}
.rdm-artifact-link:hover {
  text-decoration: underline;
}

.rdm-diagnostics {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  padding: var(--space-2) var(--space-3);
}
.rdm-diagnostics-summary {
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
  color: var(--color-text);
}
.rdm-diagnostics-list {
  margin: var(--space-2) 0 0 0;
  display: grid;
  grid-template-columns: max-content 1fr;
  gap: var(--space-1) var(--space-3);
}
.rdm-diagnostics-key {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
}
.rdm-diagnostics-value {
  font-size: 12px;
  font-family: monospace;
  color: var(--color-text);
  word-break: break-word;
  margin: 0;
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

/* Permission events */
.rdm-perm-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.rdm-perm-row {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
  font-size: 12px;
}
.rdm-perm-chip {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 99px;
  font-size: 10px;
  font-weight: 600;
  flex-shrink: 0;
}
.rdm-perm-chip[data-decision="allow"] { background: var(--badge-done-bg); color: var(--badge-done-text); }
.rdm-perm-chip[data-decision="deny"]  { background: var(--badge-blocked-bg); color: var(--badge-blocked-text); }
.rdm-perm-tool { font-family: monospace; font-weight: 600; color: var(--color-text); flex-shrink: 0; }
.rdm-perm-target { font-family: monospace; color: var(--color-text-muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 280px; }
.rdm-perm-reason { color: var(--color-text-muted); flex: 1; }
.rdm-perm-time { color: var(--color-text-muted); flex-shrink: 0; font-size: 11px; }

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
.rdm-btn-log:hover:not(:disabled) {
  background: var(--color-border);
}
.rdm-btn-log:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

/* Turn Timeline */
.rdm-turns-timeline {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.rdm-turn-card {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  overflow: hidden;
}
.rdm-turn-card--recovered {
  border-color: #f59e0b;
}
.rdm-turn-header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  cursor: pointer;
  user-select: none;
  font-size: 12px;
}
.rdm-turn-header:hover {
  background: var(--color-bg);
}
.rdm-turn-badge {
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 99px;
  font-size: 11px;
  background: var(--color-border);
  color: var(--color-text);
}
.rdm-turn-badge[data-role="system"] {
  background: #e0e7ff;
  color: #4338ca;
}
.rdm-turn-badge[data-role="user"] {
  background: #fef3c7;
  color: #92400e;
}
.rdm-turn-badge[data-role="assistant"] {
  background: #dbeafe;
  color: #1d4ed8;
}
.rdm-recovery-badge {
  font-size: 10px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 99px;
  background: #fef3c7;
  color: #b45309;
  border: 1px solid #fde68a;
}
.rdm-turn-tool-count {
  color: var(--color-text-muted);
  font-size: 11px;
}
.rdm-turn-toggle {
  margin-left: auto;
  color: var(--color-text-muted);
  font-size: 10px;
}
.rdm-turn-body {
  padding: var(--space-3);
  border-top: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  background: var(--color-bg);
}
.rdm-turn-content {
  font-size: 12px;
}
.rdm-turn-text {
  font-family: inherit;
  white-space: pre-wrap;
  word-break: break-word;
  margin: 0;
  color: var(--color-text);
}
.rdm-tool-calls-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.rdm-tool-card {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  padding: var(--space-2);
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.rdm-tool-header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: 12px;
}
.rdm-tool-name {
  font-family: monospace;
  font-weight: 600;
  color: var(--color-accent);
}
.rdm-tool-id {
  font-family: monospace;
  font-size: 10px;
  color: var(--color-text-muted);
}
.rdm-tool-rec-tag {
  font-size: 9px;
  font-weight: 700;
  padding: 0 4px;
  border-radius: 4px;
  background: #fef3c7;
  color: #b45309;
}
.rdm-tool-block {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.rdm-tool-block-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 11px;
  font-weight: 500;
  color: var(--color-text-muted);
  cursor: pointer;
  padding: 2px 4px;
  border-radius: var(--radius-sm);
}
.rdm-tool-block-header:hover {
  background: var(--color-bg);
  color: var(--color-text);
}
.rdm-tool-toggle {
  font-size: 9px;
}
.rdm-code-block {
  font-family: monospace;
  font-size: 11px;
  background: #0f172a;
  color: #e2e8f0;
  padding: var(--space-2);
  border-radius: var(--radius-sm);
  margin: 0;
  max-height: 160px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-all;
}
.rdm-code-block--res {
  color: #86efac;
}
</style>
