// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ProviderForm from './ProviderForm.vue'
import * as providersApi from '@/api/providers'

vi.mock('@/api/providers', () => ({
  testConnection: vi.fn(),
}))

describe('ProviderForm', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('populates fields when a quick preset button is clicked', async () => {
    const wrapper = mount(ProviderForm)

    const presetButtons = wrapper.findAll('.pvf-preset-chip')
    const openRouterBtn = presetButtons.find((btn) => btn.text().includes('OpenRouter'))
    expect(openRouterBtn).toBeTruthy()

    await openRouterBtn!.trigger('click')

    const nameInput = wrapper.find('#pvf-name').element as HTMLInputElement
    const urlInput = wrapper.find('#pvf-url').element as HTMLInputElement
    expect(nameInput.value).toBe('openrouter')
    expect(urlInput.value).toBe('https://openrouter.ai/api/v1')

    // Extra headers should be populated from preset
    const headerKeys = wrapper.findAll('.pvf-header-key')
    expect(headerKeys.length).toBe(2)
  })

  it('adding and removing extra header rows updates form payload', async () => {
    const wrapper = mount(ProviderForm)

    await wrapper.find('#pvf-name').setValue('custom-provider')
    await wrapper.find('#pvf-url').setValue('http://localhost:8000')

    // Add header
    const addHeaderBtn = wrapper.find('.btn-add-header')
    await addHeaderBtn.trigger('click')

    const keyInputs = wrapper.findAll('.pvf-header-key')
    const valInputs = wrapper.findAll('.pvf-header-val')
    await keyInputs[0].setValue('X-Custom-Header')
    await valInputs[0].setValue('HeaderValue')

    // Add second header then remove it
    await addHeaderBtn.trigger('click')
    expect(wrapper.findAll('.pvf-header-row').length).toBe(2)
    const removeButtons = wrapper.findAll('.btn-remove-header')
    await removeButtons[1].trigger('click')
    expect(wrapper.findAll('.pvf-header-row').length).toBe(1)

    // Submit form
    await wrapper.find('form').trigger('submit.prevent')

    expect(wrapper.emitted('submit')).toBeTruthy()
    const payload = wrapper.emitted('submit')![0][0] as any
    expect(payload.extra_headers).toEqual({ 'X-Custom-Header': 'HeaderValue' })
  })

  it('test connection calls testConnection API and displays result', async () => {
    vi.mocked(providersApi.testConnection).mockResolvedValue({
      ok: true,
      latency_ms: 45,
      message: 'Successfully connected',
    })

    const wrapper = mount(ProviderForm)
    await wrapper.find('#pvf-url').setValue('http://localhost:8080')

    const testBtn = wrapper.find('.btn-test')
    await testBtn.trigger('click')
    await flushPromises()

    expect(providersApi.testConnection).toHaveBeenCalled()
    expect(wrapper.find('.test-status--ok').text()).toContain('✓ Connected (45 ms)')
  })

  it('displays error in test status when connection fails', async () => {
    vi.mocked(providersApi.testConnection).mockResolvedValue({
      ok: false,
      error: 'Connection refused',
    })

    const wrapper = mount(ProviderForm)
    await wrapper.find('#pvf-url').setValue('http://localhost:8080')

    const testBtn = wrapper.find('.btn-test')
    await testBtn.trigger('click')
    await flushPromises()

    expect(wrapper.find('.test-status--err').text()).toContain('✗ Connection refused')
  })

  it('shows bullet placeholder when editing a provider with configured key', async () => {
    const wrapper = mount(ProviderForm, {
      props: {
        initial: {
          name: 'existing-prov',
          base_url: 'https://api.openai.com/v1',
          driver: 'openai-compatible',
          has_api_key: true,
        },
      },
    })

    const keyInput = wrapper.find('#pvf-key').element as HTMLInputElement
    expect(keyInput.placeholder).toBe('••••••••')
    expect(keyInput.value).toBe('')
  })
})
