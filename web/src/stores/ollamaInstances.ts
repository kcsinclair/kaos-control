// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useOllamaInstancesStore = defineStore('ollamaInstances', () => {
  const instances = ref<{ name: string; base_url: string; api_key?: string }[]>([])
  const health = ref(new Map<string, { ok: boolean; latency_ms?: number; error?: string }>())
  const models = ref(new Map<string, { name: string; size: number }[]>())
  const loading = ref(false)

  async function fetchInstances(): Promise<void> {}
  async function createInstance(_payload: any): Promise<void> {}
  async function updateInstance(_name: string, _payload: any): Promise<void> {}
  async function deleteInstance(_name: string): Promise<void> {}
  async function checkHealth(_name: string): Promise<void> {}
  async function fetchModels(_name: string): Promise<void> {}
  async function checkAllHealth(): Promise<void> {}

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
