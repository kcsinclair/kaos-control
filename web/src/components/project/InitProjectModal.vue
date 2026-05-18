<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref } from 'vue'
import { useProjectStore } from '@/stores/project'
import { useUiStore } from '@/stores/ui'
import { ApiError } from '@/api/client'
import type { ProjectSummary, InitProjectResult } from '@/types/api'

const props = defineProps<{
  project: ProjectSummary
}>()

const emit = defineEmits<{
  initialised: []
  close: []
}>()

const projectStore = useProjectStore()
const ui = useUiStore()

type Phase = 'confirm' | 'result'

const phase = ref<Phase>('confirm')
const initialising = ref(false)
const error = ref('')
const result = ref<InitProjectResult | null>(null)
const copyLabel = ref('Copy')

async function handleInit() {
  initialising.value = true
  error.value = ''
  try {
    result.value = await projectStore.init(props.project.name)
    phase.value = 'result'
    ui.success(`Project "${props.project.name}" initialised.`)
  } catch (err) {
    if (err instanceof ApiError) {
      error.value = err.message
    } else {
      error.value = err instanceof Error ? err.message : 'Initialisation failed.'
    }
  } finally {
    initialising.value = false
  }
}

async function copyCommands() {
  if (!result.value?.git_commands) return
  try {
    await navigator.clipboard.writeText(result.value.git_commands.join('\n'))
    copyLabel.value = 'Copied!'
    setTimeout(() => { copyLabel.value = 'Copy' }, 2000)
  } catch {
    copyLabel.value = 'Copy'
  }
}

function handleOverlayClick(e: MouseEvent) {
  if ((e.target as HTMLElement).classList.contains('modal-overlay')) emit('close')
}

function handleDone() {
  emit('initialised')
}
</script>

<template>
  <Teleport to="body">
    <div
      class="modal-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="init-project-title"
      @click="handleOverlayClick"
      @keydown.escape="emit('close')"
    >
      <div class="modal-panel">
        <div class="modal-header">
          <h2 id="init-project-title" class="modal-title">Initialise Project</h2>
          <button class="modal-close" aria-label="Close" @click="emit('close')">✕</button>
        </div>

        <!-- Confirm phase -->
        <template v-if="phase === 'confirm'">
          <div class="modal-body">
            <p class="intro">
              Initialising <strong class="project-name">{{ project.name }}</strong> will create the
              following structure inside <code class="inline-path">{{ project.path }}</code>.
              You'll be added to <code>lifecycle/config.yaml</code> as the initial owner.
            </p>
            <ul class="will-create">
              <li><code>lifecycle/config.yaml</code> — roles, stages, agents, owner</li>
              <li><code>CLAUDE.md</code> — guidance for Claude Code agents</li>
              <li><code>.claude/settings.json</code></li>
              <li><code>.gitignore</code></li>
              <li><code>devops/sample.yaml</code> — example pipeline</li>
              <li><code>lifecycle/{ideas, requirements, backend-plans, frontend-plans,</code></li>
              <li class="indent"><code>test-plans, tests, prototypes, releases, defects, docs, devops}/</code></li>
              <li><code>tests/</code>, <code>devops/</code></li>
            </ul>
            <p class="git-note">
              Existing files are left untouched (idempotent). If the directory is not already a git
              repository, one will be initialised and the scaffolding committed automatically.
            </p>

            <div v-if="error" class="general-error">{{ error }}</div>
          </div>

          <div class="modal-footer">
            <button class="btn-secondary" :disabled="initialising" @click="emit('close')">
              Cancel
            </button>
            <button class="btn-primary" :disabled="initialising" @click="handleInit">
              <span v-if="initialising" class="spinner" aria-hidden="true"></span>
              <span v-else>Initialise</span>
            </button>
          </div>
        </template>

        <!-- Result phase -->
        <template v-else-if="phase === 'result' && result">
          <div class="modal-body">
            <!-- Success / nothing to do -->
            <div v-if="result.created.length === 0" class="result-msg result-msg--info">
              Project was already fully initialised. No files were created.
            </div>

            <template v-else>
              <p class="result-label">Created:</p>
              <ul class="created-list">
                <li v-for="path in result.created" :key="path">
                  <code>{{ path }}</code>
                </li>
              </ul>
            </template>

            <!-- Git initialised freshly -->
            <div v-if="result.git_initialised" class="result-msg result-msg--ok">
              Git repository initialised and scaffolding committed.
            </div>

            <!-- Git already existed — show commands to run -->
            <template v-if="result.git_commands && result.git_commands.length">
              <p class="git-cmds-label">
                Git was already initialised. Run these commands to commit the new files:
              </p>
              <div class="git-cmds-block">
                <pre class="git-cmds-pre">{{ result.git_commands.join('\n') }}</pre>
                <button class="btn-copy" @click="copyCommands">{{ copyLabel }}</button>
              </div>
            </template>
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
  background: rgba(0,0,0,0.55);
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
  max-width: 540px;
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
.project-name { font-family: monospace; font-weight: 700; }
.inline-path { font-size: 12px; font-family: monospace; }
.will-create {
  margin: 0;
  padding-left: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.will-create li {
  font-size: var(--text-sm);
  color: var(--color-text);
}
.will-create code {
  font-family: monospace;
  font-size: 12px;
  color: var(--color-accent);
}
.will-create li.indent {
  list-style: none;
  padding-left: var(--space-4);
}
.git-note {
  margin: 0;
  font-size: var(--text-xs);
  color: var(--color-text-muted);
  line-height: 1.5;
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
.result-msg--ok {
  background: #d1fae5;
  border: 1px solid #6ee7b7;
  color: #065f46;
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
.created-list {
  margin: 0;
  padding-left: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.created-list code {
  font-family: monospace;
  font-size: 12px;
  color: var(--color-text);
}
.git-cmds-label {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-text);
  line-height: 1.5;
}
.git-cmds-block {
  position: relative;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  overflow: hidden;
}
.git-cmds-pre {
  margin: 0;
  padding: var(--space-3) var(--space-4);
  font-family: monospace;
  font-size: 12px;
  color: var(--color-text);
  white-space: pre-wrap;
  word-break: break-all;
  padding-right: 72px;
}
.btn-copy {
  position: absolute;
  top: var(--space-2);
  right: var(--space-2);
  padding: var(--space-1) var(--space-2);
  background: var(--color-border);
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
  color: var(--color-text);
  cursor: pointer;
}
.btn-copy:hover { background: var(--color-accent); color: #fff; }
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
