<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = withDefaults(defineProps<{
  totalItems: number
  currentPage?: number
  pageSize?: number
}>(), {
  currentPage: 1,
  pageSize: 25,
})

const emit = defineEmits<{
  'update:currentPage': [page: number]
  'update:pageSize': [size: number]
}>()

const PAGE_SIZE_OPTIONS = [10, 25, 50, 100]

const lastPage = computed(() => Math.max(1, Math.ceil(props.totalItems / props.pageSize)))

const rangeStart = computed(() =>
  props.totalItems === 0 ? 0 : (props.currentPage - 1) * props.pageSize + 1
)
const rangeEnd = computed(() => Math.min(props.currentPage * props.pageSize, props.totalItems))

const jumpValue = ref(String(props.currentPage))

watch(() => props.currentPage, (v) => {
  jumpValue.value = String(v)
})

function onPageSizeChange(e: Event) {
  const size = Number((e.target as HTMLSelectElement).value)
  emit('update:pageSize', size)
  emit('update:currentPage', 1)
}

function prevPage() {
  if (props.currentPage <= 1) return
  emit('update:currentPage', props.currentPage - 1)
}

function nextPage() {
  if (props.currentPage >= lastPage.value) return
  emit('update:currentPage', props.currentPage + 1)
}

function commitJump() {
  let n = parseInt(jumpValue.value, 10)
  if (isNaN(n)) n = props.currentPage
  n = Math.max(1, Math.min(n, lastPage.value))
  jumpValue.value = String(n)
  emit('update:currentPage', n)
}
</script>

<template>
  <div class="pagination-bar" role="navigation" aria-label="Table pagination">
    <div class="pagination-left">
      <label class="size-label" for="page-size-select">Rows per page</label>
      <select
        id="page-size-select"
        class="size-select"
        :value="pageSize"
        aria-label="Rows per page"
        @change="onPageSizeChange"
      >
        <option v-for="opt in PAGE_SIZE_OPTIONS" :key="opt" :value="opt">{{ opt }}</option>
      </select>
    </div>

    <span class="pagination-summary">
      <template v-if="totalItems === 0">No items</template>
      <template v-else>Showing {{ rangeStart }}–{{ rangeEnd }} of {{ totalItems }}</template>
    </span>

    <div class="pagination-controls">
      <button
        class="btn-page"
        :disabled="currentPage <= 1"
        aria-label="Previous page"
        @click="prevPage"
      >
        ← Prev
      </button>

      <label class="jump-label" for="page-jump-input">Page</label>
      <input
        id="page-jump-input"
        class="jump-input"
        type="number"
        :min="1"
        :max="lastPage"
        :value="jumpValue"
        :disabled="lastPage <= 1"
        aria-label="Jump to page"
        @input="jumpValue = ($event.target as HTMLInputElement).value"
        @change="commitJump"
        @keydown.enter.prevent="commitJump"
      />
      <span class="jump-of">of {{ lastPage }}</span>

      <button
        class="btn-page"
        :disabled="currentPage >= lastPage"
        aria-label="Next page"
        @click="nextPage"
      >
        Next →
      </button>
    </div>
  </div>
</template>

<style scoped>
.pagination-bar {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-3) var(--space-6);
  border-top: 1px solid var(--color-border);
  flex-wrap: wrap;
  flex-shrink: 0;
}

.pagination-left {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.size-label {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  white-space: nowrap;
}

.size-select {
  padding: var(--space-1) var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: var(--text-sm);
  cursor: pointer;
}

.size-select:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

.pagination-summary {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  white-space: nowrap;
  flex: 1;
  text-align: center;
}

.pagination-controls {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.btn-page {
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  padding: var(--space-1) var(--space-3);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  cursor: pointer;
  white-space: nowrap;
}

.btn-page:hover:not(:disabled) {
  background: var(--color-surface);
  color: var(--color-text);
}

.btn-page:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.btn-page:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

.jump-label {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  white-space: nowrap;
}

.jump-input {
  width: 56px;
  padding: var(--space-1) var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: var(--text-sm);
  text-align: center;
}

.jump-input:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.jump-input:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

/* Remove number input spinners */
.jump-input::-webkit-inner-spin-button,
.jump-input::-webkit-outer-spin-button {
  -webkit-appearance: none;
  margin: 0;
}
.jump-input[type=number] {
  -moz-appearance: textfield;
}

.jump-of {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  white-space: nowrap;
}

@media (max-width: 768px) {
  .pagination-bar {
    gap: var(--space-2);
    padding: var(--space-2) var(--space-4);
  }

  .pagination-summary {
    flex: none;
    order: -1;
    width: 100%;
    text-align: left;
  }
}
</style>
