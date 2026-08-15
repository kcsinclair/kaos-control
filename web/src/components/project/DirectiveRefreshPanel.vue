<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
// Self-contained panel: regenerates AGENTS.md/CLAUDE.md/GEMINI.md and
// re-patches the six standard agents from the current architecture + stack
// (FR-14). Reused directly (no modal wrapper) by:
//   - the project settings/menu "Refresh directives" action (see ProjectsView.vue)
//   - the Architecture Wizard as its opt-in final scaffolding step (FR-17,
//     see onboarding-architecture-selection) — pass `project` and listen for
//     `done` to chain into the wizard's completion screen.
import { refreshDirectives } from '@/api/directives'
import { useDirectiveApply } from '@/composables/useDirectiveApply'

const props = defineProps<{ project: string }>()
const emit = defineEmits<{
  done: []
}>()

const { phase, loading, error, result, pendingDiff, apply, reset } = useDirectiveApply(
  props.project,
  refreshDirectives,
)

function startOver() {
  reset()
}
</script>

<template>
  <div class="directive-refresh-panel">
    <template v-if="phase === 'idle'">
      <p class="drp-intro">
        Regenerate directive files and agent prompts from the current architecture + stack.
      </p>
      <div v-if="error" class="general-error">{{ error }}</div>
      <button class="btn-primary" :disabled="loading" @click="apply(false)">
        <span v-if="loading" class="spinner" aria-hidden="true"></span>
        <span v-else>Refresh directives</span>
      </button>
    </template>

    <template v-else-if="phase === 'diff' && pendingDiff">
      <p class="drp-intro">
        <code>{{ pendingDiff.path }}</code> has been edited since it was last generated.
        Overwriting will replace it with the regenerated content shown below.
      </p>
      <pre class="diff-pre">{{ pendingDiff.diff }}</pre>
      <div v-if="error" class="general-error">{{ error }}</div>
      <div class="drp-actions">
        <button class="btn-secondary" :disabled="loading" @click="startOver">Cancel</button>
        <button class="btn-primary btn-primary--danger" :disabled="loading" @click="apply(true)">
          <span v-if="loading" class="spinner" aria-hidden="true"></span>
          <span v-else>Overwrite</span>
        </button>
      </div>
    </template>

    <template v-else-if="phase === 'result' && result">
      <div v-if="result.files.length === 0" class="result-msg result-msg--info">
        Already up to date. No files were changed.
      </div>
      <template v-else>
        <p class="result-label">Files:</p>
        <ul class="file-list">
          <li v-for="f in result.files" :key="f.path">
            <code>{{ f.path }}</code>
            <span class="file-status">
              {{ f.created ? 'created' : f.changed ? 'changed' : 'skipped' }}
            </span>
          </li>
        </ul>
      </template>
      <div v-if="result.disabledAgents?.length" class="result-msg result-msg--info">
        Disabled agents (not required for this stack): {{ result.disabledAgents.join(', ') }}
      </div>
      <div v-if="result.skipped?.length" class="result-msg result-msg--info">
        Skipped: {{ result.skipped.join(', ') }}
      </div>
      <div class="drp-actions">
        <button class="btn-secondary" @click="startOver">Refresh again</button>
        <button class="btn-primary" @click="emit('done')">Done</button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.directive-refresh-panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
.drp-intro {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-text);
  line-height: 1.6;
}
.drp-intro code {
  font-family: monospace;
  font-size: 12px;
  color: var(--color-accent);
}
.diff-pre {
  margin: 0;
  padding: var(--space-3) var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-family: monospace;
  font-size: 12px;
  color: var(--color-text);
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 300px;
  overflow-y: auto;
}
.general-error {
  padding: var(--space-3);
  background: #fee2e2;
  border: 1px solid #fca5a5;
  border-radius: var(--radius-md);
  color: #991b1b;
  font-size: var(--text-sm);
}
.result-msg {
  padding: var(--space-3);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
}
.result-msg--info {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  color: var(--color-text-muted);
}
.result-label {
  margin: 0;
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
}
.file-list {
  margin: 0;
  padding-left: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.file-list code {
  font-family: monospace;
  font-size: 12px;
  color: var(--color-text);
}
.file-status {
  margin-left: var(--space-2);
  font-size: 11px;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.03em;
}
.drp-actions {
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
  display: flex;
  align-items: center;
  gap: var(--space-2);
  align-self: flex-start;
}
.btn-primary--danger {
  background: #dc2626;
}
.btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }
.btn-primary:not(:disabled):hover { opacity: 0.88; }
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
.btn-secondary:disabled { opacity: 0.6; cursor: not-allowed; }
.btn-secondary:not(:disabled):hover { background: var(--color-border); }
.spinner {
  display: inline-block;
  width: 14px;
  height: 14px;
  border: 2px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
  flex-shrink: 0;
}
@keyframes spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) { .spinner { animation: none; } }
</style>
