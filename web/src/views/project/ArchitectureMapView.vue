<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { defineAsyncComponent } from 'vue'
import { useRoute } from 'vue-router'
import { useArchitectureMap } from '@/composables/useArchitectureMap'

// Lazy-load Cytoscape 2D so it doesn't increase the 3D chunk (mirrors MapView.vue)
const Graph2DView = defineAsyncComponent(
  () => import('@/components/map/Map2DView.vue')
)

const route = useRoute()
const project = route.params.project as string

const { nodes, edges, loading, error } = useArchitectureMap(project)

// Click-through to the underlying artifact lands in Milestone 6 (FR-7).
function onNodeClick() {}
</script>

<template>
  <div class="arch-map-view">
    <div v-if="loading" class="arch-map-state" role="status" aria-live="polite">Loading architecture map…</div>
    <div v-else-if="error" class="arch-map-state error" role="alert">{{ error }}</div>
    <div v-else-if="nodes.length === 0" class="arch-map-state">
      No architectures in the catalog yet.
    </div>
    <Graph2DView
      v-else
      :nodes="nodes"
      :edges="edges"
      :on-node-click="onNodeClick"
    />
  </div>
</template>

<style scoped>
.arch-map-view {
  position: relative;
  height: 100%;
  overflow: hidden;
  background: #0f172a;
}
.arch-map-state {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--text-sm);
  color: rgba(241, 245, 249, 0.5);
}
.arch-map-state.error { color: #fca5a5; }
</style>
