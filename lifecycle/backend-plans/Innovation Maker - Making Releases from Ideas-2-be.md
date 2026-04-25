---
title: Backend Development Plan — kaos-control v1
type: plan-backend
status: done
lineage: innovation-maker
parent: requirements/Innovation Maker - Making Releases from Ideas-1.md
labels:
    - backend
    - go
    - v1
---

> Target implementer: developer agent (Sonnet). Produces a Go web application that serves the frontend, indexes artifacts, runs agents, talks to git, and exposes the authoritative REST + WebSocket API. All section numbers in the form §N.N refer to the parent requirements document.

## 1. Scope

### In scope
- Everything required for the v1 goals enumerated in §1 of the spec.
- Single static Go binary with embedded frontend (§12.3).
- REST API + WebSocket API that the frontend depends on.
- Artifact indexing pipeline, workflow state machine, agent runner, git integration, authentication.

### Out of scope (roadmap — do not implement)
- JIRA integration (§16).
- kaos-control as MCP server (§16).
- SSO / OIDC (§14.2).
- Agent drivers other than `claude-code-cli` (§7.2).
- Auto-triggered agents (§7.3).
- CRDT / real-time co-editing (§16).

## 2. Repository Layout

Single Go module at repo root. Frontend source lives under `web/` and is embedded into the Go binary at build time.

```
/
├── cmd/
│   └── kaos-control/
│       └── main.go              # entry point, CLI arg parsing, wiring
├── internal/
│   ├── artifact/                # parsing, validation, lineage, slug helpers
│   ├── index/                   # SQLite schema + index rebuild + live sync
│   ├── watcher/                 # fsnotify wrapper, debouncing, event fan-out
│   ├── git/                     # go-git wrapper + shell-out for PR ops
│   ├── workflow/                # state machine, transition authorisation
│   ├── agent/                   # runner, drivers, run tracking, kill/crash
│   ├── lock/                    # lineage lock manager
│   ├── auth/                    # argon2id, sessions, middleware
│   ├── config/                  # app + project config loaders
│   ├── http/                    # chi routes, handlers, websocket hub
│   └── project/                 # per-project services container
├── web/                         # Vite + Vue source (owned by the frontend plan)
│   └── dist/                    # build output, embedded via embed.FS
├── go.mod
├── go.sum
└── Makefile
```

Rationale: `internal/` for everything non-exported; frontend lives alongside so a single `go build` produces the shippable artifact (spec §12.3).

## 3. Dependencies

| Purpose | Module | Version pin |
|---|---|---|
| Router | `github.com/go-chi/chi/v5` | latest v5 |
| Markdown + frontmatter | `github.com/yuin/goldmark` + `go.abhg.dev/goldmark/frontmatter` | latest |
| YAML | `gopkg.in/yaml.v3` | latest |
| File watching | `github.com/fsnotify/fsnotify` | latest |
| WebSocket | `github.com/coder/websocket` (was `nhooyr.io/websocket`) | latest |
| SQLite | `modernc.org/sqlite` (pure Go) | latest |
| SQL toolkit | `github.com/jmoiron/sqlx` (thin layer over `database/sql`) | latest |
| Git | `github.com/go-git/go-git/v5` + shelled `git` for PRs | latest |
| Password hashing | `golang.org/x/crypto/argon2` | latest |
| Session cookies | `github.com/gorilla/securecookie` | latest |
| Logging | stdlib `log/slog` | Go 1.22+ |
| Validation | `github.com/go-playground/validator/v10` | latest |

No other dependencies without an explicit RFC. Prefer stdlib.

## 4. Domain Model

```go
// internal/artifact/artifact.go
type Artifact struct {
    Path        string                 // relative to project root, e.g. "lifecycle/ideas/login.md"
    Slug        string                 // "login"
    Index       int                    // 0 for originating file, else >= 2
    Stage       string                 // "ideas", "requirements", etc.
    Frontmatter Frontmatter
    Body        string                 // raw markdown body (after frontmatter)
    Links       []Link                 // extracted from body + frontmatter
    Mtime       time.Time
    SHA256      [32]byte               // content hash for change detection
}

type Frontmatter struct {
    Title     string   `yaml:"title"`
    Type      string   `yaml:"type"`
    Status    string   `yaml:"status"`
    Lineage   string   `yaml:"lineage"`
    Parent    string   `yaml:"parent,omitempty"`
    Labels    []string `yaml:"labels,omitempty"`
    DependsOn []string `yaml:"depends_on,omitempty"`
    Blocks    []string `yaml:"blocks,omitempty"`
    Related   []string `yaml:"related_to,omitempty"`
    Members   []string `yaml:"members,omitempty"`
    Release   string   `yaml:"release,omitempty"`
    Sprint    string   `yaml:"sprint,omitempty"`
    Assignees []Assignee `yaml:"assignees,omitempty"`
    Extra     map[string]any `yaml:",inline"` // preserve unknown fields verbatim
}

type Link struct {
    From, To string // relative paths from project root
    Kind     string // "parent" | "depends_on" | "blocks" | "related_to" | "members" | "wiki"
    Source   string // "frontmatter:parent" | "body:wiki" etc.
}
```

Link extraction:
- Frontmatter array fields produce typed edges (`parent`, `depends_on`, …).
- Body wiki links `[[path]]` or `[[path|label]]` produce `wiki` edges with no semantic type.
- Paths are resolved **relative to the `lifecycle/` root** and normalised (drop `.md`, drop leading `./`).

## 5. SQLite Schema (artifact index)

File: `~/.kaos-control/projects/<project-name>/index.db`. Index is a cache; disk is authoritative. Rebuilt from disk at startup if missing or a schema version mismatch is detected.

```sql
CREATE TABLE schema_version (version INTEGER NOT NULL);

CREATE TABLE artifacts (
    path TEXT PRIMARY KEY,       -- relative to project root
    slug TEXT NOT NULL,
    lineage TEXT NOT NULL,
    idx INTEGER NOT NULL,        -- 0 for originating file
    stage TEXT NOT NULL,
    type TEXT NOT NULL,
    status TEXT NOT NULL,
    title TEXT NOT NULL,
    frontmatter_json TEXT NOT NULL,
    body_sha256 BLOB NOT NULL,
    mtime INTEGER NOT NULL
);
CREATE INDEX idx_artifacts_lineage ON artifacts(lineage);
CREATE INDEX idx_artifacts_stage  ON artifacts(stage);
CREATE INDEX idx_artifacts_status ON artifacts(status);
CREATE INDEX idx_artifacts_slug   ON artifacts(slug);

CREATE TABLE links (
    src TEXT NOT NULL,
    dst TEXT NOT NULL,
    kind TEXT NOT NULL,
    source TEXT NOT NULL,
    PRIMARY KEY (src, dst, kind, source)
);
CREATE INDEX idx_links_src ON links(src);
CREATE INDEX idx_links_dst ON links(dst);

CREATE TABLE labels_index (
    label TEXT NOT NULL,
    artifact TEXT NOT NULL,
    PRIMARY KEY (label, artifact)
);

CREATE TABLE agent_runs (
    run_id TEXT PRIMARY KEY,
    agent_name TEXT NOT NULL,
    role TEXT NOT NULL,
    target_path TEXT,
    started_at INTEGER NOT NULL,
    finished_at INTEGER,
    status TEXT NOT NULL,        -- running|finished|failed|killed|crashed
    exit_code INTEGER,
    stderr_tail TEXT,
    artifacts_produced_json TEXT
);

CREATE TABLE lineage_locks (
    lineage TEXT PRIMARY KEY,
    holder TEXT NOT NULL,        -- user email or agent run_id
    kind TEXT NOT NULL,          -- editor|agent
    acquired_at INTEGER NOT NULL,
    last_heartbeat INTEGER NOT NULL
);
```

Accounts and sessions live in a **separate** DB at the app install dir (not per-project), since users span projects:

```sql
CREATE TABLE users (
    email TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    password_hash TEXT NOT NULL,  -- argon2id encoded
    created_at INTEGER NOT NULL
);
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_email TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL
);
```

Schema version starts at 1. Migrations are simple `ALTER` scripts keyed by version number; on version bump, drop the per-project index and rebuild from disk (it's a cache).

## 6. Parsing & Indexing Pipeline

1. **Enumerate** `lifecycle/**/*.md` at startup.
2. For each file:
   - Read bytes; compute sha256.
   - Split frontmatter (YAML between `---` fences) from body using `goldmark` + frontmatter extension.
   - Unmarshal frontmatter into `Frontmatter` struct; unknown keys go into `Extra`.
   - Validate required fields (§4.2): `title`, `type`, `status`, `lineage` present; `type` and `status` in the known vocabulary (else record a parse error and still index — do not drop the file).
   - Extract slug & index from filename (`<slug>(-<index>)?.md`).
   - Walk body AST for `[[…]]` wiki links (custom goldmark extension).
   - Emit `Artifact` + `Link[]` into SQLite in a single transaction.
3. Parse errors are captured per-path in a separate `parse_errors` table and exposed via `/api/parse-errors`. UI shows a badge.
4. **Live updates**: fsnotify emits path-level events. Debounce 150 ms, then re-parse just the affected files. Batches of deletes/renames are handled atomically (single TX).

Indexing throughput target: ≥ 500 artifacts/sec on first-run, ≥ 200 files/sec for incremental re-parse.

## 7. Filesystem Watcher

- Package `internal/watcher`.
- Wraps `fsnotify.Watcher`. Watches the `lifecycle/` tree recursively (walk + add on start; on directory create, add).
- Emits a normalised `Event{Path, Kind: Created|Modified|Deleted|Renamed}`.
- Ignores:
  - Paths outside `lifecycle/`.
  - Temp files matching `.*~`, `.#*`, `*.swp`, `.DS_Store`.
  - Writes inside the index DB path.
- Fans out via a channel to: indexer, websocket hub.
- **Backpressure**: if the channel blocks > 1 s, log and drop with a counter (Prometheus-ready metric).

## 8. Git Integration

Package `internal/git`.

### Operations
- `EnsureRepo(path)` — checks `.git/`, offers init via a separate API endpoint (UI asks user first).
- `CurrentBranch()`, `CheckoutOrCreate(name)`, `Add(paths)`, `Commit(msg, author)`, `Log(path, limit)`, `FileHistory(path)`, `ShowAt(sha, path)`.
- `Merge(branch, strategy)` where strategy is `fast-forward-only` for local, else no-op for remote (PR flow).
- `OpenPR(branch)` — shells out to `gh pr create` / `glab mr create` / `tea pr create` based on configured forge; if none configured, returns an instructional error.

### Branching
- `BranchNameFor(lineage, frontmatter)` evaluates the project's `branch_template` (§8.2 of spec) with placeholders `{slug}`, `{lineage}`, `{type}`, `{index}`.
- Branch created on the **first artifact commit** for a lineage — not on artifact creation UI event. Keeps the working tree clean until there's something to record.
- Merge to default branch fires on `done` state transition.

### Commit messages
Per §8.3 of spec. Always include a `Co-Authored-By: kaos-control <noreply@kaos-control.local>` trailer.

### Safety
- Reject commits with `allowed_write_paths` violations (agent scope enforcement is in `internal/agent`; this is a second-line check at the git layer).
- Refuse to commit if the working tree has uncommitted changes from a different lineage than the one being advanced.

## 9. Workflow Engine

Package `internal/workflow`.

- `States` and `Transitions` loaded from `lifecycle/config.yaml` at project open; falls back to defaults from §6.1 of spec.
- `CanTransition(from, to, roles []string) bool` consults the authorisation matrix (§6.2).
- `Transition(artifact, to, actor)` performs:
  1. Validate the edge.
  2. Validate the actor holds an authorised role.
  3. Acquire the lineage lock (see §10) for the artifact's lineage.
  4. Update frontmatter `status` in place (edit the YAML section only, not the body).
  5. Commit with templated message.
  6. Release lock, broadcast `artifact.indexed` + `git.committed` events.
- Rejected states capture the rejector's feedback in a **new child artifact** (`…-<next-index>-rejection.md`) rather than in-place edits. See §6.1 of spec.

### Required-plans check
- Per §6.3, a ticket leaves `planning` only when all `required_plans[<type>]` artifacts for that lineage are `approved`.
- `workflow.GateReady(lineage)` returns `(bool, missing []string)`.

## 10. Lineage Lock Manager

Package `internal/lock`.

- In-memory map `map[lineage]Lock` protected by a mutex, persisted to SQLite `lineage_locks` for crash recovery.
- `Acquire(lineage, holder, kind) (Lock, error)` returns `ErrLocked` if already held.
- `Heartbeat(lock)` every 30 s from the websocket connection or agent supervisor.
- `Release(lock)` deletes the row and broadcasts `lock.released`.
- `Reaper` goroutine sweeps every 60 s: locks with `last_heartbeat < now - 5m` are forcibly released with a warning log.
- Unified across editor and agent (§10.5).

## 11. Agent Runner

Package `internal/agent`.

### Driver interface
```go
type Driver interface {
    Start(ctx context.Context, run AgentRun) (Process, error)
}
type Process interface {
    Progress() <-chan Progress // optional, may be closed immediately
    Wait() error               // returns exit code as ExitError
    Kill() error               // SIGTERM then SIGKILL after 10s
}
```

### `claude-code-cli` driver (v1)
- Spawns `claude` binary in subprocess with `-p "<prompt>"` and `--cwd <project-dir>`.
- Prompt template per role is rendered from `lifecycle/config.yaml` with placeholders: `{target_path}`, `{lineage}`, `{related_artifacts}`.
- Streams stdout line-by-line; lines matching `^progress: (.*)$` become `Progress` events.
- Waits for exit; captures last 4 KB of stderr.

### Lifecycle
1. Validate: target lineage not locked; `max_concurrent_agents` not exceeded; agent has the role authorised for the stage.
2. Acquire lineage lock (kind=agent).
3. Insert `agent_runs` row with status=running.
4. Start subprocess; store `run_id` → subprocess handle.
5. Broadcast `agent.started`; forward `Progress` as `agent.progress`.
6. On exit 0: commit any newly-created files in `lifecycle/` (or `allowed_write_paths` for dev/QA agents), update `agent_runs`, broadcast `agent.finished`, release lock.
7. On exit non-zero: broadcast `agent.failed`, set status=failed.
8. On kill: status=killed; partial outputs committed with a `partial:` subject prefix.
9. On crash (process-level): same as non-zero exit, status=crashed.

### Scope enforcement
- Before each commit, diff the working tree vs the pre-run state; reject writes outside `allowed_write_paths`.
- Refuse to start if `allowed_write_paths` is empty (misconfig).

### Concurrency
- Global semaphore sized to `max_concurrent_agents` (default 4).
- Per-lineage semaphore size 1 (backed by the lineage lock).

## 12. Authentication & Authorization

### Auth
- Local accounts only in v1.
- `POST /api/auth/login {email, password}` → set `session` cookie (HttpOnly, Secure, SameSite=Lax), 24 h TTL.
- `POST /api/auth/logout` → invalidate session row.
- `POST /api/admin/users` — create user (only the first user is created without auth; bootstrap flow).
- Password hashing: `argon2id`, params `time=2, memory=64MB, threads=4`, 16-byte salt, 32-byte key.
- CSRF: double-submit cookie for non-GET methods.

### Authorization
- Middleware loads the session, resolves the user, and reads their roles from the **current project's** `lifecycle/config.yaml` `users:` section.
- Handlers that gate actions call `workflow.CanTransition(...)` or `agent.CanRun(...)`.

## 13. REST API Contract (authoritative)

All endpoints under `/api/`. JSON. Errors return `{error: {code, message, detail?}}` with appropriate HTTP status.

### Auth
- `POST /api/auth/login` → 200 with Set-Cookie; 401 on bad creds.
- `POST /api/auth/logout` → 204.
- `GET /api/auth/me` → 200 `{email, display_name, roles: {<project>: […]}}`.

### Projects
- `GET /api/projects` → `[{name, description, path}]`.
- `POST /api/projects` → register a new project (admin only).
- `PATCH /api/projects/:name` / `DELETE /api/projects/:name`.
- `POST /api/projects/:name/init-git` — run `git init` if missing.

### Artifacts (scoped `/api/p/:project/…`)
- `GET /artifacts?stage=&status=&label=&lineage=` → paginated `{items: Artifact[], total}`.
- `GET /artifacts/*path` → single artifact with rendered HTML preview.
- `POST /artifacts` body `{stage, slug, frontmatter, body}` → create, returns new artifact.
- `PUT /artifacts/*path` body `{frontmatter, body, expected_sha}` → update (optimistic concurrency).
- `DELETE /artifacts/*path`.
- `POST /artifacts/*path/transition` body `{to, comment?}` → run workflow transition.
- `POST /artifacts/*path/rename` body `{new_slug}` → slug rename with link rewrite.
- `GET /artifacts/*path/history` → git log for the file.

### Graph
- `GET /graph?filter=…` → `{nodes: [], edges: []}` ready for 3d-force-graph / Cytoscape.
- `GET /labels` → distinct label values.
- `GET /lineages` → `[{lineage, members: […], status_summary}]`.

### Agents
- `GET /agents` → configured agents for current project.
- `POST /agents/:name/run` body `{target_path}` → start run, returns `run_id`.
- `GET /agents/runs?status=` → list runs.
- `GET /agents/runs/:run_id` → run detail including tail of stdout/stderr.
- `POST /agents/runs/:run_id/kill`.

### Config
- `GET /config/project` → current `lifecycle/config.yaml`.
- `PUT /config/project` → update with validation.

### Parse errors
- `GET /parse-errors` → list of `{path, line, message}`.

## 14. WebSocket Protocol

- One socket per open project per user: `GET /api/p/:project/ws` (upgrade). Session cookie authenticates.
- Messages are JSON `{type, payload}`.
- Events sent to the client follow §11 of the spec exactly.
- Client can send `{type: "lock.heartbeat", payload: {lineage}}` every 30 s while editing; server uses this to keep the lock alive.
- Client can send `{type: "subscribe", payload: {lineages: [...]}}` to filter events; default is all project events.

## 15. Configuration Loading

Package `internal/config`.
- `LoadApp(path)` reads `<install-dir>/config.yaml`; validates; defaults applied.
- `LoadProjectRegistry(projects_dir)` enumerates `*.yaml` under the registry dir.
- `LoadProject(path)` reads `lifecycle/config.yaml`; validates stage names, role names, agent configs.
- Hot-reload: `SIGHUP` re-reads the app config. Project config is re-read on any watcher event targeting `lifecycle/config.yaml`.

## 16. Filesystem Sandbox

- Every path-accepting handler calls `sandbox.Resolve(project, userPath) (absPath, error)`:
  1. Clean the input (`filepath.Clean`).
  2. Reject absolute paths, `..` traversal, symlinks that escape the project root (check with `filepath.EvalSymlinks`).
  3. Check the resolved absolute path starts with the project's configured `path`.
- Centralising this avoids per-handler mistakes.

## 17. Metrics & Logging

- `log/slog` JSON logger to stdout, level from `LOG_LEVEL` env (`info` default).
- Correlation ID per request (chi middleware `chi/middleware.RequestID`).
- `/api/metrics` Prometheus endpoint (`prometheus/client_golang`) with counters for: HTTP requests, parse errors, agent runs by status, websocket clients.

## 18. Milestones

Deliver in this order. Each milestone ends with a demonstrable state + commit on the dev branch.

### M1 — Skeleton & Config (≈ 2 days)
- Go module, repo layout, Makefile.
- App config loader + project registry loader.
- Chi router with `/api/health` returning `{ok: true, version}`.
- `embed.FS` stub for frontend (empty index.html).
- **Acceptance**: `go build && ./kaos-control` starts the server; `GET /api/health` returns 200; config validation errors surfaced on stderr.

### M2 — Artifact Indexing (≈ 3 days)
- SQLite index, schema version, migrations.
- Artifact parser (frontmatter + body + wiki links).
- Startup full-scan indexer.
- `GET /artifacts`, `GET /artifacts/*`, `GET /graph` (read-only).
- **Acceptance**: opening this repo (kaos-control itself) indexes all existing `lifecycle/` artifacts; graph endpoint returns nodes and edges including the parent → child chain.

### M3 — Write path + Git (≈ 3 days)
- POST/PUT/DELETE artifacts with sandbox checks.
- Slug rename with link rewrite (atomic commit).
- go-git wrapper: commit per change with templated message.
- Branch-per-lineage creation on first commit.
- fsnotify watcher → incremental re-index + websocket `file.changed` event.
- **Acceptance**: creating an artifact via API creates the file, creates the branch, commits, emits events; renaming a slug rewrites inbound links and produces one commit; dropping a file externally re-indexes within < 1 s.

### M4 — Auth & Workflow (≈ 2 days)
- argon2id local accounts; login/logout; session cookie middleware.
- Workflow engine with transition authorisation.
- Transition API + rejected-child-artifact creation.
- Required-plans gate.
- **Acceptance**: login flow works; transitioning artifacts respects the role matrix; rejection creates a child artifact with the reviewer's note.

### M5 — Agent Runner (≈ 4 days)
- Driver interface + `claude-code-cli` implementation.
- Run tracking table; kill / crash / partial commit flows.
- Scope enforcement via pre-commit diff check.
- WebSocket hub + events for agent lifecycle.
- Lineage lock manager (editor + agent unified).
- **Acceptance**: triggering the configured planner agent on this very requirements doc produces BE/FE/test plan files on the ticket branch with an agent identity; kill button terminates the subprocess; crashed runs produce partial commits; same-lineage double-run is refused.

### M6 — Polish & Ship (≈ 2 days)
- Metrics endpoint; structured logging everywhere.
- Parse-errors endpoint.
- Docker image; release workflow (goreleaser).
- Embedded frontend wired up end-to-end (handoff from FE plan).
- **Acceptance**: `make release` produces a self-contained binary + Docker image; end-to-end smoke test passes (frontend reachable, graph populated, agent run succeeds).

**Total**: ≈ 16 working days for a single developer agent, substantially less with parallel tracks.

## 19. Cross-Plan Coordination

- **Frontend plan**: consumes every endpoint in §13 and every event in §14 of this document. Any additions to these contracts originate here and must be reflected on both sides in the same commit series.
- **Test plan**: the `Acceptance` bullets in §18 of this document are the contractual test scenarios; the test plan expands each into concrete scripts.

## 20. Risks

| Risk | Mitigation |
|---|---|
| `claude-code-cli` behavior changes | Pin to a specific version; capture stderr; treat "prompt not answered" as a failure. |
| fsnotify on macOS/Linux differs | Test on both; add a fallback polling watcher if fsnotify reports unsupported. |
| SQLite corruption | Index is rebuildable from disk; on open, run `PRAGMA integrity_check`; on fail, drop and rebuild. |
| Large repos (> 10k artifacts) | Pagination on all list endpoints; stream graph response. |
| Agent runaway | Hard per-run CPU/time limits configurable; default 10 min wall clock then SIGTERM. |

## 21. Open Questions for the Developer

- **Forge auto-detection** (§8): default to `gh` if `origin` hosts `github.com`, else prompt the user to configure. Confirm with project owner before finalising.
- **Prompt template storage** (§17 of spec): start with inline in `lifecycle/config.yaml`; consider a `lifecycle/prompts/` folder if templates grow.
- **Session storage at scale**: v1 uses SQLite; acceptable up to a few hundred users. If scale becomes a problem, swap to an in-memory LRU + cookie-signed tokens.

## 22. References

- Parent spec: [[requirements/Innovation Maker - Making Releases from Ideas-1]]
- Frontend plan (sibling): [[frontend-plans/Innovation Maker - Making Releases from Ideas-3-fe]]
- Test plan (sibling): [[test-plans/Innovation Maker - Making Releases from Ideas-4-test]]
