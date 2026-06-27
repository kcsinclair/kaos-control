// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from './client'
import yaml from 'js-yaml'
import type { UserBinding } from '@/types/api'

export function getConfig(project: string) {
  return api.get<{ raw: string }>(`/p/${encodeURIComponent(project)}/config`)
}

export function updateConfig(project: string, raw: string) {
  return api.put<{ ok: boolean }>(`/p/${encodeURIComponent(project)}/config`, { raw })
}

export function getRoles(project: string) {
  return api.get<{ roles: string[]; users: { email: string; roles: string[] }[] }>(
    `/p/${encodeURIComponent(project)}/roles`,
  )
}

// Parse the raw config YAML and return the typed object.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function parseConfigYaml(raw: string): any {
  return yaml.load(raw) ?? {}
}

// Serialise a config object back to YAML string.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function dumpConfigYaml(obj: any): string {
  return yaml.dump(obj, { lineWidth: -1, quotingType: '"' })
}

interface RawUserEntry {
  email?: string
  roles?: string | string[]
  linux_user?: string
}

// Fetch and parse user bindings (with linux_user) from the project config YAML.
// The /roles endpoint omits linux_user, so we derive it from the raw config.
export async function getUserBindings(project: string): Promise<UserBinding[]> {
  const { raw } = await getConfig(project)
  const parsed = parseConfigYaml(raw) as { users?: RawUserEntry[] } | null
  const users = parsed?.users
  if (!Array.isArray(users)) return []
  return users.map((u) => ({
    email: u.email ?? '',
    roles: Array.isArray(u.roles) ? u.roles : u.roles ? [String(u.roles)] : [],
    linux_user: u.linux_user || undefined,
  }))
}
