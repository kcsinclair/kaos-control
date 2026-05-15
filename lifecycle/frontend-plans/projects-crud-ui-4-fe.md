---
title: Projects Page CRUD Operations — Frontend Plan
type: plan-frontend
status: done
lineage: projects-crud-ui
parent: requirements/projects-crud-ui-2.md
---

# Projects Page CRUD Operations — Frontend Plan

This plan implements the `/projects` page with full CRUD UI, project initialisation, and the "Check Directory" pre-validation flow described in F7 and the resolved questions.

Cross-references: [[projects-crud-ui]] backend plan for API contracts, test plan for verification.

---

## Milestone 1 — API module and TypeScript types

### Description

Create the API functions and type definitions needed by all subsequent milestones. This builds on the existing `api/projects.ts` (which currently only has `listProjects()`) and `types/api.ts`.

### Files to change

- `web/src/types/api.ts` — add/update `ProjectSummary` to include `owner: string` and `initialised: boolean`; add `ProjectDetail` (same shape); add `CreateProjectPayload`, `UpdateProjectPayload`, `CheckDirectoryResult`, `InitProjectResult` types
- `web/src/api/projects.ts` — add `getProject(name)`, `createProject(payload)`, `updateProject(name, payload)`, `deleteProject(name)`, `initProject(name)`, `checkDirectory(path)` using the shared `api` client

### Acceptance criteria

- All API functions are typed and call the correct endpoints (`/api/projects`, `/api/projects/{name}`, `/api/projects/{name}/init`, `/api/projects/check-directory`).
- `listProjects()` return type updated to include `owner` and `initialised`.
- No runtime regressions in `ProjectPickerView` which consumes the existing `listProjects`.

---

## Milestone 2 — Pinia project store enhancements

### Description

Extend `useProjectStore` to support the full project lifecycle: create, update, delete, init, and check-directory. The store is the single source of truth for project data in the UI.

### Files to change

- `web/src/stores/project.ts` — add actions: `create(payload)`, `update(name, payload)`, `remove(name)`, `init(name)`, `checkDirectory(path)`; after each mutation, re-fetch the project list so the UI is always current; expose `loading` and `error` state for mutations

### Acceptance criteria

- `create` calls `POST /api/projects`, then refreshes the list.
- `update` calls `PUT /api/projects/{name}`, then refreshes the list.
- `remove` calls `DELETE /api/projects/{name}`, then refreshes the list; if the deleted project was the current project, clears current selection.
- `init` calls `POST /api/projects/{name}/init`, then refreshes the list.
- `checkDirectory` calls `POST /api/projects/check-directory` and returns the result without modifying store state.
- All actions surface errors via the `error` ref.

---

## Milestone 3 — Projects list page (`/projects` route)

### Description

Create a new top-level view at `/projects` showing all registered projects in a table with action controls. This replaces the current `ProjectPickerView` as the primary entry for project management.

### Files to change

- `web/src/views/ProjectsView.vue` — new view component with a projects table showing columns: name, description, owner, path, initialised status; each row has Edit, Delete, and (conditionally) Initialise action buttons; includes a "New Project" button in the header
- `web/src/router/index.ts` — add route `{ path: '/projects', component: ProjectsView, meta: { requiresAuth: true } }`; update root redirect if appropriate
- `web/src/components/layout/AppHeader.vue` — add "Projects" link to main navigation (if not already present)

### Acceptance criteria

- `/projects` is accessible from the main navigation and requires authentication.
- All registered projects are displayed with name, description, owner, path, and initialisation status.
- Initialisation status is shown as a visual indicator (badge or icon).
- "New Project" button is prominently placed.
- Edit and Delete action controls are shown per row.
- Uninitialised projects show an "Initialise" button or indicator.
- The table is responsive at viewports ≥ 768 px (NF5).
- Clicking a project name navigates to `/p/{project}` (preserves existing workspace navigation).

---

## Milestone 4 — Create Project dialog (F2)

### Description

A modal dialog opened by the "New Project" button. Includes fields for name, path, description, and owner, with client-side validation matching server rules and the "Check Directory" feature.

### Files to change

- `web/src/components/project/CreateProjectModal.vue` — modal with form fields: `name` (required, slug-safe validation, 3–80 chars), `path` (required, absolute path), `description` (optional), `owner` (optional); includes a "Check Directory" button next to the path field; shows inline validation errors; emits `created` on success and `close` to dismiss

### Acceptance criteria

- `name` validation: non-empty, lowercase alphanumeric + hyphens only, 3–80 characters, inline error shown immediately on blur.
- `path` validation: non-empty, must start with `/`, inline error on blur.
- "Check Directory" button calls `checkDirectory(path)` and shows results: exists (green/red), writable (green/red), initialised (info badge).
- Submit calls `projectStore.create()` and shows a success toast on `201`.
- Server-returned `400` errors are mapped to the relevant field.
- Server-returned `409` (name conflict) shows an error on the name field.
- The form is disabled while submitting (loading state on button).

---

## Milestone 5 — Edit Project dialog (F5)

### Description

A modal dialog for editing a project's mutable fields (description, owner, path). Pre-populated with the current values.

### Files to change

- `web/src/components/project/EditProjectModal.vue` — modal with editable fields: `description`, `owner`, `path` (with "Check Directory" button); `name` shown but disabled; emits `updated` on success and `close` to dismiss

### Acceptance criteria

- Modal is pre-populated with the project's current values.
- `name` is displayed but not editable.
- `path` changes trigger the same validation as create (absolute, "Check Directory" available).
- Submit calls `projectStore.update(name, payload)` and shows a success toast.
- Server errors are shown as field-level or general errors.
- After successful update the projects list refreshes.

---

## Milestone 6 — Delete Project confirmation (F6)

### Description

A confirmation dialog before deleting a project. Must clearly state that on-disk files are not removed.

### Files to change

- `web/src/components/project/DeleteProjectModal.vue` — confirmation modal showing project name and an explicit message: "This will deregister the project. Files on disk will not be deleted."; has Cancel and Delete buttons; emits `confirmed` and `close`

### Acceptance criteria

- The dialog displays the project name and a clear warning that disk files are not removed.
- "Delete" button is styled as danger (`.btn-danger`).
- Clicking "Delete" calls `projectStore.remove(name)`, shows success toast, and closes the dialog.
- Clicking "Cancel" or the overlay closes without action.
- If the deleted project was the current workspace project, the user is redirected to `/projects`.

---

## Milestone 7 — Initialise Project flow (F3)

### Description

An "Initialise" action on uninitialised projects that triggers scaffolding creation and shows the result.

### Files to change

- `web/src/components/project/InitProjectModal.vue` — modal explaining what initialisation does (creates `lifecycle/config.yaml` and stage directories); shows the result after completion: list of created files/directories, and if git was already initialised, the git commands the user should run; emits `initialised` and `close`

### Acceptance criteria

- The modal explains what will be created before the user confirms.
- On confirmation, calls `projectStore.init(name)`.
- Shows the list of newly created files/directories in the result.
- If `git_commands` are returned (git already initialised), they are displayed in a copyable code block.
- If git was freshly initialised, a success message indicates the commit was made.
- After initialisation the project's `initialised` status updates in the list.

---

## Milestone 8 — Polish and integration

### Description

Wire all components together, ensure consistent styling with the existing design system, handle edge cases, and verify the complete flow.

### Files to change

- `web/src/views/ProjectsView.vue` — integrate all modals, wire up event handlers (`@created`, `@updated`, `@confirmed`, `@initialised`) to refresh the list
- `web/src/stores/project.ts` — verify toast notifications are shown via `useUiStore` for all success/error cases
- `web/src/router/index.ts` — ensure `ProjectPickerView` (if retained) and `ProjectsView` coexist or that `ProjectPickerView` redirects to the new page

### Acceptance criteria

- Complete flow works end-to-end: list → create → edit → initialise → delete.
- All mutations cause the project list to refresh immediately.
- Toast notifications (success/error) appear for every operation.
- No console errors or TypeScript type errors.
- The page is usable at 768 px viewport width.
- Existing project workspace navigation (`/p/{project}`) is unaffected.
