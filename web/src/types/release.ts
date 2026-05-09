// SPDX-License-Identifier: AGPL-3.0-or-later

export interface Release {
  id: number
  name: string
  status: 'planned' | 'active' | 'shipped'
  start_date: string | null
  end_date: string | null
  created_at: string
  updated_at: string
}

export interface ReleaseDetail extends Release {
  idea_count: number
  defect_count: number
}

export interface CreateReleasePayload {
  name: string
  status: 'planned' | 'active' | 'shipped'
  start_date?: string | null
  end_date?: string | null
}

export interface UpdateReleasePayload {
  name?: string
  status?: 'planned' | 'active' | 'shipped'
  start_date?: string | null
  end_date?: string | null
}
