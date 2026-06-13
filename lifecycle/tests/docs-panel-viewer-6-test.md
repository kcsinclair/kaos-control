---
title: Documentation Panel Viewer — coverage map
type: test
status: draft
lineage: docs-panel-viewer
parent: lifecycle/test-plans/docs-panel-viewer-5-test.md
created: "2026-06-13T00:00:00+10:00"
---

# Documentation Panel Viewer — Coverage Map

## Automated Coverage

### Milestone 1 — Backend: list + read (`tests/integration/docs_list_test.go`)

| Test | Scenario |
|------|----------|
| `TestDocsList_EmptyWhenNoDocsDir` | `GET /docs` returns `docs_dir_present: false` and empty array when no `docs/` directory exists |
| `TestDocsList_ReturnsSortedEntries` | Three entries (root `.md`, root `.md` with H1, sub-directory `.md`) sorted case-insensitively by title; `sub_dir` field correct |
| `TestDocsList_TitleFallbackChain` | Frontmatter `title:` wins over H1; H1 wins over filename stem; filename stem used when neither is present |
| `TestDocsList_SummaryTruncation` | Body paragraph truncated to 200 runes + ellipsis; frontmatter `summary:` overrides body; frontmatter `description:` used when `summary:` absent |
| `TestDocsList_NonMarkdownEntry` | `.png` entry has `is_markdown: false`, title = filename, summary = `""` |
| `TestDocsList_NonUtf8Fallback` | `.md` file with non-UTF-8 bytes → title = filename, summary = binary fallback message |
| `TestDocsGet_HappyPath` | `GET /docs/alpha.md` returns body, `file_sha`, `is_markdown: true` |
| `TestDocsGet_Subdirectory` | `GET /docs/sub/gamma.md` returns body; wildcard route preserves subdirectory |
| `TestDocsGet_NotFound` | `GET /docs/missing.md` returns 404 with `code: "not_found"` |

### Milestone 2 — Backend: write + concurrency + permissions (`tests/integration/docs_write_test.go`)

| Test | Scenario |
|------|----------|
| `TestDocsPut_HappyPath` | PUT with valid sha; response has new `file_sha`; re-reading file shows edit |
| `TestDocsPut_ShaMismatch` | PUT with stale sha returns 409 `sha_mismatch`; file not modified on disk |
| `TestDocsPut_NotMarkdown` | PUT on `.png` returns 415 `not_markdown` |
| `TestDocsPut_CreateNotAllowed` | PUT on non-existent file returns 404 (creation is out-of-band) |
| `TestDocsPut_NoRoleForbidden` | User with no project roles → 403 `forbidden` |
| `TestDocsPut_QARoleAllowed` | `qa@test.local` (in `RolesArtifactEditors`) → 200; QA can fix doc issues directly |
| `TestDocsPut_BroadcastsDocChanged` | Hub channel receives `doc.changed` event with correct `payload.path` within 500 ms of PUT |

### Milestone 3 — Backend: path traversal + symlink rejection (`tests/integration/docs_security_test.go`)

| Test | Scenario |
|------|----------|
| `TestDocsGet_RejectsParentTraversal` | Literal `../` cleaned by `url.Parse`; URL-encoded `%2e%2e` stays literal; both return non-200 |
| `TestDocsGet_RejectsAbsolutePath` | Double-slash `//etc/passwd` normalised by chi to relative path; returns non-200 |
| `TestDocsGet_RejectsEscapingSymlink` | Symlink inside `docs/` pointing outside project root → 400 `path_traversal` (skipped on Windows) |
| `TestDocsPut_RejectsParentTraversal` | Same traversal variants on PUT; both return non-200 |

### Milestone 4 — Watcher unit tests for `doc.changed` (`internal/watcher/docs_test.go`)

Previously existing unit tests (call `handleDocChange` directly — no fsnotify):

| Test | Scenario |
|------|----------|
| `TestHandleDocChange_Create` | File created in `docs/` → `doc.changed` with `path = "architecture.md"` |
| `TestHandleDocChange_SubDir` | File in `docs/subsystems/` → `doc.changed` with `path = "subsystems/agents.md"` |
| `TestHandleDocChange_DeletedFile` | Non-existent path → `doc.changed` emitted (covers deletion) |
| `TestIsDocsFile_LifecyclePath_NotDocs` | Lifecycle path is not a docs file |
| `TestIsDocsFile_DocsPath` | Docs path is correctly identified |

New Start()-based tests (real fsnotify, 150 ms debounce):

| Test | Scenario |
|------|----------|
| `TestWatcher_DocCreateEmitsDocChanged` | Creating `docs/a.md` emits `doc.changed` with `path = "a.md"` within 500 ms |
| `TestWatcher_DocModifyEmitsDocChanged` | Modifying `docs/a.md` emits at least one `doc.changed` event |
| `TestWatcher_DocDeleteEmitsDocChanged` | Deleting `docs/a.md` emits `doc.changed`; nil index not touched (docs skip indexer) |
| `TestWatcher_LifecycleStillEmitsFileChanged` | Writing `lifecycle/ideas/x.md` emits `file.changed`; no `doc.changed` emitted |
| `TestWatcher_MissingDocsDirIsNonFatal` | Starting watcher without a `docs/` directory causes no panic or early exit |

### Milestone 5 — Frontend Vitest coverage

Delegated to the frontend-developer agent. Files owned:

- `web/src/api/__tests__/docs.test.ts`
- `web/src/stores/__tests__/docs.test.ts`
- `web/src/views/project/__tests__/DocsView.test.ts`

---

## Manual Smoke Checklist (Milestone 6)

To be executed by the QA agent against `make run` before the requirement transitions to `done`.

- [ ] Fresh login, navigate to `/p/kaos-control/docs`. Six existing cards render in ≤ 300 ms.
- [ ] Cards are sorted alphabetically (case-insensitive). Non-markdown entries render as links, not buttons.
- [ ] Typing `arch` in the search box filters the list within 100 ms; `aria-live` count updates.
- [ ] Clearing the search restores the full list.
- [ ] Searching for `nomatch-xyz` shows the "No documents match 'nomatch-xyz'" empty state with a working "Clear search" action.
- [ ] Renaming `docs/` away on disk triggers the panel's "No `docs/` folder" empty state without a page reload. Renaming back restores it.
- [ ] Opening `docs/architecture.md` loads the markdown editor with the file body.
- [ ] Saving an edit persists to disk; reopening the file shows the edit.
- [ ] Modifying the file on disk while the editor is open shows the "Disk version updated" indicator.
- [ ] Logging in as `qa@test.local` and opening a doc renders an editable editor and a `PUT` save succeeds (QA is in `RolesArtifactEditors`). A user with no project roles instead gets a 403 on save.
- [ ] `PUT /api/p/{project}/docs/../escape.md` rejected with HTTP 400 (curl).
- [ ] Tab through the panel from the sidebar: focus reaches "Documentation", Enter activates; Tab advances through the search input and each card; Enter on a card opens the editor.
