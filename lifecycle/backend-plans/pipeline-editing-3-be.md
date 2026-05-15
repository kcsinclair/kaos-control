---
title: "Backend Plan — Make Pipelines Editable"
type: plan-backend
status: draft
lineage: pipeline-editing
parent: lifecycle/requirements/pipeline-editing-2.md
created: "2026-05-15T00:00:00+10:00"
labels:
    - backend
    - devops
release: KC-Release1
assignees:
    - role: backend-developer
      who: agent
---

# Backend Plan — Make Pipelines Editable

Implements the server-side changes required by the "Make Pipelines Editable" requirement ([[pipeline-editing]]).

## Milestone 1 — GET single-pipeline endpoint

Add `GET /api/p/{project}/devops/pipelines/{slug}` that returns the raw YAML content of a pipeline file so the frontend can populate the editor.

### Files to change

- `internal/http/devops.go` — add `handleGetPipeline(w, r)` handler.
- `internal/http/server.go` — register route `r.Get("/api/p/{project}/devops/pipelines/{slug}", s.handleGetPipeline)` in the devops route group (around line 318–322).

### Implementation details

1. Extract `slug` from `chi.URLParam(r, "slug")`.
2. Auth check: `requireRole(w, r, p, RolesDevopsOrAdmin...)`.
3. Build file path: `filepath.Join(devopsDir(p.Entry.Path), slug+".yaml")`.
4. Validate slug matches the existing regex (`validSlugRe`).
5. Read file with `os.ReadFile`; return `404` if `os.IsNotExist`.
6. Return `200 OK` with `Content-Type: text/yaml` and the raw file bytes as the response body.

### Acceptance criteria

- [ ] `GET .../pipelines/{slug}` returns `200` with the verbatim YAML content for an existing pipeline.
- [ ] Returns `404 Not Found` when the slug does not exist on disk.
- [ ] Returns `401`/`403` for unauthenticated or insufficiently-privileged users.
- [ ] Invalid slugs (path traversal, invalid characters) return `400 Bad Request`.

---

## Milestone 2 — PUT update-pipeline endpoint

Add `PUT /api/p/{project}/devops/pipelines/{slug}` that validates and atomically persists an updated YAML definition.

### Files to change

- `internal/http/devops.go` — add `handleUpdatePipeline(w, r)` handler.
- `internal/http/server.go` — register route `r.Put("/api/p/{project}/devops/pipelines/{slug}", s.handleUpdatePipeline)`.

### Implementation details

1. Extract `slug` from URL; validate against `validSlugRe`.
2. Auth check: `requireRole(w, r, p, RolesDevopsOrAdmin...)`.
3. Decode JSON request body `{ "definition": "<yaml string>" }`.
4. Check pipeline file exists at `devopsDir(p.Entry.Path)/slug.yaml`; return `404` if not.
5. **Active-run guard** (NF3 concurrency safety): call `p.DevopsRunner.IsRunning(slug)` — return `409 Conflict` with descriptive message if `true`. Re-check after validation but immediately before the write to close the TOCTOU window.
6. Validate YAML via `devops.ValidateDefinition([]byte(definition))` — return `400 Bad Request` with the validation error on failure.
7. **Atomic write** (NF4 file integrity):
   - Write to a temp file in the same directory: `os.CreateTemp(dir, slug+".yaml.tmp*")`.
   - Write the definition bytes, then `f.Sync()`, then `f.Close()`.
   - `os.Rename(tmpPath, destPath)` — atomic on POSIX.
   - On error, `os.Remove(tmpPath)` in a deferred cleanup.
8. Re-read the written file with `devops.LoadPipeline(destPath)` to build the response payload.
9. Return `200 OK` with JSON `{ "slug", "name", "type", "step_count" }`.

### Acceptance criteria

- [ ] `PUT .../pipelines/{slug}` with valid YAML returns `200` and the updated summary.
- [ ] The file on disk matches the submitted definition byte-for-byte.
- [ ] Returns `409 Conflict` when any pipeline has an active run (per resolved question #2 — global lock).
- [ ] Returns `400 Bad Request` with a descriptive message for invalid YAML, missing required fields, or invalid timeout values.
- [ ] Returns `404 Not Found` when the slug does not exist.
- [ ] Returns `401`/`403` for unauthenticated or insufficiently-privileged users.
- [ ] A partial write (process killed mid-write) does not corrupt the existing file — temp-file + rename pattern confirmed.

---

## Milestone 3 — Global run guard for edits

The resolved questions in the requirement clarify that editing should be blocked while **any** pipeline is running, not just the target pipeline. Adjust the active-run check accordingly.

### Files to change

- `internal/devops/runner.go` — add `AnyRunning() bool` method that returns `true` if `len(r.bySlug) > 0` under lock.
- `internal/http/devops.go` — in `handleUpdatePipeline`, call `p.DevopsRunner.AnyRunning()` instead of `p.DevopsRunner.IsRunning(slug)`.

### Implementation details

1. In `runner.go`, add:
   ```go
   func (r *Runner) AnyRunning() bool {
       r.mu.Lock()
       defer r.mu.Unlock()
       return len(r.bySlug) > 0
   }
   ```
2. Replace the `IsRunning(slug)` call in the PUT handler with `AnyRunning()`.
3. Also call `AnyRunning()` in the second check immediately before the atomic write.

### Acceptance criteria

- [ ] `PUT .../pipelines/{slug}` returns `409` when a *different* pipeline has an active run.
- [ ] `PUT .../pipelines/{slug}` succeeds when no pipeline is running.
- [ ] The `AnyRunning()` method is safe for concurrent access (mutex-protected).

---

## Milestone 4 — WebSocket broadcast after edit

Broadcast a notification so connected clients (including the editing user's other tabs) can react to the pipeline change.

### Files to change

- `internal/http/devops.go` — after successful write in `handleUpdatePipeline`, broadcast a `pipeline.updated` event via `p.Hub`.

### Implementation details

1. Define a `PipelineUpdatedPayload` struct (or reuse an inline map) with `slug`, `name`, `type`, `step_count`.
2. After the successful rename, broadcast:
   ```go
   p.Hub.Broadcast("pipeline.updated", payload)
   ```
3. The [[pipeline-editing-4-fe]] frontend plan will subscribe to this event to refresh the card.

### Acceptance criteria

- [ ] A `pipeline.updated` WebSocket event is emitted on successful PUT.
- [ ] The payload contains at minimum the `slug` and updated `name`, `type`, `step_count`.
- [ ] No event is emitted on validation failure, 409, or 404.
