// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 5 — AgentRunningBanner warmup visual indicator tests
 * (local-model-operability: lifecycle/test-plans/local-model-operability-5-test.md)
 *
 * Covers:
 *   - The animated "Warming up model weights..." badge renders when the
 *     active job's matching run row has warmup_state 'model_loading' or
 *     'warming_up' (queueStore.isWarmingUp derives from agentsStore.runs —
 *     see web/src/stores/queue.ts).
 *   - The badge disappears once the run transitions to 'generating'.
 *
 * Approach: mount with real Pinia queue + agents stores (only the network
 * boundary is mocked, matching the pattern in ProjectQueuePanel.realtime.test.ts)
 * and drive state directly via store mutation — no WS wiring needed since
 * AgentRunningBanner only reads store state, it never calls fetch()/subscribe().
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import AgentRunningBanner from '../../web/src/components/dashboard/AgentRunningBanner.vue'
import { useQueueStore } from '../../web/src/stores/queue'
import { useAgentsStore } from '../../web/src/stores/agents'
import type { AgentRunRow } from '../../web/src/types/api'

vi.mock('@/api/queue', () => ({
  listQueue: vi.fn(),
  enqueue: vi.fn(),
  cancelQueue: vi.fn(),
  pauseQueue: vi.fn(),
  resumeQueue: vi.fn(),
}))

vi.mock('@/api/agents', () => ({
  listRuns: vi.fn(),
  listAgents: vi.fn(),
  startRun: vi.fn(),
  killRun: vi.fn(),
  getRunLog: vi.fn(),
  getReadyCounts: vi.fn(),
  listRunsByTargetPath: vi.fn(),
}))

vi.mock('@/api/ws', () => ({
  getAppWs: vi.fn(() => ({ on: vi.fn(() => () => {}) })),
  getProjectWs: vi.fn(() => ({ on: vi.fn(() => () => {}), onType: vi.fn(() => () => {}) })),
}))

function makeRunRow(overrides: Partial<AgentRunRow> = {}): AgentRunRow {
  return {
    run_id: 'run-1',
    agent_name: 'local-backend-developer',
    role: 'backend-developer',
    target_path: 'lifecycle/backend-plans/feature-3-be.md',
    started_at: '2026-08-25T10:00:00Z',
    status: 'running',
    stderr_tail: '',
    artifacts_produced: [],
    ...overrides,
  }
}

async function mountBanner() {
  const pinia = createPinia()
  setActivePinia(pinia)

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div/>' } }],
  })
  await router.push('/p/my-project')
  await router.isReady()

  const queueStore = useQueueStore()
  const agentsStore = useAgentsStore()

  const wrapper = mount(AgentRunningBanner, {
    props: { project: 'my-project' },
    global: { plugins: [pinia, router] },
  })

  return { wrapper, queueStore, agentsStore }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('AgentRunningBanner — warmup indicator', () => {
  it('renders the warmup badge when the running job is in model_loading', async () => {
    const { wrapper, queueStore, agentsStore } = await mountBanner()

    queueStore.snapshot.running = {
      id: 'job-1',
      project: 'my-project',
      artifact_path: 'lifecycle/backend-plans/feature-3-be.md',
      agent_name: 'local-backend-developer',
      state: 'running',
      attempts: 0,
      enqueued_at: '2026-08-25T09:59:00Z',
      position: 0,
      enqueued_by: 'admin@test.local',
    }
    agentsStore.runs = [
      makeRunRow({ warmup_state: 'model_loading', warmup_message: 'Loading model weights…' }),
    ]
    await wrapper.vm.$nextTick()

    const badge = wrapper.find('.warmup-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toContain('Loading model weights…')
  })

  it('renders the default warmup label when no warmup_message is set', async () => {
    const { wrapper, queueStore, agentsStore } = await mountBanner()

    queueStore.snapshot.running = {
      id: 'job-1',
      project: 'my-project',
      artifact_path: 'lifecycle/backend-plans/feature-3-be.md',
      agent_name: 'local-backend-developer',
      state: 'running',
      attempts: 0,
      enqueued_at: '2026-08-25T09:59:00Z',
      position: 0,
      enqueued_by: 'admin@test.local',
    }
    agentsStore.runs = [makeRunRow({ warmup_state: 'warming_up', warmup_message: null })]
    await wrapper.vm.$nextTick()

    const badge = wrapper.find('.warmup-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toContain('Warming up model weights...')
  })

  it('hides the warmup badge once the run transitions to generating', async () => {
    const { wrapper, queueStore, agentsStore } = await mountBanner()

    queueStore.snapshot.running = {
      id: 'job-1',
      project: 'my-project',
      artifact_path: 'lifecycle/backend-plans/feature-3-be.md',
      agent_name: 'local-backend-developer',
      state: 'running',
      attempts: 0,
      enqueued_at: '2026-08-25T09:59:00Z',
      position: 0,
      enqueued_by: 'admin@test.local',
    }
    agentsStore.runs = [makeRunRow({ warmup_state: 'model_loading', warmup_message: 'Loading…' })]
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.warmup-badge').exists()).toBe(true)

    // First content token arrives: driver transitions stage to 'generating'.
    agentsStore.runs = [makeRunRow({ warmup_state: 'generating', warmup_message: null })]
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.warmup-badge').exists()).toBe(false)
  })

  it('does not render the warmup badge for a non-warming run', async () => {
    const { wrapper, queueStore, agentsStore } = await mountBanner()

    queueStore.snapshot.running = {
      id: 'job-1',
      project: 'my-project',
      artifact_path: 'lifecycle/backend-plans/feature-3-be.md',
      agent_name: 'local-backend-developer',
      state: 'running',
      attempts: 0,
      enqueued_at: '2026-08-25T09:59:00Z',
      position: 0,
      enqueued_by: 'admin@test.local',
    }
    agentsStore.runs = [makeRunRow({ warmup_state: null, warmup_message: null })]
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.warmup-badge').exists()).toBe(false)
    expect(wrapper.find('.agent-running-banner').exists()).toBe(true)
  })
})
