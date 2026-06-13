// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Unit tests for the queue Pinia store (src/stores/queue.ts).
 *
 * Covers Suite 3.3 scenarios FS1–FS7:
 *   FS1 queue.added pushes to pending
 *   FS2 queue.started moves to running
 *   FS3 queue.finished moves to recent (capped at 10)
 *   FS4 queue.paused sets paused state
 *   FS5 queue.resumed clears paused state
 *   FS6 queue.cancelled removes from pending
 *   FS7 initial fetch sets full snapshot
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { useQueueStore } from '../../web/src/stores/queue'
import type { QueueSnapshot, QueueJob } from '../../web/src/api/queue'

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

// Track the registered WS event handler so tests can call it directly.
let _wsHandler: ((e: { type: string; payload: unknown }) => void) | null = null

vi.mock('@/api/ws', () => ({
  getAppWs: vi.fn(() => ({
    on: vi.fn((handler: (e: { type: string; payload: unknown }) => void) => {
      _wsHandler = handler
      return () => {}
    }),
  })),
}))

vi.mock('@/api/queue', () => ({
  listQueue: vi.fn(),
  enqueue: vi.fn(),
  cancelQueue: vi.fn(),
  pauseQueue: vi.fn(),
  resumeQueue: vi.fn(),
}))

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeJob(overrides: Partial<QueueJob> = {}): QueueJob {
  return {
    id: 'job-1',
    project: 'testproject',
    artifact_path: 'lifecycle/ideas/test-1.md',
    agent: 'requirements-analyst',
    state: 'pending',
    attempts: 1,
    enqueued_at: 1700000000,
    position: 1,
    enqueued_by: 'admin@test.local',
    ...overrides,
  }
}

function makeSnapshot(overrides: Partial<QueueSnapshot> = {}): QueueSnapshot {
  return {
    running: null,
    pending: [],
    recent: [],
    paused: false,
    paused_until: null,
    pause_reason: null,
    ...overrides,
  }
}

/** Dispatch a WS event to the store's registered handler. */
function dispatchWS(type: string, payload: unknown) {
  if (!_wsHandler) throw new Error('WS handler not registered — call store.fetch() first')
  _wsHandler({ type, payload })
}

// ---------------------------------------------------------------------------
// Setup / teardown
// ---------------------------------------------------------------------------

beforeEach(() => {
  _wsHandler = null
  const pinia = createPinia()
  setActivePinia(pinia)
})

afterEach(() => {
  vi.clearAllMocks()
})

// ---------------------------------------------------------------------------
// FS1 — queue.added pushes to pending
// ---------------------------------------------------------------------------

describe('FS1: queue.added pushes to pending', () => {
  it('adds a new job to pending and sorts by position', async () => {
    const { listQueue } = await import('@/api/queue')
    vi.mocked(listQueue).mockResolvedValue(makeSnapshot())

    const store = useQueueStore()
    await store.fetch()

    const job = makeJob({ id: 'new-job', position: 1 })
    dispatchWS('queue.added', job)
    await nextTick()

    expect(store.snapshot.pending).toHaveLength(1)
    expect(store.snapshot.pending[0].id).toBe('new-job')
  })

  it('maintains position-sorted order when multiple jobs are added', async () => {
    const { listQueue } = await import('@/api/queue')
    vi.mocked(listQueue).mockResolvedValue(makeSnapshot())

    const store = useQueueStore()
    await store.fetch()

    dispatchWS('queue.added', makeJob({ id: 'job-3', position: 3 }))
    dispatchWS('queue.added', makeJob({ id: 'job-1', position: 1 }))
    dispatchWS('queue.added', makeJob({ id: 'job-2', position: 2 }))
    await nextTick()

    expect(store.snapshot.pending.map((j) => j.id)).toEqual(['job-1', 'job-2', 'job-3'])
  })

  it('does not add a duplicate job (same id)', async () => {
    const { listQueue } = await import('@/api/queue')
    vi.mocked(listQueue).mockResolvedValue(makeSnapshot({ pending: [makeJob({ id: 'dup' })] }))

    const store = useQueueStore()
    await store.fetch()

    dispatchWS('queue.added', makeJob({ id: 'dup', position: 1 }))
    await nextTick()

    expect(store.snapshot.pending).toHaveLength(1)
  })
})

// ---------------------------------------------------------------------------
// FS2 — queue.started moves to running
// ---------------------------------------------------------------------------

describe('FS2: queue.started moves to running', () => {
  it('moves a pending job to running', async () => {
    const { listQueue } = await import('@/api/queue')
    vi.mocked(listQueue).mockResolvedValue(makeSnapshot({
      pending: [makeJob({ id: 'job-a', state: 'pending' })],
    }))

    const store = useQueueStore()
    await store.fetch()

    dispatchWS('queue.started', { id: 'job-a', status: 'running' })
    await nextTick()

    expect(store.snapshot.pending).toHaveLength(0)
    expect(store.snapshot.running).not.toBeNull()
    expect(store.snapshot.running?.id).toBe('job-a')
    expect(store.snapshot.running?.state).toBe('running')
  })
})

// ---------------------------------------------------------------------------
// FS3 — queue.finished moves to recent (capped at 10)
// ---------------------------------------------------------------------------

describe('FS3: queue.finished moves to recent (capped at 10)', () => {
  it('moves a running job to recent on queue.finished', async () => {
    const runningJob = makeJob({ id: 'run-job', state: 'running' })
    const { listQueue } = await import('@/api/queue')
    vi.mocked(listQueue).mockResolvedValue(makeSnapshot({ running: runningJob }))

    const store = useQueueStore()
    await store.fetch()

    // The queue.finished handler fires a silent server refresh. Under Vitest 4
    // that refresh resolves within `await nextTick()` (it didn't under Vitest 1)
    // and would re-apply the now-stale fetch mock, clobbering the optimistic
    // update. Reject it so this test verifies the WS-handler update in isolation.
    vi.mocked(listQueue).mockRejectedValue(new Error('refresh skipped'))

    dispatchWS('queue.finished', { id: 'run-job', status: 'done' })
    await nextTick()

    expect(store.snapshot.running).toBeNull()
    expect(store.snapshot.recent).toHaveLength(1)
    expect(store.snapshot.recent[0].id).toBe('run-job')
  })

  it('caps recent list at 10 entries', async () => {
    // Seed 10 existing recent entries.
    const existing = Array.from({ length: 10 }, (_, i) =>
      makeJob({ id: `old-${i}`, state: 'completed' }),
    )
    const runningJob = makeJob({ id: 'new-job', state: 'running' })
    const { listQueue } = await import('@/api/queue')
    vi.mocked(listQueue).mockResolvedValue(makeSnapshot({ running: runningJob, recent: existing }))

    const store = useQueueStore()
    await store.fetch()

    // Reject the post-finish silent refresh (see the sibling test): under
    // Vitest 4 it would otherwise resolve within nextTick and re-apply the stale
    // mock snapshot, clobbering the optimistic cap-to-10 update.
    vi.mocked(listQueue).mockRejectedValue(new Error('refresh skipped'))

    dispatchWS('queue.finished', { id: 'new-job', status: 'done' })
    await nextTick()

    expect(store.snapshot.recent).toHaveLength(10)
    // newest entry should be first
    expect(store.snapshot.recent[0].id).toBe('new-job')
  })
})

// ---------------------------------------------------------------------------
// FS4 — queue.paused sets paused state
// ---------------------------------------------------------------------------

describe('FS4: queue.paused sets paused state', () => {
  it('sets paused=true and records paused_until from the event', async () => {
    const { listQueue } = await import('@/api/queue')
    vi.mocked(listQueue).mockResolvedValue(makeSnapshot())

    const store = useQueueStore()
    await store.fetch()

    const until = '2026-05-12T20:05:00+10:00'
    dispatchWS('queue.paused', { paused_until: until, pause_reason: 'rate_limit' })
    await nextTick()

    expect(store.snapshot.paused).toBe(true)
    expect(store.snapshot.paused_until).toBe(until)
  })
})

// ---------------------------------------------------------------------------
// FS5 — queue.resumed clears paused state
// ---------------------------------------------------------------------------

describe('FS5: queue.resumed clears paused state', () => {
  it('sets paused=false and clears paused_until', async () => {
    const { listQueue } = await import('@/api/queue')
    vi.mocked(listQueue).mockResolvedValue(
      makeSnapshot({ paused: true, paused_until: '2026-05-12T20:00:00+10:00' }),
    )

    const store = useQueueStore()
    await store.fetch()

    dispatchWS('queue.resumed', {})
    await nextTick()

    expect(store.snapshot.paused).toBe(false)
    expect(store.snapshot.paused_until).toBeNull()
  })
})

// ---------------------------------------------------------------------------
// FS6 — queue.cancelled removes from pending
// ---------------------------------------------------------------------------

describe('FS6: queue.cancelled removes from pending', () => {
  it('removes a pending job by id on queue.cancelled', async () => {
    const { listQueue } = await import('@/api/queue')
    vi.mocked(listQueue).mockResolvedValue(makeSnapshot({
      pending: [
        makeJob({ id: 'keep-me' }),
        makeJob({ id: 'remove-me', artifact_path: 'lifecycle/ideas/b.md' }),
      ],
    }))

    const store = useQueueStore()
    await store.fetch()

    dispatchWS('queue.cancelled', { id: 'remove-me' })
    await nextTick()

    expect(store.snapshot.pending).toHaveLength(1)
    expect(store.snapshot.pending[0].id).toBe('keep-me')
  })
})

// ---------------------------------------------------------------------------
// FS7 — initial fetch sets full snapshot
// ---------------------------------------------------------------------------

describe('FS7: initial fetch sets full snapshot', () => {
  it('populates state from REST response', async () => {
    const runningJob = makeJob({ id: 'run-1', state: 'running' })
    const pendingJob = makeJob({ id: 'pend-1', state: 'pending' })
    const recentJob = makeJob({ id: 'rec-1', state: 'completed' })

    const { listQueue } = await import('@/api/queue')
    vi.mocked(listQueue).mockResolvedValue(makeSnapshot({
      running: runningJob,
      pending: [pendingJob],
      recent: [recentJob],
      paused: true,
      paused_until: '2026-05-12T20:00:00+10:00',
    }))

    const store = useQueueStore()
    await store.fetch()

    expect(store.snapshot.running?.id).toBe('run-1')
    expect(store.snapshot.pending).toHaveLength(1)
    expect(store.snapshot.recent).toHaveLength(1)
    expect(store.snapshot.paused).toBe(true)
    expect(store.snapshot.paused_until).toBe('2026-05-12T20:00:00+10:00')
    expect(store.loading).toBe(false)
    expect(store.error).toBeNull()
  })

  it('sets error on fetch failure', async () => {
    const { listQueue } = await import('@/api/queue')
    vi.mocked(listQueue).mockRejectedValue(new Error('Network error'))

    const store = useQueueStore()
    await store.fetch()

    expect(store.error).toContain('Network error')
    expect(store.loading).toBe(false)
  })
})
