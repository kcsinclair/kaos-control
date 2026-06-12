// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'

// Mock the API client so we can capture the URL passed to api.get.
vi.mock('@/api/client', () => ({
  api: {
    get: vi.fn().mockResolvedValue({
      summary: { overall: {}, per_model: [], per_agent: [] },
      series: [],
      series_by_model: {},
    }),
  },
}))

import { getAgentUsageReport } from '../../web/src/api/reports'
import { api } from '@/api/client'

const mockGet = vi.mocked(api.get)

beforeEach(() => {
  vi.clearAllMocks()
})

describe('getAgentUsageReport', () => {
  it('builds query params from filter', async () => {
    await getAgentUsageReport('myproject', {
      from: '2026-01-01T00:00:00Z',
      to: '2026-01-31T23:59:59Z',
      agent: ['qa', 'backend-developer'],
      status: ['done'],
      bucket: 'hour',
      tz: 'UTC',
    })

    expect(mockGet).toHaveBeenCalledOnce()
    const url: string = mockGet.mock.calls[0][0] as string
    expect(url).toContain('from=')
    expect(url).toContain('to=')
    expect(url).toContain('agent=qa')
    expect(url).toContain('agent=backend-developer')
    expect(url).toContain('status=done')
    expect(url).toContain('bucket=hour')
    expect(url).toContain('tz=UTC')
  })

  it('defaults tz to browser timezone when not supplied', async () => {
    await getAgentUsageReport('myproject', {
      bucket: 'day',
    })

    const url: string = mockGet.mock.calls[0][0] as string
    const browserTz = Intl.DateTimeFormat().resolvedOptions().timeZone
    expect(browserTz).toBeTruthy()
    expect(url).toContain('tz=')
    // The URL-encoded timezone should decode back to the browser TZ.
    const parsed = new URL('http://localhost' + url)
    expect(parsed.searchParams.get('tz')).toBe(browserTz)
  })

  it('omits unset fields but always includes tz', async () => {
    await getAgentUsageReport('myproject', {
      bucket: 'week',
    })

    const url: string = mockGet.mock.calls[0][0] as string
    const parsed = new URL('http://localhost' + url)

    expect(parsed.searchParams.has('from')).toBe(false)
    expect(parsed.searchParams.has('to')).toBe(false)
    expect(parsed.searchParams.getAll('agent')).toHaveLength(0)
    expect(parsed.searchParams.getAll('status')).toHaveLength(0)
    // tz is always appended (defaults to browser timezone).
    expect(parsed.searchParams.has('tz')).toBe(true)
  })
})
