---
title: "Backend Plan — Documentation Panel Viewer"
type: plan-backend
status: done
lineage: docs-panel-viewer
parent: lifecycle/requirements/docs-panel-viewer-2.md
created: "2026-06-12T00:00:00+10:00"
priority: normal
labels:
    - backend
    - feature
release: KC-Release3
---

# Backend Plan — Documentation Panel Viewer

This plan implements the server-side surface for [[docs-panel-viewer-2]]: an HTTP API that lists, reads, and writes markdown files under each project's `docs/` directory, plus filesystem-watcher integration so the existing fsnotify + WebSocket pipeline broadcasts disk events for those files. `docs/` files are **not** indexed in SQLite — they are surfaced lazily on each list request.

Cross-references:
- [[docs-panel-viewer-4-fe]] — Frontend plan (cards + editor + WS subscription).
- [[docs-panel-viewer-5-test]] — Test plan.
- [[auth-role-checks-mutations]] — write-permission gate reused on `PUT /docs/*`.

---

## Milestone 1 — `internal/docs` package: scan + parse

### Description

Create `internal/docs/` to encapsulate the logic for discovering markdown files under `<projectRoot>/docs/` and extracting a title/summary preview from each one. This package owns no HTTP, no fsnotify, and no project lifecycle — it is pure I/O over a directory.

### Files to change

- **New** `internal/docs/docs.go`:
  - `type DocEntry struct { Path string; Title string; Summary string; IsMarkdown bool; SubDir string }`.
    - `Path` is `filepath.ToSlash`-normalised, relative to `<projectRoot>/docs/` (e.g. `architecture.md`, `subsystems/agents.md`).
    - `SubDir` is the immediate parent directory of the file relative to `docs/`, or `""` for the root level. The frontend uses this to render alphabetical sub-group headings (resolved question in [[docs-panel-viewer-2]]).
    - `IsMarkdown` is `true` iff the lowercased extension is `.md` or `.markdown`.
  - `func List(projectRoot string) ([]DocEntry, error)`:
    - Returns `nil, nil` (not an error) when `<projectRoot>/docs/` does not exist — the frontend renders an "empty state" for this case.
    - Walks `<projectRoot>/docs/` recursively via `filepath.WalkDir`.
    - Skips entries whose name begins with `.` (dotfiles, e.g. `.DS_Store`).
    - For each `.md`/`.markdown` file, extracts `Title` and `Summary` via `extractPreview` (below). For every other regular file, sets `IsMarkdown=false`, `Title = filepath.Base(path)`, `Summary = ""` — the frontend renders these as a plain link per the resolved question in [[docs-panel-viewer-2]].
    - Returns the entries in stable insertion order (caller sorts).
  - `func extractPreview(absPath string) (title string, summary string, err error)`:
    - Reads the file (cap at 64 KiB read for preview extraction — full body is only loaded by `Read`/the editor save path).
    - If the contents are not valid UTF-8 per `utf8.Valid`, returns `title = filepath.Base(absPath)` and `summary = "(binary or non-text file — cannot preview)"` with `err = nil`. (§Non-functional 5 of [[docs-panel-viewer-2]].)
    - Reuses `artifact.Parse(raw, relPath, mtime)` to get any YAML frontmatter and the body. Title resolution:
      1. `fm.Title` if non-empty.
      2. Else the first `# ` H1 line of the body.
      3. Else `strings.TrimSuffix(filepath.Base(absPath), ext)` with extension stripped.
    - Summary resolution:
      1. `fm.Summary` (new field — see below) if non-empty.
      2. Else `fm.Description` if non-empty.
      3. Else the first non-empty, non-heading, non-fenced-code paragraph of the body. Stop at the first blank line. Raw markdown — no syntax stripping (resolved question).
    - Truncates summary to 200 runes (not bytes — multi-byte safe) and appends `"…"` if it was longer.
  - `func Read(projectRoot, relPath string) (raw []byte, err error)`:
    - Resolves through `sandbox.Resolve(filepath.Join(projectRoot, "docs"), relPath)`.
    - Reads the file and returns the raw bytes.
  - `func Write(projectRoot, relPath string, contents []byte) error`:
    - Resolves through sandbox (same as `Read`).
    - Refuses to create new files: returns `ErrNotFound` if the target does not already exist on disk. Doc creation is out-of-band (resolved question in [[docs-panel-viewer-2]]).
    - Uses an atomic write: temp file in the same directory then `os.Rename`, mode preserved from the existing file's `os.Stat`.
  - Exported errors: `ErrNotFound`, `ErrPathTraversal` (wraps `sandbox.ErrPathTraversal`).

- **Edit** `internal/artifact/frontmatter.go` (or wherever `Frontmatter` struct lives — verify): add an optional `Summary string \`yaml:"summary"\`` field. The artifact pipeline ignores it; it's only consumed by `internal/docs`. If `summary:` is already supported anywhere else, do nothing.

### Acceptance criteria

- `go build ./internal/docs/...` clean.
- `go vet ./internal/docs/...` clean.
- Unit test `internal/docs/docs_test.go` (in this same milestone — it is package-internal, not an integration test):
  - Given a tmpdir with `docs/a.md` (frontmatter title) and `docs/sub/b.md` (no frontmatter, H1 first line), `List` returns 2 entries with correct titles and `SubDir` values.
  - `List` on a project root with no `docs/` returns `(nil, nil)` (not an error).
  - `extractPreview` on a UTF-8 file with a 500-char first paragraph returns a summary of exactly 200 runes plus `…`.
  - `extractPreview` on a non-UTF-8 file returns the binary-fallback summary.
  - `extractPreview` on `# Title\n\nFirst para\n\nSecond para` returns `Title = "Title"`, `Summary = "First para"`.
  - `Read`/`Write` reject `..` (e.g. relative `../README.md`), absolute paths, and symlinks whose target escapes `docs/` — each returns an error wrapping `ErrPathTraversal`.
  - `Write` returns `ErrNotFound` when the target file does not exist.

---

## Milestone 2 — HTTP endpoints

### Description

Mount three handlers under the existing `/api/p/{project}` group in [internal/http/server.go](internal/http/server.go). Wildcard sub-paths are dispatched manually (matching the artifacts pattern at lines 206–251) because chi greedy wildcards can't share a prefix with a sibling exact match.

### Files to change

- **New** `internal/http/docs.go`:
  - `handleListDocs(w, r)` — `GET /api/p/{project}/docs`:
    - Pulls project from context (`projectFromCtx`).
    - Calls `docs.List(p.Entry.Path)`. On `(nil, nil)` returns `200 { "docs": [], "docs_dir_present": false }`. Otherwise returns `200 { "docs": [...], "docs_dir_present": true }`.
    - Response items: `{ "path": "subsystems/agents.md", "title": "Agents", "summary": "…", "is_markdown": true, "sub_dir": "subsystems" }`.
    - Sorted server-side by `strings.ToLower(title)` ascending, ties broken by `path` ascending — saves the frontend from re-sorting on every keystroke. Per §Detailed Requirements 4 of [[docs-panel-viewer-2]].
  - `handleGetDoc(w, r)` — `GET /api/p/{project}/docs/*path`:
    - Extracts `relPath` from `chi.URLParam(r, "*")`.
    - On `sandbox.ErrPathTraversal` or `sandbox.ErrAbsolutePath` returns 400 `apiError("path_traversal", ...)`.
    - Returns `200 { "path": relPath, "body": "<raw markdown>", "file_sha": "<sha256 hex>", "is_markdown": bool }`.
    - For non-markdown files: returns `200` with body as a base64-encoded `body_base64` field instead of `body`, plus `mime` (best-effort via `http.DetectContentType` on the first 512 bytes). The frontend uses this to render a download link for non-markdown content (resolved question).
    - 404 when the file does not exist.
  - `handlePutDoc(w, r)` — `PUT /api/p/{project}/docs/*path`:
    - Role gate: `if !requireRole(w, r, p, RolesArtifactEditors...) { return }` (reuse helpers from [[auth-role-checks-mutations]]).
    - Body: `{ "body": "<markdown>", "expected_sha": "<sha256>" }`.
    - If `expected_sha` is set and does not match the current file's sha, returns 409 `apiError("sha_mismatch", ...)` — same optimistic-concurrency contract as `handleUpdateArtifact`.
    - On success: writes via `docs.Write`, then synchronously broadcasts `doc.changed` to `p.Hub` (the watcher will also fire, but the synchronous broadcast removes a 150 ms perceived delay for the writing client). Returns `200 { "file_sha": "<new sha>" }`.
    - 400 on `ErrPathTraversal`; 404 on `ErrNotFound`; 415 (`apiError("not_markdown", …)`) when the target is not a markdown file (writes are only allowed for editable markdown).
- **Edit** `internal/http/server.go`:
  - In the `/p/{project}` block (after the artifacts routes at line ~252), add:
    ```go
    r.Get("/docs", s.handleListDocs)
    r.Get("/docs/*", s.handleGetDoc)
    r.Put("/docs/*", s.handlePutDoc)
    ```
  - No POST or DELETE: doc creation is out-of-band and deletion is out of scope per Non-goals in [[docs-panel-viewer-2]].

### Acceptance criteria

- `go build ./...` clean.
- `go vet ./...` clean.
- Integration tests in [[docs-panel-viewer-5-test]] cover happy-path list/get/put, path traversal (400), sha mismatch (409), not-markdown writes (415), and the read-only role gate (403 for a user without editor role).
- Manual smoke: `curl http://localhost:8080/api/p/kaos-control/docs` returns the six existing docs in `docs/` (`architecture.md`, `architecture-diagram.html`, `architecture-summary.md`, `architecture.png`, `end-to-end-smoke-tests.md`, `test-everything.md`). The `.html`/`.png` entries have `is_markdown: false`.

---

## Milestone 3 — Watcher extension: emit `doc.changed`

### Description

Extend [internal/watcher/watcher.go](internal/watcher/watcher.go) to recurse into `<projectRoot>/docs/` in addition to `lifecycle/`, and emit a new `doc.changed` event on `p.Hub` whenever a file under `docs/` is created, modified, or removed. **Do not** call `idx.IndexFile` for docs paths — these files are not artifacts.

### Files to change

- **Edit** `internal/watcher/watcher.go`:
  - Add a field `docsDir string` to `Watcher` and set it in `New` to `filepath.Join(projectRoot, "docs")`.
  - In `Start`, after `addDirRecursive(w.lifecycleDir)`, also call `addDirRecursive(w.docsDir)` — but treat a missing directory as non-fatal (project may not have a `docs/` folder). The error from `addDirRecursive` should be logged at info level and swallowed when the root does not exist (use `errors.Is(err, fs.ErrNotExist)`).
  - In the event-loop dispatch, before the existing `shouldProcess` lifecycle check, add a branch:
    ```go
    if w.isDocsFile(evt.Name) {
        path := evt.Name
        fire(evt.Name, func() { w.handleDocChange(path) })
        if evt.Has(fsnotify.Create) { _ = w.fsw.Add(evt.Name) }
        continue
    }
    ```
  - Add `func (w *Watcher) isDocsFile(p string) bool` analogous to `isReleaseFile`.
  - Add `func (w *Watcher) handleDocChange(absPath string)`:
    - Computes `relPath` rooted at `docsDir` (mirroring the symlink-safe resolution in `handleChange`).
    - Broadcasts `hub.Event{ Type: "doc.changed", Payload: map[string]string{"path": relPath} }`.
    - No indexing, no triage, no defect detection.

### Acceptance criteria

- `go test ./internal/watcher/... -short` passes (existing tests untouched).
- New unit test `internal/watcher/docs_test.go`: creating, modifying, and deleting a file under a tmpdir `docs/` triggers a `doc.changed` event each time, with `path` matching the relative path. Modifying a file under `lifecycle/` does **not** emit `doc.changed`.
- Manual smoke: with `make run`, edit `docs/architecture.md` in the project root; the WebSocket connection at `/api/p/kaos-control/ws` receives a `doc.changed` event with `payload.path = "architecture.md"`.

---

## Milestone 4 — Cross-cutting: register WebSocket event type and exclude `docs/` from the lifecycle ignore patterns

### Description

The hub broadcasts events to all WebSocket subscribers; no allow-listing is needed there. But two small consistency tasks:

1. The project loader's startup scan (in `internal/index/scan.go` or equivalent) walks `lifecycle/`, not the repo root, so it already ignores `docs/`. Confirm this — no change should be required.
2. The `internal/config.ShouldIgnore` function may be invoked over docs paths by the new watcher branch. Audit to make sure docs entries are not accidentally suppressed by the project's `ignore:` config (which is intended for `lifecycle/`, not `docs/`).

### Files to change

- **Edit** `internal/watcher/watcher.go`:
  - In `handleDocChange`, **do not** consult `config.ShouldIgnore` — docs ignore rules are not user-configurable in v1. The artifact-only ignore list stays scoped to the lifecycle branch.
- **Edit** `internal/http/ws.go` (verify file name via grep): no change. The hub already proxies arbitrary event types to subscribers.

### Acceptance criteria

- `go vet ./...` clean.
- Grep confirms no other code paths short-circuit on event type — new types pass through to the WebSocket layer untouched.

---

## Risk notes

- **Concurrency on `Write`** — the optimistic `expected_sha` check is not transactional with the rename. Two simultaneous writers with the same `expected_sha` can both pass the check before either writes. This matches the existing `handleUpdateArtifact` behaviour and is acceptable for v1 single-user-editing semantics; documented for future hardening.
- **Large `docs/` trees** — `List` re-walks the directory on every `GET /docs`. For ≤ 200 docs this is ≤ 30 ms on warm cache. Caching is out of scope; if needed later, mirror the SQLite cache pattern but with no schema (a flat in-memory map keyed by mtime).
- **Symlinks** — `sandbox.Resolve` already follows and rejects symlinks that escape root. The new `docs/` root is passed in instead of `projectRoot`, so a `docs/foo.md → ../../etc/passwd` symlink is rejected.

## Verification (end-to-end)

1. `make lint` clean.
2. `make test-unit` clean.
3. `make test-integration` clean (new tests in [[docs-panel-viewer-5-test]]).
4. Manual smoke: list, open, edit, and save a doc through the API; observe the synchronous `doc.changed` broadcast and the watcher-emitted broadcast (debounced 150 ms later). Confirm path-traversal attempts return 400 and writes from a read-only role return 403.
