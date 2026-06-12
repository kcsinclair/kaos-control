// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { listDocs, getDoc, putDoc } from '../docs'

function mockFetch(status: number, body: unknown): ReturnType<typeof vi.fn> {
  const mock = vi.fn().mockResolvedValue(
    new Response(JSON.stringify(body), {
      status,
      headers: { 'Content-Type': 'application/json' },
    }),
  )
  vi.stubGlobal('fetch', mock)
  return mock
}

describe('docs API', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  describe('listDocs', () => {
    it('encodes the project name in the URL', async () => {
      const mock = mockFetch(200, { docs: [], docs_dir_present: false })
      await listDocs('my project')
      const url: string = mock.mock.calls[0][0] as string
      expect(url).toContain('/p/my%20project/docs')
    })
  })

  describe('getDoc', () => {
    it('encodes each path segment individually, preserving slashes as separators', async () => {
      const mock = mockFetch(200, {
        path: 'subsystems/agents.md',
        file_sha: 'abc',
        is_markdown: true,
        body: '# Agents',
      })
      await getDoc('kaos-control', 'subsystems/agents.md')
      const url: string = mock.mock.calls[0][0] as string
      // Slashes between segments must NOT be encoded
      expect(url).toContain('/docs/subsystems/agents.md')
      // Double-encoding guard: slash must not appear as %2F
      expect(url).not.toContain('%2F')
    })

    it('encodes special characters within individual segments', async () => {
      const mock = mockFetch(200, {
        path: 'my dir/file name.md',
        file_sha: 'abc',
        is_markdown: true,
        body: '',
      })
      await getDoc('kaos-control', 'my dir/file name.md')
      const url: string = mock.mock.calls[0][0] as string
      expect(url).toContain('/docs/my%20dir/file%20name.md')
    })

    it('handles a root-level (non-nested) path', async () => {
      const mock = mockFetch(200, {
        path: 'architecture.md',
        file_sha: 'abc',
        is_markdown: true,
        body: '# Architecture',
      })
      await getDoc('kaos-control', 'architecture.md')
      const url: string = mock.mock.calls[0][0] as string
      expect(url).toContain('/docs/architecture.md')
    })
  })

  describe('putDoc', () => {
    it('sends body and expected_sha in the request body', async () => {
      const mock = mockFetch(200, { file_sha: 'new-sha' })
      await putDoc('kaos-control', 'subsystems/agents.md', '# Hello', 'old-sha')
      const init = mock.mock.calls[0][1] as RequestInit
      const sent = JSON.parse(init.body as string) as Record<string, unknown>
      expect(sent).toEqual({ body: '# Hello', expected_sha: 'old-sha' })
    })

    it('uses segment-by-segment encoding in the URL', async () => {
      const mock = mockFetch(200, { file_sha: 'new-sha' })
      await putDoc('kaos-control', 'subsystems/agents.md', '# Hello', 'sha')
      const url: string = mock.mock.calls[0][0] as string
      expect(url).toContain('/docs/subsystems/agents.md')
      expect(url).not.toContain('%2F')
    })

    it('returns the new file_sha on success', async () => {
      mockFetch(200, { file_sha: 'new-sha-123' })
      const result = await putDoc('kaos-control', 'readme.md', '# Hello', 'old')
      expect(result.file_sha).toBe('new-sha-123')
    })
  })
})
