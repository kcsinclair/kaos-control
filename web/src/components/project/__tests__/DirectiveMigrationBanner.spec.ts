// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import DirectiveMigrationBanner from '../DirectiveMigrationBanner.vue'
import DirectiveMigrationModal from '../DirectiveMigrationModal.vue'

// Frontend plan: lifecycle/frontend-plans/agent-directives-generation-4-fe.md — Milestone 2
//
// Verifies the banner's own affordances: it explains the multi-CLI directive
// model, "Migrate now" opens the migration modal, and "Not now" declines
// into the copyable CLI-hint state (FR-16). The modal's own diff/force flow
// is covered by DirectiveMigrationModal.spec.ts.

describe('DirectiveMigrationBanner', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('explains the multi-CLI directive model', () => {
    const wrapper = mount(DirectiveMigrationBanner, { props: { project: 'testproject' } })
    expect(wrapper.text()).toContain('multi-CLI')
    expect(wrapper.text()).toContain('AGENTS.md')
  })

  it('opens the migration modal on "Migrate now"', async () => {
    const wrapper = mount(DirectiveMigrationBanner, { props: { project: 'testproject' } })
    expect(wrapper.findComponent(DirectiveMigrationModal).exists()).toBe(false)

    const buttons = wrapper.findAll('button')
    const migrateBtn = buttons.find((b) => b.text() === 'Migrate now')
    await migrateBtn?.trigger('click')

    expect(wrapper.findComponent(DirectiveMigrationModal).exists()).toBe(true)
  })

  it('declines into the copyable CLI-hint state on "Not now"', async () => {
    const wrapper = mount(DirectiveMigrationBanner, { props: { project: 'testproject' } })
    const buttons = wrapper.findAll('button')
    const notNowBtn = buttons.find((b) => b.text() === 'Not now')
    await notNowBtn?.trigger('click')

    expect(wrapper.find('button').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('Migrate now')
    expect(wrapper.text()).toContain('kaos-control migrate-directives')
  })
})
