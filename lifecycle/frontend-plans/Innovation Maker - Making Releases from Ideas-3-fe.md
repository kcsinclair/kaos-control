---
title: Frontend Development Plan — kaos-control v1
type: plan-frontend
status: planning
parent: requirements/Innovation Maker - Making Releases from Ideas-1.md
lineage: innovation-maker
labels: [frontend, vue, vite, v1]
---

> Target implementer: developer agent (Sonnet). Produces a Vue 3 + Vite SPA that is embedded into the Go binary via `embed.FS`. All section numbers in the form §N.N refer to the parent requirements document. API and event contracts come from the backend plan (sibling).

## 1. Scope

### In scope
- All UI surfaces implied by §9 (graph), §10 (UI + editor), and §13 (config CRUD) of the spec.
- 3D graph (primary) and 2D graph (alternative) with shared data model.
- Simple markdown editor with split preview.
- Login, project picker, project workspace, agent runs panel, parse-errors display.
- Realtime via the WebSocket protocol defined in the backend plan §14.

### Out of scope (roadmap or other plans)
- WYSIWYG editor (§10.2 — v1 is split-pane markdown).
- CRDT / real-time co-editing (§16).
- JIRA integration UI (§16).
- Unit-test code authoring for backend endpoints (lives in the test plan).
- Any implementation of the embedded static asset serving; the backend plan owns `embed.FS` wiring — this plan owns producing `web/dist/`.

## 2. Directory Layout

Frontend lives under `web/` alongside the Go code. Vite builds into `web/dist/` which the Go binary embeds (§12.3 of spec).

```
web/
├── index.html
├── package.json
├── pnpm-lock.yaml             # prefer pnpm for speed; npm acceptable
├── vite.config.ts
├── tsconfig.json
├── public/                    # static assets (favicon, etc.)
└── src/
    ├── main.ts                # app entry, Pinia + Vue Router bootstrap
    ├── App.vue                # root layout
    ├── api/
    │   ├── client.ts          # typed fetch wrapper, CSRF handling, error normalisation
    │   ├── artifacts.ts
    │   ├── projects.ts
    │   ├── agents.ts
    │   ├── auth.ts
    │   ├── config.ts
    │   └── ws.ts              # WebSocket client with reconnect + typed events
    ├── stores/                # Pinia
    │   ├── auth.ts
    │   ├── project.ts
    │   ├── artifacts.ts
    │   ├── graph.ts
    │   ├── agents.ts
    │   └── locks.ts
    ├── router/
    │   └── index.ts           # / (login), /projects, /p/:project/*
    ├── views/
    │   ├── LoginView.vue
    │   ├── ProjectPickerView.vue
    │   └── project/
    │       ├── WorkspaceView.vue      # shell: left nav + main + right pane
    │       ├── GraphView.vue          # 3D graph (primary)
    │       ├── Graph2DView.vue        # Cytoscape alternative
    │       ├── ArtifactListView.vue   # table view
    │       ├── ArtifactEditorView.vue # split editor
    │       ├── AgentsRunsView.vue
    │       ├── ParseErrorsView.vue
    │       └── ProjectConfigView.vue  # lifecycle/config.yaml editor
    ├── components/
    │   ├── graph/
    │   │   ├── ForceGraph3D.vue       # wraps 3d-force-graph
    │   │   ├── ForceGraph2D.vue       # wraps cytoscape
    │   │   ├── GraphFilters.vue       # sidebar filters
    │   │   └── GraphLegend.vue
    │   ├── artifact/
    │   │   ├── ArtifactModal.vue      # node-click preview with action bar
    │   │   ├── ArtifactActions.vue    # edit/transition/run-agent/history/delete
    │   │   ├── FrontmatterPanel.vue
    │   │   ├── MarkdownEditor.vue     # CodeMirror 6
    │   │   ├── MarkdownPreview.vue    # markdown-it + wiki-link resolver
    │   │   └── LineageBreadcrumb.vue
    │   ├── agent/
    │   │   ├── RunAgentDialog.vue
    │   │   └── RunStatusChip.vue
    │   ├── auth/
    │   │   └── LoginForm.vue
    │   ├── common/
    │   │   ├── Toast.vue
    │   │   ├── ConfirmDialog.vue
    │   │   ├── LockBanner.vue
    │   │   └── LoadingShimmer.vue
    │   └── layout/
    │       ├── AppHeader.vue
    │       ├── AppSidebar.vue
    │       └── RightPane.vue
    ├── composables/
    │   ├── useWebSocket.ts
    │   ├── useGraphData.ts
    │   ├── useLock.ts              # heartbeat + release
    │   ├── useExternalChange.ts    # prompt-to-reload flow
    │   └── useKeyboardShortcuts.ts
    ├── types/
    │   ├── api.ts                  # mirrors backend DTOs
    │   └── events.ts               # WebSocket event types
    └── styles/
        ├── main.css
        └── tokens.css              # CSS custom properties (colour, spacing, type)
```

## 3. Dependencies

| Purpose | Package | Notes |
|---|---|---|
| Framework | `vue` (3.4+) | Composition API, `<script setup>` |
| State | `pinia` | |
| Router | `vue-router` | |
| Build | `vite`, `@vitejs/plugin-vue`, `typescript` | |
| 3D graph | `3d-force-graph` (+ `three`) | |
| 2D graph | `cytoscape`, `cytoscape-fcose` | fcose layout handles larger graphs better than default |
| Markdown render | `markdown-it`, `markdown-it-anchor` | plus a wiki-link plugin (custom, ~50 LOC) |
| Code editor | `@codemirror/view`, `@codemirror/lang-markdown`, `@codemirror/lang-yaml` | CodeMirror 6 |
| Icons | `lucide-vue-next` | tree-shakeable |
| HTTP | stdlib `fetch` (no axios) | typed wrapper in `api/client.ts` |
| WebSocket | stdlib `WebSocket` | reconnect logic in `api/ws.ts` |
| Forms | `@vuelidate/core` OR native HTML validation | small — prefer native where possible |
| Date utils | `date-fns` | tree-shakeable |
| Testing (unit) | `vitest`, `@vue/test-utils`, `jsdom` | test plan owns strategy |

**Styling**: defer Tailwind vs custom CSS decision until M2 (per §17 of spec, parked). Start with plain CSS + design tokens in `tokens.css`. If the component count bloats, switch to Tailwind in one commit.

## 4. Routing

```
/                                -> redirect to /login or /projects based on auth
/login                           -> LoginView
/projects                        -> ProjectPickerView (list, create, edit, delete)
/p/:project                      -> WorkspaceView (default: GraphView)
/p/:project/graph                -> GraphView (3D)
/p/:project/graph2d              -> Graph2DView (Cytoscape)
/p/:project/artifacts            -> ArtifactListView
/p/:project/artifacts/*path      -> ArtifactEditorView
/p/:project/agents               -> AgentsRunsView
/p/:project/parse-errors         -> ParseErrorsView
/p/:project/config               -> ProjectConfigView
```

Auth-required routes are guarded by a navigation guard that checks `/api/auth/me` on first load and redirects unauthenticated requests to `/login`.

## 5. State (Pinia) Model

### `useAuthStore`
- `user`, `isAuthenticated`, `rolesByProject`, `login()`, `logout()`, `fetchMe()`.

### `useProjectStore`
- Current project name, config, stage list, role list, agent list.
- Loaded on route change to `/p/:project/*`.

### `useArtifactsStore`
- In-memory cache keyed by path. Invalidated on `artifact.indexed` / `file.changed` events.
- `list(filter)` returns a ref that auto-updates on events matching the filter.

### `useGraphStore`
- Derived from artifacts + links. Recomputed on relevant events.
- Provides `{nodes, edges}` shaped for both 3d-force-graph and Cytoscape.
- Filter state: selected types, statuses, labels, sprints, releases, lineages.

### `useAgentsStore`
- Active runs (realtime via events). History (fetched on demand).
- `trigger(agent, targetPath)`, `kill(runId)`.

### `useLocksStore`
- Map lineage → `{holder, kind, acquiredAt}`.
- Updated from `lock.acquired` / `lock.released` events.
- Exposes `isLockedByMe(lineage)` and `heldByOther(lineage)`.

## 6. WebSocket Client

`src/api/ws.ts` owns a single singleton per project:

- Connects to `/api/p/:project/ws` after login.
- Auto-reconnect with exponential backoff (100 ms → 30 s cap).
- Heartbeats: client sends `lock.heartbeat` every 30 s for lineages it holds.
- Dispatches inbound messages to Pinia stores via a typed event bus.
- Connection status reflected in `AppHeader.vue` (green / amber / red dot).

## 7. Graph UI

### `ForceGraph3D.vue`
- Wraps `3d-force-graph` via `onMounted(() => forceGraph()(el.value).graphData(props.data))`.
- **Nodes**: size by lineage fan-out; colour by `type`; tooltip shows title + status.
- **Edges**: colour by `kind` per §9.1 of spec; arrowheads on directed kinds.
- **Click**: emits `node-click` → opens `ArtifactModal.vue`.
- **Hover**: highlights connected edges and neighbours.
- **Performance**: for > 2000 nodes, switch to a lineage-aggregated view (one node per lineage) and expand on click. Threshold configurable in settings.

### `ForceGraph2D.vue`
- Same data, Cytoscape + `fcose` layout.
- Supports "export as PNG/SVG" for printable dependency charts.

### `GraphFilters.vue`
- Multi-select for each of: type, status, label, sprint, release, role, lineage.
- Saved views stored in `localStorage` under `kc:savedViews:<project>`.

## 8. Artifact Modal & Editor

### `ArtifactModal.vue`
- Opens when a graph node is clicked (any node type — tickets, releases, agents, labels).
- **Top**: action bar per §9.3 of spec: Edit • Change state • Run agent • Open in IDE • View git history • Delete.
- **Body**: rendered markdown preview (markdown-it + wiki-link plugin).
- **Sidebar**: structured frontmatter (labels as chips, links as click-throughs).
- **Footer**: inbound/outbound edges listed by kind.
- "Open in IDE" posts `POST /api/open-in-editor` (new endpoint — coordinate with backend plan if missing) which launches `$EDITOR` or `code` with the absolute path.

### `ArtifactEditorView.vue`
- CodeMirror 6 with markdown + YAML frontmatter language modes.
- **Split pane**: left editor, right live preview. Preview re-renders on debounced input (150 ms).
- **Wiki-link autocomplete**: typing `[[` triggers a searchable dropdown of slugs from `useArtifactsStore`.
- **Save**: `Cmd/Ctrl+S` or an auto-save toggle. Saves via `PUT /artifacts/*path` with the last-known `sha` for optimistic concurrency.
- **Lock banner**: top-of-view banner when the lineage is locked by another holder (read-only mode). When the user is editing, we hold the lock and send heartbeats.
- **External change prompt**: `useExternalChange.ts` subscribes to `file.changed` for the active file; on change not originated by our save, shows a modal with Reload / Keep editing options per §10.3.
- **Keyboard shortcuts**: save, toggle preview, insert wiki-link (`Cmd+K`), toggle frontmatter editor.

### `FrontmatterPanel.vue`
- Typed inputs for known fields (`title`, `type`, `status`, `lineage`, `labels`, `release`, `sprint`, `depends_on`, `blocks`, `related_to`, `assignees`).
- Unknown (extra) fields shown as a raw-YAML textarea and preserved on save.
- Validation surfaces the same errors the backend would produce, client-side first.

## 9. Agent UI

### `RunAgentDialog.vue`
- Dropdown of agents whose configured role is authorised for the current artifact's stage.
- Shows a preview of the rendered prompt before confirming (pulled via a new endpoint `POST /api/p/:project/agents/:name/preview-prompt` — coordinate with backend plan).
- Confirm → `POST /agents/:name/run` → closes dialog, opens the run status chip.

### `AgentsRunsView.vue`
- Table of current and historical runs with status chip, start/end time, target path (click-through), elapsed time.
- Expandable row shows the tail of stdout/stderr.
- Kill button on running rows.

### `RunStatusChip.vue`
- Small floating component that appears when the current user has a run in flight. Clicking it opens `AgentsRunsView.vue` filtered to that run.

## 10. Project Config UI

### `ProjectConfigView.vue`
- Edits `lifecycle/config.yaml`.
- Form-based for known sections (stages, roles, users, agents, required_plans, git), with a "raw YAML" tab for power users.
- Validates on submit; shows backend validation errors inline.

## 11. Project Picker UI

### `ProjectPickerView.vue`
- Lists registered projects from `/api/projects`.
- Add / Edit / Delete → modals that CRUD project registry files (§13.2 of spec).
- Admin-only for create/edit/delete (UI reflects the user's role).

## 12. Theming & Styling

- CSS custom properties in `tokens.css`: colour palette (light + dark), type scale, spacing scale, radii, shadows.
- Dark mode via `@media (prefers-color-scheme)` with a user override toggle stored in `localStorage`.
- Node colours for the graph live in `tokens.css` as `--node-type-*` variables so legend + graph stay in sync.
- Defer Tailwind vs custom decision until M4.

## 13. Accessibility

- Keyboard-navigable everywhere; focus ring honours `:focus-visible`.
- Graph has a keyboard-navigable fallback (list view at `/p/:project/artifacts` is the canonical alternative).
- Colour-coding paired with iconography (never colour alone).
- Modals trap focus; `Escape` closes.
- WCAG AA contrast; automated check in test plan.

## 14. Error Handling

- Central `api/client.ts` normalises errors into `{code, message, detail}`.
- `ToastStore` for transient errors; in-page banner for persistent ones (e.g., lock contention).
- Offline detection: WebSocket close → banner; queued mutations resurface when the socket reconnects.

## 15. Milestones

### M1 — Scaffold (≈ 2 days)
- Vite + Vue 3 + TS + Pinia + Router.
- Login + `/projects` + `/p/:project` shell with placeholder panes.
- API client wrapper and auth store.
- `make build-web` and Go `embed.FS` handshake working (coordinate with backend M1).
- **Acceptance**: login, pick a project, see an empty workspace. No graph yet.

### M2 — Artifact List & Editor Read-Only (≈ 3 days)
- `ArtifactListView` with server-side filtering (stage/status/label).
- `ArtifactEditorView` read-only: CodeMirror + markdown-it preview + wiki-link rendering.
- Frontmatter panel as read-only summary.
- **Acceptance**: pointed at this repo, every existing artifact is listed and previews cleanly with wiki-link navigation.

### M3 — 3D Graph & Modal (≈ 3 days)
- `ForceGraph3D` + `GraphFilters` + `GraphLegend`.
- `ArtifactModal` with action bar (Edit and Open-in-IDE only in this milestone).
- `useGraphData` composable driving both graph and filters.
- **Acceptance**: the graph renders this repo's lineage correctly; filters by type/status/label work; node click opens modal.

### M4 — Write Path (≈ 3 days)
- Editor save flow (PUT with `expected_sha`).
- Frontmatter panel editable with validation.
- Wiki-link autocomplete and slug-rename UI.
- External-change prompt.
- Lock banner + heartbeat.
- **Acceptance**: editing an artifact in the GUI commits via the backend, emits events, and the graph updates live in another browser tab.

### M5 — Workflow + Agents (≈ 3 days)
- Transition action in modal + editor toolbar (role-aware dropdown).
- Rejected → capture feedback modal → new child artifact.
- `RunAgentDialog` with prompt preview.
- `AgentsRunsView` + `RunStatusChip` + kill flow.
- WebSocket event wiring for agent.* and lock.*.
- **Acceptance**: triggering the planner agent on this requirements doc from the UI produces visible run progress and the three plan artifacts appear on the graph as they land.

### M6 — Config, 2D Graph, Parse Errors, Polish (≈ 3 days)
- `ProjectConfigView` (form + raw YAML).
- `Graph2DView` with Cytoscape + fcose.
- `ParseErrorsView` + header badge.
- Dark mode, accessibility pass, loading states, empty states.
- Build optimisation (code-split per route, graph libs lazy-loaded).
- **Acceptance**: production build < 500 KB gzip for the main bundle; Lighthouse accessibility ≥ 95; graph libs only load on graph routes.

**Total**: ≈ 17 working days for a single agent.

## 16. Coordination with Backend Plan

- Any change to the REST or WebSocket contract requires a paired change in the backend plan document AND the code. The backend plan is authoritative.
- New endpoints this plan needs (coordinate before M4):
  - `POST /api/open-in-editor {path, editor?}` — launches local editor; returns 202 or 501 if the app wasn't started with `--allow-launch-editor`.
  - `POST /api/p/:project/agents/:name/preview-prompt {target_path}` — returns the rendered prompt string without starting the agent.
- The WebSocket event payload for `artifact.indexed` must include enough info to drive list + graph updates without a full refetch. If the payload ships a full `Frontmatter + links`, great; otherwise the frontend will refetch the artifact on the event.

## 17. Cross-Plan Coordination

- **Backend plan**: consumed (see §16 above).
- **Test plan**: end-to-end tests drive the UI via Playwright. Component unit tests live alongside the source (`*.spec.ts`) and are co-owned — component invariants are stated in this plan, test scripts in the test plan.

## 18. Risks

| Risk | Mitigation |
|---|---|
| 3d-force-graph perf cliff at scale | Aggregated lineage-view fallback from M3; threshold configurable. |
| Conflicting edits (GUI + IDE) | Lock banner + external-change prompt; disk wins (§10.3 of spec). |
| CodeMirror + Vue reactivity gotchas | Wrap CodeMirror imperatively inside a stable ref; never proxy its state. |
| WebSocket message storms on large imports | Client-side debounced refresh of the graph; batched UI updates via `requestAnimationFrame`. |
| Embedding bundle bloat | Code-split routes; dynamic import for graph libs; run `rollup-plugin-visualizer` in CI. |

## 19. Open Questions for the Developer

- **Tailwind vs custom CSS**: decide in M4 with a spike (the §17 spec question).
- **CodeMirror vs Monaco**: plan says CodeMirror 6 (lighter). Re-evaluate if YAML frontmatter editing UX needs Monaco's heavier tooling.
- **Markdown rendering security**: use a strict sanitiser profile in markdown-it; explicitly disable raw HTML in rendered output unless we decide to opt in for prototypes.

## 20. References

- Parent spec: [[requirements/Innovation Maker - Making Releases from Ideas-1]]
- Backend plan (sibling): [[backend-plans/Innovation Maker - Making Releases from Ideas-2-be]]
- Test plan (sibling): [[test-plans/Innovation Maker - Making Releases from Ideas-4-test]]
