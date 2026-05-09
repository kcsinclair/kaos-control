<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import {
  ArrowRightLeft,
  FilePlus,
  Play,
  CheckCircle,
  XCircle,
  Bug,
  GitCommit,
} from 'lucide-vue-next'
import type { Component } from 'vue'

defineProps<{
  activeTypes: Set<string>
}>()

const emit = defineEmits<{
  toggle: [type: string, enabled: boolean]
}>()

interface FilterChip {
  type: string
  label: string
  icon: Component
}

const CHIPS: FilterChip[] = [
  { type: 'status_transition', label: 'Transitions',   icon: ArrowRightLeft },
  { type: 'artifact_created',  label: 'Created',       icon: FilePlus },
  { type: 'agent_started',     label: 'Agent Start',   icon: Play },
  { type: 'agent_finished',    label: 'Agent Done',    icon: CheckCircle },
  { type: 'agent_failed',      label: 'Agent Failed',  icon: XCircle },
  { type: 'defect_raised',     label: 'Defects',       icon: Bug },
  { type: 'git_committed',     label: 'Commits',       icon: GitCommit },
]

function toggle(type: string, active: boolean) {
  emit('toggle', type, !active)
}
</script>

<template>
  <div class="filter-bar" role="toolbar" aria-label="Filter events">
    <button
      v-for="chip in CHIPS"
      :key="chip.type"
      class="filter-chip"
      :class="{ 'filter-chip--active': activeTypes.has(chip.type) }"
      :aria-pressed="activeTypes.has(chip.type)"
      @click="toggle(chip.type, activeTypes.has(chip.type))"
    >
      <component :is="chip.icon" :size="14" />
      <span>{{ chip.label }}</span>
    </button>
  </div>
</template>

<style scoped>
.filter-bar {
  display: flex;
  flex-wrap: nowrap;
  gap: var(--space-2);
  overflow-x: auto;
  padding-bottom: var(--space-1);
  scrollbar-width: thin;
}

.filter-chip {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-3);
  border-radius: var(--radius-full);
  border: 1px solid var(--color-border);
  background: var(--color-surface);
  color: var(--color-text-muted);
  font-size: var(--text-xs);
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: background 0.12s, color 0.12s, border-color 0.12s;
  flex-shrink: 0;
}

.filter-chip:hover {
  background: var(--color-surface-hover, var(--color-sidebar-hover));
  color: var(--color-text);
}

.filter-chip--active {
  background: color-mix(in srgb, var(--color-primary) 15%, transparent);
  border-color: var(--color-primary);
  color: var(--color-primary);
}
</style>
