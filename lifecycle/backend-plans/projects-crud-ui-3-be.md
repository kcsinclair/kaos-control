---
title: Projects Page CRUD Operations — Backend Plan
type: plan-backend
status: draft
lineage: projects-crud-ui
parent: requirements/projects-crud-ui-2.md
---

# Projects Page CRUD Operations — Backend Plan

This plan implements the REST API endpoints (F1–F6), project initialisation (F3), and supporting infrastructure for CRUD management of registered projects without server restart.

Cross-references: [[projects-crud-ui]] frontend plan for UI integration, test plan for verification.

---

## Milestone 1 — Extend `GET /api/projects` with `initialised` flag (F1, F4)

### Description

Add an `initialised` boolean to each project in the list response by checking for the existence of `lifecycle/config.yaml` at each project's path. Add a `GET /api/projects/{project}` detail endpoint returning the same shape for a single project, plus `owner`.

### Files to change

- `internal/http/server.go` — register new route `GET /api/projects/{project}`
- `internal/http/projects.go` — update `projectSummary` struct to include `owner` and `initialised` fields; add `handleGetProject` handler; modify `handleListProjects` to populate the new fields
- `internal/config/config.go` — add `IsInitialised(projectPath string) bool` helper that checks for `lifecycle/config.yaml`

### Acceptance criteria

- `GET /api/projects` returns each project with `name`, `path`, `description`, `owner`, and `initialised` (boolean).
- `GET /api/projects/{project}` returns `200` with the full project entry including `initialised`, or `404` if the project is not registered.
- `initialised` is `true` only when `<project-path>/lifecycle/config.yaml` exists on disk.

---

## Milestone 2 — `POST /api/projects` — Create project (F2)

### Description

Accept a JSON body with `name`, `path`, `description`, `owner`. Validate inputs, write the registry YAML atomically, and register the project with the running server without restart.

### Files to change

- `internal/http/projects.go` — add `handleCreateProject` with JSON binding, validation, and response
- `internal/http/server.go` — register `POST /api/projects`
- `internal/config/config.go` — add `ValidateProjectName(name string) error` (slug-safe: lowercase alphanumeric + hyphens, 3–80 chars) and ensure `SaveProjectEntry` writes atomically (write-to-temp, rename)
- `internal/http/server.go` — add a method to open and register a new project in the running server's project map (call `project.Open`, start watcher/reaper goroutines, add to map)

### Acceptance criteria

- `POST /api/projects` with valid body returns `201` with the created project including `initialised`.
- Returns `400` with field-level errors for: empty `name`, name with invalid characters, name shorter than 3 or longer than 80 chars, empty `path`, relative `path`, non-existent `path`, unreadable `path`.
- Returns `409` when `name` already exists among registered projects.
- The project's YAML file is written atomically to `~/.kaos-control/projects/<name>.yaml`.
- The new project is immediately usable via `/api/p/{project}/...` endpoints without server restart.
- NF4: `path` is validated — symlinks resolved, `..` rejected after resolution, paths inside `~/.kaos-control/` rejected.

---

## Milestone 3 — `PUT /api/projects/{project}` — Update project (F5)

### Description

Accept a JSON body with updatable fields (`description`, `owner`, `path`). `name` is immutable. Re-validate `path` if changed. Rewrite the YAML file atomically and update in-memory state.

### Files to change

- `internal/http/projects.go` — add `handleUpdateProject`
- `internal/http/server.go` — register `PUT /api/projects/{project}`
- `internal/config/config.go` — ensure `SaveProjectEntry` handles overwrite correctly with atomic write

### Acceptance criteria

- `PUT /api/projects/{project}` returns `200` with the updated project entry.
- `name` cannot be changed; if submitted it is ignored.
- `path` changes are validated the same as create (exists, readable, no traversal).
- Returns `404` if the project is not registered.
- Returns `400` for invalid `path`.
- YAML file is rewritten atomically.
- If `path` changed, the project's watcher, index, and git state are reinitialised for the new path without restart.

---

## Milestone 4 — `DELETE /api/projects/{project}` — Delete project (F6)

### Description

Remove the registry entry and unload the project from the running server. Does not delete any on-disk project files.

### Files to change

- `internal/http/projects.go` — add `handleDeleteProject`
- `internal/http/server.go` — register `DELETE /api/projects/{project}`; add a method to gracefully close and remove a project from the server's project map (call `project.Close`, stop watcher/reaper, remove from map)
- `internal/config/config.go` — ensure `DeleteProjectEntry` removes the YAML file

### Acceptance criteria

- `DELETE /api/projects/{project}` returns `200` on success.
- Returns `404` if the project is not registered.
- The project's YAML file is removed from `~/.kaos-control/projects/`.
- The project is immediately unloaded — subsequent calls to `/api/p/{project}/...` return `404`.
- No files or directories within the project's disk path are deleted.

---

## Milestone 5 — `POST /api/projects/{project}/init` — Initialise project directory (F3)

### Description

Create kaos-control scaffolding inside a registered project's path: `lifecycle/config.yaml` with sensible defaults, one subdirectory per default stage, and optionally initialise git.

### Files to change

- `internal/http/projects.go` — add `handleInitProject`
- `internal/http/server.go` — register `POST /api/projects/{project}/init`
- `internal/config/config.go` — add `DefaultProjectConfigYAML() string` returning the default `lifecycle/config.yaml` content; add `DefaultStages() []string` returning the list of default stage directories

### Acceptance criteria

- `POST /api/projects/{project}/init` creates `lifecycle/config.yaml` and all default stage subdirectories (`ideas/`, `requirements/`, `backend-plans/`, `frontend-plans/`, `dev-plans/`, `test-plans/`, `tests/`, `prototypes/`, `releases/`, `sprints/`, `defects/`).
- The endpoint is idempotent: existing files and directories are left untouched.
- Returns `200` with a list of files/directories that were newly created.
- Returns `404` if the project is not registered.
- If git is not initialised at the project path, run `git init` and commit the scaffolding.
- If git is already initialised, do not commit; return a `git_commands` field with the commands needed to add and commit the new files.
- After init, re-open the project to load the new config and start watchers; update the `initialised` flag.

---

## Milestone 6 — `POST /api/projects/{project}/check-directory` — Path validation endpoint

### Description

Provide a "Check Directory" endpoint that validates a filesystem path before form submission: checks existence, writability, and whether it is already initialised.

### Files to change

- `internal/http/projects.go` — add `handleCheckDirectory`
- `internal/http/server.go` — register `POST /api/projects/check-directory`

### Acceptance criteria

- Accepts `{"path": "..."}` and returns `200` with `{"exists": bool, "writable": bool, "initialised": bool}`.
- Validates path safety (absolute, no traversal, not inside `~/.kaos-control/`).
- Returns `400` for invalid or relative paths.
- Does not require the project to be registered; this is used during creation.

---

## Milestone 7 — Atomic writes and path safety hardening (NF2, NF4)

### Description

Ensure all registry YAML writes use write-to-temp-then-rename for atomicity. Harden path validation to resolve symlinks and reject traversal attempts.

### Files to change

- `internal/config/config.go` — refactor `SaveProjectEntry` to use `os.CreateTemp` + `os.Rename` pattern; add `ValidatePath(path string) error` that resolves symlinks via `filepath.EvalSymlinks`, rejects relative paths, paths containing `..` after resolution, and paths under the kaos-control config directory
- `internal/http/projects.go` — call `ValidatePath` in create, update, and check-directory handlers

### Acceptance criteria

- Registry YAML writes are atomic: a crash mid-write never leaves a corrupt file.
- `ValidatePath` rejects: relative paths, paths with `..` after symlink resolution, paths under `~/.kaos-control/`, non-existent paths.
- Symlinks are resolved before checking; the resolved path is stored.
- All create/update/check-directory handlers use the shared validation.

---

## Milestone 8 — Hot-reload infrastructure (NF1)

### Description

Ensure the server's project map supports concurrent add/remove operations and that adding or removing a project at runtime correctly starts or stops all associated goroutines (watcher, lock reaper, session reaper, scheduler).

### Files to change

- `internal/http/server.go` — add `RegisterProject(entry)` and `UnregisterProject(name)` methods with mutex protection; manage goroutine lifecycle
- `internal/project/project.go` — ensure `Close()` cleanly stops all goroutines and closes the SQLite index

### Acceptance criteria

- Adding a project at runtime starts its watcher, lock reaper, and session reaper.
- Removing a project at runtime stops all its goroutines and closes its index.
- Concurrent requests to different projects are not blocked by project add/remove operations.
- No goroutine leaks on project removal (verified by test).
