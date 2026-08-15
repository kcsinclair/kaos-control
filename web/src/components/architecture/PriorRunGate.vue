<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import type { WizardPriorRun } from '@/types/api'

defineProps<{
  priorRun: WizardPriorRun
}>()

const emit = defineEmits<{
  continue: []
  exit: []
}>()
</script>

<template>
  <div class="prior-run-gate" role="alertdialog" aria-labelledby="prior-run-title">
    <h3 id="prior-run-title" class="gate-title">This project already has a chosen architecture</h3>
    <p class="gate-copy">
      The Architecture Wizard has already run for this project (FR-2/FR-3). Re-running it will
      let you pick a new architecture and stack, superseding the current choice.
    </p>
    <dl class="gate-summary">
      <template v-if="priorRun.architecture">
        <dt>Architecture</dt>
        <dd>{{ priorRun.architecture }}</dd>
      </template>
      <template v-if="priorRun.tech_stack">
        <dt>Tech stack</dt>
        <dd>{{ priorRun.tech_stack }}</dd>
      </template>
      <template v-if="priorRun.adr_path">
        <dt>Decision record</dt>
        <dd>{{ priorRun.adr_path }}</dd>
      </template>
      <template v-if="priorRun.summary_path">
        <dt>Summary</dt>
        <dd>{{ priorRun.summary_path }}</dd>
      </template>
    </dl>
    <div class="gate-actions">
      <button type="button" class="btn-secondary" @click="emit('exit')">Exit</button>
      <button type="button" class="btn-primary" @click="emit('continue')">Continue (re-run)</button>
    </div>
  </div>
</template>

<style scoped>
.prior-run-gate {
  max-width: 560px;
  margin: var(--space-8) auto;
  padding: var(--space-6);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
.gate-title {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: 700;
  color: var(--color-text);
}
.gate-copy {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  line-height: 1.6;
}
.gate-summary {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: var(--space-1) var(--space-3);
  margin: 0;
  font-size: var(--text-sm);
}
.gate-summary dt {
  font-weight: 600;
  color: var(--color-text);
}
.gate-summary dd {
  margin: 0;
  color: var(--color-text-muted);
  font-family: var(--font-mono, monospace);
  word-break: break-all;
}
.gate-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
}
.btn-primary {
  padding: var(--space-2) var(--space-5);
  background: var(--color-accent);
  color: #fff;
  border: none;
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-primary:hover { opacity: 0.88; }
.btn-secondary {
  padding: var(--space-2) var(--space-4);
  background: var(--color-surface);
  color: var(--color-text);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-secondary:hover { background: var(--color-border); }
</style>
