// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ProviderFailoverModal from '../ProviderFailoverModal.vue'
import { useProviderSwitchStore } from '@/stores/providerSwitch'
import { useAuthStore } from '@/stores/auth'
import { useAgentsStore } from '@/stores/agents'
import * as providerSwitchApi from '@/api/providerSwitch'

// Frontend plan: lifecycle/frontend-plans/switch-provider-4-fe.md
// Milestone 3 — Provider Failover Modal & Preset Templates Drawer.

function setManagerRole(project: string) {
  const auth = useAuthStore()
  auth.me = {
    email: 'ops@example.com',
    display_name: 'Ops',
    roles: { [project]: ['devops'] },
  } as never
}

describe('ProviderFailoverModal', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('renders every active failover agent with its provider/model comparison', () => {
    setManagerRole('demo')
    const store = useProviderSwitchStore()
    store.status.agents = [
      {
        agent: 'requirements-analyst',
        is_failover: true,
        primary_provider: 'anthropic-cloud',
        primary_model: 'claude-3-7-sonnet',
        active_provider: 'gemini-cloud',
        active_model: 'gemini-2.5-flash',
        primary_healthy: true,
      },
    ]
    store.status.failover_active = true
    const agentsStore = useAgentsStore()
    agentsStore.agents = [
      { name: 'requirements-analyst', roles: ['analyst'], driver: 'claude-code-cli' },
    ] as never

    const wrapper = mount(ProviderFailoverModal, { props: { project: 'demo' } })

    expect(wrapper.text()).toContain('requirements-analyst')
    expect(wrapper.text()).toContain('analyst')
    expect(wrapper.text()).toContain('gemini-cloud (gemini-2.5-flash)')
    expect(wrapper.text()).toContain('anthropic-cloud (claude-3-7-sonnet)')
    expect(wrapper.text()).toContain('Recovered & Reachable')
  })

  it('shows the empty state when no agent is in failover', () => {
    setManagerRole('demo')
    const wrapper = mount(ProviderFailoverModal, { props: { project: 'demo' } })
    expect(wrapper.text()).toContain('All agents are operating on their primary providers.')
  })

  it('"Restore All" calls restoreAll and disables while in flight', async () => {
    setManagerRole('demo')
    const store = useProviderSwitchStore()
    store.status.agents = [
      { agent: 'a1', is_failover: true, active_provider: 'p2', active_model: 'm2', primary_provider: 'p1', primary_model: 'm1' },
    ]
    store.status.failover_active = true

    let resolveRestore: () => void = () => {}
    vi.spyOn(providerSwitchApi, 'restoreAllProviders').mockReturnValue(
      new Promise((resolve) => { resolveRestore = () => resolve({ ok: true, restored_agents: 1 }) }),
    )
    vi.spyOn(providerSwitchApi, 'getFailoverStatus').mockResolvedValue({ failover_active: false, agents: [] })

    const wrapper = mount(ProviderFailoverModal, { props: { project: 'demo' } })
    const restoreAllBtn = wrapper.findAll('button').find((b) => b.text().includes('Restore All Primary Providers'))!
    await restoreAllBtn.trigger('click')

    expect(restoreAllBtn.attributes('disabled')).toBeDefined()

    resolveRestore()
    await flushPromises()
  })

  it('individual "Restore Primary" button calls restoreAgent for that agent', async () => {
    setManagerRole('demo')
    const store = useProviderSwitchStore()
    store.status.agents = [
      { agent: 'a1', is_failover: true, active_provider: 'p2', active_model: 'm2', primary_provider: 'p1', primary_model: 'm1' },
    ]
    store.status.failover_active = true

    const restoreSpy = vi.spyOn(providerSwitchApi, 'restoreAgentProvider').mockResolvedValue({ ok: true, agent: 'a1', provider: 'p1', model: 'm1' })
    vi.spyOn(providerSwitchApi, 'getFailoverStatus').mockResolvedValue({ failover_active: false, agents: [] })

    const wrapper = mount(ProviderFailoverModal, { props: { project: 'demo' } })
    const restoreBtn = wrapper.findAll('button').find((b) => b.text() === 'Restore Primary')!
    await restoreBtn.trigger('click')
    await flushPromises()

    expect(restoreSpy).toHaveBeenCalledWith('demo', 'a1')
  })

  it('template dropdown renders templates from the store and applies the selection', async () => {
    setManagerRole('demo')
    const store = useProviderSwitchStore()
    store.templates = [
      { name: 'local-ai', description: 'All-local fallback', agents: { 'requirements-analyst': { provider: 'llama-cpp', model: 'qwen3-coder' } } },
    ]

    const applySpy = vi.spyOn(providerSwitchApi, 'applyProviderTemplate').mockResolvedValue({ ok: true, template: 'local-ai', updated_agents: 1 })
    vi.spyOn(providerSwitchApi, 'getFailoverStatus').mockResolvedValue({ failover_active: false, agents: [] })
    vi.spyOn(window, 'confirm').mockReturnValue(true)

    const wrapper = mount(ProviderFailoverModal, { props: { project: 'demo' } })
    const menuToggle = wrapper.findAll('button').find((b) => b.text() === 'Apply Preset Template')!
    await menuToggle.trigger('click')

    const templateItem = wrapper.findAll('button').find((b) => b.text().includes('local-ai'))!
    await templateItem.trigger('click')
    await flushPromises()

    expect(applySpy).toHaveBeenCalledWith('demo', 'local-ai')
  })
})
