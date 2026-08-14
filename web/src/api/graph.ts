// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from './client'
import type { GraphData } from '@/types/api'

export function getGraph(project: string, includeReleases?: boolean) {
  const url = includeReleases
    ? `/p/${encodeURIComponent(project)}/graph?include_releases=true`
    : `/p/${encodeURIComponent(project)}/graph`
  return api.get<GraphData>(url)
}

// getArchitectureMap loads the read-only architecture relationship map: nodes
// are the catalog's `type: architecture` artifacts, edges are their typed or
// generic relationships. Passing stackFor additionally includes that
// architecture's related_to tech-stack ring (FR-8).
export function getArchitectureMap(project: string, stackFor?: string) {
  const url = stackFor
    ? `/p/${encodeURIComponent(project)}/architecture-map?stack_for=${encodeURIComponent(stackFor)}`
    : `/p/${encodeURIComponent(project)}/architecture-map`
  return api.get<GraphData>(url)
}
