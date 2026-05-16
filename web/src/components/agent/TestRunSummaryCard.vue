<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref } from 'vue'
import type { RunSummary } from '@/types/api'

const props = defineProps<{ summary: RunSummary }>()

const gapsOpen = ref(false)

function formatElapsed(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  const s = ms / 1000
  if (s < 60) return `${s.toFixed(1)}s`
  const m = Math.floor(s / 60)
  const rem = Math.round(s % 60)
  return `${m}m ${rem}s`
}
</script>

<template>
  <div class="trs-card">
    <div class="trs-header">Test Run Summary</div>

    <!-- Per-suite stats table -->
    <table class="trs-table" v-if="summary.suites.length">
      <thead>
        <tr>
          <th>Suite</th>
          <th class="trs-num">Total</th>
          <th class="trs-num trs-pass">Pass</th>
          <th class="trs-num trs-fail">Fail</th>
          <th class="trs-num trs-skip">Skip</th>
          <th class="trs-num">Time</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="suite in summary.suites" :key="suite.name">
          <td class="trs-suite-name">{{ suite.name }}</td>
          <td class="trs-num">{{ suite.total }}</td>
          <td class="trs-num trs-pass">{{ suite.passed }}</td>
          <td class="trs-num trs-fail">{{ suite.failed }}</td>
          <td class="trs-num trs-skip">{{ suite.skipped }}</td>
          <td class="trs-num trs-muted">{{ formatElapsed(suite.elapsed) }}</td>
        </tr>
      </tbody>
    </table>
    <div v-else class="trs-muted trs-empty">No test suites recorded.</div>

    <!-- Summary line -->
    <div class="trs-summary-line">
      <span class="trs-stat" :class="{ 'trs-stat--warn': summary.defectsCreated > 0 }">
        {{ summary.defectsCreated }} defect{{ summary.defectsCreated !== 1 ? 's' : '' }} created
      </span>
      <span class="trs-sep">·</span>
      <span class="trs-stat">{{ summary.duplicatesFound }} duplicate{{ summary.duplicatesFound !== 1 ? 's' : '' }} found</span>
      <span class="trs-sep">·</span>
      <span class="trs-stat" :class="{ 'trs-stat--warn': summary.orphanedFailures > 0 }">
        {{ summary.orphanedFailures }} orphaned failure{{ summary.orphanedFailures !== 1 ? 's' : '' }}
      </span>
      <span class="trs-sep">·</span>
      <span class="trs-stat trs-muted">{{ formatElapsed(summary.elapsed) }} total</span>
    </div>

    <!-- Coverage gaps -->
    <div v-if="summary.coverageGaps.length" class="trs-gaps">
      <button class="trs-gaps-toggle" @click="gapsOpen = !gapsOpen" :aria-expanded="gapsOpen">
        <span class="trs-gaps-arrow" :class="{ 'trs-gaps-arrow--open': gapsOpen }">▶</span>
        Coverage gaps ({{ summary.coverageGaps.length }})
      </button>
      <ul v-if="gapsOpen" class="trs-gaps-list">
        <li v-for="gap in summary.coverageGaps" :key="gap" class="trs-gaps-item">{{ gap }}</li>
      </ul>
    </div>
  </div>
</template>

<style scoped>
.trs-card {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.trs-header {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  padding: var(--space-2) var(--space-3);
  border-bottom: 1px solid var(--color-border);
  background: var(--color-surface);
}
.trs-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-xs);
}
.trs-table th {
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
  padding: var(--space-1) var(--space-3);
  border-bottom: 1px solid var(--color-border);
  text-align: left;
}
.trs-table td {
  padding: var(--space-1) var(--space-3);
  border-bottom: 1px solid color-mix(in srgb, var(--color-border) 50%, transparent);
}
.trs-table tbody tr:last-child td { border-bottom: none; }
.trs-suite-name {
  font-family: monospace;
  font-size: 11px;
  color: var(--color-text);
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.trs-num { text-align: right; font-variant-numeric: tabular-nums; font-size: 12px; }
.trs-pass { color: #16a34a; }
.trs-fail { color: #dc2626; }
.trs-skip { color: var(--color-text-muted); }
.trs-muted { color: var(--color-text-muted); }
.trs-empty { padding: var(--space-2) var(--space-3); font-size: var(--text-sm); }
.trs-summary-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-1);
  padding: var(--space-2) var(--space-3);
  border-top: 1px solid var(--color-border);
  font-size: var(--text-xs);
}
.trs-stat { color: var(--color-text); }
.trs-stat--warn { color: #dc2626; font-weight: 600; }
.trs-sep { color: var(--color-text-muted); }
.trs-gaps {
  border-top: 1px solid var(--color-border);
  padding: var(--space-1) var(--space-3);
}
.trs-gaps-toggle {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  background: none;
  border: none;
  cursor: pointer;
  font-size: var(--text-xs);
  color: var(--color-text-muted);
  padding: var(--space-1) 0;
  font-family: inherit;
}
.trs-gaps-toggle:hover { color: var(--color-text); }
.trs-gaps-arrow {
  font-size: 8px;
  transition: transform 0.15s;
}
.trs-gaps-arrow--open { transform: rotate(90deg); }
.trs-gaps-list {
  list-style: none;
  margin: var(--space-1) 0 0 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.trs-gaps-item {
  font-family: monospace;
  font-size: 11px;
  color: var(--color-text-muted);
  padding: 1px var(--space-1);
}
</style>
