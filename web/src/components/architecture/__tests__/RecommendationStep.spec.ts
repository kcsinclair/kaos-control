// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, beforeEach, vi } from 'vitest'
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

    const wrapper = mount(RecommendationStep, { props: { project: 'testproject' } })

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

    const wrapper = mount(RecommendationStep, { props: { project: 'testproject' } })

    const banner = wrapper.find('.dropped-banner')
    expect(banner.exists()).toBe(true)
    const items = banner.findAll('li').map((li) => li.text())
    expect(items).toEqual(['mobile', 'ai-ml'])
  })

  it('does not render the dropped-constraints banner when there are none', () => {
    const store = useArchitectureWizardStore()
    store.recommendations = [recommendation()]
    store.droppedConstraints = []

    const wrapper = mount(RecommendationStep, { props: { project: 'testproject' } })
    expect(wrapper.find('.dropped-banner').exists()).toBe(false)
  })

  it('the override expander falls back to Browse', async () => {
    const store = useArchitectureWizardStore()
    store.recommendations = [recommendation()]

    const wrapper = mount(RecommendationStep, { props: { project: 'testproject' } })
    await wrapper.find('button.override-toggle').trigger('click')
    await wrapper.find('button.show-everything-btn').trigger('click')

    expect(wrapper.emitted('browse-anyway')).toBeTruthy()
  })

  it('summarises the questions and the answers given', () => {
    const store = useArchitectureWizardStore()
    store.questions = [
      {
        id: 'offline',
        prompt: 'Must it work offline?',
        kind: 'hard',
        options: [
          { value: 'yes', label: 'Yes, fully offline' },
          { value: 'no', label: 'No' },
        ],
      },
      { id: 'scale', prompt: 'Expected scale?', kind: 'soft', options: [{ value: 'high', label: 'High' }] },
    ]
    store.answers = [{ question_id: 'offline', value: 'yes' }]
    store.recommendations = [recommendation()]

    const wrapper = mount(RecommendationStep, { props: { project: 'testproject' } })
    const summary = wrapper.find('.qa-summary')
    expect(summary.exists()).toBe(true)
    expect(summary.text()).toContain('Must it work offline?')
    expect(summary.text()).toContain('Yes, fully offline') // answer label, not the raw value
    // An unanswered/skipped question is shown as "No preference".
    expect(summary.text()).toContain('No preference')
  })

  it('shows an explicit best-fit heading and badge when the match is exact', () => {
    const store = useArchitectureWizardStore()
    store.recommendations = [recommendation()]
    store.droppedConstraints = []

    const wrapper = mount(RecommendationStep, { props: { project: 'testproject' } })
    expect(wrapper.find('.recommend-title').text()).toBe('Your best-fit architecture')
    expect(wrapper.find('.recommend-card--best .recommend-badge').text()).toBe('Best fit')
  })

  it('flags the result as a closest match, not a best fit, when constraints were dropped', () => {
    const store = useArchitectureWizardStore()
    store.recommendations = [recommendation()]
    store.droppedConstraints = ['mobile']

    const wrapper = mount(RecommendationStep, { props: { project: 'testproject' } })
    expect(wrapper.find('.recommend-title').text()).toContain('closest fits')
    expect(wrapper.find('.recommend-card--best .recommend-badge').text()).toBe('Closest match')
  })

  it('fetches on mount when answers exist but no results are loaded (refresh / back-nav / race)', () => {
    const store = useArchitectureWizardStore()
    store.answers = [{ question_id: 'offline', value: 'yes' }]
    store.recommendations = []
    const spy = vi.spyOn(store, 'fetchRecommendations').mockResolvedValue()

    mount(RecommendationStep, { props: { project: 'testproject' } })

    expect(spy).toHaveBeenCalledWith('testproject')
  })

  it('does not refetch when results are already loaded', () => {
    const store = useArchitectureWizardStore()
    store.answers = [{ question_id: 'offline', value: 'yes' }]
    store.recommendations = [recommendation()]
    const spy = vi.spyOn(store, 'fetchRecommendations').mockResolvedValue()

    mount(RecommendationStep, { props: { project: 'testproject' } })

    expect(spy).not.toHaveBeenCalled()
  })

  it('says so and offers Browse when there is no match at all', async () => {
    const store = useArchitectureWizardStore()
    store.recommendations = []

    const wrapper = mount(RecommendationStep, { props: { project: 'testproject' } })
    expect(wrapper.find('.recommend-title').text()).toBe('No clear match')
    expect(wrapper.find('.recommend-cards').exists()).toBe(false)
    const noMatch = wrapper.find('.no-match')
    expect(noMatch.exists()).toBe(true)
    await noMatch.find('button.show-everything-btn').trigger('click')
    expect(wrapper.emitted('browse-anyway')).toBeTruthy()
  })
})
