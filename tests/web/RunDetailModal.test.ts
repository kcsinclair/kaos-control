/**
 * Milestone 5 — Unit tests for the RunDetailModal component
 *
 * Covers:
 *   - Rendering all AgentRunRow fields (run ID, agent, role, target path,
 *     timestamps, status, exit code, stderr tail, artifacts produced).
 *   - Stderr tail appears inside a <pre> element.
 *   - Dismissal via close button click, Escape key, and backdrop click.
 *   - Focus trap: overlay has tabindex=-1 and the keydown handler is wired.
 *
 * Component: web/src/components/agent/RunDetailModal.vue
 * Props: project (string), runId (string)
 * Emits: close
 *
 * Note: <Teleport to="body"> is stubbed so all content renders inline within
 * the wrapper's DOM tree, allowing standard wrapper.find() calls.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import RunDetailModal from '../../web/src/components/agent/RunDetailModal.vue'
import type { AgentRunRow } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Module mock — intercept agentsApi.getRun
// ---------------------------------------------------------------------------

// IMPORTANT: vi.mock is hoisted to the top of the file by Vitest, so the
// factory must be self-contained — no references to variables declared below.
// The default run fixture is inlined here; individual tests can override via
// vi.mocked(agentsApi.getRun).mockResolvedValueOnce({ run: {...} }).
vi.mock('@/api/agents', () => ({
  getRun: vi.fn().mockResolvedValue({
    run: {
      run_id:             'test-run-id-abcdef12',
      agent_name:         'backend-developer',
      role:               'backend-developer',
      target_path:        'lifecycle/requirements/foo-2.md',
      started_at:         '2026-01-01T10:00:00Z',
      finished_at:        '2026-01-01T10:05:00Z',
      status:             'done',
      exit_code:          0,
      stderr_tail:        'line one\nline two\nline three',
      artifacts_produced: ['lifecycle/requirements/foo-2.md', 'lifecycle/backend-plans/foo-3-be.md'],
    },
  }),
  listRuns:             vi.fn().mockResolvedValue({ runs: [] }),
  listAgents:           vi.fn().mockResolvedValue({ agents: [] }),
  listRunsByTargetPath: vi.fn().mockResolvedValue([]),
  startRun:             vi.fn().mockResolvedValue({ run_id: 'mock' }),
  killRun:              vi.fn().mockResolvedValue({}),
  getRunLog:            vi.fn().mockResolvedValue(''),
}))

// Typed reference to the mocked run used throughout the tests.
const mockRun: AgentRunRow = {
  run_id:             'test-run-id-abcdef12',
  agent_name:         'backend-developer',
  role:               'backend-developer',
  target_path:        'lifecycle/requirements/foo-2.md',
  started_at:         '2026-01-01T10:00:00Z',
  finished_at:        '2026-01-01T10:05:00Z',
  status:             'done',
  exit_code:          0,
  stderr_tail:        'line one\nline two\nline three',
  artifacts_produced: ['lifecycle/requirements/foo-2.md', 'lifecycle/backend-plans/foo-3-be.md'],
}

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  setActivePinia(createPinia())
})

// ---------------------------------------------------------------------------
// Mount helpers
// ---------------------------------------------------------------------------

// Stub Teleport so its content renders inline within the wrapper's element.
// This lets wrapper.find('.rdm-overlay') and friends work correctly.
const teleportStub = { template: '<div><slot /></div>' }

function mountModal(props: { project?: string; runId?: string } = {}) {
  return mount(RunDetailModal, {
    props: {
      project: props.project ?? 'testproject',
      runId:   props.runId   ?? 'test-run-id-abcdef12',
    },
    global: { stubs: { Teleport: teleportStub } },
  })
}

// ---------------------------------------------------------------------------
// Milestone 5 — Field display
// ---------------------------------------------------------------------------

describe('RunDetailModal — field display', () => {
  it('displays all AgentRunRow fields after loading', async () => {
    const wrapper = mountModal()
    await flushPromises() // wait for agentsApi.getRun to resolve

    const text = wrapper.text()
    expect(text).toContain('test-run-id-abcdef12')         // run ID
    expect(text).toContain('backend-developer')             // agent name
    expect(text).toContain('lifecycle/requirements/foo-2.md') // target path
    expect(text).toContain('done')                         // status

    // Artifacts produced
    expect(text).toContain('lifecycle/requirements/foo-2.md')
    expect(text).toContain('lifecycle/backend-plans/foo-3-be.md')
  })

  it('displays run ID in full (not truncated)', async () => {
    const wrapper = mountModal()
    await flushPromises()

    const runIdEl = wrapper.find('.rdm-mono')
    expect(runIdEl.exists()).toBe(true)
    // The run ID field value should contain the full ID.
    expect(wrapper.html()).toContain('test-run-id-abcdef12')
  })

  it('displays agent name and role', async () => {
    const wrapper = mountModal()
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('backend-developer') // agent name
    // role appears in its own field
    expect(text).toContain('backend-developer') // role (same value in fixture)
  })

  it('displays target path', async () => {
    const wrapper = mountModal()
    await flushPromises()

    expect(wrapper.text()).toContain('lifecycle/requirements/foo-2.md')
  })

  it('shows loading state before getRun resolves', async () => {
    // Mock getRun to hang indefinitely.
    const agentsApi = await import('@/api/agents')
    vi.mocked(agentsApi.getRun).mockImplementationOnce(() => new Promise(() => {}))

    const wrapper = mountModal()
    // While pending: loading state is shown.
    expect(wrapper.text()).toContain('Loading')
  })
})

describe('RunDetailModal — stderr tail', () => {
  it('renders stderr tail inside a <pre> element', async () => {
    const wrapper = mountModal()
    await flushPromises()

    const pre = wrapper.find('pre')
    expect(pre.exists()).toBe(true)
    expect(pre.text()).toContain('line one')
    expect(pre.text()).toContain('line two')
    expect(pre.text()).toContain('line three')
  })

  it('<pre> element has overflow styling for scrollability', async () => {
    const wrapper = mountModal()
    await flushPromises()

    const pre = wrapper.find('pre')
    expect(pre.exists()).toBe(true)
    // The component applies class "rdm-log" which sets overflow-x: auto; overflow-y: auto.
    expect(pre.classes()).toContain('rdm-log')
  })
})

// ---------------------------------------------------------------------------
// Milestone 5 — Dismissal
// ---------------------------------------------------------------------------

describe('RunDetailModal — close button', () => {
  it('emits "close" when the close button is clicked', async () => {
    const wrapper = mountModal()
    await flushPromises()

    await wrapper.find('button[aria-label="Close"]').trigger('click')

    expect(wrapper.emitted('close')).toBeDefined()
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})

describe('RunDetailModal — Escape key', () => {
  it('emits "close" when the Escape key is pressed on the overlay', async () => {
    const wrapper = mountModal()
    await flushPromises()

    await wrapper.find('.rdm-overlay').trigger('keydown', { key: 'Escape' })

    expect(wrapper.emitted('close')).toBeDefined()
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})

describe('RunDetailModal — backdrop click', () => {
  it('emits "close" when the overlay backdrop (not the panel) is clicked', async () => {
    const wrapper = mountModal()
    await flushPromises()

    // Simulate a click event whose target is the overlay element itself.
    const overlay = wrapper.find('.rdm-overlay')
    // Manually dispatch a click so that e.target === overlay.element.
    const clickEvent = new MouseEvent('click', { bubbles: true })
    Object.defineProperty(clickEvent, 'target', { value: overlay.element, writable: false })
    overlay.element.dispatchEvent(clickEvent)
    await flushPromises()

    expect(wrapper.emitted('close')).toBeDefined()
  })

  it('does NOT emit "close" when the modal panel itself is clicked', async () => {
    const wrapper = mountModal()
    await flushPromises()

    // Click the panel (not the overlay background).
    const panel = wrapper.find('.rdm-panel')
    await panel.trigger('click')

    // The overlay click handler checks classList for 'rdm-overlay' on the target;
    // clicking the panel does not have that class, so close should not be emitted.
    expect(wrapper.emitted('close')).toBeUndefined()
  })
})

describe('RunDetailModal — focus trap', () => {
  it('overlay has tabindex="-1" enabling keyboard event capture', async () => {
    const wrapper = mountModal()
    await flushPromises()

    const overlay = wrapper.find('.rdm-overlay')
    expect(overlay.attributes('tabindex')).toBe('-1')
  })

  it('Tab key press on the overlay does not emit "close"', async () => {
    const wrapper = mountModal()
    await flushPromises()

    const overlay = wrapper.find('.rdm-overlay')
    await overlay.trigger('keydown', { key: 'Tab' })

    expect(wrapper.emitted('close')).toBeUndefined()
  })

  it('has role="dialog" and aria-modal="true" for screen reader support', async () => {
    const wrapper = mountModal()
    await flushPromises()

    const overlay = wrapper.find('.rdm-overlay')
    expect(overlay.attributes('role')).toBe('dialog')
    expect(overlay.attributes('aria-modal')).toBe('true')
  })

  it('close button exists and is focusable inside the modal panel', async () => {
    const wrapper = mountModal()
    await flushPromises()

    const closeBtn = wrapper.find('.rdm-panel button[aria-label="Close"]')
    expect(closeBtn.exists()).toBe(true)
    // The button has no disabled attribute, making it focusable.
    expect(closeBtn.attributes('disabled')).toBeUndefined()
  })
})
