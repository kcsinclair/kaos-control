<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useProvidersStore } from '@/stores/providers'
import { useProviderSwitchStore } from '@/stores/providerSwitch'
import { useUiStore } from '@/stores/ui'

const props = defineProps<{
  project: string
  agentName: string
}>()

const emit = defineEmits<{
  switched: []
  cancel: []
}>()

const providersStore = useProvidersStore()
const providerSwitchStore = useProviderSwitchStore()
const ui = useUiStore()

const provider = ref('')
const model = ref('')
const reason = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

const providerModels = computed(() => {
  if (!provider.value) return []
  return providersStore.models.get(provider.value) ?? []
})

watch(provider, (val) => {
  model.value = ''
  if (val) void providersStore.fetchModels(val)
})

onMounted(async () => {
  if (!providersStore.providers.length) {
    await providersStore.fetchProviders().catch(() => {})
  }
})

async function handleSubmit() {
  if (!provider.value || !model.value.trim()) {
    error.value = 'Provider and model are required.'
    return
  }
  error.value = null
  submitting.value = true
  try {
    await providerSwitchStore.switchAgent(props.project, props.agentName, {
      provider: provider.value,
      model: model.value.trim(),
      reason: reason.value.trim() || undefined,
    })
    ui.success(`${props.agentName} switched to ${provider.value}/${model.value.trim()}`)
    emit('switched')
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to switch provider'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="modal-overlay" @click.self="emit('cancel')">
    <div class="modal-panel" role="dialog" aria-modal="true" :aria-label="`Switch provider for ${agentName}`">
      <div class="modal-header">
        <h3 class="modal-title">Switch Provider — {{ agentName }}</h3>
        <button class="btn-icon" aria-label="Cancel" @click="emit('cancel')">✕</button>
      </div>

      <form class="modal-body" @submit.prevent="handleSubmit">
        <div class="spm-field">
          <label class="spm-label" for="spm-provider">Target Provider</label>
          <select id="spm-provider" v-model="provider" class="spm-select">
            <option value="">— select provider —</option>
            <option v-for="p in providersStore.providers" :key="p.name" :value="p.name">
              {{ p.name }} ({{ p.base_url }})
            </option>
          </select>
        </div>

        <div class="spm-field">
          <label class="spm-label" for="spm-model">Target Model</label>
          <select
            v-if="providerModels.length"
            id="spm-model"
            v-model="model"
            class="spm-select"
          >
            <option value="">— select model —</option>
            <option v-for="m in providerModels" :key="m.id" :value="m.id">
              {{ m.name && m.name !== m.id ? `${m.name} (${m.id})` : m.id }}
            </option>
          </select>
          <input
            v-else
            id="spm-model"
            v-model="model"
            class="spm-input"
            type="text"
            placeholder="e.g. gemini-2.5-flash"
            autocomplete="off"
          />
        </div>

        <div class="spm-field">
          <label class="spm-label" for="spm-reason">Reason <span class="spm-optional">(optional)</span></label>
          <input
            id="spm-reason"
            v-model="reason"
            class="spm-input"
            type="text"
            placeholder="e.g. Manual operator switch"
            autocomplete="off"
          />
        </div>

        <p v-if="error" class="spm-error">{{ error }}</p>

        <div class="modal-footer">
          <button type="submit" class="btn-primary" :disabled="submitting">
            {{ submitting ? 'Switching…' : 'Switch Target' }}
          </button>
          <button type="button" class="btn-ghost" :disabled="submitting" @click="emit('cancel')">Cancel</button>
        </div>
      </form>
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
  width: 460px;
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
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-4) var(--space-6);
}
.spm-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.spm-label {
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
}
.spm-optional {
  font-weight: 400;
  color: var(--color-text-muted);
  font-size: 12px;
}
.spm-select,
.spm-input {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: var(--text-sm);
  font-family: inherit;
  outline: none;
}
.spm-select:focus,
.spm-input:focus { border-color: var(--color-accent); }
.spm-error {
  font-size: 12px;
  color: var(--color-error);
  margin: 0;
}
.modal-footer {
  display: flex;
  gap: var(--space-2);
  padding-top: var(--space-2);
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
.btn-ghost:hover { background: var(--color-surface); }
.btn-ghost:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
