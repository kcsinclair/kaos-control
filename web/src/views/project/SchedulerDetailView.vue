<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useSchedulerStore } from '@/stores/scheduler'
import { useUiStore } from '@/stores/ui'
import { usePagination } from '@/composables/usePagination'
import TablePagination from '@/components/common/TablePagination.vue'
import JobForm from '@/components/scheduler/JobForm.vue'
import type { SchedulerJob, SchedulerRun } from '@/types/api'

const route  = useRoute()
const router = useRouter()
const store  = useSchedulerStore()
const ui     = useUiStore()

const project = route.params.project as string
const jobName = computed(() => route.params.name as string)

// ─── edit mode ────────────────────────────────────────────────────────────────
const editMode = ref(false)

function onSaved(job: SchedulerJob) {
  editMode.value = false
  store.fetchJob(project, job.name)
}

// ─── actions ─────────────────────────────────────────────────────────────────
async function trigger() {
  try {
    await store.triggerJob(project, jobName.value)
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to trigger')
  }
}

async function toggleEnabled() {
  if (!store.selectedJob) return
  try {
    if (store.selectedJob.enabled) {
      await store.pauseJob(project, jobName.value)
    } else {
      await store.resumeJob(project, jobName.value)
    }
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to update job')
  }
}

async function deleteJob() {
  if (!confirm(`Delete job "${jobName.value}"? This cannot be undone.`)) return
  try {
    await store.deleteJob(project, jobName.value)
    ui.success(`Job "${jobName.value}" deleted`)
    router.push(`/p/${project}/scheduler`)
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to delete job')
  }
}

// ─── run history ─────────────────────────────────────────────────────────────
const { currentPage, pageSize, setPage, setPageSize } = usePagination({ queryPrefix: 'runs' })

async function loadRuns() {
  await store.fetchRuns(project, jobName.value, currentPage.value, pageSize.value)
}

watch([currentPage, pageSize], loadRuns)

// ─── log viewer ───────────────────────────────────────────────────────────────
const logVisible  = ref(new Map<number, boolean>())
const logLoading  = ref(new Map<number, boolean>())
const logContent  = ref(new Map<number, string>())

async function viewLog(run: SchedulerRun) {
  if (logVisible.value.get(run.id)) {
    logVisible.value.set(run.id, false)
    return
  }
  logLoading.value.set(run.id, true)
  try {
    const text = await store.fetchRunLog(project, run.job_name, run.id)
    logContent.value.set(run.id, text)
    logVisible.value.set(run.id, true)
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to load log')
  } finally {
    logLoading.value.set(run.id, false)
  }
}

// ─── helpers ─────────────────────────────────────────────────────────────────
function formatDate(iso?: string): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleString(undefined, {
    day: '2-digit', month: 'short', year: 'numeric',
    hour: '2-digit', minute: '2-digit', second: '2-digit',
  })
}

function duration(run: SchedulerRun): string {
  if (!run.end_time) return '—'
  const ms = new Date(run.end_time).getTime() - new Date(run.start_time).getTime()
  const s = Math.round(ms / 1000)
  if (s < 60) return `${s}s`
  return `${Math.floor(s / 60)}m ${s % 60}s`
}

// ─── mount ────────────────────────────────────────────────────────────────────
onMounted(async () => {
  await store.fetchJob(project, jobName.value)
  await loadRuns()
})

// refresh runs when WS events arrive for this job
watch(
  () => store.selectedJob?.last_run_at,
  (newVal, oldVal) => {
    if (newVal && newVal !== oldVal) loadRuns()
  },
)
</script>

<template>
  <div class="detail-view">

    <!-- Loading -->
    <div v-if="!store.selectedJob" class="state-msg">Loading…</div>

    <template v-else>
      <!-- Header -->
      <div class="detail-header">
        <div class="detail-header-left">
          <button class="back-btn" @click="router.push(`/p/${project}/scheduler`)">← Scheduler</button>
          <h2 class="detail-title">{{ store.selectedJob.name }}</h2>
          <span
            v-if="store.selectedJob.last_run_status"
            class="status-chip"
            :data-status="store.selectedJob.last_run_status"
          >{{ store.selectedJob.last_run_status }}</span>
        </div>
        <div class="detail-actions">
          <button class="btn-action" @click="trigger">▶ Trigger</button>
          <button
            class="btn-action"
            :class="{ 'btn-action--active': store.selectedJob.enabled }"
            @click="toggleEnabled"
          >{{ store.selectedJob.enabled ? 'Pause' : 'Resume' }}</button>
          <button class="btn-action" @click="editMode = !editMode">
            {{ editMode ? 'Cancel edit' : 'Edit' }}
          </button>
          <button class="btn-action btn-action--danger" @click="deleteJob">Delete</button>
        </div>
      </div>

      <!-- Edit form -->
      <div v-if="editMode" class="edit-panel">
        <JobForm
          mode="edit"
          :project="project"
          :initial="store.selectedJob"
          @saved="onSaved"
          @cancel="editMode = false"
        />
      </div>

      <!-- Config section (read-only) -->
      <div v-else class="config-section">
        <h3 class="section-title">Configuration</h3>
        <div class="config-grid">
          <div class="config-row">
            <span class="config-key">Target type</span>
            <span class="config-val">{{ store.selectedJob.target_type }}</span>
          </div>
          <div class="config-row">
            <span class="config-key">Target</span>
            <span class="config-val config-val--mono">{{ store.selectedJob.target }}</span>
          </div>
          <div class="config-row">
            <span class="config-key">Schedule</span>
            <span class="config-val config-val--mono">
              {{ store.selectedJob.schedule.type }}: {{ store.selectedJob.schedule.expression }}
            </span>
          </div>
          <div class="config-row">
            <span class="config-key">Priority</span>
            <span class="config-val">{{ store.selectedJob.priority }}</span>
          </div>
          <div class="config-row">
            <span class="config-key">Timeout</span>
            <span class="config-val">{{ store.selectedJob.timeout_sec }}s</span>
          </div>
          <div class="config-row">
            <span class="config-key">Enabled</span>
            <span class="config-val">{{ store.selectedJob.enabled ? 'Yes' : 'No' }}</span>
          </div>
          <div v-if="store.selectedJob.next_run_at" class="config-row">
            <span class="config-key">Next run</span>
            <span class="config-val">{{ formatDate(store.selectedJob.next_run_at) }}</span>
          </div>
          <div v-if="store.selectedJob.last_run_at" class="config-row">
            <span class="config-key">Last run</span>
            <span class="config-val">{{ formatDate(store.selectedJob.last_run_at) }}</span>
          </div>
          <div v-if="store.selectedJob.preconditions?.length" class="config-row config-row--block">
            <span class="config-key">Preconditions</span>
            <ul class="precond-list">
              <li v-for="(pc, i) in store.selectedJob.preconditions" :key="i" class="precond-item">
                <span class="precond-type">{{ pc.type }}</span>
                <span class="config-val--mono">{{ pc.value }}</span>
              </li>
            </ul>
          </div>
          <div v-if="store.selectedJob.args && Object.keys(store.selectedJob.args).length" class="config-row config-row--block">
            <span class="config-key">Args</span>
            <ul class="args-list">
              <li v-for="(v, k) in store.selectedJob.args" :key="k" class="arg-item config-val--mono">
                {{ k }} = {{ v }}
              </li>
            </ul>
          </div>
        </div>
      </div>

      <!-- Run history -->
      <div class="runs-section">
        <h3 class="section-title">Run history</h3>
        <div v-if="store.loadingRuns" class="state-msg">Loading runs…</div>
        <div v-else-if="!store.runs.length" class="state-msg">No runs recorded yet.</div>
        <table v-else class="runs-table">
          <thead>
            <tr>
              <th>Run ID</th>
              <th>Started</th>
              <th>Duration</th>
              <th>Status</th>
              <th>Log</th>
            </tr>
          </thead>
          <tbody>
            <template v-for="run in store.runs" :key="run.id">
              <tr class="run-row">
                <td class="cell-mono">{{ run.id }}</td>
                <td class="cell-muted">{{ formatDate(run.start_time) }}</td>
                <td class="cell-muted">{{ duration(run) }}</td>
                <td>
                  <span class="status-chip" :data-status="run.status">{{ run.status }}</span>
                </td>
                <td>
                  <button
                    class="btn-link"
                    :disabled="logLoading.get(run.id) === true"
                    @click="viewLog(run)"
                  >
                    {{ logLoading.get(run.id) ? 'Loading…' : (logVisible.get(run.id) ? 'Hide log' : 'View log') }}
                  </button>
                </td>
              </tr>
              <tr v-if="logVisible.get(run.id)" class="log-row">
                <td colspan="5" class="log-cell">
                  <pre class="log-pre">{{ logContent.get(run.id) }}</pre>
                </td>
              </tr>
            </template>
          </tbody>
        </table>

        <TablePagination
          v-if="!store.loadingRuns && store.runsTotal > 0"
          :total-items="store.runsTotal"
          :current-page="currentPage"
          :page-size="pageSize"
          @update:current-page="setPage"
          @update:page-size="setPageSize"
        />
      </div>
    </template>
  </div>
</template>

<style scoped>
.detail-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow-y: auto;
}

.state-msg {
  padding: var(--space-8) var(--space-6);
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}

/* Header */
.detail-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
  gap: var(--space-4);
  flex-wrap: wrap;
}

.detail-header-left {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.back-btn {
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
  color: var(--color-accent);
  font-size: var(--text-sm);
}
.back-btn:hover { text-decoration: underline; }

.detail-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}

.detail-actions {
  display: flex;
  gap: var(--space-2);
  flex-wrap: wrap;
}

/* Status chip */
.status-chip {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 500;
  background: var(--color-border);
  color: var(--color-text);
}
.status-chip[data-status="success"] { background: var(--badge-done-bg);          color: var(--badge-done-text); }
.status-chip[data-status="failure"] { background: var(--badge-blocked-bg);        color: var(--badge-blocked-text); }
.status-chip[data-status="timeout"] { background: var(--badge-in-progress-bg);    color: var(--badge-in-progress-text); }
.status-chip[data-status="running"] { background: var(--badge-approved-bg);       color: var(--badge-approved-text); }
.status-chip[data-status="skipped"] { background: var(--badge-rejected-bg);       color: var(--badge-rejected-text); }

/* Action buttons */
.btn-action {
  padding: var(--space-1) var(--space-3);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  cursor: pointer;
  color: var(--color-text-muted);
  transition: background 0.12s, color 0.12s;
}
.btn-action:hover { background: var(--color-bg); color: var(--color-text); }
.btn-action--active {
  background: var(--badge-in-progress-bg);
  color: var(--badge-in-progress-text);
  border-color: transparent;
}
.btn-action--danger:hover { background: var(--badge-blocked-bg); color: var(--badge-blocked-text); border-color: var(--color-error); }

/* Edit panel */
.edit-panel {
  border-bottom: 1px solid var(--color-border);
}

/* Config section */
.config-section {
  padding: var(--space-6);
  border-bottom: 1px solid var(--color-border);
}

.section-title {
  font-size: var(--text-sm);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  margin: 0 0 var(--space-4);
}

.config-grid {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.config-row {
  display: flex;
  align-items: baseline;
  gap: var(--space-3);
}

.config-row--block {
  flex-direction: column;
  gap: var(--space-2);
}

.config-key {
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text-muted);
  min-width: 100px;
  flex-shrink: 0;
}

.config-val {
  font-size: var(--text-sm);
  color: var(--color-text);
}
.config-val--mono {
  font-family: monospace;
  font-size: 12px;
  color: var(--color-text);
}

.precond-list,
.args-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.precond-item,
.arg-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-sm);
}

.precond-type {
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  background: var(--color-border);
  color: var(--color-text-muted);
  padding: 1px 5px;
  border-radius: var(--radius-sm);
  flex-shrink: 0;
}

/* Runs section */
.runs-section {
  padding: var(--space-6);
  flex: 1;
}

.runs-table {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: var(--space-4);
}

.runs-table th {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  padding: var(--space-2) var(--space-3);
  text-align: left;
  border-bottom: 1px solid var(--color-border);
}

.run-row {
  border-bottom: 1px solid var(--color-border);
}
.run-row:hover { background: var(--color-surface); }

.runs-table td {
  padding: var(--space-2) var(--space-3);
  font-size: var(--text-sm);
  vertical-align: middle;
}

.cell-mono  { font-family: monospace; color: var(--color-text-muted); }
.cell-muted { color: var(--color-text-muted); }

.btn-link {
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
  color: var(--color-accent);
  font-size: var(--text-sm);
}
.btn-link:hover { text-decoration: underline; }
.btn-link:disabled { opacity: 0.5; cursor: not-allowed; }

.log-row { background: var(--color-surface); }
.log-cell { padding: var(--space-3) var(--space-6) !important; }
.log-pre {
  font-family: monospace;
  font-size: 12px;
  background: #0f172a;
  color: #e2e8f0;
  padding: var(--space-3);
  border-radius: var(--radius-sm);
  overflow-x: auto;
  max-height: 300px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
}
</style>
