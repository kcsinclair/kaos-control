<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useKanbanBoard } from '@/composables/useKanbanBoard'
import { useArtifactsStore } from '@/stores/artifacts'
import { useReleasesStore } from '@/stores/releases'
import { useUiStore } from '@/stores/ui'
import { useWebSocket } from '@/composables/useWebSocket'
import KanbanCard from '@/components/artifact/KanbanCard.vue'
import StatusCheckPanel from '@/components/artifact/StatusCheckPanel.vue'
import TextFilter from '@/components/TextFilter.vue'
import { useTextFilterShortcut } from '@/composables/useTextFilterShortcut'
import { ShieldCheck } from 'lucide-vue-next'
import type { WsEvent } from '@/types/api'

const route = useRoute()
const router = useRouter()
const project = route.params.project as string

const store = useArtifactsStore()
const releasesStore = useReleasesStore()
const uiStore = useUiStore()

const {
  loading,
  hasConfig,
  columns,
  cardFields,
  hideTerminal,
  searchText,
  refresh,
  applyFilters,
  reorderColumns,
  ageOf,
  staleArtifactPaths,
} = useKanbanBoard(project)

const showCompleted = ref(false)
watch(showCompleted, v => { hideTerminal.value = !v }, { immediate: true })
const showStatusPanel = ref(false)

const textFilterRef = ref<{ focus: () => void } | null>(null)
useTextFilterShortcut(textFilterRef)

const stageOptions = ['', 'ideas', 'requirements', 'backend-plans', 'frontend-plans', 'test-plans', 'dev-plans', 'tests', 'prototypes', 'defects', 'releases']
const statusOptions = ['', 'draft', 'clarifying', 'planning', 'in-development', 'in-qa', 'in-progress', 'done', 'approved', 'blocked', 'rejected', 'abandoned']
const typeOptions = ['', 'idea', 'requirement', 'plan-backend', 'plan-frontend', 'plan-test', 'test', 'prototype', 'defect']

const selectedStage = ref('')
const selectedStatus = ref('')
const selectedLabel = ref('')
const selectedType = ref('')
const selectedPriority = ref('')
const selectedRelease = ref('')

// Read release filter from URL on mount
function initFromRoute() {
  const r = route.query.release
  if (typeof r === 'string' && r) {
    selectedRelease.value = r
  }
}

// Sync release filter to URL
watch(selectedRelease, (val) => {
  const query = { ...route.query }
  if (val) {
    query.release = val
  } else {
    delete query.release
  }
  router.replace({ query })
})

function onFilterChange() {
  applyFilters({
    stage: selectedStage.value || undefined,
    status: selectedStatus.value || undefined,
    label: selectedLabel.value || undefined,
    type: selectedType.value || undefined,
    priority: selectedPriority.value || undefined,
    release: selectedRelease.value || undefined,
  })
}

function resetFilters() {
  selectedStage.value = ''
  selectedStatus.value = ''
  selectedLabel.value = ''
  selectedType.value = ''
  selectedPriority.value = ''
  selectedRelease.value = ''
  searchText.value = ''
  onFilterChange()
}

// Drag-to-reorder state
const dragSourceIndex = ref<number | null>(null)
const dragOverIndex = ref<number | null>(null)

function onDragStart(e: DragEvent, index: number) {
  dragSourceIndex.value = index
  e.dataTransfer!.effectAllowed = 'move'
  e.dataTransfer!.setData('text/plain', String(index))
}

function onDragOver(e: DragEvent, index: number) {
  e.preventDefault()
  e.dataTransfer!.dropEffect = 'move'
  dragOverIndex.value = index
}

function onDragEnter(_e: DragEvent, index: number) {
  dragOverIndex.value = index
}

function onDragLeave(_e: DragEvent) {
  // Only clear if leaving to outside any column header
}

function onDrop(_e: DragEvent, toIndex: number) {
  if (dragSourceIndex.value !== null && dragSourceIndex.value !== toIndex) {
    reorderColumns(dragSourceIndex.value, toIndex)
  }
  dragSourceIndex.value = null
  dragOverIndex.value = null
}

function onDragEnd() {
  dragSourceIndex.value = null
  dragOverIndex.value = null
}

// Re-fetch artifacts when any artifact is indexed (status may have changed).
// Debounce to coalesce bursts (e.g. an agent run that rewrites many files in
// rapid succession) — a single refresh after the burst is enough.
let refreshDebounce: ReturnType<typeof setTimeout> | null = null
useWebSocket(project, 'artifact.indexed', (_e: WsEvent) => {
  if (refreshDebounce) clearTimeout(refreshDebounce)
  refreshDebounce = setTimeout(() => {
    refreshDebounce = null
    refresh()
  }, 500)
})

onMounted(async () => {
  initFromRoute()
  await Promise.all([
    refresh(),
    store.fetchLabels(project),
    store.fetchPriorities(project),
    releasesStore.releases.length === 0 ? releasesStore.fetch(project) : Promise.resolve(),
  ])
  // Apply route-initialised release filter after data is loaded
  if (selectedRelease.value) onFilterChange()
})
</script>

<template>
  <div class="board-view">
    <div class="board-header">
      <h2 class="board-title">Board</h2>
      <label class="toggle-label">
        <input
          type="checkbox"
          class="toggle-input"
          v-model="showCompleted"
        />
        <span class="toggle-text">Show completed</span>
      </label>
      <label class="toggle-label">
        <input
          type="checkbox"
          class="toggle-input"
          v-model="uiStore.showTestsOnKanban"
        />
        <span class="toggle-text">Show Tests</span>
      </label>
      <button class="btn-check-status" @click="showStatusPanel = !showStatusPanel">
        <ShieldCheck :size="15" />
        Check statuses
      </button>
    </div>

    <div v-if="showStatusPanel" class="status-panel-wrap">
      <StatusCheckPanel :project="project" @close="showStatusPanel = false" />
    </div>

    <!-- Filter bar -->
    <div v-if="hasConfig" class="filter-bar">
      <TextFilter ref="textFilterRef" v-model="searchText" />
      <select v-model="selectedStage" @change="onFilterChange">
        <option value="">All stages</option>
        <option v-for="s in stageOptions.slice(1)" :key="s" :value="s">{{ s }}</option>
      </select>
      <select v-model="selectedStatus" @change="onFilterChange">
        <option value="">All statuses</option>
        <option v-for="s in statusOptions.slice(1)" :key="s" :value="s">{{ s }}</option>
      </select>
      <select v-model="selectedType" @change="onFilterChange">
        <option value="">All types</option>
        <option v-for="t in typeOptions.slice(1)" :key="t" :value="t">{{ t }}</option>
      </select>
      <select v-model="selectedLabel" @change="onFilterChange" v-if="store.labels.length">
        <option value="">All labels</option>
        <option v-for="l in store.labels" :key="l" :value="l">{{ l }}</option>
      </select>
      <select v-model="selectedPriority" @change="onFilterChange" v-if="store.priorities.length">
        <option value="">All priorities</option>
        <option v-for="p in store.priorities" :key="p" :value="p">{{ p }}</option>
      </select>
      <select v-model="selectedRelease" @change="onFilterChange">
        <option value="">All Releases</option>
        <option v-for="r in releasesStore.releases" :key="r.id" :value="r.name">{{ r.name }}</option>
        <option value="__unassigned__">Unassigned</option>
      </select>
      <button class="btn-ghost" @click="resetFilters">Reset</button>
    </div>

    <div v-if="loading" class="state-msg">Loading…</div>
    <div v-else-if="!hasConfig" class="state-msg">
      No Kanban configuration found. Add a <code>kanban</code> section to your project's config.yaml.
    </div>

    <!-- Board -->
    <div v-else class="board-columns">
      <div
        v-for="(col, colIndex) in columns"
        :key="col.name"
        class="board-column"
        :class="{ 'board-column--drag-over': dragOverIndex === colIndex }"
        role="region"
        :aria-label="col.name"
      >
        <div
          class="column-header"
          draggable="true"
          @dragstart="onDragStart($event, colIndex)"
          @dragover="onDragOver($event, colIndex)"
          @dragenter="onDragEnter($event, colIndex)"
          @dragleave="onDragLeave($event)"
          @drop="onDrop($event, colIndex)"
          @dragend="onDragEnd"
        >
          <span class="column-name">{{ col.name }}</span>
          <span class="column-count">{{ col.cards.length }}</span>
        </div>
        <div class="column-cards">
          <div v-if="col.cards.length === 0" class="column-empty">
            {{ searchText ? 'No matching items' : 'No artefacts' }}
          </div>
          <KanbanCard
            v-for="card in col.cards"
            :key="card.path"
            :artifact="card"
            :card-fields="cardFields"
            :age="ageOf(card)"
            :project="project"
            :is-stale="staleArtifactPaths.has(card.path)"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.board-view {
  position: relative;
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}
.board-header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-4) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.board-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
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
.filter-bar {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-wrap: wrap;
  flex-shrink: 0;
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
.btn-ghost:hover { background: var(--color-surface); color: var(--color-text); }
.state-msg {
  padding: var(--space-8) var(--space-6);
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.board-columns {
  display: flex;
  flex: 1;
  overflow-x: auto;
  overflow-y: hidden;
  gap: var(--space-4);
  padding: var(--space-4) var(--space-6);
  align-items: flex-start;
  /* Ensure horizontal scroll works on touch devices */
  -webkit-overflow-scrolling: touch;
}

@media (max-width: 767px) {
  .board-columns {
    padding: var(--space-3);
    gap: var(--space-3);
  }
  .board-column {
    min-width: 240px;
    max-width: 240px;
  }
  .filter-bar {
    padding: var(--space-2) var(--space-3);
  }
  .board-header {
    padding: var(--space-3) var(--space-3);
  }
}
.board-column {
  display: flex;
  flex-direction: column;
  min-width: 280px;
  max-width: 280px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  flex-shrink: 0;
  height: 100%;
  overflow: hidden;
  transition: border-color 0.12s;
}
.board-column--drag-over {
  border-color: var(--color-accent);
  box-shadow: 0 0 0 2px var(--color-accent);
}
.column-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) var(--space-3);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
  cursor: grab;
  user-select: none;
}
.column-header:active {
  cursor: grabbing;
}
.column-name {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
}
.column-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: var(--radius-full);
  background: var(--color-border);
  font-size: 11px;
  font-weight: 600;
  color: var(--color-text-muted);
}
.column-cards {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
}
.column-empty {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  text-align: center;
  padding: var(--space-4) 0;
}
.btn-check-status {
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
