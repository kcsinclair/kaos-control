// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import AgentPanelRow from './AgentPanelRow.vue'

describe('AgentPanelRow', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders provider badge when driver is openai-compatible and provider is set', () => {
    const wrapper = mount(AgentPanelRow, {
      props: {
        agents: [
          {
            name: 'custom-coder',
            roles: ['backend-developer'],
            driver: 'openai-compatible',
            provider: 'openrouter',
            model: 'anthropic/claude-3.5-sonnet',
          },
        ],
      },
    })

    expect(wrapper.find('.panel-provider-badge').exists()).toBe(true)
    expect(wrapper.find('.panel-provider-badge').text()).toBe('openrouter')
    expect(wrapper.find('.panel-model').text()).toBe('anthropic/claude-3.5-sonnet')
  })
})
