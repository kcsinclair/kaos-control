// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ProviderSettingsView from './ProviderSettingsView.vue'
import { useProvidersStore } from '@/stores/providers'
import { ApiError } from '@/api/client'

describe('ProviderSettingsView', () => {
  let wrapper: ReturnType<typeof mount> | null = null

  beforeEach(() => {
    document.body.innerHTML = ''
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  afterEach(() => {
    if (wrapper) {
      wrapper.unmount()
      wrapper = null
    }
    document.body.innerHTML = ''
  })

  it('renders empty state when no providers exist', async () => {
    const store = useProvidersStore()
    vi.spyOn(store, 'fetchProviders').mockResolvedValue(undefined)
    vi.spyOn(store, 'probeAll').mockResolvedValue(undefined)
    store.providers = []

    wrapper = mount(ProviderSettingsView)
    await flushPromises()

    expect(wrapper.find('.psv-title').text()).toBe('Providers')
    expect(wrapper.find('.psv-state').text()).toContain('No providers configured. Click Add Provider to register one.')
  })

  it('renders provider table and health dots correctly', async () => {
    const store = useProvidersStore()
    vi.spyOn(store, 'fetchProviders').mockResolvedValue(undefined)
    vi.spyOn(store, 'probeAll').mockResolvedValue(undefined)
    store.providers = [
      { name: 'llama-local', base_url: 'http://localhost:8080', driver: 'openai-compatible' },
    ]
    store.health.set('llama-local', { ok: true, latency_ms: 25 })
    store.probeResults.set('llama-local', {
      ok: true,
      latency_ms: 25,
      models: [{ id: 'qwen-30b', name: 'Qwen 30B' }],
    })

    wrapper = mount(ProviderSettingsView)
    await flushPromises()

    expect(wrapper.find('.cell-name').text()).toBe('llama-local')
    expect(wrapper.find('.cell-url').text()).toBe('http://localhost:8080')
    expect(wrapper.find('.driver-badge').text()).toBe('openai-compatible')
    expect(wrapper.find('.health-dot--ok').exists()).toBe(true)
    expect(wrapper.find('.cell-latency').text()).toBe('25 ms')
    expect(wrapper.find('.btn-models').text()).toBe('1 models')
  })

  it('clicking Add Provider opens ProviderForm modal', async () => {
    const store = useProvidersStore()
    vi.spyOn(store, 'fetchProviders').mockResolvedValue(undefined)
    vi.spyOn(store, 'probeAll').mockResolvedValue(undefined)

    wrapper = mount(ProviderSettingsView)
    await flushPromises()

    const addBtn = wrapper.find('.btn-primary')
    await addBtn.trigger('click')

    const modal = document.querySelector('.modal-panel')
    expect(modal).toBeTruthy()
    expect(modal?.textContent).toContain('Add Provider')
  })

  it('delete confirmation handles 409 Conflict error with visible explanation', async () => {
    const store = useProvidersStore()
    vi.spyOn(store, 'fetchProviders').mockResolvedValue(undefined)
    vi.spyOn(store, 'probeAll').mockResolvedValue(undefined)
    store.providers = [
      { name: 'in-use-provider', base_url: 'http://localhost:8080', driver: 'openai-compatible' },
    ]

    vi.spyOn(store, 'deleteProvider').mockRejectedValue(
      new ApiError('conflict', 'cannot delete provider "in-use-provider": in use by project "kaos-control"', 409)
    )

    wrapper = mount(ProviderSettingsView)
    await flushPromises()

    // Click delete button
    const deleteBtn = wrapper.find('.btn-icon--danger')
    await deleteBtn.trigger('click')

    // Confirmation modal should be open
    const modal = document.querySelector('.modal-panel')
    expect(modal?.textContent).toContain('Delete provider in-use-provider? This cannot be undone.')

    // Click confirm delete in modal
    const confirmBtn = document.querySelector('.btn-danger') as HTMLElement
    confirmBtn.click()
    await flushPromises()

    expect(document.querySelector('.confirm-error')?.textContent).toContain(
      'cannot delete provider "in-use-provider": in use by project "kaos-control"'
    )
  })
})
