// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import type { WsEvent } from '@/types/api'

// Frontend plan: lifecycle/frontend-plans/idea-archiving-4-fe.md — Milestone 4
//
// Verifies file.changed matching compares the full artifact path (not a bare
// filename), so nested/archived artifacts get the same conflict-banner /
// auto-refresh behaviour as flat ones.

let capturedHandler: ((e: WsEvent) => void) | null = null
const unsub = vi.fn()

vi.mock('@/api/ws', () => ({
  getProjectWs: () => ({
    onType: (_type: string, handler: (e: WsEvent) => void) => {
      capturedHandler = handler
      return unsub
    },
  }),
}))

import { useExternalChange } from '../useExternalChange'

function mountComposable(path: string, opts?: Parameters<typeof useExternalChange>[2]) {
  let result!: ReturnType<typeof useExternalChange>
  mount(defineComponent({
    setup() {
      result = useExternalChange('proj', path, opts)
      return () => h('div')
    },
  }))
  return result
}

describe('useExternalChange — nested artifact paths', () => {
  beforeEach(() => {
    capturedHandler = null
    unsub.mockClear()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('flags a conflict when a nested artifact changes externally while dirty', () => {
    const result = mountComposable('lifecycle/ideas/archive/foo.md', { isDirty: () => true })
    capturedHandler!({ type: 'file.changed', payload: { path: 'lifecycle/ideas/archive/foo.md' } })
    expect(result.hasExternalChange.value).toBe(true)
  })

  it('ignores file.changed events for an unrelated nested path', () => {
    const result = mountComposable('lifecycle/ideas/archive/foo.md', { isDirty: () => true })
    capturedHandler!({ type: 'file.changed', payload: { path: 'lifecycle/ideas/archive/bar.md' } })
    expect(result.hasExternalChange.value).toBe(false)
  })

  it('auto-refreshes a deeply nested artifact when not dirty', () => {
    vi.useFakeTimers()
    const onAutoRefresh = vi.fn()
    const result = mountComposable('lifecycle/ideas/2026/q3/foo.md', { isDirty: () => false, onAutoRefresh })
    capturedHandler!({ type: 'file.changed', payload: { path: 'lifecycle/ideas/2026/q3/foo.md' } })
    vi.advanceTimersByTime(300)
    expect(onAutoRefresh).toHaveBeenCalledTimes(1)
    expect(result.hasExternalChange.value).toBe(false)
  })
})
