// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from './client'
import type { GenerateResult } from '@/types/api'

export function migrateDirectives(project: string, opts: { force: boolean }) {
  return api.post<GenerateResult>(
    `/projects/${encodeURIComponent(project)}/migrate-directives`,
    { force: opts.force },
  )
}

export function refreshDirectives(project: string, opts: { force: boolean }) {
  return api.post<GenerateResult>(
    `/p/${encodeURIComponent(project)}/directives/refresh`,
    { force: opts.force },
  )
}
