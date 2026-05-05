<script setup lang="ts">
import type { Pipeline } from '@/api/devops'

const props = defineProps<{
  pipeline: Pipeline
}>()
</script>

<template>
  <div class="pipeline-card">
    <div class="card-header">
      <span class="pipeline-name">{{ props.pipeline.name }}</span>
      <span class="type-badge">{{ props.pipeline.type }}</span>
    </div>
    <div class="card-meta">
      <span class="step-count">{{ props.pipeline.steps.length }} step{{ props.pipeline.steps.length !== 1 ? 's' : '' }}</span>
    </div>
    <div class="card-actions">
      <button class="btn-run" disabled>Run</button>
    </div>
  </div>
</template>

<style scoped>
.pipeline-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-2);
}
.pipeline-name {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
  flex: 1;
  min-width: 0;
  word-break: break-word;
}
.type-badge {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 99px;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  background: var(--color-border);
  color: var(--color-text-muted);
  flex-shrink: 0;
}
.card-meta {
  font-size: var(--text-xs);
  color: var(--color-text-muted);
}
.step-count {
  font-size: var(--text-xs);
  color: var(--color-text-muted);
}
.card-actions {
  display: flex;
  gap: var(--space-2);
}
.btn-run {
  padding: var(--space-1) var(--space-3);
  background: var(--color-accent);
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-run:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.btn-run:not(:disabled):hover { opacity: 0.88; }
</style>
