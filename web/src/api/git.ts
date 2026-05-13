// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from './client'
import type { GitStatusResponse } from '@/types/api'

export function fetchGitStatus(project: string): Promise<GitStatusResponse> {
  return api.get<GitStatusResponse>(`/p/${encodeURIComponent(project)}/git/status`)
}
