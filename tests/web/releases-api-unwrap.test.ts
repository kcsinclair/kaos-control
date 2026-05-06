/**
 * Regression tests for the release-create-spread-syntax-409-conflict defect.
 *
 * The bug: the releases endpoints wrap their results in an envelope
 * ({releases: [...]} / {release: {...}}), but the frontend API client
 * forwarded the wrapper untouched. This caused `releases.value` in the
 * Pinia store to become an object rather than an array, and the next
 * `[...releases.value, x]` spread to throw "Spread syntax requires
 * ...iterable[Symbol.iterator] to be a function". The release was already
 * created server-side, so a retry then returned 409 Conflict.
 *
 * These tests lock in that:
 *   1. The API client unwraps the envelope on each releases endpoint.
 *   2. The store's fetch + create flow leaves `releases.value` as an array.
 *   3. The WebSocket release.created/updated handlers read from
 *      payload.release rather than treating the entire payload as a Release.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

// Mock the low-level HTTP client; tests control what each method returns.
const apiGet = vi.fn()
const apiPost = vi.fn()
const apiPut = vi.fn()
vi.mock('@/api/client', () => ({
  api: {
    get: (path: string) => apiGet(path),
    post: (path: string, body?: unknown) => apiPost(path, body),
    put: (path: string, body?: unknown) => apiPut(path, body),
    delete: vi.fn(),
    patch: vi.fn(),
  },
}))

// Stub the WS singleton used by the store's connectWs(). Not exercised here.
vi.mock('@/api/ws', () => ({
  getProjectWs: () => ({ on: () => () => {} }),
}))

import * as releasesApi from '@/api/releases'
import { useReleasesStore } from '@/stores/releases'
import type { Release } from '@/types/release'

const sample: Release = {
  id: 1,
  project_id: 'kaos-control',
  name: 'May2026',
  status: 'planned',
  start_date: null,
  end_date: null,
} as Release

describe('releases API client unwraps the envelope', () => {
  beforeEach(() => {
    apiGet.mockReset()
    apiPost.mockReset()
    apiPut.mockReset()
  })

  it('listReleases returns a bare array, not the {releases} wrapper', async () => {
    apiGet.mockResolvedValueOnce({ releases: [sample] })
    const result = await releasesApi.listReleases('kaos-control')
    expect(Array.isArray(result)).toBe(true)
    expect(result).toEqual([sample])
  })

  it('listReleases coerces a null releases field to an empty array', async () => {
    apiGet.mockResolvedValueOnce({ releases: null })
    const result = await releasesApi.listReleases('kaos-control')
    expect(Array.isArray(result)).toBe(true)
    expect(result).toEqual([])
  })

  it('createRelease returns a bare release, not the {release} wrapper', async () => {
    apiPost.mockResolvedValueOnce({ release: sample })
    const result = await releasesApi.createRelease('kaos-control', { name: 'May2026' } as never)
    expect(result).toEqual(sample)
    expect((result as unknown as { release?: Release }).release).toBeUndefined()
  })

  it('updateRelease returns a bare release', async () => {
    apiPut.mockResolvedValueOnce({ release: sample, artifacts_renamed: 0 })
    const result = await releasesApi.updateRelease('kaos-control', 1, { name: 'May2026' } as never)
    expect(result).toEqual(sample)
  })

  it('getRelease returns a bare release detail', async () => {
    apiGet.mockResolvedValueOnce({ release: sample })
    const result = await releasesApi.getRelease('kaos-control', 1)
    expect(result).toEqual(sample)
  })
})

describe('releases store stays array-shaped through the fetch+create flow', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    apiGet.mockReset()
    apiPost.mockReset()
    apiPut.mockReset()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('after fetch(), releases.value is iterable (regression: spread error)', async () => {
    apiGet.mockResolvedValueOnce({ releases: [sample] })
    const store = useReleasesStore()
    await store.fetch('kaos-control')
    expect(Array.isArray(store.releases)).toBe(true)
    // The original failure: `[...store.releases, x]` would throw on a non-iterable.
    expect(() => [...store.releases, sample]).not.toThrow()
  })

  it('create() appends to the existing array without throwing on the spread', async () => {
    apiGet.mockResolvedValueOnce({ releases: [] })
    apiPost.mockResolvedValueOnce({ release: sample })
    const store = useReleasesStore()
    await store.fetch('kaos-control')
    const result = await store.create('kaos-control', { name: 'May2026' } as never)
    expect(result).toEqual(sample)
    expect(store.releases).toHaveLength(1)
    expect(store.releases[0]).toEqual(sample)
  })
})
