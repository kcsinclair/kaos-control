// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { useProvidersStore } from '@/stores/providers'
import { useProviderSwitchStore } from '@/stores/providerSwitch'
import { useQueueStore } from '@/stores/queue'
import { ApiError } from '@/api/client'
import SwitchProviderModal from '../SwitchProviderModal.vue'

// agent-switchover-and-failover frontend plan, Milestone 5 — FR-8.2: a manual
// switch must be blocked (proactively and on the backend's 409) while a run
// is executing, naming the running job(s) rather than a generic error.
enableAutoUnmount(afterEach)

function mountModal() {
  return mount(SwitchProviderModal, { props: { project: 'testproject', agentName: 'agent-a' } })
}

describe('SwitchProviderModal — running-jobs guard (FR-8.2)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
    const providersStore = useProvidersStore()
    vi.spyOn(providersStore, 'fetchProviders').mockResolvedValue(undefined)
    providersStore.providers = [{ name: 'openai-a', base_url: 'http://x', driver: 'openai-compatible' }]
  })

  it('disables submit and names the running job when one is already known to be running', async () => {
    const queueStore = useQueueStore()
    queueStore.snapshot.running = {
      id: 'job-1',
      project: 'testproject',
      artifact_path: 'lifecycle/requirements/foo-2.md',
      agent_name: 'agent-a',
      state: 'running',
      attempts: 1,
      enqueued_at: '2026-09-04T00:00:00Z',
      position: 0,
      enqueued_by: 'tester',
    }

    const wrapper = mountModal()
    await flushPromises()

    expect(wrapper.find('.spm-running-warning').exists()).toBe(true)
    expect(wrapper.find('.spm-running-jobs-list').text()).toContain('agent-a')
    expect(wrapper.find('.spm-running-jobs-list').text()).toContain('lifecycle/requirements/foo-2.md')
    expect(wrapper.find('button[type="submit"]').attributes('disabled')).toBeDefined()
  })

  it('shows the rejected running jobs from a 409 instead of a generic error', async () => {
    const providerSwitchStore = useProviderSwitchStore()
    vi.spyOn(providerSwitchStore, 'switchAgent').mockRejectedValue(
      new ApiError('runs_in_progress', 'cannot switch provider while runs are in progress', 409, {
        running_jobs: [{ id: 'job-2', agent: 'agent-a', artifact_path: 'lifecycle/requirements/bar-2.md' }],
      }),
    )

    const wrapper = mountModal()
    await flushPromises()

    await wrapper.find('#spm-provider').setValue('openai-a')
    await wrapper.find('#spm-model').setValue('gpt-4o')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(wrapper.find('.spm-error').exists()).toBe(false)
    expect(wrapper.find('.spm-running-warning').exists()).toBe(true)
    expect(wrapper.find('.spm-running-jobs-list').text()).toContain('lifecycle/requirements/bar-2.md')
  })
})
