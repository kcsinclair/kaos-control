<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { useReleasesStore } from '@/stores/releases'
import GanttChart from '@/components/releases/GanttChart.vue'
import ReleaseFormModal from '@/components/releases/ReleaseFormModal.vue'
import ReleaseDeleteModal from '@/components/releases/ReleaseDeleteModal.vue'
import ReleaseDetailModal from '@/components/releases/ReleaseDetailModal.vue'
import * as releasesApi from '@/api/releases'
import type { Release, ReleaseDetail } from '@/types/release'

const route = useRoute()
const project = route.params.project as string

const store = useReleasesStore()

type Granularity = 'week' | 'month' | 'quarter' | 'half-year' | 'year'
const granularity = ref<Granularity>('month')
const viewMode = ref<'gantt' | 'graph'>('gantt')

const showCreateModal = ref(false)
const editRelease = ref<Release | null>(null)
const deleteRelease = ref<Release | null>(null)
const deleteArtifactCount = ref(0)
const detailReleaseId = ref<number | null>(null)

// Cache for release details (summary badge counts)
const releaseDetails = ref<Map<number, ReleaseDetail>>(new Map())

async function loadDetails() {
  for (const r of store.releases) {
    if (!releaseDetails.value.has(r.id)) {
      try {
        const detail = await releasesApi.getRelease(project, r.id)
        releaseDetails.value = new Map(releaseDetails.value).set(r.id, detail)
      } catch {
        // non-fatal; badge just won't show
      }
    }
  }
}

onMounted(async () => {
  await store.fetch(project)
  store.connectWs(project)
  loadDetails()
})

onUnmounted(() => {
  store.disconnectWs()
})

function onReleaseCreated(release: Release) {
  showCreateModal.value = false
  editRelease.value = null
  // Refresh detail cache for the new release
  releasesApi.getRelease(project, release.id).then((detail) => {
    releaseDetails.value = new Map(releaseDetails.value).set(release.id, detail)
  }).catch(() => {})
}

async function openDelete(releaseId: number) {
  detailReleaseId.value = null
  const release = store.byId(releaseId)
  if (!release) return
  deleteRelease.value = release
  try {
    const arts = await releasesApi.listReleaseArtifacts(project, releaseId)
    deleteArtifactCount.value = arts?.length ?? 0
  } catch {
    deleteArtifactCount.value = 0
  }
}

async function confirmDelete(reassignTo?: number) {
  if (!deleteRelease.value) return
  await store.remove(project, deleteRelease.value.id, reassignTo)
  deleteRelease.value = null
}

function openEdit(releaseId: number) {
  detailReleaseId.value = null
  editRelease.value = store.byId(releaseId) ?? null
}
</script>

<template>
  <div class="roadmap-view">
    <!-- Toolbar -->
    <div class="toolbar">
      <h2 class="page-title">Roadmap</h2>

      <div class="toolbar-controls">
        <!-- Granularity toggle (only when Gantt is active) -->
        <div v-if="viewMode === 'gantt'" class="segmented" role="group" aria-label="Granularity">
          <button
            v-for="g in (['week', 'month', 'quarter', 'half-year', 'year'] as const)"
            :key="g"
            class="seg-btn"
            :class="{ 'seg-btn--active': granularity === g }"
            @click="granularity = g"
          >{{ g }}</button>
        </div>

        <!-- View toggle -->
        <div class="segmented" role="group" aria-label="View mode">
          <button
            class="seg-btn"
            :class="{ 'seg-btn--active': viewMode === 'gantt' }"
            @click="viewMode = 'gantt'"
          >Gantt</button>
          <button
            class="seg-btn"
            :class="{ 'seg-btn--active': viewMode === 'graph' }"
            @click="viewMode = 'graph'"
          >Graph</button>
        </div>

        <button class="btn-primary" @click="showCreateModal = true">+ Create Release</button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="store.loading" class="state-msg">Loading releases…</div>

    <!-- Gantt view -->
    <GanttChart
      v-else-if="viewMode === 'gantt'"
      :releases="store.releases"
      :granularity="granularity"
      :project="project"
      :release-details="releaseDetails"
      @click-release="detailReleaseId = $event"
      @create="showCreateModal = true"
    />

    <!-- Graph view placeholder — implemented in Milestone 5 -->
    <div v-else class="state-msg">Graph view loading…</div>

    <!-- Create / Edit modal -->
    <ReleaseFormModal
      v-if="showCreateModal"
      :project="project"
      @saved="onReleaseCreated"
      @close="showCreateModal = false"
    />
    <ReleaseFormModal
      v-if="editRelease"
      :release="editRelease"
      :project="project"
      @saved="onReleaseCreated"
      @close="editRelease = null"
    />

    <!-- Detail modal -->
    <ReleaseDetailModal
      v-if="detailReleaseId !== null"
      :release-id="detailReleaseId"
      :project="project"
      @close="detailReleaseId = null"
      @edit="openEdit(detailReleaseId!)"
      @delete="openDelete(detailReleaseId!)"
    />

    <!-- Delete modal -->
    <ReleaseDeleteModal
      v-if="deleteRelease"
      :release="deleteRelease"
      :project="project"
      :artifact-count="deleteArtifactCount"
      @confirmed="confirmDelete"
      @close="deleteRelease = null"
    />
  </div>
</template>

<style scoped>
.roadmap-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}
.toolbar {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-4) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
  flex-wrap: wrap;
}
.page-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}
.toolbar-controls {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-left: auto;
  flex-wrap: wrap;
}
.segmented {
  display: flex;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.seg-btn {
  padding: var(--space-1) var(--space-3);
  background: none;
  border: none;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  cursor: pointer;
  transition: background 0.12s, color 0.12s;
  text-transform: capitalize;
  white-space: nowrap;
}
.seg-btn + .seg-btn {
  border-left: 1px solid var(--color-border);
}
.seg-btn--active {
  background: var(--color-accent);
  color: #fff;
}
.seg-btn:hover:not(.seg-btn--active) {
  background: var(--color-surface);
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
  white-space: nowrap;
}
.btn-primary:hover { opacity: 0.88; }
.state-msg {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
</style>
