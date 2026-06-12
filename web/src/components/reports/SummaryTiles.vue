<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import type { AgentUsageSummary } from '@/types/api'

const props = defineProps<{ summary: AgentUsageSummary }>()

function fmt(n: number | null | undefined, decimals: number, suffix = ''): string {
  if (n == null) return '—'
  return n.toFixed(decimals) + suffix
}

function fmtCost(n: number | null | undefined): string {
  if (n == null) return '—'
  return '$' + n.toFixed(2)
}

function fmtDuration(ms: number | null | undefined): string {
  if (ms == null) return '—'
  const totalSec = Math.round(ms / 1000)
  if (totalSec < 60) return `${totalSec}s`
  const m = Math.floor(totalSec / 60)
  const s = totalSec % 60
  return `${m}m ${s}s`
}

function successRate(s: typeof props.summary.overall): string {
  if (s.run_count === 0) return '—'
  return ((s.success_count / s.run_count) * 100).toFixed(1) + '%'
}
</script>

<template>
  <div class="summary-tiles">
    <div class="dash-tile">
      <span class="tile-label">Total runs</span>
      <span class="tile-value">{{ summary.overall.run_count }}</span>
    </div>
    <div class="dash-tile">
      <span class="tile-label">Success rate</span>
      <span class="tile-value">{{ successRate(summary.overall) }}</span>
    </div>
    <div class="dash-tile">
      <span class="tile-label">Total cost</span>
      <span class="tile-value">{{ fmtCost(summary.overall.total_cost_usd) }}</span>
    </div>
    <div class="dash-tile">
      <span class="tile-label">Mean output tokens/s</span>
      <span class="tile-value">{{ fmt(summary.overall.mean_output_tokens_per_second, 1) }}</span>
    </div>
    <div class="dash-tile">
      <span class="tile-label">Mean TTFT</span>
      <span class="tile-value">{{ fmtDuration(summary.overall.mean_ttft_ms) }}</span>
    </div>
    <div class="dash-tile">
      <span class="tile-label">Cache hit ratio</span>
      <span class="tile-value">{{ fmt(summary.overall.cache_hit_ratio != null ? summary.overall.cache_hit_ratio * 100 : null, 1, '%') }}</span>
    </div>
  </div>
</template>

<style scoped>
.summary-tiles {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: var(--space-4);
}
.dash-tile {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.tile-label {
  font-size: var(--text-xs);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--color-text-muted);
}
.tile-value {
  font-size: var(--text-xl);
  font-weight: 700;
  color: var(--color-text);
}
</style>
