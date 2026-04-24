<script setup lang="ts">
const nodeTypes = [
  { type: 'idea', color: '#f59e0b', label: 'Idea' },
  { type: 'requirement', color: '#3b82f6', label: 'Requirement' },
  { type: 'plan', color: '#8b5cf6', label: 'Plan' },
  { type: 'implementation', color: '#10b981', label: 'Implementation' },
  { type: 'test', color: '#06b6d4', label: 'Test' },
  { type: 'release', color: '#ef4444', label: 'Release' },
]

const edgeKinds = [
  { kind: 'parent', color: '#94a3b8', label: 'Parent' },
  { kind: 'depends_on', color: '#f97316', label: 'Depends on' },
  { kind: 'blocks', color: '#ef4444', label: 'Blocks' },
  { kind: 'related_to', color: '#64748b', label: 'Related to' },
]
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
