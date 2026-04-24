# Project Plan — kaos-control / Innovation Maker

Living document summarising project state. Updated on every commit per the Commit Conventions in [CLAUDE.md](../CLAUDE.md).

**Current stage:** FE-M1 complete — Vite + Vue 3 + TS + Pinia + Router scaffold, login, project picker, workspace shell. Next: FE-M2 (artifact list + read-only editor).

---

## Recent Changes

Rolling log — add a dated bullet per commit.

- **2026-04-24** — Initial commit: original idea captured.
- **2026-04-24** — Requirements flow completed: Q&A rounds, detailed spec distilled, lifecycle directory structure established, `CLAUDE.md` created with commit conventions, first implementation-plan artifact (`plans/create-claude-md.md`) saved.
- **2026-04-24** — Three development plans generated from the detailed requirements: backend (`…-2-be.md`), frontend (`…-3-fe.md`), test (`…-4-test.md`). Each phased into 6 milestones with cross-plan coordination noted.
- **2026-04-24** — M1 + M2 implemented: repo scaffold (`cmd/`, `internal/`, `web/`, `Makefile`), `internal/artifact` parser, `internal/index` SQLite layer, `internal/project` container, full read-only HTTP API (`/artifacts`, `/graph`, `/labels`, `/lineages`, `/parse-errors`). Fixed chi wildcard routing (greedy wildcard requires inline dispatch for suffix-matched sub-routes). Acceptance verified: 7 lifecycle artifacts indexed, graph returns 7 nodes + 19 edges.
- **2026-04-24** — M3 implemented: `internal/sandbox` (path traversal guard), `internal/git` (go-git wrapper: branch-per-lineage, AddAndCommit, Log, identity resolution), `internal/hub` (WebSocket broadcast), `internal/watcher` (fsnotify debouncer → incremental re-index + events), write API (`POST /artifacts`, `PUT /artifacts/*`, `DELETE /artifacts/*`, `POST /artifacts/*/rename`), WebSocket endpoint (`GET /api/p/:project/ws`), git history handler. Acceptance verified: create via API → file on disk + branch created + commit; rename → inbound links rewritten in one atomic commit; external file drop → re-indexed in < 500 ms.
- **2026-04-24** — FE-M1 implemented: Vite 5 + Vue 3.5 + TypeScript + Pinia + Vue Router scaffold under `web/`; typed fetch API client with CSRF double-submit; auth store (login/logout/fetchMe); project store; toast/ui store; Vue Router with auth guard (nav-guard calls `/api/auth/me` on first load); LoginView + LoginForm; ProjectPickerView (lists projects, shows user roles as chips); WorkspaceView shell (AppHeader, AppSidebar, RouterView); ArtifactListView placeholder; dark-sidebar layout with design tokens; `pnpm build` → `web/dist/` → embedded by Go binary. `.gitignore` updated: `/dist/` roots the Go binary ignore, `web/node_modules/` excluded.
- **2026-04-24** — M5 implemented: `internal/lock` (lineage lock manager with SQLite persistence, heartbeat, reaper goroutine), `internal/agent` (Driver interface, ClaudeCodeDriver spawning `claude --dangerously-skip-permissions -p`, ring-buffer stderr, Manager with global semaphore + per-lineage locking, supervisor goroutine for exit/commit/broadcast, crash recovery), `internal/http/agents.go` (GET /agents, POST /agents/:name/run, GET/POST /agents/runs/…), git `ModifiedFiles` for scope-enforced post-run commit, index CRUD for `agent_runs` and `lineage_locks` tables, lock reaper wired into startup. `lifecycle/config.yaml` extended with backend-planner agent config.

---

## Completed

- Original idea captured: [lifecycle/ideas/Innovation Maker - Making Releases from Ideas.md](../lifecycle/ideas/Innovation Maker - Making Releases from Ideas.md)
- Clarifying Q&A (two rounds): [lifecycle/ideas/Innovation Maker - Making Releases from Ideas-questions.md](../lifecycle/ideas/Innovation Maker - Making Releases from Ideas-questions.md)
- Detailed requirements spec: [lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md](../lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md)
- Repo guidance for Claude Code: [CLAUDE.md](../CLAUDE.md) (includes commit conventions)
- Plan: Create CLAUDE.md — [plans/create-claude-md.md](create-claude-md.md)
- Backend development plan — [lifecycle/backend-plans/Innovation Maker - Making Releases from Ideas-2-be.md](../lifecycle/backend-plans/Innovation Maker - Making Releases from Ideas-2-be.md)
- Frontend development plan — [lifecycle/frontend-plans/Innovation Maker - Making Releases from Ideas-3-fe.md](../lifecycle/frontend-plans/Innovation Maker - Making Releases from Ideas-3-fe.md)
- Test plan — [lifecycle/test-plans/Innovation Maker - Making Releases from Ideas-4-test.md](../lifecycle/test-plans/Innovation Maker - Making Releases from Ideas-4-test.md)
- **M1 (skeleton)**: `cmd/kaos-control/main.go`, `Makefile`, `web/embed.go`, config loading, project registry, signal handling
- **M2 (artifact indexing)**: `internal/artifact` (parser, links, types), `internal/index` (SQLite schema, scan, all queries), `internal/project` (runtime container), HTTP API (`/artifacts`, `/graph`, `/labels`, `/lineages`, `/parse-errors`)
- **M3 (write path + git)**: `internal/sandbox`, `internal/git`, `internal/hub`, `internal/watcher`; write API + WebSocket endpoint; git history; watcher-driven re-index
- **M4 (auth + workflow)**: `internal/auth` (argon2id, session store), `internal/workflow` (state machine, GateReady); login/logout/me/create-user endpoints; CSRF double-submit; session middleware; `POST /transition` with role-matrix enforcement; rejection child artifact creation; `lifecycle/config.yaml` user binding
- **M5 (agent runner)**: `internal/lock` (lineage lock manager), `internal/agent` (Driver interface, ClaudeCodeDriver, Manager with semaphore + supervisor goroutine, crash recovery), agent HTTP API (list agents, start run, list/get runs, kill), `git.ModifiedFiles`, index CRUD for agent_runs + lineage_locks, lock reaper, agent config in lifecycle/config.yaml
- **FE-M1 (scaffold)**: `web/` Vite + Vue 3 + TS + Pinia + Router; API client with CSRF; auth/project/ui stores; LoginView, ProjectPickerView, WorkspaceView shell with sidebar; design tokens; `pnpm build` → embedded by Go

---

## Planned

### Next: FE-M2 — Artifact List & Editor Read-Only (≈ 3 days)
- `ArtifactListView` with server-side filtering (stage/status/label) via `GET /api/p/:project/artifacts`
- `ArtifactEditorView` read-only: markdown-it preview + wiki-link rendering
- Frontmatter panel as read-only summary
- `useWebSocket` composable wiring `file.changed` events to invalidate artifact cache
- **Acceptance**: pointed at this repo, every existing artifact is listed and previews cleanly with wiki-link navigation

### Roadmap: M6 — Frontend SPA (≈ 4 days)
- Vue 3 SPA with Vite build pipeline
- Artifact list + graph visualisation (3d-force-graph / Cytoscape.js)
- Artifact detail view with markdown rendering (markdown-it)
- Login / session management UI
- Transition controls + agent run trigger UI
- WebSocket integration for live updates
- **Acceptance**: browser shows graph of all indexed artifacts; transition + agent run work end-to-end; auth gates protected views

---

## Open Questions (parked)

Carry-overs from §17 of the spec — decide during implementation.
- Finalise product name: **kaos-control** vs **Innovation Maker**.
- Styling system: Tailwind vs small custom CSS layer.
- Agent prompt template storage location.
- SQLite schema migration strategy for index rebuilds across app versions.
- Auto-collection cadence for labels/types (realtime fsnotify vs manual re-index).

---

## References

- Authoritative spec: [lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md](../lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md)
- Workflow log & prompt library: [project-notes.md](../project-notes.md)
- Agent/Claude guidance: [CLAUDE.md](../CLAUDE.md)
