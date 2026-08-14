// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from './client'

export interface PromoteArchitectureResult {
  promoted_architecture: string
  promoted_tech_stack: string
  archived: string[]
}

export interface CreateAdrResult {
  path: string
  number: number
}

// promoteArchitecture promotes a chosen catalog architecture + tech-stack pair
// to the project's architecture zone. Exposed here for the onboarding wizard
// ([[onboarding-architecture-selection]]) to reuse — this plan does not build
// the wizard screens.
export function promoteArchitecture(
  project: string,
  data: { architecturePath: string; techStackPath: string },
): Promise<PromoteArchitectureResult> {
  return api.post<PromoteArchitectureResult>(
    `/p/${encodeURIComponent(project)}/architecture/promote`,
    { architecture_path: data.architecturePath, tech_stack_path: data.techStackPath },
  )
}

export function nextAdrNumber(project: string): Promise<number> {
  return api
    .get<{ number: number }>(`/p/${encodeURIComponent(project)}/architecture/adrs/next`)
    .then((r) => r.number)
}

export function createAdr(
  project: string,
  data: { slug: string; title: string; status: string; body: string },
): Promise<CreateAdrResult> {
  return api.post<CreateAdrResult>(`/p/${encodeURIComponent(project)}/architecture/adrs`, data)
}
