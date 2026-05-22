// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

// Test plan: lifecycle/test-plans/raw-artefact-status-5-test.md §Milestone 3, Scenario 1
//
// Verifies that useBrainDumpStore.createDoc() sends status:'raw' in the API
// payload, which is the "brain-dump default" required by the spec.

// Mock the API modules so no real HTTP calls happen.
vi.mock('@/api/ideaChat', () => ({
  generateIdea: vi.fn(),
}))
vi.mock('@/api/client', () => {
  const postMock = vi.fn()
  return {
    api: { post: postMock },
    ApiError: class ApiError extends Error {
      constructor(public code: string, message: string, public status: number) {
        super(message)
        this.name = 'ApiError'
      }
    },
  }
})

import { api } from '@/api/client'
import { useBrainDumpStore } from '@/stores/brainDump'

describe('brainDump store — createDoc', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  // TC1: createDoc sends status:'raw' in the API payload.
  it('sends status "raw" in the POST payload when creating a doc', async () => {
    vi.mocked(api.post).mockResolvedValue({ artifact: { path: 'lifecycle/docs/test.md' } })

    const store = useBrainDumpStore()
    store.input = 'This is a quick brain dump of an idea.'

    const path = await store.createDoc('testproject')

    expect(path).toBe('lifecycle/docs/test.md')
    expect(vi.mocked(api.post)).toHaveBeenCalledOnce()

    const [_url, body] = vi.mocked(api.post).mock.calls[0] as [string, Record<string, unknown>]
    const frontmatter = body.frontmatter as Record<string, unknown>
    expect(frontmatter.status).toBe('raw')
    expect(frontmatter.type).toBe('doc')
  })

  // TC2: createDoc uses 'raw' as status even when a sourceLineage is provided.
  it('still sends status "raw" when sourceLineage is provided', async () => {
    vi.mocked(api.post).mockResolvedValue({ artifact: { path: 'lifecycle/docs/my-feature.md' } })

    const store = useBrainDumpStore()
    store.input = 'Feature documentation request.'

    await store.createDoc('testproject', { sourceLineage: 'my-feature', sourcePath: 'lifecycle/ideas/my-feature.md' })

    const [_url, body] = vi.mocked(api.post).mock.calls[0] as [string, Record<string, unknown>]
    const frontmatter = body.frontmatter as Record<string, unknown>
    expect(frontmatter.status).toBe('raw')
    expect(frontmatter.parent).toBe('lifecycle/ideas/my-feature.md')
  })

  // TC3: createDoc returns null and does not call the API when input is empty.
  it('returns null and makes no API call when input is empty', async () => {
    const store = useBrainDumpStore()
    store.input = '   '

    const result = await store.createDoc('testproject')

    expect(result).toBeNull()
    expect(vi.mocked(api.post)).not.toHaveBeenCalled()
  })
})
