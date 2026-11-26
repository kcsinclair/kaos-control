<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useProviderSwitchStore } from '@/stores/providerSwitch'
import { useAuthStore } from '@/stores/auth'
import { useAgentsStore } from '@/stores/agents'
import { useUiStore } from '@/stores/ui'
import { useNow } from '@/composables/useNow'
import type { FailoverAgent, ResetBucket } from '@/types/providerSwitch'

const route = useRoute()
const project = route.params.project as string

const providerSwitchStore = useProviderSwitchStore()
const authStore = useAuthStore()
const agentsStore = useAgentsStore()
const ui = useUiStore()
const now = useNow()

onMounted(() => {
  void providerSwitchStore.fetchStatus(project)
  if (!agentsStore.agents.length) void agentsStore.fetchAgents(project)
})

const canManage = computed(() => {
  const roles = authStore.rolesForProject(project)
  return roles.includes('product-owner') || roles.includes('devops')
})

function roleFor(agentName: string): string | undefined {
  return agentsStore.agents.find((a) => a.name === agentName)?.roles?.[0]
}

// Agents that need attention here: currently failed over, or partially
// paused with no secondary (FR-3.4) — both are "not on primary" states.
const affectedAgents = computed(() =>
  providerSwitchStore.status.agents.filter((a) => a.is_failover || a.partial_pause),
)

const bucketLabel: Record<ResetBucket, string> = {
  five_hour: '5-hour window',
  weekly: 'weekly window',
}

function resetTimeLabel(agent: FailoverAgent): string | null {
  if (!agent.resets_at_unix) return null
  const resetDate = new Date(agent.resets_at_unix * 1000)
  return resetDate.toLocaleString()
}

function resetCountdown(agent: FailoverAgent): string | null {
  if (!agent.resets_at_unix) return null
  const diffMs = agent.resets_at_unix * 1000 - now.value.getTime()
  if (diffMs <= 0) return 'passed'
  const diffSec = Math.ceil(diffMs / 1000)
  if (diffSec < 60) return `in ${diffSec}s`
  const mins = Math.floor(diffSec / 60)
  if (mins < 60) return `in ${mins}m`
  const hrs = Math.floor(mins / 60)
  const remMins = mins % 60
  if (hrs < 24) return remMins > 0 ? `in ${hrs}h ${remMins}m` : `in ${hrs}h`
  const days = Math.floor(hrs / 24)
  return `in ${days}d`
}

// FR-9.3: the recorded reset time gates whether "recovered" may be shown at
// all — a healthy probe alone (agent.primary_healthy / reachability.healthy)
// is not sufficient evidence for a rate-limit failover. Recomputed
// independently here rather than trusting the WS-driven primary_healthy flag
// verbatim, so a stale flag can't slip an unqualified "recovered" onto screen.
function resetTimeHasPassed(agent: FailoverAgent): boolean {
  if (!agent.resets_at_unix) return true // not quota-gated (e.g. unreachable/auth_error)
  return now.value.getTime() >= agent.resets_at_unix * 1000
}

function primaryReachability(agent: FailoverAgent) {
  if (!agent.primary_provider) return undefined
  return providerSwitchStore.status.reachability[agent.primary_provider]
}

function recoveryState(agent: FailoverAgent): 'recovered' | 'reachable-pending-reset' | 'unreachable' | 'unknown' {
  const reach = primaryReachability(agent)
  if (!reach) return 'unknown'
  if (!reach.healthy) return 'unreachable'
  return resetTimeHasPassed(agent) ? 'recovered' : 'reachable-pending-reset'
}

function recoveryLabel(agent: FailoverAgent): string {
  switch (recoveryState(agent)) {
    case 'recovered':
      return 'Recovered & reachable'
    case 'reachable-pending-reset':
      return 'Reachable — reset time not yet confirmed passed'
    case 'unreachable':
      return 'Unavailable'
    default:
      return 'Not yet probed'
  }
}

function lastProbedLabel(agent: FailoverAgent): string | null {
  const reach = primaryReachability(agent)
  if (!reach?.last_probed_at) return null
  return new Date(reach.last_probed_at * 1000).toLocaleString()
}

const restoring = ref<string | null>(null)
const restoringAll = ref(false)

async function failback(agentName: string) {
  restoring.value = agentName
  try {
    await providerSwitchStore.restoreAgent(project, agentName)
    ui.success(`${agentName} failed back to primary provider`)
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to fail back to primary provider')
  } finally {
    restoring.value = null
  }
}

async function failbackAll() {
  restoringAll.value = true
  try {
    await providerSwitchStore.restoreAll(project)
    ui.success('All agents failed back to primary providers')
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to fail back all agents')
  } finally {
    restoringAll.value = false
  }
}
</script>

<template>
  <div class="failback-view">
    <div class="view-header">
      <div class="header-left">
        <h2 class="view-title">Fail Back to Primary</h2>
        <p class="view-subtitle">
          Failback is manual only. Review the expected reset time and reachability
          before switching an agent back to its primary provider.
        </p>
      </div>
      <div v-if="canManage" class="header-actions">
        <button
          class="btn-primary"
          :disabled="!affectedAgents.length || restoringAll"
          @click="failbackAll"
        >
          {{ restoringAll ? 'Failing back…' : 'Fail Back All' }}
        </button>
      </div>
    </div>

    <div v-if="!affectedAgents.length" class="empty-state">
      <span class="empty-icon" aria-hidden="true">✓</span>
      <p>All agents are operating on their primary providers.</p>
    </div>

    <div v-else class="agent-table-wrap">
      <table class="agent-table">
        <thead>
          <tr>
            <th>Agent</th>
            <th>Active</th>
            <th>Primary</th>
            <th>Reason</th>
            <th>Switched</th>
            <th>Expected reset</th>
            <th>Primary reachability</th>
            <th v-if="canManage">Action</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="agent in affectedAgents" :key="agent.agent">
            <td>
              <span class="agent-name">{{ agent.agent }}</span>
              <span v-if="roleFor(agent.agent)" class="agent-role">{{ roleFor(agent.agent) }}</span>
            </td>
            <td>
              <span v-if="agent.partial_pause" class="value-muted">paused (no secondary)</span>
              <span v-else class="value-active">{{ agent.active_provider }} ({{ agent.active_model }})</span>
            </td>
            <td>{{ agent.primary_provider || '—' }}<template v-if="agent.primary_model"> ({{ agent.primary_model }})</template></td>
            <td>{{ agent.reason || '—' }}</td>
            <td>{{ agent.switched_at ? new Date(agent.switched_at).toLocaleString() : '—' }}</td>
            <td>
              <template v-if="agent.resets_at_unix">
                <div>{{ resetTimeLabel(agent) }}</div>
                <div class="reset-countdown">
                  {{ resetCountdown(agent) }}
                  <span v-if="agent.bucket" class="reset-bucket">({{ bucketLabel[agent.bucket] }})</span>
                </div>
              </template>
              <span v-else class="value-muted">not rate-limit gated</span>
            </td>
            <td>
              <span class="reach-badge" :class="`reach-badge--${recoveryState(agent)}`">
                {{ recoveryLabel(agent) }}
              </span>
              <div v-if="lastProbedLabel(agent)" class="reach-probed">last probed {{ lastProbedLabel(agent) }}</div>
            </td>
            <td v-if="canManage">
              <button
                v-if="!agent.partial_pause"
                class="btn-secondary"
                :disabled="restoring === agent.agent"
                @click="failback(agent.agent)"
              >
                {{ restoring === agent.agent ? 'Failing back…' : 'Fail back' }}
              </button>
              <span v-else class="value-muted">restore its secondary config to resume</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.failback-view {
  padding: var(--space-6);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
.view-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-4);
}
.view-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0 0 var(--space-1) 0;
  color: var(--color-text);
}
.view-subtitle {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  margin: 0;
  max-width: 60ch;
}
.header-actions {
  flex-shrink: 0;
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
.agent-table-wrap {
  overflow-x: auto;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}
.agent-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-sm);
}
.agent-table th {
  text-align: left;
  padding: var(--space-2) var(--space-3);
  background: var(--color-surface);
  color: var(--color-text-muted);
  font-weight: 600;
  border-bottom: 1px solid var(--color-border);
  white-space: nowrap;
}
.agent-table td {
  padding: var(--space-2) var(--space-3);
  border-bottom: 1px solid var(--color-border);
  vertical-align: top;
  color: var(--color-text);
}
.agent-table tr:last-child td {
  border-bottom: none;
}
.agent-name {
  font-weight: 600;
}
.agent-role {
  margin-left: var(--space-2);
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 99px;
  background: var(--color-surface);
  color: var(--color-text-muted);
  border: 1px solid var(--color-border);
}
.value-active {
  color: #f59e0b;
  font-weight: 600;
}
.value-muted {
  color: var(--color-text-muted);
}
.reset-countdown {
  font-size: 12px;
  color: var(--color-text-muted);
}
.reset-bucket {
  font-size: 11px;
}
.reach-badge {
  display: inline-block;
  font-size: 11px;
  font-weight: 600;
  padding: 1px 8px;
  border-radius: 99px;
  white-space: nowrap;
}
.reach-badge--recovered {
  background: rgba(34, 197, 94, 0.15);
  color: #16a34a;
  border: 1px solid rgba(34, 197, 94, 0.35);
}
.reach-badge--reachable-pending-reset {
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
  border: 1px solid rgba(245, 158, 11, 0.35);
}
.reach-badge--unreachable {
  background: rgba(239, 68, 68, 0.12);
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.35);
}
.reach-badge--unknown {
  background: var(--color-surface);
  color: var(--color-text-muted);
  border: 1px solid var(--color-border);
}
.reach-probed {
  font-size: 11px;
  color: var(--color-text-muted);
  margin-top: 2px;
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
  white-space: nowrap;
}
.btn-secondary:hover:not(:disabled) { border-color: var(--color-text-muted); color: var(--color-text); }
.btn-secondary:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
