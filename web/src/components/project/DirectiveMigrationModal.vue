<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref } from 'vue'
import { useProjectStore } from '@/stores/project'
import { ApiError } from '@/api/client'
import { migrateDirectives } from '@/api/directives'
import type { GenerateResult, DirectiveFileWrite } from '@/types/api'

const props = defineProps<{ project: string }>()
const emit = defineEmits<{
  close: []
  migrated: []
}>()

const projectStore = useProjectStore()

type Phase = 'confirm' | 'diff' | 'result'

const phase = ref<Phase>('confirm')
const migrating = ref(false)
const error = ref('')
const result = ref<GenerateResult | null>(null)
const pendingDiff = ref<DirectiveFileWrite | null>(null)

async function migrate(force: boolean) {
  migrating.value = true
  error.value = ''
  try {
    const res = await migrateDirectives(props.project, { force })
    const diffFile = !force ? res.files.find((f) => f.diff) : undefined
    if (diffFile) {
      pendingDiff.value = diffFile
      phase.value = 'diff'
      return
    }
    result.value = res
    phase.value = 'result'
    await projectStore.refreshCurrent()
  } catch (err) {
    error.value =
      err instanceof ApiError ? err.message : err instanceof Error ? err.message : 'Migration failed.'
  } finally {
    migrating.value = false
  }
}

function handleDone() {
  emit('migrated')
}
</script>

<template>
  <Teleport to="body">
    <div
      class="modal-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="migrate-directives-title"
      @keydown.escape="emit('close')"
    >
      <div class="modal-panel">
        <div class="modal-header">
          <h2 id="migrate-directives-title" class="modal-title">Migrate Directives</h2>
          <button class="modal-close" aria-label="Close" @click="emit('close')">✕</button>
        </div>

        <!-- Confirm phase -->
        <template v-if="phase === 'confirm'">
          <div class="modal-body">
            <p class="intro">This will make the following changes at the project root:</p>
            <ul class="change-list">
              <li>Rename <code>CLAUDE.md</code> → <code>AGENTS.md</code> (canonical directive file)</li>
              <li><code>CLAUDE.md</code> becomes a pointer: <code>@AGENTS.md</code></li>
              <li>Add <code>GEMINI.md</code> (pointer, if a Gemini driver is configured)</li>
            </ul>
            <div v-if="error" class="general-error">{{ error }}</div>
          </div>
          <div class="modal-footer">
            <button class="btn-secondary" :disabled="migrating" @click="emit('close')">Cancel</button>
            <button class="btn-primary" :disabled="migrating" @click="migrate(false)">
              <span v-if="migrating" class="spinner" aria-hidden="true"></span>
              <span v-else>Migrate now</span>
            </button>
          </div>
        </template>

        <!-- Diff phase: AGENTS.md was hand-edited, require explicit overwrite -->
        <template v-else-if="phase === 'diff' && pendingDiff">
          <div class="modal-body">
            <p class="intro">
              <code>{{ pendingDiff.path }}</code> has been edited since it was last generated.
              Overwriting will replace it with the migrated content shown below.
            </p>
            <pre class="diff-pre">{{ pendingDiff.diff }}</pre>
            <div v-if="error" class="general-error">{{ error }}</div>
          </div>
          <div class="modal-footer">
            <button class="btn-secondary" :disabled="migrating" @click="emit('close')">Cancel</button>
            <button class="btn-primary btn-primary--danger" :disabled="migrating" @click="migrate(true)">
              <span v-if="migrating" class="spinner" aria-hidden="true"></span>
              <span v-else>Overwrite</span>
            </button>
          </div>
        </template>

        <!-- Result phase -->
        <template v-else-if="phase === 'result' && result">
          <div class="modal-body">
            <div v-if="result.files.length === 0" class="result-msg result-msg--info">
              Already migrated. No files were changed.
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
          </div>
          <div class="modal-footer">
            <button class="btn-primary" @click="handleDone">Done</button>
          </div>
        </template>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
  padding: var(--space-6);
}
.modal-panel {
  position: relative;
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  width: 100%;
  max-width: 560px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-5) var(--space-6) var(--space-4);
  border-bottom: 1px solid var(--color-border);
}
.modal-title {
  font-size: var(--text-lg);
  font-weight: 700;
  color: var(--color-text);
  margin: 0;
}
.modal-close {
  background: none;
  border: none;
  font-size: var(--text-lg);
  color: var(--color-text-muted);
  cursor: pointer;
  line-height: 1;
  padding: var(--space-1);
}
.modal-close:hover { color: var(--color-text); }
.modal-body {
  padding: var(--space-5) var(--space-6);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  overflow-y: auto;
  max-height: 65vh;
}
.intro {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-text);
  line-height: 1.6;
}
.change-list {
  margin: 0;
  padding-left: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  font-size: var(--text-sm);
  color: var(--color-text);
}
.change-list code {
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
.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
  padding: var(--space-4) var(--space-6);
  border-top: 1px solid var(--color-border);
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
