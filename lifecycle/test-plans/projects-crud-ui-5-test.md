---
title: Projects Page CRUD Operations — Test Plan
type: plan-test
status: in-development
lineage: projects-crud-ui
parent: requirements/projects-crud-ui-2.md
---

# Projects Page CRUD Operations — Test Plan

This plan covers integration tests (Go), unit tests (Go), and frontend component tests (TypeScript/Vitest) for the projects CRUD feature.

Cross-references: [[projects-crud-ui]] backend plan for API contracts, frontend plan for UI behaviour.

---

## Milestone 1 — Unit tests for project validation and config helpers

### Description

Test the pure validation and helper functions added to `internal/config/` — name validation, path validation, atomic writes, and initialisation detection.

### Files to change

- `internal/config/config_test.go` — add test cases for:
  - `ValidateProjectName`: valid slugs, too short, too long, uppercase, special chars, empty
  - `ValidatePath`: absolute paths, relative paths, paths with `..`, paths under `~/.kaos-control/`, symlink resolution, non-existent paths, unreadable paths
  - `IsInitialised`: returns true when `lifecycle/config.yaml` exists, false otherwise
  - `SaveProjectEntry` atomicity: verify temp file is used (check that a concurrent reader never sees a partial file)
  - `DefaultProjectConfigYAML` and `DefaultStages`: verify expected content and stage list

### Acceptance criteria

- All name validation edge cases have explicit test functions.
- Path validation tests use `t.TempDir()` with symlinks and nested directories.
- Atomic write test confirms the file is either absent or complete (no partial state).
- Tests pass with `go test ./internal/config/ -count=1`.

---

## Milestone 2 — Integration tests for `GET /api/projects` and `GET /api/projects/{project}`

### Description

Test the list and detail endpoints including the `initialised` flag, authentication requirements, and 404 handling.

### Files to change

- `tests/integration/projects_crud_test.go` — new file with build tag `//go:build integration`; test cases:
  - `TestListProjects_ReturnsAllWithInitialisedFlag`: seed two projects (one initialised, one not), verify response shape and `initialised` values
  - `TestListProjects_IncludesOwner`: verify `owner` field is present
  - `TestGetProject_Found`: verify detail response matches registry entry
  - `TestGetProject_NotFound`: verify 404 for non-existent project name
  - `TestListProjects_RequiresAuth`: verify 401 without session cookie

### Acceptance criteria

- Tests use `newTestEnv` helper pattern with seeded projects.
- Response JSON is decoded and each field is asserted individually.
- `initialised` flag correctly reflects the presence of `lifecycle/config.yaml`.
- Tests pass with `go test ./tests/integration/ -count=1 -tags=integration -run TestListProjects -run TestGetProject`.

---

## Milestone 3 — Integration tests for `POST /api/projects` (create)

### Description

Test project creation including validation, conflict detection, atomic persistence, and hot-reload (project immediately usable).

### Files to change

- `tests/integration/projects_crud_test.go` — add test cases:
  - `TestCreateProject_Success`: valid payload returns 201 with all fields; YAML file exists on disk; project accessible via `/api/p/{name}/artifacts`
  - `TestCreateProject_NameValidation`: empty, too short, too long, uppercase, special chars → 400 with field-level errors
  - `TestCreateProject_PathValidation`: relative path, non-existent, unreadable, traversal (`..`), inside config dir → 400
  - `TestCreateProject_NameConflict`: duplicate name → 409
  - `TestCreateProject_AtomicWrite`: create succeeds and YAML is well-formed
  - `TestCreateProject_NoRestartRequired`: after create, `GET /api/p/{name}/artifacts` returns 200

### Acceptance criteria

- Validation tests assert specific field names and messages in the 400 response body.
- Conflict test verifies 409 status code.
- Hot-reload test confirms the project's per-project endpoints are live immediately after creation.
- Tests clean up created YAML files via `t.Cleanup`.

---

## Milestone 4 — Integration tests for `PUT /api/projects/{project}` (update)

### Description

Test project update including field immutability, path re-validation, and in-memory state refresh.

### Files to change

- `tests/integration/projects_crud_test.go` — add test cases:
  - `TestUpdateProject_Success`: update description and owner, verify 200 and persisted YAML
  - `TestUpdateProject_PathChange`: change path to a valid new directory, verify the project re-indexes from new path
  - `TestUpdateProject_PathValidation`: invalid new path → 400
  - `TestUpdateProject_NameImmutable`: submitting a different name is ignored; original name persists
  - `TestUpdateProject_NotFound`: 404 for non-existent project

### Acceptance criteria

- Updated fields are verified both in the response and by re-reading the YAML file.
- Path change test confirms the watcher and index switch to the new path.
- Name immutability is verified by checking the response and disk file name.

---

## Milestone 5 — Integration tests for `DELETE /api/projects/{project}`

### Description

Test project deletion: registry removal, runtime unloading, and disk files left untouched.

### Files to change

- `tests/integration/projects_crud_test.go` — add test cases:
  - `TestDeleteProject_Success`: returns 200; YAML file removed; `GET /api/projects` no longer includes it; `GET /api/p/{name}/artifacts` returns 404
  - `TestDeleteProject_DiskUntouched`: project directory and its files still exist after delete
  - `TestDeleteProject_NotFound`: 404 for non-existent project
  - `TestDeleteProject_NoGoroutineLeaks`: after delete, no watcher or reaper goroutines for the project remain (test via channel or runtime check)

### Acceptance criteria

- Disk-untouched test explicitly checks that project files remain after deregistration.
- Goroutine-leak test verifies clean shutdown of project services.
- Subsequent API calls to the deleted project's scoped endpoints return 404.

---

## Milestone 6 — Integration tests for `POST /api/projects/{project}/init`

### Description

Test the initialisation endpoint including idempotency, git behaviour, and scaffolding completeness.

### Files to change

- `tests/integration/projects_crud_test.go` — add test cases:
  - `TestInitProject_CreatesScaffolding`: verify `lifecycle/config.yaml` and all default stage directories are created
  - `TestInitProject_Idempotent`: run twice; second call creates nothing new; existing files are untouched
  - `TestInitProject_GitInit`: on a non-git directory, verify `git init` is run and an initial commit is made
  - `TestInitProject_GitAlreadyInit`: on a git repo, verify no commit is made and `git_commands` are returned
  - `TestInitProject_NotFound`: 404 for non-existent project
  - `TestInitProject_ReloadsProject`: after init, the project's `initialised` flag is true in subsequent `GET /api/projects`

### Acceptance criteria

- Scaffolding test checks every expected directory and the config file.
- Idempotency test verifies file modification times are unchanged on second run.
- Git tests use `t.TempDir()` with and without pre-existing `.git/`.
- Reload test confirms the running server reflects the new state.

---

## Milestone 7 — Integration tests for `POST /api/projects/check-directory`

### Description

Test the path validation endpoint used by the frontend's "Check Directory" button.

### Files to change

- `tests/integration/projects_crud_test.go` — add test cases:
  - `TestCheckDirectory_ExistsWritableInitialised`: directory exists, is writable, has `lifecycle/config.yaml` → all three true
  - `TestCheckDirectory_ExistsWritableNotInitialised`: directory exists, writable, no config → `initialised: false`
  - `TestCheckDirectory_ExistsNotWritable`: read-only directory → `writable: false` (skip on platforms where root ignores permissions)
  - `TestCheckDirectory_NotExists`: non-existent path → `exists: false`
  - `TestCheckDirectory_InvalidPath`: relative path → 400
  - `TestCheckDirectory_TraversalAttempt`: path with `..` → 400

### Acceptance criteria

- Each response field is individually asserted.
- Read-only test uses `os.Chmod` on a temp directory (with cleanup to restore permissions).
- Traversal test confirms the path is rejected before any filesystem access.

---

## Milestone 8 — Frontend component tests

### Description

Test the Vue components for the projects CRUD UI: list rendering, form validation, modal behaviour, and store integration.

### Files to change

- `tests/web/ProjectsView.test.ts` — test cases:
  - Renders project table with correct columns from store data
  - "New Project" button opens CreateProjectModal
  - Edit button opens EditProjectModal with correct project data
  - Delete button opens DeleteProjectModal
  - Initialise button shown only for uninitialised projects
- `tests/web/CreateProjectModal.test.ts` — test cases:
  - Name validation: shows error for empty, too short, invalid chars
  - Path validation: shows error for relative path
  - "Check Directory" button calls API and displays results
  - Submit disabled while loading
  - Emits `created` on successful submission
  - Displays server-side field errors from 400 response
  - Displays conflict error from 409 response
- `tests/web/EditProjectModal.test.ts` — test cases:
  - Pre-populates fields from project data
  - Name field is disabled
  - Emits `updated` on success
- `tests/web/DeleteProjectModal.test.ts` — test cases:
  - Shows project name and "files will not be deleted" warning
  - Emits `confirmed` on delete click
  - Emits `close` on cancel
- `tests/web/InitProjectModal.test.ts` — test cases:
  - Shows explanation of what will be created
  - Displays created files list after success
  - Shows git commands in code block when returned
- `tests/web/helpers/seed_projects.ts` — factory functions: `makeProjectSummary(overrides)` returning a `ProjectSummary` with sensible defaults

### Acceptance criteria

- All tests use `@vue/test-utils` mount with Pinia test stores.
- Form validation tests trigger blur events and assert error message visibility.
- Modal emit tests verify the correct event names and payloads.
- Tests pass with `pnpm --filter @kaos-control/tests-web test`.
