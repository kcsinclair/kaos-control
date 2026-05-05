<script setup lang="ts">
import { computed } from 'vue'
import type { StepState } from '@/stores/devops'

const props = defineProps<{
  step: StepState
  index: number
}>()

const emit = defineEmits<{
  (e: 'toggle-output'): void
}>()

const durationLabel = computed((): string | null => {
  if (props.step.durationMs != null) {
    const ms = props.step.durationMs
    if (ms < 1000) return `${ms}ms`
    return `${(ms / 1000).toFixed(1)}s`
  }
  return null
})
</script>

<template>
  <div class="step-row" :class="`step-row--${props.step.status}`">
    <span class="step-icon" :title="props.step.status">
      <span v-if="props.step.status === 'pending'" class="icon-pending">○</span>
      <span v-else-if="props.step.status === 'running'" class="icon-running">◉</span>
      <span v-else-if="props.step.status === 'passed'" class="icon-passed">✓</span>
      <span v-else-if="props.step.status === 'failed'" class="icon-failed">✗</span>
      <span v-else-if="props.step.status === 'cancelled'" class="icon-cancelled">⊘</span>
    </span>
    <span class="step-name">{{ props.step.name }}</span>
    <span v-if="durationLabel" class="step-duration">{{ durationLabel }}</span>
    <button
      v-if="props.step.output.length > 0"
      class="btn-toggle-output"
      @click="emit('toggle-output')"
    >output</button>
  </div>
</template>

<style scoped>
.step-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-1) 0;
  font-size: var(--text-xs);
}
.step-icon {
  width: 16px;
  text-align: center;
  flex-shrink: 0;
  font-size: 13px;
  line-height: 1;
}
.icon-pending  { color: var(--color-text-muted); }
.icon-running  {
  color: var(--color-accent);
  display: inline-block;
  animation: spin 1s linear infinite;
}
.icon-passed   { color: #22c55e; }
.icon-failed   { color: var(--color-error); }
.icon-cancelled { color: var(--color-text-muted); }

@keyframes spin {
  from { transform: rotate(0deg); }
  to   { transform: rotate(360deg); }
}

.step-name {
  flex: 1;
  color: var(--color-text);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.step-row--pending .step-name  { color: var(--color-text-muted); }
.step-row--failed .step-name   { color: var(--color-error); }

.step-duration {
  color: var(--color-text-muted);
  font-size: 10px;
  flex-shrink: 0;
}
.btn-toggle-output {
  padding: 0 var(--space-1);
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: 10px;
  color: var(--color-text-muted);
  cursor: pointer;
  flex-shrink: 0;
}
.btn-toggle-output:hover {
  background: var(--color-surface);
  color: var(--color-text);
}
</style>
