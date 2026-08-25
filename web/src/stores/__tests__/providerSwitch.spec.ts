// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useProviderSwitchStore } from '../providerSwitch'
import * as providerSwitchApi from '@/api/providerSwitch'

// Frontend plan: lifecycle/frontend-plans/switch-provider-4-fe.md
// Milestone 1 — REST client, TypeScript interfaces & Pinia store.

describe('providerSwitch store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('fetchStatus', () => {
    it('populates failover state from the API', async () => {
      vi.spyOn(providerSwitchApi, 'getFailoverStatus').mockResolvedValue({
        failover_active: true,
        agents: [
          {
            agent: 'requirements-analyst',
            is_failover: true,
            primary_provider: 'anthropic-cloud',
            primary_model: 'claude-3-7-sonnet',
            active_provider: 'gemini-cloud',
            active_model: 'gemini-2.5-flash',
            fallback_provider: 'gemini-cloud',
            fallback_model: 'gemini-2.5-flash',
          },
          {
            agent: 'backend-developer',
            is_failover: false,
            active_provider: 'anthropic-cloud',
            active_model: 'claude-3-7-sonnet',
          },
        ],
      })

      const store = useProviderSwitchStore()
      await store.fetchStatus('demo')

      expect(store.isFailoverActive).toBe(true)
      expect(store.failoverCount).toBe(1)
      expect(store.failoverAgents.map((a) => a.agent)).toEqual(['requirements-analyst'])
    })

    it('sets error and does not throw on failure', async () => {
      vi.spyOn(providerSwitchApi, 'getFailoverStatus').mockRejectedValue(new Error('network down'))

      const store = useProviderSwitchStore()
      await store.fetchStatus('demo')

      expect(store.error).toBe('network down')
      expect(store.status.agents).toEqual([])
    })
  })

  describe('WS events', () => {
    it('provider.switched patches the matching agent without a refetch', async () => {
      const spy = vi.spyOn(providerSwitchApi, 'getFailoverStatus')
      const store = useProviderSwitchStore()

      store.onWsEvent('demo', 'provider.switched', {
        agent: 'requirements-analyst',
        provider: 'gemini-cloud',
        model: 'gemini-2.5-flash',
        reason: 'HTTP 529 Overloaded',
        is_failover: true,
        primary_provider: 'anthropic-cloud',
        primary_model: 'claude-3-7-sonnet',
      })

      expect(spy).not.toHaveBeenCalled()
      expect(store.isFailoverActive).toBe(true)
      expect(store.failoverAgents).toHaveLength(1)
      const agent = store.failoverAgents[0]
      expect(agent.active_provider).toBe('gemini-cloud')
      expect(agent.active_model).toBe('gemini-2.5-flash')
      expect(agent.primary_provider).toBe('anthropic-cloud')
    })

    it('provider.restored clears is_failover on the matching agent', () => {
      const store = useProviderSwitchStore()
      store.onWsEvent('demo', 'provider.switched', {
        agent: 'requirements-analyst',
        provider: 'gemini-cloud',
        model: 'gemini-2.5-flash',
        is_failover: true,
        primary_provider: 'anthropic-cloud',
        primary_model: 'claude-3-7-sonnet',
      })
      expect(store.isFailoverActive).toBe(true)

      store.onWsEvent('demo', 'provider.restored', {
        agent: 'requirements-analyst',
        provider: 'anthropic-cloud',
        model: 'claude-3-7-sonnet',
      })

      expect(store.isFailoverActive).toBe(false)
      expect(store.failoverCount).toBe(0)
    })

    it('provider.restored with an agents array (restore-all) clears every listed agent', () => {
      const store = useProviderSwitchStore()
      store.onWsEvent('demo', 'provider.switched', { agent: 'a1', provider: 'p2', model: 'm2', is_failover: true, primary_provider: 'p1', primary_model: 'm1' })
      store.onWsEvent('demo', 'provider.switched', { agent: 'a2', provider: 'p2', model: 'm2', is_failover: true, primary_provider: 'p1', primary_model: 'm1' })
      expect(store.failoverCount).toBe(2)

      store.onWsEvent('demo', 'provider.restored', { agents: ['a1', 'a2'], count: 2 })

      expect(store.failoverCount).toBe(0)
      expect(store.isFailoverActive).toBe(false)
    })

    it('provider.primary_recovered marks recovered and updates primary_healthy', () => {
      const store = useProviderSwitchStore()
      store.onWsEvent('demo', 'provider.switched', {
        agent: 'requirements-analyst',
        provider: 'gemini-cloud',
        model: 'gemini-2.5-flash',
        is_failover: true,
        primary_provider: 'anthropic-cloud',
        primary_model: 'claude-3-7-sonnet',
      })

      store.onWsEvent('demo', 'provider.primary_recovered', { provider: 'anthropic-cloud', project: 'demo' })

      expect(store.hasRecoveredPrimary).toBe(true)
      expect(store.recoveredProviders).toContain('anthropic-cloud')
      expect(store.failoverAgents[0].primary_healthy).toBe(true)
    })

    it('config.reloaded triggers fetchStatus and fetchTemplates', async () => {
      const statusSpy = vi.spyOn(providerSwitchApi, 'getFailoverStatus').mockResolvedValue({ failover_active: false, agents: [] })
      const templatesSpy = vi.spyOn(providerSwitchApi, 'listProviderTemplates').mockResolvedValue({ templates: [] })

      const store = useProviderSwitchStore()
      store.onWsEvent('demo', 'config.reloaded', { agents: 3 })
      await Promise.resolve()
      await Promise.resolve()

      expect(statusSpy).toHaveBeenCalledWith('demo')
      expect(templatesSpy).toHaveBeenCalledWith('demo')
    })
  })

  describe('actions', () => {
    it('switchAgent invokes the API and refetches status', async () => {
      const switchSpy = vi.spyOn(providerSwitchApi, 'switchAgentProvider').mockResolvedValue({
        ok: true, agent: 'requirements-analyst', provider: 'gemini-cloud', model: 'gemini-2.5-flash',
      })
      const statusSpy = vi.spyOn(providerSwitchApi, 'getFailoverStatus').mockResolvedValue({ failover_active: false, agents: [] })

      const store = useProviderSwitchStore()
      await store.switchAgent('demo', 'requirements-analyst', { provider: 'gemini-cloud', model: 'gemini-2.5-flash', reason: 'manual' })

      expect(switchSpy).toHaveBeenCalledWith('demo', 'requirements-analyst', { provider: 'gemini-cloud', model: 'gemini-2.5-flash', reason: 'manual' })
      expect(statusSpy).toHaveBeenCalledWith('demo')
    })

    it('switchAgent propagates errors and sets store.error', async () => {
      vi.spyOn(providerSwitchApi, 'switchAgentProvider').mockRejectedValue(new Error('provider offline'))

      const store = useProviderSwitchStore()
      await expect(
        store.switchAgent('demo', 'requirements-analyst', { provider: 'gemini-cloud', model: 'gemini-2.5-flash' }),
      ).rejects.toThrow('provider offline')
      expect(store.error).toBe('provider offline')
    })

    it('restoreAgent invokes the API and refetches status', async () => {
      const restoreSpy = vi.spyOn(providerSwitchApi, 'restoreAgentProvider').mockResolvedValue({
        ok: true, agent: 'requirements-analyst', provider: 'anthropic-cloud', model: 'claude-3-7-sonnet',
      })
      vi.spyOn(providerSwitchApi, 'getFailoverStatus').mockResolvedValue({ failover_active: false, agents: [] })

      const store = useProviderSwitchStore()
      await store.restoreAgent('demo', 'requirements-analyst')

      expect(restoreSpy).toHaveBeenCalledWith('demo', 'requirements-analyst')
    })

    it('restoreAll invokes the API and refetches status', async () => {
      const restoreAllSpy = vi.spyOn(providerSwitchApi, 'restoreAllProviders').mockResolvedValue({ ok: true, restored_agents: 2 })
      vi.spyOn(providerSwitchApi, 'getFailoverStatus').mockResolvedValue({ failover_active: false, agents: [] })

      const store = useProviderSwitchStore()
      await store.restoreAll('demo')

      expect(restoreAllSpy).toHaveBeenCalledWith('demo')
    })

    it('applyTemplate invokes the API and refetches status', async () => {
      const applySpy = vi.spyOn(providerSwitchApi, 'applyProviderTemplate').mockResolvedValue({ ok: true, template: 'local-ai', updated_agents: 4 })
      vi.spyOn(providerSwitchApi, 'getFailoverStatus').mockResolvedValue({ failover_active: false, agents: [] })

      const store = useProviderSwitchStore()
      await store.applyTemplate('demo', 'local-ai')

      expect(applySpy).toHaveBeenCalledWith('demo', 'local-ai')
    })
  })
})
