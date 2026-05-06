<script setup lang="ts">
import { ref, computed } from 'vue'
import { useReleasesStore } from '@/stores/releases'
import type { Release } from '@/types/release'

const props = defineProps<{
  release: Release
  project: string
  artifactCount: number
}>()

const emit = defineEmits<{
  confirmed: [reassignTo?: number]
  close: []
}>()

const store = useReleasesStore()

const reassignId = ref<number | ''>('')
const confirming = ref(false)

const otherReleases = computed(() =>
  store.releases.filter((r) => r.id !== props.release.id)
)

async function confirm() {
  confirming.value = true
  try {
    const reassignTo = reassignId.value !== '' ? Number(reassignId.value) : undefined
    emit('confirmed', reassignTo)
  } finally {
    confirming.value = false
  }
}
</script>

<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal-panel" role="dialog" aria-modal="true" aria-label="Delete release">
      <div class="modal-header">
        <h3 class="modal-title">Delete Release</h3>
        <button class="btn-icon" aria-label="Close" @click="emit('close')">✕</button>
      </div>

      <div class="modal-body">
        <p class="confirm-text">
          Are you sure you want to delete <strong>{{ release.name }}</strong>?
        </p>

        <p v-if="artifactCount > 0" class="artifact-count-text">
          This release has <strong>{{ artifactCount }}</strong> assigned
          {{ artifactCount === 1 ? 'artifact' : 'artifacts' }}.
        </p>

        <div v-if="artifactCount > 0 && otherReleases.length > 0" class="form-field">
          <label class="field-label" for="reassign-select">Reassign artifacts to</label>
          <select id="reassign-select" v-model="reassignId" class="field-input field-select">
            <option value="">Leave unassigned (orphaned)</option>
            <option v-for="r in otherReleases" :key="r.id" :value="r.id">
              {{ r.name }}
            </option>
          </select>
          <span class="field-hint">
            {{ reassignId !== '' ? 'Artifacts will be reassigned to the selected release.' : 'Artifacts will have no release assigned.' }}
          </span>
        </div>

        <p v-else-if="artifactCount > 0" class="field-hint">
          No other releases exist. Artifacts will be left unassigned.
        </p>
      </div>

      <div class="modal-footer">
        <button class="btn-danger" :disabled="confirming" @click="confirm">
          {{ confirming ? 'Deleting…' : 'Delete' }}
        </button>
        <button class="btn-ghost" :disabled="confirming" @click="emit('close')">Cancel</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 300;
}
.modal-panel {
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  width: 420px;
  max-width: calc(100vw - 2rem);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.modal-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}
.btn-icon {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--color-text-muted);
  font-size: var(--text-sm);
  padding: var(--space-1);
  border-radius: var(--radius-sm);
  line-height: 1;
}
.btn-icon:hover { background: var(--color-surface); color: var(--color-text); }
.modal-body {
  padding: var(--space-4) var(--space-6);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.confirm-text {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-text);
}
.artifact-count-text {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.form-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.field-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
}
.field-input {
  padding: var(--space-1) var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: var(--text-sm);
  font-family: inherit;
  width: 100%;
  box-sizing: border-box;
}
.field-input:focus { outline: none; border-color: var(--color-accent); }
.field-select {
  appearance: none;
  -webkit-appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6' viewBox='0 0 10 6'%3E%3Cpath d='M0 0l5 6 5-6z' fill='%23888'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right var(--space-2) center;
  padding-right: calc(var(--space-2) * 2 + 10px);
  cursor: pointer;
}
.field-hint {
  font-size: 11px;
  color: var(--color-text-muted);
}
.modal-footer {
  display: flex;
  gap: var(--space-2);
  padding: var(--space-4) var(--space-6);
  border-top: 1px solid var(--color-border);
  flex-shrink: 0;
}
.btn-danger {
  padding: var(--space-2) var(--space-4);
  background: #dc2626;
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-danger:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-danger:hover:not(:disabled) { opacity: 0.88; }
.btn-ghost {
  padding: var(--space-2) var(--space-3);
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  cursor: pointer;
}
.btn-ghost:hover { background: var(--color-surface); }
.btn-ghost:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
