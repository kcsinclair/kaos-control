// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { useArchitectureWizardStore } from '@/stores/architectureWizard'
import RecommendationStep from '../RecommendationStep.vue'
import type { CatalogItem, WizardRecommendation } from '@/types/api'

// Frontend plan: lifecycle/frontend-plans/onboarding-architecture-selection-4-fe.md — Milestone 5
//
// Given 2-3 recommendations, all render with a visible "why" (NFR-4);
// confirming emits the chosen item; a dropped-constraints payload renders
// the closest-match banner listing exactly the dropped constraints (OQ-2).

function recommendation(overrides: Partial<CatalogItem> = {}, why: string[] = ['Offline-capable? → yes']): WizardRecommendation {
  return {
    item: {
      path: 'lifecycle/architecture/architectures/modular-monolith.md',
      slug: 'modular-monolith',
      title: 'Modular Monolith',
      summary: 'A single deployable, internally modular.',
      type: 'architecture',
      labels: ['offline-capable'],
      related_to: [],
      ...overrides,
    },
    score: 2,
    why,
  }
}

describe('RecommendationStep', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('renders all recommendations with a visible why, and emits chosen on pick', async () => {
    const store = useArchitectureWizardStore()
    store.recommendations = [
      recommendation(),
      recommendation({ slug: 'single-service-saas', title: 'Single-Service SaaS', path: 'lifecycle/architecture/architectures/single-service-saas.md' }, ['Scale? → high']),
      recommendation({ slug: 'edge-hybrid', title: 'Edge Hybrid', path: 'lifecycle/architecture/architectures/edge-hybrid.md' }, ['Default bias — no strong signal either way']),
    ]

    const wrapper = mount(RecommendationStep)

    const cards = wrapper.findAll('li.recommend-card')
    expect(cards).toHaveLength(3)
    for (const card of cards) {
      expect(card.find('.recommend-why').exists()).toBe(true)
      expect(card.find('.recommend-why').text().length).toBeGreaterThan(0)
    }

    await cards[1].find('button.recommend-card-btn').trigger('click')
    expect(wrapper.emitted('chosen')?.[0]).toEqual([store.recommendations[1].item])
  })

  it('renders the closest-match banner listing exactly the dropped constraints', () => {
    const store = useArchitectureWizardStore()
    store.recommendations = [recommendation()]
    store.droppedConstraints = ['mobile', 'ai-ml']

    const wrapper = mount(RecommendationStep)

    const banner = wrapper.find('.dropped-banner')
    expect(banner.exists()).toBe(true)
    const items = banner.findAll('li').map((li) => li.text())
    expect(items).toEqual(['mobile', 'ai-ml'])
  })

  it('does not render the dropped-constraints banner when there are none', () => {
    const store = useArchitectureWizardStore()
    store.recommendations = [recommendation()]
    store.droppedConstraints = []

    const wrapper = mount(RecommendationStep)
    expect(wrapper.find('.dropped-banner').exists()).toBe(false)
  })

  it('the override expander falls back to Browse', async () => {
    const store = useArchitectureWizardStore()
    store.recommendations = [recommendation()]

    const wrapper = mount(RecommendationStep)
    await wrapper.find('button.override-toggle').trigger('click')
    await wrapper.find('button.show-everything-btn').trigger('click')

    expect(wrapper.emitted('browse-anyway')).toBeTruthy()
  })
})
