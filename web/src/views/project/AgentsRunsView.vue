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
import TablePagination from '@/components/common/TablePagination.vue'
import SortHeader from '@/components/SortHeader.vue'
import type { AgentSummary, AgentRunRow } from '@/types/api'
import type { AgentFormData } from '@/components/agent/AgentConfigForm.vue'
import { useProjectConfigStore } from '@/stores/projectConfig'

function agentDriver(agentName: string, agents: AgentSummary[]): string {
  const a = agents.find((ag) => ag.name === agentName)
  if (!a) return ''
  if (a.driver === 'ollama') return 'Ollama'
  if (a.driver === 'claude-code-cli') return 'Claude Code'
  return a.driver
}

const route = useRoute()
const router = useRouter()
const store = useAgentsStore()
const ui = useUiStore()
const configStore = useProjectConfigStore()
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

// Per-run log state. logVisible.get(id)===true means the log pane is shown
// and logContent has the fetched text.
const logVisible = ref(new Map<string, boolean>())
const logLoading = ref(new Map<string, boolean>())
const logContent = ref(new Map<string, string>())

async function loadLog(runId: string) {
  logLoading.value.set(runId, true)
  try {
    const text = await agentsApi.getRunLog(project, runId)
    logContent.value.set(runId, text || '(empty log)')
    logVisible.value.set(runId, true)
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to load log')
  } finally {
    logLoading.value.set(runId, false)
  }
}

function toggleExpand(runId: string) {
  expandedRun.value = expandedRun.value === runId ? null : runId
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
})
</script>

<template>
  <div class="runs-view">
    <div class="runs-header">
      <h2 class="runs-title">Agent Runs</h2>
      <div class="runs-header-actions">
        <button class="btn-secondary" @click="openNewAgent">New Agent</button>
        <button class="btn-primary" @click="showRunDialog = true">Run Agent</button>
      </div>
    </div>

    <AgentPanelRow
      :agents="store.agents"
      @select="selectedAgent = $event"
      @edit="openEditAgent($event)"
    />

    <div v-if="store.loading" class="state-msg">Loading…</div>
    <div v-else-if="!store.runs.length" class="state-msg">No runs yet.</div>

    <table v-else class="runs-table">
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
              <!-- Live progress for running runs -->
              <div v-if="run.status === 'running' && store.progressLines.get(run.run_id)?.length" class="detail-section">
                <div class="detail-label">Progress</div>
                <pre class="detail-log">{{ store.progressLines.get(run.run_id)!.slice(-30).join('\n') }}</pre>
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
              <!-- Full log -->
              <div class="detail-section">
                <div class="detail-label">Run log</div>
                <button
                  v-if="logVisible.get(run.run_id) !== true"
                  class="btn-link"
                  @click="loadLog(run.run_id)"
                  :disabled="logLoading.get(run.run_id) === true"
                >{{ logLoading.get(run.run_id) ? 'Loading…' : 'View full log' }}</button>
                <pre v-else class="detail-log">{{ logContent.get(run.run_id) }}</pre>
              </div>
              <div v-if="!run.stderr_tail && !run.artifacts_produced?.length && run.status !== 'running'" class="detail-empty">
                No output recorded.
              </div>
            </td>
          </tr>
        </template>
      </tbody>
    </table>

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
  height: 100%;
  overflow: hidden;
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
.btn-primary {
  padding: var(--space-2) var(--space-4);
  background: var(--color-accent);
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-primary:hover { opacity: 0.88; }
.state-msg {
  padding: var(--space-8) var(--space-6);
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.runs-table {
  width: 100%;
  border-collapse: collapse;
  overflow-y: auto;
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
</style>
