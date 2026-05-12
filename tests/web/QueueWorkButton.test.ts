// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Unit tests for QueueWorkButton (src/components/artifact/QueueWorkButton.vue).
 *
 * Covers Suite 3.1 scenarios FB1–FB6:
 *   FB1 renders when artifact is approved and agent matches
 *   FB2 hides when status is not approved
 *   FB3 hides when no agent matches the type
 *   FB4 defect falls back to assignee role
 *   FB5 click calls queueStore.enqueue
 *   FB6 replaces button with "Queued — position N" badge
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { ref } from 'vue'
import QueueWorkButton from '../../web/src/components/artifact/QueueWorkButton.vue'
import type { ArtifactDetail } from '../../web/src/types/api'
import type { QueueSnapshot } from '../../web/src/api/queue'

// ---------------------------------------------------------------------------
// Reactive store state shared across mocks
// ---------------------------------------------------------------------------

const _snapshotRef = ref<QueueSnapshot>({
  running: null,
  pending: [],
  recent: [],
  paused: false,
  paused_until: null,
  pause_reason: null,
})

const _enqueueMock = vi.fn().mockResolvedValue(undefined)

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

vi.mock('@/api/ws', () => ({
  getAppWs: vi.fn(() => ({
    on: vi.fn(() => () => {}),
  })),
}))

vi.mock('@/api/queue', () => ({
  listQueue: vi.fn(),
  enqueue: vi.fn().mockResolvedValue(undefined),
  cancelQueue: vi.fn(),
  pauseQueue: vi.fn(),
  resumeQueue: vi.fn(),
}))

vi.mock('@/stores/queue', () => ({
  useQueueStore: () => ({
    get snapshot() { return _snapshotRef.value },
    enqueue: _enqueueMock,
  }),
}))

vi.mock('@/stores/agents', () => ({
  useAgentsStore: () => ({
    agents: [
      { name: 'requirements-analyst', roles: ['analyst'] },
      { name: 'backend-developer', roles: ['backend-developer'] },
    ],
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    me: { email: 'admin@test.local', roles: { testproject: ['product-owner'] } },
    isAuthenticated: true,
    rolesForProject: (_project: string) => ['product-owner'],
  }),
}))

vi.mock('@/stores/ui', () => ({
  useUiStore: () => ({
    success: vi.fn(),
    error: vi.fn(),
  }),
}))

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeArtifact(overrides: Partial<ArtifactDetail['frontmatter']> & { path?: string } = {}): ArtifactDetail {
  const { path: artifactPath = 'lifecycle/ideas/test.md', ...fm } = overrides
  return {
    path: artifactPath,
    status: fm.status ?? 'approved',
    frontmatter: {
      title: 'Test Artifact',
      type: fm.type ?? 'idea',
      status: fm.status ?? 'approved',
      lineage: 'test',
      ...fm,
    },
    body: 'Body.',
    last_commit: null,
    history: [],
    run_history: [],
    children: [],
    is_locked: false,
    lock: null,
    parse_error: null,
  } as unknown as ArtifactDetail
}

function makeRouter() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div/>' } }],
  })
  router.push('/')
  return router
}

async function mountButton(artifact: ArtifactDetail, project = 'testproject') {
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = makeRouter()
  await router.isReady()

  const wrapper = mount(QueueWorkButton, {
    props: { artifact, project },
    global: { plugins: [pinia, router] },
  })
  await flushPromises()
  return wrapper
}

// ---------------------------------------------------------------------------
// Teardown
// ---------------------------------------------------------------------------

beforeEach(() => {
  _snapshotRef.value = {
    running: null,
    pending: [],
    recent: [],
    paused: false,
    paused_until: null,
    pause_reason: null,
  }
  _enqueueMock.mockClear()
})

afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

// ---------------------------------------------------------------------------
// FB1 — renders when artifact is approved and agent matches
// ---------------------------------------------------------------------------

describe('FB1: renders when artifact is approved and agent matches', () => {
  it('shows "Queue Work" button for an approved idea', async () => {
    const wrapper = await mountButton(makeArtifact({ type: 'idea', status: 'approved' }))
    expect(wrapper.find('.btn-queue').exists()).toBe(true)
    expect(wrapper.find('.btn-queue').text()).toBe('Queue Work')
  })
})

// ---------------------------------------------------------------------------
// FB2 — hides when status is not approved
// ---------------------------------------------------------------------------

describe('FB2: hides when status is not approved', () => {
  it('does not render button when status is draft', async () => {
    const wrapper = await mountButton(makeArtifact({ type: 'idea', status: 'draft' }))
    expect(wrapper.find('.btn-queue').exists()).toBe(false)
  })

  it('does not render button when status is in-development', async () => {
    const wrapper = await mountButton(makeArtifact({ type: 'idea', status: 'in-development' }))
    expect(wrapper.find('.btn-queue').exists()).toBe(false)
  })
})

// ---------------------------------------------------------------------------
// FB3 — hides when no agent matches the type
// ---------------------------------------------------------------------------

describe('FB3: hides when no agent matches the type', () => {
  it('does not render button for a release artifact (no agent mapped)', async () => {
    const wrapper = await mountButton(makeArtifact({ type: 'release', status: 'approved' }))
    expect(wrapper.find('.btn-queue').exists()).toBe(false)
    expect(wrapper.find('.queued-badge').exists()).toBe(false)
  })
})

// ---------------------------------------------------------------------------
// FB4 — defect falls back to assignee role
// ---------------------------------------------------------------------------

describe('FB4: defect falls back to assignee role', () => {
  it('shows button for a defect with an assignee matching backend-developer role', async () => {
    const artifact = makeArtifact({
      type: 'defect',
      status: 'approved',
      assignees: [{ role: 'backend-developer' }],
    } as unknown as Partial<ArtifactDetail['frontmatter']> & { path?: string })
    const wrapper = await mountButton(artifact)
    // The button or queued badge should be present since backend-developer agent exists
    const hasButton = wrapper.find('.btn-queue').exists()
    const hasBadge = wrapper.find('.queued-badge').exists()
    expect(hasButton || hasBadge).toBe(true)
  })

  it('does not render for a defect with no assignees', async () => {
    const artifact = makeArtifact({
      type: 'defect',
      status: 'approved',
      assignees: [],
    } as unknown as Partial<ArtifactDetail['frontmatter']> & { path?: string })
    const wrapper = await mountButton(artifact)
    expect(wrapper.find('.btn-queue').exists()).toBe(false)
    expect(wrapper.find('.queued-badge').exists()).toBe(false)
  })
})

// ---------------------------------------------------------------------------
// FB5 — click calls queueStore.enqueue
// ---------------------------------------------------------------------------

describe('FB5: click calls queueStore.enqueue', () => {
  it('calls enqueue with correct args when button is clicked', async () => {
    const artifact = makeArtifact({ type: 'idea', status: 'approved' })
    const wrapper = await mountButton(artifact, 'testproject')

    const btn = wrapper.find('.btn-queue')
    expect(btn.exists()).toBe(true)

    await btn.trigger('click')
    await flushPromises()

    expect(_enqueueMock).toHaveBeenCalledOnce()
    expect(_enqueueMock).toHaveBeenCalledWith({
      project: 'testproject',
      artifact_path: 'lifecycle/ideas/test.md',
      agent: 'requirements-analyst',
    })
  })
})

// ---------------------------------------------------------------------------
// FB6 — replaces button with "Queued — position N" badge
// ---------------------------------------------------------------------------

describe('FB6: replaces button with "Queued — position N" badge', () => {
  it('shows queued badge when artifact is in pending list at position 2', async () => {
    const artifact = makeArtifact({ type: 'idea', status: 'approved' })
    _snapshotRef.value = {
      running: null,
      pending: [
        {
          id: 'job-1',
          project: 'testproject',
          artifact_path: 'lifecycle/ideas/test.md',
          agent: 'requirements-analyst',
          state: 'pending',
          attempts: 1,
          enqueued_at: 1700000000,
          position: 2,
          enqueued_by: 'admin@test.local',
        },
      ],
      recent: [],
      paused: false,
      paused_until: null,
      pause_reason: null,
    }

    const wrapper = await mountButton(artifact, 'testproject')

    expect(wrapper.find('.btn-queue').exists()).toBe(false)
    const badge = wrapper.find('.queued-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toContain('Queued')
    expect(badge.text()).toContain('2')
  })

  it('shows "Running…" badge when artifact is the running job', async () => {
    const artifact = makeArtifact({ type: 'idea', status: 'approved' })
    _snapshotRef.value = {
      running: {
        id: 'job-running',
        project: 'testproject',
        artifact_path: 'lifecycle/ideas/test.md',
        agent: 'requirements-analyst',
        state: 'running',
        attempts: 1,
        enqueued_at: 1700000000,
        position: 1,
        enqueued_by: 'admin@test.local',
      },
      pending: [],
      recent: [],
      paused: false,
      paused_until: null,
      pause_reason: null,
    }

    const wrapper = await mountButton(artifact, 'testproject')

    expect(wrapper.find('.btn-queue').exists()).toBe(false)
    const badge = wrapper.find('.queued-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toContain('Running')
  })
})
