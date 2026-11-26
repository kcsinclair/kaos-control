// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as providersApi from '@/api/providers'
import type { ProviderConfig, ProviderHealth, DiscoveredModel, ProviderProbeResult } from '@/types/api'

export const useProvidersStore = defineStore('providers', () => {
  const providers = ref<ProviderConfig[]>([])
  const health = ref(new Map<string, ProviderHealth>())
  const models = ref(new Map<string, DiscoveredModel[]>())
  const probeResults = ref(new Map<string, ProviderProbeResult>())
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchProviders(): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const data = await providersApi.listProviders()
      providers.value = [...(data.providers ?? [])]
    } catch (err: unknown) {
      error.value = err instanceof Error ? err.message : 'Failed to load providers'
      providers.value = []
    } finally {
      loading.value = false
    }
  }

  async function createProvider(data: ProviderConfig): Promise<ProviderConfig> {
    loading.value = true
    error.value = null
    try {
      const res = await providersApi.createProvider(data)
      const created = res.provider ?? data
      const idx = providers.value.findIndex((p) => p.name === created.name)
      if (idx >= 0) {
        providers.value[idx] = created
      } else {
        providers.value.push(created)
      }
      return created
    } catch (err: unknown) {
      error.value = err instanceof Error ? err.message : 'Failed to create provider'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateProvider(
    name: string,
    data: Partial<ProviderConfig>,
  ): Promise<ProviderConfig> {
    loading.value = true
    error.value = null
    try {
      const res = await providersApi.updateProvider(name, data)
      const updated = res.provider
      const idx = providers.value.findIndex((p) => p.name === name)
      if (idx >= 0) {
        providers.value[idx] = updated
      }
      return updated
    } catch (err: unknown) {
      error.value = err instanceof Error ? err.message : 'Failed to update provider'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function saveProvider(provider: ProviderConfig, isEdit = false): Promise<ProviderConfig> {
    if (isEdit) {
      return updateProvider(provider.name, {
        base_url: provider.base_url,
        driver: provider.driver,
        api_key: provider.api_key,
        extra_headers: provider.extra_headers,
      })
    }
    return createProvider(provider)
  }

  async function deleteProvider(name: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      await providersApi.deleteProvider(name)
      providers.value = providers.value.filter((p) => p.name !== name)
      health.value.delete(name)
      models.value.delete(name)
      probeResults.value.delete(name)
    } catch (err: unknown) {
      error.value = err instanceof Error ? err.message : 'Failed to delete provider'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function checkHealth(name: string): Promise<ProviderHealth> {
    try {
      const h = await providersApi.getProviderHealth(name)
      health.value = new Map(health.value).set(name, h)
      probeResults.value = new Map(probeResults.value).set(name, {
        ok: h.ok,
        latency_ms: h.latency_ms,
        error: h.error,
        models: models.value.get(name) ?? [],
      })
      return h
    } catch (err: unknown) {
      const fallback: ProviderHealth = {
        ok: false,
        error: err instanceof Error ? err.message : 'Health check failed',
      }
      health.value = new Map(health.value).set(name, fallback)
      return fallback
    }
  }

  async function checkAllHealth(): Promise<void> {
    await Promise.all(providers.value.map((p) => checkHealth(p.name).catch(() => null)))
  }

  async function fetchModels(name: string): Promise<DiscoveredModel[]> {
    try {
      const res = await providersApi.getProviderModels(name)
      const list = res.models ?? []
      models.value = new Map(models.value).set(name, list)
      return list
    } catch {
      return []
    }
  }

  async function probeProvider(name: string, model?: string): Promise<ProviderProbeResult> {
    try {
      const result = await providersApi.testProvider(name, model)
      probeResults.value = new Map(probeResults.value).set(name, result)
      health.value = new Map(health.value).set(name, {
        ok: result.ok,
        latency_ms: result.latency_ms,
        error: result.error,
      })
      if (result.models && result.models.length > 0) {
        models.value = new Map(models.value).set(name, result.models)
      }
      return result
    } catch (err: unknown) {
      const fallback: ProviderProbeResult = {
        ok: false,
        error: err instanceof Error ? err.message : 'Probe failed',
        models: [],
      }
      probeResults.value = new Map(probeResults.value).set(name, fallback)
      health.value = new Map(health.value).set(name, {
        ok: false,
        error: fallback.error,
      })
      return fallback
    }
  }

  async function probeAll(): Promise<void> {
    await Promise.all(providers.value.map((p) => probeProvider(p.name).catch(() => null)))
  }

  return {
    providers,
    health,
    models,
    probeResults,
    loading,
    error,
    fetchProviders,
    createProvider,
    updateProvider,
    saveProvider,
    deleteProvider,
    checkHealth,
    checkAllHealth,
    fetchModels,
    probeProvider,
    probeAll,
  }
})
