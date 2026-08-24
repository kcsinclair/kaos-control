// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, afterEach } from 'vitest'
import {
  listProviders,
  getProvider,
  createProvider,
  updateProvider,
  deleteProvider,
  getProviderHealth,
  getProviderModels,
} from './providers'
import { ApiError } from './client'

function mockFetch(status: number, body: unknown): ReturnType<typeof vi.fn> {
  const mock = vi.fn().mockResolvedValue(
    new Response(status === 204 ? null : JSON.stringify(body), {
      status,
      headers: { 'Content-Type': 'application/json' },
    }),
  )
  vi.stubGlobal('fetch', mock)
  return mock
}

describe('providers API client', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  describe('listProviders', () => {
    it('sends GET /api/providers and returns provider list', async () => {
      const providers = [
        { name: 'llama-local', base_url: 'http://localhost:8080', driver: 'openai-compatible' },
      ]
      const mock = mockFetch(200, { providers })
      const res = await listProviders()

      expect(mock).toHaveBeenCalledTimes(1)
      const url = mock.mock.calls[0][0] as string
      const init = mock.mock.calls[0][1] as RequestInit
      expect(url).toContain('/api/providers')
      expect(init.method).toBe('GET')
      expect(res).toEqual({ providers })
    })
  })

  describe('getProvider', () => {
    it('sends GET /api/providers/:name with URI encoding', async () => {
      const provider = {
        name: 'custom/provider',
        base_url: 'https://api.openai.com/v1',
        driver: 'openai-compatible',
      }
      const mock = mockFetch(200, { provider })
      const res = await getProvider('custom/provider')

      expect(mock).toHaveBeenCalledTimes(1)
      const url = mock.mock.calls[0][0] as string
      expect(url).toContain('/api/providers/custom%2Fprovider')
      expect(res).toEqual({ provider })
    })
  })

  describe('createProvider', () => {
    it('sends POST /api/providers with JSON payload', async () => {
      const payload = {
        name: 'openrouter',
        base_url: 'https://openrouter.ai/api/v1',
        driver: 'openai-compatible',
        api_key: 'sk-or-123',
        extra_headers: { 'HTTP-Referer': 'https://kaos-control.local' },
      }
      const mock = mockFetch(201, { provider: { ...payload, api_key: '***' } })
      const res = await createProvider(payload)

      expect(mock).toHaveBeenCalledTimes(1)
      const url = mock.mock.calls[0][0] as string
      const init = mock.mock.calls[0][1] as RequestInit
      expect(url).toContain('/api/providers')
      expect(init.method).toBe('POST')
      expect(JSON.parse(init.body as string)).toEqual(payload)
      expect(res.provider.api_key).toBe('***')
    })
  })

  describe('updateProvider', () => {
    it('sends PUT /api/providers/:name with updated data', async () => {
      const updateData = {
        base_url: 'https://openrouter.ai/api/v2',
        extra_headers: { 'X-Title': 'kaos-control' },
      }
      const mock = mockFetch(200, {
        provider: {
          name: 'openrouter',
          base_url: 'https://openrouter.ai/api/v2',
          driver: 'openai-compatible',
          extra_headers: { 'X-Title': 'kaos-control' },
        },
      })
      const res = await updateProvider('openrouter', updateData)

      expect(mock).toHaveBeenCalledTimes(1)
      const url = mock.mock.calls[0][0] as string
      const init = mock.mock.calls[0][1] as RequestInit
      expect(url).toContain('/api/providers/openrouter')
      expect(init.method).toBe('PUT')
      expect(JSON.parse(init.body as string)).toEqual(updateData)
      expect(res.provider.base_url).toBe('https://openrouter.ai/api/v2')
    })
  })

  describe('deleteProvider', () => {
    it('sends DELETE /api/providers/:name and returns ok with deleted name', async () => {
      const mock = mockFetch(204, null)
      const res = await deleteProvider('old-provider')

      expect(mock).toHaveBeenCalledTimes(1)
      const url = mock.mock.calls[0][0] as string
      const init = mock.mock.calls[0][1] as RequestInit
      expect(url).toContain('/api/providers/old-provider')
      expect(init.method).toBe('DELETE')
      expect(res).toEqual({ ok: true, deleted: 'old-provider' })
    })

    it('throws ApiError on 409 conflict when in use', async () => {
      mockFetch(409, {
        error: { code: 'conflict', message: 'cannot delete provider: in use by project demo' },
      })
      await expect(deleteProvider('in-use')).rejects.toThrow(ApiError)
    })
  })

  describe('getProviderHealth', () => {
    it('sends GET /api/providers/:name/health and normalizes response', async () => {
      const mock = mockFetch(200, { healthy: true, latency_ms: 35 })
      const res = await getProviderHealth('llama-local')

      expect(mock).toHaveBeenCalledTimes(1)
      const url = mock.mock.calls[0][0] as string
      expect(url).toContain('/api/providers/llama-local/health')
      expect(res).toEqual({ ok: true, latency_ms: 35, error: undefined })
    })

    it('handles ok format and errors', async () => {
      mockFetch(200, { healthy: false, error: 'connection refused', latency_ms: 5000 })
      const res = await getProviderHealth('llama-local')
      expect(res).toEqual({ ok: false, error: 'connection refused', latency_ms: 5000 })
    })
  })

  describe('getProviderModels', () => {
    it('sends GET /api/providers/:name/models and returns discovered models', async () => {
      const models = [
        { id: 'qwen-30b', name: 'Qwen 30B', supports_tools: true },
        { id: 'llama-70b', name: 'Llama 70B', supports_tools: false },
      ]
      const mock = mockFetch(200, { models })
      const res = await getProviderModels('openrouter')

      expect(mock).toHaveBeenCalledTimes(1)
      const url = mock.mock.calls[0][0] as string
      expect(url).toContain('/api/providers/openrouter/models')
      expect(res).toEqual({ models })
    })
  })
})
