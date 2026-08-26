// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import type { AgentRunRow } from '@/types/api'
import { useAgentsStore } from '@/stores/agents'
import * as agentsApi from '@/api/agents'
import RunDetailModal from '../RunDetailModal.vue'

// Frontend plan: lifecycle/frontend-plans/agent-logging-provider-driver-4-fe.md
// Milestone 4 — given an AgentRunRow with driver/provider set, assert both
// are rendered; given empty provider, assert '—'; given empty driver, assert
// the config fallback is used.
//
// RunDetailModal renders via <Teleport to="body">, which portals outside the
// wrapper's own element tree — query the real document body instead of the
// wrapper (see DirectiveMigrationModal.spec.ts for precedent).
enableAutoUnmount(afterEach)

function makeRun(overrides: Partial<AgentRunRow> = {}): AgentRunRow {
  return {
    run_id: 'run-1',
    agent_name: 'agent-a',
    role: 'backend-developer',
    target_path: 'lifecycle/requirements/foo-2.md',
    started_at: '2026-08-01T00:00:00Z',
    finished_at: '2026-08-01T00:01:00Z',
    status: 'done',
    stderr_tail: '',
    artifacts_produced: [],
    ...overrides,
  }
}

async function mountModal() {
  mount(RunDetailModal, { props: { project: 'testproject', runId: 'run-1' } })
  await flushPromises()
  return new DOMWrapper(document.body)
}

describe('RunDetailModal — Driver + Provider fields', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
    vi.spyOn(agentsApi, 'getRunResult').mockResolvedValue({ result: null })
    vi.spyOn(agentsApi, 'getRunLog').mockResolvedValue('')
  })

  it('renders the recorded run.driver and run.provider', async () => {
    vi.spyOn(agentsApi, 'getRun').mockResolvedValue({
      run: makeRun({ driver: 'openai-compatible', provider: 'my-openai-provider' }),
    })
    const agentsStore = useAgentsStore()
    agentsStore.agents = [{ name: 'agent-a', roles: ['backend-developer'], driver: 'gemini-cli' }] as never

    const wrapper = await mountModal()

    const fields = wrapper.findAll('.rdm-field')
    const driverField = fields.find((f) => f.find('.rdm-field-label').text() === 'Driver')!
    const providerField = fields.find((f) => f.find('.rdm-field-label').text() === 'Provider')!
    expect(driverField.find('.rdm-field-value').text()).toBe('OpenAI Compatible')
    expect(providerField.find('.rdm-field-value').text()).toBe('my-openai-provider')
  })

  it('renders a dash for an empty provider without throwing', async () => {
    vi.spyOn(agentsApi, 'getRun').mockResolvedValue({
      run: makeRun({ driver: 'gemini-cli', provider: undefined }),
    })
    const agentsStore = useAgentsStore()
    agentsStore.agents = [{ name: 'agent-a', roles: ['backend-developer'], driver: 'gemini-cli' }] as never

    const wrapper = await mountModal()

    const fields = wrapper.findAll('.rdm-field')
    const providerField = fields.find((f) => f.find('.rdm-field-label').text() === 'Provider')!
    expect(providerField.find('.rdm-field-value').text()).toBe('—')
  })

  it('falls back to the agent current config driver when run.driver is empty (legacy row)', async () => {
    vi.spyOn(agentsApi, 'getRun').mockResolvedValue({
      run: makeRun({ driver: undefined, provider: undefined }),
    })
    const agentsStore = useAgentsStore()
    agentsStore.agents = [{ name: 'agent-a', roles: ['backend-developer'], driver: 'claude-mediated' }] as never

    const wrapper = await mountModal()

    const fields = wrapper.findAll('.rdm-field')
    const driverField = fields.find((f) => f.find('.rdm-field-label').text() === 'Driver')!
    expect(driverField.find('.rdm-field-value').text()).toBe('Claude Mediated')
  })
})
