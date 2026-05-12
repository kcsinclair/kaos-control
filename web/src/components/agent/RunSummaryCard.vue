<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed } from 'vue'
import type { RunResult } from '@/types/api'

const props = defineProps<{
  result: RunResult | null
  driverAvailable: boolean
}>()

function formatMs(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  const totalSeconds = Math.floor(ms / 1000)
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  if (minutes === 0) return `${seconds}s`
  return `${minutes}m ${seconds}s`
}

const formattedCost = computed(() => {
  if (!props.result) return '—'
  return `$${props.result.total_cost_usd.toFixed(4)}`
})

const formattedDuration = computed(() => {
  if (!props.result) return '—'
  const wall = formatMs(props.result.duration_ms)
  const api = formatMs(props.result.duration_api_ms)
  return `${wall} (API: ${api})`
})

const cacheHitRatio = computed<number | null>(() => {
  if (!props.result) return null
  const { input_tokens, cache_creation_input_tokens, cache_read_input_tokens } = props.result.usage
  const denominator = cache_read_input_tokens + cache_creation_input_tokens + input_tokens
  if (denominator === 0) return null
  return cache_read_input_tokens / denominator
})

const cacheHitDisplay = computed<string>(() => {
  if (cacheHitRatio.value === null) return 'N/A'
  return `${(cacheHitRatio.value * 100).toFixed(1)}%`
})

interface CacheQuality {
  label: string
  color: 'green' | 'blue' | 'amber' | 'red'
}

const cacheQuality = computed<CacheQuality | null>(() => {
  const ratio = cacheHitRatio.value
  if (ratio === null) return null
  if (ratio >= 0.90) return { label: 'Excellent', color: 'green' }
  if (ratio >= 0.75) return { label: 'Good', color: 'blue' }
  if (ratio >= 0.50) return { label: 'Fair', color: 'amber' }
  return { label: 'Poor', color: 'red' }
})

const cacheAriaLabel = computed<string>(() => {
  if (cacheHitRatio.value === null) return 'Cache efficiency: N/A'
  if (!cacheQuality.value) return `Cache efficiency: ${cacheHitDisplay.value}`
  return `Cache efficiency: ${cacheHitDisplay.value} — ${cacheQuality.value.label}`
})

function fmtTokens(n: number): string {
  return n.toLocaleString()
}
</script>

<template>
  <!-- Driver doesn't support token metrics -->
  <div v-if="!driverAvailable && !result" class="rsc-unavailable">
    Token metrics not available for this driver.
  </div>

  <!-- Driver available but result missing -->
  <div v-else-if="driverAvailable && !result" class="rsc-unavailable">
    Summary unavailable.
  </div>

  <!-- Full summary card -->
  <div v-else-if="result" class="rsc-card">
    <!-- Header row: subtype, cost, duration, turns -->
    <div class="rsc-header">
      <span class="rsc-subtype-badge">{{ result.subtype }}</span>
      <span class="rsc-metric"><span class="rsc-metric-label">Cost</span> {{ formattedCost }}</span>
      <span class="rsc-metric"><span class="rsc-metric-label">Duration</span> {{ formattedDuration }}</span>
      <span class="rsc-metric"><span class="rsc-metric-label">Turns</span> {{ result.num_turns }}</span>
    </div>

    <!-- Token usage table -->
    <table class="rsc-table">
      <caption class="rsc-table-caption">Token Usage</caption>
      <tbody>
        <tr>
          <td class="rsc-td-label">Input</td>
          <td class="rsc-td-value">{{ fmtTokens(result.usage.input_tokens) }}</td>
        </tr>
        <tr>
          <td class="rsc-td-label">Cache Creation</td>
          <td class="rsc-td-value">{{ fmtTokens(result.usage.cache_creation_input_tokens) }}</td>
        </tr>
        <tr>
          <td class="rsc-td-label">Cache Read</td>
          <td class="rsc-td-value">{{ fmtTokens(result.usage.cache_read_input_tokens) }}</td>
        </tr>
        <tr>
          <td class="rsc-td-label">Output</td>
          <td class="rsc-td-value">{{ fmtTokens(result.usage.output_tokens) }}</td>
        </tr>
      </tbody>
    </table>

    <!-- Cache hit ratio -->
    <div class="rsc-cache-row">
      <span class="rsc-cache-label">Cache hit ratio</span>
      <span class="rsc-cache-value">{{ cacheHitDisplay }}</span>
      <span
        v-if="cacheQuality"
        class="rsc-quality-badge"
        :data-color="cacheQuality.color"
        :aria-label="cacheAriaLabel"
      >{{ cacheQuality.label }}</span>
    </div>

    <!-- Permission denials (collapsible when long) -->
    <div v-if="result.permission_denials && result.permission_denials.length > 0" class="rsc-denials">
      <details v-if="result.permission_denials.length > 3">
        <summary class="rsc-denials-summary">
          Permission denials ({{ result.permission_denials.length }})
        </summary>
        <ul class="rsc-denials-list">
          <li
            v-for="(denial, i) in result.permission_denials"
            :key="i"
            class="rsc-denial-item"
          >{{ JSON.stringify(denial) }}</li>
        </ul>
      </details>
      <div v-else>
        <div class="rsc-denials-summary">Permission denials</div>
        <ul class="rsc-denials-list">
          <li
            v-for="(denial, i) in result.permission_denials"
            :key="i"
            class="rsc-denial-item"
          >{{ JSON.stringify(denial) }}</li>
        </ul>
      </div>
    </div>
  </div>
</template>

<style scoped>
.rsc-unavailable {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  padding: var(--space-3) 0;
}

.rsc-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  background: var(--color-bg-subtle, var(--color-border));
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-4);
}

/* Header row */
.rsc-header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-3);
}

.rsc-subtype-badge {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 500;
  background: var(--badge-done-bg);
  color: var(--badge-done-text);
  text-transform: capitalize;
}

.rsc-metric {
  font-size: var(--text-sm);
  color: var(--color-text);
  display: flex;
  align-items: center;
  gap: var(--space-1);
}

.rsc-metric-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
}

/* Token table */
.rsc-table {
  border-collapse: collapse;
  width: 100%;
  font-size: var(--text-sm);
}

.rsc-table-caption {
  text-align: left;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  padding-bottom: var(--space-1);
  caption-side: top;
}

.rsc-td-label {
  color: var(--color-text-muted);
  padding: 2px var(--space-3) 2px 0;
  width: 40%;
}

.rsc-td-value {
  color: var(--color-text);
  font-variant-numeric: tabular-nums;
  text-align: right;
}

/* Cache hit row */
.rsc-cache-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-sm);
}

.rsc-cache-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
}

.rsc-cache-value {
  color: var(--color-text);
  font-variant-numeric: tabular-nums;
}

.rsc-quality-badge {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 500;
}

.rsc-quality-badge[data-color="green"] {
  background: var(--badge-done-bg);
  color: var(--badge-done-text);
}
.rsc-quality-badge[data-color="blue"] {
  background: var(--badge-approved-bg);
  color: var(--badge-approved-text);
}
.rsc-quality-badge[data-color="amber"] {
  background: var(--badge-in-progress-bg);
  color: var(--badge-in-progress-text);
}
.rsc-quality-badge[data-color="red"] {
  background: var(--badge-blocked-bg);
  color: var(--badge-blocked-text);
}

/* Permission denials */
.rsc-denials {
  font-size: var(--text-sm);
}

.rsc-denials-summary {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  cursor: pointer;
  margin-bottom: var(--space-1);
}

details > .rsc-denials-summary {
  list-style: none;
}

details > summary.rsc-denials-summary::before {
  content: '▶ ';
}

details[open] > summary.rsc-denials-summary::before {
  content: '▼ ';
}

.rsc-denials-list {
  margin: var(--space-1) 0 0;
  padding-left: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.rsc-denial-item {
  font-family: monospace;
  font-size: 12px;
  color: var(--badge-blocked-text);
  word-break: break-all;
}
</style>
