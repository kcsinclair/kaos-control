// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Unit tests for CreatePipelineDialog validation logic.
 *
 * Milestone 4 — YAML editor validation (frontend)
 *
 * Covers:
 *   1. Valid YAML + valid slug — no error shown, createPipeline is called
 *   2. Invalid YAML — YAML parse error message is displayed, no API call made
 *   3. Empty slug — validation prevents submission, error is shown
 *   4. Invalid slug pattern (uppercase, spaces) — validation error is shown
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

// Mock the devops store so we can intercept createPipeline calls.
const mockCreatePipeline = vi.fn()
vi.mock('@/stores/devops', () => ({
  useDevOpsStore: () => ({
    createPipeline: mockCreatePipeline,
    activeRuns: new Map(),
  }),
}))

// Stub YamlEditor to a simple textarea so we can control the v-model value.
// The real component uses CodeMirror, which requires a DOM environment it
// cannot fully initialise under happy-dom.
vi.mock('@/components/common/YamlEditor.vue', () => ({
  default: {
    name: 'YamlEditor',
    props: ['modelValue', 'readonly'],
    emits: ['update:modelValue'],
    template: `<textarea
      :value="modelValue"
      :disabled="readonly"
      data-testid="yaml-editor"
      @input="$emit('update:modelValue', $event.target.value)"
    />`,
  },
}))

// ---------------------------------------------------------------------------
// Import the component under test AFTER mocks are in place.
// ---------------------------------------------------------------------------

import CreatePipelineDialog from '../../web/src/components/devops/CreatePipelineDialog.vue'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const VALID_YAML = `name: My Pipeline
type: build
steps:
  - name: Step 1
    command: echo hello
`

const INVALID_YAML = 'not: valid: yaml: ['

async function mountDialog(open = true) {
  setActivePinia(createPinia())

  const wrapper = mount(CreatePipelineDialog, {
    props: { open, project: 'testproject' },
  })
  await flushPromises()
  return wrapper
}

/** Set the slug input field value and trigger Vue reactivity. */
async function setSlug(wrapper: ReturnType<typeof mount>, value: string) {
  const input = wrapper.find('#pipeline-slug')
  await input.setValue(value)
}

/** Set the YAML editor value via the stub textarea. */
async function setDefinition(wrapper: ReturnType<typeof mount>, value: string) {
  const editor = wrapper.find('[data-testid="yaml-editor"]')
  await editor.setValue(value)
  // Trigger the update:modelValue event that the real YamlEditor emits.
  await editor.trigger('input')
}

/** Click the Create button and wait for async work to complete. */
async function clickCreate(wrapper: ReturnType<typeof mount>) {
  await wrapper.find('.btn-primary').trigger('click')
  await flushPromises()
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('CreatePipelineDialog — validation', () => {
  beforeEach(() => {
    mockCreatePipeline.mockReset()
  })

  it('submits successfully when slug and YAML are both valid', async () => {
    mockCreatePipeline.mockResolvedValue({ slug: 'my-pipeline', name: 'My Pipeline', type: 'build' })

    const wrapper = await mountDialog()
    await setSlug(wrapper, 'my-pipeline')
    await setDefinition(wrapper, VALID_YAML)
    await clickCreate(wrapper)

    expect(mockCreatePipeline).toHaveBeenCalledOnce()
    expect(mockCreatePipeline).toHaveBeenCalledWith('testproject', 'my-pipeline', VALID_YAML)
    // No error message should be visible.
    expect(wrapper.find('.cpd-error').exists()).toBe(false)
  })

  it('shows a YAML parse error and does not call the API for invalid YAML', async () => {
    const wrapper = await mountDialog()
    await setSlug(wrapper, 'bad-yaml')
    await setDefinition(wrapper, INVALID_YAML)
    await clickCreate(wrapper)

    expect(mockCreatePipeline).not.toHaveBeenCalled()
    const errorEl = wrapper.find('.cpd-error')
    expect(errorEl.exists()).toBe(true)
    // The error message must mention YAML.
    expect(errorEl.text().toLowerCase()).toContain('yaml')
  })

  it('shows a validation error and does not call the API when the slug is empty', async () => {
    const wrapper = await mountDialog()
    // Leave slug empty (default is '').
    await setDefinition(wrapper, VALID_YAML)
    await clickCreate(wrapper)

    expect(mockCreatePipeline).not.toHaveBeenCalled()
    const errorEl = wrapper.find('.cpd-error')
    expect(errorEl.exists()).toBe(true)
    // Error message must mention the slug field.
    expect(errorEl.text().toLowerCase()).toContain('slug')
  })

  it('shows a validation error and does not call the API for an invalid slug pattern', async () => {
    const invalidSlugs = ['MY PIPELINE', 'My-Pipeline', '-leading-hyphen', 'trailing-hyphen-']

    for (const badSlug of invalidSlugs) {
      const wrapper = await mountDialog()
      await setSlug(wrapper, badSlug)
      await setDefinition(wrapper, VALID_YAML)
      await clickCreate(wrapper)

      expect(mockCreatePipeline).not.toHaveBeenCalled()
      const errorEl = wrapper.find('.cpd-error')
      expect(errorEl.exists()).toBe(true, `expected error for slug "${badSlug}"`)
    }
  })

  it('does not render when open is false', async () => {
    const wrapper = await mountDialog(false)
    expect(wrapper.find('.cpd-overlay').exists()).toBe(false)
  })

  it('renders the dialog when open is true', async () => {
    const wrapper = await mountDialog(true)
    expect(wrapper.find('.cpd-overlay').exists()).toBe(true)
    expect(wrapper.find('.cpd-title').text()).toBe('Create Pipeline')
  })
})
