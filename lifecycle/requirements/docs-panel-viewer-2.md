---
title: Documentation Panel Viewer
type: requirement
status: planning
lineage: docs-panel-viewer
created: "2026-05-16T00:00:00+10:00"
priority: normal
parent: lifecycle/ideas/docs-panel-viewer.md
labels:
    - feature
    - frontend
    - vue
    - usability
release: KC-Release3
assignees:
    - role: product-owner
      who: agent
---

# Documentation Panel Viewer

## Problem

Project-level reference documentation lives in the `docs/` folder at the repository root, but the SPA exposes no way to browse, search, or edit it. To read a document users must leave the app and open the file in a separate editor or file browser; to find a relevant document they must already know its filename. This breaks the "single pane of glass" goal that lifecycle artifacts already enjoy and discourages contributors from keeping `docs/` current.

## Goals / Non-goals

### Goals

- Make every file under `docs/` discoverable and openable from inside the SPA.
- Provide a card-based browsing UI with title + short summary, alphabetically sorted.
- Provide real-time client-side search filtering across visible cards.
- Allow documents to be opened in the existing markdown editor used for lifecycle artifacts, so reading and editing use the same UX.
- Keep the panel responsive to disk changes (new/edited/deleted files in `docs/`) using the existing fsnotify + WebSocket pipeline.

### Non-goals

- Indexing `docs/` files as artifacts in the SQLite cache or graph view (they are reference documentation, not lifecycle artifacts and have no `type`/`status`/`lineage`).
- Creating new documents from the UI (v1 is read + edit only).
- Deleting documents from the UI.
- Full-text search across the body of every document (v1 searches title + summary only).
- Rendering non-markdown files (PDFs, images, etc.) — these are out of scope for v1.
- Subdirectory navigation beyond a flat listing of all `.md` files found recursively.

## Detailed Requirements

### Functional

1. **Navigation entry** — A "Documentation" item must appear in the left navigation panel for every project, using a recognisable icon (e.g. book/file from `lucide-vue-next`). Clicking it routes to a new `/projects/:id/docs` view.
2. **Source folder** — The panel lists every `*.md` file found recursively under the project's `docs/` directory (path resolved through the existing sandbox). If `docs/` does not exist, the panel renders an empty state with explanatory text ("No `docs/` folder in this project").
3. **Card content** — Each card shows:
   - **Title** — taken from frontmatter `title:` if present, otherwise the first `# H1` heading, otherwise the filename without extension.
   - **Summary** — taken from frontmatter `summary:` or `description:` if present, otherwise the first non-empty, non-heading paragraph of the body, truncated to 200 characters with an ellipsis.
   - **Filename / relative path** as a secondary muted line so users can disambiguate same-titled docs.
4. **Sort order** — Cards are sorted case-insensitively, ascending, by title. Ties are broken by relative path.
5. **Search box** — A search input is pinned to the top of the panel. Typing filters the visible cards in real time (debounced ≤ 100 ms) to those whose title **or** summary contain the query (case-insensitive substring match). Clearing the input restores the full list.
6. **Open in editor** — Clicking a card opens the document in the existing markdown editor component, reusing the same save / live-refresh / disk-change behaviour as lifecycle artifacts. Edits write back to the original file on disk via a new API endpoint scoped to `docs/`.
7. **Live updates** — When a file under `docs/` is added, modified, or removed on disk, the panel updates without a full page reload (re-using the existing fsnotify watcher and WebSocket broadcast mechanism, broadcasting a new event class such as `doc.changed`).
8. **API surface** — A new backend endpoint pair under the project sandbox:
   - `GET /api/projects/:id/docs` — returns the list of docs with `path`, `title`, `summary`.
   - `GET /api/projects/:id/docs/*path` — returns the raw markdown body and any frontmatter.
   - `PUT /api/projects/:id/docs/*path` — saves edits. Path traversal protected by the existing `sandbox` package.
9. **Empty / no-match states** — When the search query matches no cards, the panel shows a "No documents match '<query>'" empty state with a clear-search action.
10. **Read-only fallback** — If the user lacks write permission for the project, the editor opens in read-only mode for `docs/` files (consistent with existing artifact editor behaviour).

### Non-functional

1. **Performance** — Initial render of the docs list must complete within 300 ms for a project with ≤ 200 docs. Search filtering must feel instant (no visible lag) up to 1,000 cards.
2. **Bundle size** — No new third-party dependency may be added; the panel reuses existing components (markdown-it, CodeMirror, lucide icons).
3. **Accessibility** — Search input has an accessible label. Cards are keyboard-focusable, activate on Enter/Space, and have visible focus styling. The list announces filter result counts to screen readers (aria-live polite).
4. **Security** — Path traversal protection on all docs endpoints (`..`, absolute paths, symlinks escaping `docs/` rejected with HTTP 400).
5. **Encoding** — Files are treated as UTF-8; non-UTF-8 files surface a card whose summary shows "(binary or non-text file — cannot preview)" and are excluded from search match logic.

## Acceptance Criteria

- [ ] "Documentation" entry appears in the left navigation for every project with a recognisable icon and is keyboard-reachable.
- [ ] Clicking the entry routes to `/projects/:id/docs` and renders a card list of all `*.md` files found recursively under `docs/`.
- [ ] Each card shows title, summary (≤ 200 chars + ellipsis), and relative path; titles fall back through frontmatter → first H1 → filename per §Detailed Requirements 3.
- [ ] Cards are sorted alphabetically by title (case-insensitive), ties broken by path.
- [ ] The search box filters cards in real time (≤ 100 ms debounce) on title or summary substring (case-insensitive).
- [ ] Clearing the search box restores the full list.
- [ ] A "no results" empty state appears when the filter matches zero cards and offers a clear-search action.
- [ ] If `docs/` is missing, a friendly empty state explains this instead of erroring.
- [ ] Clicking a card opens the file in the existing markdown editor; edits saved through the editor persist to disk under `docs/`.
- [ ] Adding, modifying, or deleting a file under `docs/` on disk updates the panel and any open editor view without a page reload.
- [ ] `GET/PUT` docs endpoints reject path-traversal attempts (`../`, absolute paths, escaping symlinks) with HTTP 400.
- [ ] Initial card list renders in ≤ 300 ms on a project with 200 docs.
- [ ] Editor honours existing project write-permission rules — read-only users see a read-only editor (see [[auth-role-checks-mutations]]).
- [ ] Live updates piggy-back on the existing fsnotify + WebSocket pipeline (see [[editor-live-refresh-on-disk-change]]).

## Resolved Questions

- Should subdirectories under `docs/` be reflected in the UI (e.g. grouped sections, breadcrumbs) or remain flat as proposed? If grouped, what grouping signal — top-level folder name, frontmatter `category:`, or alphabetical letter buckets?

> Yes, show subdirectories in alphabetical order for now.

- Are non-markdown files in `docs/` (e.g. `.pdf`, `.png`, `.txt`) to be ignored entirely, listed but unopenable, or opened in a different viewer? v1 currently ignores them.

> If other files exist in the directory, provide a link to open them for now.

- Should the "Documentation" nav entry be visible to all roles, or hidden from roles that don't need it (e.g. limited to those who can read project files)?

> All roles

- Where exactly in the left nav should "Documentation" sit — above "Artifacts", below it, or grouped with settings/help items?

> Added to bottom for now, going to restructure left menu later.

- Is creating a new doc from the UI a v1.1 follow-up, or should we leave doc creation out-of-band entirely (filesystem only)?

> out-of-band doc creation.

- Should documents be added to the activity feed when edited via the SPA, or remain silent (since they are not lifecycle artifacts)?

> make no changes to current handling of docs in the activity feed

- Does the summary extraction need to strip markdown syntax (links, emphasis, inline code) before truncation, or is the raw first-paragraph text acceptable?

> raw first paragraph works for now.
