// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useProvidersStore } from '../providers'
import * as providersApi from '@/api/providers'

vi.mock('@/api/providers', () => ({
  getProviders: vi.fn().mockResolvedValue([
    { name: 'llama-cpp', base_url: 'http://127.0.0.1:8080', driver: 'openai-compatible' },
  ]),
  createProvider: vi.fn().mockImplementation((p) => Promise.resolve(p)),
  updateProvider: vi.fn().mockImplementation((name, p) => Promise.resolve({ name, ...p })),
  saveProvider: vi.fn().mockImplementation((p) => Promise.resolve(p)),
  deleteProvider: vi.fn().mockResolvedValue(undefined),
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

  it('fetches providers and sets state', async () => {
    const store = useProvidersStore()
    await store.fetchProviders()
    expect(store.providers).toHaveLength(1)
    expect(store.providers[0].name).toBe('llama-cpp')
  })

  it('creates a provider and refreshes list', async () => {
    const store = useProvidersStore()
    await store.createProvider({
      name: 'openrouter',
      base_url: 'https://openrouter.ai/api',
      driver: 'openai-compatible',
    })
    expect(providersApi.createProvider).toHaveBeenCalled()
    expect(providersApi.getProviders).toHaveBeenCalled()
  })

  it('probes a provider and caches probeResult and models', async () => {
    const store = useProvidersStore()
    const res = await store.probeProvider('llama-cpp')
    expect(res.ok).toBe(true)
    expect(store.probeResults.get('llama-cpp')?.ok).toBe(true)
    expect(store.models.get('llama-cpp')).toHaveLength(1)
  })

  it('deletes a provider and clears cached probe/models', async () => {
    const store = useProvidersStore()
    await store.probeProvider('llama-cpp')
    expect(store.probeResults.has('llama-cpp')).toBe(true)

    await store.deleteProvider('llama-cpp')
    expect(providersApi.deleteProvider).toHaveBeenCalledWith('llama-cpp')
    expect(store.probeResults.has('llama-cpp')).toBe(false)
    expect(store.models.has('llama-cpp')).toBe(false)
  })
})
