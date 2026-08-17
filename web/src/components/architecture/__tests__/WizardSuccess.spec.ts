// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { useArchitectureWizardStore } from '@/stores/architectureWizard'
import WizardSuccess from '../WizardSuccess.vue'

// Frontend plan: lifecycle/frontend-plans/onboarding-architecture-selection-4-fe.md — Milestone 6
//
// On commit success, WizardSuccess renders the created-file links and
// offers scaffolding.

vi.mock('vue-router', () => ({
  RouterLink: { template: '<a><slot /></a>' },
}))

describe('WizardSuccess', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('renders links to every created file and offers scaffolding', async () => {
    const store = useArchitectureWizardStore()
    store.commitResult = {
      promoted_architecture: 'lifecycle/architecture/modular-monolith.md',
      promoted_tech_stack: 'lifecycle/architecture/go-vue.md',
      archived: [],
      adr_path: 'lifecycle/architecture/decisions/adr-0001-modular-monolith.md',
      superseded_adr_path: '',
      summary_path: 'lifecycle/architecture/architecture-summary.md',
    }

    const wrapper = mount(WizardSuccess, { props: { project: 'demo' } })

    expect(wrapper.findAll('.success-links li')).toHaveLength(4)

    await wrapper.find('button.btn-primary').trigger('click')
    expect(wrapper.emitted('scaffold')).toBeTruthy()
  })

  it('additionally shows the superseded ADR link when present', () => {
    const store = useArchitectureWizardStore()
    store.commitResult = {
      promoted_architecture: 'lifecycle/architecture/modular-monolith.md',
      promoted_tech_stack: 'lifecycle/architecture/go-vue.md',
      archived: [],
      adr_path: 'lifecycle/architecture/decisions/adr-0002-readopt-modular-monolith.md',
      superseded_adr_path: 'lifecycle/architecture/decisions/adr-0001-local-web.md',
      summary_path: 'lifecycle/architecture/architecture-summary.md',
    }

    const wrapper = mount(WizardSuccess, { props: { project: 'demo' } })
    expect(wrapper.findAll('.success-links li')).toHaveLength(5)
  })
})
