// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as providerSwitchApi from '@/api/providerSwitch'
import * as configApi from '@/api/config'
import { useAgentsStore } from '@/stores/agents'
import type {
  FailoverAgent,
  FailoverStatus,
  ProviderTemplate,
  SwitchoverPolicy,
  SwitchProviderPayload,
} from '@/types/providerSwitch'

const emptyStatus = (): FailoverStatus => ({ failover_active: false, agents: [], reachability: {} })

/** FR-1 mode of operation, derived from configured secondaries + the policy toggle. */
export type SwitchoverMode = 'single' | 'multiple' | 'manual' | 'automated'

/** Which side the project is currently operating on (FR-9.1); 'partial' when
 *  FR-3.4 has left at least one agent paused with no secondary. */
export type SwitchoverSide = 'primary' | 'secondary' | 'partial'

export const useProviderSwitchStore = defineStore('providerSwitch', () => {
  const agentsStore = useAgentsStore()
  const status = ref<FailoverStatus>(emptyStatus())
  const policy = ref<SwitchoverPolicy | null>(null)
  const templates = ref<ProviderTemplate[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  // Providers a recovery probe has confirmed reachable while at least one
  // agent is still failed over to them (project/recovery_prober.go).
  const recoveredProviders = ref<string[]>([])
  // Whether the global ProviderFailoverModal drawer is open — toggled by the
  // AppHeader badge, read by WorkspaceView (which hosts the modal).
  const modalOpen = ref(false)
  function openModal(): void { modalOpen.value = true }
  function closeModal(): void { modalOpen.value = false }

  const isFailoverActive = computed(() => status.value.failover_active)
  const failoverAgents = computed(() => status.value.agents.filter((a) => a.is_failover))
  const failoverCount = computed(() => failoverAgents.value.length)
  const hasRecoveredPrimary = computed(() => recoveredProviders.value.length > 0)
  const partiallyPausedAgents = computed(() => status.value.agents.filter((a) => a.partial_pause))
  const awaitingDecisionAgents = computed(() => status.value.agents.filter((a) => a.awaiting_decision))

  // FR-1.1/FR-9.1: a secondary is configured for at least one agent — this is
  // what makes the status button and failback screen relevant at all.
  const hasSecondaryConfigured = computed(() =>
    agentsStore.agents.some((a) => !!a.fallback_provider),
  )

  // FR-1's four modes of operation. Single/multiple depend only on the
  // declared config (no secondary anywhere); manual vs automated depend on
  // the effective automated_switchover toggle once secondaries exist.
  const mode = computed<SwitchoverMode | null>(() => {
    if (!agentsStore.agents.length) return null
    if (!hasSecondaryConfigured.value) {
      const providers = new Set(agentsStore.agents.map((a) => a.provider).filter(Boolean))
      return providers.size > 1 ? 'multiple' : 'single'
    }
    return policy.value?.automated_switchover ? 'automated' : 'manual'
  })

  // FR-9.1: which side the project is currently operating on, for the status
  // button. 'partial' takes priority — it means action is needed regardless
  // of how many agents are still on their primary.
  const currentSide = computed<SwitchoverSide>(() => {
    if (partiallyPausedAgents.value.length > 0) return 'partial'
    return isFailoverActive.value ? 'secondary' : 'primary'
  })

  // Drops recovered-provider entries that no longer have any agent still
  // failed over to them (e.g. after a restore).
  function pruneRecoveredProviders(): void {
    const stillFailedOver = new Set(
      status.value.agents
        .filter((a) => a.is_failover && a.primary_provider)
        .map((a) => a.primary_provider as string),
    )
    recoveredProviders.value = recoveredProviders.value.filter((p) => stillFailedOver.has(p))
  }

  async function fetchStatus(project: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const data = await providerSwitchApi.getFailoverStatus(project)
      status.value = {
        failover_active: data.failover_active,
        agents: data.agents ?? [],
        reachability: data.reachability ?? {},
      }
      pruneRecoveredProviders()
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to load provider failover status'
    } finally {
      loading.value = false
    }
  }

  // fetchPolicy loads the effective event->action policy (FR-2.4): every
  // classified reason resolved to an action, defaults filled in.
  async function fetchPolicy(project: string): Promise<void> {
    try {
      policy.value = await providerSwitchApi.getSwitchoverPolicy(project)
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to load switchover policy'
    }
  }

  // setAutomatedSwitchover persists the FR-2.1 toggle (disabled by default)
  // via the raw-config round-trip: parse lifecycle/config.yaml, patch only
  // switchover.automated_switchover, write it back. Mirrors the pattern used
  // for agent config edits (AgentsRunsView.vue's handleAgentFormSubmit) so
  // every other config key survives untouched.
  async function setAutomatedSwitchover(project: string, enabled: boolean): Promise<void> {
    const res = await configApi.getConfig(project)
    const cfg = configApi.parseConfigYaml(res.raw) as Record<string, unknown>
    const switchover = (cfg.switchover as Record<string, unknown> | undefined) ?? {}
    switchover.automated_switchover = enabled
    cfg.switchover = switchover
    await configApi.updateConfig(project, configApi.dumpConfigYaml(cfg))
    await fetchPolicy(project)
  }

  async function fetchTemplates(project: string): Promise<void> {
    try {
      const data = await providerSwitchApi.listProviderTemplates(project)
      templates.value = data.templates ?? []
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to load provider templates'
    }
  }

  async function switchAgent(project: string, agent: string, payload: SwitchProviderPayload): Promise<void> {
    error.value = null
    try {
      await providerSwitchApi.switchAgentProvider(project, agent, payload)
      await fetchStatus(project)
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to switch provider'
      throw e
    }
  }

  async function restoreAgent(project: string, agent: string): Promise<void> {
    error.value = null
    try {
      await providerSwitchApi.restoreAgentProvider(project, agent)
      await fetchStatus(project)
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to restore primary provider'
      throw e
    }
  }

  async function restoreAll(project: string): Promise<void> {
    error.value = null
    try {
      await providerSwitchApi.restoreAllProviders(project)
      await fetchStatus(project)
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to restore all primary providers'
      throw e
    }
  }

  async function applyTemplate(project: string, template: string): Promise<void> {
    error.value = null
    try {
      await providerSwitchApi.applyProviderTemplate(project, template)
      await fetchStatus(project)
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to apply provider template'
      throw e
    }
  }

  function findAgent(name: string): FailoverAgent | undefined {
    return status.value.agents.find((a) => a.agent === name)
  }

  function recomputeFailoverActive(): void {
    status.value.failover_active = status.value.agents.some((a) => a.is_failover)
  }

  // onWsEvent handles the provider.switched / provider.restored /
  // provider.failover_project_wide / provider.primary_recovered /
  // config.reloaded events broadcast on the project WS hub (see
  // WorkspaceView.vue's subscribeWs). Unlike sibling stores' onWsEvent
  // (agents, scheduler), this one takes `project`: batch switch/restore/
  // template operations don't carry per-agent payloads, so those refetch
  // status directly rather than relying on config.reloaded — since
  // agent-switchover-and-failover Milestone 2, switch/failover/restore never
  // write lifecycle/config.yaml, so the fsnotify-driven config.reloaded
  // broadcast no longer follows a provider mutation. Single-agent
  // switch/restore events patch state locally for the plan's <100ms
  // responsiveness requirement (NFR-4).
  function onWsEvent(project: string, type: string, payload: Record<string, unknown>): void {
    switch (type) {
      case 'provider.switched': {
        const agentName = payload.agent as string | undefined
        if (!agentName) {
          // Batch switch-all / template-apply — no per-agent identity to patch.
          void fetchStatus(project)
          break
        }
        const existing = findAgent(agentName)
        // Fields the payload never carries (resets_at_unix/bucket/partial_pause/
        // awaiting_decision) fall back to the last known value; the
        // provider.failover_project_wide handler below refetches authoritative
        // state shortly after for the automated-failover path that sets them.
        const patched: FailoverAgent = {
          agent: agentName,
          is_failover: (payload.is_failover as boolean | undefined) ?? true,
          primary_provider: (payload.primary_provider as string) || undefined,
          primary_model: (payload.primary_model as string) || undefined,
          active_provider: (payload.provider as string | undefined) ?? existing?.active_provider ?? '',
          active_model: (payload.model as string | undefined) ?? existing?.active_model ?? '',
          fallback_provider: existing?.fallback_provider,
          fallback_model: existing?.fallback_model,
          primary_healthy: existing?.primary_healthy,
          switched_at: (payload.switched_at as string | undefined) ?? new Date().toISOString(),
          reason: (payload.reason as string | undefined) ?? existing?.reason,
          resets_at_unix: existing?.resets_at_unix,
          bucket: existing?.bucket,
          partial_pause: false,
          awaiting_decision: existing?.awaiting_decision,
          awaiting_decision_job_id: existing?.awaiting_decision_job_id,
        }
        const idx = status.value.agents.findIndex((a) => a.agent === agentName)
        if (idx >= 0) status.value.agents[idx] = patched
        else status.value.agents.push(patched)
        recomputeFailoverActive()
        break
      }
      case 'provider.restored': {
        const names = (payload.agents as string[] | undefined) ?? (payload.agent ? [payload.agent as string] : [])
        for (const name of names) {
          const idx = status.value.agents.findIndex((a) => a.agent === name)
          if (idx >= 0) {
            status.value.agents[idx] = {
              ...status.value.agents[idx],
              is_failover: false,
              primary_provider: undefined,
              primary_model: undefined,
              primary_healthy: undefined,
              switched_at: undefined,
              reason: undefined,
              resets_at_unix: undefined,
              bucket: undefined,
              partial_pause: false,
            }
          }
        }
        recomputeFailoverActive()
        pruneRecoveredProviders()
        break
      }
      case 'provider.failover_project_wide': {
        // FR-3.1 automated failover: per-agent provider.switched events have
        // already patched the switched agents, but resets_at_unix/bucket and
        // FR-3.4 partial-pause entries (no per-agent event at all) are only
        // authoritative from a refetch.
        void fetchStatus(project)
        break
      }
      case 'provider.primary_recovered': {
        const provider = payload.provider as string | undefined
        if (!provider) break
        if (!recoveredProviders.value.includes(provider)) recoveredProviders.value.push(provider)
        status.value.agents = status.value.agents.map((a) =>
          a.is_failover && a.primary_provider === provider ? { ...a, primary_healthy: true } : a,
        )
        break
      }
      case 'config.reloaded': {
        void fetchStatus(project)
        void fetchTemplates(project)
        break
      }
    }
  }

  return {
    status,
    policy,
    templates,
    loading,
    error,
    recoveredProviders,
    modalOpen,
    openModal,
    closeModal,
    isFailoverActive,
    failoverAgents,
    failoverCount,
    hasRecoveredPrimary,
    partiallyPausedAgents,
    awaitingDecisionAgents,
    hasSecondaryConfigured,
    mode,
    currentSide,
    fetchStatus,
    fetchPolicy,
    setAutomatedSwitchover,
    fetchTemplates,
    switchAgent,
    restoreAgent,
    restoreAll,
    applyTemplate,
    onWsEvent,
  }
})
