<script setup lang="ts">
import { ref, defineAsyncComponent } from 'vue'
import { useRoute } from 'vue-router'
import { useGraphData } from '@/composables/useGraphData'
import ForceGraph3D from '@/components/graph/ForceGraph3D.vue'
import GraphFilters from '@/components/graph/GraphFilters.vue'
import GraphLegend from '@/components/graph/GraphLegend.vue'
import ArtifactModal from '@/components/artifact/ArtifactModal.vue'
import type { GraphNode } from '@/types/api'

// Lazy-load Cytoscape 2D so it doesn't increase the 3D chunk
const Graph2DView = defineAsyncComponent(
  () => import('@/components/graph/Graph2DView.vue')
)

const route = useRoute()
const project = route.params.project as string

const store = useGraphData(project)

const selectedNode = ref<GraphNode | null>(null)
const view = ref<'3d' | '2d'>('3d')

function onNodeClick(node: GraphNode) {
  selectedNode.value = node
}

function closeModal() {
  selectedNode.value = null
}
</script>

<template>
  <div class="graph-view">
    <GraphFilters
      :filter="store.filter"
      :unique-types="store.uniqueTypes"
      :unique-statuses="store.uniqueStatuses"
      :unique-lineages="store.uniqueLineages"
      :unique-labels="store.uniqueLabels"
      :unique-priorities="store.uniquePriorities"
      :node-count="store.filteredNodes.length"
      :total-count="store.rawNodes.length"
      @toggle="store.toggleFilterValue"
      @reset="store.setFilter({ types: [], statuses: [], lineages: [], labels: [], priorities: [] })"
    />

    <div class="graph-main">
      <div class="view-toggle" role="group" aria-label="Graph view mode">
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

      <div v-if="store.loading" class="graph-state" role="status" aria-live="polite">Loading graph…</div>
      <div v-else-if="store.error" class="graph-state error" role="alert">{{ store.error }}</div>
      <div v-else-if="store.rawNodes.length === 0" class="graph-state">
        No artifacts indexed yet.
      </div>

      <template v-else>
        <ForceGraph3D
          v-if="view === '3d'"
          :nodes="store.filteredNodes"
          :edges="store.filteredEdges"
          @node-click="onNodeClick"
        />
        <Graph2DView
          v-else
          :nodes="store.filteredNodes"
          :edges="store.filteredEdges"
          :on-node-click="onNodeClick"
        />
      </template>

      <div class="graph-legend-wrap">
        <GraphLegend />
      </div>

      <div class="graph-hint" v-if="!store.loading && store.rawNodes.length > 0">
        <template v-if="view === '3d'">Scroll to zoom · Drag to orbit · Click node to inspect</template>
        <template v-else>Scroll to zoom · Drag to pan · Click node to inspect</template>
      </div>
    </div>
  </div>

  <ArtifactModal
    :node="selectedNode"
    :project="project"
    :edges="store.rawEdges"
    @close="closeModal"
  />
</template>

<style scoped>
.graph-view {
  display: flex;
  height: 100%;
  overflow: hidden;
}
.graph-main {
  position: relative;
  flex: 1;
  overflow: hidden;
  background: #0f172a;
}
.view-toggle {
  position: absolute;
  top: var(--space-3);
  right: var(--space-3);
  z-index: 10;
  display: flex;
  border: 1px solid rgba(255,255,255,0.15);
  border-radius: var(--radius-sm);
  overflow: hidden;
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
.graph-state {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--text-sm);
  color: rgba(241, 245, 249, 0.5);
}
.graph-state.error { color: #fca5a5; }
.graph-legend-wrap {
  position: absolute;
  bottom: var(--space-4);
  right: var(--space-4);
  pointer-events: none;
}
.graph-hint {
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
