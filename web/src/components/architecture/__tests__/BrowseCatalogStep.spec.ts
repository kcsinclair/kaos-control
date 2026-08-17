// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import type { CatalogItem } from '@/types/api'

// Frontend plan: lifecycle/frontend-plans/onboarding-architecture-selection-4-fe.md — Milestone 3
//
// Browse renders one card per catalog architecture with its labels/pros/cons
// (FR-5); picking one emits the chosen item; an empty catalog renders the
// guidance state instead of a crash/blank (dependency note in the plan).

vi.mock('vue-router', () => ({
  RouterLink: { template: '<a><slot /></a>' },
}))

vi.mock('@/api/architecture', () => ({
  listCatalog: vi.fn(),
}))

import { listCatalog } from '@/api/architecture'
import BrowseCatalogStep from '../BrowseCatalogStep.vue'

function arch(overrides: Partial<CatalogItem> = {}): CatalogItem {
  return {
    path: 'lifecycle/architecture/architectures/local-web.md',
    slug: 'local-web',
    title: 'Local Web-based Application',
    summary: 'Thin browser clients talking to one centralised server.',
    type: 'architecture',
    labels: ['collaborative', 'low-complexity'],
    related_to: ['lifecycle/architecture/tech-stacks/go-vue.md'],
    pros: ['Centralised data'],
    cons: ['Single point of failure'],
    ...overrides,
  }
}

describe('BrowseCatalogStep', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders one card per catalog architecture with labels/pros/cons and emits chosen on pick', async () => {
    const a = arch()
    const b = arch({
      path: 'lifecycle/architecture/architectures/modular-monolith.md',
      slug: 'modular-monolith',
      title: 'Modular Monolith',
    })
    vi.mocked(listCatalog).mockResolvedValue({ architectures: [a, b], techStacks: [] })

    const wrapper = mount(BrowseCatalogStep, { props: { project: 'demo' } })
    await flushPromises()

    const cards = wrapper.findAll('li.arch-card')
    expect(cards).toHaveLength(2)
    expect(cards[0].text()).toContain('Local Web-based Application')
    expect(cards[0].text()).toContain('collaborative')
    expect(cards[0].text()).toContain('Centralised data')
    expect(cards[0].text()).toContain('Single point of failure')

    await cards[0].find('button.arch-card-btn').trigger('click')
    expect(wrapper.emitted('chosen')?.[0]).toEqual([a])
  })

  it('renders the guidance state, not a crash/blank, for an empty catalog', async () => {
    vi.mocked(listCatalog).mockResolvedValue({ architectures: [], techStacks: [] })

    const wrapper = mount(BrowseCatalogStep, { props: { project: 'demo' } })
    await flushPromises()

    expect(wrapper.find('.browse-state.empty').exists()).toBe(true)
    expect(wrapper.find('li.arch-card').exists()).toBe(false)
  })
})
