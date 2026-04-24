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

export interface ArtifactAssignee {
  role: string
  who: string
}

export interface ArtifactFrontmatter {
  title: string
  type: string
  status: string
  lineage: string
  parent?: string
  labels?: string[]
  depends_on?: string[]
  blocks?: string[]
  related_to?: string[]
  members?: string[]
  release?: string
  sprint?: string
  assignees?: ArtifactAssignee[]
}

export interface ArtifactRow {
  path: string
  slug: string
  lineage: string
  index: number
  stage: string
  type: string
  status: string
  title: string
  frontmatter: ArtifactFrontmatter
  mtime: string
}

export interface ArtifactDetail extends ArtifactRow {
  body: string
  body_html: string
}

export interface ArtifactFilter {
  stage?: string
  status?: string
  label?: string
  lineage?: string
  type?: string
  limit?: number
  offset?: number
}

export interface LineageSummary {
  lineage: string
  members: string[]
  statuses: Record<string, number>
}

export type WsEventType =
  | 'file.changed'
  | 'artifact.indexed'
  | 'git.committed'
  | 'lock.acquired'
  | 'lock.released'
  | 'agent.started'
  | 'agent.progress'
  | 'agent.finished'
  | 'agent.failed'

export interface WsEvent {
  type: WsEventType
  payload: Record<string, unknown>
}
