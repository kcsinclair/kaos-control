// SPDX-License-Identifier: AGPL-3.0-or-later

/** Rate-limit reset bucket recorded alongside a rate-limit failover (FR-3.3). */
export type ResetBucket = 'five_hour' | 'weekly'

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
  /** Authoritative rate-limit reset time (FR-9.2) — Unix seconds. */
  resets_at_unix?: number
  bucket?: ResetBucket
  /** FR-3.4: this agent has no secondary to fail over to; its jobs are paused. */
  partial_pause?: boolean
  /** FR-7.3: a job for this agent was interrupted with a suspected partial
   *  commit and needs an operator decision. */
  awaiting_decision?: boolean
  awaiting_decision_job_id?: string
}

/** Most recently probed reachability of one provider (FR-5, all modes). */
export interface ProviderReachability {
  healthy: boolean
  last_probed_at?: number
  since?: number
}

export interface FailoverStatus {
  failover_active: boolean
  agents: FailoverAgent[]
  reachability: Record<string, ProviderReachability>
}

/** Effective event -> action switchover policy (FR-2.4), defaults resolved. */
export interface SwitchoverPolicy {
  automated_switchover: boolean
  /** reason -> action, one entry per classified reason. */
  actions: Record<string, string>
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
