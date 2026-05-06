<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useOllamaInstancesStore } from '@/stores/ollamaInstances'
import { useUiStore } from '@/stores/ui'
import { ApiError } from '@/api/client'
import OllamaInstanceForm from '@/components/ollama/OllamaInstanceForm.vue'
import type { OllamaInstance } from '@/types/api'

const store = useOllamaInstancesStore()
const ui = useUiStore()

const showModal = ref(false)
const editTarget = ref<OllamaInstance | null>(null)
const deleteTarget = ref<OllamaInstance | null>(null)
const deleteError = ref<string | null>(null)
const confirmText = ref('')
const refreshing = ref(false)

const existingNames = computed(() => store.instances.map((i) => i.name))

onMounted(async () => {
  await store.fetchInstances()
  await store.checkAllHealth()
})

function openAdd() {
  editTarget.value = null
  showModal.value = true
}

function openEdit(inst: OllamaInstance) {
  editTarget.value = inst
  showModal.value = true
}

function openDelete(inst: OllamaInstance) {
  deleteTarget.value = inst
  deleteError.value = null
  confirmText.value = ''
}

function closeModal() {
  showModal.value = false
  editTarget.value = null
}

async function handleFormSubmit(payload: OllamaInstance) {
  try {
    if (editTarget.value) {
      await store.updateInstance(editTarget.value.name, {
        base_url: payload.base_url,
        api_key: payload.api_key,
      })
    } else {
      await store.createInstance(payload)
    }
    closeModal()
    await store.checkAllHealth()
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Operation failed')
  }
}

async function confirmDelete() {
  if (!deleteTarget.value) return
  deleteError.value = null
  try {
    await store.deleteInstance(deleteTarget.value.name)
    deleteTarget.value = null
  } catch (e: unknown) {
    if (e instanceof ApiError && e.status === 409) {
      deleteError.value = e.message
    } else {
      deleteError.value = e instanceof Error ? e.message : 'Delete failed'
    }
  }
}

async function refresh() {
  refreshing.value = true
  try {
    await store.fetchInstances()
    await store.checkAllHealth()
  } finally {
    refreshing.value = false
  }
}

function healthDot(name: string): 'ok' | 'error' | 'unknown' {
  const h = store.health.get(name)
  if (!h) return 'unknown'
  return h.ok ? 'ok' : 'error'
}

function healthLabel(name: string): string {
  const h = store.health.get(name)
  if (!h) return '—'
  if (h.ok) return h.latency_ms != null ? `${h.latency_ms} ms` : 'OK'
  return h.error ?? 'Unreachable'
}
</script>

<template>
  <div class="osv">
    <div class="osv-header">
      <h2 class="osv-title">Ollama Instances</h2>
      <div class="osv-header-actions">
        <button class="btn-secondary" :disabled="refreshing" @click="refresh">
          {{ refreshing ? 'Refreshing…' : 'Refresh' }}
        </button>
        <button class="btn-primary" @click="openAdd">Add Instance</button>
      </div>
    </div>

    <div v-if="store.loading" class="osv-state">Loading…</div>
    <div v-else-if="!store.instances.length" class="osv-state">
      No Ollama instances configured. Click <strong>Add Instance</strong> to register one.
    </div>

    <table v-else class="osv-table">
      <thead>
        <tr>
          <th>Name</th>
          <th>Base URL</th>
          <th>Health</th>
          <th>Latency</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="inst in store.instances" :key="inst.name">
          <td class="cell-name">{{ inst.name }}</td>
          <td class="cell-url">{{ inst.base_url }}</td>
          <td class="cell-health">
            <span
              class="health-dot"
              :class="`health-dot--${healthDot(inst.name)}`"
              :title="healthLabel(inst.name)"
            />
          </td>
          <td class="cell-latency">{{ healthLabel(inst.name) }}</td>
          <td class="cell-actions">
            <button class="btn-icon" title="Edit" @click="openEdit(inst)">✎</button>
            <button class="btn-icon btn-icon--danger" title="Delete" @click="openDelete(inst)">✕</button>
          </td>
        </tr>
      </tbody>
    </table>

    <!-- Add / Edit modal -->
    <Teleport to="body">
      <div v-if="showModal" class="modal-overlay" @click.self="closeModal">
        <div class="modal-panel" role="dialog" aria-modal="true" :aria-label="editTarget ? 'Edit Ollama instance' : 'Add Ollama instance'">
          <div class="modal-header">
            <h3 class="modal-title">{{ editTarget ? 'Edit Instance' : 'Add Ollama Instance' }}</h3>
            <button class="modal-close" aria-label="Close" @click="closeModal">✕</button>
          </div>
          <div class="modal-body">
            <OllamaInstanceForm
              :initial="editTarget"
              :existing-names="existingNames"
              @submit="handleFormSubmit"
              @cancel="closeModal"
            />
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Delete confirmation modal -->
    <Teleport to="body">
      <div v-if="deleteTarget" class="modal-overlay" @click.self="deleteTarget = null">
        <div class="modal-panel" role="dialog" aria-modal="true" aria-label="Confirm deletion">
          <div class="modal-header">
            <h3 class="modal-title">Delete Instance</h3>
            <button class="modal-close" aria-label="Close" @click="deleteTarget = null">✕</button>
          </div>
          <div class="modal-body">
            <p class="confirm-text">
              Delete <strong>{{ deleteTarget.name }}</strong>? This cannot be undone.
            </p>
            <p v-if="deleteError" class="confirm-error">{{ deleteError }}</p>
            <div class="oif-actions">
              <button class="btn-secondary" @click="deleteTarget = null">Cancel</button>
              <button class="btn-danger" @click="confirmDelete">Delete</button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.osv {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}
.osv-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.osv-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}
.osv-header-actions {
  display: flex;
  gap: var(--space-2);
}
.osv-state {
  padding: var(--space-8) var(--space-6);
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.osv-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-sm);
}
.osv-table th {
  position: sticky;
  top: 0;
  background: var(--color-bg);
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  padding: var(--space-2) var(--space-4);
  text-align: left;
  border-bottom: 1px solid var(--color-border);
  z-index: 1;
}
.osv-table td {
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid var(--color-border);
  vertical-align: middle;
}
.cell-name { font-weight: 600; color: var(--color-text); }
.cell-url { font-family: monospace; font-size: 12px; color: var(--color-text-muted); }
.cell-health { width: 40px; text-align: center; }
.cell-latency { font-size: 12px; color: var(--color-text-muted); min-width: 80px; }
.cell-actions {
  display: flex;
  gap: var(--space-2);
  align-items: center;
}
.health-dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--color-border);
}
.health-dot--ok    { background: #22c55e; }
.health-dot--error { background: #ef4444; }
.health-dot--unknown { background: var(--color-border); }

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
.btn-primary:hover { opacity: 0.88; }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-secondary {
  padding: var(--space-2) var(--space-4);
  background: transparent;
  color: var(--color-text-muted);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-secondary:hover { border-color: var(--color-text-muted); color: var(--color-text); }
.btn-secondary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-icon {
  background: none;
  border: none;
  padding: 4px 6px;
  font-size: 14px;
  cursor: pointer;
  color: var(--color-text-muted);
  border-radius: var(--radius-sm);
}
.btn-icon:hover { background: var(--color-surface); color: var(--color-text); }
.btn-icon--danger:hover { background: #fee2e2; color: #dc2626; }
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
.btn-danger:hover { opacity: 0.88; }

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 300;
  padding: var(--space-6);
}
.modal-panel {
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  width: 100%;
  max-width: 480px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-5) var(--space-6) var(--space-3);
  border-bottom: 1px solid var(--color-border);
}
.modal-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
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
}
.confirm-text {
  font-size: var(--text-sm);
  color: var(--color-text);
  margin: 0 0 var(--space-4);
}
.confirm-error {
  font-size: var(--text-sm);
  color: var(--color-error);
  margin: 0 0 var(--space-4);
}
.oif-actions {
  display: flex;
  gap: var(--space-2);
  justify-content: flex-end;
}
</style>
