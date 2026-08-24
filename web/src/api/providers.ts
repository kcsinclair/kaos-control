// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from './client'
import type { ProviderConfig, ProviderHealth, DiscoveredModel, ProviderProbeResult } from '@/types/api'

export async function listProviders(): Promise<{ providers: ProviderConfig[] }> {
  return api.get<{ providers: ProviderConfig[] }>('/providers')
}

export async function getProvider(name: string): Promise<{ provider: ProviderConfig }> {
  return api.get<{ provider: ProviderConfig }>(`/providers/${encodeURIComponent(name)}`)
}

export async function createProvider(data: ProviderConfig): Promise<{ provider: ProviderConfig }> {
  return api.post<{ provider: ProviderConfig }>('/providers', data)
}

export async function updateProvider(
  name: string,
  data: Partial<ProviderConfig>,
): Promise<{ provider: ProviderConfig }> {
  return api.put<{ provider: ProviderConfig }>(
    `/providers/${encodeURIComponent(name)}`,
    data,
  )
}

export async function deleteProvider(name: string): Promise<{ ok: boolean; deleted: string }> {
  await api.delete<void>(`/providers/${encodeURIComponent(name)}`)
  return { ok: true, deleted: name }
}

export async function getProviderHealth(name: string): Promise<ProviderHealth> {
  const res = await api.get<{ healthy?: boolean; ok?: boolean; latency_ms?: number; error?: string }>(
    `/providers/${encodeURIComponent(name)}/health`,
  )
  return {
    ok: res.ok ?? res.healthy ?? false,
    latency_ms: res.latency_ms,
    error: res.error,
  }
}

export async function getProviderModels(name: string): Promise<{ models: DiscoveredModel[] }> {
  return api.get<{ models: DiscoveredModel[] }>(
    `/providers/${encodeURIComponent(name)}/models`,
  )
}

// ── Compatibility and convenience helpers ─────────────────────────────────

export async function getProviders(): Promise<ProviderConfig[]> {
  const data = await listProviders()
  return data.providers ?? []
}

export async function saveProvider(provider: ProviderConfig, isEdit = false): Promise<ProviderConfig> {
  if (isEdit) {
    const res = await updateProvider(provider.name, {
      base_url: provider.base_url,
      driver: provider.driver,
      api_key: provider.api_key,
      extra_headers: provider.extra_headers,
    })
    return res.provider
  }
  const res = await createProvider(provider)
  return res.provider
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

export async function listModels(name: string): Promise<DiscoveredModel[]> {
  const data = await getProviderModels(name)
  return data.models ?? []
}

export async function testProvider(name: string, model?: string): Promise<ProviderProbeResult> {
  try {
    const testRes = await testConnection({ name, model })
    let models: DiscoveredModel[] = []
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
