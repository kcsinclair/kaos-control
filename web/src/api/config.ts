import { api } from './client'

export function getConfig(project: string) {
  return api.get<{ raw: string }>(`/p/${encodeURIComponent(project)}/config`)
}

export function updateConfig(project: string, raw: string) {
  return api.put<{ ok: boolean }>(`/p/${encodeURIComponent(project)}/config`, { raw })
}
