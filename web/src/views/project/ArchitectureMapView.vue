<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { defineAsyncComponent, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useArchitectureMap } from '@/composables/useArchitectureMap'
import { useViewport } from '@/composables/useViewport'
import ForceGraph3D from '@/components/map/ForceGraph3D.vue'

// Lazy-load Cytoscape 2D so it doesn't increase the 3D chunk (mirrors MapView.vue)
const Graph2DView = defineAsyncComponent(
  () => import('@/components/map/Map2DView.vue')
)

const route = useRoute()
const project = route.params.project as string

const { nodes, edges, loading, error } = useArchitectureMap(project)

// Click-through to the underlying artifact lands in Milestone 6 (FR-7).
function onNodeClick() {}

// Last-used engine is a per-browser nicety, not part of the FR set.
const VIEW_STORAGE_KEY = 'kc:architecture-map:view'
const storedView = localStorage.getItem(VIEW_STORAGE_KEY)
const view = ref<'2d' | '3d'>(storedView === '3d' ? '3d' : '2d')

const { isMobile } = useViewport()

onMounted(() => {
  // On phones, force 2D — the 3D force-graph is unplayable on a small
  // touchscreen (mirrors MapView.vue).
  if (isMobile.value) view.value = '2d'
})

watch(view, (v) => {
  if (!isMobile.value) localStorage.setItem(VIEW_STORAGE_KEY, v)
})
</script>

<template>
  <div class="arch-map-view">
    <div class="view-controls">
      <div v-if="!isMobile" class="view-toggle" role="group" aria-label="Architecture map view mode">
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
    </div>

    <div v-if="loading" class="arch-map-state" role="status" aria-live="polite">Loading architecture map…</div>
    <div v-else-if="error" class="arch-map-state error" role="alert">{{ error }}</div>
    <div v-else-if="nodes.length === 0" class="arch-map-state">
      No architectures in the catalog yet.
    </div>

    <template v-else>
      <ForceGraph3D
        v-if="view === '3d'"
        :nodes="nodes"
        :edges="edges"
        @node-click="onNodeClick"
      />
      <Graph2DView
        v-else
        :nodes="nodes"
        :edges="edges"
        :on-node-click="onNodeClick"
      />
    </template>
  </div>
</template>

<style scoped>
.arch-map-view {
  position: relative;
  height: 100%;
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
