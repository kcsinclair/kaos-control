---
title: Projects Page CRUD Operations — Requirements
type: requirement
status: done
lineage: projects-crud-ui
priority: high
parent: ideas/projects-crud-ui.md
labels:
    - feature
    - frontend
    - backend
    - v1
release: KC-Release2
assignees:
    - role: product-owner
      who: agent
---

# Projects Page CRUD Operations — Requirements

## Problem

Users must currently register and manage projects by manually creating and editing YAML files under `~/.kaos-control/projects/` and restarting the server. There is no GUI path to add, inspect, modify, or remove a project. This creates friction during onboarding and day-to-day use, especially for users unfamiliar with the YAML schema or the expected directory layout.

## Goals / Non-goals

### Goals

- Provide a full CRUD interface in the web UI for managing registered projects.
- Expose REST API endpoints that back every UI operation so the GUI never writes YAML directly.
- When registering a new project whose target directory lacks kaos-control scaffolding (`lifecycle/config.yaml`, stage subdirectories), offer to initialise it automatically.
- Make project management possible without a server restart — changes take effect immediately.

### Non-goals

- Editing per-project `lifecycle/config.yaml` content (stages, agents, roles, transitions, etc.) through this UI. That is a separate feature; this covers only the project registry entry (`name`, `path`, `description`, `owner`).
- Git repository initialisation. If the target path is not already a git repo the user must initialise it themselves.
- Remote/cloud project paths. Only local filesystem paths are in scope for v1.
- Multi-user permission checks on project operations beyond the existing session auth.

## Detailed Requirements

### Functional

#### F1 — List Projects

- `GET /api/projects` already returns all registered projects. Extend the response to include an `initialised` boolean indicating whether `lifecycle/config.yaml` exists at the project path.
- The UI shall display a table/card list showing: project name, description, owner, path, and initialisation status.
- Each row shall have Edit and Delete action controls.

#### F2 — Create Project

- `POST /api/projects` accepts a JSON body with fields: `name` (required, unique), `path` (required, absolute filesystem path), `description` (optional), `owner` (optional).
- **Validation rules:**
  - `name`: non-empty, unique among registered projects, slug-safe characters only (lowercase alphanumeric + hyphens, 3–80 chars).
  - `path`: must be an absolute path; the directory must exist on disk and be readable by the server process.
- On success the endpoint writes `~/.kaos-control/projects/<name>.yaml` and registers the project with the running server (no restart required).
- The endpoint returns the created project entry including the `initialised` flag.

#### F3 — Initialise Project Directory

- `POST /api/projects/{project}/init` creates the kaos-control scaffolding inside an already-registered project's path:
  1. `lifecycle/config.yaml` with sensible defaults (default stages, empty roles/agents).
  2. One subdirectory per default stage (`ideas/`, `requirements/`, `backend-plans/`, `frontend-plans/`, `dev-plans/`, `test-plans/`, `tests/`, `prototypes/`, `releases/`, `sprints/`, `defects/`).
- The endpoint is idempotent: directories and files that already exist are left untouched.
- Returns the list of files/directories created.
- The UI shall prompt for initialisation when a newly created (or existing) project is not yet initialised, via a clearly labelled button or inline prompt.

#### F4 — Read Project

- `GET /api/projects/{project}` returns the full project registry entry plus the `initialised` flag.

#### F5 — Update Project

- `PUT /api/projects/{project}` accepts a JSON body with updatable fields: `description`, `owner`, `path`.
  - `name` is immutable after creation (it is the registry key).
  - Changing `path` re-validates the new path (must exist, must be readable).
- On success the endpoint rewrites the project's YAML file and updates the in-memory state without restart.

#### F6 — Delete Project

- `DELETE /api/projects/{project}` removes the registry entry (`~/.kaos-control/projects/<name>.yaml`) and unloads the project from the running server.
- **Does not** delete the project directory or any files on disk. This is a deregistration, not a destructive delete.
- The UI must present a confirmation dialog before executing the delete, clearly stating that files on disk will not be removed.

#### F7 — Projects Page (Frontend)

- A new top-level route `/projects` accessible from the main navigation.
- Contains:
  - A table or card grid listing all projects (F1).
  - A "New Project" button opening a create form/dialog (F2).
  - Per-project Edit (F5) and Delete (F6) actions.
  - An "Initialise" button/indicator for uninitialised projects (F3).
- Form validation must mirror server-side rules (F2 validation) and show inline errors.
- After any mutation the list must refresh to reflect the current state.

### Non-functional

#### NF1 — No Restart Required

All CRUD operations must take effect immediately on the running server. The project list, routing, and watcher infrastructure must update without a process restart.

#### NF2 — Atomic File Writes

Registry YAML files must be written atomically (write-to-temp then rename) to avoid corruption from concurrent reads or crashes mid-write.

#### NF3 — Error Handling

- API endpoints must return appropriate HTTP status codes: `201` (created), `200` (success), `400` (validation error with field-level detail), `404` (project not found), `409` (name conflict).
- The frontend must display server-returned error messages clearly.

#### NF4 — Path Safety

The `path` field must be validated to prevent path-traversal attacks. Symlinks should be resolved and the resolved path checked. The server must not follow paths outside the filesystem (e.g., reject paths containing `..` after resolution or paths pointing into `~/.kaos-control/` itself).

#### NF5 — Responsive UI

The projects page must be usable on viewports ≥ 768 px wide, consistent with the existing UI's responsive behaviour.

## Acceptance Criteria

- [ ] `GET /api/projects` returns all registered projects with `name`, `path`, `description`, `owner`, and `initialised` fields.
- [ ] `POST /api/projects` creates a new project registry entry; returns `201`; entry is immediately available without restart.
- [ ] `POST /api/projects` returns `400` with field-level errors for invalid `name` or `path`.
- [ ] `POST /api/projects` returns `409` when `name` already exists.
- [ ] `GET /api/projects/{project}` returns the project detail including `initialised`.
- [ ] `PUT /api/projects/{project}` updates mutable fields and persists changes.
- [ ] `DELETE /api/projects/{project}` removes the registry entry without deleting on-disk files.
- [ ] `POST /api/projects/{project}/init` creates `lifecycle/config.yaml` and all default stage directories; is idempotent.
- [ ] The `/projects` page lists all projects and reflects mutations immediately.
- [ ] The create form validates `name` uniqueness and `path` existence before submission.
- [ ] The delete action requires explicit user confirmation.
- [ ] Uninitialised projects show an "Initialise" control that triggers F3.
- [ ] Registry YAML writes are atomic (temp + rename).
- [ ] Path validation rejects traversal attempts and non-absolute paths.
- [ ] [[projects-crud-ui]] lineage is preserved across all derived artifacts.

## Resolved Questions

1. Should the "Initialise" action also create an initial git commit for the scaffolding, or leave that to the user?

> If git has not been initialised, it should git init and commit.  If git already setup, do not commit but suggest the commands needed to add files and make the commit.

2. Should there be a "Test Connection" or path-validation button in the create/edit form that checks the path before submission, or is submit-time validation sufficient?

> Yes, call it Check Directory, which will check for existence and that kaos-control can write to the directory for the process running kaos-control.

3. Are there additional project registry fields beyond `name`, `path`, `description`, and `owner` that should be exposed in v1 (e.g., a project icon or colour for the nav)?

> No, standard fields for now.  I will raise an enhancement to get custom icons and colours for projects, great idea.
