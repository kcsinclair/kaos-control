// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from './client'
import type { IdeaConverseResponse, IdeaGenerateResponse } from '@/types/api'

export function generateIdea(
  project: string,
  input: string,
  type?: 'idea' | 'defect' | 'doc',
  sourceLineage?: string,
  sourcePath?: string,
): Promise<IdeaGenerateResponse> {
  return api.post<IdeaGenerateResponse>(`/p/${encodeURIComponent(project)}/ideas/generate`, {
    input,
    ...(type !== undefined ? { type } : {}),
    ...(sourceLineage !== undefined ? { source_lineage: sourceLineage } : {}),
    ...(sourcePath !== undefined ? { source_path: sourcePath } : {}),
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
