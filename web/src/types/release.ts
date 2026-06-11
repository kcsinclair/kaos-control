// SPDX-License-Identifier: AGPL-3.0-or-later

export type ReleaseStatus = 'planned' | 'active' | 'shipped' | 'unscheduled'

export interface Release {
  id: number
  slug: string
  name: string
  status: ReleaseStatus
  start_date: string | null
  end_date: string | null
  file_path: string
  created_at: string
  updated_at: string
}

export interface ReleaseDetail extends Release {
  idea_count: number
  defect_count: number
}

export interface CreateReleasePayload {
  name: string
  status: ReleaseStatus
  start_date?: string | null
  end_date?: string | null
}

export interface UpdateReleasePayload {
  name?: string
  status?: ReleaseStatus
  start_date?: string | null
  end_date?: string | null
  updated_at?: string
}
