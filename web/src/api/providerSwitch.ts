// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from './client'
import type {
  FailoverStatus,
  ProviderTemplate,
  SwitchAllPayload,
  SwitchoverPolicy,
  SwitchProviderPayload,
} from '@/types/providerSwitch'

export function getFailoverStatus(project: string): Promise<FailoverStatus> {
  return api.get<FailoverStatus>(
    `/p/${encodeURIComponent(project)}/provider-switch/status`,
  )
}

// getSwitchoverPolicy returns the project's effective event->action policy
// (FR-2.4): automated_switchover and one action per classified reason,
// configured overrides applied on top of the FR-2.3 defaults.
export function getSwitchoverPolicy(project: string): Promise<SwitchoverPolicy> {
  return api.get<SwitchoverPolicy>(
    `/p/${encodeURIComponent(project)}/provider-switch/policy`,
  )
}

export function switchAgentProvider(
  project: string,
  agent: string,
  payload: SwitchProviderPayload,
): Promise<{ ok: boolean; agent: string; provider: string; model: string }> {
  return api.post(
    `/p/${encodeURIComponent(project)}/agents/${encodeURIComponent(agent)}/switch-provider`,
    payload,
  )
}

export function restoreAgentProvider(
  project: string,
  agent: string,
): Promise<{ ok: boolean; agent: string; provider: string; model: string }> {
  return api.post(
    `/p/${encodeURIComponent(project)}/agents/${encodeURIComponent(agent)}/restore-provider`,
  )
}

export function switchAllProviders(
  project: string,
  payload: SwitchAllPayload,
): Promise<{ ok: boolean; switched_agents: number; from_provider: string; to_provider: string }> {
  return api.post(
    `/p/${encodeURIComponent(project)}/provider-switch/switch-all`,
    payload,
  )
}

export function restoreAllProviders(
  project: string,
): Promise<{ ok: boolean; restored_agents: number }> {
  return api.post(`/p/${encodeURIComponent(project)}/provider-switch/restore-all`)
}

export function listProviderTemplates(
  project: string,
): Promise<{ templates: ProviderTemplate[] }> {
  return api.get(`/p/${encodeURIComponent(project)}/provider-templates`)
}

export function applyProviderTemplate(
  project: string,
  template: string,
): Promise<{ ok: boolean; template: string; updated_agents: number }> {
  return api.post(
    `/p/${encodeURIComponent(project)}/provider-templates/apply`,
    { template },
  )
}
