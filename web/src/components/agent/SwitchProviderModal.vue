<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useProvidersStore } from '@/stores/providers'
import { useProviderSwitchStore } from '@/stores/providerSwitch'
import { useQueueStore } from '@/stores/queue'
import { useUiStore } from '@/stores/ui'
import { ApiError } from '@/api/client'
import type { RunningJobInfo } from '@/types/providerSwitch'

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
const queueStore = useQueueStore()
const ui = useUiStore()

const provider = ref('')
const model = ref('')
const reason = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)
const rejectedRunningJobs = ref<RunningJobInfo[] | null>(null)

// FR-8.2: a run is currently executing for this project — the backend
// rejects a manual switch outright, so block the submit button up front
// rather than letting the operator hit the 409 first.
const hasRunningJob = computed(() => queueStore.snapshot.running != null)

// The rejection warning shows once the backend has actually rejected a
// submit (rejectedRunningJobs set, even to []) or proactively once the queue
// store already knows a job is running for this project.
const showRunningWarning = computed(() => rejectedRunningJobs.value !== null || hasRunningJob.value)

// Jobs to name in the warning: the backend's 409 body once rejected, else
// (proactively, before submit) whatever the queue store already knows is running.
const runningJobsToShow = computed<RunningJobInfo[]>(() => {
  if (rejectedRunningJobs.value) return rejectedRunningJobs.value
  const running = queueStore.snapshot.running
  if (!running) return []
  return [{ id: running.id, agent: running.agent_name, artifact_path: running.artifact_path }]
})

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
  if (hasRunningJob.value) return
  if (!provider.value || !model.value.trim()) {
    error.value = 'Provider and model are required.'
    return
  }
  error.value = null
  rejectedRunningJobs.value = null
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
    if (e instanceof ApiError && e.code === 'runs_in_progress') {
      const body = e.body as { running_jobs?: RunningJobInfo[] } | undefined
      rejectedRunningJobs.value = body?.running_jobs ?? []
    } else {
      error.value = e instanceof Error ? e.message : 'Failed to switch provider'
    }
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

        <div v-if="showRunningWarning" class="spm-running-warning" role="alert">
          <p class="spm-running-warning-heading">
            Cannot switch provider while runs are in progress (FR-8.2).
          </p>
          <ul v-if="runningJobsToShow.length" class="spm-running-jobs-list">
            <li v-for="job in runningJobsToShow" :key="job.id">
              <span class="spm-running-job-agent">{{ job.agent }}</span> — {{ job.artifact_path }}
            </li>
          </ul>
        </div>

        <p v-if="error" class="spm-error">{{ error }}</p>

        <div class="modal-footer">
          <button type="submit" class="btn-primary" :disabled="submitting || hasRunningJob">
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
.spm-running-warning {
  padding: var(--space-3);
  background: var(--badge-blocked-bg);
  border: 1px solid var(--color-error);
  border-radius: var(--radius-sm);
  color: var(--badge-blocked-text);
}
.spm-running-warning-heading {
  font-size: var(--text-sm);
  font-weight: 600;
  margin: 0;
}
.spm-running-jobs-list {
  margin: var(--space-2) 0 0 0;
  padding-left: var(--space-4);
  font-size: 12px;
}
.spm-running-job-agent {
  font-weight: 600;
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
