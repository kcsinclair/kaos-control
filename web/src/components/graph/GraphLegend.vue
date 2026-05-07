<script setup lang="ts">
import { computed } from 'vue'
import { useGraphTheme } from './graphConstants'

const props = defineProps<{
  showLabelNodes?: boolean
  showReleases?: boolean
}>()

const { palette, isDark } = useGraphTheme()

// Node types that belong to the release overlay
const RELEASE_NODE_TYPES = new Set(['release', 'backlog'])
// Edge kinds that belong to the release overlay
const RELEASE_EDGE_KINDS = new Set(['timeline', 'assigned'])

const nodeTypes = computed(() =>
  Object.entries(palette.value.nodeColors)
    .filter(([type]) => {
      if (type === 'label' && !props.showLabelNodes) return false
      if (RELEASE_NODE_TYPES.has(type) && !props.showReleases) return false
      return true
    })
    .map(([type, color]) => ({
      type,
      color,
      label: type.replace(/-/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase()),
    }))
)

const priorityEntries = computed(() =>
  Object.entries(palette.value.priorityColors).map(([level, color]) => ({
    level,
    color,
    label: level.charAt(0).toUpperCase() + level.slice(1),
  }))
)

const edgeKinds = computed(() =>
  Object.entries(palette.value.edgeColors)
    .filter(([kind]) => {
      if (kind === 'label' && !props.showLabelNodes) return false
      if (RELEASE_EDGE_KINDS.has(kind) && !props.showReleases) return false
      return true
    })
    .map(([kind, color]) => ({
      kind,
      color,
      label: kind.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase()),
    }))
)

const legendStyle = computed(() => ({
  background: isDark.value ? 'rgba(15, 23, 42, 0.85)' : 'rgba(255, 255, 255, 0.92)',
  color: palette.value.labelColor,
}))

const titleStyle = computed(() => ({
  color: isDark.value
    ? 'rgba(241, 245, 249, 0.5)'
    : 'rgba(15, 23, 42, 0.45)',
}))

const itemLabelStyle = computed(() => ({
  color: isDark.value
    ? 'rgba(241, 245, 249, 0.8)'
    : 'rgba(15, 23, 42, 0.75)',
}))
</script>

<template>
  <div class="legend" :style="legendStyle">
    <div class="legend-section">
      <div class="legend-title" :style="titleStyle">Nodes</div>
      <div v-for="item in nodeTypes" :key="item.type" class="legend-item">
        <span class="legend-dot" :style="{ background: item.color }" />
        <span class="legend-label" :style="itemLabelStyle">{{ item.label }}</span>
      </div>
    </div>
    <div class="legend-section">
      <div class="legend-title" :style="titleStyle">Priority</div>
      <div v-for="item in priorityEntries" :key="item.level" class="legend-item">
        <span class="legend-ring" :style="{ borderColor: item.color }" />
        <span class="legend-label" :style="itemLabelStyle">{{ item.label }}</span>
      </div>
    </div>
    <div class="legend-section">
      <div class="legend-title" :style="titleStyle">Edges</div>
      <div v-for="item in edgeKinds" :key="item.kind" class="legend-item">
        <span class="legend-line" :style="{ background: item.color }" />
        <span class="legend-label" :style="itemLabelStyle">{{ item.label }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.legend {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  border: 1px solid rgba(148, 163, 184, 0.15);
  border-radius: var(--radius-md);
  padding: var(--space-3);
  backdrop-filter: blur(4px);
}
.legend-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.legend-title {
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  margin-bottom: var(--space-1);
}
.legend-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.legend-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}
.legend-ring {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  border: 2.5px solid;
  background: transparent;
  flex-shrink: 0;
}
.legend-line {
  width: 16px;
  height: 2px;
  border-radius: 1px;
  flex-shrink: 0;
}
.legend-label {
  font-size: 11px;
}
</style>
