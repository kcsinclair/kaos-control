// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { useArchitectureWizardStore } from '@/stores/architectureWizard'
import ScaffoldStep from '../ScaffoldStep.vue'
import type { CatalogItem } from '@/types/api'

// Frontend plan: lifecycle/frontend-plans/onboarding-architecture-selection-4-fe.md — Milestone 7
//
// With available:false, renders the not-yet-available note and issues no
// runScaffold call; with available:true, naming fields render with working
// "decide for me" defaults and submit dispatches runScaffold (FR-17, FR-18).

vi.mock('@/api/architecture', () => ({
  getScaffold: vi.fn(),
  runScaffold: vi.fn(),
}))

import { getScaffold, runScaffold } from '@/api/architecture'

function arch(overrides: Partial<CatalogItem> = {}): CatalogItem {
  return {
    path: 'lifecycle/architecture/architectures/modular-monolith.md',
    slug: 'modular-monolith',
    title: 'Modular Monolith',
    summary: '',
    type: 'architecture',
    labels: [],
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

describe('ScaffoldStep', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders the not-yet-available note and issues no runScaffold call when available is false', async () => {
    const store = useArchitectureWizardStore()
    store.chosenArchitecture = arch()
    store.chosenStack = stack()
    vi.mocked(getScaffold).mockResolvedValue({ available: false, message: 'Not yet available.' })

    const wrapper = mount(ScaffoldStep, { props: { project: 'demo' } })
    await flushPromises()

    expect(wrapper.find('.scaffold-state.not-available').exists()).toBe(true)
    expect(wrapper.text()).toContain('Not yet available.')
    expect(runScaffold).not.toHaveBeenCalled()
  })

  it('renders naming fields with working defaults and dispatches runScaffold on submit', async () => {
    const store = useArchitectureWizardStore()
    store.chosenArchitecture = arch()
    store.chosenStack = stack()
    vi.mocked(getScaffold).mockResolvedValue({
      available: true,
      steps: [
        {
          key: 'config',
          title: 'Project config',
          description: 'Seed lifecycle/config.yaml',
          name_fields: [{ key: 'project_name', label: 'Project name', default_value: 'demo-app' }],
        },
      ],
    })
    vi.mocked(runScaffold).mockResolvedValue({ available: true, result: { applied: ['lifecycle/config.yaml'], skipped: [] } })

    const wrapper = mount(ScaffoldStep, { props: { project: 'demo' } })
    await flushPromises()

    // Naming fields render only once the step's `selected` checkbox is on.
    await wrapper.find('input[type="checkbox"]').setValue(true)

    const input = wrapper.find('input[type="text"]')
    expect((input.element as HTMLInputElement).value).toBe('demo-app')

    await wrapper.find('button.run-scaffold-btn').trigger('click')
    await flushPromises()

    expect(runScaffold).toHaveBeenCalledWith('demo', 'modular-monolith', 'go-vue', [
      { step_key: 'config', values: { project_name: 'demo-app' }, use_defaults: true, selected: true },
    ])
    expect(wrapper.text()).toContain('lifecycle/config.yaml')
  })

  it('editing a field flips that step off defaults', async () => {
    const store = useArchitectureWizardStore()
    store.chosenArchitecture = arch()
    store.chosenStack = stack()
    vi.mocked(getScaffold).mockResolvedValue({
      available: true,
      steps: [
        {
          key: 'config',
          title: 'Project config',
          description: 'Seed lifecycle/config.yaml',
          name_fields: [{ key: 'project_name', label: 'Project name', default_value: 'demo-app' }],
        },
      ],
    })
    vi.mocked(runScaffold).mockResolvedValue({ available: true, result: { applied: [], skipped: [] } })

    const wrapper = mount(ScaffoldStep, { props: { project: 'demo' } })
    await flushPromises()

    await wrapper.find('input[type="checkbox"]').setValue(true)
    await wrapper.find('input[type="text"]').setValue('custom-name')
    await wrapper.find('button.run-scaffold-btn').trigger('click')
    await flushPromises()

    expect(runScaffold).toHaveBeenCalledWith('demo', 'modular-monolith', 'go-vue', [
      { step_key: 'config', values: { project_name: 'custom-name' }, use_defaults: false, selected: true },
    ])
  })

  it('"decide for me" resets an edited field back to its default', async () => {
    const store = useArchitectureWizardStore()
    store.chosenArchitecture = arch()
    store.chosenStack = stack()
    vi.mocked(getScaffold).mockResolvedValue({
      available: true,
      steps: [
        {
          key: 'config',
          title: 'Project config',
          description: 'Seed lifecycle/config.yaml',
          name_fields: [{ key: 'project_name', label: 'Project name', default_value: 'demo-app' }],
        },
      ],
    })
    vi.mocked(runScaffold).mockResolvedValue({ available: true, result: { applied: [], skipped: [] } })

    const wrapper = mount(ScaffoldStep, { props: { project: 'demo' } })
    await flushPromises()

    await wrapper.find('input[type="checkbox"]').setValue(true)
    await wrapper.find('input[type="text"]').setValue('custom-name')
    await wrapper.find('button.decide-btn').trigger('click')
    expect((wrapper.find('input[type="text"]').element as HTMLInputElement).value).toBe('demo-app')

    await wrapper.find('button.run-scaffold-btn').trigger('click')
    await flushPromises()

    expect(runScaffold).toHaveBeenCalledWith('demo', 'modular-monolith', 'go-vue', [
      { step_key: 'config', values: { project_name: 'demo-app' }, use_defaults: true, selected: true },
    ])
  })
})
