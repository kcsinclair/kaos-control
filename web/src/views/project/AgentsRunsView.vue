<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAgentsStore } from '@/stores/agents'
import RunAgentDialog from '@/components/agent/RunAgentDialog.vue'

const route = useRoute()
const router = useRouter()
const store = useAgentsStore()
const project = route.params.project as string

const showRunDialog = ref(false)
const expandedRun = ref<string | null>(null)

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

onMounted(() => store.fetchRuns(project))
</script>

<template>
  <div class="runs-view">
    <div class="runs-header">
      <h2 class="runs-title">Agent Runs</h2>
      <button class="btn-primary" @click="showRunDialog = true">Run Agent</button>
    </div>

    <div v-if="store.loading" class="state-msg">Loading…</div>
    <div v-else-if="!store.runs.length" class="state-msg">No runs yet.</div>

    <table v-else class="runs-table">
      <thead>
        <tr>
          <th>Run ID</th>
          <th>Agent</th>
          <th>Target</th>
          <th>Status</th>
          <th>Started</th>
          <th>Elapsed</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <template v-for="run in store.runs" :key="run.run_id">
          <tr class="run-row" @click="toggleExpand(run.run_id)">
            <td class="cell-mono">{{ run.run_id.slice(0, 8) }}…</td>
            <td>{{ run.agent_name }}</td>
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
            <td colspan="7" class="detail-cell">
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
              <div v-if="!run.stderr_tail && !run.artifacts_produced?.length && run.status !== 'running'" class="detail-empty">
                No output recorded.
              </div>
            </td>
          </tr>
        </template>
      </tbody>
    </table>

    <RunAgentDialog
      v-if="showRunDialog"
      :project="project"
      @started="showRunDialog = false"
      @cancel="showRunDialog = false"
    />
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
  background: #fee2e2;
  color: #991b1b;
  border: 1px solid #fca5a5;
  border-radius: var(--radius-sm);
  font-size: 11px;
  cursor: pointer;
}
.btn-kill:hover { background: #fecaca; }
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
.status-chip[data-status="running"] { background: #dbeafe; color: #1d4ed8; }
.status-chip[data-status="done"] { background: #d1fae5; color: #065f46; }
.status-chip[data-status="failed"] { background: #fee2e2; color: #991b1b; }
.status-chip[data-status="killed"] { background: #fef3c7; color: #92400e; }
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
</style>
