// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * @deprecated Use `@/api/providers` instead. This module is retained for backwards compatibility.
 */

import { api } from './client'
import type { OllamaInstance, OllamaHealthResponse, OllamaModel } from '@/types/api'

/** @deprecated Use `getProviders` from `@/api/providers` */
export function listInstances() {
  return api.get<{ instances: OllamaInstance[] }>('/ollama/instances')
}

/** @deprecated Use `createProvider` from `@/api/providers` */
export function createInstance(payload: OllamaInstance) {
  return api.post<{ instance: OllamaInstance }>('/ollama/instances', payload)
}

/** @deprecated Use `updateProvider` from `@/api/providers` */
export function updateInstance(name: string, payload: Partial<Omit<OllamaInstance, 'name'>>) {
  return api.put<{ instance: OllamaInstance }>(
    `/ollama/instances/${encodeURIComponent(name)}`,
    payload,
  )
}

/** @deprecated Use `deleteProvider` from `@/api/providers` */
export function deleteInstance(name: string) {
  return api.delete<void>(`/providers/${encodeURIComponent(name)}`)
}

/** @deprecated Use `testConnection` or `testProvider` from `@/api/providers` */
export function getHealth(name: string) {
  return api.get<OllamaHealthResponse>(`/ollama/instances/${encodeURIComponent(name)}/health`)
}

/** @deprecated Use `listModels` from `@/api/providers` */
export function listModels(name: string) {
  return api.get<{ models: OllamaModel[] }>(
    `/providers/${encodeURIComponent(name)}/models`,
  )
}

