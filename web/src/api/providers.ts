// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from './client'
import type { Provider, ProviderModel, ProviderProbeResult } from '@/types/api'

export async function getProviders(): Promise<Provider[]> {
  const data = await api.get<{ providers: Provider[] }>('/providers')
  return data.providers ?? []
}

export async function createProvider(provider: Provider): Promise<Provider> {
  const data = await api.post<{ provider: Provider }>('/providers', provider)
  return data.provider
}

export async function updateProvider(
  name: string,
  provider: Partial<Omit<Provider, 'name'>>,
): Promise<Provider> {
  const data = await api.put<{ provider: Provider }>(
    `/providers/${encodeURIComponent(name)}`,
    provider,
  )
  return data.provider
}

export async function saveProvider(provider: Provider, isEdit = false): Promise<Provider> {
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

export async function deleteProvider(name: string): Promise<void> {
  await api.delete<void>(`/providers/${encodeURIComponent(name)}`)
}

export async function testConnection(payload: {
  name?: string
  base_url?: string
  driver?: string
  api_key?: string
  extra_headers?: Record<string, string>
  model?: string
}): Promise<{ ok: boolean; latency_ms?: number; message?: string; error?: string }> {
  return api.post<{ ok: boolean; latency_ms?: number; message?: string; error?: string }>(
    '/providers/test',
    payload,
  )
}

export async function listModels(name: string): Promise<ProviderModel[]> {
  const data = await api.get<{ models: ProviderModel[] }>(
    `/providers/${encodeURIComponent(name)}/models`,
  )
  return data.models ?? []
}

export async function testProvider(name: string, model?: string): Promise<ProviderProbeResult> {
  try {
    const testRes = await testConnection({ name, model })
    let models: ProviderModel[] = []
    if (testRes.ok) {
      try {
        models = await listModels(name)
      } catch {
        // If models probe fails, still return probe result with empty models
      }
    }
    return {
      ok: testRes.ok,
      latency_ms: testRes.latency_ms,
      message: testRes.message,
      error: testRes.error,
      models,
    }
  } catch (err: unknown) {
    return {
      ok: false,
      error: err instanceof Error ? err.message : 'Connection test failed',
      models: [],
    }
  }
}
