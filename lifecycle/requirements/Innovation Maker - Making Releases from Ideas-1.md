---
title: Innovation Maker — Detailed Requirements
type: requirement
status: done
lineage: innovation-maker
parent: ideas/Innovation Maker - Making Releases from Ideas-questions.md
labels:
    - backend
    - go
    - v1
---

requirements/Innovation Maker - Making Releases from Ideas-1.md

# Innovation Maker — Detailed Requirements

> Working name: **kaos-control**. A Go web application that turns ideas into releases by shepherding markdown artifacts through a configurable lifecycle, with AI agents performing work at each stage.

---

## 1. Goals & Non-Goals

### Goals
- Capture ideas, requirements, plans, dev work, and test artifacts as plain markdown files in a git-tracked project directory.
- Visualise artifacts and their relationships as an interactive 3D graph (and a 2D alternative) so a product owner can see the shape of the work.
- Orchestrate AI agents (pluggable LLMs, potentially different per role) to advance artifacts from stage to stage.
- Support a flexible lifecycle whose stages and directory names can be adapted per project.
- Run locally against a directory; expose a web UI accessible over LAN/VPN/reverse proxy.

### Non-Goals (v1)
- Full JIRA parity or bidirectional sync (roadmap).
- Multi-user collaboration with real-time co-editing (GUI lock only, v1).
- Exposing kaos-control as an MCP server (roadmap — MCP *consumption* by agents is in scope).
- Cloud-hosted SaaS deployment.
- Fine-grained per-artifact ACLs beyond Unix file/group permissions.

---

## 2. Personas & Roles

### Role list (v1)
| Role | Responsibility |
|---|---|
| `product-owner` | Creates and curates ideas; decides when work starts |
| `analyst` | Reads ideas → writes detailed requirements; reads requirements → writes backend, frontend and test plans |
| `backend-developer` | Implements code from approved backend plans (Go in `internal/`, `cmd/`) |
| `frontend-developer` | Implements code from approved frontend plans (Vue/TS in `web/src/`) |
| `test-developer` | Implements integration tests from approved test plans (in repo-root `tests/`) |
| `qa` | Runs tests, raises defects in `lifecycle/defects/` and assigns them to the right developer role |
| `reviewer` | Reviews artifacts, may reject with feedback |
| `approver` | Approves artifacts to unlock the next stage |

### Rules
- A **role** is a named capability. A **user** or **agent** is bound to one or more roles.
- Role bindings are stored in project configuration.
- Role assignment gates state transitions (see §6).
- Both humans and agents can hold the same role; a team can have human and agentic developers in parallel.

---

## 3. Core Concepts

### 3.1 Project
A directory on disk containing both a code tree (structured per the language's conventions) and a `lifecycle/` subdirectory where kaos-control artifacts live. Registered with the app via a project config file (see §12).

### 3.2 Artifact
A markdown file in `lifecycle/<stage>/` representing one piece of work at one stage. Artifacts are the unit of storage, linking, and version control.

### 3.3 Lineage
A chain of related artifacts produced from a single originating idea. Each subsequent artifact in the lineage gets the next monotonic integer suffix:
- `lifecycle/ideas/login.md` (no suffix — the originating file)
- `lifecycle/requirements/login-2.md`
- `lifecycle/backend-plans/login-3.md`
- `lifecycle/frontend-plans/login-4.md`
- `lifecycle/dev-plans/login-5.md`
- `lifecycle/test-plans/login-6.md`

Rules:
- **Slug** = the stem before the `-N` suffix. Shared across the lineage.
- **Index** = monotonic, shared across all stages in one lineage.
- A rejected-and-replanned artifact gets the **next available index**, not a reused one. The superseded artifact remains in place (and in git history); links are updated to point to the new one.
- Each artifact MUST declare `parent:` in its YAML frontmatter, pointing to the previous artifact in its lineage (except the originating idea).

### 3.4 Ticket vs Epic
- **Ticket**: work deliverable within a single release.
- **Epic**: work spanning multiple releases. Modelled as an artifact with `type: epic`.
- A ticket can be **promoted** to an epic; this is a type change recorded in frontmatter. Lineage is preserved.

### 3.5 Release and Sprint
- **Release**: a named bundle of tickets intended to ship together. Modelled as its own artifact (`lifecycle/releases/<name>.md`) that links to member tickets.
- **Sprint**: a time-boxed window containing tickets. Modelled as `lifecycle/sprints/<name>.md`. A sprint typically resolves to a release but not always.
- A ticket belongs to **exactly one** release (else it's an epic). A ticket may belong to zero or one sprint.

### 3.6 Label
Free-form tags set in frontmatter (`labels: [auth, security]`). The app auto-collects all label values project-wide to populate filter UIs. No central taxonomy in v1.

---

## 4. File Format

### 4.1 Canonical shape
Markdown with YAML frontmatter. Frontmatter carries structured name/value fields; the markdown body uses `##` headings as flexible section names with free-form content beneath.

```markdown
---
title: User login
type: ticket
status: requirements
parent: ideas/login.md
lineage: login
labels: [auth, security]
release: r-2026.05
sprint: s-2026-04-b
depends_on:
  - requirements/password-reset-2.md
blocks: []
assignees:
  - role: product-owner
    who: keith@sinclair.org.au
---

## Problem
Users need a way to authenticate…

## Acceptance Criteria
- [[requirements/password-reset-2]] must be complete
- Session duration configurable
```

### 4.2 Field rules
- **Required frontmatter fields**: `title`, `type`, `status`, `lineage`. Everything else is optional.
- **`type`** vocabulary (v1): `idea`, `ticket`, `epic`, `plan-backend`, `plan-frontend`, `plan-dev`, `plan-test`, `test`, `prototype`, `release`, `sprint`, `defect`.
- **`status`** vocabulary: `draft`, `clarifying`, `planning`, `in-development`, `in-qa`, `approved`, `rejected`, `abandoned`, `done`, `blocked` (an agent self-marks `blocked` when it cannot proceed due to missing input; only `product-owner` or `analyst` can re-open it).
- Frontmatter fields not in the canonical list are preserved verbatim — teams can add their own.

### 4.3 Relationships (for the graph)
Two equally valid encodings, both indexed by the app:
1. **Frontmatter arrays** — `parent`, `depends_on`, `blocks`, `related_to`, `members` (for release/sprint artifacts). Values are relative paths from `lifecycle/`.
2. **Wiki-style body links** — `[[path/to/artifact]]` or `[[path/to/artifact|display text]]`. Also rendered as hyperlinks in the UI.

Both forms produce edges in the graph. Edge direction follows the field semantic (e.g., `depends_on` points from the depender to the dependency).

### 4.4 Filenames and slugs
- Filenames: `<slug>.md` or `<slug>-<index>.md`.
- **Slug rules**: lowercase, kebab-case, ASCII letters/digits/hyphens only. 3–80 characters. Must be unique within a project across all lifecycle directories — an attempt to create a colliding slug is an error surfaced in the UI.
- **Rename**: renaming a slug rewrites all inbound links (frontmatter + wiki) across the project in a single atomic commit.
- **The originating file has no index suffix**; indices start at `-2` and increment monotonically per lineage.

---

## 5. Directory Layout

### 5.1 Project root
```
my-project/
├── src/                    # language-specific code tree (developer agents write here)
├── tests/                  # integration tests (test-developer agent writes here)
├── lifecycle/
│   ├── config.yaml         # per-project kaos-control config
│   ├── ideas/
│   ├── requirements/
│   ├── backend-plans/
│   ├── frontend-plans/
│   ├── dev-plans/
│   ├── test-plans/
│   ├── tests/              # `test` artifacts documenting suites in /tests
│   ├── prototypes/
│   ├── releases/
│   ├── sprints/
│   └── defects/            # raised by the qa agent
└── .git/
```

### 5.2 Scope of access
- The app reads and writes **inside `lifecycle/` only**, plus the project root's `.git/` for version control operations.
- Developer agents (`backend-developer`, `frontend-developer`, `test-developer`) write code to the project's code tree; this is the **only** exception to the lifecycle scope and is gated by each agent's configured `allowed_write_paths` (see §7.4).
- The `test-developer` agent writes integration test code to a repo-root `tests/` directory in addition to a companion artifact in `lifecycle/tests/`.

### 5.3 Configurable stage directories
Stage directory names are declared in `lifecycle/config.yaml`. Defaults above; teams can rename, add, or remove stages. Example override:
```yaml
stages:
  - name: ideas
    dir: inbox
  - name: requirements
    dir: specs
  - name: backend-plans
    dir: plans/be
```

The app validates that every referenced stage exists and that every artifact's frontmatter `status` maps to a declared stage.

---

## 6. Workflow State Machine

### 6.1 States
Default state graph (declared and overridable in `config.yaml`):

```
draft → clarifying → planning ⇄ rejected
                        ↓
                   in-development ⇄ rejected
                        ↓
                      in-qa ⇄ rejected
                        ↓
                     approved → done
                        ↓
                    abandoned (terminal)
```

- `planning` encompasses any active plan artifacts (BE, FE, dev, test). Multiple plan artifacts may be in flight in parallel (§6.3).
- `rejected` always returns to a prior active stage with feedback captured in a child artifact.
- `abandoned` is terminal and can be entered from any non-terminal state.

### 6.2 Transition authority
Transitions are stored as events on the artifact (see §8.2) and are permitted only if the acting user/agent holds a role authorised for that transition. Default matrix:

| From → To | Authorised roles |
|---|---|
| `draft → clarifying` | `product-owner`, `analyst` |
| `clarifying → planning` | `product-owner`, `reviewer`, `analyst` |
| `planning → in-development` | `approver` |
| `in-development → in-qa` | `backend-developer`, `frontend-developer`, `test-developer` |
| `in-qa → approved` | `qa` |
| `approved → done` | `approver` |
| any → `rejected` | `reviewer` |
| any → `abandoned` | `product-owner`, `approver` |
| any → `blocked` | `analyst`, `backend-developer`, `frontend-developer`, `test-developer`, `qa` (agent self-block) |
| `blocked → draft` | `product-owner`, `analyst` (after answering Open Questions) |

The matrix is overridable per project in `config.yaml`.

### 6.3 Plan branches (backend/frontend/test)
- A `ticket` requires all three plan types — `plan-backend`, `plan-frontend`, `plan-test` — in `approved` state before it can leave `planning`. The default is set in `config.yaml` (`required_plans.ticket`).
- An `epic` has no plan requirement by default; it spans multiple tickets each with their own plans.
- All three plans are produced by the same `analyst` role (typically the `planning-analyst` agent), and they progress in parallel through review and approval.

### 6.4 Clarifying questions
An agent in `clarifying` produces a `questions.md` artifact linked to the requirement. Answering a question updates the artifact in place (no new lineage index) — this is the only stage where in-place edit is standard, because questions are conversational.

---

## 7. Agents

### 7.1 Agent model
An **agent** is a configured pluggable LLM runner bound to one or more roles. Agent configuration is per project (in `config.yaml`), and can reference global credentials.

```yaml
agents:
  - name: planning-analyst
    role: [analyst]
    driver: claude-code-cli
    model: claude-sonnet-4-6
    allowed_write_paths:
      - lifecycle/backend-plans
      - lifecycle/frontend-plans
      - lifecycle/test-plans
    git_identity:
      name: planning-analyst
      email: planning-analyst@innovation-maker.local
  - name: backend-developer
    role: [backend-developer]
    driver: claude-code-cli
    allowed_write_paths:
      - internal
      - cmd
    git_identity:
      name: backend-developer
      email: backend-developer@innovation-maker.local
```

A single role can be served by multiple agents (e.g., `requirements-analyst` and `planning-analyst` both share `role: [analyst]`), and the operator picks which agent to invoke from the UI.

### 7.2 Execution drivers
- **v1**: `claude-code-cli` — spawn Claude Code as a subprocess with a prompt and working directory. Invocation is `claude --dangerously-skip-permissions -p <prompt> --output-format stream-json --verbose [--model <name>]`.
  - `model` (optional, per-agent in `config.yaml`) — Claude alias (`opus`, `sonnet`, `haiku`). Empty defaults to the user's CLI default.
  - `timeout_minutes` (optional, per-agent) — `0` disables; otherwise after N minutes the run is auto-killed and marked `killed-timeout` (distinct from a user-initiated `killed`).
  - Stream-json output is parsed line-by-line into structured progress events broadcast over WebSocket; the raw stream is also tee'd to `<data_dir>/<project>/runs/<run_id>.log` for post-mortem.
- **Roadmap**: `anthropic-api`, `openai-api`, `ollama-local`, `mcp` (agent is itself an MCP server the app calls).

### 7.3 Trigger model
- Agents are **triggered on demand** from the UI (e.g., "Run planning-analyst on this requirement") or by a role-authorised user.
- No automatic state-change triggers in v1 (roadmap).
- When triggered, the app records an **agent run** entity with start time, status, artifact(s) produced, and exit code.

### 7.4 Scope and safety
- Each agent config declares `allowed_write_paths` (defaults to `lifecycle/` only; developer agents extend to code tree; QA agents extend to a nominated test harness).
- The app enforces scope by pre-checking any file write in agent output.
- Agents run with the project directory as working directory; the app passes the target artifact path via a prompt template.

### 7.5 Agent output
- Agents **write files directly** to the project directory.
- Each run produces at least one new artifact (the agent's output) and commits it on the ticket branch with the agent's configured git identity.
- Humans or the `reviewer` role review via the file or the git commit/PR before approval.

### 7.6 Visibility and control (v1)
- Active agent runs are listed in the UI with: agent name, role, target artifact, start time, status, elapsed time, and a **Kill** button.
- During a run the UI shows live progress (parsed assistant text and tool-call lines from the stream-json stream); after the run the full log file can be opened from the same panel.
- On kill: subprocess is terminated; any partial output already written within `allowed_write_paths` is committed; the run is marked `killed` (user-initiated) or `killed-timeout` (per-agent `timeout_minutes` exceeded).
- On crash: same behaviour, marked `failed`, with the captured subprocess exit code and last stderr line.
- An agent that cannot proceed should self-mark its target artifact `status: blocked` and assign it to `product-owner` (see §6.2). The run itself finishes `done` since the agent stopped cleanly.

### 7.7 Concurrency
- **Hard rule**: one agent run per ticket lineage at a time.
- The project-level **ticket lock** (§10.5) is unified across GUI edits and agent runs: holding the lock blocks both.
- Multiple agent runs against **different** lineages may run in parallel, bounded by a configurable global `max_concurrent_agents` (default 4).

---

## 8. Git Integration

### 8.1 Requirements
- A project directory **must** be a git repository (`git init` run by the user or offered by the app on first-open).
- Remote is optional. If present, `origin` is used for PR-based merges; if absent, fast-forward merges happen locally.

### 8.2 Branching
- Every lineage gets its own branch. Default naming: `ticket/<slug>`.
- Branch naming is **configurable** via `config.yaml`:
  ```yaml
  git:
    branch_template: "ticket/{slug}"
    # alternatives teams might pick: "feat/{slug}", "{type}/{slug}", "{lineage}"
  ```
  Supported placeholders: `{slug}`, `{lineage}`, `{type}`, `{index}`.
- Branch is created on the first artifact commit for a lineage.
- All artifacts and code changes for that lineage land on that branch.
- Merge to `main` (or configured default branch) happens when lineage reaches `done`:
  - If remote exists: open a PR via the configured forge (GitHub, GitLab, Gitea) and prompt the approver to merge.
  - If remote absent: fast-forward merge locally.

### 8.3 Commits
- Each artifact creation or update is its own commit.
- Commit author is:
  - the **logged-in user** for human actions
  - the **agent's configured `git_identity`** for agent actions
- Commit message template:
  ```
  [<lineage>] <short summary>

  Stage: <status before> → <status after>
  Actor: <role> <name>
  Artifact: <relative path>
  ```
- All commits include a `Co-Authored-By` trailer naming the project and app version.

### 8.4 File watching
- The app watches the `lifecycle/` tree with `fsnotify`.
- External edits (e.g., IDE, `git checkout`) are detected and emitted as `file.changed` WebSocket events.

---

## 9. Visualisation

### 9.1 3D graph (primary)
- Library: [3d-force-graph](https://github.com/vasturiano/3d-force-graph).
- **Nodes**: every artifact, plus releases, sprints, labels, and agents. Node colour by `type`.
- **Edges**: `parent`, `depends_on`, `blocks`, `related_to`, `member_of` (release/sprint), `produced_by` (agent → artifact). Edges are **directed** and colour-coded per type.
- **Filters** (sidebar): by type, status, label, sprint, release, role, or lineage.
- **Scale**: designed for hundreds of nodes per project. Beyond ~2000, graph switches to an aggregated view (lineage-level nodes with expand-on-click).

### 9.2 2D graph (alternative)
- Library: [Cytoscape.js](https://js.cytoscape.org/).
- Same data model; useful for dense dependency charts, printable exports.
- Toggle in UI.

### 9.3 Node modal
Clicking a node opens a modal with:
- Action bar at the top: **Edit**, **Change state**, **Run agent**, **Open in IDE**, **View git history**, **Delete**.
- A rendered markdown preview of the artifact body.
- Frontmatter shown as a structured sidebar.
- Inbound and outbound edges listed with click-through to other nodes.

---

## 10. UI and Editor

### 10.1 App shell
- Login screen → project picker → project workspace.
- Project name appears in the URL: `/p/<project>/…`.
- Left nav: stage filter, saved views, active agent runs.
- Main pane: toggleable between Graph and Artifact views.
- Right pane: artifact preview / details.

### 10.2 Editor
- Simple markdown editor with syntax highlighting and live split-pane preview.
- No WYSIWYG in v1.
- Keyboard shortcuts for common actions (save, toggle preview, insert wiki-link).

### 10.3 External editing
- Users are expected to also edit files from their IDE.
- `fsnotify` detects external changes and prompts open-in-GUI viewers to reload.
- **Conflict resolution**: if a file changes on disk while open in the GUI editor, the disk version wins. The GUI shows a dialog: *"File changed externally. Reload and discard your edits?"* with **Reload** and **Keep editing (copy to clipboard)** options.

### 10.4 Realtime
- WebSocket channel per open project per user.
- Events (see §11) push graph updates, agent progress, and file changes without polling.

### 10.5 Unified ticket lock
- One lock per lineage.
- Acquired when:
  - A user opens the GUI editor on any artifact in the lineage, **or**
  - An agent run starts against the lineage.
- Blocks:
  - Other users from opening the GUI editor on the same lineage.
  - Any agent from starting against the same lineage.
- Released on:
  - Editor close / save-and-close.
  - Agent run completion (success, kill, or crash).
  - 5 minutes of editor inactivity or WebSocket disconnect.
- Lock state is broadcast via `lock.acquired` / `lock.released` events.

---

## 11. WebSocket Events

Server → client, per project channel:

| Event | Payload | Fires when |
|---|---|---|
| `file.changed` | `{ path, change: created\|modified\|deleted }` | fsnotify detects disk change |
| `artifact.indexed` | `{ path, frontmatter, links }` | app finishes re-indexing a changed file |
| `agent.started` | `{ run_id, agent, role, target, started_at }` | agent subprocess launches |
| `agent.progress` | `{ run_id, message, pct? }` | agent streams progress |
| `agent.finished` | `{ run_id, artifacts_produced, commit }` | agent exits 0 |
| `agent.failed` | `{ run_id, reason, exit_code, stderr_tail }` | agent exits non-zero or is killed |
| `lock.acquired` | `{ lineage, holder, kind }` | lineage lock taken |
| `lock.released` | `{ lineage }` | lineage lock freed |
| `git.committed` | `{ branch, sha, author, subject }` | any commit lands |

---

## 12. Tech Stack

### 12.1 Backend
- **Language**: Go (1.22+).
- **Router**: `go-chi/chi`.
- **HTTP**: stdlib `net/http`.
- **Markdown parsing**: `goldmark` with YAML frontmatter extension.
- **YAML**: `gopkg.in/yaml.v3`.
- **File watching**: `fsnotify`.
- **WebSockets**: `nhooyr.io/websocket` or `gorilla/websocket`.
- **DB**: SQLite via `modernc.org/sqlite` (pure-Go, no cgo) for the artifact index. **Files on disk are the source of truth**; SQLite is rebuilt from disk at startup and kept in sync via fsnotify.
- **Git**: `go-git/go-git` for in-process ops; shell out to `git` binary for PRs / remote-specific features.

### 12.2 Frontend
- **Build pipeline**: **Vite**. Produces static assets served by the Go binary (embedded via `embed.FS`).
- **Core framework**: **Vue 3** (Single-File Components) for app shell, editor, modals, filters.
- **Graph**: **3d-force-graph** (3D) and **Cytoscape.js** (2D), wrapped as Vue components.
- **Markdown rendering**: `markdown-it` in the editor preview; `marked` acceptable alternative.
- **Styling**: TailwindCSS or a small custom system — to be decided during prototyping.

### 12.3 Distribution
- Single static Go binary with embedded frontend.
- SQLite index written under `~/.kaos-control/` or the app install dir.

---

## 13. Configuration

### 13.1 App-level (install)
Location: `<install-dir>/config.yaml` (default `/home/<user>/kaos-control/config.yaml`).
```yaml
server:
  listen: ":8080"
  tls:
    enabled: false
    cert_file: ""
    key_file: ""
auth:
  method: local       # v1 only; "sso" in roadmap
  session_ttl: 24h
projects_dir: "/home/keith/kaos-control/projects"
limits:
  max_concurrent_agents: 4
```

### 13.2 Project registration
Location: `<install-dir>/projects/<name>.yaml`.
```yaml
name: my-new-project
path: /home/keith/Projects/my-new-project
description: "Customer portal rewrite"
owner: keith@sinclair.org.au
```
- A UI under the project picker allows full CRUD of these files (create, edit, remove) — no manual file dropping required.
- Project URL: `/p/my-new-project/…`.

### 13.3 Project-level (inside the repo)
Location: `lifecycle/config.yaml` (version-controlled with the project).
```yaml
stages:
  - name: ideas
    dir: ideas
  - name: requirements
    dir: requirements
  # …defaults match §5.1
git:
  default_branch: main
  branch_template: "ticket/{slug}"
roles:
  - product-owner
  - analyst
  - backend-developer
  - frontend-developer
  - test-developer
  - qa
  - reviewer
  - approver
transitions:
  # overrides of the default matrix, if any
users:
  - email: keith@sinclair.org.au
    roles: [product-owner, analyst, reviewer, approver]
agents:
  # see §7.1
required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []
```

---

## 14. Security

### 14.1 Transport
- App can terminate TLS itself (cert/key in app config), or serve HTTP behind a reverse proxy that handles TLS.
- Auth is **always** handled by the app; the reverse proxy is not trusted as the auth boundary.

### 14.2 Authentication
- **v1**: local accounts. Credentials stored in SQLite, passwords hashed with `argon2id`. Session cookies with secure flags.
- **Roadmap**: SSO (OIDC).

### 14.3 Authorisation
- Role-based. Role bindings in `lifecycle/config.yaml`.
- Every state transition and agent trigger checks the acting identity's role against the authorisation matrix.

### 14.4 Filesystem sandbox
- The app resolves project paths, rejects symlinks that escape the project root, and refuses absolute-path writes outside the configured `projects_dir`.
- OS-level access to a project directory is governed by Unix file/group permissions. For multi-user deployments, use group-owned project directories.
- Docker deployment is supported (containerised app + bind-mounted project directories) as an optional hardening step, not mandatory.

---

## 15. Default Assumptions (confirmed)

- **Slug collisions**: error on create.
- **Slug renames**: rewrite all inbound links (frontmatter and wiki) in a single atomic commit.
- **Lock timeout**: 5 minutes of inactivity or disconnect.
- **Epic promotion**: ticket → epic via frontmatter `type` change, preserving lineage.
- **Labels**: free-form, collected for filter UIs.
- **UI prototypes**: static HTML/CSS under `lifecycle/prototypes/<lineage>/`, opened in a new tab.
- **Node modal actions**: Edit, Change state, Run agent, Open in IDE, View git history, Delete.

---

## 16. Roadmap (explicitly out of v1)

| Item | Notes |
|---|---|
| SSO authentication | OIDC, replacing or augmenting local accounts |
| JIRA integration | TBD: bidirectional vs mirror; Cloud vs Server/DC — decide when prioritised |
| kaos-control as MCP **server** | Expose artifacts/actions to external agents like Claude Code, Cursor |
| Additional agent drivers | `anthropic-api`, `openai-api`, `ollama-local`, direct `mcp` client |
| Automatic state-change triggers | Auto-run agent on state transition, with safety gates |
| Real-time co-editing | Replace lock with CRDT or OT |
| Advanced agent visibility | Streaming logs, pause/resume, checkpointing |
| Multi-project dashboards | Cross-project views for portfolio management |
| PR review UX inside app | Comment threads on diffs for agent output |

---

## 17. Open Questions (parked — decide during build)

- **Styling system** (§12.2): Tailwind vs a small custom CSS layer — defer until the first real UI iteration.
- **Prompt templates for agents**: where are they stored (in `lifecycle/config.yaml`? separate `lifecycle/prompts/`? bundled with the app?) — decide during agent driver implementation.
- **Auto-collection cadence for labels/types**: realtime via fsnotify is the default, but a manual "re-index" button may be needed for large imports.
- **SQLite schema versioning**: migration strategy for index rebuilds across app versions.
