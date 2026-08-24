// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as providersApi from '@/api/providers'
import type { Provider, ProviderModel, ProviderProbeResult } from '@/types/api'

export const useProvidersStore = defineStore('providers', () => {
  const providers = ref<Provider[]>([])
  const loading = ref(false)
  const probeResults = ref(new Map<string, ProviderProbeResult>())
  const models = ref(new Map<string, ProviderModel[]>())

  async function fetchProviders(): Promise<void> {
    loading.value = true
    try {
      providers.value = await providersApi.getProviders()
    } finally {
      loading.value = false
    }
  }

  async function createProvider(provider: Provider): Promise<Provider> {
    const created = await providersApi.createProvider(provider)
    await fetchProviders()
    return created
  }

  async function updateProvider(
    name: string,
    payload: Partial<Omit<Provider, 'name'>>,
  ): Promise<Provider> {
    const updated = await providersApi.updateProvider(name, payload)
    await fetchProviders()
    return updated
  }

  async function saveProvider(provider: Provider, isEdit = false): Promise<Provider> {
    const saved = await providersApi.saveProvider(provider, isEdit)
    await fetchProviders()
    return saved
  }

  async function deleteProvider(name: string): Promise<void> {
    await providersApi.deleteProvider(name)
    probeResults.value.delete(name)
    models.value.delete(name)
    await fetchProviders()
  }

  async function probeProvider(name: string, model?: string): Promise<ProviderProbeResult> {
    const result = await providersApi.testProvider(name, model)
    probeResults.value = new Map(probeResults.value).set(name, result)
    if (result.models && result.models.length > 0) {
      models.value = new Map(models.value).set(name, result.models)
    }
    return result
  }

  async function fetchModels(name: string): Promise<ProviderModel[]> {
    try {
      const list = await providersApi.listModels(name)
      models.value = new Map(models.value).set(name, list)
      return list
    } catch {
      return []
    }
  }

  async function probeAll(): Promise<void> {
    await Promise.all(providers.value.map((p) => probeProvider(p.name)))
  }

  return {
    providers,
    loading,
    probeResults,
    models,
    fetchProviders,
    createProvider,
    updateProvider,
    saveProvider,
    deleteProvider,
    probeProvider,
    fetchModels,
    probeAll,
  }
})
