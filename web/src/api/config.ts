// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from './client'
import yaml from 'js-yaml'

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
