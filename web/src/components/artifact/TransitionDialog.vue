<script setup lang="ts">
import { ref, computed } from 'vue'
import { transitionArtifact } from '@/api/artifacts'
import { useArtifactsStore } from '@/stores/artifacts'

const props = defineProps<{
  project: string
  path: string
  currentStatus: string
}>()

const emit = defineEmits<{
  transitioned: [newStatus: string]
  cancel: []
}>()

const store = useArtifactsStore()

const STATUSES = [
  'draft', 'clarifying', 'planning', 'in-progress', 'in-development',
  'in-qa', 'done', 'approved', 'blocked', 'rejected', 'abandoned',
].filter((s) => s !== props.currentStatus)

const selectedStatus = ref('')
const comment = ref('')
const saving = ref(false)
const error = ref<string | null>(null)

const needsComment = computed(() => selectedStatus.value === 'rejected')

async function confirm() {
  if (!selectedStatus.value) { error.value = 'Please select a target status.'; return }
  saving.value = true
  error.value = null
  try {
    await transitionArtifact(props.project, props.path, selectedStatus.value, comment.value || undefined)
    store.invalidate(props.path)
    emit('transitioned', selectedStatus.value)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Transition failed'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="td-overlay" @click.self="emit('cancel')">
    <div class="td-panel" role="dialog" aria-modal="true" aria-label="Change status">
      <h3 class="td-title">Change Status</h3>
      <p class="td-current">Current: <strong>{{ currentStatus }}</strong></p>

      <label class="td-field">
        <span class="td-label">New status</span>
        <select class="td-select" v-model="selectedStatus">
          <option value="" disabled>Select…</option>
          <option v-for="s in STATUSES" :key="s" :value="s">{{ s }}</option>
        </select>
      </label>

      <label class="td-field" v-if="needsComment">
        <span class="td-label">Rejection reason <span class="td-required">(required)</span></span>
        <textarea
          class="td-textarea"
          v-model="comment"
          rows="4"
          placeholder="Explain why this is being rejected…"
        />
      </label>

      <div v-if="error" class="td-error">{{ error }}</div>

      <div class="td-actions">
        <button class="btn-primary" :disabled="saving || !selectedStatus" @click="confirm">
          {{ saving ? 'Saving…' : 'Confirm' }}
        </button>
        <button class="btn-ghost" :disabled="saving" @click="emit('cancel')">Cancel</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.td-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 300;
}
.td-panel {
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  padding: var(--space-6);
  width: 360px;
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
.td-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}
.td-current {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.td-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.td-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
}
.td-required { font-weight: normal; }
.td-select, .td-textarea {
  padding: var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: var(--text-sm);
  font-family: inherit;
  width: 100%;
  box-sizing: border-box;
}
.td-select:focus, .td-textarea:focus {
  outline: none;
  border-color: var(--color-accent);
}
.td-textarea { resize: vertical; }
.td-error {
  font-size: var(--text-sm);
  color: #dc2626;
  background: #fee2e2;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
}
.td-actions {
  display: flex;
  gap: var(--space-2);
}
.btn-primary {
  padding: var(--space-2) var(--space-4);
  background: var(--color-accent);
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-primary:hover:not(:disabled) { opacity: 0.88; }
.btn-ghost {
  padding: var(--space-2) var(--space-3);
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  cursor: pointer;
}
.btn-ghost:hover:not(:disabled) { background: var(--color-surface); }
</style>
