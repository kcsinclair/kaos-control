// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 3 — Save merges and preserves non-exposed fields (web logic)
 *
 * Verifies [[agent-editor-incomplete-config-load]] FR-1/FR-3/FR-4/FR-5/FR-7:
 * handleAgentFormSubmit (web/src/views/project/AgentsRunsView.vue) merges the
 * AgentConfigForm submit payload onto the existing raw config.yaml entry
 * instead of replacing it wholesale. The merge is not extracted into a
 * standalone helper, so this drives it through a mounted AgentsRunsView with
 * @/api/config mocked (getConfig/updateConfig), per the test plan's fallback
 * instruction. parseConfigYaml/dumpConfigYaml are left as the real js-yaml
 * implementations so the captured PUT body is genuine YAML.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import AgentsRunsView from '../../web/src/views/project/AgentsRunsView.vue'
import AgentPanelRow from '../../web/src/components/agent/AgentPanelRow.vue'
import AgentConfigForm from '../../web/src/components/agent/AgentConfigForm.vue'
import type { AgentFormData } from '../../web/src/components/agent/AgentConfigForm.vue'
import { useAgentsStore } from '../../web/src/stores/agents'
import type { AgentSummary } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

vi.mock('@/api/agents', () => ({
  listRuns:   vi.fn().mockResolvedValue({ runs: [] }),
  listAgents: vi.fn().mockResolvedValue({ agents: [] }),
  startRun:   vi.fn().mockResolvedValue({ run_id: 'new-run' }),
  killRun:    vi.fn().mockResolvedValue({}),
  getRunLog:  vi.fn().mockResolvedValue(''),
}))

// AgentConfigForm mounts useOllamaInstancesStore() and calls fetchInstances()
// + checkAllHealth() onMounted; without this mock those leak real fetches
// (ECONNREFUSED in tests — see lifecycle/defects/frontend-tests-leak-unmocked-fetches.md).
vi.mock('@/api/ollama', () => ({
  listInstances: vi.fn().mockResolvedValue({ instances: [] }),
  getHealth: vi.fn().mockResolvedValue({ ok: true }),
  listModels: vi.fn().mockResolvedValue({ models: [] }),
}))

vi.mock('@/api/config', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../web/src/api/config')>()
  return {
    ...actual,
    getConfig: vi.fn(),
    updateConfig: vi.fn().mockResolvedValue({ ok: true }),
    getRoles: vi.fn().mockResolvedValue({ roles: [], users: [] }),
  }
})

import * as configApi from '../../web/src/api/config'

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// The raw config.yaml "on disk" — mirrors what the frontend reads on save.
// The target agent carries every field the editor does NOT manage, per the
// requirement's FR-3 list.
const RAW_CONFIG_YAML = `agents:
  - name: backend-developer
    role: [backend-developer]
    driver: claude-code-cli
    model: claude-opus-4-6
    active_status: in-development
    done_on_success: true
    source_types: [ticket]
    allowed_write_paths:
      - internal
    timeout_minutes: 30
    git_identity:
      name: Backend Dev Agent
      email: backend-dev@test.local
    prompt_templates:
      backend-developer: "BE prompt for {target_path}"
    on_denial: abort
    observe_only: true
    bash_allowlist:
      - "go test *"
    bash_denylist:
      - "rm *"
    endpoint: "https://legacy.example.com/endpoint"
    base_url: "http://localhost:9999"
    auth_token: "s3cr3t-must-survive-untouched"
`

// The AgentSummary the (non-secret) GET /agents payload would return for the
// same entry — this is what AgentConfigForm receives as `props.initial`.
const AGENT_SUMMARY: AgentSummary = {
  name: 'backend-developer',
  roles: ['backend-developer'],
  driver: 'claude-code-cli',
  model: 'claude-opus-4-6',
  timeout_minutes: 30,
  git_identity: { name: 'Backend Dev Agent', email: 'backend-dev@test.local' },
  prompt_templates: { 'backend-developer': 'BE prompt for {target_path}' },
  allowed_write_paths: ['internal'],
  active_status: 'in-development',
  done_on_success: true,
  source_types: ['ticket'],
  on_denial: 'abort',
  observe_only: true,
  bash_allowlist: ['go test *'],
  bash_denylist: ['rm *'],
  endpoint: 'https://legacy.example.com/endpoint',
  base_url: 'http://localhost:9999',
}

function loadedFormData(overrides: Partial<AgentFormData> = {}): AgentFormData {
  return {
    name: 'backend-developer',
    roles: ['backend-developer'],
    driver: 'claude-code-cli',
    model: 'claude-opus-4-6',
    ollama_instance: '',
    ollama_endpoint: 'chat',
    allowed_write_paths: ['internal'],
    timeout_minutes: 30,
    git_identity_name: 'Backend Dev Agent',
    git_identity_email: 'backend-dev@test.local',
    prompt_templates: { 'backend-developer': 'BE prompt for {target_path}' },
    ...overrides,
  }
}

// ---------------------------------------------------------------------------
// Mount + drive helpers
// ---------------------------------------------------------------------------

async function mountView() {
  const pinia = createPinia()
  setActivePinia(pinia)

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/p/:project', component: AgentsRunsView },
      { path: '/:pathMatch(.*)*', component: { template: '<div/>' } },
    ],
  })
  await router.push('/p/testproject')
  await router.isReady()

  const wrapper = mount(AgentsRunsView, {
    global: { plugins: [pinia, router] },
  })
  await flushPromises()

  const store = useAgentsStore()
  store.$patch({ agents: [AGENT_SUMMARY], runs: [], loading: false })
  await flushPromises()

  return { wrapper, router, store }
}

// Opens the edit form for `backend-developer` and submits the given form
// payload directly on the mounted AgentConfigForm, exercising
// handleAgentFormSubmit's merge logic without depending on DOM input wiring
// (already covered by AgentConfigForm.load.spec.ts, Milestone 2).
async function editAndSubmit(wrapper: ReturnType<typeof mount>, formData: AgentFormData) {
  const panelRow = wrapper.findComponent(AgentPanelRow)
  panelRow.vm.$emit('edit', AGENT_SUMMARY)
  await flushPromises()

  const form = wrapper.findComponent(AgentConfigForm)
  expect(form.exists()).toBe(true)
  form.vm.$emit('submit', formData)
  await flushPromises()
}

function capturedPutYaml(): string {
  const mockFn = vi.mocked(configApi.updateConfig)
  expect(mockFn).toHaveBeenCalledTimes(1)
  const [, raw] = mockFn.mock.calls[0]
  return raw as string
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.mocked(configApi.getConfig).mockResolvedValue({ raw: RAW_CONFIG_YAML })
})

afterEach(() => {
  vi.clearAllMocks()
})

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('AgentsRunsView — save merges onto the existing entry (FR-3, FR-5)', () => {
  it('editing only model preserves every non-exposed field', async () => {
    const { wrapper } = await mountView()

    await editAndSubmit(wrapper, loadedFormData({ model: 'claude-opus-5-1' }))

    const raw = capturedPutYaml()
    const cfg = configApi.parseConfigYaml(raw) as { agents: Record<string, unknown>[] }
    const entry = cfg.agents.find((a) => a.name === 'backend-developer')!

    expect(entry.model).toBe('claude-opus-5-1')
    expect(entry.active_status).toBe('in-development')
    expect(entry.done_on_success).toBe(true)
    expect(entry.source_types).toEqual(['ticket'])
    expect(entry.on_denial).toBe('abort')
    expect(entry.observe_only).toBe(true)
    expect(entry.bash_allowlist).toEqual(['go test *'])
    expect(entry.bash_denylist).toEqual(['rm *'])
    expect(entry.endpoint).toBe('https://legacy.example.com/endpoint')
    expect(entry.base_url).toBe('http://localhost:9999')
    expect(entry.auth_token).toBe('s3cr3t-must-survive-untouched')
  })

  it('open-then-save with no edits produces a semantically identical entry (FR-1 round-trip)', async () => {
    const { wrapper } = await mountView()

    await editAndSubmit(wrapper, loadedFormData())

    const raw = capturedPutYaml()
    const cfg = configApi.parseConfigYaml(raw) as { agents: Record<string, unknown>[] }
    const entry = cfg.agents.find((a) => a.name === 'backend-developer')!

    const original = configApi.parseConfigYaml(RAW_CONFIG_YAML) as { agents: Record<string, unknown>[] }
    const originalEntry = original.agents.find((a) => a.name === 'backend-developer')!

    expect(entry).toEqual(originalEntry)
  })
})

describe('AgentsRunsView — clearing an exposed field removes it (FR-4)', () => {
  it('clearing allowed_write_paths removes the key', async () => {
    const { wrapper } = await mountView()

    await editAndSubmit(wrapper, loadedFormData({ allowed_write_paths: [] }))

    const raw = capturedPutYaml()
    const cfg = configApi.parseConfigYaml(raw) as { agents: Record<string, unknown>[] }
    const entry = cfg.agents.find((a) => a.name === 'backend-developer')!

    expect(entry.allowed_write_paths).toBeUndefined()
  })

  it('clearing git_identity name and email removes the object', async () => {
    const { wrapper } = await mountView()

    await editAndSubmit(wrapper, loadedFormData({ git_identity_name: '', git_identity_email: '' }))

    const raw = capturedPutYaml()
    const cfg = configApi.parseConfigYaml(raw) as { agents: Record<string, unknown>[] }
    const entry = cfg.agents.find((a) => a.name === 'backend-developer')!

    expect(entry.git_identity).toBeUndefined()
  })
})

describe('AgentsRunsView — create emits only populated keys (FR-7)', () => {
  it('creating a new agent with only required fields adds no spurious empty values', async () => {
    const { wrapper } = await mountView()

    // "New Agent" opens the form with initial=null.
    await wrapper.find('button.btn-secondary').trigger('click')
    await flushPromises()

    const form = wrapper.findComponent(AgentConfigForm)
    expect(form.exists()).toBe(true)
    form.vm.$emit('submit', {
      name: 'new-agent',
      roles: ['backend-developer'],
      driver: 'codex-cli',
      model: '',
      ollama_instance: '',
      ollama_endpoint: 'chat',
      allowed_write_paths: [],
      timeout_minutes: 0,
      git_identity_name: '',
      git_identity_email: '',
      prompt_templates: {},
    } satisfies AgentFormData)
    await flushPromises()

    const raw = capturedPutYaml()
    const cfg = configApi.parseConfigYaml(raw) as { agents: Record<string, unknown>[] }
    const entry = cfg.agents.find((a) => a.name === 'new-agent')!

    expect(entry).toBeDefined()
    expect(entry.model).toBeUndefined()
    expect(entry.allowed_write_paths).toBeUndefined()
    expect(entry.git_identity).toBeUndefined()
    expect(entry.prompt_templates).toBeUndefined()
    expect(entry.ollama_instance).toBeUndefined()
    expect(entry.ollama_endpoint).toBeUndefined()
    // timeout_minutes is always emitted by design (0 = unlimited).
    expect(entry.timeout_minutes).toBe(0)
  })
})
