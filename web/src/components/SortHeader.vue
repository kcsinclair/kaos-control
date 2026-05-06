<script setup lang="ts">
import { computed } from 'vue'
import { ArrowUp, ArrowDown, ArrowUpDown } from 'lucide-vue-next'
import type { SortDirection } from '@/composables/useSortableTable'

const props = defineProps<{
  /** Display text for the column header */
  label: string
  /** Column key this header controls */
  column: string
  /** Currently active sort column (from composable) */
  sortColumn: string | null
  /** Current sort direction (from composable) */
  sortDirection: SortDirection
  /** Whether this column is sortable. Non-sortable columns render as plain <th>. */
  sortable?: boolean
}>()

const emit = defineEmits<{
  (e: 'toggle', column: string): void
}>()

const isActive = computed(() => props.sortable && props.sortColumn === props.column)

function handleToggle() {
  if (props.sortable) emit('toggle', props.column)
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    handleToggle()
  }
}
</script>

<template>
  <th
    v-if="sortable"
    class="sort-th sort-th--sortable"
    :class="{ 'sort-th--active': isActive }"
    tabindex="0"
    role="columnheader"
    :aria-sort="isActive ? (sortDirection === 'asc' ? 'ascending' : 'descending') : 'none'"
    @click="handleToggle"
    @keydown="handleKeydown"
  >
    <span class="sort-th__inner">
      {{ label }}
      <ArrowUp v-if="isActive && sortDirection === 'asc'" :size="12" class="sort-icon sort-icon--active" />
      <ArrowDown v-else-if="isActive && sortDirection === 'desc'" :size="12" class="sort-icon sort-icon--active" />
      <ArrowUpDown v-else :size="12" class="sort-icon sort-icon--inactive" />
    </span>
  </th>
  <th v-else>{{ label }}</th>
</template>

<style scoped>
.sort-th--sortable {
  cursor: pointer;
  user-select: none;
}
.sort-th--sortable:hover .sort-icon--inactive {
  color: var(--color-text);
}
.sort-th--sortable:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: -2px;
}
.sort-th__inner {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.sort-icon {
  flex-shrink: 0;
  vertical-align: middle;
}
.sort-icon--inactive {
  color: var(--color-text-muted);
  opacity: 0.5;
}
.sort-icon--active {
  color: var(--color-accent);
}
.sort-th--active {
  color: var(--color-text);
}
</style>
