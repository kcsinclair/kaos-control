// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import QueuePauseBanner from '../QueuePauseBanner.vue'
import { useQueueStore } from '@/stores/queue'
import { useAuthStore } from '@/stores/auth'
import { useProvidersStore } from '@/stores/providers'
import * as providerSwitchApi from '@/api/providerSwitch'
import * as queueApi from '@/api/queue'

// Frontend plan: lifecycle/frontend-plans/switch-provider-4-fe.md
// Milestone 5 — Queue Pause Banner Failover & Resume Integration.

function setManagerRole() {
  const auth = useAuthStore()
  auth.me = {
    email: 'ops@example.com',
    display_name: 'Ops',
    roles: { demo: ['devops'] },
  } as never
}

function pausedForRateLimit() {
  const queue = useQueueStore()
  queue.snapshot.paused = true
  queue.snapshot.pause_reason = 'rate_limit'
  queue.snapshot.pending = [
    {
      id: 'job-2',
      project: 'demo',
      artifact_path: 'lifecycle/requirements/foo.md',
      agent_name: 'requirements-analyst',
      state: 'pending',
      attempts: 1,
      enqueued_at: '2026-08-25T08:00:00Z',
      position: -1,
      enqueued_by: 'system',
    },
  ]
  return queue
}

describe('QueuePauseBanner — switch provider & resume', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('shows the "Switch Provider & Resume" button on a rate-limit pause', () => {
    setManagerRole()
    pausedForRateLimit()

    const wrapper = mount(QueuePauseBanner)
    const btn = wrapper.findAll('button').find((b) => b.text().includes('Switch Provider & Resume'))
    expect(btn).toBeTruthy()
  })

  it('does not show the button when paused manually', () => {
    setManagerRole()
    const queue = useQueueStore()
    queue.snapshot.paused = true
    queue.snapshot.pause_reason = 'manual'

    const wrapper = mount(QueuePauseBanner)
    const btn = wrapper.findAll('button').find((b) => b.text().includes('Switch Provider & Resume'))
    expect(btn).toBeFalsy()
  })

  it('clicking the button opens the switch modal pre-populated with the failed job agent', async () => {
    setManagerRole()
    pausedForRateLimit()
    vi.spyOn(providerSwitchApi, 'getFailoverStatus').mockResolvedValue({ failover_active: false, agents: [] })

    const wrapper = mount(QueuePauseBanner)
    const btn = wrapper.findAll('button').find((b) => b.text().includes('Switch Provider & Resume'))!
    await btn.trigger('click')

    expect(wrapper.text()).toContain('Switch Provider — requirements-analyst')
  })

  it('successfully switching triggers queue resumption', async () => {
    setManagerRole()
    pausedForRateLimit()
    const providersStore = useProvidersStore()
    providersStore.providers = [
      { name: 'gemini-cloud', base_url: 'https://generativelanguage.googleapis.com', driver: 'openai-compatible' },
    ]
    vi.spyOn(providerSwitchApi, 'switchAgentProvider').mockResolvedValue({
      ok: true, agent: 'requirements-analyst', provider: 'gemini-cloud', model: 'gemini-2.5-flash',
    })
    vi.spyOn(providerSwitchApi, 'getFailoverStatus').mockResolvedValue({ failover_active: true, agents: [] })
    const resumeSpy = vi.spyOn(queueApi, 'resumeQueue').mockResolvedValue(undefined)

    const wrapper = mount(QueuePauseBanner)
    const btn = wrapper.findAll('button').find((b) => b.text().includes('Switch Provider & Resume'))!
    await btn.trigger('click')

    await wrapper.find('#spm-provider').setValue('gemini-cloud')
    await wrapper.find('#spm-model').setValue('gemini-2.5-flash')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(resumeSpy).toHaveBeenCalledTimes(1)
  })
})
