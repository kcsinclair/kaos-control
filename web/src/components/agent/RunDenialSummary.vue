<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import type { DenialRecord } from '@/types/api'

const props = defineProps<{
  denials: DenialRecord[]
  observeOnly?: boolean
}>()
</script>

<template>
  <div class="denial-summary" role="alert">
    <div class="denial-summary__header">
      <span class="denial-summary__icon" aria-hidden="true">&#9888;</span>
      <strong class="denial-summary__heading">
        <template v-if="props.observeOnly">
          {{ props.denials.length }} tool call{{ props.denials.length === 1 ? '' : 's' }}
          would have been denied (observe-only mode — no enforcement)
        </template>
        <template v-else>
          {{ props.denials.length }} tool call{{ props.denials.length === 1 ? '' : 's' }}
          {{ props.denials.length === 1 ? 'was' : 'were' }} denied during this run
        </template>
      </strong>
    </div>

    <ul class="denial-summary__list">
      <li v-for="(d, idx) in props.denials" :key="idx" class="denial-summary__item">
        <template v-if="d.path || d.command">
          <span class="denial-summary__verb">
            {{ d.path ? `Write to` : `Bash` }}
          </span>
          <code class="denial-summary__target">{{ d.path ?? d.command }}</code>
          <span class="denial-summary__sep">—</span>
        </template>
        <template v-else>
          <span class="denial-summary__verb">{{ d.tool_name }}</span>
          <span class="denial-summary__sep">—</span>
        </template>
        <span class="denial-summary__reason">denied: {{ d.reason }}</span>
      </li>
    </ul>

    <p v-if="!props.observeOnly" class="denial-summary__note">
      Auto-commit was skipped. Queue is paused.
    </p>
  </div>
</template>

<style scoped>
.denial-summary {
  border: 1px solid var(--color-error, #dc2626);
  background: var(--badge-blocked-bg);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  margin-bottom: var(--space-4);
  color: var(--badge-blocked-text);
}

.denial-summary__header {
  display: flex;
  align-items: flex-start;
  gap: var(--space-2);
  margin-bottom: var(--space-3);
}

.denial-summary__icon {
  font-size: var(--text-base, 1rem);
  color: var(--color-error, #dc2626);
  flex-shrink: 0;
  line-height: 1.4;
}

.denial-summary__heading {
  font-size: var(--text-sm);
  font-weight: 600;
  line-height: 1.4;
}

.denial-summary__list {
  list-style: none;
  margin: 0 0 var(--space-3) 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  border-top: 1px solid rgba(0,0,0,0.08);
  padding-top: var(--space-2);
}

.denial-summary__item {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 4px;
  font-size: var(--text-sm);
}

.denial-summary__item::before {
  content: '•';
  flex-shrink: 0;
  color: var(--color-error, #dc2626);
}

.denial-summary__verb {
  color: var(--badge-blocked-text);
}

.denial-summary__target {
  font-family: monospace;
  font-size: 0.85em;
  background: rgba(0,0,0,0.08);
  padding: 1px 4px;
  border-radius: var(--radius-sm);
  word-break: break-all;
}

.denial-summary__sep {
  color: var(--badge-blocked-text);
  opacity: 0.6;
}

.denial-summary__reason {
  color: var(--badge-blocked-text);
  opacity: 0.85;
}

.denial-summary__note {
  font-size: var(--text-sm);
  margin: 0;
  opacity: 0.75;
  border-top: 1px solid rgba(0,0,0,0.08);
  padding-top: var(--space-2);
}
</style>
