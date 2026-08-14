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

import { api, ApiError } from '@/api/client'
import { generateIdea } from '@/api/ideaChat'
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

// Frontend plan: lifecycle/frontend-plans/defect-generate-missing-template-3-fe.md
// §Milestone 1 — error mapping; §Milestone 4 — these tests.
describe('brainDump store — generate() error mapping', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('maps a 422 template_unavailable error to actionable guidance, never the raw message', async () => {
    vi.mocked(generateIdea).mockRejectedValue(
      new ApiError('template_unavailable', 'idea-capture agent has no template "defect-generate"', 422),
    )

    const store = useBrainDumpStore()
    store.artifactType = 'defect'
    store.input = 'The submit button does nothing when clicked.'

    await store.generate('testproject')

    expect(store.phase).toBe('input')
    expect(store.errorKind).toBe('config')
    expect(store.error).not.toContain('has no template')
    expect(store.error).toContain("Defect generation isn't configured")
  })

  it('maps a config_error code the same way as template_unavailable', async () => {
    vi.mocked(generateIdea).mockRejectedValue(new ApiError('config_error', 'boom', 500))

    const store = useBrainDumpStore()
    store.artifactType = 'idea'
    store.input = 'An idea worth capturing.'

    await store.generate('testproject')

    expect(store.errorKind).toBe('config')
    expect(store.error).toContain("Idea generation isn't configured")
  })

  it('keeps the raw message for an unrelated error code', async () => {
    vi.mocked(generateIdea).mockRejectedValue(new ApiError('rate_limited', 'Too many requests', 429))

    const store = useBrainDumpStore()
    store.input = 'Some idea text.'

    await store.generate('testproject')

    expect(store.phase).toBe('input')
    expect(store.errorKind).toBe('generic')
    expect(store.error).toBe('Too many requests')
  })
})

describe('brainDump store — createDefectManually', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('posts a minimal defect artifact and returns the created path', async () => {
    vi.mocked(api.post).mockResolvedValue({ artifact: { path: 'lifecycle/defects/button-broken.md' } })

    const store = useBrainDumpStore()
    store.input = 'Button does nothing when clicked.'

    const path = await store.createDefectManually('testproject')

    expect(path).toBe('lifecycle/defects/button-broken.md')
    const [url, body] = vi.mocked(api.post).mock.calls[0] as [string, Record<string, unknown>]
    expect(url).toBe('/p/testproject/artifacts')
    expect(body.stage).toBe('defects')
    const frontmatter = body.frontmatter as Record<string, unknown>
    expect(frontmatter.type).toBe('defect')
    expect(frontmatter.status).toBe('raw')
    expect(frontmatter.labels).toEqual(['defect'])
  })

  it('returns null and makes no API call when input is empty', async () => {
    const store = useBrainDumpStore()
    store.input = '   '

    const result = await store.createDefectManually('testproject')

    expect(result).toBeNull()
    expect(vi.mocked(api.post)).not.toHaveBeenCalled()
  })
})
