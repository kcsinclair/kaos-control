// SPDX-License-Identifier: AGPL-3.0-or-later

export interface FailoverAgent {
  agent: string
  is_failover: boolean
  primary_provider?: string
  primary_model?: string
  active_provider: string
  active_model: string
  fallback_provider?: string
  fallback_model?: string
  switched_at?: string
  reason?: string
  primary_healthy?: boolean
}

export interface FailoverStatus {
  failover_active: boolean
  agents: FailoverAgent[]
}

export interface ProviderTemplateAgentBinding {
  provider: string
  model: string
}

export interface ProviderTemplate {
  name: string
  description?: string
  agents: Record<string, ProviderTemplateAgentBinding>
}

export interface SwitchProviderPayload {
  provider: string
  model: string
  reason?: string
}

// Wire shape for POST /provider-switch/switch-all — matches the backend's
// switchAllProviders request struct (to_model, not model).
export interface SwitchAllPayload {
  from_provider: string
  to_provider: string
  to_model: string
  reason?: string
}
