# kaos-control — Features

A snapshot of what kaos-control does today. Every feature listed is shipped
and reachable from the running binary — either as an HTTP API route, a Vue
view in the SPA, or both.

> **Status:** pre-1.0, active development. Working towards a first stable
> release on `KC-Release0`.
>
> **Roadmap:** kaos-control's own roadmap lives inside the running app
> rather than in this file. Open the **Roadmap** view for the Gantt-chart
> and 3D-graph views of upcoming releases and the artifacts assigned to
> them — both are live, dogfooded, and the canonical source for what's
> next.

---

## Lifecycle & artifacts

The core idea: every project's work — from initial sketch through to
shipped release — is markdown files on disk with YAML frontmatter, and
kaos-control indexes them into something queryable, browsable, and
agent-driveable.

- **Markdown-on-disk source of truth.** All artifacts live as `*.md` files
  under `<project>/lifecycle/`. The SQLite index is a cache; disk wins
  every reconciliation.
- **Stage directories.** `ideas/`, `requirements/`, `backend-plans/`,
  `frontend-plans/`, `test-plans/`, `tests/`, `defects/`, `releases/`,
  `sprints/`, `prototypes/`, `devops/` — one directory per artifact stage.
- **Type vocabulary.** `idea`, `requirement`, `plan-backend`,
  `plan-frontend`, `plan-test`, `test`, `prototype`, `defect`, `release`,
  `sprint`, plus configurable types per project.
- **Frontmatter parser.** Required fields: `title`, `type`, `status`,
  `lineage`. Optional: `priority`, `release`, `parent`, `assignees`,
  `labels`, `created`. Parse errors surface in the **Parse Errors**
  view rather than silently breaking.
- **Lineage tracking.** Every artifact in a chain shares a slug and
  carries a monotonic per-lineage index across stages
  (`login.md → login-2.md → login-3-be.md → login-4-fe.md`). Indexes
  never reuse, even after rejection — full history is preserved.
- **Live indexing.** `fsnotify` watcher with 150 ms debounce; external
  edits (e.g. from your text editor or an agent) re-index within
  milliseconds and broadcast `artifact.indexed` to connected clients.
- **Markdown editor.** CodeMirror 6 with frontmatter dropdowns for
  enum fields (status, type, role, who, priority), live preview via
  `markdown-it`, optional line-wrap toggle, and external-change
  detection (warns if the file changed under you).
- **Artifact list.** Filter by stage, status, type, label, priority,
  release, or full-text. Sortable on every column. Paginated. Show /
  hide done items toggle. Hide tests by default.
- **Inline status & priority changes.** Click the status pill in the
  list to transition without opening the editor. Same for priority.
- **Defect creation.** "New defect" button captures stack/log/repro
  in a single submit; auto-routes to the right developer role based
  on labels.

## Workflow & state machine

- **Status vocabulary.** `draft → clarifying → planning → in-development →
  in-qa → approved → done` plus `rejected`, `abandoned`, `blocked`.
- **Role-gated transitions.** Per-edge rules in
  `lifecycle/config.yaml` decide which roles may move which artifact
  type from which status to which. Type-aware (e.g. `test` artifacts
  have their own `approved → in-qa` cycle).
- **Plan-completion gate.** A requirement can only leave `planning` once
  every `required_plan` type for it has at least one approved artifact.
- **Product-owner bypass.** The `product-owner` role can take any
  transition between known states, for recovery / smoothing edge cases.
- **Self-transition guard.** Same-state transitions are rejected up
  front (no `draft → draft` no-ops).
- **Concurrency-safe transitions.** Per-path mutex + on-disk from-status
  re-check, so two parallel calls to the same artifact produce one
  advance, not two.
- **Auto-block on Open Questions.** Saving an artifact whose body has
  a populated `## Open Questions` H2 forces `status: blocked` and
  routes it to a `product-owner agent` assignee.
- **Lineage status checker.** A view that shows every lineage at the
  same time and one-click advances stale ones to their next valid
  state.
- **Allowed-targets API.** `GET /artifacts/.../allowed-targets` for
  per-user, per-status, per-type valid next-state lookups.

## Graph & visualisation

- **3D force graph.** Three.js + 3d-force-graph; nodes coloured by type,
  rings for priority + active status (in-development = green pulse,
  in-qa = amber pulse, etc.); zoom, pan, click-to-edit.
- **2D graph.** Cytoscape.js + fcose; layout selector
  (fcose / circle / breadthfirst / dagre); filter relayout uses the
  current layout; light/dark theme-aware palette.
- **Graph filters.** By stage, status, type, label, priority, release,
  or full-text search; matched nodes highlight, others dim, no relayout.
- **Graph show-toggles.** Show / hide releases overlay, show / hide
  tests, hide done.
- **Roadmap graph.** Time-ordered chain of releases with assigned
  artifacts shown as children; "Backlog" anchor for unassigned items;
  "Unscheduled" terminus for releases without dates.
- **Roadmap Gantt chart.** Releases as bars on a timeline; period
  options (daily / weekly / monthly) coming soon.

## Agents

- **Configured per role.** Each agent is bound to one or more roles
  with a focused prompt template, a sandboxed `allowed_write_paths`
  allowlist, optional model override, optional timeout.
- **Drivers.** `claude-code-cli` (default) shells out to `claude
  --dangerously-skip-permissions -p`, parsing stream-json events for
  live progress. `ollama` driver for local models.
- **Active-status lifecycle.** When an agent starts, the target's
  status moves to a configured `active_status` (e.g. `in-development`)
  and back to `done` on success — bundled into the agent's own commit.
- **Run history.** Every run persists to SQLite with status, exit
  code, stderr tail, artifacts produced, target, role, started/
  finished timestamps. Survives schema rebuilds (cache-resilient).
- **Live progress.** Stdout / stderr streams to the **Agents** page
  via WebSocket; expandable per-run detail with the last ~4 KB of
  stderr.
- **Lineage locks.** Every run acquires an exclusive lock on the
  target's lineage; concurrent runs on the same lineage are rejected.
  Stale locks reaped after 5 min of no heartbeat.
- **Crash recovery.** Runs left in `running` after a server restart
  are automatically marked `failed`.
- **Kill button.** SIGTERM the running agent process from the UI.
- **Agent launcher panels.** Per-agent cards on the Agents page show
  the current ready-work for that role, with one-click run.

## Idea capture

- **Conversational capture.** A chat-style UI that drives Claude
  through clarifying rounds and produces an idea artifact when ready
  (max 3 rounds before forcing a proposal).
- **Single-submit capture.** Direct "new idea" / "new defect" form
  with full preview before write.

## Releases & roadmap

- **First-class release records.** Stored in SQLite (not as markdown
  artifacts), with name, status, optional `start_date` / `end_date`.
- **Artifact assignment.** Artifacts carry an optional `release:`
  frontmatter field. Assigned artifacts show in the release detail
  view and on the roadmap graph as children of their release.
- **Release CRUD via REST + WS broadcast.** Create / update / delete
  / list with `release.created` / `.updated` / `.deleted` events.
- **Rename propagation.** Renaming a release auto-rewrites every
  assigned artifact's `release:` frontmatter and commits the change.
- **Reassign on delete.** Optional `?reassign_to=<id>` on delete
  moves the doomed release's artifacts onto another release.

## Dashboard

- **Summary cards.** Total work-items, in-progress, blocked, completed
  this week. Configurable `tracked_types` per project (defaults to
  `[ticket]`; set in `lifecycle/config.yaml`).
- **Status distribution pie.** Click a wedge to filter the artifacts
  list by that status.
- **Velocity bar chart.** Daily / weekly / monthly granularity, 90-day
  default lookback. Echarts. DataZoom appears for crowded ranges.
- **Activity feed widget.** Recent agent runs, transitions, defects
  raised, artifacts created — newest first, click to navigate.

## Kanban board

- **Configurable columns.** Per-project `kanban.columns` config maps
  statuses to columns (e.g. "In Progress" can collapse `clarifying`,
  `planning`, `in-development`, `in-qa`).
- **Card fields.** Title, type, priority, labels, age — configurable.
- **Column-aware testing board.** Separate Kanban view for tests with
  text filter and approved-count badge.

## DevOps pipelines

- **Declarative YAML.** One file per pipeline in `<project>/lifecycle/devops/`,
  with `name`, `type` (`build` / `deploy` / `release` / arbitrary),
  ordered `steps`. Each step has `name`, `command`, optional
  `description` and `timeout`.
- **Trigger from the UI.** Cards grouped by type; Run / Cancel /
  re-run with one click.
- **Live output streaming.** Per-step stdout / stderr to the browser
  via WebSocket: `pipeline.run.started → step.started → step.output*
  → step.completed → run.completed`.
- **Run log persistence.** NDJSON logs at
  `~/.kaos-control/devops/<project>/<run_id>.log`. Browseable
  post-mortem.
- **Role-gated.** Only `product-owner` / `devops` can trigger.
- **Cancellation + timeout.** SIGTERM running step; per-step timeouts
  honoured; failed step skips the rest.

## Scheduler

- **Cron-style job definitions.** Per-project SQLite-backed jobs
  with cron expressions, timeouts, target type (shell / agent),
  and a precondition expression.
- **Job CRUD UI.** List / create / edit / delete / pause / resume /
  trigger now. Run history per job. Live status via WebSocket.

## Ollama (local LLMs)

- **Multi-instance management.** Configure multiple Ollama endpoints
  app-wide (CRUD via `/api/ollama/instances`); per-instance health
  + model listing.
- **Agent driver.** Use any Ollama instance + model as an agent
  driver alongside Claude.

## Project feed

- **Live event log.** Every status transition, agent run, git
  commit, defect raised, artifact created — streamed to the **Feed**
  view via WebSocket. Configurable retention (`feed.retention_days`,
  `feed.max_events`).

## Multi-project

- **Project registry.** YAML files in `~/.kaos-control/projects/*.yaml`
  register each project (name, path, owner, description). The
  picker on first load lets you choose.
- **Per-project config.** Roles, users, agents, plan-gates, kanban
  layout, dashboard tracked types, scheduler defaults, and DevOps
  pipelines all live in `<project>/lifecycle/config.yaml`.
- **Per-project SQLite cache.** Each project gets its own
  `~/.kaos-control/data/<project>/index.db`; rebuilt from disk if
  the schema version changes.

## Auth & authorisation

- **Local accounts.** Argon2id-hashed passwords in `~/.kaos-control/data/auth.db`.
- **Bootstrap-friendly.** First user can be created without
  authentication; subsequent users require a logged-in session.
- **Session cookies + CSRF.** HTTP-only session cookie + matching
  CSRF token on every mutating request.
- **Per-project role bindings.** `users:` block in
  `lifecycle/config.yaml` binds each email to a list of roles
  (`product-owner`, `analyst`, `backend-developer`, `frontend-developer`,
  `test-developer`, `qa`, `reviewer`, `approver`, `devops`).

## Git integration

- **Auto-commit on every write.** Each artifact create / edit /
  transition / agent run produces a structured git commit
  (`transition(<lineage>): <from> → <to>`, `agent(<name>): run
  <id> [<status>]`, etc.).
- **Identity per actor.** Agents commit under their configured
  `git_identity`; user actions commit under the logged-in user's
  email.
- **History API.** `GET /artifacts/.../history` returns the commit
  chain for any artifact.
- **First-commit-date backfill.** Artifacts that lack `created:`
  frontmatter get a `created` value derived from git history; cached
  in the index.

## Operations

- **Single binary, embedded SPA.** Go server with `embed.FS` —
  one file to deploy.
- **WebSocket hub.** Per-project broadcast channel for indexed
  events, agent progress, pipeline output, scheduler ticks, locks.
- **Parse-error view.** Every YAML / frontmatter parse failure
  surfaces with file path + line + message; reload re-attempts.
- **App config.** Server listen address, TLS, auth method, projects
  directory, data directory, devops log directory all in
  `~/.kaos-control/config.yaml`.
- **Lock management.** Per-lineage editor / agent locks with
  heartbeat-based stale-reaper.

---

## Notes

- This is a snapshot of `main` at the time of writing. For the
  authoritative living state-of-the-project record see
  [plans/PROJECT_PLAN.md](plans/PROJECT_PLAN.md); for what's next,
  open the **Roadmap** view in the app.
- Marketing-tuned variants of this list can be derived from it later.
  None exist yet — this document is for real engineering work, not the
  launch page.
