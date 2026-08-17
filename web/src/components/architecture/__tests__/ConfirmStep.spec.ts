// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { useArchitectureWizardStore } from '@/stores/architectureWizard'
import ConfirmStep from '../ConfirmStep.vue'
import type { CatalogItem } from '@/types/api'

// Frontend plan: lifecycle/frontend-plans/onboarding-architecture-selection-4-fe.md — Milestone 6
//
// ConfirmStep shows architecture + stack and standards, and makes no API
// write until "Confirm & write" (NFR-1); on commit success it emits
// committed. Also builds the breaking-requirements mapping from hard
// questions whose chosen option carries hard: true.

vi.mock('@/api/architecture', () => ({
  commitWizard: vi.fn(),
}))

import { commitWizard } from '@/api/architecture'

function arch(overrides: Partial<CatalogItem> = {}): CatalogItem {
  return {
    path: 'lifecycle/architecture/architectures/modular-monolith.md',
    slug: 'modular-monolith',
    title: 'Modular Monolith',
    summary: '',
    type: 'architecture',
    labels: ['offline-capable'],
    related_to: [],
    ...overrides,
  }
}

function stack(overrides: Partial<CatalogItem> = {}): CatalogItem {
  return {
    path: 'lifecycle/architecture/tech-stacks/go-vue.md',
    slug: 'go-vue',
    title: 'Go + Vue',
    summary: '',
    type: 'tech-stack',
    labels: [],
    related_to: [],
    ...overrides,
  }
}

describe('ConfirmStep', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('shows the architecture + stack summary and issues no commit call before confirming', () => {
    const store = useArchitectureWizardStore()
    store.chosenArchitecture = arch()
    store.chosenStack = stack()

    const wrapper = mount(ConfirmStep, { props: { project: 'demo' } })

    expect(wrapper.text()).toContain('Modular Monolith')
    expect(wrapper.text()).toContain('Go + Vue')
    expect(commitWizard).not.toHaveBeenCalled()
  })

  it('builds the breaking-requirements mapping from hard questions and commits on confirm', async () => {
    const store = useArchitectureWizardStore()
    store.chosenArchitecture = arch()
    store.chosenStack = stack()
    store.questions = [
      {
        id: 'offline',
        prompt: 'Offline-capable?',
        kind: 'hard',
        options: [
          { value: 'yes', label: 'Yes', labels: ['offline-capable'], hard: true },
          { value: 'no', label: 'No' },
        ],
      },
    ]
    store.setAnswer('offline', 'yes')
    vi.mocked(commitWizard).mockResolvedValue({
      promoted_architecture: 'lifecycle/architecture/modular-monolith.md',
      promoted_tech_stack: 'lifecycle/architecture/go-vue.md',
      archived: [],
      adr_path: 'lifecycle/architecture/decisions/adr-0001-modular-monolith.md',
      superseded_adr_path: '',
      summary_path: 'lifecycle/architecture/architecture-summary.md',
    })

    const wrapper = mount(ConfirmStep, { props: { project: 'demo' } })
    expect(wrapper.text()).toContain('Offline-capable?')
    expect(wrapper.text()).toContain('supports this')

    await wrapper.find('button.confirm-write-btn').trigger('click')
    await flushPromises()

    expect(commitWizard).toHaveBeenCalledWith('demo', {
      architecture_path: 'architectures/modular-monolith.md',
      tech_stack_path: 'tech-stacks/go-vue.md',
      answers: [{ question_id: 'offline', value: 'yes' }],
      breaking_requirements: [
        {
          Label: 'offline-capable',
          Requirement: 'Offline-capable?',
          Mapping: 'Modular Monolith supports this.',
        },
      ],
      qa: [{ Question: 'Offline-capable?', Answer: 'Yes' }],
    })
    expect(wrapper.emitted('committed')).toBeTruthy()
  })

  it('does not emit committed when the commit fails', async () => {
    const store = useArchitectureWizardStore()
    store.chosenArchitecture = arch()
    store.chosenStack = stack()
    vi.mocked(commitWizard).mockRejectedValue(new Error('boom'))

    const wrapper = mount(ConfirmStep, { props: { project: 'demo' } })
    await wrapper.find('button.confirm-write-btn').trigger('click')
    await flushPromises()

    expect(wrapper.emitted('committed')).toBeFalsy()
  })
})
