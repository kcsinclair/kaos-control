// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { shallowMount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import type { AgentRunRow } from '@/types/api'
import * as agentsApi from '@/api/agents'
import * as configApi from '@/api/config'

// Frontend plan: lifecycle/frontend-plans/agent-logging-provider-driver-4-fe.md
// Milestone 4 — assert the Driver cell prefers run.driver over the current
// config driver, and that the provider sub-label reflects run.provider.

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { project: 'testproject' }, query: {} }),
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
}))

// Import component after all mocks are in place.
import AgentsRunsView from '../AgentsRunsView.vue'

function makeRun(overrides: Partial<AgentRunRow> = {}): AgentRunRow {
  return {
    run_id: 'run-1',
    agent_name: 'agent-a',
    role: 'backend-developer',
    target_path: 'lifecycle/requirements/foo-2.md',
    started_at: '2026-08-01T00:00:00Z',
    status: 'done',
    stderr_tail: '',
    artifacts_produced: [],
    ...overrides,
  }
}

async function mountRunsView() {
  const wrapper = shallowMount(AgentsRunsView, { global: { stubs: { RouterLink: true } } })
  await flushPromises()
  return wrapper
}

describe('AgentsRunsView — driver/provider from the recorded run', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
    vi.spyOn(configApi, 'getRoles').mockResolvedValue({ roles: [], users: [] })
    vi.spyOn(agentsApi, 'getReadyCounts').mockResolvedValue({ counts: {} })
  })

  it('shows the run.driver (not the agent current config driver) when they differ', async () => {
    vi.spyOn(agentsApi, 'listAgents').mockResolvedValue({
      agents: [{ name: 'agent-a', roles: ['backend-developer'], driver: 'gemini-cli' }],
    })
    vi.spyOn(agentsApi, 'listRuns').mockResolvedValue({
      runs: [makeRun({ driver: 'claude-mediated', provider: 'anthropic' })],
    })

    const wrapper = await mountRunsView()

    const badge = wrapper.find('.driver-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toBe('Claude Mediated')
    expect(badge.attributes('data-driver')).toBe('claude-mediated')
    // Current agent config driver ('gemini-cli') must not be shown.
    expect(wrapper.text()).not.toContain('Gemini CLI')
  })

  it('shows the run.provider as a sub-label', async () => {
    vi.spyOn(agentsApi, 'listAgents').mockResolvedValue({
      agents: [{ name: 'agent-a', roles: ['backend-developer'], driver: 'openai-compatible' }],
    })
    vi.spyOn(agentsApi, 'listRuns').mockResolvedValue({
      runs: [makeRun({ driver: 'openai-compatible', provider: 'my-openai-provider' })],
    })

    const wrapper = await mountRunsView()

    expect(wrapper.find('.provider-sublabel').text()).toBe('my-openai-provider')
  })

  it('falls back to the agent current config driver for legacy rows with no run.driver', async () => {
    vi.spyOn(agentsApi, 'listAgents').mockResolvedValue({
      agents: [{ name: 'agent-a', roles: ['backend-developer'], driver: 'gemini-cli' }],
    })
    vi.spyOn(agentsApi, 'listRuns').mockResolvedValue({
      runs: [makeRun({ driver: undefined, provider: undefined })],
    })

    const wrapper = await mountRunsView()

    const badge = wrapper.find('.driver-badge')
    expect(badge.text()).toBe('Gemini CLI')
    expect(badge.attributes('data-driver')).toBe('gemini-cli')
    expect(wrapper.find('.provider-sublabel').exists()).toBe(false)
  })
})
