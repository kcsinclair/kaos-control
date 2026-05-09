// SPDX-License-Identifier: AGPL-3.0-or-later

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
  priority?: string
  parent?: string
  labels?: string[]
  depends_on?: string[]
  blocks?: string[]
  related_to?: string[]
  members?: string[]
  release?: string
  sprint?: string
  assignees?: ArtifactAssignee[]
  created?: string
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
  created: string
}

export interface ArtifactDetail extends ArtifactRow {
  body: string
  body_html: string
  file_sha: string
}

export interface LockRow {
  lineage: string
  holder: string
  kind: string
  acquired_at: string
  last_heartbeat: string
}

export interface OllamaInstance {
  name: string
  base_url: string
  api_key?: string
}

export interface OllamaHealthResponse {
  ok: boolean
  latency_ms?: number
  error?: string
}

export interface OllamaModel {
  name: string
  size: number
}

export interface AgentSummary {
  name: string
  roles: string[]
  driver: string
  model?: string
  active_status?: string
  allowed_write_paths?: string[]
  ollama_instance?: string
  ollama_endpoint?: string
}

export interface AgentRunRow {
  run_id: string
  agent_name: string
  role: string
  target_path: string
  started_at: string
  finished_at?: string
  status: string
  exit_code?: number
  stderr_tail: string
  artifacts_produced: string[]
}

export interface ArtifactFilter {
  stage?: string
  status?: string
  label?: string
  lineage?: string
  type?: string
  priority?: string
  release?: string
  q?: string
  sort?: string
  limit?: number
  offset?: number
}

export interface LineageSummary {
  lineage: string
  members: string[]
  statuses: Record<string, number>
}

export interface GraphNode {
  id: string
  title: string
  type: string
  status: string
  stage: string
  lineage: string
  slug: string
  index: number
  priority?: string
  labels?: string[]
  /** True for synthetic nodes such as the Backlog root in the roadmap graph */
  synthetic?: boolean
}

export interface GraphEdge {
  source: string
  target: string
  kind: string
  /** Human-readable duration label for timeline edges (e.g. "2 weeks") */
  label?: string
}

export interface GraphData {
  nodes: GraphNode[]
  edges: GraphEdge[]
}

export interface GraphFilter {
  types?: string[]
  statuses?: string[]
  lineages?: string[]
  labels?: string[]
  priorities?: string[]
}

export interface ParseErrorRow {
  path: string
  message: string
}

export interface IdeaGenerateResponse {
  slug: string
  title: string
  labels: string[]
  body: string
  frontmatter: Record<string, unknown>
  target_dir: string
}

export interface IdeaConverseResponse {
  session_id: string
  reply: string
  status: 'conversing' | 'proposed' | 'created'
  preview: { frontmatter: Record<string, unknown>; body: string } | null
  artifact_path: string | null
}

export interface FeedEvent {
  id: number
  event_type: string
  timestamp: number
  actor: string
  artifact_path?: string
  run_id?: string
  summary: string
  payload_json?: string
}

export interface FeedResponse {
  events: FeedEvent[]
  next_cursor: number | null
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
  | 'feed.new'
  | 'pipeline.run.started'
  | 'pipeline.step.started'
  | 'pipeline.step.output'
  | 'pipeline.step.completed'
  | 'pipeline.run.completed'
  | 'scheduler.job.started'
  | 'scheduler.job.completed'
  | 'release.created'
  | 'release.updated'
  | 'release.deleted'

export interface ScheduleSpec {
  type: 'cron' | 'interval' | 'once'
  expression: string
}

export interface Precondition {
  type: 'after_job' | 'file_exists' | 'http_ok' | 'shell'
  value: string
}

export type RunStatus = 'running' | 'success' | 'failure' | 'timeout' | 'skipped'

export interface SchedulerJob {
  name: string
  target_type: 'agent' | 'shell'
  target: string
  args?: Record<string, string>
  schedule: ScheduleSpec
  preconditions?: Precondition[]
  enabled: boolean
  priority: number
  timeout_sec: number
  next_run_at?: string
  last_run_status?: RunStatus
  last_run_at?: string
  created_at: string
  updated_at: string
}

export interface SchedulerRun {
  id: number
  job_name: string
  start_time: string
  end_time?: string
  status: RunStatus
  log_path?: string
}

export interface WsEvent {
  type: WsEventType
  payload: Record<string, unknown>
}

export const TERMINAL_STATUSES = ['done', 'rejected', 'abandoned'] as const
export type TerminalStatus = typeof TERMINAL_STATUSES[number]
