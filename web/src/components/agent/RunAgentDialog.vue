<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useAgentsStore } from '@/stores/agents'
import { useUiStore } from '@/stores/ui'
import type { AgentSummary } from '@/types/api'

const props = defineProps<{
  project: string
  targetPath?: string
}>()

const emit = defineEmits<{
  started: [runId: string]
  cancel: []
}>()

const store = useAgentsStore()
const ui = useUiStore()

const selectedAgent = ref<AgentSummary | null>(null)
const selectedRole = ref('')
const targetPath = ref(props.targetPath ?? '')
const starting = ref(false)
const error = ref<string | null>(null)

const roles = computed(() => selectedAgent.value?.roles ?? [])

function selectAgent(a: AgentSummary) {
  selectedAgent.value = a
  selectedRole.value = a.roles[0] ?? ''
}

async function start() {
  if (!selectedAgent.value || !targetPath.value) {
    error.value = 'Select an agent and provide a target path.'
    return
  }
  starting.value = true
  error.value = null
  try {
    const runId = await store.startRun(
      props.project,
      selectedAgent.value.name,
      targetPath.value,
      selectedRole.value || undefined,
    )
    ui.success(`Agent run started (${runId.slice(0, 8)}…)`)
    emit('started', runId)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to start run'
  } finally {
    starting.value = false
  }
}

onMounted(() => {
  if (!store.agents.length) store.fetchAgents(props.project)
})
</script>

<template>
  <Teleport to="body">
  <div class="rad-overlay" @click.self="emit('cancel')">
    <div class="rad-panel" role="dialog" aria-modal="true" aria-label="Run agent">
      <h3 class="rad-title">Run Agent</h3>

      <div v-if="!store.agents.length" class="rad-empty">No agents configured for this project.</div>
      <template v-else>
        <div class="rad-field">
          <div class="rad-label">Agent</div>
          <div class="agent-list">
            <button
              v-for="a in store.agents"
              :key="a.name"
              class="agent-chip"
              :class="{ 'agent-chip--active': selectedAgent?.name === a.name }"
              @click="selectAgent(a)"
            >
              <span class="agent-name">{{ a.name }}</span>
              <span class="agent-driver">{{ a.driver }}</span>
            </button>
          </div>
        </div>

        <label class="rad-field" v-if="roles.length > 1">
          <span class="rad-label">Role</span>
          <select class="rad-select" v-model="selectedRole">
            <option v-for="r in roles" :key="r" :value="r">{{ r }}</option>
          </select>
        </label>

        <label class="rad-field">
          <span class="rad-label">Target artifact path</span>
          <input
            class="rad-input"
            type="text"
            v-model="targetPath"
            placeholder="lifecycle/requirements/…"
          />
        </label>

        <div v-if="error" class="rad-error">{{ error }}</div>

        <div class="rad-actions">
          <button
            class="btn-primary"
            :disabled="starting || !selectedAgent || !targetPath"
            @click="start"
          >
            {{ starting ? 'Starting…' : 'Run' }}
          </button>
          <button class="btn-ghost" :disabled="starting" @click="emit('cancel')">Cancel</button>
        </div>
      </template>
    </div>
  </div>
  </Teleport>
</template>

<style scoped>
.rad-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 300;
}
.rad-panel {
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  padding: var(--space-6);
  width: 420px;
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
.rad-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}
.rad-empty {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.rad-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.rad-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
}
.agent-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.agent-chip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: none;
  cursor: pointer;
  text-align: left;
  font-family: inherit;
}
.agent-chip:hover { background: var(--color-surface); }
.agent-chip--active { border-color: var(--color-accent); background: color-mix(in srgb, var(--color-accent) 10%, transparent); }
.agent-name { font-size: var(--text-sm); font-weight: 500; color: var(--color-text); }
.agent-driver { font-size: 11px; color: var(--color-text-muted); font-family: monospace; }
.rad-select, .rad-input {
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
.rad-select:focus, .rad-input:focus { outline: none; border-color: var(--color-accent); }
.rad-error {
  font-size: var(--text-sm);
  color: #dc2626;
  background: #fee2e2;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
}
.rad-actions { display: flex; gap: var(--space-2); }
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
</style>
