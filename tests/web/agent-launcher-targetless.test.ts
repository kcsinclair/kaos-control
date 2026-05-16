// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 7 — Agent Launcher Targetless Tests
 *
 * Verifies the AgentLaunchModal correctly handles target-less agents
 * (agents with source_types: []) by:
 *   - Hiding the artifact picker
 *   - Showing an informational message
 *   - Enabling the Run button without a target selection
 *
 * Also verifies that agents WITH source_types still show the target picker.
 *
 * Component: web/src/components/agent/AgentLaunchModal.vue
 * Run with:  pnpm --prefix tests/web test agent-launcher-targetless
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import AgentLaunchModal from '../../web/src/components/agent/AgentLaunchModal.vue'
import type { AgentSummary } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Module mocks — prevent real network I/O
// ---------------------------------------------------------------------------

vi.mock('@/api/artifacts', () => ({
  listArtifacts: vi.fn().mockResolvedValue({ items: [], total: 0 }),
}))

vi.mock('@/stores/agents', () => ({
  useAgentsStore: vi.fn(() => ({
    startRun: vi.fn().mockResolvedValue('abc123'),
  })),
}))

vi.mock('@/stores/ui', () => ({
  useUiStore: vi.fn(() => ({
    error: vi.fn(),
    success: vi.fn(),
  })),
}))

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeAgent(overrides: Partial<AgentSummary> = {}): AgentSummary {
  return {
    name: 'test-runner',
    roles: ['qa'],
    active_status: 'in-qa',
    driver: 'claude-code-cli',
    source_types: [],
    ...overrides,
  } as AgentSummary
}

function mountModal(agent: AgentSummary) {
  setActivePinia(createPinia())
  return mount(AgentLaunchModal, {
    props: { agent, project: 'testproject' },
    global: { stubs: { teleport: true } },
  })
}

// ---------------------------------------------------------------------------
// Tests: target-less agent (source_types: [])
// ---------------------------------------------------------------------------

describe('target-less agent (source_types: [])', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('hides the artifact picker', async () => {
    const wrapper = mountModal(makeAgent({ source_types: [] }))
    await flushPromises()

    // The artifact list/picker elements should not be rendered.
    expect(wrapper.find('.artifact-list').exists()).toBe(false)
    expect(wrapper.find('[role="listbox"]').exists()).toBe(false)
  })

  it('shows an informational message instead of the picker', async () => {
    const wrapper = mountModal(makeAgent({ source_types: [] }))
    await flushPromises()

    const infoMsg = wrapper.find('.state-msg--info')
    expect(infoMsg.exists()).toBe(true)
    // Message should communicate that no target is required.
    expect(infoMsg.text()).toMatch(/no target|all test/i)
  })

  it('enables the Run button without any target selection', async () => {
    const wrapper = mountModal(makeAgent({ source_types: [] }))
    await flushPromises()

    const runBtn = wrapper.find('.btn-primary')
    expect(runBtn.exists()).toBe(true)
    expect(runBtn.attributes('disabled')).toBeUndefined()
  })

  it('does not call listArtifacts for a target-less agent', async () => {
    const { listArtifacts } = await import('@/api/artifacts')
    mountModal(makeAgent({ source_types: [] }))
    await flushPromises()

    expect(listArtifacts).not.toHaveBeenCalled()
  })
})

// ---------------------------------------------------------------------------
// Tests: target-requiring agent (source_types is non-empty or absent)
// ---------------------------------------------------------------------------

describe('target-requiring agent (source_types set)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows the artifact picker area when source_types is non-empty', async () => {
    const wrapper = mountModal(
      makeAgent({ name: 'backend-developer', source_types: ['plan-backend'], roles: ['backend-developer'] }),
    )
    await flushPromises()

    // The info message should NOT appear.
    expect(wrapper.find('.state-msg--info').exists()).toBe(false)
  })

  it('shows the artifact picker area when source_types is absent (undefined)', async () => {
    const agent = makeAgent({ name: 'planning-analyst', roles: ['analyst'] })
    delete (agent as Partial<AgentSummary>).source_types
    const wrapper = mountModal(agent)
    await flushPromises()

    expect(wrapper.find('.state-msg--info').exists()).toBe(false)
  })

  it('disables the Run button when no artifact is selected', async () => {
    const wrapper = mountModal(
      makeAgent({ name: 'backend-developer', source_types: ['plan-backend'], roles: ['backend-developer'] }),
    )
    await flushPromises()

    const runBtn = wrapper.find('.btn-primary')
    expect(runBtn.attributes('disabled')).toBeDefined()
  })
})
