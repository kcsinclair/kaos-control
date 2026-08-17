// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import type { CatalogItem } from '@/types/api'

// Frontend plan: lifecycle/frontend-plans/onboarding-architecture-selection-4-fe.md — Milestone 3
//
// Shared by both paths: lists compatible, language-ranked stacks for the
// already-chosen architecture (FR-6, FR-10), flags the top match as
// recommended, and lets the user override by picking any other card.

vi.mock('@/api/architecture', () => ({
  listStacks: vi.fn(),
}))

import { listStacks } from '@/api/architecture'
import { useArchitectureWizardStore } from '@/stores/architectureWizard'
import StackChoiceStep from '../StackChoiceStep.vue'

function stack(overrides: Partial<CatalogItem> = {}): CatalogItem {
  return {
    path: 'lifecycle/architecture/tech-stacks/go-vue.md',
    slug: 'go-vue',
    title: 'Go + Vue',
    summary: 'Lean single-binary server + reactive SPA.',
    type: 'tech-stack',
    labels: ['go', 'vue'],
    related_to: [],
    ...overrides,
  }
}

describe('StackChoiceStep', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('lists compatible stacks with the top match flagged recommended, and emits chosen', async () => {
    const store = useArchitectureWizardStore()
    store.chosenArchitecture = {
      path: 'lifecycle/architecture/architectures/local-web.md',
      slug: 'local-web',
      title: 'Local Web',
      summary: '',
      type: 'architecture',
      labels: [],
      related_to: [],
    }
    const top = stack()
    const other = stack({ path: 'lifecycle/architecture/tech-stacks/python-fastapi.md', slug: 'python-fastapi', title: 'Python + FastAPI' })
    vi.mocked(listStacks).mockResolvedValue([top, other])

    const wrapper = mount(StackChoiceStep, { props: { project: 'demo' } })
    await flushPromises()

    const cards = wrapper.findAll('li.stack-card')
    expect(cards).toHaveLength(2)
    expect(cards[0].find('.stack-recommended-badge').exists()).toBe(true)
    expect(cards[1].find('.stack-recommended-badge').exists()).toBe(false)

    await cards[1].find('button.stack-card-btn').trigger('click')
    expect(wrapper.emitted('chosen')?.[0]).toEqual([other])
  })

  it('re-fetches when the language filter changes', async () => {
    const store = useArchitectureWizardStore()
    store.chosenArchitecture = {
      path: 'lifecycle/architecture/architectures/local-web.md',
      slug: 'local-web',
      title: 'Local Web',
      summary: '',
      type: 'architecture',
      labels: [],
      related_to: [],
    }
    store.questions = [
      {
        id: 'language',
        prompt: "What's your team's strongest language?",
        kind: 'language',
        options: [
          { value: 'go', label: 'Go' },
          { value: 'python', label: 'Python' },
        ],
      },
    ]
    vi.mocked(listStacks).mockResolvedValue([stack()])

    const wrapper = mount(StackChoiceStep, { props: { project: 'demo' } })
    await flushPromises()
    expect(listStacks).toHaveBeenCalledWith('demo', 'local-web', undefined)

    await wrapper.find('select').setValue('go')
    await flushPromises()
    expect(listStacks).toHaveBeenCalledWith('demo', 'local-web', 'go')
  })
})
