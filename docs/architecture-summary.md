# kaos-control architecture summary

## Shape

Single Go binary that embeds a Vue 3 SPA. Files on disk are the source of truth; SQLite is a query cache. Go server boots, scans `lifecycle/**/*.md`, indexes them, then serves an HTTP+WebSocket API plus the embedded SPA on one port (default `:8042`).

## Go side ([cmd/kaos-control](../cmd/kaos-control), [internal/](../internal))

**Entry:** [cmd/kaos-control/main.go](../cmd/kaos-control/main.go) dispatches subcommands (`init`, `backfill-created`, `hook-helper`, `authcmd`) or falls through to the server.

**Packages** ([internal/](../internal)) by responsibility:

| Package | What it does |
|---|---|
| `agent` | Driver interface + 7 drivers (claude-code-cli, claude-mediated, codex-cli, ollama, gemini, gemini-cli, shell-stub); run lifecycle, supervisor, policy/permission engine, lineage locks |
| `artifact` | Markdown + YAML-frontmatter parser, type/status vocabulary, edge-kind constants |
| `auth` | argon2id hashing, sessions, role membership |
| `config` | `~/.kaos-control/config.yaml` (app) + `lifecycle/config.yaml` (project) loaders |
| `git` | go-git wrapper — branches, commits, history walks |
| `http` | chi router; REST handlers + WebSocket hub bridge; CSRF, session middleware; the embedded-SPA serving |
| `hub` | Per-project WebSocket fanout |
| `index` | SQLite (modernc.org/sqlite, pure-Go) schema + read/write paths |
| `lock` | Lineage locks with heartbeat + reaper (prevents two agents racing the same lineage) |
| `project` | Per-project runtime container — couples index, hub, watcher, agent manager, sandbox |
| `queue` | Per-agent job queue + dispatcher with rate-limit-aware re-enqueue |
| `sandbox` | Path-traversal-safe filesystem resolver |
| `scheduler`, `statuscheck`, `testrunner`, `devops`, `release` | Lifecycle automations |
| `watcher` | fsnotify → 150 ms debounced reindex + WS broadcast |
| `workflow` | Status transition state machine + role/plan gates |
| `initcmd`, `backfillcmd` | CLI subcommands; `initcmd` ships a templated project scaffold |

Hook helper lives at [cmd/kaos-control/hookcmd](../cmd/kaos-control/hookcmd) — separate subcommand invoked by Claude Code for PreToolUse permission checks.

## Vue side ([web/](../web))

Vite 5/6, Vue 3.5 SFCs, TypeScript, Pinia, Vue Router 4. Built to `web/dist/`, embedded into the Go binary via `embed.FS`.

| Dir | Role |
|---|---|
| `src/views/` | Route-level pages — Dashboard, Workspace, Agents/Runs, Queue, Releases, etc. |
| `src/components/<area>/` | Feature-scoped components (`agent/`, `artifact/`, `releases/`, `map/`, `queue/`, `ollama/`, …) |
| `src/stores/` | Pinia stores per domain (agents, queue, releases, roadmap settings, …) |
| `src/composables/` | Reusable Vue hooks — `useWebSocket`, `useFormatDate`, etc. |
| `src/api/` | Typed REST client wrappers |
| `src/types/` | Shared TS interfaces mirroring the Go API |
| `src/router/` | Route table |
| `src/styles/` | Design tokens, status palettes |

**Key viz libs:** `3d-force-graph` + three.js (3D graph), Cytoscape.js + cytoscape-fcose/dagre (2D map), CodeMirror 6 (editor), markdown-it (rendering), ECharts (dashboard charts), lucide-vue-next (icons).

## Build, lint, test ([Makefile](../Makefile))

| Target | What it runs |
|---|---|
| `make build-web` | `cd web && pnpm install && pnpm run build` → `web/dist/` |
| `make build` | `go build` → `dist/kaos-control` (web/dist embedded) |
| `make run` | development server, `LOG_LEVEL=debug` |
| `make lint` | `go vet` + staticcheck + govulncheck + gosec (with curated `-exclude=G…`) + gitleaks |
| `make test-unit` | `go test ./... -count=1 -short` |
| `make test-integration` | `go test ./... -count=1 -tags=integration` |
| `make test-e2e` | builds the binary then `cd tests/e2e && pnpm install && pnpm test` |
| `make test-all` | unit + integration + e2e + frontend (`tests/web pnpm test`) |
| `make release` / `make package` | cross-compile to all platforms in `PLATFORMS` + zipped artefacts + `SHA256SUMS` |

**Frontend testing:** Vitest + `@vue/test-utils` + jsdom in [tests/web/](../tests/web). 1452 tests today. No JS/TS lint yet — there's an open idea `frontend-lint-gap` proposing ESLint + wiring `vue-tsc --noEmit` into `make lint`.

**E2E:** Playwright in [tests/e2e/](../tests/e2e), 23 flows covering login, edits, transitions, agent runs (with a fake driver), graph clicks, doc workflows, run count column.

## Distribution

Single static binary per platform. Frontend embedded; SQLite is pure-Go (no cgo); no external runtime deps. Distributed as zipped binary + LICENSE + README + CONTRIBUTING via `make package`. Project config lives at `~/.kaos-control/config.yaml`; per-project registrations at `~/.kaos-control/projects/*.yaml`.
