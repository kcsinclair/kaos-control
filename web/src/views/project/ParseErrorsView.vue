<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed, ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '@/api/client'
import { usePagination } from '@/composables/usePagination'
import { useSortableTable } from '@/composables/useSortableTable'
import TablePagination from '@/components/common/TablePagination.vue'
import SortHeader from '@/components/SortHeader.vue'
import type { ParseErrorRow } from '@/types/api'

const route = useRoute()
const project = route.params.project as string

const errors = ref<ParseErrorRow[]>([])
const loading = ref(false)
const loadError = ref<string | null>(null)

const { currentPage, pageSize, sliceStart, sliceEnd, setPage, setPageSize } = usePagination({ queryPrefix: 'pe' })

const { sortColumn, sortDirection, sortedRows: sortedErrors, toggleSort } = useSortableTable(
  errors,
  {
    path:    { type: 'string' },
    message: { type: 'string' },
  },
)

const paginatedErrors = computed(() => sortedErrors.value.slice(sliceStart.value, sliceEnd.value))

// Reset to page 1 on sort change
watch([sortColumn, sortDirection], () => setPage(1))

async function load() {
  loading.value = true
  loadError.value = null
  try {
    const res = await api.get<{ errors: ParseErrorRow[] | null }>(
      `/p/${encodeURIComponent(project)}/parse-errors`
    )
    errors.value = res.errors ?? []
  } catch (e: unknown) {
    loadError.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="parse-errors-view">
    <div class="view-header">
      <h2 class="view-title">Parse Errors</h2>
      <button class="btn-ghost" @click="load" :disabled="loading" aria-label="Reload parse errors">
        Reload
      </button>
    </div>

    <div v-if="loading" class="state-msg" role="status" aria-live="polite">Loading…</div>
    <div v-else-if="loadError" class="state-msg error" role="alert">{{ loadError }}</div>
    <div v-else-if="!errors.length" class="state-msg success">
      No parse errors — all artifacts are clean.
    </div>

    <table v-else class="errors-table" aria-label="Parse errors">
      <thead>
        <tr>
          <SortHeader label="File"  column="path"    :sort-column="sortColumn" :sort-direction="sortDirection" :sortable="true" @toggle="toggleSort" />
          <SortHeader label="Error" column="message" :sort-column="sortColumn" :sort-direction="sortDirection" :sortable="true" @toggle="toggleSort" />
        </tr>
      </thead>
      <tbody>
        <tr v-for="err in paginatedErrors" :key="err.path" class="error-row">
          <td class="cell-path">{{ err.path }}</td>
          <td class="cell-msg">{{ err.message }}</td>
        </tr>
      </tbody>
    </table>

    <TablePagination
      v-if="!loading && errors.length > 0"
      :total-items="errors.length"
      :current-page="currentPage"
      :page-size="pageSize"
      @update:current-page="setPage"
      @update:page-size="setPageSize"
    />
  </div>
</template>

<style scoped>
.parse-errors-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}
.view-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.view-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}
.btn-ghost {
  padding: var(--space-1) var(--space-3);
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  cursor: pointer;
}
.btn-ghost:hover:not(:disabled) { background: var(--color-surface); }
.btn-ghost:disabled { opacity: 0.5; cursor: not-allowed; }
.state-msg {
  padding: var(--space-8) var(--space-6);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.state-msg.error { color: var(--color-error); }
.state-msg.success { color: var(--color-success); }
.errors-table {
  width: 100%;
  border-collapse: collapse;
  overflow-y: auto;
}
.errors-table th {
  position: sticky;
  top: 0;
  background: var(--color-bg);
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  padding: var(--space-2) var(--space-4);
  text-align: left;
  border-bottom: 1px solid var(--color-border);
  z-index: 1;
}
.error-row {
  border-bottom: 1px solid var(--color-border);
}
.errors-table td {
  padding: var(--space-3) var(--space-4);
  vertical-align: top;
  font-size: var(--text-sm);
}
.cell-path {
  font-family: monospace;
  color: var(--color-text-muted);
  width: 35%;
  word-break: break-all;
}
.cell-msg {
  color: var(--color-error);
}
</style>
