// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from './client'
import type {
  WizardAnswer,
  WizardCommitRequest,
  WizardCommitResult,
  WizardRecommendResponse,
  WizardScaffoldAvailability,
  WizardScaffoldRunResult,
  WizardStartResponse,
  WizardState,
  CatalogItem,
  ScaffoldChoice,
} from '@/types/api'

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

// Architecture Wizard ([[onboarding-architecture-selection]]) — see
// lifecycle/frontend-plans/onboarding-architecture-selection-4-fe.md Milestone 1.

export function getWizard(project: string): Promise<WizardStartResponse> {
  return api.get<WizardStartResponse>(`/p/${encodeURIComponent(project)}/architecture/wizard`)
}

export function recommend(
  project: string,
  answers: WizardAnswer[],
): Promise<WizardRecommendResponse> {
  return api.post<WizardRecommendResponse>(
    `/p/${encodeURIComponent(project)}/architecture/wizard/recommend`,
    { answers },
  )
}

export function listStacks(
  project: string,
  architecture: string,
  language?: string,
): Promise<CatalogItem[]> {
  const params = new URLSearchParams({ architecture })
  if (language) params.set('language', language)
  return api
    .get<{ stacks: CatalogItem[] }>(
      `/p/${encodeURIComponent(project)}/architecture/wizard/stacks?${params.toString()}`,
    )
    .then((r) => r.stacks)
}

export function saveWizardState(project: string, state: WizardState): Promise<void> {
  return api
    .put<{ saved: boolean }>(`/p/${encodeURIComponent(project)}/architecture/wizard/state`, state)
    .then(() => undefined)
}

export function discardWizardState(project: string): Promise<void> {
  return api
    .delete<{ cleared: boolean }>(`/p/${encodeURIComponent(project)}/architecture/wizard/state`)
    .then(() => undefined)
}

export function commitWizard(
  project: string,
  payload: WizardCommitRequest,
): Promise<WizardCommitResult> {
  return api.post<WizardCommitResult>(
    `/p/${encodeURIComponent(project)}/architecture/wizard/commit`,
    payload,
  )
}

export function getScaffold(
  project: string,
  architecture: string,
  techStack: string,
): Promise<WizardScaffoldAvailability> {
  const params = new URLSearchParams({ architecture, tech_stack: techStack })
  return api.get<WizardScaffoldAvailability>(
    `/p/${encodeURIComponent(project)}/architecture/wizard/scaffold?${params.toString()}`,
  )
}

export function runScaffold(
  project: string,
  architecture: string,
  techStack: string,
  choices: ScaffoldChoice[],
): Promise<WizardScaffoldRunResult> {
  return api.post<WizardScaffoldRunResult>(
    `/p/${encodeURIComponent(project)}/architecture/wizard/scaffold`,
    { architecture, tech_stack: techStack, choices },
  )
}
