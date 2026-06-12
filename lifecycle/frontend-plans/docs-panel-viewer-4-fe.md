---
title: "Frontend Plan — Documentation Panel Viewer"
type: plan-frontend
status: done
lineage: docs-panel-viewer
parent: lifecycle/requirements/docs-panel-viewer-2.md
created: "2026-06-12T00:00:00+10:00"
priority: normal
labels:
    - frontend
    - vue
    - feature
release: KC-Release3
---

# Frontend Plan — Documentation Panel Viewer

This plan implements the SPA-side surface for [[docs-panel-viewer-2]]: a new "Documentation" left-nav entry, a card-list view with client-side search, a markdown editor reused from the artifact pipeline, and live updates over the existing WebSocket. It consumes the API delivered by [[docs-panel-viewer-3-be]] and is verified by [[docs-panel-viewer-5-test]].

No new third-party dependency is added (Non-functional 2 of [[docs-panel-viewer-2]]).

---

## Milestone 1 — API client + types

### Description

Add a thin API client around the new docs endpoints, mirroring the pattern used by `web/src/api/artifacts.ts`.

### Files to change

- **New** `web/src/api/docs.ts`:
  - `export interface DocEntry { path: string; title: string; summary: string; is_markdown: boolean; sub_dir: string }`.
  - `export interface DocListResponse { docs: DocEntry[]; docs_dir_present: boolean }`.
  - `export interface DocReadResponse { path: string; body?: string; body_base64?: string; mime?: string; file_sha: string; is_markdown: boolean }`.
  - `export async function listDocs(project: string): Promise<DocListResponse>` → `api.get(\`/p/\${encodeURIComponent(project)}/docs\`)`.
  - `export async function getDoc(project: string, relPath: string): Promise<DocReadResponse>` → encodes each segment of `relPath` individually (`relPath.split('/').map(encodeURIComponent).join('/')`) so subdirectory slashes survive.
  - `export async function putDoc(project: string, relPath: string, body: string, expectedSha: string): Promise<{ file_sha: string }>` — same segment-by-segment encoding; body is `{ body, expected_sha: expectedSha }`.
- **New** `web/src/api/__tests__/docs.test.ts`:
  - Verifies path encoding (esp. that `subsystems/agents.md` does not get double-encoded to `subsystems%2Fagents.md`).
  - Verifies request shape for `putDoc`.

### Acceptance criteria

- `pnpm typecheck` clean.
- `pnpm test web/src/api/__tests__/docs.test.ts` green.

---

## Milestone 2 — Pinia store

### Description

A small store that owns the docs list, the current filter query, and a derived filtered+grouped list. Filtering is **client-side** (resolved question / §Performance in [[docs-panel-viewer-2]]).

### Files to change

- **New** `web/src/stores/docs.ts`:
  - State:
    - `docs: DocEntry[]` — full list from the server.
    - `docsDirPresent: boolean` — drives the "no `docs/` folder" empty state.
    - `loading: boolean`, `error: string | null`.
    - `query: string` — bound to the search input.
  - Getters:
    - `filteredDocs(): DocEntry[]` — case-insensitive substring match on `title` OR `summary` (§Detailed Requirements 5 of [[docs-panel-viewer-2]]). Non-UTF-8 cards (summary === "(binary or non-text file — cannot preview)") are excluded from match logic per §Non-functional 5: empty query shows them; a non-empty query hides them unless their title matches.
    - `groupedDocs(): { subDir: string; docs: DocEntry[] }[]` — groups by `sub_dir`, sub-groups alphabetised by name, each group's docs already sorted by the server. Root-level (`sub_dir === ''`) docs are always the first group.
  - Actions:
    - `async fetch(project: string)` — calls `listDocs`, sets `docs`, `docsDirPresent`, `loading`, `error`.
    - `setQuery(q: string)` — assigns `query`. The view applies the 100 ms debounce; the store stays synchronous so reactive recomputation is cheap.
    - `clearQuery()` — sets `query = ''`.
    - `applyDocChanged(path: string)` — refetches the full list. Re-walking via API is simpler than locally patching and is well within the 300 ms budget for ≤ 200 docs.
- **New** `web/src/stores/__tests__/docs.test.ts`:
  - `filteredDocs` returns all docs for empty query.
  - `filteredDocs` matches title and summary (case-insensitive substring).
  - `filteredDocs` excludes binary-fallback docs when query is non-empty unless the title still matches.
  - `groupedDocs` orders groups alphabetically with root first.

### Acceptance criteria

- `pnpm typecheck` clean.
- `pnpm test web/src/stores/__tests__/docs.test.ts` green.

---

## Milestone 3 — Router + sidebar entry

### Description

Wire two new routes and the left-nav entry at the **bottom** of the project navigation (resolved question in [[docs-panel-viewer-2]]: "Added to bottom for now, going to restructure left menu later"). The nav entry is visible to all roles (resolved question).

### Files to change

- **Edit** `web/src/router/index.ts`:
  - After the existing `devops` child route, add:
    ```ts
    {
      path: 'docs',
      name: 'docs',
      component: () => import('@/views/project/DocsView.vue'),
    },
    {
      path: 'docs/:pathMatch(.*)+',
      name: 'docs-editor',
      component: () => import('@/views/project/DocsEditorView.vue'),
    },
    ```
- **Edit** `web/src/components/layout/AppSidebar.vue`:
  - Import `BookOpen` from `lucide-vue-next` (recognisable book icon per §Detailed Requirements 1 of [[docs-panel-viewer-2]]).
  - Append to the `navItems` array (after `Ollama`, before the `if (hasDevOpsAccess)` block — i.e. unconditionally visible):
    ```ts
    { label: 'Documentation', to: `/p/${p}/docs`, icon: BookOpen },
    ```
  - No role gate; no badge.

### Acceptance criteria

- `pnpm typecheck` clean.
- Manual smoke: a fresh login routes to `/p/kaos-control/dashboard`; the sidebar shows "Documentation" at the bottom (above DevOps when present, below otherwise). Clicking it routes to `/p/kaos-control/docs`.
- Keyboard reachable: Tab from the sidebar header reaches "Documentation"; Enter activates it. The icon has the same `aria-label` treatment as siblings.

---

## Milestone 4 — `DocsView.vue`: card list + search + empty states

### Description

The card-list panel itself, sourced from `useDocsStore().groupedDocs`.

### Files to change

- **New** `web/src/views/project/DocsView.vue`:
  - On mount: `docsStore.fetch(projectName)`. Subscribe via `useWebSocket(projectName, 'doc.changed', () => docsStore.applyDocChanged(...))`.
  - Template structure:
    - Header row: a search `<input>` with `aria-label="Search documents"`, bound to `localQuery` (local ref). A `watchEffect` with `debounce(100ms)` mirrors `localQuery` to `docsStore.setQuery`. (Use a tiny inline debounce; no new dependency.)
    - `aria-live="polite"` status line: `"{n} of {total} documents"`. Updates as filter changes (Accessibility §Non-functional 3).
    - For each entry in `groupedDocs`:
      - If `subDir !== ''`, render `<h2 class="docs-subgroup">{{ subDir }}</h2>`.
      - Card grid: each card is a `<button>` (keyboard-focusable, activates on Enter/Space — §Non-functional 3) for markdown docs. For non-markdown docs, render an `<a :href="rawUrl(doc.path)" target="_blank" rel="noopener">` instead, so clicking opens the asset directly (resolved question: "provide a link to open them for now").
      - Card content: `<h3>{{ doc.title }}</h3>`, `<p class="summary">{{ doc.summary }}</p>`, `<span class="path-muted">{{ doc.path }}</span>` per §Detailed Requirements 3.
    - Empty states:
      - `!docsStore.docsDirPresent`: "No `docs/` folder in this project" with no further actions.
      - `docsDirPresent && groupedDocs.length === 0 && query === ''`: "This project has a `docs/` folder but it contains no markdown or supported files yet."
      - `groupedDocs.length === 0 && query !== ''`: "No documents match '<query>'." with a `<button @click="clearQuery">Clear search</button>` action (§Detailed Requirements 9).
  - Click handler on a markdown card: `router.push({ name: 'docs-editor', params: { project, pathMatch: doc.path.split('/') } })`.
  - `rawUrl(p)` returns `/api/p/${project}/docs/${p.split('/').map(encodeURIComponent).join('/')}` — the backend `GET` endpoint serves the raw bytes (encoded as `body_base64` for non-markdown). For non-markdown link-out, point the `<a>` directly at the URL so the browser handles content-type negotiation.
- **New** `web/src/views/project/__tests__/DocsView.test.ts`:
  - Renders all cards initially; renders the "no docs" empty state when API returns `docs_dir_present: false`.
  - Typing in the search box filters cards after debounce (use vi.useFakeTimers).
  - Clicking a markdown card navigates to `docs-editor`.

### Acceptance criteria

- `pnpm typecheck` clean.
- `pnpm test web/src/views/project/__tests__/DocsView.test.ts` green.
- Manual smoke: route to `/p/kaos-control/docs`, see six cards from the existing `docs/`; type "arch" and only architecture-related cards remain; clear and the full list returns; tab through cards (focus visible); Enter on a card opens the editor route.

---

## Milestone 5 — `DocsEditorView.vue`: reuse the markdown editor

### Description

Open a `docs/` file in the same `MarkdownEditor.vue` already used for lifecycle artifacts (§Detailed Requirements 6 of [[docs-panel-viewer-2]]). Reading and saving go through the new API surface from [[docs-panel-viewer-3-be]] instead of the artifact endpoints. The editor reads project write-permission via the existing role-store helpers — if the user does not hold an editor role, it renders in read-only mode (§Detailed Requirements 10 / [[auth-role-checks-mutations]]).

### Files to change

- **New** `web/src/views/project/DocsEditorView.vue`:
  - Resolves `relPath` from `route.params.pathMatch` (joined with `/`).
  - On mount: `getDoc(project, relPath)`. Holds `body`, `fileSha` in local refs.
  - Renders an existing `<MarkdownEditor>` (`web/src/components/artifact/MarkdownEditor.vue`) bound to `body`. Pass a `:readonly="!canEdit"` prop — check the component first; if a prop doesn't already exist, add a minimal one that disables editing in CodeMirror via `EditorState.readOnly.of(true)`. Reuse, not refork.
  - Save flow: a "Save" button posts via `putDoc(project, relPath, body, fileSha)`. On 409 sha-mismatch, surface a "Document was modified on disk — reload to see latest" banner with a "Reload" action (mirrors the existing artifact editor concurrency UX). On success, update `fileSha` to the response value.
  - WebSocket subscription: on `doc.changed` with matching `path`, if the editor has unsaved changes, show a non-blocking "Disk version updated" indicator. Else silently refetch and update `body` (§Detailed Requirements 7). This mirrors [[editor-live-refresh-on-disk-change]].
  - Header: file path as a non-editable breadcrumb (`docs / subsystems / agents.md`), a "Back to documents" link to `/p/${project}/docs`.
  - If `getDoc` returns a non-markdown response (`is_markdown: false`), render a fallback panel: filename + "This file type can't be edited inline." + download link.
  - 404 from `getDoc` renders an empty-state with "Document not found — it may have been removed" and a back-to-list action.
- **Edit** `web/src/components/artifact/MarkdownEditor.vue` — only if it does not already expose a `readonly` prop:
  - Add `defineProps<{ ..., readonly?: boolean }>()`. When `readonly` is `true`, append `EditorState.readOnly.of(true)` to the CodeMirror extensions list and hide the Save button.

### Acceptance criteria

- `pnpm typecheck` clean.
- Manual smoke: click a markdown card from the docs list, the file opens in the markdown editor; edit a line and Save; reload the page and the change persists. While editing, modify the same file on disk; the "Disk version updated" indicator appears.
- Read-only smoke: log in as a `qa`-only user without an editor role on the project, open a doc; the editor renders without a Save button and the textarea/editor is not editable.

---

## Milestone 6 — Live updates wiring for the list view

### Description

The list view subscribes to `doc.changed` in Milestone 4. This milestone confirms the cross-view behaviour and the empty-state transition: navigating away from `DocsView` cleans up the subscription, and an `add`-then-`delete` on disk transitions the list view through "no results" then back without a manual reload.

### Files to change

- No new files. Verify `useWebSocket` from `web/src/composables/useWebSocket.ts` handles unsubscribe on component unmount (it should, based on the pattern used in `AppSidebar.vue`).

### Acceptance criteria

- Manual smoke: with `make run`, route to `/p/kaos-control/docs`; in another terminal, `touch docs/zzz-new.md && echo '# zzz' >> docs/zzz-new.md`; the list view shows a new "zzz" card within ~200 ms without a page refresh. `rm docs/zzz-new.md` removes it within the same window.
- The browser DevTools "Network" tab shows exactly one `GET /api/p/kaos-control/docs` call per `doc.changed` event (refetch is debounced or one-shot — not a per-card patch).

---

## Risk notes

- **Re-fetch storm on bulk file changes** — if someone runs `git pull` and 50 docs change in 200 ms, the list view will fire 50 refetches. Mitigation: in `applyDocChanged`, throttle refetches to one per 150 ms with a trailing call. Implement only if a smoke test reveals visible thrash.
- **MarkdownEditor refactor scope** — adding a `readonly` prop touches a shared component used by all artifact editing. Keep the diff to the absolute minimum (prop + CodeMirror extension + Save-button conditional). No restyling, no naming changes.
- **Sub-directory path encoding** — segment-by-segment `encodeURIComponent` is mandatory; encoding the full path collapses slashes and breaks the wildcard route. Tested in `web/src/api/__tests__/docs.test.ts`.

## Verification (end-to-end)

1. `pnpm typecheck` clean.
2. `pnpm test` clean (new unit tests + existing suite green).
3. `make build-web && make build` produces a binary that serves the docs panel and editor at runtime.
4. Manual smoke covers: nav entry visible, list renders ≤ 300 ms for 200 docs (use a one-off seeded dir), search debounces and filters, empty states render, editor opens / saves / handles 409, read-only mode honours role gate, live `doc.changed` updates the list and any open editor.
