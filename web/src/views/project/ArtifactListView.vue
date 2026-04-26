<script setup lang="ts">
import { nextTick, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useArtifactsStore } from '@/stores/artifacts'
import { useWebSocket } from '@/composables/useWebSocket'
import BrainDumpModal from '@/components/idea/BrainDumpModal.vue'
import { useUiStore } from '@/stores/ui'
import { MessageSquarePlus, Bug } from 'lucide-vue-next'
import type { WsEvent } from '@/types/api'

const route = useRoute()
const router = useRouter()
const store = useArtifactsStore()
const ui = useUiStore()

const showBrainDump = ref(false)
const brainDumpType = ref<'idea' | 'defect'>('idea')
const newIdeaButtonEl = ref<HTMLButtonElement | null>(null)

function openBrainDump(type: 'idea' | 'defect' = 'idea') {
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

const stageOptions = ['', 'ideas', 'requirements', 'backend-plans', 'frontend-plans', 'test-plans', 'dev-plans', 'tests', 'prototypes', 'releases']
const statusOptions = ['', 'draft', 'in-progress', 'done', 'approved', 'blocked', 'rejected']
const typeOptions = ['', 'idea', 'requirement', 'plan-backend', 'plan-frontend', 'plan-test', 'test', 'prototype', 'defect']

const selectedStage = ref(store.filter.stage ?? '')
const selectedStatus = ref(store.filter.status ?? '')
const selectedLabel = ref(store.filter.label ?? '')
const selectedType = ref(store.filter.type ?? '')
const selectedPriority = ref(store.filter.priority ?? '')

function applyFilters() {
  store.fetchList(project, {
    stage: selectedStage.value || undefined,
    status: selectedStatus.value || undefined,
    label: selectedLabel.value || undefined,
    type: selectedType.value || undefined,
    priority: selectedPriority.value || undefined,
    offset: 0,
  })
}

function resetFilters() {
  selectedStage.value = ''
  selectedStatus.value = ''
  selectedLabel.value = ''
  selectedType.value = ''
  selectedPriority.value = ''
  applyFilters()
}

function prevPage() {
  if (store.filter.offset! <= 0) return
  store.fetchList(project, { offset: Math.max(0, (store.filter.offset ?? 0) - (store.filter.limit ?? 50)) })
}

function nextPage() {
  const limit = store.filter.limit ?? 50
  if ((store.filter.offset ?? 0) + limit >= store.total) return
  store.fetchList(project, { offset: (store.filter.offset ?? 0) + limit })
}

const currentPage = () => Math.floor((store.filter.offset ?? 0) / (store.filter.limit ?? 50)) + 1
const totalPages = () => Math.ceil(store.total / (store.filter.limit ?? 50))

function openArtifact(path: string) {
  router.push(`/p/${project}/artifacts/${path}`)
}

// Re-fetch when an artifact is indexed via WebSocket
useWebSocket(project, 'artifact.indexed', (_e: WsEvent) => {
  store.invalidate()
  store.fetchList(project)
})

onMounted(async () => {
  await Promise.all([
    store.fetchList(project),
    store.fetchLabels(project),
    store.fetchPriorities(project),
  ])
})
</script>

<template>
  <div class="list-view">
    <div class="list-header">
      <h2 class="list-title">Artifacts</h2>
      <span class="list-count" v-if="!store.loading">{{ store.total }} total</span>
      <button class="btn-new-defect" @click="openBrainDump('defect')">
        <Bug :size="15" />
        New Defect
      </button>
      <button class="btn-new-idea" ref="newIdeaButtonEl" @click="openBrainDump('idea')">
        <MessageSquarePlus :size="15" />
        New Idea
      </button>
    </div>

    <BrainDumpModal
      v-if="showBrainDump"
      :project="project"
      :artifact-type="brainDumpType"
      @close="onBrainDumpClose"
      @created="onBrainDumpCreated"
    />

    <div class="filter-bar">
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
      <button class="btn-ghost" @click="resetFilters">Reset</button>
    </div>

    <div class="table-wrap">
      <div v-if="store.loading" class="state-msg">Loading…</div>
      <div v-else-if="store.items.length === 0" class="state-msg">No artifacts found.</div>
      <table v-else class="artifact-table">
        <thead>
          <tr>
            <th>Path</th>
            <th>Stage</th>
            <th>Status</th>
            <th>Type</th>
            <th>Modified</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="row in store.items"
            :key="row.path"
            class="artifact-row"
            @click="openArtifact(row.path)"
            tabindex="0"
            @keydown.enter="openArtifact(row.path)"
          >
            <td class="cell-path">
              <span class="artifact-title">{{ row.title || row.slug }}</span>
              <span class="artifact-path">{{ row.path }}</span>
            </td>
            <td><span class="stage-tag">{{ row.stage }}</span></td>
            <td><span class="badge" :data-status="row.status">{{ row.status }}</span></td>
            <td class="muted">{{ row.type }}</td>
            <td class="muted cell-date">{{ new Date(row.mtime).toLocaleDateString() }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="pagination" v-if="store.total > (store.filter.limit ?? 50)">
      <button class="btn-ghost" :disabled="(store.filter.offset ?? 0) <= 0" @click="prevPage">← Prev</button>
      <span class="page-info">Page {{ currentPage() }} of {{ totalPages() }}</span>
      <button class="btn-ghost" :disabled="currentPage() >= totalPages()" @click="nextPage">Next →</button>
    </div>
  </div>
</template>

<style scoped>
.list-view {
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
.btn-new-defect {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  margin-left: auto;
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
.btn-new-idea {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
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
.artifact-title {
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
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
.muted { color: var(--color-text-muted); font-size: var(--text-sm); }
.cell-date { white-space: nowrap; }
.pagination {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-6);
  border-top: 1px solid var(--color-border);
  justify-content: center;
}
.page-info {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
</style>
