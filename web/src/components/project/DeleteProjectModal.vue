<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref } from 'vue'
import { useProjectStore } from '@/stores/project'
import { useUiStore } from '@/stores/ui'
import { ApiError } from '@/api/client'
import type { ProjectSummary } from '@/types/api'

const props = defineProps<{
  project: ProjectSummary
}>()

const emit = defineEmits<{
  confirmed: []
  close: []
}>()

const projectStore = useProjectStore()
const ui = useUiStore()

const deleting = ref(false)
const error = ref('')

async function handleDelete() {
  deleting.value = true
  error.value = ''
  try {
    await projectStore.remove(props.project.name)
    ui.success(`Project "${props.project.name}" deregistered.`)
    emit('confirmed')
  } catch (err) {
    if (err instanceof ApiError) {
      error.value = err.message
    } else {
      error.value = err instanceof Error ? err.message : 'Failed to delete project.'
    }
  } finally {
    deleting.value = false
  }
}

</script>

<template>
  <Teleport to="body">
    <div
      class="modal-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="delete-project-title"
      @keydown.escape="emit('close')"
    >
      <div class="modal-panel">
        <div class="modal-header">
          <h2 id="delete-project-title" class="modal-title">Delete Project</h2>
          <button class="modal-close" aria-label="Close" @click="emit('close')">✕</button>
        </div>

        <div class="modal-body">
          <p class="confirm-message">
            Are you sure you want to deregister
            <strong class="project-name">{{ project.name }}</strong>?
          </p>
          <p class="safety-note">
            This will deregister the project. Files on disk will not be deleted.
          </p>

          <div v-if="error" class="general-error">{{ error }}</div>
        </div>

        <div class="modal-footer">
          <button class="btn-secondary" :disabled="deleting" @click="emit('close')">
            Cancel
          </button>
          <button class="btn-danger" :disabled="deleting" @click="handleDelete">
            <span v-if="deleting" class="spinner" aria-hidden="true"></span>
            <span v-else>Delete</span>
          </button>
        </div>
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
  max-width: 440px;
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
  gap: var(--space-3);
}
.confirm-message {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-text);
  line-height: 1.5;
}
.project-name {
  font-family: monospace;
  font-weight: 700;
}
.safety-note {
  margin: 0;
  padding: var(--space-3);
  background: #fef3c7;
  border: 1px solid #fcd34d;
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  color: #78350f;
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
.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
  padding: var(--space-4) var(--space-6);
  border-top: 1px solid var(--color-border);
}
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
.btn-danger {
  padding: var(--space-2) var(--space-5);
  background: #dc2626;
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
.btn-danger:disabled { opacity: 0.6; cursor: not-allowed; }
.btn-danger:not(:disabled):hover { background: #b91c1c; }
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
