// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 3 — AgentsRunsView renders RunFailureBanner for precheck failures
 *
 * Tests:
 *   - When the expanded run has state=failed and failure_reason set, the
 *     RunFailureBanner is rendered.
 *   - When the expanded run has state=failed but failure_reason is null/absent,
 *     the banner is NOT rendered.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import AgentsRunsView from '../../web/src/views/project/AgentsRunsView.vue'
import { useAgentsStore } from '../../web/src/stores/agents'
import type { AgentRunRow } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

vi.mock('@/api/agents', () => ({
  listRuns:       vi.fn().mockResolvedValue({ runs: [] }),
  listAgents:     vi.fn().mockResolvedValue({ agents: [] }),
  startRun:       vi.fn().mockResolvedValue({ run_id: 'new-run' }),
  killRun:        vi.fn().mockResolvedValue({}),
  getRunLog:      vi.fn().mockResolvedValue(''),
  getReadyCounts: vi.fn().mockResolvedValue({ counts: {} }),
}))

vi.mock('@/api/config', () => ({
  getRoles:        vi.fn().mockResolvedValue({ roles: [] }),
  getConfig:       vi.fn().mockResolvedValue({ raw: '' }),
  parseConfigYaml: vi.fn().mockReturnValue({}),
  dumpConfigYaml:  vi.fn().mockReturnValue(''),
  updateConfig:    vi.fn().mockResolvedValue({}),
}))

vi.mock('vue-router', async (importActual) => {
  const actual = await importActual<typeof import('vue-router')>()
  return {
    ...actual,
    useRoute:  vi.fn(() => ({ params: { project: 'testproject' }, query: {} })),
    useRouter: vi.fn(() => ({ push: vi.fn(), replace: vi.fn() })),
  }
})

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

function makeRun(overrides: Partial<AgentRunRow> = {}): AgentRunRow {
  return {
    run_id:             'aaaaaaaa-0000-0000-0000-000000000001',
    agent_name:         'backend-developer',
    role:               'developer',
    target_path:        'lifecycle/requirements/test.md',
    started_at:         '2026-01-01T10:00:00Z',
    finished_at:        '2026-01-01T10:01:00Z',
    status:             'failed',
    stderr_tail:        '',
    artifacts_produced: [],
    ...overrides,
  }
}

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

function mountView() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes:  [{ path: '/', component: { template: '<div/>' } }],
  })
  return mount(AgentsRunsView, {
    global: { plugins: [router] },
  })
}

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  setActivePinia(createPinia())
})

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('AgentsRunsView — RunFailureBanner integration', () => {
  it('renders RunFailureBanner when expanded run is failed with failure_reason', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({
      runs: [
        makeRun({
          failure_reason: 'permission_mode_default',
          observed_permission_mode: 'default',
          remediation: ['Run `claude config set permission-mode bypassPermissions`'],
        }),
      ],
    })
    await flushPromises()

    // Click the row to expand it
    const row = wrapper.find('tr.run-row')
    await row.trigger('click')

    expect(wrapper.find('.failure-banner').exists()).toBe(true)
    expect(wrapper.text()).toContain('Claude Code is in default permission mode')
  })

  it('does NOT render RunFailureBanner when expanded run is failed without failure_reason', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({
      runs: [
        // A "classic" failure with no precheck fields
        makeRun({ failure_reason: null }),
      ],
    })
    await flushPromises()

    const row = wrapper.find('tr.run-row')
    await row.trigger('click')

    expect(wrapper.find('.failure-banner').exists()).toBe(false)
  })

  it('does NOT render RunFailureBanner for a non-failed run even if failure_reason is set', async () => {
    const wrapper = mountView()
    await flushPromises()

    const store = useAgentsStore()
    store.$patch({
      runs: [
        makeRun({
          status: 'done',
          failure_reason: 'permission_mode_default',
        }),
      ],
    })
    await flushPromises()

    const row = wrapper.find('tr.run-row')
    await row.trigger('click')

    expect(wrapper.find('.failure-banner').exists()).toBe(false)
  })
})
