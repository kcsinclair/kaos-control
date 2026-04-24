<script setup lang="ts">
import { ref } from 'vue'
import { useRoute } from 'vue-router'
import { useGraphData } from '@/composables/useGraphData'
import ForceGraph3D from '@/components/graph/ForceGraph3D.vue'
import GraphFilters from '@/components/graph/GraphFilters.vue'
import GraphLegend from '@/components/graph/GraphLegend.vue'
import ArtifactModal from '@/components/artifact/ArtifactModal.vue'
import type { GraphNode } from '@/types/api'

const route = useRoute()
const project = route.params.project as string

const store = useGraphData(project)

const selectedNode = ref<GraphNode | null>(null)

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
      :node-count="store.filteredNodes.length"
      :total-count="store.rawNodes.length"
      @toggle="store.toggleFilterValue"
      @reset="store.setFilter({ types: [], statuses: [], lineages: [] })"
    />

    <div class="graph-main">
      <div v-if="store.loading" class="graph-state">Loading graph…</div>
      <div v-else-if="store.error" class="graph-state error">{{ store.error }}</div>
      <div v-else-if="store.rawNodes.length === 0" class="graph-state">
        No artifacts indexed yet.
      </div>
      <ForceGraph3D
        v-else
        :nodes="store.filteredNodes"
        :edges="store.filteredEdges"
        @node-click="onNodeClick"
      />

      <div class="graph-legend-wrap">
        <GraphLegend />
      </div>

      <div class="graph-hint" v-if="!store.loading && store.rawNodes.length > 0">
        Scroll to zoom · Drag to orbit · Click node to inspect
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
