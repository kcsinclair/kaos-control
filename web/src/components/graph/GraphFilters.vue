<script setup lang="ts">
import type { GraphFilter } from '@/types/api'

const props = defineProps<{
  filter: GraphFilter
  uniqueTypes: string[]
  uniqueStatuses: string[]
  uniqueLineages: string[]
  uniqueLabels: string[]
  uniquePriorities: string[]
  nodeCount: number
  totalCount: number
  showLabelNodes: boolean
}>()

const emit = defineEmits<{
  toggle: [key: keyof GraphFilter, value: string]
  reset: []
  toggleLabelNodes: []
}>()

const isActive = (key: keyof GraphFilter, value: string) =>
  (props.filter[key] ?? []).includes(value)

const hasFilters = () =>
  (props.filter.types?.length ?? 0) +
  (props.filter.statuses?.length ?? 0) +
  (props.filter.lineages?.length ?? 0) +
  (props.filter.labels?.length ?? 0) +
  (props.filter.priorities?.length ?? 0) > 0
</script>

<template>
  <aside class="graph-filters">
    <div class="filter-header">
      <span class="filter-title">Filters</span>
      <button v-if="hasFilters()" class="btn-reset" @click="emit('reset')">Reset</button>
    </div>
    <div class="filter-count">
      {{ nodeCount }} / {{ totalCount }} nodes
    </div>

    <div class="filter-group">
      <label class="toggle-label">
        <input
          type="checkbox"
          class="toggle-input"
          :checked="showLabelNodes"
          @change="emit('toggleLabelNodes')"
        />
        <span class="toggle-text">Show label nodes</span>
      </label>
    </div>

    <div class="filter-group" v-if="uniqueTypes.length">
      <div class="group-label">Type</div>
      <div class="chip-list">
        <button
          v-for="t in uniqueTypes"
          :key="t"
          class="chip"
          :class="{ 'chip--active': isActive('types', t) }"
          @click="emit('toggle', 'types', t)"
        >{{ t }}</button>
      </div>
    </div>

    <div class="filter-group" v-if="uniqueStatuses.length">
      <div class="group-label">Status</div>
      <div class="chip-list">
        <button
          v-for="s in uniqueStatuses"
          :key="s"
          class="chip"
          :class="{ 'chip--active': isActive('statuses', s) }"
          @click="emit('toggle', 'statuses', s)"
        >{{ s }}</button>
      </div>
    </div>

    <div class="filter-group" v-if="uniqueLineages.length">
      <div class="group-label">Lineage</div>
      <div class="chip-list">
        <button
          v-for="l in uniqueLineages"
          :key="l"
          class="chip"
          :class="{ 'chip--active': isActive('lineages', l) }"
          @click="emit('toggle', 'lineages', l)"
        >{{ l }}</button>
      </div>
    </div>

    <div class="filter-group" v-if="uniqueLabels.length">
      <div class="group-label">Label</div>
      <div class="chip-list">
        <button
          v-for="l in uniqueLabels"
          :key="l"
          class="chip"
          :class="{ 'chip--active': isActive('labels', l) }"
          @click="emit('toggle', 'labels', l)"
        >{{ l }}</button>
      </div>
    </div>

    <div class="filter-group" v-if="uniquePriorities.length">
      <div class="group-label">Priority</div>
      <div class="chip-list">
        <button
          v-for="p in uniquePriorities"
          :key="p"
          class="chip"
          :class="{ 'chip--active': isActive('priorities', p) }"
          @click="emit('toggle', 'priorities', p)"
        >{{ p }}</button>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.graph-filters {
  width: 200px;
  flex-shrink: 0;
  background: var(--color-surface);
  border-right: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding: var(--space-4);
  overflow-y: auto;
}
.filter-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.filter-title {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
}
.btn-reset {
  background: none;
  border: none;
  padding: 0;
  font-size: 11px;
  color: var(--color-accent);
  cursor: pointer;
}
.btn-reset:hover { text-decoration: underline; }
.filter-count {
  font-size: 11px;
  color: var(--color-text-muted);
}
.filter-group {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.group-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--color-text-muted);
}
.chip-list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1);
}
.chip {
  padding: 2px 8px;
  border: 1px solid var(--color-border);
  border-radius: 99px;
  background: none;
  font-size: 11px;
  color: var(--color-text-muted);
  cursor: pointer;
  transition: background 0.1s, color 0.1s, border-color 0.1s;
}
.chip:hover { background: var(--color-border); color: var(--color-text); }
.chip--active {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: #fff;
}
.toggle-label {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  cursor: pointer;
  user-select: none;
}
.toggle-input {
  accent-color: var(--color-accent);
  width: 14px;
  height: 14px;
  cursor: pointer;
}
.toggle-text {
  font-size: 11px;
  color: var(--color-text-muted);
}
</style>
