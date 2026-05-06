import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as ollamaApi from '@/api/ollama'
import type { OllamaInstance, OllamaHealthResponse, OllamaModel } from '@/types/api'

export const useOllamaInstancesStore = defineStore('ollamaInstances', () => {
  const instances = ref<OllamaInstance[]>([])
  const health = ref(new Map<string, OllamaHealthResponse>())
  const models = ref(new Map<string, OllamaModel[]>())
  const loading = ref(false)

  async function fetchInstances(): Promise<void> {
    loading.value = true
    try {
      const data = await ollamaApi.listInstances()
      instances.value = data.instances ?? []
    } finally {
      loading.value = false
    }
  }

  async function createInstance(payload: OllamaInstance): Promise<void> {
    await ollamaApi.createInstance(payload)
    await fetchInstances()
  }

  async function updateInstance(
    name: string,
    payload: Partial<Omit<OllamaInstance, 'name'>>,
  ): Promise<void> {
    await ollamaApi.updateInstance(name, payload)
    await fetchInstances()
  }

  async function deleteInstance(name: string): Promise<void> {
    await ollamaApi.deleteInstance(name)
    health.value.delete(name)
    models.value.delete(name)
    await fetchInstances()
  }

  async function checkHealth(name: string): Promise<void> {
    const result = await ollamaApi.getHealth(name)
    health.value = new Map(health.value).set(name, result)
  }

  async function fetchModels(name: string): Promise<void> {
    const data = await ollamaApi.listModels(name)
    models.value = new Map(models.value).set(name, data.models ?? [])
  }

  async function checkAllHealth(): Promise<void> {
    await Promise.all(instances.value.map((inst) => checkHealth(inst.name)))
  }

  return {
    instances,
    health,
    models,
    loading,
    fetchInstances,
    createInstance,
    updateInstance,
    deleteInstance,
    checkHealth,
    fetchModels,
    checkAllHealth,
  }
})
