# CLAUDE.md

Guidance for Claude Code (claude.ai/code) when working in this repository.

## What this is

**kaos-control** (product name *Innovation Maker*, to be finalised) is a single-binary lifecycle management tool: a Go server with an embedded Vue 3 SPA that indexes markdown artifacts under `lifecycle/`, exposes a graph + editor UI, and orchestrates agent runs that produce new artifacts and code.

The project is **meta**: it enforces a specific requirements-to-release lifecycle, and the same lifecycle structure is applied to this repo to produce the tool itself.

## Authoritative spec

[lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md](lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md) is the source of truth for product scope, file format, workflow states, roles, and configuration. Read it before any substantive design discussion. Do not restate it here.

## Repository layout

```
kaos-control/
├── cmd/kaos-control/        Go binary entry point
├── internal/                Go packages
│   ├── agent/               agent runner, ClaudeCodeDriver, supervisor
│   ├── artifact/            markdown + frontmatter parser, type vocab
│   ├── auth/                argon2id password hashing, sessions
│   ├── config/              app + project YAML config loaders
│   ├── git/                 go-git wrapper (branches, commits, history)
│   ├── http/                chi router, REST + WebSocket handlers
│   ├── hub/                 WebSocket broadcast hub
│   ├── index/               SQLite cache (modernc.org/sqlite, pure Go)
│   ├── lock/                lineage lock manager (heartbeat + reaper)
│   ├── project/             per-project runtime container
│   ├── reports/             analytics aggregation (agent usage report)
│   ├── sandbox/             path-traversal-safe filesystem resolver
│   ├── watcher/             fsnotify → incremental re-index
│   └── workflow/            transition state machine + plan gating
├── web/                     Vue 3 + Vite 5 + TypeScript + Pinia SPA
│   ├── src/                 source
│   └── dist/                built assets, embedded into Go binary
├── lifecycle/               this project's own artifacts
│   ├── config.yaml          per-project config (roles, agents, gates)
│   ├── ideas/               originating idea docs
│   ├── requirements/        detailed specs
│   ├── backend-plans/       per-feature backend implementation plans
│   ├── frontend-plans/      per-feature frontend implementation plans
│   ├── test-plans/          per-feature test plans
│   ├── tests/               artifacts describing what the test code in /tests covers
│   ├── prototypes/
│   ├── defects/             defect artifacts raised by the qa agent
│   ├── releases/
│   └── sprints/
├── tests/                   integration test code (test-developer agent target)
├── plans/                   PROJECT_PLAN.md + per-change implementation plans
└── Makefile                 build, run, test, lint targets
```

## Build & run

```sh
make build-web    # pnpm install + pnpm build → web/dist/
make build        # go build → ./dist/kaos-control (embeds web/dist)
make run          # development mode (LOG_LEVEL=debug)
make lint         # go vet + staticcheck
make test-unit    # go test ./... -short
```

App config lives at `~/.kaos-control/config.yaml`; project registrations at `~/.kaos-control/projects/*.yaml`.

## Tech stack (in use)

**Backend** — Go 1.25+ stdlib `net/http` + `go-chi/chi`; `goldmark` (markdown), `gopkg.in/yaml.v3`, `fsnotify`, `coder/websocket`, `modernc.org/sqlite` (pure-Go), `go-git/go-git`.

**Frontend** — Vite 5, Vue 3.5 SFCs, TypeScript, Pinia, Vue Router 4, `markdown-it`, `3d-force-graph` + three.js (3D), Cytoscape.js + cytoscape-fcose (2D), CodeMirror 6 (editor), lucide-vue-next (icons).

**Distribution** — single Go binary; frontend embedded via `embed.FS` from `web/dist/`.

## Roles & agents

Roles split by lifecycle phase:
- **Think**: `analyst` — reads ideas → writes requirements; reads requirements → writes 3 plans.
- **Make**: `backend-developer` (Go in `internal/`, `cmd/`), `frontend-developer` (Vue/TS in `web/src/`), `test-developer` (integration tests in `tests/` + artifacts in `lifecycle/tests/`).
- **Verify**: `qa` — runs tests, raises defects in `lifecycle/defects/`, assigns to the right developer role.
- **Cross-cutting**: `product-owner`, `reviewer`, `approver`.

Six agents are configured in [lifecycle/config.yaml](lifecycle/config.yaml): `requirements-analyst`, `planning-analyst`, `backend-developer`, `frontend-developer`, `test-developer`, `qa`. Each has scoped `allowed_write_paths` and a focused prompt template.

`required_plans.ticket = [plan-backend, plan-frontend, plan-test]` gates `planning → in-development`.

## Lineage filename convention

Artifacts for a single idea share a **slug** and carry a **monotonic index** across stages, with optional stage suffix:

```
lifecycle/ideas/login.md            (originating, no suffix)
lifecycle/requirements/login-2.md
lifecycle/backend-plans/login-3-be.md
lifecycle/frontend-plans/login-4-fe.md
lifecycle/test-plans/login-5-test.md
```

- First file in a lineage has **no suffix**. Indices start at `-2`.
- Index is monotonic **per lineage, across stages** — never reused.
- Rejected-and-replanned artifacts get the **next** index; superseded files stay in place and in git history.
- Every non-originating artifact has `parent:` in its YAML frontmatter pointing to the previous file.

Full rule: §3.3 and §4.4 of the spec.

## Frontmatter requirements

Required fields on every artifact: `title`, `type`, `status`, `lineage`. Type vocabulary (§4.2 of spec): `idea`, `plan-backend`, `plan-frontend`, `plan-dev`, `plan-test`, `test`, `prototype`, `release`, `sprint`, `defect`. Status vocabulary: `raw` (unprocessed quick-capture input), `draft`, `clarifying`, `planning`, `in-development`, `in-qa`, `approved`, `rejected`, `abandoned`, `done`, `blocked`. The canonical list lives in `KnownStatuses` at [internal/artifact/artifact.go](internal/artifact/artifact.go) — verify there if in doubt.

## Indexing behaviour

The SQLite index is a cache; disk is authoritative. Updates happen at:
1. **Startup** — full scan of `lifecycle/**/*.md` (after schema check).
2. **Live** — `fsnotify` watcher with 150 ms debounce, broadcasts `artifact.indexed` and `file.changed` WS events.
3. **API writes** — `PUT /artifacts/*`, `POST /artifacts` re-index synchronously before responding.

## Commit conventions

- **Project plan**: every commit updates `plans/PROJECT_PLAN.md` — bump the "Recent Changes" log and any affected "Completed"/"Planned" entries.
- **Implementation plans**: when a Claude Code plan file (`~/.claude/plans/*.md`) drove the change, copy it into `plans/<descriptive-name>.md` and include it in the same commit.
- Commits should be small and focused. Don't amend; create new commits.
