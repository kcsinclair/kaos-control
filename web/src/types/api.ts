export interface User {
  email: string
  display_name: string
  created_at?: string
}

export interface MeResponse {
  email: string
  display_name: string
  roles: Record<string, string[]>
}

export interface ProjectSummary {
  name: string
  description: string
  path: string
}

export interface ApiErrorBody {
  code: string
  message: string
}
