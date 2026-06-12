<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { AgentUsageGroupSummary } from '@/types/api'

type ModelRow = AgentUsageGroupSummary & { model: string }

const props = defineProps<{ rows: ModelRow[] }>()

type ColKey =
  | 'model'
  | 'run_count'
  | 'success_pct'
  | 'total_cost_usd'
  | 'mean_cost_usd'
  | 'total_input_cost_usd'
  | 'total_output_cost_usd'
  | 'mean_output_tokens_per_second'
  | 'mean_ttft_ms'
  | 'cache_hit_ratio'
  | 'metrics_unavailable_count'

interface ColDef {
  key: ColKey
  label: string
}

const COLS: ColDef[] = [
  { key: 'model',                       label: 'Model' },
  { key: 'run_count',                   label: 'Runs' },
  { key: 'success_pct',                 label: 'Success %' },
  { key: 'total_cost_usd',              label: 'Total cost' },
  { key: 'mean_cost_usd',               label: 'Mean cost/run' },
  { key: 'total_input_cost_usd',        label: 'Input cost' },
  { key: 'total_output_cost_usd',       label: 'Output cost' },
  { key: 'mean_output_tokens_per_second', label: 'Output tokens/s' },
  { key: 'mean_ttft_ms',               label: 'Mean TTFT' },
  { key: 'cache_hit_ratio',            label: 'Cache hit %' },
  { key: 'metrics_unavailable_count',  label: 'Metrics N/A' },
]

const sortCol = ref<ColKey>('total_cost_usd')
const sortDir = ref<'asc' | 'desc'>('desc')

function toggleSort(col: ColKey) {
  if (sortCol.value === col) {
    sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortCol.value = col
    sortDir.value = col === 'model' ? 'asc' : 'desc'
  }
}

function rowValue(row: ModelRow, col: ColKey): number | string | null {
  if (col === 'success_pct') {
    return row.run_count > 0 ? (row.success_count / row.run_count) * 100 : null
  }
  if (col === 'cache_hit_ratio') {
    return row.cache_hit_ratio != null ? row.cache_hit_ratio * 100 : null
  }
  return (row as unknown as Record<string, number | string | null>)[col] ?? null
}

const sortedRows = computed(() => {
  return [...props.rows].sort((a, b) => {
    const av = rowValue(a, sortCol.value)
    const bv = rowValue(b, sortCol.value)
    if (av == null && bv == null) return 0
    if (av == null) return 1
    if (bv == null) return -1
    const cmp = av < bv ? -1 : av > bv ? 1 : 0
    return sortDir.value === 'asc' ? cmp : -cmp
  })
})

function fmt(v: number | string | null, decimals = 2, prefix = '', suffix = ''): string {
  if (v == null) return '—'
  if (typeof v === 'string') return v
  return prefix + v.toFixed(decimals) + suffix
}

function fmtDuration(ms: number | null): string {
  if (ms == null) return '—'
  const totalSec = Math.round(ms / 1000)
  if (totalSec < 60) return `${totalSec}s`
  return `${Math.floor(totalSec / 60)}m ${totalSec % 60}s`
}

function displayValue(row: ModelRow, col: ColKey): string {
  if (col === 'model') return row.model
  if (col === 'run_count') return String(row.run_count)
  if (col === 'metrics_unavailable_count') return String(row.metrics_unavailable_count)
  if (col === 'success_pct') {
    const v = rowValue(row, col)
    return fmt(v, 1, '', '%')
  }
  if (col === 'total_cost_usd') return fmt(row.total_cost_usd, 2, '$')
  if (col === 'mean_cost_usd') return fmt(row.mean_cost_usd, 4, '$')
  if (col === 'total_input_cost_usd') return fmt(row.total_input_cost_usd, 2, '$')
  if (col === 'total_output_cost_usd') return fmt(row.total_output_cost_usd, 2, '$')
  if (col === 'mean_output_tokens_per_second') return fmt(row.mean_output_tokens_per_second, 1)
  if (col === 'mean_ttft_ms') return fmtDuration(row.mean_ttft_ms)
  if (col === 'cache_hit_ratio') {
    return row.cache_hit_ratio != null ? (row.cache_hit_ratio * 100).toFixed(1) + '%' : '—'
  }
  return '—'
}

function exportCsv() {
  const header = COLS.map((c) => c.label).join(',')
  const lines = sortedRows.value.map((row) =>
    COLS.map((c) => {
      const v = displayValue(row, c.key)
      return v.includes(',') ? `"${v}"` : v
    }).join(','),
  )
  const csv = [header, ...lines].join('\n')
  const blob = new Blob([csv], { type: 'text/csv' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'agent-usage-per-model.csv'
  a.click()
  URL.revokeObjectURL(url)
}
</script>

<template>
  <div class="per-model-table">
    <div class="table-header">
      <h2 class="table-title">Per-model summary</h2>
      <button class="btn-secondary" @click="exportCsv">Export CSV</button>
    </div>
    <div class="table-scroll">
      <table>
        <thead>
          <tr>
            <th
              v-for="col in COLS"
              :key="col.key"
              class="sortable-th"
              :class="{ 'th--active': sortCol === col.key }"
              :aria-sort="sortCol === col.key ? (sortDir === 'asc' ? 'ascending' : 'descending') : 'none'"
              @click="toggleSort(col.key)"
            >
              {{ col.label }}
              <span v-if="sortCol === col.key" class="sort-indicator" aria-hidden="true">
                {{ sortDir === 'asc' ? '▲' : '▼' }}
              </span>
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="sortedRows.length === 0">
            <td :colspan="COLS.length" class="empty-cell">No data</td>
          </tr>
          <tr v-for="row in sortedRows" :key="row.model">
            <td v-for="col in COLS" :key="col.key" :class="{ 'td--num': col.key !== 'model' }">
              {{ displayValue(row, col.key) }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.per-model-table {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}
.table-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4);
  border-bottom: 1px solid var(--color-border);
}
.table-title {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}
.table-scroll {
  overflow-x: auto;
}
table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-sm);
}
th {
  text-align: left;
  padding: var(--space-2) var(--space-3);
  font-size: var(--text-xs);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--color-text-muted);
  white-space: nowrap;
  border-bottom: 1px solid var(--color-border);
  background: var(--color-surface);
}
.sortable-th {
  cursor: pointer;
  user-select: none;
}
.sortable-th:hover {
  color: var(--color-text);
}
.th--active {
  color: var(--color-text);
}
.sort-indicator {
  margin-left: var(--space-1);
  font-size: 10px;
}
td {
  padding: var(--space-2) var(--space-3);
  border-bottom: 1px solid var(--color-border);
  color: var(--color-text);
  white-space: nowrap;
}
.td--num {
  text-align: right;
  font-variant-numeric: tabular-nums;
}
tr:last-child td {
  border-bottom: none;
}
tr:hover td {
  background: var(--color-sidebar-hover);
}
.empty-cell {
  text-align: center;
  color: var(--color-text-muted);
  padding: var(--space-6);
}
.btn-secondary {
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: var(--text-xs);
  padding: var(--space-1) var(--space-3);
  color: var(--color-text-muted);
  transition: background 0.12s, color 0.12s;
}
.btn-secondary:hover {
  background: var(--color-sidebar-hover);
  color: var(--color-text);
}
</style>
