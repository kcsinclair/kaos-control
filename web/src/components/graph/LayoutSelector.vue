<script setup lang="ts">
import { computed } from 'vue'
import { useGraphStore } from '@/stores/graph'
import { LAYOUT_CONFIGS } from './layoutConfigs'

const store = useGraphStore()

const layouts = computed(() => Object.values(LAYOUT_CONFIGS))

const isAnimating = defineModel<boolean>('isAnimating', { default: false })

function onLayoutChange(e: Event) {
  const key = (e.target as HTMLSelectElement).value
  store.setLayout(key)
}
</script>

<template>
  <div class="layout-selector" aria-label="2D graph layout controls">
    <label class="layout-label" for="layout-select">Layout</label>
    <select
      id="layout-select"
      class="layout-select"
      :value="store.activeLayout"
      :disabled="isAnimating"
      aria-label="Select graph layout algorithm"
      @change="onLayoutChange"
    >
      <option v-for="layout in layouts" :key="layout.key" :value="layout.key">
        {{ layout.label }}
      </option>
    </select>
    <button
      class="directed-btn"
      :class="{ active: store.directed }"
      :aria-pressed="store.directed"
      aria-label="Toggle directed graph mode"
      :disabled="isAnimating"
      @click="store.toggleDirected()"
    >
      Directed
    </button>
  </div>
</template>

<style scoped>
.layout-selector {
  display: flex;
  align-items: center;
  gap: 6px;
}

.layout-label {
  font-size: 11px;
  font-weight: 600;
  color: rgba(241, 245, 249, 0.6);
  white-space: nowrap;
  user-select: none;
}

.layout-select {
  padding: 4px 6px;
  background: rgba(15, 23, 42, 0.8);
  color: rgba(241, 245, 249, 0.85);
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  outline: none;
  transition: border-color 0.12s;
  /* Prevent select from being too wide */
  max-width: 160px;
}

.layout-select:focus {
  border-color: var(--color-accent);
}

.layout-select:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.directed-btn {
  padding: 4px 10px;
  background: rgba(15, 23, 42, 0.8);
  color: rgba(241, 245, 249, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.12s, color 0.12s;
  white-space: nowrap;
}

.directed-btn.active {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: #fff;
}

.directed-btn:hover:not(.active):not(:disabled) {
  background: rgba(255, 255, 255, 0.08);
  color: #fff;
}

.directed-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
