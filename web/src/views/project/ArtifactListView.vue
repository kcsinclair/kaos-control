<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useArtifactsStore } from '@/stores/artifacts'
import { useReleasesStore } from '@/stores/releases'
import { useWebSocket } from '@/composables/useWebSocket'
import { usePagination } from '@/composables/usePagination'
import { useSortableTable } from '@/composables/useSortableTable'
import BrainDumpModal from '@/components/idea/BrainDumpModal.vue'
import TablePagination from '@/components/common/TablePagination.vue'
import SortHeader from '@/components/SortHeader.vue'
import StatusCheckPanel from '@/components/artifact/StatusCheckPanel.vue'
import TextFilter from '@/components/TextFilter.vue'
import { useTextFilterShortcut } from '@/composables/useTextFilterShortcut'
import { useUiStore } from '@/stores/ui'
import { MessageSquarePlus, Bug, ShieldCheck, BookOpen, Bot } from 'lucide-vue-next'
import type { WsEvent } from '@/types/api'
import { TERMINAL_STATUSES } from '@/types/api'
import { formatShortDate, formatFullDateTime } from '@/composables/useFormatDate'

const route = useRoute()
const router = useRouter()
const store = useArtifactsStore()
const releasesStore = useReleasesStore()
const ui = useUiStore()

const showBrainDump = ref(false)
const brainDumpType = ref<'idea' | 'defect' | 'doc'>('idea')
const newIdeaButtonEl = ref<HTMLButtonElement | null>(null)
const showCompleted = ref(false)
const showStatusPanel = ref(false)

const textFilterRef = ref<{ focus: () => void } | null>(null)
useTextFilterShortcut(textFilterRef)

const { currentPage, pageSize, sliceStart, sliceEnd, setPage, setPageSize } = usePagination()

const visibleItems = computed(() => {
  const base = showCompleted.value
    ? store.items
    : store.items.filter(r => !(TERMINAL_STATUSES as readonly string[]).includes(r.status))
  if (!searchText.value) return base
  const q = searchText.value.toLowerCase()
  return base.filter(r => {
    const titleMatch = (r.title || r.slug || '').toLowerCase().includes(q)
    const labelMatch = (r.frontmatter?.labels ?? []).some(l => l.toLowerCase().includes(q))
    return titleMatch || labelMatch
  })
})

function priorityOrder(value: string | undefined): number {
  switch (value) {
    case 'critical': return 4
    case 'high':     return 3
    case 'normal':   return 2
    case 'low':      return 1
    default:         return 0
  }
}

const { sortColumn, sortDirection, sortedRows, toggleSort, resetSort } = useSortableTable(
  visibleItems,
  {
    title:           { type: 'string' },
    stage:           { type: 'string' },
    status:          { type: 'string' },
    type:            { type: 'string' },
    agent_run_count: { type: 'number' },
    created:         { type: 'date' },
    mtime:    { type: 'date' },
    priority: {
      type: 'number',
      getValue: (row) => priorityOrder((row as { frontmatter?: { priority?: string } }).frontmatter?.priority),
    },
    release: {
      type: 'string',
      getValue: (row) => (row as { frontmatter?: { release?: string } }).frontmatter?.release ?? '',
    },
  },
)

const paginatedItems = computed(() => sortedRows.value.slice(sliceStart.value, sliceEnd.value))

function openBrainDump(type: 'idea' | 'defect' | 'doc' = 'idea') {
  brainDumpType.value = type
  showBrainDump.value = true
}

function onBrainDumpClose() {
  showBrainDump.value = false
  nextTick(() => newIdeaButtonEl.value?.focus())
}

function onBrainDumpCreated(path: string) {
  showBrainDump.value = false
  ui.success('Artifact created!')
  router.push(`/p/${project}/artifacts/${path}`)
}

const project = route.params.project as string

const stageOptions = ['', 'ideas', 'requirements', 'backend-plans', 'frontend-plans', 'test-plans', 'dev-plans', 'tests', 'prototypes', 'defects', 'releases']
const statusOptions = ['', 'draft', 'clarifying', 'planning', 'in-development', 'in-qa', 'in-progress', 'done', 'approved', 'blocked', 'rejected', 'abandoned']
const typeOptions = ['', 'idea', 'requirement', 'plan-backend', 'plan-frontend', 'plan-test', 'test', 'prototype', 'defect']

const selectedStage = ref(store.filter.stage ?? '')
const selectedStatus = ref(store.filter.status ?? '')
const selectedLabel = ref(store.filter.label ?? '')
const selectedType = ref(store.filter.type ?? '')
const selectedPriority = ref(store.filter.priority ?? '')
const selectedRelease = ref(store.filter.release ?? '')
const searchText = ref('')

function applyFilters() {
  resetSort()
  setPage(1)
  store.fetchList(project, {
    stage: selectedStage.value || undefined,
    status: selectedStatus.value || undefined,
    label: selectedLabel.value || undefined,
    type: selectedType.value || undefined,
    priority: selectedPriority.value || undefined,
    release: selectedRelease.value === '__unassigned__' ? '__unassigned__' : (selectedRelease.value || undefined),
    q: searchText.value || undefined,
    limit: 0,
    offset: undefined,
  })
}

function onSearchText(v: string) {
  searchText.value = v
  applyFilters()
}

function resetFilters() {
  selectedStage.value = ''
  selectedStatus.value = ''
  selectedLabel.value = ''
  selectedType.value = ''
  selectedPriority.value = ''
  selectedRelease.value = ''
  searchText.value = ''
  applyFilters()
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function highlightMatch(text: string): string {
  if (!searchText.value) return escapeHtml(text)
  const q = searchText.value.toLowerCase()
  const lower = text.toLowerCase()
  const idx = lower.indexOf(q)
  if (idx === -1) return escapeHtml(text)
  return (
    escapeHtml(text.slice(0, idx)) +
    '<mark>' + escapeHtml(text.slice(idx, idx + q.length)) + '</mark>' +
    escapeHtml(text.slice(idx + q.length))
  )
}

// Reset to page 1 when showCompleted toggle changes
watch(showCompleted, () => { resetSort(); setPage(1) })

// Reset to page 1 after sort change
watch([sortColumn, sortDirection], () => setPage(1))

function onToggleSort(col: string) {
  toggleSort(col)
}

function openArtifact(path: string) {
  router.push(`/p/${project}/artifacts/${path}`)
}

// Re-fetch when an artifact is indexed via WebSocket
useWebSocket(project, 'artifact.indexed', (_e: WsEvent) => {
  store.invalidate()
  store.fetchList(project, { limit: 0, offset: undefined })
})

// Re-fetch immediately when an agent run starts so the "Agent Running" pill
// shows on the target artifact. Without this, the pill would only ever
// appear if an `artifact.indexed` event happened to fire mid-run.
useWebSocket(project, 'agent.started', (_e: WsEvent) => {
  store.invalidate()
  store.fetchList(project, { limit: 0, offset: undefined })
})

// Re-fetch immediately when an agent run finishes so counts and pills update
useWebSocket(project, 'agent.finished', (_e: WsEvent) => {
  store.invalidate()
  store.fetchList(project, { limit: 0, offset: undefined })
})

function initFiltersFromQuery() {
  const q = route.query
  if (typeof q.status === 'string') selectedStatus.value = q.status
  if (typeof q.stage === 'string') selectedStage.value = q.stage
  if (typeof q.type === 'string') selectedType.value = q.type
  if (typeof q.label === 'string') selectedLabel.value = q.label
  if (typeof q.priority === 'string') selectedPriority.value = q.priority
  if (typeof q.release === 'string') selectedRelease.value = q.release
  if (typeof q.q === 'string') searchText.value = q.q
}

onMounted(async () => {
  initFiltersFromQuery()
  await Promise.all([
    store.fetchList(project, {
      stage: selectedStage.value || undefined,
      status: selectedStatus.value || undefined,
      label: selectedLabel.value || undefined,
      type: selectedType.value || undefined,
      priority: selectedPriority.value || undefined,
      release: selectedRelease.value === '__unassigned__' ? '__unassigned__' : (selectedRelease.value || undefined),
      q: searchText.value || undefined,
      limit: 0,
      offset: undefined,
    }),
    store.fetchLabels(project),
    store.fetchPriorities(project),
    releasesStore.fetch(project),
  ])
})
</script>

<template>
  <div class="list-view">
    <div class="list-header">
      <h2 class="list-title">Artefacts</h2>
      <span class="list-count" v-if="!store.loading">{{ visibleItems.length }} total</span>
      <label class="toggle-label" v-if="!store.loading">
        <input
          type="checkbox"
          class="toggle-input"
          v-model="showCompleted"
        />
        <span class="toggle-text">Show completed</span>
      </label>
      <button class="btn-check-status" @click="showStatusPanel = !showStatusPanel">
        <ShieldCheck :size="15" />
        Check statuses
      </button>
      <button class="btn-new-idea" ref="newIdeaButtonEl" @click="openBrainDump('idea')">
        <MessageSquarePlus :size="15" />
        New Idea
      </button>
      <button class="btn-new-defect" @click="openBrainDump('defect')">
        <Bug :size="15" />
        New Defect
      </button>
      <button class="btn-new-docs" @click="openBrainDump('doc')">
        <BookOpen :size="15" />
        New Docs
      </button>
    </div>

    <BrainDumpModal
      v-if="showBrainDump"
      :project="project"
      :artifact-type="brainDumpType"
      @close="onBrainDumpClose"
      @created="onBrainDumpCreated"
    />

    <div v-if="showStatusPanel" class="status-panel-wrap">
      <StatusCheckPanel :project="project" @close="showStatusPanel = false" />
    </div>

    <div class="filter-bar">
      <TextFilter ref="textFilterRef" :model-value="searchText" @update:model-value="onSearchText" />
      <select v-model="selectedStage" @change="applyFilters">
        <option value="">All stages</option>
        <option v-for="s in stageOptions.slice(1)" :key="s" :value="s">{{ s }}</option>
      </select>
      <select v-model="selectedStatus" @change="applyFilters">
        <option value="">All statuses</option>
        <option v-for="s in statusOptions.slice(1)" :key="s" :value="s">{{ s }}</option>
      </select>
      <select v-model="selectedType" @change="applyFilters">
        <option value="">All types</option>
        <option v-for="t in typeOptions.slice(1)" :key="t" :value="t">{{ t }}</option>
      </select>
      <select v-model="selectedLabel" @change="applyFilters" v-if="store.labels.length">
        <option value="">All labels</option>
        <option v-for="l in store.labels" :key="l" :value="l">{{ l }}</option>
      </select>
      <select v-model="selectedPriority" @change="applyFilters" v-if="store.priorities.length">
        <option value="">All priorities</option>
        <option v-for="p in store.priorities" :key="p" :value="p">{{ p }}</option>
      </select>
      <label class="sr-only" for="release-filter">Release</label>
      <select id="release-filter" v-model="selectedRelease" @change="applyFilters">
        <option value="">All releases</option>
        <option v-for="r in releasesStore.releases" :key="r.id" :value="r.name">{{ r.name }}</option>
        <option value="__unassigned__">Unassigned</option>
      </select>
      <button class="btn-ghost" @click="resetFilters">Reset</button>
    </div>

    <div class="table-wrap">
      <div v-if="store.loading" class="state-msg">Loading…</div>
      <div v-else-if="visibleItems.length === 0" class="state-msg">No artifacts found.</div>
      <table v-else class="artifact-table">
        <thead>
          <tr>
            <SortHeader label="Path" column="title" :sort-column="sortColumn" :sort-direction="sortDirection" :sortable="true" @toggle="onToggleSort" />
            <SortHeader label="Stage" column="stage" :sort-column="sortColumn" :sort-direction="sortDirection" :sortable="true" @toggle="onToggleSort" />
            <SortHeader label="Status" column="status" :sort-column="sortColumn" :sort-direction="sortDirection" :sortable="true" @toggle="onToggleSort" />
            <SortHeader label="Priority" column="priority" :sort-column="sortColumn" :sort-direction="sortDirection" :sortable="true" @toggle="onToggleSort" />
            <SortHeader label="Release" column="release" :sort-column="sortColumn" :sort-direction="sortDirection" :sortable="true" @toggle="onToggleSort" />
            <SortHeader label="Type" column="type" :sort-column="sortColumn" :sort-direction="sortDirection" :sortable="true" @toggle="onToggleSort" />
            <SortHeader label="Runs" column="agent_run_count" title="Agent Run Count" :sort-column="sortColumn" :sort-direction="sortDirection" :sortable="true" @toggle="onToggleSort" />
            <SortHeader label="Created" column="created" :sort-column="sortColumn" :sort-direction="sortDirection" :sortable="true" @toggle="onToggleSort" />
            <SortHeader label="Modified" column="mtime" :sort-column="sortColumn" :sort-direction="sortDirection" :sortable="true" @toggle="onToggleSort" />
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="row in paginatedItems"
            :key="row.path"
            class="artifact-row"
            @click="openArtifact(row.path)"
            tabindex="0"
            @keydown.enter="openArtifact(row.path)"
          >
            <td class="cell-path">
              <span class="cell-path-title-row">
                <span class="artifact-title" v-html="highlightMatch(row.title || row.slug)" />
                <span
                  v-if="row.type === 'defect' && row.frontmatter?.labels?.includes('auto-filed')"
                  class="auto-filed-badge"
                  title="Auto-filed by test-runner agent"
                  aria-label="Auto-filed by test-runner agent"
                >
                  <Bot :size="11" />
                </span>
              </span>
              <span
                v-if="row.active_agent_status"
                class="agent-status-pill"
                :data-status="row.active_agent_status"
              >{{ row.active_agent_status === 'running' ? 'Agent Running' : 'Work Queued' }}</span>
              <span class="artifact-path">{{ row.path }}</span>
            </td>
            <td><span class="stage-tag">{{ row.stage }}</span></td>
            <td><span class="badge" :data-status="row.status">{{ row.status }}</span></td>
            <td class="cell-priority">
              <span
                v-if="row.frontmatter?.priority"
                class="priority-pill"
                :class="`priority-${row.frontmatter.priority}`"
              >{{ row.frontmatter.priority }}</span>
              <span v-else class="muted">—</span>
            </td>
            <td class="cell-release muted">{{ row.frontmatter?.release || '—' }}</td>
            <td class="muted">{{ row.type }}</td>
            <td class="cell-runs">{{ row.agent_run_count }}</td>
            <td class="muted cell-date">
              <span :title="formatFullDateTime(row.created)">{{ formatShortDate(row.created) }}</span>
            </td>
            <td class="muted cell-date">
              <span :title="formatFullDateTime(row.mtime)">{{ formatShortDate(row.mtime) }}</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <TablePagination
      v-if="!store.loading"
      :total-items="visibleItems.length"
      :current-page="currentPage"
      :page-size="pageSize"
      @update:current-page="setPage"
      @update:page-size="setPageSize"
    />
  </div>
</template>

<style scoped>
.list-view {
  position: relative;
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}
.list-header {
  display: flex;
  align-items: baseline;
  gap: var(--space-3);
  padding: var(--space-4) var(--space-6);
  border-bottom: 1px solid var(--color-border);
}
.list-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}
.list-count {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.toggle-label {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  cursor: pointer;
  user-select: none;
}
.toggle-input {
  accent-color: var(--color-accent);
  width: 14px;
  height: 14px;
  cursor: pointer;
}
.toggle-input:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}
.toggle-text {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.btn-new-defect {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-3);
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text-muted);
  cursor: pointer;
}
.btn-new-defect:hover { background: var(--color-surface); color: var(--color-text); }
.btn-new-defect:focus-visible { outline: 2px solid var(--color-accent); outline-offset: 2px; }
.btn-new-docs {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-3);
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text-muted);
  cursor: pointer;
}
.btn-new-docs:hover { background: var(--color-surface); color: var(--color-text); }
.btn-new-docs:focus-visible { outline: 2px solid var(--color-accent); outline-offset: 2px; }
.btn-new-idea {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  margin-left: auto;
  padding: var(--space-1) var(--space-3);
  background: var(--color-accent);
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-new-idea:hover { opacity: 0.88; }
.btn-new-idea:focus-visible { outline: 2px solid var(--color-accent); outline-offset: 2px; }
.filter-bar {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-wrap: wrap;
}
.filter-bar select {
  padding: var(--space-1) var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: var(--text-sm);
  cursor: pointer;
}
.btn-ghost {
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  padding: var(--space-1) var(--space-3);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  cursor: pointer;
}
.btn-ghost:hover:not(:disabled) { background: var(--color-surface); color: var(--color-text); }
.btn-ghost:disabled { opacity: 0.4; cursor: not-allowed; }
.table-wrap {
  flex: 1;
  overflow-y: auto;
}
.state-msg {
  padding: var(--space-8) var(--space-6);
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.artifact-table {
  width: 100%;
  border-collapse: collapse;
}
.artifact-table th {
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
.artifact-row {
  cursor: pointer;
  border-bottom: 1px solid var(--color-border);
}
.artifact-row:hover { background: var(--color-surface); }
.artifact-row:focus { outline: 2px solid var(--color-accent); outline-offset: -2px; }
.artifact-table td {
  padding: var(--space-2) var(--space-4);
  vertical-align: middle;
}
.cell-path {
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.cell-path-title-row {
  display: flex;
  align-items: center;
  gap: var(--space-1);
}
.artifact-title {
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
}
.auto-filed-badge {
  display: inline-flex;
  align-items: center;
  color: var(--color-text-muted);
  flex-shrink: 0;
  opacity: 0.7;
  cursor: default;
}
.artifact-path {
  font-size: 11px;
  color: var(--color-text-muted);
  font-family: monospace;
}
.stage-tag {
  font-size: 11px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 4px;
  padding: 1px 6px;
  white-space: nowrap;
  color: var(--color-text);
}
.badge {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 500;
  background: var(--color-border);
  color: var(--color-text);
  white-space: nowrap;
}
.badge[data-status="done"]          { background: var(--badge-done-bg);          color: var(--badge-done-text); }
.badge[data-status="approved"]      { background: var(--badge-approved-bg);      color: var(--badge-approved-text); }
.badge[data-status="in-progress"]   { background: var(--badge-in-progress-bg);   color: var(--badge-in-progress-text); }
.badge[data-status="in-development"]{ background: var(--badge-in-dev-bg);        color: var(--badge-in-dev-text); }
.badge[data-status="in-qa"]         { background: var(--badge-in-qa-bg);         color: var(--badge-in-qa-text); }
.badge[data-status="blocked"]       { background: var(--badge-blocked-bg);       color: var(--badge-blocked-text); }
.badge[data-status="rejected"]      { background: var(--badge-rejected-bg);      color: var(--badge-rejected-text); }
.badge[data-status="clarifying"]   { background: var(--badge-clarifying-bg);    color: var(--badge-clarifying-text); }
.badge[data-status="planning"]     { background: var(--badge-planning-bg);      color: var(--badge-planning-text); }
.muted { color: var(--color-text-muted); font-size: var(--text-sm); }
.cell-date { white-space: nowrap; }
.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}
.priority-pill {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 500;
  white-space: nowrap;
  text-transform: capitalize;
}
.priority-critical { background: #fee2e2; color: #b91c1c; }
.priority-high     { background: #ffedd5; color: #c2410c; }
.priority-normal   { background: #dbeafe; color: #1d4ed8; }
.priority-low      { background: var(--color-surface); color: var(--color-text-muted); border: 1px solid var(--color-border); }
.cell-priority { white-space: nowrap; min-width: 72px; }
.agent-status-pill {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 500;
  white-space: nowrap;
  margin-left: var(--space-2);
}
.agent-status-pill[data-status="running"] {
  background: var(--badge-in-progress-bg);
  color: var(--badge-in-progress-text);
  animation: pulse 1.8s ease-in-out infinite;
}
.agent-status-pill[data-status="queued"] {
  background: var(--badge-planning-bg);
  color: var(--badge-planning-text);
}
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50%       { opacity: 0.55; }
}
.cell-release  { white-space: nowrap; min-width: 80px; max-width: 140px; overflow: hidden; text-overflow: ellipsis; }
.cell-runs {
  text-align: right;
  font-variant-numeric: tabular-nums;
  min-width: 56px;
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
/* Priority = 4th column, Release = 5th column — hide on narrow viewports */
@media (max-width: 1023px) {
  .artifact-table thead tr th:nth-child(4),
  .artifact-table thead tr th:nth-child(5),
  .cell-priority,
  .cell-release { display: none; }
}
/* Runs column hidden on narrow viewports */
@media (max-width: 767px) {
  .artifact-table thead tr th:nth-child(7),
  .cell-runs { display: none; }
}
.btn-check-status {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-3);
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text-muted);
  cursor: pointer;
}
.btn-check-status:hover { background: var(--color-surface); color: var(--color-text); }
.btn-check-status:focus-visible { outline: 2px solid var(--color-accent); outline-offset: 2px; }
.status-panel-wrap {
  position: absolute;
  top: var(--space-3);
  right: var(--space-3);
  z-index: 50;
  width: 380px;
  max-height: calc(100% - var(--space-6));
  overflow: hidden;
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-lg);
  border-radius: var(--radius-lg);
}
</style>
