<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useProviderSwitchStore } from '@/stores/providerSwitch'
import { useAuthStore } from '@/stores/auth'
import { useAgentsStore } from '@/stores/agents'
import { useUiStore } from '@/stores/ui'
import ProviderTemplateMenu from './ProviderTemplateMenu.vue'
import type { FailoverAgent } from '@/types/providerSwitch'

const props = defineProps<{
  project: string
}>()

const emit = defineEmits<{
  close: []
}>()

const providerSwitchStore = useProviderSwitchStore()
const authStore = useAuthStore()
const agentsStore = useAgentsStore()
const ui = useUiStore()

const restoring = ref<string | null>(null)
const restoringAll = ref(false)

const canManage = computed(() => {
  const roles = authStore.rolesForProject(props.project)
  return roles.includes('product-owner') || roles.includes('devops')
})

function roleFor(agentName: string): string | undefined {
  return agentsStore.agents.find((a) => a.name === agentName)?.roles?.[0]
}

function reasonLabel(agent: FailoverAgent): string | null {
  if (!agent.reason) return null
  if (!agent.switched_at) return agent.reason
  return `${agent.reason} — ${new Date(agent.switched_at).toLocaleString()}`
}

async function restore(agentName: string) {
  restoring.value = agentName
  try {
    await providerSwitchStore.restoreAgent(props.project, agentName)
    ui.success(`${agentName} restored to primary provider`)
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to restore primary provider')
  } finally {
    restoring.value = null
  }
}

async function restoreAll() {
  restoringAll.value = true
  try {
    await providerSwitchStore.restoreAll(props.project)
    ui.success('All agents restored to primary providers')
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to restore all primary providers')
  } finally {
    restoringAll.value = false
  }
}

onMounted(() => {
  if (!providerSwitchStore.templates.length) {
    void providerSwitchStore.fetchTemplates(props.project)
  }
})
</script>

<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal-panel" role="dialog" aria-modal="true" aria-label="Active Provider Failovers">
      <div class="modal-header">
        <div class="modal-title-row">
          <h3 class="modal-title">Active Provider Failovers</h3>
          <span class="failover-count-badge">{{ providerSwitchStore.failoverCount }}</span>
        </div>
        <button class="btn-icon" aria-label="Close" @click="emit('close')">✕</button>
      </div>

      <div v-if="canManage" class="modal-toolbar">
        <button
          class="btn-primary"
          :disabled="!providerSwitchStore.failoverCount || restoringAll"
          @click="restoreAll"
        >
          {{ restoringAll ? 'Restoring…' : 'Restore All Primary Providers' }}
        </button>
        <ProviderTemplateMenu :project="props.project" />
      </div>

      <div class="modal-body">
        <div v-if="!providerSwitchStore.failoverAgents.length" class="empty-state">
          <span class="empty-icon" aria-hidden="true">✓</span>
          <p>All agents are operating on their primary providers.</p>
        </div>

        <div v-else class="agent-card-list">
          <div v-for="agent in providerSwitchStore.failoverAgents" :key="agent.agent" class="agent-card">
            <div class="agent-card-header">
              <span class="agent-card-name">{{ agent.agent }}</span>
              <span v-if="roleFor(agent.agent)" class="agent-card-role">{{ roleFor(agent.agent) }}</span>
            </div>

            <div class="agent-card-compare">
              <div class="compare-row compare-row--active">
                <span class="compare-label">Active (Fallback)</span>
                <span class="compare-value">{{ agent.active_provider }} ({{ agent.active_model }})</span>
              </div>
              <div class="compare-row compare-row--primary">
                <span class="compare-label">Original (Primary)</span>
                <span class="compare-value">{{ agent.primary_provider }} ({{ agent.primary_model }})</span>
              </div>
            </div>

            <p v-if="reasonLabel(agent)" class="agent-card-reason">{{ reasonLabel(agent) }}</p>

            <div class="agent-card-footer">
              <span
                v-if="agent.primary_healthy === true"
                class="health-badge health-badge--ok"
              >Recovered &amp; Reachable</span>
              <span
                v-else-if="agent.primary_healthy === false"
                class="health-badge health-badge--error"
              >Unavailable</span>

              <button
                v-if="canManage"
                class="btn-secondary"
                :disabled="restoring === agent.agent"
                @click="restore(agent.agent)"
              >
                {{ restoring === agent.agent ? 'Restoring…' : 'Restore Primary' }}
              </button>
            </div>
          </div>
        </div>
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
  width: 640px;
  max-width: calc(100vw - 2rem);
  max-height: calc(100vh - 4rem);
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
.modal-title-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.modal-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}
.failover-count-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 1.5rem;
  padding: 1px 7px;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
  border: 1px solid rgba(245, 158, 11, 0.35);
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
.modal-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-4) var(--space-6);
}
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-8) var(--space-4);
  color: var(--color-text-muted);
  font-size: var(--text-sm);
  text-align: center;
}
.empty-icon {
  font-size: 1.5rem;
  color: #22c55e;
}
.agent-card-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.agent-card {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-3) var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.agent-card-header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.agent-card-name {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
}
.agent-card-role {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 99px;
  background: var(--color-surface);
  color: var(--color-text-muted);
  border: 1px solid var(--color-border);
}
.agent-card-compare {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.compare-row {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
  font-size: 12px;
}
.compare-label {
  min-width: 110px;
  color: var(--color-text-muted);
  font-weight: 500;
}
.compare-row--active .compare-value {
  color: #f59e0b;
  font-weight: 600;
}
.compare-row--primary .compare-value {
  color: var(--color-text-muted);
}
.agent-card-reason {
  font-size: 11px;
  color: var(--color-text-muted);
  margin: 0;
}
.agent-card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
}
.health-badge {
  font-size: 11px;
  font-weight: 600;
  padding: 1px 8px;
  border-radius: 99px;
}
.health-badge--ok {
  background: rgba(34, 197, 94, 0.15);
  color: #16a34a;
  border: 1px solid rgba(34, 197, 94, 0.35);
}
.health-badge--error {
  background: rgba(239, 68, 68, 0.12);
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.35);
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
.btn-primary:hover:not(:disabled) { opacity: 0.88; }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-secondary {
  padding: var(--space-1) var(--space-3);
  background: transparent;
  color: var(--color-text-muted);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
}
.btn-secondary:hover:not(:disabled) { border-color: var(--color-text-muted); color: var(--color-text); }
.btn-secondary:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
