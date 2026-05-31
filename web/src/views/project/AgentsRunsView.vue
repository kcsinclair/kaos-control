<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed, ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAgentsStore } from '@/stores/agents'
import { useUiStore } from '@/stores/ui'
import { usePagination } from '@/composables/usePagination'
import { useSortableTable } from '@/composables/useSortableTable'
import * as agentsApi from '@/api/agents'
import * as configApi from '@/api/config'
import RunAgentDialog from '@/components/agent/RunAgentDialog.vue'
import AgentPanelRow from '@/components/agent/AgentPanelRow.vue'
import AgentLaunchModal from '@/components/agent/AgentLaunchModal.vue'
import AgentConfigForm from '@/components/agent/AgentConfigForm.vue'
import RunFailureBanner from '@/components/agent/RunFailureBanner.vue'
import RunSummaryCard from '@/components/agent/RunSummaryCard.vue'
import RunDenialSummary from '@/components/agent/RunDenialSummary.vue'
import RawLogModal from '@/components/agent/RawLogModal.vue'
import TablePagination from '@/components/common/TablePagination.vue'
import SortHeader from '@/components/SortHeader.vue'
import type { AgentSummary, AgentRunRow } from '@/types/api'
import type { AgentFormData } from '@/components/agent/AgentConfigForm.vue'
import { useProjectConfigStore } from '@/stores/projectConfig'
import { useQueueStore } from '@/stores/queue'

function agentDriver(agentName: string, agents: AgentSummary[]): string {
  const a = agents.find((ag) => ag.name === agentName)
  if (!a) return ''
  if (a.driver === 'ollama') return 'Ollama'
  if (a.driver === 'claude-code-cli') return 'Claude Code'
  if (a.driver === 'claude-mediated') return 'Claude Mediated'
  if (a.driver === 'codex-cli') return 'Codex'
  if (a.driver === 'gemini') return 'Gemini'
  if (a.driver === 'gemini-cli') return 'Gemini CLI'
  return a.driver
}

function agentHasTokenMetrics(agentName: string): boolean {
  const driver = store.agents.find((ag) => ag.name === agentName)?.driver
  if (!driver) return true
  return driver === 'claude-code-cli' || driver === 'claude-mediated'
}

const route = useRoute()
const router = useRouter()
const store = useAgentsStore()
const ui = useUiStore()
const configStore = useProjectConfigStore()
const queueStore = useQueueStore()
const project = route.params.project as string

const showRunDialog = ref(false)
const expandedRun = ref<string | null>(null)
const selectedAgent = ref<AgentSummary | null>(null)

// Agent config form modal state
const showAgentForm = ref(false)
const editAgent = ref<AgentSummary | null>(null)

function openNewAgent() {
  editAgent.value = null
  showAgentForm.value = true
}

function openEditAgent(agent: AgentSummary) {
  editAgent.value = agent
  showAgentForm.value = true
}

function closeAgentForm() {
  showAgentForm.value = false
  editAgent.value = null
}

async function handleAgentFormSubmit(data: AgentFormData) {
  try {
    const res = await configApi.getConfig(project)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const cfg = configApi.parseConfigYaml(res.raw) as any
    const agents: unknown[] = Array.isArray(cfg.agents) ? cfg.agents : []

    // Build agent entry object
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const entry: Record<string, any> = {
      name: data.name,
      role: data.roles,
      driver: data.driver,
      model: data.model || undefined,
      timeout_minutes: data.timeout_minutes,
    }
    if (data.driver === 'ollama') {
      entry.ollama_instance = data.ollama_instance || undefined
      entry.ollama_endpoint = data.ollama_endpoint !== 'chat' ? data.ollama_endpoint : undefined
    }
    if (data.allowed_write_paths.length) {
      entry.allowed_write_paths = data.allowed_write_paths
    }
    if (data.git_identity_name || data.git_identity_email) {
      entry.git_identity = {
        name: data.git_identity_name || undefined,
        email: data.git_identity_email || undefined,
      }
    }
    if (Object.keys(data.prompt_templates).length) {
      entry.prompt_templates = data.prompt_templates
    }

    const idx = agents.findIndex((a) => (a as Record<string, unknown>).name === data.name)
    if (idx >= 0) {
      agents[idx] = entry
    } else {
      agents.push(entry)
    }
    cfg.agents = agents

    await configApi.updateConfig(project, configApi.dumpConfigYaml(cfg))
    await store.fetchAgents(project)
    closeAgentForm()
    ui.success(editAgent.value ? 'Agent updated.' : 'Agent created.')
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to save agent')
  }
}

const { currentPage, pageSize, sliceStart, sliceEnd, setPage, setPageSize } = usePagination({ queryPrefix: 'runs' })

function elapsedMs(row: AgentRunRow): number {
  const start = new Date(row.started_at).getTime()
  const end = row.finished_at ? new Date(row.finished_at).getTime() : Date.now()
  return end - start
}

const runsRef = computed(() => store.runs)

const { sortColumn, sortDirection, sortedRows: sortedRuns, toggleSort } = useSortableTable(
  runsRef,
  {
    run_id:      { type: 'string' },
    agent_name:  { type: 'string' },
    target_path: { type: 'string' },
    status:      { type: 'string' },
    started_at:  { type: 'date' },
    elapsed:     { type: 'number', getValue: (row) => elapsedMs(row as AgentRunRow) },
  },
)

const paginatedRuns = computed(() => sortedRuns.value.slice(sliceStart.value, sliceEnd.value))

// Reset to page 1 on sort change
watch([sortColumn, sortDirection], () => setPage(1))

// run_id currently being viewed in the full-height RawLogModal (one open at a
// time across the whole runs table). null = closed.
const fullLogRunId = ref<string | null>(null)

// Run statuses that have produced a terminal type:result line we can summarise.
// `running` / `queued` runs don't have one yet so the summary section is hidden.
const TERMINAL_RUN_STATUSES = new Set(['done', 'failed', 'killed', 'killed-timeout'])

// Tracks in-flight summary fetches per run_id so the loading state is per-row,
// not shared across expansions. Result values live in agentsStore.runResults
// (also populated by WS events as runs complete).
const summaryLoading = ref(new Set<string>())

async function loadRunSummary(runId: string) {
  if (store.runResults.has(runId)) return
  if (summaryLoading.value.has(runId)) return
  summaryLoading.value.add(runId)
  try {
    const { result } = await agentsApi.getRunResult(project, runId)
    if (result) store.runResults.set(runId, result)
  } catch {
    // The endpoint returns null for runs with no type:result line (e.g. precheck
    // failures). Surface those silently — the summary card handles null.
  } finally {
    summaryLoading.value.delete(runId)
  }
}

function toggleExpand(runId: string) {
  const opening = expandedRun.value !== runId
  expandedRun.value = opening ? runId : null
  if (opening) {
    const row = paginatedRuns.value.find((r) => r.run_id === runId)
    if (row && TERMINAL_RUN_STATUSES.has(row.status)) {
      void loadRunSummary(runId)
    }
  }
}

function elapsed(row: { started_at: string; finished_at?: string }): string {
  const start = new Date(row.started_at).getTime()
  const end = row.finished_at ? new Date(row.finished_at).getTime() : Date.now()
  const secs = Math.round((end - start) / 1000)
  if (secs < 60) return `${secs}s`
  const mins = Math.floor(secs / 60)
  return `${mins}m ${secs % 60}s`
}

async function kill(runId: string) {
  try {
    await store.killRun(project, runId)
  } catch {
    // error shown via store
  }
}

onMounted(() => {
  store.fetchRuns(project)
  if (!store.agents.length) store.fetchAgents(project)
  // Ready counts populate the per-agent badge; without this initial fetch the
  // badges would read 0 until the first artifact.indexed WebSocket event.
  void store.fetchReadyCounts(project)
  configStore.fetchRoles(project)
  void queueStore.fetch()
})
</script>

<template>
  <div class="runs-view">
    <div class="runs-header">
      <h2 class="runs-title">Agent Runs</h2>
      <div class="runs-header-actions">
        <button class="btn-secondary" @click="openNewAgent">New Agent</button>
        <button class="btn-run-agent" @click="showRunDialog = true">Run Agent</button>
      </div>
    </div>

    <!-- Queue pause banner -->
    <div v-if="queueStore.isPaused" class="queue-pause-banner" role="alert">
      <span class="queue-pause-banner__icon" aria-hidden="true">&#9888;</span>
      <span class="queue-pause-banner__text">
        Agent queue is paused due to denied tool calls. Review the denied calls and resume the queue.
      </span>
      <button class="queue-pause-banner__btn" @click="queueStore.resume()">Resume Queue</button>
    </div>

    <AgentPanelRow
      :agents="store.agents"
      @select="selectedAgent = $event"
      @edit="openEditAgent($event)"
    />

    <div v-if="store.loading" class="state-msg">Loading…</div>
    <div v-else-if="!store.runs.length" class="state-msg">No runs yet.</div>

    <div v-else class="table-scroll">
    <table class="runs-table">
      <thead>
        <tr>
          <SortHeader label="Run ID"  column="run_id"      :sort-column="sortColumn" :sort-direction="sortDirection" :sortable="true" @toggle="toggleSort" />
          <SortHeader label="Agent"   column="agent_name"  :sort-column="sortColumn" :sort-direction="sortDirection" :sortable="true" @toggle="toggleSort" />
          <th>Driver</th>
          <SortHeader label="Target"  column="target_path" :sort-column="sortColumn" :sort-direction="sortDirection" :sortable="true" @toggle="toggleSort" />
          <SortHeader label="Status"  column="status"      :sort-column="sortColumn" :sort-direction="sortDirection" :sortable="true" @toggle="toggleSort" />
          <SortHeader label="Started" column="started_at"  :sort-column="sortColumn" :sort-direction="sortDirection" :sortable="true" @toggle="toggleSort" />
          <SortHeader label="Elapsed" column="elapsed"     :sort-column="sortColumn" :sort-direction="sortDirection" :sortable="true" @toggle="toggleSort" />
          <th></th>
        </tr>
      </thead>
      <tbody>
        <template v-for="run in paginatedRuns" :key="run.run_id">
          <tr class="run-row" @click="toggleExpand(run.run_id)">
            <td class="cell-mono">{{ run.run_id.slice(0, 8) }}…</td>
            <td>{{ run.agent_name }}</td>
            <td>
              <span
                v-if="agentDriver(run.agent_name, store.agents)"
                class="driver-badge"
                :data-driver="store.agents.find(a => a.name === run.agent_name)?.driver"
              >{{ agentDriver(run.agent_name, store.agents) }}</span>
            </td>
            <td class="cell-path">
              <button
                class="path-link"
                @click.stop="router.push(`/p/${project}/artifacts/${run.target_path}`)"
              >{{ run.target_path }}</button>
            </td>
            <td>
              <span class="status-chip" :data-status="run.status">{{ run.status }}</span>
            </td>
            <td class="cell-muted">{{ new Date(run.started_at).toLocaleTimeString() }}</td>
            <td class="cell-muted">{{ elapsed(run) }}</td>
            <td class="cell-actions" @click.stop>
              <button
                v-if="run.status === 'running'"
                class="btn-kill"
                @click="kill(run.run_id)"
              >Kill</button>
              <span class="expand-toggle">{{ expandedRun === run.run_id ? '▲' : '▼' }}</span>
            </td>
          </tr>
          <tr v-if="expandedRun === run.run_id" class="run-detail">
            <td colspan="8" class="detail-cell">
              <!-- Precheck failure banner -->
              <RunFailureBanner
                v-if="run.status === 'failed' && run.failure_reason"
                :failure-reason="run.failure_reason"
                :observed-mode="run.observed_permission_mode"
                :remediation="run.remediation"
              />
              <!-- Denial notice for done runs that had denials (on_denial: continue) -->
              <RunFailureBanner
                v-else-if="run.status === 'done' && run.denied_tool_calls?.length"
                :denial-count="run.denied_tool_calls.length"
              />
              <!-- Live progress for running runs -->
              <div v-if="run.status === 'running' && store.progressLines.get(run.run_id)?.length" class="detail-section">
                <div class="detail-label">Progress</div>
                <pre class="detail-log">{{ store.progressLines.get(run.run_id)!.slice(-30).join('\n') }}</pre>
              </div>
              <!-- Permission events -->
              <div v-if="store.permissionEvents.get(run.run_id)?.length" class="detail-section">
                <div class="detail-label">Permission Events</div>
                <!-- Observe-only mode notice -->
                <div
                  v-if="store.agents.find(a => a.name === run.agent_name)?.observe_only"
                  class="observe-notice"
                >
                  Observe-only mode — all tool calls were allowed.
                  Decisions shown are what would have been enforced.
                </div>
                <div class="perm-event-list">
                  <div
                    v-for="(ev, idx) in store.permissionEvents.get(run.run_id)"
                    :key="idx"
                    class="perm-event-row"
                  >
                    <span class="perm-chip" :data-decision="ev.decision">{{ ev.decision }}</span>
                    <span class="perm-tool">{{ ev.tool_name }}</span>
                    <span class="perm-target">{{ ev.target_path ?? ev.command ?? '' }}</span>
                    <span class="perm-reason">{{ ev.reason }}</span>
                    <span class="perm-time">{{ new Date(ev.timestamp).toLocaleTimeString() }}</span>
                  </div>
                </div>
              </div>
              <!-- Stderr tail for completed -->
              <div v-if="run.stderr_tail" class="detail-section">
                <div class="detail-label">Stderr tail</div>
                <pre class="detail-log detail-log--err">{{ run.stderr_tail }}</pre>
              </div>
              <!-- Artifacts produced -->
              <div v-if="run.artifacts_produced?.length" class="detail-section">
                <div class="detail-label">Artifacts produced</div>
                <div class="artifact-list">
                  <button
                    v-for="p in run.artifacts_produced"
                    :key="p"
                    class="artifact-link"
                    @click="router.push(`/p/${project}/artifacts/${p}`)"
                  >{{ p }}</button>
                </div>
              </div>
              <!-- Denied-calls summary (runs with denials) -->
              <RunDenialSummary
                v-if="run.denied_tool_calls?.length"
                :denials="run.denied_tool_calls"
                :observe-only="store.agents.find(a => a.name === run.agent_name)?.observe_only"
              />
              <!-- Run summary (terminal runs only) -->
              <div v-if="TERMINAL_RUN_STATUSES.has(run.status)" class="detail-section">
                <div class="detail-label">Run summary</div>
                <div v-if="summaryLoading.has(run.run_id)" class="detail-empty">Loading summary…</div>
                <RunSummaryCard
                  v-else
                  :result="store.runResults.get(run.run_id) ?? null"
                  :driver-available="agentHasTokenMetrics(run.agent_name)"
                />
              </div>
              <!-- Full log — opens in a full-height modal (same component
                   used by the artefact-screen run-detail modal). -->
              <div class="detail-section">
                <div class="detail-label">Run log</div>
                <button class="btn-link" @click="fullLogRunId = run.run_id">View full log</button>
              </div>
              <div v-if="!run.stderr_tail && !run.artifacts_produced?.length && run.status !== 'running'" class="detail-empty">
                No output recorded.
              </div>
            </td>
          </tr>
        </template>
      </tbody>
    </table>
    </div>

    <TablePagination
      v-if="!store.loading && store.runs.length > 0"
      :total-items="store.runs.length"
      :current-page="currentPage"
      :page-size="pageSize"
      @update:current-page="setPage"
      @update:page-size="setPageSize"
    />

    <RunAgentDialog
      v-if="showRunDialog"
      :project="project"
      @started="showRunDialog = false"
      @cancel="showRunDialog = false"
    />

    <AgentLaunchModal
      v-if="selectedAgent"
      :agent="selectedAgent"
      :project="project"
      @started="selectedAgent = null; store.fetchRuns(project)"
      @cancel="selectedAgent = null"
    />

    <!-- Full-height log viewer — same component the artefact-screen
         run-detail modal uses, so the experience matches across views. -->
    <RawLogModal
      v-if="fullLogRunId"
      :project="project"
      :run-id="fullLogRunId"
      @close="fullLogRunId = null"
    />

    <!-- Agent config form modal -->
    <Teleport to="body">
      <div v-if="showAgentForm" class="modal-overlay" @click.self="closeAgentForm">
        <div class="modal-panel" role="dialog" aria-modal="true" :aria-label="editAgent ? 'Edit agent' : 'New agent'">
          <div class="modal-header">
            <h3 class="modal-title">{{ editAgent ? 'Edit Agent' : 'New Agent' }}</h3>
            <button class="modal-close" aria-label="Close" @click="closeAgentForm">✕</button>
          </div>
          <div class="modal-body">
            <AgentConfigForm
              :initial="editAgent"
              :available-roles="configStore.roles"
              :existing-names="store.agents.map(a => a.name)"
              @submit="handleAgentFormSubmit"
              @cancel="closeAgentForm"
            />
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.runs-view {
  display: flex;
  flex-direction: column;
  min-height: 100%;
}
.runs-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.runs-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}
.btn-run-agent {
  padding: var(--space-2) var(--space-4);
  background: var(--color-accent);
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-run-agent:hover { opacity: 0.88; }
.state-msg {
  padding: var(--space-8) var(--space-6);
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.runs-table {
  width: 100%;
  border-collapse: collapse;
}
.runs-table th {
  position: sticky;
  top: 0;
  background: var(--color-bg);
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  padding: var(--space-2) var(--space-4);
  text-align: left;
  border-bottom: 1px solid var(--color-border);
  z-index: 1;
}
.run-row {
  cursor: pointer;
  border-bottom: 1px solid var(--color-border);
}
.run-row:hover { background: var(--color-surface); }
.runs-table td {
  padding: var(--space-2) var(--space-4);
  vertical-align: middle;
  font-size: var(--text-sm);
}
.cell-mono { font-family: monospace; color: var(--color-text-muted); }
.cell-muted { color: var(--color-text-muted); }
.cell-path { max-width: 260px; overflow: hidden; }
.path-link {
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
  color: var(--color-accent);
  font-size: var(--text-sm);
  font-family: monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
  display: block;
}
.path-link:hover { text-decoration: underline; }
.cell-actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.btn-kill {
  padding: 2px var(--space-2);
  background: var(--badge-blocked-bg);
  color: var(--badge-blocked-text);
  border: 1px solid var(--color-error);
  border-radius: var(--radius-sm);
  font-size: 11px;
  cursor: pointer;
}
.btn-kill:hover { opacity: 0.85; }
.expand-toggle { font-size: 10px; color: var(--color-text-muted); }
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
.run-detail { background: var(--color-surface); }
.detail-cell { padding: var(--space-4) var(--space-6) !important; }
.detail-section { margin-bottom: var(--space-3); }
.detail-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  margin-bottom: var(--space-1);
}
.detail-log {
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
.detail-log--err { color: #fca5a5; }
.artifact-list { display: flex; flex-direction: column; gap: var(--space-1); }
.artifact-link {
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
  color: var(--color-accent);
  font-size: 12px;
  font-family: monospace;
  text-align: left;
}
.artifact-link:hover { text-decoration: underline; }
.detail-empty { font-size: var(--text-sm); color: var(--color-text-muted); }
.driver-badge {
  display: inline-block;
  font-size: 10px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 99px;
  background: var(--color-border);
  color: var(--color-text-muted);
  white-space: nowrap;
}
.driver-badge[data-driver="ollama"] { background: #dbeafe; color: #1d4ed8; }
.driver-badge[data-driver="claude-code-cli"] { background: #f3e8ff; color: #7e22ce; }
.driver-badge[data-driver="claude-mediated"] { background: #fef3c7; color: #92400e; }
.driver-badge[data-driver="codex-cli"] { background: #dcfce7; color: #166534; }
.driver-badge[data-driver="gemini"] { background: #e0e7ff; color: #4338ca; }
.driver-badge[data-driver="gemini-cli"] { background: #ccfbf1; color: #0f766e; }
/* Queue pause banner */
.queue-pause-banner {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-6);
  background: var(--badge-blocked-bg);
  border-bottom: 1px solid var(--color-error, #dc2626);
  color: var(--badge-blocked-text);
  font-size: var(--text-sm);
}
.queue-pause-banner__icon {
  font-size: var(--text-base, 1rem);
  flex-shrink: 0;
}
.queue-pause-banner__text {
  flex: 1;
}
.queue-pause-banner__btn {
  padding: var(--space-1) var(--space-3);
  background: var(--color-error, #dc2626);
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
  flex-shrink: 0;
}
.queue-pause-banner__btn:hover { opacity: 0.85; }
/* Observe-only notice */
.observe-notice {
  font-size: 12px;
  color: #92400e;
  background: #fef3c7;
  border: 1px solid #fbbf24;
  border-radius: var(--radius-sm);
  padding: var(--space-2) var(--space-3);
  margin-bottom: var(--space-2);
}
/* Permission events */
.perm-event-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.perm-event-row {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
  font-size: 12px;
}
.perm-chip {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 99px;
  font-size: 10px;
  font-weight: 600;
  flex-shrink: 0;
}
.perm-chip[data-decision="allow"] { background: var(--badge-done-bg); color: var(--badge-done-text); }
.perm-chip[data-decision="deny"]  { background: var(--badge-blocked-bg); color: var(--badge-blocked-text); }
.perm-tool { font-family: monospace; font-weight: 600; color: var(--color-text); flex-shrink: 0; }
.perm-target { font-family: monospace; color: var(--color-text-muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 280px; }
.perm-reason { color: var(--color-text-muted); flex: 1; }
.perm-time { color: var(--color-text-muted); flex-shrink: 0; font-size: 11px; }
.runs-header-actions { display: flex; gap: var(--space-2); }
.btn-secondary {
  padding: var(--space-2) var(--space-4);
  background: transparent;
  color: var(--color-text-muted);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-secondary:hover { border-color: var(--color-text-muted); color: var(--color-text); }
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 300;
  padding: var(--space-6);
}
.modal-panel {
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  width: 100%;
  max-width: 580px;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-5) var(--space-6) var(--space-3);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.modal-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}
.modal-close {
  background: none;
  border: none;
  font-size: var(--text-lg);
  color: var(--color-text-muted);
  cursor: pointer;
  line-height: 1;
  padding: var(--space-1);
}
.modal-close:hover { color: var(--color-text); }
.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-5) var(--space-6);
}

/* Mobile: tighter header padding, drop the title size, and let the
   header-actions wrap below the title if needed. */
@media (max-width: 640px) {
  .runs-header {
    padding: var(--space-3) var(--space-4);
    flex-wrap: wrap;
    gap: var(--space-3);
  }
  .runs-header-actions {
    width: 100%;
    justify-content: flex-end;
  }
  .modal-body {
    padding: var(--space-4);
  }
}
</style>
