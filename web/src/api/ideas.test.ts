// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { triageIdea } from './ideas'
import { ApiError } from './client'

function mockFetch(status: number, body: unknown): void {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue(
      new Response(JSON.stringify(body), {
        status,
        headers: { 'Content-Type': 'application/json' },
      }),
    ),
  )
}

describe('triageIdea', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('returns run_id on success', async () => {
    mockFetch(202, { run_id: 'abc-123' })
    const result = await triageIdea('my-project', 'my-slug')
    expect(result).toEqual({ run_id: 'abc-123' })
  })

  it('throws ApiError with reason code on 409', async () => {
    mockFetch(409, { error: { code: 'wrong_status', message: 'cannot triage: wrong_status' } })
    const err = await triageIdea('my-project', 'my-slug').catch((e) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect((err as ApiError).code).toBe('wrong_status')
    expect((err as ApiError).status).toBe(409)
  })

  it('throws ApiError with status 401', async () => {
    mockFetch(401, { error: { code: 'unauthorized', message: 'Unauthorized' } })
    const err = await triageIdea('my-project', 'my-slug').catch((e) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect((err as ApiError).status).toBe(401)
  })
})
