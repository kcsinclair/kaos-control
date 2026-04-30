/**
 * Integration tests for AppHeader — Running Agents Indicator
 *
 * Covers:
 *   Milestone 1  — Mock store helper provides controllable activeRuns
 *   Milestone 2  — Indicator visibility (shown/hidden based on state and route)
 *   Milestone 3  — Count display and singular/plural grammar + reactive updates
 *   Milestone 4  — Click navigation to /p/:project/agents
 *   Milestone 5  — Accessibility: aria-label and prefers-reduced-motion CSS
 *   Milestone 6  — RunStatusChip removal verification
 *
 * Notes on testing approach:
 * ─────────────────────────
 * The agents store is intercepted via vi.mock so that `useAgentsStore` returns
 * our mock factory, giving tests full control over `activeRuns` without any
 * real API or WebSocket calls.
 *
 * Router navigation is verified by inspecting `router.currentRoute` after
 * triggering a click on the indicator link. Since the indicator is rendered as
 * a <RouterLink> we also assert the rendered `href` attribute.
 *
 * CSS inspection: happy-dom does not compute layout or inject scoped styles,
 * so the `prefers-reduced-motion` test checks the raw component source for the
 * media-query rule rather than `getComputedStyle`.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { ref, nextTick } from 'vue'
import * as fs from 'node:fs'
import * as path from 'node:path'
import AppHeader from '../../web/src/components/layout/AppHeader.vue'
import { makeRunningRun, createMockAgentsStore } from './helpers/mockAgentsStore'
import type { AgentRunRow } from '../../web/src/types/api'

// ---------------------------------------------------------------------------
// Per-test state — each test gets its own runsRef controlled by the helper
// ---------------------------------------------------------------------------

let _runsRef = ref<AgentRunRow[]>([])

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

vi.mock('@/api/client', () => ({
  api: {
    get: vi.fn().mockResolvedValue({}),
  },
  ApiError: class ApiError extends Error {},
}))

vi.mock('@/api/ws', () => ({
  getProjectWs: vi.fn(() => ({
    onType: vi.fn(() => () => {}),
  })),
}))

vi.mock('@/stores/agents', () => ({
  useAgentsStore: () => {
    const { useMockAgentsStore } = createMockAgentsStore(_runsRef.value)
    // We need to return an object that proxies the reactive runsRef
    // Instead, we return a live computed object
    const activeRuns = {
      get value() {
        return _runsRef.value.filter((r) => r.status === 'running')
      },
    }
    return {
      get activeRuns() {
        return _runsRef.value.filter((r) => r.status === 'running')
      },
    }
  },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: vi.fn(() => ({
    me: null,
    isAuthenticated: false,
    logout: vi.fn(),
  })),
}))

vi.mock('@/stores/ui', () => ({
  useUiStore: vi.fn(() => ({
    error: vi.fn(),
  })),
}))

vi.mock('@/stores/theme', () => ({
  useThemeStore: vi.fn(() => ({
    isDark: false,
    toggle: vi.fn(),
  })),
}))

// ---------------------------------------------------------------------------
// Router factory
// ---------------------------------------------------------------------------

function makeRouter(path = '/p/my-project') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/p/:project/:sub*', component: { template: '<div/>' } },
      { path: '/:pathMatch(.*)*', component: { template: '<div/>' } },
    ],
  })
  router.push(path)
  return router
}

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

async function mountHeader(opts: { path?: string } = {}) {
  const pinia = createPinia()
  setActivePinia(pinia)

  const router = makeRouter(opts.path ?? '/p/my-project')
  await router.isReady()

  const wrapper = mount(AppHeader, {
    global: { plugins: [pinia, router] },
    attachTo: document.body,
  })

  await flushPromises()
  return { wrapper, router }
}

// ---------------------------------------------------------------------------
// Cleanup
// ---------------------------------------------------------------------------

beforeEach(() => {
  _runsRef = ref<AgentRunRow[]>([])
})

afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

// ===========================================================================
// Milestone 1 — Mock Store Helper
// ===========================================================================

describe('AppHeaderRunIndicator — Milestone 1: mock store helper', () => {
  it('helper starts with zero active runs', () => {
    expect(_runsRef.value.filter((r) => r.status === 'running')).toHaveLength(0)
  })

  it('helper can be set to 1 active run', () => {
    _runsRef.value = [makeRunningRun('run-1')]
    expect(_runsRef.value.filter((r) => r.status === 'running')).toHaveLength(1)
  })

  it('helper can be set to N active runs', () => {
    _runsRef.value = [makeRunningRun('r1'), makeRunningRun('r2'), makeRunningRun('r3')]
    expect(_runsRef.value.filter((r) => r.status === 'running')).toHaveLength(3)
  })

  it('runs can be mutated mid-test to simulate agents stopping', () => {
    _runsRef.value = [makeRunningRun('r1'), makeRunningRun('r2')]
    expect(_runsRef.value.filter((r) => r.status === 'running')).toHaveLength(2)
    _runsRef.value = []
    expect(_runsRef.value.filter((r) => r.status === 'running')).toHaveLength(0)
  })

  it('makeRunningRun produces a run with status "running"', () => {
    const run = makeRunningRun('test-id')
    expect(run.status).toBe('running')
    expect(run.run_id).toBe('test-id')
  })
})

// ===========================================================================
// Milestone 2 — Indicator Visibility
// ===========================================================================

describe('AppHeaderRunIndicator — Milestone 2: indicator visibility', () => {
  it('indicator is hidden when activeRuns is empty on a project route', async () => {
    _runsRef.value = []
    const { wrapper } = await mountHeader({ path: '/p/my-project' })
    expect(wrapper.find('.header-run-indicator').exists()).toBe(false)
  })

  it('indicator is visible when 1 agent is running on a project route', async () => {
    _runsRef.value = [makeRunningRun('r1')]
    const { wrapper } = await mountHeader({ path: '/p/my-project' })
    expect(wrapper.find('.header-run-indicator').exists()).toBe(true)
  })

  it('indicator is hidden on a non-project route even if agents are running', async () => {
    _runsRef.value = [makeRunningRun('r1')]
    const { wrapper } = await mountHeader({ path: '/login' })
    expect(wrapper.find('.header-run-indicator').exists()).toBe(false)
  })

  it('indicator is visible when multiple agents are running', async () => {
    _runsRef.value = [makeRunningRun('r1'), makeRunningRun('r2')]
    const { wrapper } = await mountHeader({ path: '/p/my-project' })
    expect(wrapper.find('.header-run-indicator').exists()).toBe(true)
  })
})

// ===========================================================================
// Milestone 3 — Count Display and Grammar
// ===========================================================================

describe('AppHeaderRunIndicator — Milestone 3: count display and grammar', () => {
  it('shows "1 running agent" (singular) when 1 agent is running', async () => {
    _runsRef.value = [makeRunningRun('r1')]
    const { wrapper } = await mountHeader({ path: '/p/my-project' })
    const indicator = wrapper.find('.header-run-indicator')
    expect(indicator.exists()).toBe(true)
    expect(indicator.text()).toContain('1')
    expect(indicator.text()).toContain('running agent')
    // No trailing "s" — singular form
    expect(indicator.text()).not.toMatch(/running agents/)
  })

  it('shows "3 running agents" (plural) when 3 agents are running', async () => {
    _runsRef.value = [makeRunningRun('r1'), makeRunningRun('r2'), makeRunningRun('r3')]
    const { wrapper } = await mountHeader({ path: '/p/my-project' })
    const indicator = wrapper.find('.header-run-indicator')
    expect(indicator.exists()).toBe(true)
    expect(indicator.text()).toContain('3')
    expect(indicator.text()).toContain('running agents')
  })

  it('shows "2 running agents" (plural) when 2 agents are running', async () => {
    _runsRef.value = [makeRunningRun('r1'), makeRunningRun('r2')]
    const { wrapper } = await mountHeader({ path: '/p/my-project' })
    const indicator = wrapper.find('.header-run-indicator')
    expect(indicator.text()).toContain('2')
    expect(indicator.text()).toContain('running agents')
  })

  it('indicator disappears when all runs are removed', async () => {
    _runsRef.value = [makeRunningRun('r1')]
    const { wrapper } = await mountHeader({ path: '/p/my-project' })
    expect(wrapper.find('.header-run-indicator').exists()).toBe(true)

    _runsRef.value = []
    await nextTick()
    expect(wrapper.find('.header-run-indicator').exists()).toBe(false)
  })

  it('text updates reactively when a second run is added', async () => {
    _runsRef.value = [makeRunningRun('r1')]
    const { wrapper } = await mountHeader({ path: '/p/my-project' })
    expect(wrapper.find('.header-run-indicator').text()).toContain('1')
    expect(wrapper.find('.header-run-indicator').text()).not.toMatch(/running agents/)

    _runsRef.value = [makeRunningRun('r1'), makeRunningRun('r2')]
    await nextTick()
    expect(wrapper.find('.header-run-indicator').text()).toContain('2')
    expect(wrapper.find('.header-run-indicator').text()).toContain('running agents')
  })

  it('removes then restores indicator reactively', async () => {
    _runsRef.value = [makeRunningRun('r1')]
    const { wrapper } = await mountHeader({ path: '/p/my-project' })
    expect(wrapper.find('.header-run-indicator').exists()).toBe(true)

    _runsRef.value = []
    await nextTick()
    expect(wrapper.find('.header-run-indicator').exists()).toBe(false)

    _runsRef.value = [makeRunningRun('r2')]
    await nextTick()
    expect(wrapper.find('.header-run-indicator').exists()).toBe(true)
    expect(wrapper.find('.header-run-indicator').text()).toContain('1')
  })
})

// ===========================================================================
// Milestone 4 — Click Navigation
// ===========================================================================

describe('AppHeaderRunIndicator — Milestone 4: click navigation', () => {
  it('indicator renders as a link to /p/:project/agents', async () => {
    _runsRef.value = [makeRunningRun('r1')]
    const { wrapper } = await mountHeader({ path: '/p/my-project' })
    const indicator = wrapper.find('.header-run-indicator')
    expect(indicator.exists()).toBe(true)
    // RouterLink renders an <a> with the resolved href
    const href = indicator.attributes('href')
    expect(href).toBe('/p/my-project/agents')
  })

  it('clicking the indicator navigates to the agents view', async () => {
    _runsRef.value = [makeRunningRun('r1')]
    const { wrapper, router } = await mountHeader({ path: '/p/my-project' })
    const indicator = wrapper.find('.header-run-indicator')
    expect(indicator.exists()).toBe(true)

    await indicator.trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.fullPath).toBe('/p/my-project/agents')
  })

  it('navigation uses the correct project slug from the route', async () => {
    _runsRef.value = [makeRunningRun('r1')]
    const { wrapper } = await mountHeader({ path: '/p/other-project' })
    const indicator = wrapper.find('.header-run-indicator')
    const href = indicator.attributes('href')
    expect(href).toBe('/p/other-project/agents')
  })
})

// ===========================================================================
// Milestone 5 — Accessibility
// ===========================================================================

describe('AppHeaderRunIndicator — Milestone 5: accessibility', () => {
  it('indicator has an aria-label when 1 agent is running', async () => {
    _runsRef.value = [makeRunningRun('r1')]
    const { wrapper } = await mountHeader({ path: '/p/my-project' })
    const indicator = wrapper.find('.header-run-indicator')
    const label = indicator.attributes('aria-label')
    expect(label).toBeTruthy()
    expect(label).toContain('1')
    expect(label).toContain('running agent')
  })

  it('aria-label reflects "running agents" (plural) with 2 active runs', async () => {
    _runsRef.value = [makeRunningRun('r1'), makeRunningRun('r2')]
    const { wrapper } = await mountHeader({ path: '/p/my-project' })
    const indicator = wrapper.find('.header-run-indicator')
    const label = indicator.attributes('aria-label')
    expect(label).toContain('2')
    expect(label).toContain('running agents')
  })

  it('aria-label includes a navigation hint', async () => {
    _runsRef.value = [makeRunningRun('r1')]
    const { wrapper } = await mountHeader({ path: '/p/my-project' })
    const indicator = wrapper.find('.header-run-indicator')
    const label = indicator.attributes('aria-label') ?? ''
    // The label should tell users what clicking does
    expect(label.toLowerCase()).toMatch(/view|click|agent/)
  })

  it('AppHeader source contains a prefers-reduced-motion media query', () => {
    // We read the component source to verify the CSS rule exists,
    // since happy-dom does not inject scoped styles into getComputedStyle.
    const componentPath = path.resolve(
      __dirname,
      '../../web/src/components/layout/AppHeader.vue',
    )
    const source = fs.readFileSync(componentPath, 'utf-8')
    expect(source).toContain('prefers-reduced-motion')
  })

  it('AppHeader source disables pulse animation under prefers-reduced-motion', () => {
    const componentPath = path.resolve(
      __dirname,
      '../../web/src/components/layout/AppHeader.vue',
    )
    const source = fs.readFileSync(componentPath, 'utf-8')
    // The reduced-motion rule should set animation to none
    expect(source).toContain('animation: none')
  })
})

// ===========================================================================
// Milestone 6 — RunStatusChip Removal Verification
// ===========================================================================

describe('AppHeaderRunIndicator — Milestone 6: RunStatusChip removal', () => {
  it('RunStatusChip.vue does not exist on disk', () => {
    const chipPath = path.resolve(
      __dirname,
      '../../web/src/components/agent/RunStatusChip.vue',
    )
    expect(fs.existsSync(chipPath)).toBe(false)
  })

  it('WorkspaceView.vue does not reference RunStatusChip', () => {
    const viewPath = path.resolve(
      __dirname,
      '../../web/src/views/project/WorkspaceView.vue',
    )
    const source = fs.readFileSync(viewPath, 'utf-8')
    expect(source).not.toContain('RunStatusChip')
  })

  it('WorkspaceView.vue has no <Teleport> for a running-agents indicator', () => {
    const viewPath = path.resolve(
      __dirname,
      '../../web/src/views/project/WorkspaceView.vue',
    )
    const source = fs.readFileSync(viewPath, 'utf-8')
    // A Teleport used for the old pill would reference run-status or similar
    if (source.includes('Teleport')) {
      expect(source).not.toMatch(/Teleport[^>]*>[\s\S]*?run.*status/i)
    } else {
      // No Teleport at all — clearly satisfied
      expect(source).not.toContain('Teleport')
    }
  })

  it('no file in web/src imports RunStatusChip', () => {
    // Walk web/src recursively and check every .vue/.ts file
    function collectFiles(dir: string): string[] {
      const results: string[] = []
      for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
        const full = path.join(dir, entry.name)
        if (entry.isDirectory()) {
          results.push(...collectFiles(full))
        } else if (/\.(vue|ts)$/.test(entry.name)) {
          results.push(full)
        }
      }
      return results
    }

    const srcRoot = path.resolve(__dirname, '../../web/src')
    const files = collectFiles(srcRoot)
    for (const file of files) {
      const content = fs.readFileSync(file, 'utf-8')
      expect(content, `${file} should not import RunStatusChip`).not.toContain('RunStatusChip')
    }
  })
})
