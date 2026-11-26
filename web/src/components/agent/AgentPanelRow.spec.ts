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

  // Frontend plan: lifecycle/frontend-plans/switch-provider-4-fe.md — Milestone 4

  it('renders fallback badge and Restore Primary button when is_failover is true', () => {
    const wrapper = mount(AgentPanelRow, {
      props: {
        agents: [
          {
            name: 'requirements-analyst',
            roles: ['analyst'],
            driver: 'openai-compatible',
            provider: 'gemini-cloud',
            model: 'gemini-2.5-flash',
            is_failover: true,
            primary_provider: 'anthropic-cloud',
            primary_model: 'claude-3-7-sonnet',
          },
        ],
      },
    })

    const badge = wrapper.find('.panel-fallback-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toContain('gemini-2.5-flash')
    expect(badge.text()).toContain('claude-3-7-sonnet')
    expect(wrapper.find('.panel-restore-btn').exists()).toBe(true)
  })

  it('clicking Restore Primary dispatches a restore-provider event', async () => {
    const agent = {
      name: 'requirements-analyst',
      roles: ['analyst'],
      driver: 'openai-compatible',
      provider: 'gemini-cloud',
      model: 'gemini-2.5-flash',
      is_failover: true,
      primary_provider: 'anthropic-cloud',
      primary_model: 'claude-3-7-sonnet',
    }
    const wrapper = mount(AgentPanelRow, { props: { agents: [agent] } })

    await wrapper.find('.panel-restore-btn').trigger('click')

    expect(wrapper.emitted('restore-provider')).toBeTruthy()
    expect(wrapper.emitted('restore-provider')![0][0]).toEqual(agent)
  })

  it('the Switch Provider action emits switch-provider for an agent in failover', async () => {
    const agent = {
      name: 'requirements-analyst',
      roles: ['analyst'],
      driver: 'openai-compatible',
      provider: 'gemini-cloud',
      model: 'gemini-2.5-flash',
      is_failover: true,
      primary_provider: 'anthropic-cloud',
      primary_model: 'claude-3-7-sonnet',
    }
    const wrapper = mount(AgentPanelRow, { props: { agents: [agent] } })

    await wrapper.find('.panel-switch-btn').trigger('click')

    expect(wrapper.emitted('switch-provider')).toBeTruthy()
    expect(wrapper.emitted('switch-provider')![0][0]).toEqual(agent)
  })

  it('does not render the fallback badge for an agent on its primary provider', () => {
    const wrapper = mount(AgentPanelRow, {
      props: {
        agents: [
          { name: 'backend-developer', roles: ['backend-developer'], driver: 'claude-code-cli', model: 'sonnet' },
        ],
      },
    })

    expect(wrapper.find('.panel-fallback-badge').exists()).toBe(false)
    expect(wrapper.find('.panel-switch-btn').exists()).toBe(false)
  })
})
