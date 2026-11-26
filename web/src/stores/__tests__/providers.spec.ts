// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useProvidersStore } from '../providers'
import * as providersApi from '@/api/providers'

vi.mock('@/api/providers', () => ({
  listProviders: vi.fn().mockImplementation(() =>
    Promise.resolve({
      providers: [
        { name: 'llama-cpp', base_url: 'http://127.0.0.1:8080', driver: 'openai-compatible' },
      ],
    }),
  ),
  getProviders: vi.fn().mockImplementation(() =>
    Promise.resolve([
      { name: 'llama-cpp', base_url: 'http://127.0.0.1:8080', driver: 'openai-compatible' },
    ]),
  ),
  getProvider: vi.fn().mockImplementation((name: string) =>
    Promise.resolve({
      provider: { name, base_url: 'http://127.0.0.1:8080', driver: 'openai-compatible' },
    }),
  ),
  createProvider: vi.fn().mockImplementation((p) => Promise.resolve({ provider: { ...p } })),
  updateProvider: vi.fn().mockImplementation((name, p) => Promise.resolve({ provider: { name, ...p } })),
  saveProvider: vi.fn().mockImplementation((p) => Promise.resolve({ ...p })),
  deleteProvider: vi.fn().mockResolvedValue({ ok: true, deleted: 'llama-cpp' }),
  getProviderHealth: vi.fn().mockResolvedValue({
    ok: true,
    latency_ms: 42,
  }),
  getProviderModels: vi.fn().mockResolvedValue({
    models: [{ id: 'model-1', name: 'Model 1', supports_tools: true }],
  }),
  testProvider: vi.fn().mockResolvedValue({
    ok: true,
    latency_ms: 42,
    models: [{ id: 'model-1', name: 'Model 1', supports_tools: true }],
  }),
  listModels: vi.fn().mockResolvedValue([{ id: 'model-1', name: 'Model 1' }]),
}))

describe('providers store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('initializes with empty state', () => {
    const store = useProvidersStore()
    expect(store.providers).toEqual([])
    expect(store.health.size).toBe(0)
    expect(store.models.size).toBe(0)
    expect(store.loading).toBe(false)
    expect(store.error).toBeNull()
  })

  it('fetches providers and sets state', async () => {
    const store = useProvidersStore()
    await store.fetchProviders()
    expect(store.providers).toHaveLength(1)
    expect(store.providers[0].name).toBe('llama-cpp')
    expect(store.loading).toBe(false)
    expect(store.error).toBeNull()
  })

  it('creates a provider and updates store list', async () => {
    const store = useProvidersStore()
    await store.fetchProviders()
    const newProv = {
      name: 'openrouter',
      base_url: 'https://openrouter.ai/api',
      driver: 'openai-compatible',
    }
    await store.createProvider(newProv)
    expect(providersApi.createProvider).toHaveBeenCalledWith(newProv)
    expect(store.providers).toHaveLength(2)
    expect(store.providers[1].name).toBe('openrouter')
  })

  it('updates an existing provider', async () => {
    const store = useProvidersStore()
    await store.fetchProviders()
    await store.updateProvider('llama-cpp', { base_url: 'http://127.0.0.1:9090' })
    expect(providersApi.updateProvider).toHaveBeenCalledWith('llama-cpp', { base_url: 'http://127.0.0.1:9090' })
    expect(store.providers[0].base_url).toBe('http://127.0.0.1:9090')
  })

  it('checks health and updates health map', async () => {
    const store = useProvidersStore()
    const res = await store.checkHealth('llama-cpp')
    expect(res.ok).toBe(true)
    expect(res.latency_ms).toBe(42)
    expect(store.health.get('llama-cpp')?.ok).toBe(true)
    expect(store.health.get('llama-cpp')?.latency_ms).toBe(42)
  })

  it('fetches models and updates models map', async () => {
    const store = useProvidersStore()
    const list = await store.fetchModels('llama-cpp')
    expect(list).toHaveLength(1)
    expect(store.models.get('llama-cpp')).toHaveLength(1)
    expect(store.models.get('llama-cpp')?.[0].id).toBe('model-1')
  })

  it('probes a provider and caches probeResult and models', async () => {
    const store = useProvidersStore()
    const res = await store.probeProvider('llama-cpp')
    expect(res.ok).toBe(true)
    expect(store.probeResults.get('llama-cpp')?.ok).toBe(true)
    expect(store.models.get('llama-cpp')).toHaveLength(1)
  })

  it('deletes a provider, cleans state and handles errors', async () => {
    const store = useProvidersStore()
    await store.fetchProviders()
    await store.checkHealth('llama-cpp')
    await store.fetchModels('llama-cpp')
    expect(store.providers).toHaveLength(1)
    expect(store.health.has('llama-cpp')).toBe(true)
    expect(store.models.has('llama-cpp')).toBe(true)

    await store.deleteProvider('llama-cpp')
    expect(providersApi.deleteProvider).toHaveBeenCalledWith('llama-cpp')
    expect(store.providers).toHaveLength(0)
    expect(store.health.has('llama-cpp')).toBe(false)
    expect(store.models.has('llama-cpp')).toBe(false)

    // Error handling
    vi.mocked(providersApi.deleteProvider).mockRejectedValueOnce(new Error('Conflict: in use'))
    await expect(store.deleteProvider('llama-cpp')).rejects.toThrow('Conflict: in use')
    expect(store.error).toBe('Conflict: in use')
  })
})
