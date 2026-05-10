// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from '@/api/client'

export async function fetchVersion(): Promise<string> {
  try {
    const data = await api.get<{ version: string }>('/version')
    return data.version ?? 'unknown'
  } catch {
    return 'unknown'
  }
}
