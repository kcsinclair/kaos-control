<script setup lang="ts">
import { computed } from 'vue'
import { NODE_COLORS, PRIORITY_COLORS, EDGE_COLORS } from './graphConstants'

const props = defineProps<{
  showLabelNodes?: boolean
}>()

const nodeTypes = computed(() =>
  Object.entries(NODE_COLORS)
    .filter(([type]) => type !== 'label' || props.showLabelNodes)
    .map(([type, color]) => ({
      type,
      color,
      label: type.replace(/-/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase()),
    }))
)

const priorityEntries = Object.entries(PRIORITY_COLORS).map(([level, color]) => ({
  level,
  color,
  label: level.charAt(0).toUpperCase() + level.slice(1),
}))

const edgeKinds = computed(() =>
  Object.entries(EDGE_COLORS)
    .filter(([kind]) => kind !== 'label' || props.showLabelNodes)
    .map(([kind, color]) => ({
      kind,
      color,
      label: kind.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase()),
    }))
)
</script>

<template>
  <div class="legend">
    <div class="legend-section">
      <div class="legend-title">Nodes</div>
      <div v-for="item in nodeTypes" :key="item.type" class="legend-item">
        <span class="legend-dot" :style="{ background: item.color }" />
        <span class="legend-label">{{ item.label }}</span>
      </div>
    </div>
    <div class="legend-section">
      <div class="legend-title">Priority</div>
      <div v-for="item in priorityEntries" :key="item.level" class="legend-item">
        <span class="legend-ring" :style="{ borderColor: item.color }" />
        <span class="legend-label">{{ item.label }}</span>
      </div>
    </div>
    <div class="legend-section">
      <div class="legend-title">Edges</div>
      <div v-for="item in edgeKinds" :key="item.kind" class="legend-item">
        <span class="legend-line" :style="{ background: item.color }" />
        <span class="legend-label">{{ item.label }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.legend {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  background: rgba(15, 23, 42, 0.85);
  border: 1px solid rgba(148, 163, 184, 0.15);
  border-radius: var(--radius-md);
  padding: var(--space-3);
  backdrop-filter: blur(4px);
  color: #f1f5f9;
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
  color: rgba(241, 245, 249, 0.5);
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
  color: rgba(241, 245, 249, 0.8);
}
</style>
