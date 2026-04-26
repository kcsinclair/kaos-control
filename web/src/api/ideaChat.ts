import { api } from './client'
import type { IdeaConverseResponse } from '@/types/api'

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
