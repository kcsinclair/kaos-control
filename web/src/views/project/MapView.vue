<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, defineAsyncComponent, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useGraphData } from '@/composables/useGraphData'
import { useViewport } from '@/composables/useViewport'
import ForceGraph3D from '@/components/map/ForceGraph3D.vue'
import GraphFilters from '@/components/map/MapFilters.vue'
import GraphLegend from '@/components/map/MapLegend.vue'
import LabelModal from '@/components/map/LabelModal.vue'
import LayoutSelector from '@/components/map/LayoutSelector.vue'
import ArtifactModal from '@/components/artifact/ArtifactModal.vue'
import StatusCheckPanel from '@/components/artifact/StatusCheckPanel.vue'
import { useTextFilterShortcut } from '@/composables/useTextFilterShortcut'
import type { GraphNode } from '@/types/api'

// Lazy-load Cytoscape 2D so it doesn't increase the 3D chunk
const Graph2DView = defineAsyncComponent(
  () => import('@/components/map/Map2DView.vue')
)

const route = useRoute()
const project = route.params.project as string

const store = useGraphData(project)

const selectedNode = ref<GraphNode | null>(null)
const selectedLabelName = ref<string | null>(null)
const view = ref<'3d' | '2d'>('2d')
const showStatusPanel = ref(false)

const graphFiltersRef = ref<{ focus: () => void } | null>(null)
useTextFilterShortcut(graphFiltersRef)

function onNodeClick(node: GraphNode) {
  if (node.type === 'label') {
    // Label nodes have id like 'label::<name>'
    selectedLabelName.value = node.title || node.slug
    selectedNode.value = null
  } else {
    selectedNode.value = node
    selectedLabelName.value = null
  }
}

function closeModal() {
  selectedNode.value = null
  selectedLabelName.value = null
}

const { isMobile } = useViewport()
const mobileFiltersOpen = ref(false)

// Milestone 5: reset hideTerminal to true on every mount so navigating
// away and returning always starts with terminal nodes hidden.
onMounted(() => {
  store.hideTerminal = true
  store.hideTests = true
  // On phones, force 2D — the 3D force-graph (three.js / WebGL) is
  // genuinely unplayable on a small touchscreen: orbit controls compete
  // with browser scroll/zoom, node hits are imprecise, and the framerate
  // drops on mid-tier phones.
  if (isMobile.value) view.value = '2d'
})
</script>

<template>
  <div class="map-view">
    <GraphFilters
      ref="graphFiltersRef"
      :filter="store.filter"
      :unique-types="store.uniqueTypes"
      :unique-statuses="store.uniqueStatuses"
      :unique-lineages="store.uniqueLineages"
      :unique-labels="store.uniqueLabels"
      :unique-priorities="store.uniquePriorities"
      :node-count="store.augmentedNodes.length"
      :total-count="store.rawNodes.length"
      :show-label-nodes="store.showLabelNodes"
      :show-releases="store.showReleases"
      :hide-terminal="store.hideTerminal"
      :hide-tests="store.hideTests"
      :show-node-titles="store.showNodeTitles"
      :show-node-lineage="store.showNodeLineage"
      :search-text="store.searchText"
      :mobile-open="mobileFiltersOpen"
      @toggle="store.toggleFilterValue"
      @reset="store.setFilter({ types: [], statuses: [], lineages: [], labels: [], priorities: [] })"
      @toggle-label-nodes="store.toggleShowLabelNodes"
      @toggle-show-releases="store.toggleShowReleases(project)"
      @toggle-hide-terminal="store.toggleHideTerminal"
      @toggle-hide-tests="store.toggleHideTests"
      @toggle-show-node-titles="store.toggleShowNodeTitles"
      @toggle-show-node-lineage="store.toggleShowNodeLineage"
      @update:search-text="store.searchText = $event"
    />

    <div class="map-main">
      <div class="view-controls">
        <div v-if="!isMobile" class="view-toggle" role="group" aria-label="Map view mode">
          <button
            class="toggle-btn"
            :class="{ active: view === '3d' }"
            @click="view = '3d'"
            aria-pressed="view === '3d'"
          >3D</button>
          <button
            class="toggle-btn"
            :class="{ active: view === '2d' }"
            @click="view = '2d'"
            aria-pressed="view === '2d'"
          >2D</button>
        </div>
        <LayoutSelector v-if="view === '2d'" />
        <button class="check-status-btn" @click="showStatusPanel = true">
          Check all statuses
        </button>
        <button
          v-if="isMobile"
          class="check-status-btn mobile-filters-toggle"
          @click="mobileFiltersOpen = !mobileFiltersOpen"
          :aria-expanded="mobileFiltersOpen"
        >
          {{ mobileFiltersOpen ? 'Close filters' : 'Filters' }}
        </button>
      </div>

      <div v-if="store.loading" class="map-state" role="status" aria-live="polite">Loading map…</div>
      <div v-else-if="store.error" class="map-state error" role="alert">{{ store.error }}</div>
      <div v-else-if="store.rawNodes.length === 0" class="map-state">
        No artifacts indexed yet.
      </div>

      <template v-else>
        <ForceGraph3D
          v-if="view === '3d'"
          :nodes="store.augmentedNodes"
          :edges="store.augmentedEdges"
          :matched-node-ids="store.matchedNodeIds"
          :show-node-titles="store.showNodeTitles"
          :show-node-lineage="store.showNodeLineage"
          @node-click="onNodeClick"
        />
        <Graph2DView
          v-else
          :nodes="store.augmentedNodes"
          :edges="store.augmentedEdges"
          :on-node-click="onNodeClick"
          :matched-node-ids="store.matchedNodeIds"
        />
      </template>

      <div class="map-legend-wrap">
        <GraphLegend :show-label-nodes="store.showLabelNodes" :show-releases="store.showReleases" />
      </div>

      <div class="map-hint" v-if="!store.loading && store.rawNodes.length > 0">
        <template v-if="view === '3d'">Scroll to zoom · Drag to orbit · Click node to inspect</template>
        <template v-else>Scroll to zoom · Drag to pan · Click node to inspect</template>
      </div>

      <div v-if="showStatusPanel" class="map-status-panel-wrap">
        <StatusCheckPanel
          :project="project"
          @close="showStatusPanel = false"
        />
      </div>
    </div>
  </div>

  <ArtifactModal
    :node="selectedNode"
    :project="project"
    :edges="store.rawEdges"
    @close="closeModal"
    @navigate-artifact="(path) => { selectedNode = store.rawNodes.find(n => n.id === path) ?? null }"
  />

  <LabelModal
    v-if="selectedLabelName"
    :label-name="selectedLabelName"
    :project="project"
    :all-nodes="store.augmentedNodes"
    @close="closeModal"
  />
</template>

<style scoped>
.map-view {
  display: flex;
  height: 100%;
  overflow: hidden;
}
.map-main {
  position: relative;
  flex: 1;
  overflow: hidden;
  background: #0f172a;
}
.view-controls {
  position: absolute;
  top: var(--space-3);
  right: var(--space-3);
  z-index: 100;
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.view-toggle {
  display: flex;
  border: 1px solid rgba(255,255,255,0.15);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.check-status-btn {
  padding: 4px 10px;
  background: rgba(15,23,42,0.8);
  color: rgba(241,245,249,0.85);
  border: 1px solid rgba(255,255,255,0.15);
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.12s, color 0.12s;
}
.check-status-btn:hover {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: #fff;
}
.map-status-panel-wrap {
  position: absolute;
  top: var(--space-3);
  left: var(--space-3);
  z-index: 200;
  width: 380px;
  max-height: calc(100% - var(--space-6));
  overflow: hidden;
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-lg);
  border-radius: var(--radius-lg);
}
.toggle-btn {
  padding: 4px 10px;
  background: rgba(15,23,42,0.8);
  color: rgba(241,245,249,0.6);
  border: none;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.12s, color 0.12s;
}
.toggle-btn + .toggle-btn {
  border-left: 1px solid rgba(255,255,255,0.15);
}
.toggle-btn.active {
  background: var(--color-accent);
  color: #fff;
}
.toggle-btn:hover:not(.active) {
  background: rgba(255,255,255,0.08);
  color: #fff;
}
.map-state {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--text-sm);
  color: rgba(241, 245, 249, 0.5);
}
.map-state.error { color: #fca5a5; }
.map-legend-wrap {
  position: absolute;
  bottom: var(--space-4);
  right: var(--space-4);
  pointer-events: none;
}
.map-hint {
  position: absolute;
  bottom: var(--space-4);
  left: 50%;
  transform: translateX(-50%);
  font-size: 11px;
  color: rgba(241, 245, 249, 0.3);
  pointer-events: none;
  white-space: nowrap;
}
</style>
