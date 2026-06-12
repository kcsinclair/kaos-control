// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from './client'
import type { AgentUsageReport, AgentUsageFilter } from '@/types/api'

export function getAgentUsageReport(
  project: string,
  filter: AgentUsageFilter,
): Promise<AgentUsageReport> {
  const q = new URLSearchParams()
  if (filter.from) q.append('from', filter.from)
  if (filter.to) q.append('to', filter.to)
  if (filter.bucket) q.append('bucket', filter.bucket)
  q.append('tz', filter.tz ?? Intl.DateTimeFormat().resolvedOptions().timeZone)
  for (const a of filter.agent ?? []) q.append('agent', a)
  for (const s of filter.status ?? []) q.append('status', s)
  return api.get<AgentUsageReport>(
    `/p/${encodeURIComponent(project)}/reports/agent-usage?${q.toString()}`,
  )
}
