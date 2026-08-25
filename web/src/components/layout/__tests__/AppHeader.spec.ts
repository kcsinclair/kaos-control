// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'

// Frontend plan: lifecycle/frontend-plans/switch-provider-4-fe.md
// Milestone 2 — App Header Failover Alert Badge & Navigation Integration.

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { project: 'demo' } }),
  useRouter: () => ({ push: vi.fn() }),
}))

import AppHeader from '../AppHeader.vue'
import { useProviderSwitchStore } from '@/stores/providerSwitch'

const routerLinkStub = { template: '<a><slot /></a>' }

function mountHeader() {
  return mount(AppHeader, {
    global: {
      stubs: { RouterLink: routerLinkStub },
    },
  })
}

describe('AppHeader — failover badge', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    // jsdom doesn't implement matchMedia; the theme store's isDark computed
    // reads it on every render.
    vi.stubGlobal('matchMedia', vi.fn().mockReturnValue({ matches: false, addEventListener: vi.fn(), removeEventListener: vi.fn() }))
  })

  it('is hidden when no agent is in failover', () => {
    const wrapper = mountHeader()
    expect(wrapper.find('.header-failover-badge').exists()).toBe(false)
  })

  it('appears with amber styling and agent count when failover is active', () => {
    const store = useProviderSwitchStore()
    store.status.agents = [
      { agent: 'a1', is_failover: true, active_provider: 'p2', active_model: 'm2' },
    ]
    store.status.failover_active = true

    const wrapper = mountHeader()
    const badge = wrapper.find('.header-failover-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toContain('Failover Active (1)')
  })

  it('shows the green recovery dot when a primary provider has recovered', () => {
    const store = useProviderSwitchStore()
    store.status.agents = [
      { agent: 'a1', is_failover: true, active_provider: 'p2', active_model: 'm2', primary_provider: 'p1' },
    ]
    store.status.failover_active = true
    store.recoveredProviders = ['p1']

    const wrapper = mountHeader()
    expect(wrapper.find('.failover-recovered-dot').exists()).toBe(true)
  })

  it('clicking the badge opens the provider failover modal', async () => {
    const store = useProviderSwitchStore()
    store.status.agents = [
      { agent: 'a1', is_failover: true, active_provider: 'p2', active_model: 'm2' },
    ]
    store.status.failover_active = true

    const wrapper = mountHeader()
    expect(store.modalOpen).toBe(false)
    await wrapper.find('.header-failover-badge').trigger('click')
    expect(store.modalOpen).toBe(true)
  })
})
