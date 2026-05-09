// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 1 — Unit tests for the `useExternalChange` composable
 *
 * Covers the auto-refresh path (clean editor), conflict banner path (dirty
 * editor), debounce coalescing, save-grace suppression, path filtering,
 * backward-compatibility, and cleanup on unmount.
 *
 * Uses vi.useFakeTimers() for deterministic debounce testing.
 * Mocks `getProjectWs` so no real WebSocket server is required.
 *
 * Composable location: web/src/composables/useExternalChange.ts
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { defineComponent, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { useExternalChange } from '@/composables/useExternalChange'

// ---------------------------------------------------------------------------
// Hoisted mocks — the WsClient stub must be visible inside vi.mock() factory
// ---------------------------------------------------------------------------

const mocks = vi.hoisted(() => {
  // Per-test handler registrations keyed by event type.
  const handlers = new Map<string, Array<(e: { type: string; payload: Record<string, unknown> }) => void>>()

  const wsClient = {
    handlers,
    // Simulate ws.onType(type, handler) — stores handler, returns unsub fn.
    onType: vi.fn((type: string, handler: (e: { type: string; payload: Record<string, unknown> }) => void) => {
      if (!handlers.has(type)) handlers.set(type, [])
      handlers.get(type)!.push(handler)
      return () => {
        const arr = handlers.get(type) ?? []
        const idx = arr.indexOf(handler)
        if (idx !== -1) arr.splice(idx, 1)
      }
    }),
  }

  return {
    wsClient,
    getProjectWs: vi.fn(() => wsClient),
  }
})

vi.mock('@/api/ws', () => ({
  getProjectWs: mocks.getProjectWs,
}))

// ---------------------------------------------------------------------------
// Helper — emit a file.changed event to all registered handlers
// ---------------------------------------------------------------------------

function emitFileChanged(path: string) {
  const handlers = mocks.wsClient.handlers.get('file.changed') ?? []
  const event = { type: 'file.changed', payload: { path } }
  handlers.forEach((h) => h(event))
}

// ---------------------------------------------------------------------------
// Mount helper — runs the composable inside a component's setup() so that
// Vue's onUnmounted lifecycle hook works correctly.
// ---------------------------------------------------------------------------

interface ComposableOptions {
  isDirty?: () => boolean
  onAutoRefresh?: () => void
}

function setupComposable(
  artifactPath: string,
  options?: ComposableOptions,
) {
  const pinia = createPinia()
  setActivePinia(pinia)

  let result!: ReturnType<typeof import('../../web/src/composables/useExternalChange').useExternalChange>

  const wrapper = mount(
    defineComponent({
      setup() {
        result = useExternalChange('testproject', artifactPath, options)
        return {}
      },
      template: '<div/>',
    }),
    { global: { plugins: [pinia] } },
  )

  return { result, wrapper }
}

// ---------------------------------------------------------------------------
// Setup / teardown
// ---------------------------------------------------------------------------

beforeEach(() => {
  // Reset handler registrations between tests.
  mocks.wsClient.handlers.clear()
  mocks.wsClient.onType.mockClear()
  mocks.getProjectWs.mockClear()
  vi.useFakeTimers()
  setActivePinia(createPinia())
})

afterEach(() => {
  vi.useRealTimers()
  vi.clearAllMocks()
})

// ---------------------------------------------------------------------------
// Test 1 — Auto-refresh fires when not dirty
// ---------------------------------------------------------------------------

describe('useExternalChange — auto-refresh fires when not dirty', () => {
  it('calls onAutoRefresh after 300 ms when isDirty returns false', async () => {
    const onAutoRefresh = vi.fn()
    const { result } = setupComposable('lifecycle/ideas/test.md', {
      isDirty: () => false,
      onAutoRefresh,
    })

    emitFileChanged('lifecycle/ideas/test.md')
    // Before debounce fires, callback must not have been called yet.
    expect(onAutoRefresh).not.toHaveBeenCalled()

    vi.advanceTimersByTime(300)
    await nextTick()

    expect(onAutoRefresh).toHaveBeenCalledOnce()
    // hasExternalChange must remain false in auto-refresh mode.
    expect(result.hasExternalChange.value).toBe(false)
  })
})

// ---------------------------------------------------------------------------
// Test 2 — Conflict banner when dirty
// ---------------------------------------------------------------------------

describe('useExternalChange — conflict banner when dirty', () => {
  it('sets hasExternalChange to true and never calls onAutoRefresh when isDirty returns true', async () => {
    const onAutoRefresh = vi.fn()
    const { result } = setupComposable('lifecycle/ideas/dirty.md', {
      isDirty: () => true,
      onAutoRefresh,
    })

    emitFileChanged('lifecycle/ideas/dirty.md')
    vi.advanceTimersByTime(1000)
    await nextTick()

    expect(result.hasExternalChange.value).toBe(true)
    expect(onAutoRefresh).not.toHaveBeenCalled()
  })
})

// ---------------------------------------------------------------------------
// Test 3 — Debounce coalesces rapid events
// ---------------------------------------------------------------------------

describe('useExternalChange — debounce coalesces rapid events', () => {
  it('calls onAutoRefresh exactly once after three events within 100 ms intervals', async () => {
    const onAutoRefresh = vi.fn()
    setupComposable('lifecycle/ideas/rapid.md', {
      isDirty: () => false,
      onAutoRefresh,
    })

    emitFileChanged('lifecycle/ideas/rapid.md')
    vi.advanceTimersByTime(100)
    emitFileChanged('lifecycle/ideas/rapid.md')
    vi.advanceTimersByTime(100)
    emitFileChanged('lifecycle/ideas/rapid.md')

    // Advance less than 300 ms — callback must not have fired yet.
    vi.advanceTimersByTime(299)
    expect(onAutoRefresh).not.toHaveBeenCalled()

    // Complete the last debounce window.
    vi.advanceTimersByTime(1)
    await nextTick()
    expect(onAutoRefresh).toHaveBeenCalledOnce()
  })
})

// ---------------------------------------------------------------------------
// Test 4 — Save-grace suppresses auto-refresh (clean editor)
// ---------------------------------------------------------------------------

describe('useExternalChange — save-grace suppresses auto-refresh', () => {
  it('does not call onAutoRefresh when file.changed arrives within SAVE_GRACE_MS after markSaved()', async () => {
    const onAutoRefresh = vi.fn()
    const { result } = setupComposable('lifecycle/ideas/saved.md', {
      isDirty: () => false,
      onAutoRefresh,
    })

    result.markSaved()
    emitFileChanged('lifecycle/ideas/saved.md')
    vi.advanceTimersByTime(1000)
    await nextTick()

    expect(onAutoRefresh).not.toHaveBeenCalled()
    expect(result.hasExternalChange.value).toBe(false)
  })
})

// ---------------------------------------------------------------------------
// Test 5 — Save-grace suppresses conflict banner (dirty editor)
// ---------------------------------------------------------------------------

describe('useExternalChange — save-grace suppresses conflict banner', () => {
  it('does not set hasExternalChange when dirty editor receives file.changed within SAVE_GRACE_MS', async () => {
    const onAutoRefresh = vi.fn()
    const { result } = setupComposable('lifecycle/ideas/saved-dirty.md', {
      isDirty: () => true,
      onAutoRefresh,
    })

    result.markSaved()
    emitFileChanged('lifecycle/ideas/saved-dirty.md')
    vi.advanceTimersByTime(1000)
    await nextTick()

    expect(result.hasExternalChange.value).toBe(false)
    expect(onAutoRefresh).not.toHaveBeenCalled()
  })
})

// ---------------------------------------------------------------------------
// Test 6 — Events for other paths are ignored
// ---------------------------------------------------------------------------

describe('useExternalChange — events for other paths are ignored', () => {
  it('does not call onAutoRefresh when file.changed is for a different path', async () => {
    const onAutoRefresh = vi.fn()
    const { result } = setupComposable('lifecycle/ideas/target.md', {
      isDirty: () => false,
      onAutoRefresh,
    })

    emitFileChanged('lifecycle/ideas/other.md')
    vi.advanceTimersByTime(1000)
    await nextTick()

    expect(onAutoRefresh).not.toHaveBeenCalled()
    expect(result.hasExternalChange.value).toBe(false)
  })
})

// ---------------------------------------------------------------------------
// Test 7 — Backward compatibility (no options)
// ---------------------------------------------------------------------------

describe('useExternalChange — backward compatibility without options', () => {
  it('sets hasExternalChange to true when no options are provided (original behaviour)', async () => {
    // Instantiate without any options — the composable should fall back to the
    // conflict-banner path since there is no onAutoRefresh callback.
    const { result } = setupComposable('lifecycle/ideas/compat.md')

    emitFileChanged('lifecycle/ideas/compat.md')
    vi.advanceTimersByTime(1000)
    await nextTick()

    expect(result.hasExternalChange.value).toBe(true)
  })
})

// ---------------------------------------------------------------------------
// Test 8 — Cleanup on unmount
// ---------------------------------------------------------------------------

describe('useExternalChange — cleanup on unmount', () => {
  it('does not invoke onAutoRefresh after the component is unmounted', async () => {
    const onAutoRefresh = vi.fn()
    const { wrapper } = setupComposable('lifecycle/ideas/cleanup.md', {
      isDirty: () => false,
      onAutoRefresh,
    })

    // Emit event, then unmount before the debounce timer fires.
    emitFileChanged('lifecycle/ideas/cleanup.md')
    wrapper.unmount()

    // Advance past the 300 ms debounce window — timer should have been cleared.
    vi.advanceTimersByTime(500)
    await nextTick()

    expect(onAutoRefresh).not.toHaveBeenCalled()
  })
})
