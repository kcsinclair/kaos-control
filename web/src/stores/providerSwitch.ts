// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as providerSwitchApi from '@/api/providerSwitch'
import type { FailoverAgent, FailoverStatus, ProviderTemplate, SwitchProviderPayload } from '@/types/providerSwitch'

const emptyStatus = (): FailoverStatus => ({ failover_active: false, agents: [] })

export const useProviderSwitchStore = defineStore('providerSwitch', () => {
  const status = ref<FailoverStatus>(emptyStatus())
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
      status.value = { failover_active: data.failover_active, agents: data.agents ?? [] }
      pruneRecoveredProviders()
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to load provider failover status'
    } finally {
      loading.value = false
    }
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
  // provider.primary_recovered / config.reloaded events broadcast on the
  // project WS hub (see WorkspaceView.vue's subscribeWs). Unlike sibling
  // stores' onWsEvent (agents, scheduler), this one takes `project`: batch
  // switch/restore/template operations don't carry per-agent payloads, so
  // config.reloaded — which the backend broadcasts after every provider
  // mutation, in addition to the specific event — is the fallback that
  // refetches authoritative state. Single-agent switch/restore events patch
  // state locally for the plan's <100ms responsiveness requirement.
  function onWsEvent(project: string, type: string, payload: Record<string, unknown>): void {
    switch (type) {
      case 'provider.switched': {
        const agentName = payload.agent as string | undefined
        if (!agentName) break // batch switch-all — no per-agent identity; config.reloaded will refetch
        const existing = findAgent(agentName)
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
            }
          }
        }
        recomputeFailoverActive()
        pruneRecoveredProviders()
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
    fetchStatus,
    fetchTemplates,
    switchAgent,
    restoreAgent,
    restoreAll,
    applyTemplate,
    onWsEvent,
  }
})
