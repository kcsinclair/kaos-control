<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useProjectStore } from '@/stores/project'
import { useUiStore } from '@/stores/ui'
import { ApiError } from '@/api/client'
import type { CheckDirectoryResult } from '@/types/api'

const emit = defineEmits<{
  created: []
  close: []
}>()

const projectStore = useProjectStore()
const ui = useUiStore()

const form = reactive({
  name: '',
  path: '',
  description: '',
  owner: '',
})

const errors = reactive({
  name: '',
  path: '',
  general: '',
})

const submitting = ref(false)
const checkingDir = ref(false)
const dirResult = ref<CheckDirectoryResult | null>(null)

const NAME_RE = /^[a-z0-9][a-z0-9-]{1,78}[a-z0-9]$|^[a-z0-9]{3}$/

function validateName(): boolean {
  if (!form.name) {
    errors.name = 'Name is required.'
    return false
  }
  if (!NAME_RE.test(form.name)) {
    errors.name = 'Name must be 3–80 lowercase alphanumeric characters or hyphens.'
    return false
  }
  errors.name = ''
  return true
}

function validatePath(): boolean {
  if (!form.path) {
    errors.path = 'Path is required.'
    return false
  }
  if (!form.path.startsWith('/')) {
    errors.path = 'Path must be an absolute path starting with /.'
    return false
  }
  errors.path = ''
  return true
}

async function handleCheckDirectory() {
  if (!validatePath()) return
  checkingDir.value = true
  dirResult.value = null
  try {
    dirResult.value = await projectStore.checkDirectory(form.path)
  } catch (err) {
    errors.path = err instanceof Error ? err.message : 'Check failed'
  } finally {
    checkingDir.value = false
  }
}

async function handleSubmit() {
  errors.general = ''
  const nameOk = validateName()
  const pathOk = validatePath()
  if (!nameOk || !pathOk) return

  submitting.value = true
  try {
    await projectStore.create({
      name: form.name,
      path: form.path,
      description: form.description || undefined,
      owner: form.owner || undefined,
    })
    ui.success(`Project "${form.name}" created.`)
    emit('created')
  } catch (err) {
    if (err instanceof ApiError) {
      if (err.status === 409) {
        errors.name = `A project named "${form.name}" already exists.`
      } else if (err.code === 'invalid_name') {
        errors.name = err.message
      } else if (err.code === 'invalid_path') {
        errors.path = err.message
      } else {
        errors.general = err.message
      }
    } else {
      errors.general = err instanceof Error ? err.message : 'An error occurred.'
    }
  } finally {
    submitting.value = false
  }
}

</script>

<template>
  <Teleport to="body">
    <div
      class="modal-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="create-project-title"
      @keydown.escape="emit('close')"
    >
      <div class="modal-panel">
        <div class="modal-header">
          <h2 id="create-project-title" class="modal-title">New Project</h2>
          <button class="modal-close" aria-label="Close" @click="emit('close')">✕</button>
        </div>

        <form class="modal-body" @submit.prevent="handleSubmit">
          <!-- Name -->
          <div class="field">
            <label class="field-label" for="cp-name">Name <span class="required">*</span></label>
            <input
              id="cp-name"
              v-model="form.name"
              class="field-input"
              :class="{ 'field-input--error': errors.name }"
              type="text"
              placeholder="my-project"
              autocomplete="off"
              :disabled="submitting"
              @blur="validateName"
            />
            <span v-if="errors.name" class="field-error">{{ errors.name }}</span>
            <span class="field-hint">Lowercase alphanumeric and hyphens, 3–80 characters.</span>
          </div>

          <!-- Path -->
          <div class="field">
            <label class="field-label" for="cp-path">Path <span class="required">*</span></label>
            <div class="path-row">
              <input
                id="cp-path"
                v-model="form.path"
                class="field-input path-input"
                :class="{ 'field-input--error': errors.path }"
                type="text"
                placeholder="/home/user/projects/my-project"
                autocomplete="off"
                :disabled="submitting"
                @blur="validatePath"
                @input="dirResult = null"
              />
              <button
                type="button"
                class="btn-check"
                :disabled="submitting || checkingDir"
                @click="handleCheckDirectory"
              >
                <span v-if="checkingDir" class="spinner" aria-hidden="true"></span>
                <span v-else>Check</span>
              </button>
            </div>
            <span v-if="errors.path" class="field-error">{{ errors.path }}</span>

            <!-- Directory check result -->
            <div v-if="dirResult" class="dir-result">
              <span class="dir-check" :class="dirResult.exists ? 'ok' : 'fail'">
                {{ dirResult.exists ? '✓' : '✗' }} Directory exists
              </span>
              <span v-if="dirResult.exists" class="dir-check" :class="dirResult.writable ? 'ok' : 'fail'">
                {{ dirResult.writable ? '✓' : '✗' }} Writable
              </span>
              <span v-if="dirResult.exists && dirResult.initialised" class="dir-check info">
                ℹ Already initialised
              </span>
            </div>
          </div>

          <!-- Description -->
          <div class="field">
            <label class="field-label" for="cp-desc">Description</label>
            <input
              id="cp-desc"
              v-model="form.description"
              class="field-input"
              type="text"
              placeholder="Optional project description"
              :disabled="submitting"
            />
          </div>

          <!-- Owner -->
          <div class="field">
            <label class="field-label" for="cp-owner">Owner</label>
            <input
              id="cp-owner"
              v-model="form.owner"
              class="field-input"
              type="text"
              placeholder="Optional owner name or email"
              :disabled="submitting"
            />
          </div>

          <div v-if="errors.general" class="general-error">{{ errors.general }}</div>

          <div class="modal-footer">
            <button type="button" class="btn-secondary" :disabled="submitting" @click="emit('close')">
              Cancel
            </button>
            <button type="submit" class="btn-primary" :disabled="submitting">
              <span v-if="submitting" class="spinner" aria-hidden="true"></span>
              <span v-else>Create Project</span>
            </button>
          </div>
        </form>
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
  max-width: 520px;
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
}
.field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.field-label {
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
}
.required {
  color: #dc2626;
  margin-left: 2px;
}
.field-input {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-text);
  font-size: var(--text-sm);
  width: 100%;
  box-sizing: border-box;
}
.field-input:focus {
  outline: none;
  border-color: var(--color-accent);
}
.field-input--error {
  border-color: #dc2626;
}
.field-input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.field-error {
  font-size: var(--text-xs);
  color: #dc2626;
}
.field-hint {
  font-size: var(--text-xs);
  color: var(--color-text-muted);
}
.path-row {
  display: flex;
  gap: var(--space-2);
}
.path-input {
  flex: 1;
}
.btn-check {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-text);
  font-size: var(--text-sm);
  cursor: pointer;
  white-space: nowrap;
  display: flex;
  align-items: center;
  gap: var(--space-1);
}
.btn-check:disabled { opacity: 0.6; cursor: not-allowed; }
.btn-check:not(:disabled):hover { background: var(--color-border); }
.dir-result {
  display: flex;
  gap: var(--space-3);
  flex-wrap: wrap;
}
.dir-check {
  font-size: var(--text-xs);
  font-weight: 500;
}
.dir-check.ok { color: #059669; }
.dir-check.fail { color: #dc2626; }
.dir-check.info { color: #2563eb; }
.general-error {
  padding: var(--space-3);
  background: #fee2e2;
  border: 1px solid #fca5a5;
  border-radius: var(--radius-md);
  color: #991b1b;
  font-size: var(--text-sm);
}
.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
  padding-top: var(--space-2);
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
