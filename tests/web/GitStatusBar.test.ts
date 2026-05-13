// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Component tests for GitStatusBar.
 *
 * Covers test-plan milestone M3 scenarios:
 *   M3-TC1  renders branch name, SHA, commit message, and GitBranch icon (clean tree)
 *   M3-TC2  dirty indicator is shown when dirty=true, absent when dirty=false
 *   M3-TC3  component renders nothing when available=false
 *   M3-TC4  collapsed mode: branch label hidden, icon visible
 *   M3-TC5  WebSocket update reactivity: store mutation re-renders without remount
 *   M3-TC6  accessibility: role="status", aria-label on root and dirty indicator
 *
 * Approach
 * ────────
 * The Pinia gitStatus store is mocked directly with a reactive ref so tests can
 * control state without involving real API calls or WebSocket connections.
 * @/api/ws is mocked to prevent real WS connections from the useWebSocket composable.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { ref, nextTick } from 'vue'
import GitStatusBar from '../../web/src/components/layout/GitStatusBar.vue'

// ---------------------------------------------------------------------------
// Reactive mock state — shared across all test cases
// ---------------------------------------------------------------------------

const _git = ref({
  available: false,
  branch: '',
  dirty: false,
  headSha: '',
  headMessage: '',
  headAuthor: '',
  headWhen: '',
})

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

vi.mock('@/stores/gitStatus', () => ({
  useGitStatusStore: () => ({
    get available()    { return _git.value.available },
    get branch()      { return _git.value.branch },
    get dirty()       { return _git.value.dirty },
    get headSha()     { return _git.value.headSha },
    get headMessage() { return _git.value.headMessage },
    get headAuthor()  { return _git.value.headAuthor },
    get headWhen()    { return _git.value.headWhen },
    fetch: vi.fn().mockResolvedValue(undefined),
    applyWsEvent: vi.fn(),
    reset: vi.fn(),
  }),
}))

vi.mock('@/api/ws', () => ({
  getProjectWs: vi.fn(() => ({
    onType: vi.fn(() => () => {}),
  })),
}))

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function defaultGitState() {
  return {
    available: false,
    branch: '',
    dirty: false,
    headSha: '',
    headMessage: '',
    headAuthor: '',
    headWhen: '',
  }
}

async function mountBar(
  props: { project?: string; collapsed?: boolean } = {},
): Promise<ReturnType<typeof mount>> {
  const pinia = createPinia()
  setActivePinia(pinia)
  const wrapper = mount(GitStatusBar, {
    props: {
      project:   props.project   ?? 'testproject',
      collapsed: props.collapsed ?? false,
    },
    global: { plugins: [pinia] },
    attachTo: document.body,
  })
  await flushPromises()
  return wrapper
}

// ---------------------------------------------------------------------------
// Reset
// ---------------------------------------------------------------------------

beforeEach(() => {
  _git.value = defaultGitState()
})

afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

// ===========================================================================
// M3-TC1 — renders branch and commit info
// ===========================================================================

describe('M3-TC1: renders branch name, SHA, message, and icon when available', () => {
  it('renders the branch name in the branch-name span', async () => {
    _git.value = {
      available: true,
      branch: 'feature/login',
      dirty: false,
      headSha: 'a1b2c3d',
      headMessage: 'fix login redirect',
      headAuthor: 'Test User',
      headWhen: '2026-05-01T10:00:00Z',
    }

    const wrapper = await mountBar()
    expect(wrapper.find('.git-branch-name').text()).toBe('feature/login')
  })

  it('renders a GitBranch SVG icon', async () => {
    _git.value = { ...defaultGitState(), available: true, branch: 'main' }

    const wrapper = await mountBar()
    expect(wrapper.find('.git-branch-row svg').exists()).toBe(true)
  })

  it('renders the first 7 characters of the SHA', async () => {
    _git.value = {
      available: true,
      branch: 'main',
      dirty: false,
      headSha: 'a1b2c3d',
      headMessage: 'fix login redirect',
      headAuthor: 'Test User',
      headWhen: '2026-05-01T10:00:00Z',
    }

    const wrapper = await mountBar()
    expect(wrapper.find('.git-sha').text()).toBe('a1b2c3d')
  })

  it('renders the commit message', async () => {
    _git.value = {
      available: true,
      branch: 'main',
      dirty: false,
      headSha: 'a1b2c3d',
      headMessage: 'fix login redirect',
      headAuthor: 'Test User',
      headWhen: '2026-05-01T10:00:00Z',
    }

    const wrapper = await mountBar()
    expect(wrapper.find('.git-commit-msg').text()).toBe('fix login redirect')
  })

  it('renders only the first line of a multi-line commit message', async () => {
    _git.value = {
      available: true,
      branch: 'main',
      dirty: false,
      headSha: 'a1b2c3d',
      headMessage: 'feat: add login\n\nCo-Authored-By: kaos-control',
      headAuthor: 'Test User',
      headWhen: '2026-05-01T10:00:00Z',
    }

    const wrapper = await mountBar()
    expect(wrapper.find('.git-commit-msg').text()).toBe('feat: add login')
  })

  it('shows the "clean" dirty indicator when dirty=false', async () => {
    _git.value = { ...defaultGitState(), available: true, branch: 'main' }

    const wrapper = await mountBar()
    const indicator = wrapper.find('.git-dirty-indicator')
    expect(indicator.exists()).toBe(true)
    expect(indicator.text()).toBe('clean')
  })
})

// ===========================================================================
// M3-TC2 — dirty indicator
// ===========================================================================

describe('M3-TC2: dirty indicator shown when dirty=true', () => {
  it('renders the "modified" text when dirty=true', async () => {
    _git.value = {
      available: true,
      branch: 'main',
      dirty: true,
      headSha: 'a1b2c3d',
      headMessage: 'fix login redirect',
      headAuthor: 'Test User',
      headWhen: '2026-05-01T10:00:00Z',
    }

    const wrapper = await mountBar()
    const indicator = wrapper.find('.git-dirty-indicator')
    expect(indicator.exists()).toBe(true)
    expect(indicator.text()).toBe('modified')
  })

  it('applies the --dirty modifier class when dirty=true', async () => {
    _git.value = { ...defaultGitState(), available: true, branch: 'main', dirty: true }

    const wrapper = await mountBar()
    expect(wrapper.find('.git-dirty-indicator--dirty').exists()).toBe(true)
    expect(wrapper.find('.git-dirty-indicator--clean').exists()).toBe(false)
  })

  it('dirty indicator aria-label describes uncommitted changes when dirty=true', async () => {
    _git.value = { ...defaultGitState(), available: true, branch: 'main', dirty: true }

    const wrapper = await mountBar()
    const indicator = wrapper.find('.git-dirty-indicator')
    expect(indicator.attributes('aria-label')).toBe('Working tree has uncommitted changes')
  })

  it('dirty indicator aria-label describes clean state when dirty=false', async () => {
    _git.value = { ...defaultGitState(), available: true, branch: 'main', dirty: false }

    const wrapper = await mountBar()
    const indicator = wrapper.find('.git-dirty-indicator')
    expect(indicator.attributes('aria-label')).toBe('Working tree is clean')
  })
})

// ===========================================================================
// M3-TC3 — hidden when unavailable
// ===========================================================================

describe('M3-TC3: component renders nothing when available=false', () => {
  it('does not render the git-status-bar element when available=false', async () => {
    // _git.value.available is false by default (set in beforeEach)
    const wrapper = await mountBar()
    expect(wrapper.find('.git-status-bar').exists()).toBe(false)
  })

  it('renders the git-status-bar when available switches to true', async () => {
    const wrapper = await mountBar()
    expect(wrapper.find('.git-status-bar').exists()).toBe(false)

    _git.value.available = true
    _git.value.branch = 'main'
    await nextTick()

    expect(wrapper.find('.git-status-bar').exists()).toBe(true)
  })
})

// ===========================================================================
// M3-TC4 — collapsed sidebar state
// ===========================================================================

describe('M3-TC4: collapsed mode hides branch label and shows icon only', () => {
  it('shows the icon wrapper instead of the expanded content when collapsed=true', async () => {
    _git.value = { ...defaultGitState(), available: true, branch: 'feature/login' }

    const wrapper = await mountBar({ collapsed: true })
    expect(wrapper.find('.git-icon-wrap').exists()).toBe(true)
    expect(wrapper.find('.git-branch-row').exists()).toBe(false)
  })

  it('branch name span is not rendered in collapsed mode', async () => {
    _git.value = { ...defaultGitState(), available: true, branch: 'feature/login' }

    const wrapper = await mountBar({ collapsed: true })
    expect(wrapper.find('.git-branch-name').exists()).toBe(false)
  })

  it('GitBranch icon is present in collapsed mode', async () => {
    _git.value = { ...defaultGitState(), available: true, branch: 'main' }

    const wrapper = await mountBar({ collapsed: true })
    expect(wrapper.find('.git-icon-wrap svg').exists()).toBe(true)
  })

  it('dirty dot has --dirty class when dirty=true in collapsed mode', async () => {
    _git.value = { ...defaultGitState(), available: true, branch: 'main', dirty: true }

    const wrapper = await mountBar({ collapsed: true })
    expect(wrapper.find('.git-dirty-dot--dirty').exists()).toBe(true)
    expect(wrapper.find('.git-dirty-dot--clean').exists()).toBe(false)
  })

  it('dirty dot has --clean class when dirty=false in collapsed mode', async () => {
    _git.value = { ...defaultGitState(), available: true, branch: 'main', dirty: false }

    const wrapper = await mountBar({ collapsed: true })
    expect(wrapper.find('.git-dirty-dot--clean').exists()).toBe(true)
    expect(wrapper.find('.git-dirty-dot--dirty').exists()).toBe(false)
  })

  it('shows expanded branch-row when collapsed=false', async () => {
    _git.value = { ...defaultGitState(), available: true, branch: 'main' }

    const wrapper = await mountBar({ collapsed: false })
    expect(wrapper.find('.git-branch-row').exists()).toBe(true)
    expect(wrapper.find('.git-icon-wrap').exists()).toBe(false)
  })
})

// ===========================================================================
// M3-TC5 — WebSocket update reactivity
// ===========================================================================

describe('M3-TC5: store mutation re-renders without remounting', () => {
  it('branch name updates when store branch changes', async () => {
    _git.value = { ...defaultGitState(), available: true, branch: 'main' }

    const wrapper = await mountBar()
    expect(wrapper.find('.git-branch-name').text()).toBe('main')

    _git.value.branch = 'develop'
    await nextTick()

    expect(wrapper.find('.git-branch-name').text()).toBe('develop')
  })

  it('dirty indicator updates when dirty flag changes', async () => {
    _git.value = { ...defaultGitState(), available: true, branch: 'main', dirty: false }

    const wrapper = await mountBar()
    expect(wrapper.find('.git-dirty-indicator').text()).toBe('clean')

    _git.value.dirty = true
    await nextTick()

    expect(wrapper.find('.git-dirty-indicator').text()).toBe('modified')
  })

  it('commit SHA updates when headSha changes', async () => {
    _git.value = { ...defaultGitState(), available: true, branch: 'main', headSha: 'abc1234' }

    const wrapper = await mountBar()
    expect(wrapper.find('.git-sha').text()).toBe('abc1234')

    _git.value.headSha = 'fed9876'
    await nextTick()

    expect(wrapper.find('.git-sha').text()).toBe('fed9876')
  })
})

// ===========================================================================
// M3-TC6 — accessibility
// ===========================================================================

describe('M3-TC6: accessibility attributes', () => {
  it('root element has role="status"', async () => {
    _git.value = { ...defaultGitState(), available: true, branch: 'main' }

    const wrapper = await mountBar()
    expect(wrapper.find('.git-status-bar').attributes('role')).toBe('status')
  })

  it('root element has aria-label "Git repository status"', async () => {
    _git.value = { ...defaultGitState(), available: true, branch: 'main' }

    const wrapper = await mountBar()
    expect(wrapper.find('.git-status-bar').attributes('aria-label')).toBe('Git repository status')
  })

  it('dirty indicator has aria-label when expanded', async () => {
    _git.value = { ...defaultGitState(), available: true, branch: 'main', dirty: true }

    const wrapper = await mountBar()
    const indicator = wrapper.find('.git-dirty-indicator')
    expect(indicator.attributes('aria-label')).toBeTruthy()
  })

  it('dirty dot in collapsed mode has aria-label when dirty=true', async () => {
    _git.value = { ...defaultGitState(), available: true, branch: 'main', dirty: true }

    const wrapper = await mountBar({ collapsed: true })
    const dot = wrapper.find('.git-dirty-dot')
    expect(dot.attributes('aria-label')).toBe('Working tree has uncommitted changes')
  })

  it('dirty dot in collapsed mode has aria-label when dirty=false', async () => {
    _git.value = { ...defaultGitState(), available: true, branch: 'main', dirty: false }

    const wrapper = await mountBar({ collapsed: true })
    const dot = wrapper.find('.git-dirty-dot')
    expect(dot.attributes('aria-label')).toBe('Working tree is clean')
  })
})
