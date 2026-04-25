# Project Plan — kaos-control / Innovation Maker

Living document summarising project state. Updated on every commit per the Commit Conventions in [CLAUDE.md](../CLAUDE.md).

**Current stage:** Documentation refresh — CLAUDE.md updated to reflect actual repo state (Go + Vue code exists, six agents configured, defects stage, full directory tree); short README.md added as a placeholder until comprehensive docs land. Ready to start exercising the workflow with real agent runs.

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
- **2026-04-24** — FE-M4 implemented: backend `file_sha` (SHA256 of raw file added to GET artifact response), lock HTTP API (GET/POST /locks, DELETE/POST /locks/:lineage/heartbeat) wired to existing lock.Manager. Frontend: `api/locks.ts`, `stores/locks.ts` (WS event-driven map), `composables/useLock.ts` (acquire on enter-edit, 30s heartbeat interval, release on cancel/save/unmount, 503 treated as lock-free), `composables/useExternalChange.ts` (file.changed WS + 3s save-grace window), `MarkdownEditor.vue` (CodeMirror 6 with basicSetup + markdown + oneDark theme + Cmd+S keymap), `FrontmatterEditor.vue` (typed inputs for title/status/labels/release/sprint/depends_on/blocks), `LockBanner.vue`. `ArtifactEditorView.vue` rewritten: read mode (existing preview) ↔ edit mode (CodeMirror | MarkdownPreview split + FrontmatterEditor), optimistic PUT with expected_sha, conflict error messaging, lock banner for held locks, external-change reload-or-keep prompt. TypeScript clean, 610-module Vite build clean.
- **2026-04-25** — Index path-escape fix: `IndexFile` and the watcher both compute `filepath.Rel` against the symlink-resolved project root (matching the earlier sandbox fix); they refuse to index files whose computed relative path begins with `..` or is absolute. Added `pruneEscapingPaths()` at startup which deletes existing rows from `artifacts`, `parse_errors`, `links`, and `labels_index` whose paths begin with `..`, `/`, or contain `/../` — fixes stale entries from past firmlink/Rel mismatches.
- **2026-04-25** — Documentation refresh: rewrote CLAUDE.md to reflect actual state (was still describing the pre-code phase) — added repository layout tree, build/run commands, tech-stack-in-use, role/agent overview, indexing behaviour, and frontmatter requirements. Added a short README.md as a quick-start placeholder; comprehensive docs deferred.
- **2026-04-25** — Role vocabulary migration: renamed `backend-planner`→`backend-developer`, `frontend-planner`→`frontend-developer`, `developer`→`test-developer`; added `analyst` role; added `defect` artifact type + `lifecycle/defects/` stage. Six agent configs in `lifecycle/config.yaml`: `analyst-requirements`, `analyst-planner`, `backend-developer`, `frontend-developer`, `test-developer`, `qa` — each with focused prompt template and scoped `allowed_write_paths`. `internal/config/config.go` defaults updated; `internal/workflow/workflow.go` transition matrix updated (analyst can self-submit, three developer roles authorised for in-development→in-qa); `internal/artifact/artifact.go` `KnownTypes` and `stageToType()` extended for defects. `required_plans.ticket` set to `[plan-backend, plan-frontend, plan-test]` (gates planning→in-development). Spec (`Innovation Maker - Making Releases from Ideas-1.md`) updated in same commit: §2 personas, §4.2 type vocabulary, §5.1 directory layout, §6.2 transition matrix, §6.3 plan gating, §7.1 agent example, §13.3 example config. `tests/.gitkeep` added at repo root for test-developer's write target.
- **2026-04-25** — FE-M6 implemented: `ParseErrorsView.vue` (table of path + message, Reload button, success state), `ProjectConfigView.vue` (YAML textarea editor, unsaved-changes indicator, Save), `Graph2DView.vue` (Cytoscape + cytoscape-fcose, lazy-loaded via `defineAsyncComponent`), 3D/2D toggle in `GraphView.vue`, parse-error count badge in `AppSidebar.vue` (WS-driven refresh on `artifact.indexed`), `GET/PUT /api/p/:project/config` backend endpoint. Bundle: main 45 KB gzip; vendor-three 359 KB, vendor-codemirror 208 KB, vendor-cytoscape 176 KB — all loaded only when feature is used. TypeScript clean, 638-module Vite build clean.
- **2026-04-24** — FE-M5 implemented: `TransitionDialog.vue` (fixed status list, comment textarea for rejections, calls POST /transition, emits transitioned event), `RunAgentDialog.vue` (agent chip selector, role select, target path input, calls agentsStore.startRun), `AgentsRunsView.vue` (table of runs with live status, expandable rows showing progress lines / stderr tail / artifacts produced, Kill button for running runs), `RunStatusChip.vue` (Teleported fixed-position pulsing chip for in-flight runs, navigates to agents view). `WorkspaceView.vue` rewritten to multiplex WS events to agentsStore (agent.*) and locksStore (lock.*). `ArtifactModal.vue` and `ArtifactEditorView.vue` extended with Change Status + Run Agent toolbar buttons. `api/agents.ts` + `stores/agents.ts` (runs list, progressLines map, WS event handlers, kill action). TypeScript clean, 624-module Vite build clean.
- **2026-04-24** — FE-M3 implemented: `GraphView` (3D force graph with dark canvas, orbit/zoom controls), `ForceGraph3D.vue` (wraps 3d-force-graph; node colour by type, size by lineage index, directed arrowheads, HTML tooltips, ResizeObserver fill, `_destructor` on unmount), `GraphFilters.vue` (chip-based multi-select for type/status/lineage, live filtered node/edge count), `GraphLegend.vue` (overlay matching token colours), `ArtifactModal.vue` (Teleported overlay, fetches artifact detail from store, markdown preview, inbound/outbound edge list, Edit action navigates to editor), `useGraphData.ts` composable (fetches on mount, re-fetches on `artifact.indexed` WS event), `stores/graph.ts` (rawNodes/rawEdges, computed filteredNodes/filteredEdges from reactive filter, uniqueTypes/statuses/lineages), `api/graph.ts`. Router default `/p/:project` now goes to graph. GraphView chunk ≈ 1.3 MB gzip 363 KB due to three.js — lazy-load deferred to M6 per plan.
- **2026-04-24** — FE-M2 implemented: `ArtifactListView` (server-side filter bar for stage/status/type/label, paginated table, WebSocket `artifact.indexed` invalidation), `ArtifactEditorView` (breadcrumb nav, markdown preview, frontmatter panel, WS live-reload), artifact components (`LineageBreadcrumb`, `FrontmatterPanel`, `MarkdownPreview` with markdown-it + wiki-link inline rule → `/p/:project/artifacts?lineage=…`), `useWebSocket` composable, `stores/artifacts` (items/filter/detailCache/labels), `api/artifacts.ts`, `api/ws.ts` (WsClient with exponential backoff reconnect, singleton per project). Router extended with `artifacts/:pathMatch(.*)+` editor route. TypeScript clean (`vue-tsc --noEmit`), Vite build clean (157 modules).

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
- **FE-M2 (artifact list + read-only editor)**: `ArtifactListView` (filter bar, paginated table, WS invalidation), `ArtifactEditorView` (markdown preview, frontmatter panel, WS live-reload), `LineageBreadcrumb`, `FrontmatterPanel`, `MarkdownPreview` (markdown-it + wiki-link rule), `useWebSocket` composable, artifacts Pinia store + API layer, WsClient singleton with reconnect
- **FE-M3 (3D graph + modal)**: `GraphView`, `ForceGraph3D.vue` (3d-force-graph wrapper), `GraphFilters.vue`, `GraphLegend.vue`, `ArtifactModal.vue`, `useGraphData.ts`, `stores/graph.ts`, `api/graph.ts`; workspace default navigates to graph
- **FE-M4 (write path)**: backend lock HTTP API + file_sha in GET response; `MarkdownEditor.vue` (CodeMirror 6), `FrontmatterEditor.vue`, `LockBanner.vue`, `api/locks.ts`, `stores/locks.ts`, `composables/useLock.ts`, `composables/useExternalChange.ts`; `ArtifactEditorView` full read/edit toggle with optimistic PUT, lock lifecycle, external-change prompt
- **FE-M5 (workflow + agents)**: `TransitionDialog.vue`, `RunAgentDialog.vue`, `AgentsRunsView.vue`, `RunStatusChip.vue`; `api/agents.ts`, `stores/agents.ts`; WS multiplex for agent.*/lock.* events; Change Status + Run Agent wired into modal and editor toolbar
- **FE-M6 (config + graph 2D + polish)**: `ParseErrorsView.vue`, `ProjectConfigView.vue`, `Graph2DView.vue` (Cytoscape + fcose, lazy-loaded), 3D/2D graph toggle, parse-error badge in sidebar, backend config GET/PUT endpoint, `manualChunks` bundle split

---

## Planned

### Next: Post-M6 — E2E Testing + Hardening
- Playwright or Vitest browser-mode smoke tests for core flows
- Error boundary components for async route failures
- Skeleton loading states for artifact list and editor
- `POST /api/open-in-editor` for local editor launch (spec §16)
- `POST /api/p/:project/agents/:name/preview-prompt` (spec §16)

### Roadmap: Remaining Frontend Milestones
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
