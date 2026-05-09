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

/**
 * Augmented edge list: if the graph contains a synthetic `release:unscheduled`
 * terminus, add a `timeline` edge from the last node in the main scheduled
 * chain (Backlog → … → scheduledN) to `release:unscheduled` so the terminus
 * is visually attached to the end of the timeline rather than floating.
 *
 * The backend emits edges from each unscheduled release *to* the terminus but
 * does not connect the last scheduled release to it; we fill that gap here.
 */
const allEdges = computed<GraphEdge[]>(() => {
  const edges = rawData.value.edges
  const nodes = rawData.value.nodes

  const UNSCHEDULED_ID = 'release:unscheduled'

  // Only augment when the synthetic terminus exists.
  if (!nodes.some((n) => n.id === UNSCHEDULED_ID && n.synthetic)) return edges

  // Find the last node in the main timeline chain by looking at timeline edges
  // that do NOT target the unscheduled terminus.  The last node is one that
  // appears as a target but never as a source (it has no outgoing timeline
  // edge to another scheduled/backlog node).
  const chainSources = new Set<string>()
  const chainTargets = new Set<string>()
  for (const e of edges) {
    if (e.kind === 'timeline' && e.target !== UNSCHEDULED_ID) {
      chainSources.add(e.source)
      chainTargets.add(e.target)
    }
  }

  // Determine the tail: a chain target that is not itself a chain source.
  // If no scheduled releases exist the tail falls back to the Backlog root.
  const BACKLOG_ID = 'release:backlog'
  const tails = [...chainTargets].filter((id) => !chainSources.has(id))
  const lastChainNode = tails.length > 0 ? tails[0] : BACKLOG_ID

  // Skip augmentation if the edge already exists (defensive).
  if (edges.some((e) => e.source === lastChainNode && e.target === UNSCHEDULED_ID && e.kind === 'timeline')) {
    return edges
  }

  return [...edges, { source: lastChainNode, target: UNSCHEDULED_ID, kind: 'timeline' }]
})

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
