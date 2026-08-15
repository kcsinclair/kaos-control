// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 2 — Form populates every exposed field (web component)
 *
 * Verifies [[agent-editor-incomplete-config-load]] FR-1/FR-6/FR-7: opening
 * an existing agent in AgentConfigForm.vue loads timeout_minutes,
 * git_identity, and a multi-key prompt_templates map from `props.initial`,
 * and that submitting with no edits round-trips all of them losslessly.
 * Also covers FR-7: creating a new agent (no `initial`) starts empty.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import AgentConfigForm from '../../web/src/components/agent/AgentConfigForm.vue'
import type { AgentFormData } from '../../web/src/components/agent/AgentConfigForm.vue'
import type { AgentSummary } from '../../web/src/types/api'

vi.mock('@/api/ollama', () => ({
  listInstances: vi.fn().mockResolvedValue({ instances: [] }),
  getHealth: vi.fn().mockResolvedValue({ ok: true }),
  listModels: vi.fn().mockResolvedValue({ models: [] }),
}))

const AVAILABLE_ROLES = ['analyst', 'backend-developer', 'frontend-developer', 'test-developer', 'qa', 'reviewer']

// Mirrors idea-capture: three role-keyed prompt templates, the exact shape
// that a lossy single-textarea representation collapses.
const IDEA_CAPTURE_TEMPLATES: Record<string, string> = {
  analyst: 'Capture the idea at {target_path}.',
  'backend-developer': 'Generate an idea from {target_path}.',
  qa: 'Raise a defect for {target_path}.',
}

function makeInitial(overrides: Partial<AgentSummary> = {}): AgentSummary {
  return {
    name: 'idea-capture',
    roles: ['analyst'],
    driver: 'claude-code-cli',
    model: 'claude-opus-4-6',
    timeout_minutes: 45,
    git_identity: { name: 'Idea Capture Agent', email: 'idea-capture@test.local' },
    prompt_templates: { ...IDEA_CAPTURE_TEMPLATES },
    allowed_write_paths: ['lifecycle/ideas'],
    ...overrides,
  }
}

beforeEach(() => {
  setActivePinia(createPinia())
})

describe('AgentConfigForm — loads every exposed field from props.initial', () => {
  it('populates the timeout input with the configured value, not 0', async () => {
    const wrapper = mount(AgentConfigForm, {
      props: { initial: makeInitial(), availableRoles: AVAILABLE_ROLES },
    })
    await flushPromises()

    const timeoutInput = wrapper.find<HTMLInputElement>('#acf-timeout')
    expect(timeoutInput.exists()).toBe(true)
    expect(timeoutInput.element.value).toBe('45')
  })

  it('populates git identity name and email inputs', async () => {
    const wrapper = mount(AgentConfigForm, {
      props: { initial: makeInitial(), availableRoles: AVAILABLE_ROLES },
    })
    await flushPromises()

    expect(wrapper.find<HTMLInputElement>('#acf-git-name').element.value).toBe('Idea Capture Agent')
    expect(wrapper.find<HTMLInputElement>('#acf-git-email').element.value).toBe('idea-capture@test.local')
  })

  it('renders all three prompt templates, each keyed by role', async () => {
    const wrapper = mount(AgentConfigForm, {
      props: { initial: makeInitial(), availableRoles: AVAILABLE_ROLES },
    })
    await flushPromises()

    const roleNames = wrapper.findAll('.acf-prompt-role-name').map((n) => n.text())
    expect(roleNames.sort()).toEqual(['analyst', 'backend-developer', 'qa'].sort())

    const textareas = wrapper.findAll<HTMLTextAreaElement>('.acf-prompt-entry textarea')
    expect(textareas.length).toBe(3)
    const bodies = textareas.map((t) => t.element.value)
    expect(bodies).toContain(IDEA_CAPTURE_TEMPLATES.analyst)
    expect(bodies).toContain(IDEA_CAPTURE_TEMPLATES['backend-developer'])
    expect(bodies).toContain(IDEA_CAPTURE_TEMPLATES.qa)
  })
})

describe('AgentConfigForm — submit round-trips loaded fields losslessly', () => {
  it('emits the loaded timeout_minutes and git identity unchanged on submit', async () => {
    const wrapper = mount(AgentConfigForm, {
      props: { initial: makeInitial(), availableRoles: AVAILABLE_ROLES },
    })
    await flushPromises()

    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    const emitted = wrapper.emitted('submit')
    expect(emitted).toBeTruthy()
    const data = emitted![0][0] as AgentFormData
    expect(data.timeout_minutes).toBe(45)
    expect(data.git_identity_name).toBe('Idea Capture Agent')
    expect(data.git_identity_email).toBe('idea-capture@test.local')
  })

  it('emits all three prompt_templates keys intact, with no dropped or merged keys (FR-6)', async () => {
    const wrapper = mount(AgentConfigForm, {
      props: { initial: makeInitial(), availableRoles: AVAILABLE_ROLES },
    })
    await flushPromises()

    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    const data = wrapper.emitted('submit')![0][0] as AgentFormData
    expect(Object.keys(data.prompt_templates).sort()).toEqual(
      ['analyst', 'backend-developer', 'qa'].sort(),
    )
    expect(data.prompt_templates.analyst).toBe(IDEA_CAPTURE_TEMPLATES.analyst)
    expect(data.prompt_templates['backend-developer']).toBe(IDEA_CAPTURE_TEMPLATES['backend-developer'])
    expect(data.prompt_templates.qa).toBe(IDEA_CAPTURE_TEMPLATES.qa)
  })
})

describe('AgentConfigForm — create mode starts with empty defaults (FR-7)', () => {
  it('renders empty timeout, git identity, and no prompt templates when initial is absent', async () => {
    const wrapper = mount(AgentConfigForm, {
      props: { initial: null, availableRoles: AVAILABLE_ROLES },
    })
    await flushPromises()

    expect(wrapper.find<HTMLInputElement>('#acf-timeout').element.value).toBe('0')
    expect(wrapper.find<HTMLInputElement>('#acf-git-name').element.value).toBe('')
    expect(wrapper.find<HTMLInputElement>('#acf-git-email').element.value).toBe('')
    expect(wrapper.findAll('.acf-prompt-entry').length).toBe(0)
  })
})
