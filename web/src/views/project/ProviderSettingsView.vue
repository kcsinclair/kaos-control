<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useProvidersStore } from '@/stores/providers'
import { useUiStore } from '@/stores/ui'
import { ApiError } from '@/api/client'
import ProviderForm from '@/components/provider/ProviderForm.vue'
import type { Provider, ProviderModel } from '@/types/api'

const store = useProvidersStore()
const ui = useUiStore()

const showModal = ref(false)
const editTarget = ref<Provider | null>(null)
const deleteTarget = ref<Provider | null>(null)
const deleteError = ref<string | null>(null)
const refreshing = ref(false)
const probingProvider = ref<string | null>(null)

// Models dialog
const showModelsModal = ref(false)
const selectedProviderModels = ref<{ providerName: string; models: ProviderModel[] } | null>(null)

const existingNames = computed(() => store.providers.map((p) => p.name))

onMounted(async () => {
  await store.fetchProviders()
  await store.probeAll()
})

function openAdd() {
  editTarget.value = null
  showModal.value = true
}

function openEdit(prov: Provider) {
  editTarget.value = prov
  showModal.value = true
}

function openDelete(prov: Provider) {
  deleteTarget.value = prov
  deleteError.value = null
}

function closeModal() {
  showModal.value = false
  editTarget.value = null
}

async function handleFormSubmit(payload: Provider) {
  try {
    if (editTarget.value) {
      await store.updateProvider(editTarget.value.name, {
        base_url: payload.base_url,
        driver: payload.driver,
        api_key: payload.api_key,
        extra_headers: payload.extra_headers,
      })
      ui.success(`Provider "${editTarget.value.name}" updated.`)
    } else {
      await store.createProvider(payload)
      ui.success(`Provider "${payload.name}" created.`)
    }
    closeModal()
    await store.probeProvider(payload.name)
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Operation failed')
  }
}

async function confirmDelete() {
  if (!deleteTarget.value) return
  deleteError.value = null
  try {
    await store.deleteProvider(deleteTarget.value.name)
    ui.success(`Provider "${deleteTarget.value.name}" deleted.`)
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
    await store.fetchProviders()
    await store.probeAll()
  } finally {
    refreshing.value = false
  }
}

async function handleProbe(name: string) {
  probingProvider.value = name
  try {
    const res = await store.probeProvider(name)
    if (res.ok) {
      ui.success(`Connected to ${name} (${res.latency_ms ?? 0} ms)`)
    } else {
      ui.error(`Probe failed for ${name}: ${res.error || 'Unknown error'}`)
    }
  } catch (err: unknown) {
    ui.error(err instanceof Error ? err.message : 'Probe failed')
  } finally {
    probingProvider.value = null
  }
}

function viewModels(name: string) {
  const probe = store.probeResults.get(name)
  const modelsList = probe?.models ?? store.models.get(name) ?? []
  selectedProviderModels.value = {
    providerName: name,
    models: modelsList,
  }
  showModelsModal.value = true
}

function probeStatus(name: string): 'ok' | 'error' | 'unknown' {
  const res = store.probeResults.get(name)
  if (!res) return 'unknown'
  return res.ok ? 'ok' : 'error'
}

function probeLabel(name: string): string {
  const res = store.probeResults.get(name)
  if (!res) return '—'
  if (res.ok) return res.latency_ms != null ? `${res.latency_ms} ms` : 'OK'
  return res.error ?? 'Unreachable'
}

function getModelCount(name: string): number {
  const probe = store.probeResults.get(name)
  if (probe?.models) return probe.models.length
  return store.models.get(name)?.length ?? 0
}
</script>

<template>
  <div class="psv">
    <div class="psv-header">
      <h2 class="psv-title">Provider Settings</h2>
      <div class="psv-header-actions">
        <button class="btn-secondary" :disabled="refreshing" @click="refresh">
          {{ refreshing ? 'Refreshing…' : 'Refresh' }}
        </button>
        <button class="btn-primary" @click="openAdd">Add Provider</button>
      </div>
    </div>

    <div v-if="store.loading && !store.providers.length" class="psv-state">Loading…</div>
    <div v-else-if="!store.providers.length" class="psv-state">
      No providers configured. Click <strong>Add Provider</strong> to register a local or cloud LLM provider.
    </div>

    <div v-else class="table-scroll">
      <table class="psv-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Base URL</th>
            <th>Driver</th>
            <th>API Key</th>
            <th>Health</th>
            <th>Latency</th>
            <th>Models</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="prov in store.providers" :key="prov.name">
            <td class="cell-name">{{ prov.name }}</td>
            <td class="cell-url">{{ prov.base_url }}</td>
            <td class="cell-driver">
              <span class="driver-badge">{{ prov.driver }}</span>
            </td>
            <td class="cell-key">
              <span v-if="prov.has_api_key || prov.api_key" class="key-badge">Configured</span>
              <span v-else class="key-badge key-badge--none">None</span>
            </td>
            <td class="cell-health">
              <span
                class="health-dot"
                :class="`health-dot--${probeStatus(prov.name)}`"
                :title="probeLabel(prov.name)"
              />
            </td>
            <td class="cell-latency">{{ probeLabel(prov.name) }}</td>
            <td class="cell-models">
              <button
                v-if="getModelCount(prov.name) > 0"
                class="btn-models"
                title="View detected models"
                @click="viewModels(prov.name)"
              >
                {{ getModelCount(prov.name) }} models
              </button>
              <span v-else class="cell-muted">—</span>
            </td>
            <td class="cell-actions">
              <button
                class="btn-icon"
                title="Test connection"
                :disabled="probingProvider === prov.name"
                @click="handleProbe(prov.name)"
              >
                {{ probingProvider === prov.name ? '…' : '⚡' }}
              </button>
              <button class="btn-icon" title="Edit" @click="openEdit(prov)">✎</button>
              <button class="btn-icon btn-icon--danger" title="Delete" @click="openDelete(prov)">✕</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Add / Edit modal -->
    <Teleport to="body">
      <div v-if="showModal" class="modal-overlay">
        <div class="modal-panel" role="dialog" aria-modal="true" :aria-label="editTarget ? 'Edit provider' : 'Add provider'">
          <div class="modal-header">
            <h3 class="modal-title">{{ editTarget ? 'Edit Provider' : 'Add Provider' }}</h3>
            <button class="modal-close" aria-label="Close" @click="closeModal">✕</button>
          </div>
          <div class="modal-body">
            <ProviderForm
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
      <div v-if="deleteTarget" class="modal-overlay">
        <div class="modal-panel" role="dialog" aria-modal="true" aria-label="Confirm deletion">
          <div class="modal-header">
            <h3 class="modal-title">Delete Provider</h3>
            <button class="modal-close" aria-label="Close" @click="deleteTarget = null">✕</button>
          </div>
          <div class="modal-body">
            <p class="confirm-text">
              Delete provider <strong>{{ deleteTarget.name }}</strong>? This cannot be undone.
            </p>
            <p v-if="deleteError" class="confirm-error">{{ deleteError }}</p>
            <div class="modal-actions">
              <button class="btn-secondary" @click="deleteTarget = null">Cancel</button>
              <button class="btn-danger" @click="confirmDelete">Delete</button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Models viewer modal -->
    <Teleport to="body">
      <div v-if="showModelsModal && selectedProviderModels" class="modal-overlay">
        <div class="modal-panel" role="dialog" aria-modal="true" :aria-label="`Models for ${selectedProviderModels.providerName}`">
          <div class="modal-header">
            <h3 class="modal-title">Models — {{ selectedProviderModels.providerName }}</h3>
            <button class="modal-close" aria-label="Close" @click="showModelsModal = false">✕</button>
          </div>
          <div class="modal-body models-modal-body">
            <div v-if="!selectedProviderModels.models.length" class="cell-muted">
              No models returned by this provider.
            </div>
            <ul v-else class="models-list">
              <li v-for="m in selectedProviderModels.models" :key="m.id" class="model-item">
                <span class="model-id">{{ m.id }}</span>
                <span v-if="m.name && m.name !== m.id" class="model-name">{{ m.name }}</span>
                <span
                  v-if="m.supported_parameters?.includes('tools')"
                  class="model-tool-badge"
                  title="Advertises tool calling support"
                >
                  tools
                </span>
              </li>
            </ul>
            <div class="modal-actions">
              <button class="btn-secondary" @click="showModelsModal = false">Close</button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.psv {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}
.psv-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.psv-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}
.psv-header-actions {
  display: flex;
  gap: var(--space-2);
}
.psv-state {
  padding: var(--space-8) var(--space-6);
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.psv-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-sm);
}
.psv-table th {
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
.psv-table td {
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid var(--color-border);
  vertical-align: middle;
}
.cell-name { font-weight: 600; color: var(--color-text); }
.cell-url { font-family: monospace; font-size: 12px; color: var(--color-text-muted); }
.cell-driver { font-size: 12px; }
.cell-key { font-size: 12px; }
.cell-health { width: 40px; text-align: center; }
.cell-latency { font-size: 12px; color: var(--color-text-muted); min-width: 80px; }
.cell-models { font-size: 12px; }
.cell-muted { color: var(--color-text-muted); font-size: 12px; }
.cell-actions {
  display: flex;
  gap: var(--space-2);
  align-items: center;
}
.driver-badge {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 500;
  background: #d1fae5;
  color: #065f46;
}
.key-badge {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 99px;
  font-size: 10px;
  font-weight: 500;
  background: var(--color-surface);
  color: var(--color-text);
  border: 1px solid var(--color-border);
}
.key-badge--none {
  color: var(--color-text-muted);
  opacity: 0.7;
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

.btn-models {
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  padding: 2px 8px;
  font-size: 11px;
  color: var(--color-accent);
  cursor: pointer;
}
.btn-models:hover {
  background: var(--color-surface);
  border-color: var(--color-accent);
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
.btn-primary:hover { opacity: 0.88; }
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
.btn-icon {
  background: none;
  border: none;
  padding: 4px 6px;
  font-size: 14px;
  cursor: pointer;
  color: var(--color-text-muted);
  border-radius: var(--radius-sm);
}
.btn-icon:hover:not(:disabled) { background: var(--color-surface); color: var(--color-text); }
.btn-icon:disabled { opacity: 0.4; cursor: not-allowed; }
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
  max-width: 540px;
  max-height: 85vh;
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
  flex-shrink: 0;
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
  overflow-y: auto;
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
.modal-actions {
  display: flex;
  gap: var(--space-2);
  justify-content: flex-end;
  margin-top: var(--space-4);
}
.models-list {
  list-style: none;
  margin: 0 0 var(--space-4);
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  max-height: 320px;
  overflow-y: auto;
}
.model-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  font-size: 13px;
}
.model-id {
  font-family: monospace;
  font-weight: 600;
  color: var(--color-text);
}
.model-name {
  color: var(--color-text-muted);
  font-size: 12px;
}
.model-tool-badge {
  font-size: 10px;
  font-weight: 600;
  padding: 1px 5px;
  border-radius: 99px;
  background: #dbeafe;
  color: #1d4ed8;
  margin-left: auto;
}
</style>
