<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, defineAsyncComponent } from 'vue'
import { useWebSocket } from '@/composables/useWebSocket'
import * as releasesApi from '@/api/releases'
import type { GraphNode, GraphEdge, GraphData } from '@/types/api'
import ForceGraph3D from '@/components/graph/ForceGraph3D.vue'

// Lazy-load 2D graph to avoid increasing 3D chunk
const Graph2DView = defineAsyncComponent(
  () => import('@/components/graph/Graph2DView.vue')
)

const props = defineProps<{
  project: string
}>()

const emit = defineEmits<{
  /** Emitted when a non-Backlog release node is clicked */
  releaseClick: [releaseId: number]
  /** Emitted when an idea or defect artifact node is clicked */
  artifactClick: [node: GraphNode, edges: GraphEdge[]]
}>()

const view = ref<'3d' | '2d'>('3d')
const loading = ref(true)
const error = ref<string | null>(null)
const rawData = ref<GraphData>({ nodes: [], edges: [] })

const allNodes = computed<GraphNode[]>(() => rawData.value.nodes)
const allEdges = computed<GraphEdge[]>(() => rawData.value.edges)

async function load() {
  loading.value = true
  error.value = null
  try {
    rawData.value = await releasesApi.getRoadmapGraph(props.project)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to load roadmap graph.'
  } finally {
    loading.value = false
  }
}

/** Handle a node click from either graph subcomponent. */
function handleNodeClick(node: GraphNode) {
  if (node.type === 'release') {
    // Backlog synthetic node — no action
    if (node.synthetic) return
    // Extract numeric release ID from "release:<id>"
    const parts = node.id.split(':')
    const id = parseInt(parts[1] ?? '', 10)
    if (!isNaN(id)) emit('releaseClick', id)
  } else {
    // Idea or defect artifact
    emit('artifactClick', node, allEdges.value)
  }
}

onMounted(load)

// Refresh on release or artifact WS events
useWebSocket(props.project, 'release.created', load)
useWebSocket(props.project, 'release.updated', load)
useWebSocket(props.project, 'release.deleted', load)
useWebSocket(props.project, 'artifact.indexed', load)
</script>

<template>
  <div class="roadmap-graph">
    <div class="view-controls">
      <div class="view-toggle" role="group" aria-label="Graph view mode">
        <button
          class="toggle-btn"
          :class="{ active: view === '3d' }"
          @click="view = '3d'"
        >3D</button>
        <button
          class="toggle-btn"
          :class="{ active: view === '2d' }"
          @click="view = '2d'"
        >2D</button>
      </div>
    </div>

    <div v-if="loading" class="state-msg" role="status">Loading graph…</div>
    <div v-else-if="error" class="state-msg state-msg--error" role="alert">{{ error }}</div>
    <div v-else-if="allNodes.length === 0" class="state-msg">No data to display.</div>

    <template v-else>
      <ForceGraph3D
        v-if="view === '3d'"
        :nodes="allNodes"
        :edges="allEdges"
        dag-mode="lr"
        @node-click="handleNodeClick"
      />
      <Graph2DView
        v-else
        :nodes="allNodes"
        :edges="allEdges"
        :directed="true"
        :on-node-click="handleNodeClick"
      />
    </template>
  </div>
</template>

<style scoped>
.roadmap-graph {
  flex: 1;
  position: relative;
  overflow: hidden;
  background: #0f172a;
}
.view-controls {
  position: absolute;
  top: var(--space-3);
  right: var(--space-3);
  z-index: 100;
}
.view-toggle {
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
.state-msg {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--text-sm);
  color: rgba(241, 245, 249, 0.5);
}
.state-msg--error { color: #fca5a5; }
</style>
