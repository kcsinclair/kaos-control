<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useSchedulerStore } from '@/stores/scheduler'
import { useUiStore } from '@/stores/ui'
import { usePagination } from '@/composables/usePagination'
import { useSortableTable } from '@/composables/useSortableTable'
import TablePagination from '@/components/common/TablePagination.vue'
import SortHeader from '@/components/SortHeader.vue'
import type { SchedulerJob } from '@/types/api'

// Milestone 6 — JobForm is used in the create modal
// imported lazily to avoid circular-dep issues at build time
import JobForm from '@/components/scheduler/JobForm.vue'

const route = useRoute()
const router = useRouter()
const store = useSchedulerStore()
const ui = useUiStore()
const project = route.params.project as string

// ─── filters ────────────────────────────────────────────────────────────────
const filterStatus = ref<'all' | 'enabled' | 'paused'>('all')
const filterTargetType = ref<'all' | 'agent' | 'shell'>('all')

const filteredJobs = computed(() => {
  return store.jobs.filter((j) => {
    if (filterStatus.value === 'enabled' && !j.enabled) return false
    if (filterStatus.value === 'paused' && j.enabled) return false
    if (filterTargetType.value !== 'all' && j.target_type !== filterTargetType.value) return false
    return true
  })
})

// ─── sort ────────────────────────────────────────────────────────────────────
const { sortColumn, sortDirection, sortedRows: sortedJobs, toggleSort } =
  useSortableTable<SchedulerJob & Record<string, unknown>>(
    filteredJobs as ReturnType<typeof computed>,
    {
      name:        { type: 'string' },
      priority:    { type: 'number' },
      next_run_at: { type: 'date' },
      last_run_at: { type: 'date' },
    },
  )

// ─── pagination ──────────────────────────────────────────────────────────────
const { currentPage, pageSize, sliceStart, sliceEnd, setPage, setPageSize } =
  usePagination({ queryPrefix: 'sched' })

const paginatedJobs = computed(() =>
  sortedJobs.value.slice(sliceStart.value, sliceEnd.value),
)

watch([sortColumn, sortDirection, filterStatus, filterTargetType], () => setPage(1))

// ─── create modal ────────────────────────────────────────────────────────────
const showCreate = ref(false)

async function onJobCreated() {
  showCreate.value = false
  await store.fetchJobs(project)
}

// ─── actions ─────────────────────────────────────────────────────────────────
async function trigger(name: string) {
  try {
    await store.triggerJob(project, name)
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to trigger job')
  }
}

async function toggleEnabled(job: SchedulerJob) {
  try {
    if (job.enabled) {
      await store.pauseJob(project, job.name)
    } else {
      await store.resumeJob(project, job.name)
    }
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to update job')
  }
}

const deletingName = ref<string | null>(null)

async function confirmDelete(name: string) {
  if (!confirm(`Delete job "${name}"? This cannot be undone.`)) return
  deletingName.value = name
  try {
    await store.deleteJob(project, name)
    ui.success(`Job "${name}" deleted`)
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to delete job')
  } finally {
    deletingName.value = null
  }
}

// ─── helpers ─────────────────────────────────────────────────────────────────
function scheduleLabel(job: SchedulerJob): string {
  return `${job.schedule.type}: ${job.schedule.expression}`
}

function formatDate(iso?: string): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleString(undefined, {
    day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit',
  })
}

// ─── mount ────────────────────────────────────────────────────────────────────
onMounted(() => store.fetchJobs(project))
</script>

<template>
  <div class="sched-view">
    <!-- Header -->
    <div class="sched-header">
      <h2 class="sched-title">Scheduler</h2>
      <button class="btn-primary" @click="showCreate = true">New Job</button>
    </div>

    <!-- Filter bar -->
    <div class="sched-filters">
      <label class="filter-label">
        Status
        <select v-model="filterStatus" class="filter-select">
          <option value="all">All</option>
          <option value="enabled">Enabled</option>
          <option value="paused">Paused</option>
        </select>
      </label>
      <label class="filter-label">
        Target type
        <select v-model="filterTargetType" class="filter-select">
          <option value="all">All</option>
          <option value="agent">Agent</option>
          <option value="shell">Shell</option>
        </select>
      </label>
    </div>

    <!-- Loading / empty -->
    <div v-if="store.loadingJobs" class="state-msg">Loading…</div>
    <div v-else-if="!store.jobs.length" class="state-msg empty-state">
      <p>No scheduled jobs yet.</p>
      <button class="btn-primary" @click="showCreate = true">Create your first job</button>
    </div>
    <div v-else-if="!filteredJobs.length" class="state-msg">No jobs match the current filters.</div>

    <!-- Table -->
    <div v-else class="sched-table-wrap">
      <table class="sched-table">
        <thead>
          <tr>
            <SortHeader label="Name"     column="name"        :sort-column="sortColumn" :sort-direction="sortDirection" :sortable="true" @toggle="toggleSort" />
            <th>Target</th>
            <th>Schedule</th>
            <SortHeader label="Priority" column="priority"    :sort-column="sortColumn" :sort-direction="sortDirection" :sortable="true" @toggle="toggleSort" />
            <th>Last Run</th>
            <SortHeader label="Next Run" column="next_run_at" :sort-column="sortColumn" :sort-direction="sortDirection" :sortable="true" @toggle="toggleSort" />
            <th>Enabled</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="job in paginatedJobs"
            :key="job.name"
            class="sched-row"
            @click="router.push(`/p/${project}/scheduler/${encodeURIComponent(job.name)}`)"
          >
            <td class="cell-name">
              <button
                class="name-link"
                @click.stop="router.push(`/p/${project}/scheduler/${encodeURIComponent(job.name)}`)"
              >{{ job.name }}</button>
            </td>
            <td>
              <span class="target-type">{{ job.target_type }}</span>
              <span class="cell-muted">{{ job.target }}</span>
            </td>
            <td class="cell-muted cell-mono">{{ scheduleLabel(job) }}</td>
            <td class="cell-center">{{ job.priority }}</td>
            <td>
              <span
                v-if="job.last_run_status"
                class="status-chip"
                :data-status="job.last_run_status"
              >{{ job.last_run_status }}</span>
              <span v-else class="cell-muted">—</span>
              <span v-if="job.last_run_at" class="cell-muted cell-date">{{ formatDate(job.last_run_at) }}</span>
            </td>
            <td class="cell-muted">{{ formatDate(job.next_run_at) }}</td>
            <td class="cell-center" @click.stop>
              <button
                class="toggle-btn"
                :class="{ 'toggle-btn--on': job.enabled }"
                :aria-label="job.enabled ? 'Pause job' : 'Resume job'"
                @click="toggleEnabled(job)"
              >{{ job.enabled ? 'On' : 'Off' }}</button>
            </td>
            <td class="cell-actions" @click.stop>
              <button class="btn-action" title="Trigger now" @click="trigger(job.name)">▶</button>
              <button
                class="btn-action btn-action--danger"
                title="Delete job"
                :disabled="deletingName === job.name"
                @click="confirmDelete(job.name)"
              >✕</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Pagination -->
    <TablePagination
      v-if="!store.loadingJobs && filteredJobs.length > 0"
      :total-items="filteredJobs.length"
      :current-page="currentPage"
      :page-size="pageSize"
      @update:current-page="setPage"
      @update:page-size="setPageSize"
    />

    <!-- Create modal -->
    <div v-if="showCreate" class="modal-backdrop" @click.self="showCreate = false">
      <div class="modal-box">
        <div class="modal-header">
          <h3 class="modal-title">New Job</h3>
          <button class="modal-close" aria-label="Close" @click="showCreate = false">✕</button>
        </div>
        <JobForm
          mode="create"
          :project="project"
          @saved="onJobCreated"
          @cancel="showCreate = false"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.sched-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.sched-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.sched-title {
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

.sched-filters {
  display: flex;
  gap: var(--space-4);
  padding: var(--space-3) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
  background: var(--color-surface);
}

.filter-label {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}

.filter-select {
  padding: var(--space-1) var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  background: var(--color-bg);
  color: var(--color-text);
}

.state-msg {
  padding: var(--space-8) var(--space-6);
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-8) var(--space-6);
}

.sched-table-wrap {
  flex: 1;
  overflow-y: auto;
}

.sched-table {
  width: 100%;
  border-collapse: collapse;
}

.sched-table th {
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

.sched-row {
  cursor: pointer;
  border-bottom: 1px solid var(--color-border);
}
.sched-row:hover { background: var(--color-surface); }

.sched-table td {
  padding: var(--space-2) var(--space-4);
  vertical-align: middle;
  font-size: var(--text-sm);
}

.cell-name { white-space: nowrap; }

.name-link {
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
  color: var(--color-accent);
  font-size: var(--text-sm);
  font-weight: 500;
}
.name-link:hover { text-decoration: underline; }

.target-type {
  display: inline-block;
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  background: var(--color-border);
  color: var(--color-text-muted);
  margin-right: var(--space-2);
}

.cell-muted { color: var(--color-text-muted); }
.cell-mono  { font-family: monospace; font-size: 12px; }
.cell-center { text-align: center; }
.cell-date  { display: block; font-size: 11px; }

.cell-actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

/* Run status badges */
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

/* Enabled toggle */
.toggle-btn {
  padding: 2px var(--space-3);
  border-radius: var(--radius-full);
  border: 1px solid var(--color-border);
  background: var(--color-bg);
  color: var(--color-text-muted);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.12s, color 0.12s;
}
.toggle-btn--on {
  background: var(--badge-done-bg);
  color: var(--badge-done-text);
  border-color: transparent;
}
.toggle-btn:hover { opacity: 0.85; }

/* Action buttons */
.btn-action {
  padding: 2px var(--space-2);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: 11px;
  cursor: pointer;
  color: var(--color-text-muted);
  transition: background 0.12s, color 0.12s;
}
.btn-action:hover { background: var(--color-bg); color: var(--color-text); }
.btn-action--danger:hover { background: var(--badge-blocked-bg); color: var(--badge-blocked-text); border-color: var(--color-error); }
.btn-action:disabled { opacity: 0.5; cursor: not-allowed; }

/* Modal */
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
}

.modal-box {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  width: min(640px, 95vw);
  max-height: 90vh;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.modal-title {
  font-size: var(--text-base);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}

.modal-close {
  background: none;
  border: none;
  cursor: pointer;
  font-size: var(--text-base);
  color: var(--color-text-muted);
  padding: var(--space-1);
  border-radius: var(--radius-sm);
}
.modal-close:hover { background: var(--color-bg); color: var(--color-text); }
</style>
