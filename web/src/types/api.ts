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
  owner: string
  initialised: boolean
  directivesMigrationAvailable?: boolean
}

/** One AGENTS.md/CLAUDE.md/GEMINI.md write attempt (mirrors internal/directives.FileWrite). */
export interface DirectiveFileWrite {
  path: string
  created: boolean
  changed: boolean
  skipped: boolean
  /** Set when an existing file lost its managed-region markers and the whole-file
   *  replacement was withheld pending force — a human confirms before it's applied. */
  diff?: string
}

/** Report from POST .../migrate-directives or .../directives/refresh (mirrors
 *  internal/directives.GenerateResult). */
export interface GenerateResult {
  files: DirectiveFileWrite[]
  disabledAgents?: string[]
  skipped?: string[]
}

export type ProjectDetail = ProjectSummary

export interface CreateProjectPayload {
  name: string
  mode: 'existing' | 'new'
  description?: string
  owner?: string
  /** Existing-mode target path. */
  path?: string
  /** New-mode parent directory. */
  parent?: string
  /** New-mode target directory name (distinct from the project name). */
  dirName?: string
}

export interface CreateProjectResult extends ProjectSummary {
  resolvedPath: string
  created?: string[]
  alreadyInitialised: boolean
  partialCompletion: boolean
}

export interface UpdateProjectPayload {
  description?: string
  owner?: string
  path?: string
}

export interface UserBinding {
  email: string
  roles: string[]
  linux_user?: string
}

export interface CheckDirectoryRequest {
  mode: 'existing' | 'new'
  path?: string
  parent?: string
  name?: string
}

export interface CheckDirectoryResult {
  // Existing-mode fields.
  exists: boolean
  isDir: boolean
  writable: boolean
  initialised: boolean

  // New-mode fields.
  parentExists: boolean
  parentWritable: boolean
  nameValid: boolean
  targetExists: boolean

  resolvedPath: string
  reason?: string
}

export interface InitProjectResult {
  created: string[]
  git_initialised: boolean
  git_commands?: string[]
}

export interface ApiErrorBody {
  code: string
  message: string
}

/** One config self-repair applied in-memory when the project config loaded. */
export interface RepairNote {
  agent: string
  template_key: string
  reason: string
}

export interface ConfigHealthResponse {
  repairs: RepairNote[]
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
  summary?: string
  members?: string[]
  /** Functional area a feature belongs to; the Features view groups by it. */
  function?: string
  release?: string
  sprint?: string
  assignees?: ArtifactAssignee[]
  created?: string
  rice_reach?: number
  rice_impact?: number
  rice_confidence?: number
  rice_effort?: number
}

export interface ArtifactRow {
  path: string
  rel_path: string
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
  agent_run_count: number
  active_agent_status?: 'running' | 'queued'
  rice_score?: number
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

export interface ProviderConfig {
  name: string
  base_url: string
  driver: string
  api_key?: string
  has_api_key?: boolean
  extra_headers?: Record<string, string>
}

export type Provider = ProviderConfig

export interface ProviderHealth {
  ok: boolean
  latency_ms?: number
  error?: string
}

export interface DiscoveredModel {
  id: string
  name: string
  owned_by?: string
  supports_tools?: boolean
  supported_parameters?: string[]
}

export type ProviderModel = DiscoveredModel

export interface ProviderProbeResult {
  ok: boolean
  error?: string
  latency_ms?: number
  message?: string
  models: DiscoveredModel[]
}

export interface AgentConfig {
  name: string
  role?: string | string[]
  roles?: string[]
  /** driver: 'claude-code-cli' | 'claude-mediated' | 'codex-cli' | 'gemini' | 'gemini-cli' | 'inline' | 'claude-env' | 'openai-compatible' */
  driver: string
  provider?: string
  model?: string
  max_tool_iterations?: number
  allowed_write_paths?: string[]
  timeout_minutes?: number
  git_identity?: { name?: string; email?: string }
  prompt_templates?: Record<string, string>
}

export interface AgentSummary {
  name: string
  roles: string[]
  /** driver: 'claude-code-cli' | 'claude-mediated' | 'codex-cli' | 'gemini' | 'gemini-cli' | 'inline' | 'claude-env' | 'openai-compatible' */
  driver: string
  provider?: string
  model?: string
  max_tool_iterations?: number
  active_status?: string
  source_types?: string[]
  allowed_write_paths?: string[]
  timeout_minutes?: number
  git_identity?: { name?: string; email?: string }
  prompt_templates?: Record<string, string>
  done_on_success?: boolean
  endpoint?: string
  shell_command?: string
  observe_only?: boolean
  bash_allowlist?: string[]
  bash_denylist?: string[]
  on_denial?: string
}

export interface DenialRecord {
  tool_name: string
  path?: string
  command?: string
  reason: string
  rule: string
}

export interface RunSuiteSummary {
  name: string
  total: number
  passed: number
  failed: number
  skipped: number
  elapsed: number
}

export interface RunSummary {
  suites: RunSuiteSummary[]
  defectsCreated: number
  duplicatesFound: number
  orphanedFailures: number
  coverageGaps: string[]
  elapsed: number
}

/**
 * Structured failure-reason codes (mirrors internal/agent.FailureReason* and
 * the precheck-specific codes recorded by killAndFail). Loosely typed
 * (`| string`) so an unclassified/future backend code still type-checks
 * instead of narrowing the whole union.
 */
export type FailureReason =
  | 'tools_unsupported'
  | 'model_not_found'
  | 'model_unloaded'
  | 'endpoint_unreachable'
  | 'context_window_exceeded'
  | 'turn_token_ceiling'
  | 'max_iterations_reached'
  | 'auth_error'
  | 'timeout'
  | 'permission_mode_default'
  | 'precheck_timeout'
  | 'precheck_unknown'
  | (string & {})

/** Live model-loading/generation phase surfaced by warmup WS events (local-model-operability FR-5). */
export type WarmupState = 'model_loading' | 'warming_up' | 'generating'

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
  /** Stable reason code on failure; null on success / pending. */
  failure_reason?: FailureReason | null
  /** Set when failure_reason === 'permission_mode_default'. */
  observed_permission_mode?: string | null
  /** Set on precheck-related failures; up to ~5 short remediation lines. */
  remediation?: string[] | null
  /** Contextual failure details (already secret-masked server-side); shape varies by failure_reason. */
  error_details?: Record<string, unknown> | null
  /** Tool calls denied by the mediated driver permission hooks. */
  denied_tool_calls?: DenialRecord[] | null
  /** Populated for test-runner agent runs. */
  run_summary?: RunSummary
  /** Time to first token in milliseconds (recorded for streaming drivers). */
  ttft_ms?: number | null
  /**
   * Live warmup/loading phase, populated client-side from agent.status /
   * agent.progress WS events (never persisted server-side — cleared once the
   * run reaches a terminal status).
   */
  warmup_state?: WarmupState | null
  warmup_message?: string | null
}

export interface ToolCallRecord {
  id: string
  name: string
  arguments: string
  result?: string
  is_recovered?: boolean
}

export interface RunTurn {
  turn_number: number
  role: 'system' | 'user' | 'assistant' | 'tool'
  content?: string
  tool_calls?: ToolCallRecord[]
  is_recovered?: boolean
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
  /** Restrict to artefacts that are `blocked` with a non-empty `## Open Questions` section. */
  awaiting_answers?: boolean
}

export interface OpenQuestion {
  index: number
  text: string
  answer: string
}

export interface OpenQuestionsResponse {
  heading: string
  format: string
  questions: OpenQuestion[]
  can_resolve: boolean
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
  /** summary: frontmatter field, shown in the graph tooltip when present. */
  summary?: string
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

/**
 * Relationship kinds the architecture map knows how to style distinctly.
 * Loose string-compatible (not a strict enum) so unknown/future kinds still
 * type-check and degrade to generic edge styling (NFR-5).
 */
export type ArchitectureEdgeKind =
  | 'related'
  | 'evolves_into'
  | 'alternative_to'
  | 'composed_with'
  | 'related_to'
  | (string & {})

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

export interface GitStatusResponse {
  available: boolean
  branch?: string
  dirty?: boolean
  head_sha?: string
  head_message?: string
  head_author?: string
  head_when?: string
}

/**
 * Payload for the `agent.status` WS event (mirrors
 * Manager.preflightModelAvailability's broadcast verbatim — local-model-
 * operability FR-2). Broadcast before the run's `agent.started` event, so it
 * carries `target_path` rather than `run_id`; consumers match on that.
 */
export interface AgentStatusEvent {
  agent: string
  target_path: string
  state: 'model_loading'
  details?: string
}

export interface PermissionDecision {
  run_id: string
  tool_name: string
  target_path?: string
  command?: string
  decision: 'allow' | 'deny'
  reason: string
  policy_rule: string
  timestamp: string
}

export type WsEventType =
  | 'file.changed'
  | 'doc.changed'
  | 'artifact.indexed'
  | 'git.committed'
  | 'git.status'
  | 'lock.acquired'
  | 'lock.released'
  | 'agent.started'
  | 'agent.progress'
  | 'agent.status'
  | 'agent.finished'
  | 'agent.failed'
  | 'agent.permission'
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
  | 'release.changed'
  | 'release.deleted'
  | 'queue.added'
  | 'queue.started'
  | 'queue.finished'
  | 'queue.skipped'
  | 'queue.cancelled'
  | 'queue.paused'
  | 'queue.resumed'
  | 'agent.quota_status'
  | 'provider.switched'
  | 'provider.restored'
  | 'provider.primary_recovered'
  | 'config.reloaded'

/** Payload for the `agent.quota_status` WS event (mirrors the backend wire shape verbatim). */
export interface QuotaStatusPayload {
  run_id: string
  bucket: 'five_hour' | 'weekly' | 'unknown'
  status: 'allowed' | 'warning' | 'rejected' | 'unknown'
  /** RFC3339 UTC; omitted when unknown. */
  resets_at?: string
  overage_available: boolean
  overage_disabled_reason?: string
}

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

// Mirrors internal/architecture's architectureDir constant.
const ARCHITECTURE_DIR = 'lifecycle/architecture'

/**
 * True when a row belongs to the architecture *zone* — the whole
 * `lifecycle/architecture/` tree (catalog candidates, the chosen
 * architecture/stack, ADRs, standards, the summary, and the archive alike) —
 * rather than a lifecycle work item (FR-9). Reference material, not
 * something to triage on the list/board, so it's hidden by default behind
 * the "show architecture inline" escape hatch (FR-9a). Role-based (by path),
 * never `type:`-based, since the summary/standards deliberately share
 * `type: doc` with ordinary docs. The `catalog` label check is a defensive
 * fallback for any catalog-labelled artifact that somehow lives outside the
 * zone path. See [[architecture-overview-view]].
 */
export function isArchitectureZone(row: Pick<ArtifactRow, 'path' | 'frontmatter'>): boolean {
  const path = row.path ?? ''
  if (path === ARCHITECTURE_DIR || path.startsWith(ARCHITECTURE_DIR + '/')) return true
  const labels = row.frontmatter?.labels ?? []
  if (labels.includes('catalog')) return true
  return false
}

/** @deprecated use {@link isArchitectureZone} — kept for callers not yet migrated. */
export const isCatalogMaterial = isArchitectureZone

export interface RunResultUsage {
  input_tokens: number
  cache_creation_input_tokens: number
  cache_read_input_tokens: number
  output_tokens: number
}

export interface RunResult {
  subtype: string
  /** Authoritative success/failure flag. Claude Code emits subtype:"success"
   *  alongside is_error:true (e.g. a mid-response API error), so UI must prefer
   *  this over `subtype` — see RunSummaryCard. */
  is_error?: boolean
  /** Terminal message; carries the reason when is_error is true. */
  result?: string
  total_cost_usd: number
  duration_ms: number
  duration_api_ms: number
  num_turns: number
  usage: RunResultUsage
  permission_denials: unknown[]
  session_id: string
}

export interface AgentUsageBucketPoint {
  bucket_start: string
  run_count: number
  success_count: number
  failure_count: number
  mean_duration_ms: number | null
  mean_cost_usd: number | null
  mean_output_tokens_per_second: number | null
  mean_ttft_ms: number | null
  cache_hit_ratio: number | null
}

export interface AgentUsageGroupSummary {
  run_count: number
  success_count: number
  failure_count: number
  metrics_unavailable_count: number
  total_cost_usd: number
  total_input_cost_usd: number
  total_output_cost_usd: number
  total_duration_ms: number
  total_input_tokens: number
  total_cache_creation_tokens: number
  total_cache_read_tokens: number
  total_output_tokens: number
  mean_duration_ms: number | null
  median_duration_ms: number | null
  p95_duration_ms: number | null
  mean_cost_usd: number | null
  mean_output_tokens_per_second: number | null
  mean_ttft_ms: number | null
  p95_ttft_ms: number | null
  cache_hit_ratio: number | null
}

export interface AgentUsageSummary {
  overall: AgentUsageGroupSummary
  per_model: (AgentUsageGroupSummary & { model: string })[]
  per_agent: (AgentUsageGroupSummary & { agent_name: string })[]
}

export interface AgentUsageReport {
  summary: AgentUsageSummary
  series: AgentUsageBucketPoint[]
  series_by_model: Record<string, AgentUsageBucketPoint[]>
  series_by_agent?: Record<string, AgentUsageBucketPoint[]>
}

export interface AgentUsageFilter {
  from?: string
  to?: string
  agent?: string[]
  status?: string[]
  bucket?: 'hour' | 'day' | 'week'
  tz?: string
}

// Architecture Wizard — mirrors internal/architecture and internal/config
// (config.WizardQuestion / config.WizardOption, architecture.CatalogItem,
// architecture.Answer, architecture.Recommendation, architecture.WizardState,
// architecture.ScaffoldStep/ScaffoldChoice/ScaffoldResult). See
// lifecycle/frontend-plans/onboarding-architecture-selection-4-fe.md Milestone 1.

export interface WizardOption {
  value: string
  label: string
  labels?: string[]
  hard?: boolean
}

export interface WizardQuestion {
  id: string
  prompt: string
  kind: 'hard' | 'soft' | 'language'
  options?: WizardOption[]
}

export interface CatalogItem {
  path: string
  slug: string
  title: string
  summary: string
  type: 'architecture' | 'tech-stack'
  labels: string[]
  related_to: string[]
  pros?: string[]
  cons?: string[]
}

export interface WizardAnswer {
  question_id: string
  value: string
}

export interface WizardRecommendation {
  item: CatalogItem
  score: number
  why: string[]
}

export interface WizardPriorRun {
  detected: boolean
  architecture?: string
  tech_stack?: string
  adr_path?: string
  summary_path?: string
}

export interface WizardState {
  path: 'browse' | 'guided'
  answers: WizardAnswer[]
  chosen_architecture?: string
  chosen_tech_stack?: string
  step: string
  updated_unix: number
}

export interface WizardStartResponse {
  questions: WizardQuestion[]
  default_architecture: string
  prior_run: WizardPriorRun
  resumable_state: WizardState | null
}

export interface WizardRecommendResponse {
  recommendations: WizardRecommendation[]
  dropped_constraints: string[]
}

// BreakingReq/QAPair mirror internal/architecture/summary.go, which has no
// json tags — Go's default encoding uses the exported field name verbatim.
export interface WizardBreakingReq {
  Label: string
  Requirement: string
  Mapping: string
}

export interface WizardQAPair {
  Question: string
  Answer: string
}

export interface WizardCommitRequest {
  architecture_path: string
  tech_stack_path: string
  answers: WizardAnswer[]
  breaking_requirements: WizardBreakingReq[]
  qa: WizardQAPair[]
}

export interface WizardCommitResult {
  promoted_architecture: string
  promoted_tech_stack: string
  archived: string[]
  adr_path: string
  superseded_adr_path: string
  summary_path: string
}

export interface ScaffoldNameField {
  key: string
  label: string
  default_value: string
}

export interface ScaffoldStep {
  key: string
  title: string
  description: string
  name_fields?: ScaffoldNameField[]
  // Optional so a no-scaffolder / older response still type-checks;
  // treated as false when absent (NFR-4). See
  // lifecycle/frontend-plans/wizard-skip-scaffolding-4-fe.md Milestone 1.
  present?: boolean
}

export interface ScaffoldChoice {
  step_key: string
  values?: Record<string, string>
  use_defaults: boolean
  // Required — every emitted choice states its selection explicitly (OQ-2).
  selected: boolean
}

export interface ScaffoldResult {
  applied: string[]
  skipped: string[]
  // committed is true when kaos-control auto-committed the applied files (a
  // repo it created). git_commands, for a pre-existing user repo, holds the
  // git add/commit to run to track them.
  committed?: boolean
  git_commands?: string[]
}

export interface WizardScaffoldAvailability {
  available: boolean
  steps?: ScaffoldStep[]
  message?: string
}

export interface WizardScaffoldRunResult {
  available: boolean
  result?: ScaffoldResult
}

// Architecture Overview — mirrors internal/architecture/overview.go (Overview,
// OverviewItem, Role). See [[architecture-overview-view]].

export type CatalogRole =
  | 'catalog'
  | 'chosen-architecture'
  | 'chosen-stack'
  | 'summary'
  | 'standard'
  | 'adr'
  | 'archive'

export interface OverviewItem {
  path: string
  title: string
  status: string
  type: string
  created?: string
  catalog_role: CatalogRole
}

export interface ArchitectureOverview {
  has_chosen_architecture: boolean
  chosen_architecture: OverviewItem | null
  chosen_stack: OverviewItem | null
  summary: OverviewItem | null
  standards: OverviewItem[]
  adrs: OverviewItem[]
  archive: OverviewItem[]
  catalog: OverviewItem[]
}
