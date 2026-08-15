// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, afterEach } from 'vitest'
import { migrateDirectives, refreshDirectives } from '../directives'

// Frontend plan: lifecycle/frontend-plans/agent-directives-generation-4-fe.md — Milestone 1

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

describe('directives API', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  describe('migrateDirectives', () => {
    it('posts to /projects/{project}/migrate-directives with the force flag', async () => {
      const mock = mockFetch(200, { files: [] })
      await migrateDirectives('kaos-control', { force: true })
      const url: string = mock.mock.calls[0][0] as string
      const init = mock.mock.calls[0][1] as RequestInit
      expect(url).toContain('/projects/kaos-control/migrate-directives')
      expect(JSON.parse(init.body as string)).toEqual({ force: true })
    })

    it('encodes the project name in the URL', async () => {
      const mock = mockFetch(200, { files: [] })
      await migrateDirectives('my project', { force: false })
      const url: string = mock.mock.calls[0][0] as string
      expect(url).toContain('/projects/my%20project/migrate-directives')
    })
  })

  describe('refreshDirectives', () => {
    it('posts to /p/{project}/directives/refresh with the force flag', async () => {
      const mock = mockFetch(200, { files: [] })
      await refreshDirectives('kaos-control', { force: false })
      const url: string = mock.mock.calls[0][0] as string
      const init = mock.mock.calls[0][1] as RequestInit
      expect(url).toContain('/p/kaos-control/directives/refresh')
      expect(JSON.parse(init.body as string)).toEqual({ force: false })
    })

    it('returns the typed GenerateResult', async () => {
      mockFetch(200, {
        files: [{ path: 'AGENTS.md', created: true, changed: false, skipped: false }],
        disabledAgents: ['backend-developer'],
        skipped: ['GEMINI.md'],
      })
      const result = await refreshDirectives('kaos-control', { force: false })
      expect(result.files).toEqual([
        { path: 'AGENTS.md', created: true, changed: false, skipped: false },
      ])
      expect(result.disabledAgents).toEqual(['backend-developer'])
      expect(result.skipped).toEqual(['GEMINI.md'])
    })
  })
})
