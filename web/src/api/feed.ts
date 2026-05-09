// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from './client'
import type { FeedResponse } from '@/types/api'

export function fetchFeed(
  project: string,
  params?: { limit?: number; before?: number; types?: string },
): Promise<FeedResponse> {
  const p = new URLSearchParams()
  if (params?.limit !== undefined) p.set('limit', String(params.limit))
  if (params?.before !== undefined) p.set('before', String(params.before))
  if (params?.types) p.set('types', params.types)
  const qs = p.toString()
  return api.get<FeedResponse>(
    `/p/${encodeURIComponent(project)}/feed${qs ? '?' + qs : ''}`,
  )
}
