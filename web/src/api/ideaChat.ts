// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from './client'
import type { IdeaConverseResponse, IdeaGenerateResponse } from '@/types/api'

export function generateIdea(
  project: string,
  input: string,
  type?: 'idea' | 'defect',
): Promise<IdeaGenerateResponse> {
  return api.post<IdeaGenerateResponse>(`/p/${encodeURIComponent(project)}/ideas/generate`, {
    input,
    ...(type !== undefined ? { type } : {}),
  })
}

export function converseIdea(
  project: string,
  sessionId: string | null,
  message: string,
): Promise<IdeaConverseResponse> {
  return api.post<IdeaConverseResponse>(`/p/${encodeURIComponent(project)}/ideas/converse`, {
    session_id: sessionId,
    message,
  })
}
