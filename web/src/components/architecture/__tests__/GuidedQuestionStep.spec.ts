// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'

// Frontend plan: lifecycle/frontend-plans/onboarding-architecture-selection-4-fe.md — Milestone 4
//
// Exactly the configured questions render (≤10), each skippable; skipping a
// question omits it from the answer payload; completing/skipping all
// triggers a recommend call (FR-7, FR-8, NFR-5).

vi.mock('@/api/architecture', () => ({
  saveWizardState: vi.fn().mockResolvedValue(undefined),
  recommend: vi.fn(),
}))

import { recommend } from '@/api/architecture'
import { useArchitectureWizardStore } from '@/stores/architectureWizard'
import GuidedQuestionStep from '../GuidedQuestionStep.vue'

describe('GuidedQuestionStep', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders exactly the configured questions, each skippable, one at a time', async () => {
    const store = useArchitectureWizardStore()
    store.setPath('guided')
    store.questions = [
      {
        id: 'offline',
        prompt: 'Offline-capable?',
        kind: 'hard',
        options: [
          { value: 'yes', label: 'Yes' },
          { value: 'no', label: 'No' },
        ],
      },
      {
        id: 'realtime',
        prompt: 'Realtime?',
        kind: 'soft',
        options: [
          { value: 'yes', label: 'Yes' },
          { value: 'no', label: 'No' },
        ],
      },
    ]
    vi.mocked(recommend).mockResolvedValue({ recommendations: [], dropped_constraints: [] })

    const wrapper = mount(GuidedQuestionStep, { props: { project: 'demo' } })
    await flushPromises()

    expect(wrapper.text()).toContain('Offline-capable?')
    expect(wrapper.find('button.skip-btn').exists()).toBe(true)
    expect(wrapper.find('button.decide-btn').exists()).toBe(true)

    await wrapper.find('button.skip-btn').trigger('click')
    await flushPromises()

    expect(store.answerFor('offline')).toBeUndefined()
    expect(wrapper.text()).toContain('Realtime?')
  })

  it('completing/skipping all questions triggers a recommend call and emits complete', async () => {
    const store = useArchitectureWizardStore()
    store.setPath('guided')
    store.questions = [
      {
        id: 'offline',
        prompt: 'Offline-capable?',
        kind: 'hard',
        options: [{ value: 'yes', label: 'Yes' }],
      },
    ]
    vi.mocked(recommend).mockResolvedValue({ recommendations: [], dropped_constraints: [] })

    const wrapper = mount(GuidedQuestionStep, { props: { project: 'demo' } })
    await flushPromises()

    await wrapper.find('button.option-btn').trigger('click')
    await flushPromises()

    expect(store.answerFor('offline')).toBe('yes')
    expect(vi.mocked(recommend)).toHaveBeenCalledWith('demo', [{ question_id: 'offline', value: 'yes' }])
    expect(wrapper.emitted('complete')).toBeTruthy()
  })

  it('emits browse-anyway when "show me everything anyway" is clicked', async () => {
    const store = useArchitectureWizardStore()
    store.setPath('guided')
    store.questions = [
      { id: 'offline', prompt: 'Offline-capable?', kind: 'hard', options: [{ value: 'yes', label: 'Yes' }] },
    ]

    const wrapper = mount(GuidedQuestionStep, { props: { project: 'demo' } })
    await flushPromises()

    await wrapper.find('button.show-everything-btn').trigger('click')
    expect(wrapper.emitted('browse-anyway')).toBeTruthy()
  })
})
