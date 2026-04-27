---
title: "Backend plan: configurable ignore patterns for lifecycle indexer"
type: plan-backend
status: in-development
lineage: ignore-readme-files-in-lifecycle-dir
parent: lifecycle/defects/ignore-readme-files-in-lifecycle-dir.md
---

# Backend plan: configurable ignore patterns for lifecycle indexer

This plan adds a configurable `ignore` list to `lifecycle/config.yaml` and threads it through the startup scan, live watcher, and `IndexFile` path so that matching files (e.g. `README.md`) are silently skipped.

## Milestone 1 — Add `Ignore` field to project config and parse it

**Description:** Extend `config.Project` with an `Ignore []string` field that holds glob patterns (e.g. `README.md`, `*.draft.md`). Parse and validate the field in `LoadProject`. Supply a sensible default (`["README.md"]`) when the field is absent.

**Files to change:**
- `internal/config/config.go` — add `Ignore []string \`yaml:"ignore"\`` to `Project`; set the default in `defaultProject()`; optionally validate patterns in `validateProject()`.

**Acceptance criteria:**
- [ ] `config.Project` exposes an `Ignore []string` field.
- [ ] When `lifecycle/config.yaml` omits `ignore`, the default list is `["README.md"]`.
- [ ] When `lifecycle/config.yaml` includes `ignore: ["README.md", "*.draft.md"]`, both patterns are loaded.
- [ ] Invalid glob syntax in `ignore` produces a clear validation error on startup.

---

## Milestone 2 — Add a shared `ShouldIgnore` helper

**Description:** Create a small helper function (e.g. in `internal/config/` or `internal/artifact/`) that takes a file path and the ignore-pattern list and returns `true` when the file's base name matches any pattern via `filepath.Match`.

**Files to change:**
- `internal/config/config.go` — add `ShouldIgnore(path string, patterns []string) bool`.

**Acceptance criteria:**
- [ ] `ShouldIgnore("lifecycle/ideas/README.md", []string{"README.md"})` returns `true`.
- [ ] `ShouldIgnore("lifecycle/ideas/login.md", []string{"README.md"})` returns `false`.
- [ ] Glob patterns like `*.draft.md` are supported via `filepath.Match`.

---

## Milestone 3 — Thread ignore patterns through `Index.Scan`

**Description:** Pass the ignore list into `Scan` (or make it available on the `Index` struct) so the `filepath.WalkDir` callback calls `ShouldIgnore` before `IndexFile`. Ignored files are skipped silently (no warning log).

**Files to change:**
- `internal/index/index.go` — update `Scan` signature or `Index` struct to accept/store ignore patterns; add the `ShouldIgnore` check in the walk callback.

**Acceptance criteria:**
- [ ] A `README.md` inside any lifecycle stage directory is not inserted into the SQLite index during a full scan.
- [ ] Non-matching `.md` files continue to be indexed normally.
- [ ] No warning or error is logged for ignored files.

---

## Milestone 4 — Thread ignore patterns through `Watcher.shouldProcess`

**Description:** Pass the ignore list to the `Watcher` so that `shouldProcess` rejects files matching any ignore pattern before debounce/index. This prevents live fsnotify events for ignored files from triggering `IndexFile`.

**Files to change:**
- `internal/watcher/watcher.go` — add an `ignore []string` field to `Watcher`; accept it in `New`; extend `shouldProcess` to call `ShouldIgnore`.

**Acceptance criteria:**
- [ ] Creating or modifying a `README.md` inside `lifecycle/` does not trigger `handleChange`.
- [ ] Other `.md` file events continue to be processed normally.

---

## Milestone 5 — Thread ignore patterns through `IndexFile` guard

**Description:** Add a final safety-net check inside `IndexFile` itself so that even a direct API call to index an ignored file is rejected. This is defence-in-depth; `Scan` and the watcher already filter, but `IndexFile` is also called from HTTP handlers.

**Files to change:**
- `internal/index/index.go` — store ignore patterns on the `Index` struct (set during `Open`); check in `IndexFile` before parsing.

**Acceptance criteria:**
- [ ] Calling `idx.IndexFile("/path/to/lifecycle/ideas/README.md")` returns an error and does not insert a row.
- [ ] The guard works regardless of whether the caller is `Scan`, the watcher, or an HTTP handler.

---

## Milestone 6 — Wire ignore config through `project.New` / startup

**Description:** Ensure the loaded `config.Project.Ignore` list is passed to `index.Open` (or set on the `Index` after open) and to `watcher.New` during project initialisation.

**Files to change:**
- `internal/project/` (project container) — pass `cfg.Ignore` when constructing the index and watcher.
- `cmd/kaos-control/` — no change expected unless the project container API changes.

**Acceptance criteria:**
- [ ] The server starts with `ignore: ["README.md"]` in config and a `lifecycle/ideas/README.md` on disk; the README is not present in the artifact list API response.
- [ ] `go build ./...` and `go vet ./...` pass.

---

## Cross-references

- [[ignore-readme-files-in-lifecycle-dir]] frontend plan (index 3): no frontend code changes are expected for this defect, but if a future UI is added to manage ignore patterns, it would consume the config endpoint surfaced here.
- [[ignore-readme-files-in-lifecycle-dir]] test plan (index 4): integration tests covering scan, watcher, and API behaviour with ignored files.
