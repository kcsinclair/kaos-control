// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import AgentConfigForm from './AgentConfigForm.vue'
import { useProvidersStore } from '@/stores/providers'

describe('AgentConfigForm', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('selects provider and populates model options from store', async () => {
    const store = useProvidersStore()
    store.providers = [
      { name: 'llama-cpp', base_url: 'http://localhost:8080', driver: 'openai-compatible' },
    ]
    store.models.set('llama-cpp', [
      { id: 'qwen3-coder:30b', name: 'Qwen 3 Coder 30B' },
      { id: 'llama3:8b', name: 'Llama 3 8B' },
    ])
    store.health.set('llama-cpp', { ok: true, latency_ms: 18 })

    const wrapper = mount(AgentConfigForm, {
      props: {
        availableRoles: ['analyst', 'backend-developer'],
        initial: {
          name: 'coder-agent',
          roles: ['backend-developer'],
          driver: 'openai-compatible',
          provider: 'llama-cpp',
          model: 'qwen3-coder:30b',
        },
      },
    })
    await flushPromises()

    const providerSelect = wrapper.find('#acf-provider')
    expect(providerSelect.exists()).toBe(true)
    expect((providerSelect.element as HTMLSelectElement).value).toBe('llama-cpp')

    const modelSelect = wrapper.find('#acf-provider-model')
    expect(modelSelect.exists()).toBe(true)
    const options = modelSelect.findAll('option')
    // 1 empty placeholder + 2 models
    expect(options.length).toBe(3)
    expect(options[1].text()).toContain('Qwen 3 Coder 30B')
  })

  it('validates provider is required when driver is openai-compatible', async () => {
    const store = useProvidersStore()
    store.providers = [
      { name: 'llama-cpp', base_url: 'http://localhost:8080', driver: 'openai-compatible' },
    ]

    const wrapper = mount(AgentConfigForm, {
      props: {
        availableRoles: ['analyst'],
      },
    })
    await flushPromises()

    // Fill name and role
    await wrapper.find('#acf-name').setValue('my-agent')
    const roleChip = wrapper.find('.acf-role-chip')
    await roleChip.trigger('click')

    // Select openai-compatible driver
    const radios = wrapper.findAll('input[type="radio"]')
    const openaiRadio = radios.find((r) => (r.element as HTMLInputElement).value === 'openai-compatible')
    await openaiRadio!.setValue(true)

    // Submit without selecting provider or model
    await wrapper.find('form').trigger('submit.prevent')

    expect(wrapper.emitted('submit')).toBeFalsy()
    expect(wrapper.text()).toContain('Select a provider.')
    expect(wrapper.text()).toContain('Model is required for OpenAI-compatible driver.')
  })

  it('emits correct AgentFormData on submit with provider and model', async () => {
    const store = useProvidersStore()
    store.providers = [
      { name: 'openrouter', base_url: 'https://openrouter.ai/api/v1', driver: 'openai-compatible' },
    ]
    store.models.set('openrouter', [
      { id: 'anthropic/claude-3.5-sonnet', name: 'Claude 3.5 Sonnet' },
    ])

    const wrapper = mount(AgentConfigForm, {
      props: {
        availableRoles: ['analyst'],
      },
    })
    await flushPromises()

    await wrapper.find('#acf-name').setValue('openrouter-analyst')
    const roleChip = wrapper.find('.acf-role-chip')
    await roleChip.trigger('click')

    const radios = wrapper.findAll('input[type="radio"]')
    const openaiRadio = radios.find((r) => (r.element as HTMLInputElement).value === 'openai-compatible')
    await openaiRadio!.setValue(true)

    await wrapper.find('#acf-provider').setValue('openrouter')
    await flushPromises()

    await wrapper.find('#acf-provider-model').setValue('anthropic/claude-3.5-sonnet')
    await wrapper.find('#acf-max-iterations').setValue('30')

    await wrapper.find('form').trigger('submit.prevent')

    expect(wrapper.emitted('submit')).toBeTruthy()
    const emitted = wrapper.emitted('submit')![0][0] as any
    expect(emitted.name).toBe('openrouter-analyst')
    expect(emitted.driver).toBe('openai-compatible')
    expect(emitted.provider).toBe('openrouter')
    expect(emitted.model).toBe('anthropic/claude-3.5-sonnet')
    expect(emitted.max_tool_iterations).toBe(30)
  })
})
