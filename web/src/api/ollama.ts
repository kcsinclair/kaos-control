import { api } from './client'
import type { OllamaInstance, OllamaHealthResponse, OllamaModel } from '@/types/api'

export function listInstances() {
  return api.get<{ instances: OllamaInstance[] }>('/ollama/instances')
}

export function createInstance(payload: OllamaInstance) {
  return api.post<{ instance: OllamaInstance }>('/ollama/instances', payload)
}

export function updateInstance(name: string, payload: Partial<Omit<OllamaInstance, 'name'>>) {
  return api.put<{ instance: OllamaInstance }>(
    `/ollama/instances/${encodeURIComponent(name)}`,
    payload,
  )
}

export function deleteInstance(name: string) {
  return api.delete<void>(`/ollama/instances/${encodeURIComponent(name)}`)
}

export function getHealth(name: string) {
  return api.get<OllamaHealthResponse>(`/ollama/instances/${encodeURIComponent(name)}/health`)
}

export function listModels(name: string) {
  return api.get<{ models: OllamaModel[] }>(
    `/ollama/instances/${encodeURIComponent(name)}/models`,
  )
}
