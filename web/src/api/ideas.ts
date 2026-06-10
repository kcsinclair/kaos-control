// SPDX-License-Identifier: AGPL-3.0-or-later

import { api, ApiError } from './client'

export interface TriageResponse {
  run_id: string
}

export interface TriageError {
  error: string
  reason?: string
}

export async function triageIdea(project: string, slug: string): Promise<TriageResponse> {
  return api.post<TriageResponse>(
    `/p/${encodeURIComponent(project)}/ideas/${encodeURIComponent(slug)}/triage`,
  )
}
