<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed } from 'vue'

export interface WizardStepperStep {
  key: string
  label: string
}

const props = defineProps<{
  steps: WizardStepperStep[]
  currentKey: string
}>()

const currentIndex = computed(() => props.steps.findIndex((s) => s.key === props.currentKey))

function stateFor(index: number): 'done' | 'current' | 'upcoming' {
  if (index < currentIndex.value) return 'done'
  if (index === currentIndex.value) return 'current'
  return 'upcoming'
}
</script>

<template>
  <ol class="wizard-stepper" role="list" aria-label="Architecture wizard progress">
    <li
      v-for="(step, index) in steps"
      :key="step.key"
      class="stepper-item"
      :class="`stepper-item--${stateFor(index)}`"
      :aria-current="stateFor(index) === 'current' ? 'step' : undefined"
    >
      <span class="stepper-marker" aria-hidden="true">
        <span v-if="stateFor(index) === 'done'">✓</span>
        <span v-else>{{ index + 1 }}</span>
      </span>
      <span class="stepper-label">{{ step.label }}</span>
    </li>
  </ol>
</template>

<style scoped>
.wizard-stepper {
  display: flex;
  align-items: center;
  list-style: none;
  margin: 0;
  padding: 0;
  gap: var(--space-2);
}
.stepper-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.stepper-item:not(:last-child)::after {
  content: '';
  width: 24px;
  height: 1px;
  background: var(--color-border);
  margin-left: var(--space-2);
}
.stepper-marker {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: var(--radius-full);
  border: 1px solid var(--color-border);
  font-size: 11px;
  font-weight: 600;
  flex-shrink: 0;
}
.stepper-item--current .stepper-marker {
  border-color: var(--color-accent);
  background: var(--color-accent);
  color: #fff;
}
.stepper-item--done .stepper-marker {
  border-color: var(--color-accent);
  color: var(--color-accent);
}
.stepper-item--current .stepper-label {
  color: var(--color-text);
  font-weight: 600;
}
.stepper-item--done .stepper-label {
  color: var(--color-text);
}
</style>
