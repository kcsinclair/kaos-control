---
title: "Tests: Projects CRUD UI"
type: test
status: in-qa
lineage: projects-crud-ui
parent: lifecycle/test-plans/projects-crud-ui-5-test.md
---

# Tests: Projects CRUD UI

This artifact documents the test suites written for the Projects CRUD UI feature.

## Scenarios covered

### Milestone 1 — Config unit tests (`internal/config/config_test.go`)

Added to the existing config package test file:

- `TestValidateProjectName` — valid slugs pass; empty, too-short, too-long, uppercase, and special-character names are rejected with descriptive errors.
- `TestValidatePathFormat` — relative paths and dotdot traversals are rejected; absolute paths pass; paths inside the kaos-control config directory are rejected.
- `TestValidatePath` — non-existent paths rejected; symlinks resolved to canonical path; symlinks that point into the config directory are rejected.
- `TestIsInitialised` — returns `true` when `lifecycle/config.yaml` exists, `false` otherwise (missing dir, or dir present without config file).
- `TestSaveProjectEntry_Atomic` — write succeeds, YAML is well-formed on reload, no `.tmp` files left behind.
- `TestDefaultStages` — all spec-mandated stage directories (`ideas`, `requirements`, `backend-plans`, …) are present.
- `TestDefaultProjectConfigYAML` — output is non-empty, loads as a valid `Project`, contains every default stage dir.

Run with:
```sh
go test ./internal/config/ -count=1 -run "TestValidateProjectName|TestValidatePathFormat|TestValidatePath|TestIsInitialised|TestSaveProjectEntry_Atomic|TestDefaultStages|TestDefaultProjectConfigYAML"
```

### Milestones 2–7 — API integration tests (`tests/integration/projects_crud_test.go`)

Build tag: `//go:build integration`

#### Milestone 2 — GET /api/projects and GET /api/projects/{project}

- `TestListProjects_ReturnsAllWithInitialisedFlag` — two seeded projects (one initialised, one not); response includes both with correct `initialised` values.
- `TestListProjects_IncludesOwner` — `owner` field present for every project in the list.
- `TestGetProject_Found` — detail response matches registry entry including `path` and `initialised` fields.
- `TestGetProject_NotFound` — 404 for unknown project name.
- `TestListProjects_RequiresAuth` — 401 without session cookie.

#### Milestone 3 — POST /api/projects (create)

- `TestCreateProject_Success` — 201 response with all fields; YAML file created on disk.
- `TestCreateProject_NameValidation` — empty, too-short, uppercase, and invalid-character names return 400 with `invalid_name` code.
- `TestCreateProject_PathValidation` — empty, relative, dotdot, and non-existent paths return 400.
- `TestCreateProject_NameConflict` — duplicate name returns 409 with `conflict` code.
- `TestCreateProject_AtomicWrite` — YAML file is well-formed after create (loadable via `LoadProjectRegistry`).
- `TestCreateProject_NoRestartRequired` — `GET /api/p/{name}/artifacts` returns 200 immediately after create.

#### Milestone 4 — PUT /api/projects/{project} (update)

- `TestUpdateProject_Success` — description and owner updated in response and persisted to YAML.
- `TestUpdateProject_PathChange` — new valid path reflected in response and YAML file.
- `TestUpdateProject_PathValidation` — non-existent path returns 400.
- `TestUpdateProject_NameImmutable` — project name unchanged in response and YAML filename after update.
- `TestUpdateProject_NotFound` — 404 for unknown project.

#### Milestone 5 — DELETE /api/projects/{project}

- `TestDeleteProject_Success` — 200 with `{"ok":true}`; YAML removed; project absent from list; scoped `/api/p/{name}/artifacts` returns 404.
- `TestDeleteProject_DiskUntouched` — project directory and `lifecycle/config.yaml` still exist after deregistration.
- `TestDeleteProject_NotFound` — 404 for unknown project.
- `TestDeleteProject_NoGoroutineLeaks` — goroutine count after delete is within 10 of the pre-create baseline, confirming watcher/reaper goroutines are stopped.

#### Milestone 6 — POST /api/projects/{project}/init

- `TestInitProject_CreatesScaffolding` — `lifecycle/config.yaml` and all default stage directories created; `created` list non-empty.
- `TestInitProject_Idempotent` — second call returns empty `created` list and does not modify `config.yaml` mtime.
- `TestInitProject_GitInit` — `git_initialised: true` and `.git/` created for a non-git directory.
- `TestInitProject_GitAlreadyInit` — `git_initialised: false` for an already-git directory.
- `TestInitProject_NotFound` — 404 for unknown project.
- `TestInitProject_ReloadsProject` — `GET /api/projects/{project}` reports `initialised: true` after init.

#### Milestone 7 — POST /api/projects/check-directory

- `TestCheckDirectory_ExistsWritableInitialised` — all three flags `true` for writable dir with `lifecycle/config.yaml`.
- `TestCheckDirectory_ExistsWritableNotInitialised` — `initialised: false` for writable dir without config.
- `TestCheckDirectory_ExistsNotWritable` — `writable: false` for read-only directory (skipped as root or on Windows).
- `TestCheckDirectory_NotExists` — `exists: false` for non-existent path.
- `TestCheckDirectory_InvalidPath` — 400 with `invalid_path` code for relative path.
- `TestCheckDirectory_TraversalAttempt` — 400 with `invalid_path` code for `../` relative path.

Run with:
```sh
go test ./tests/integration/ -count=1 -tags=integration \
  -run "TestListProjects|TestGetProject|TestCreateProject|TestUpdateProject|TestDeleteProject|TestInitProject|TestCheckDirectory"
```

### Milestone 8 — Frontend component tests (PENDING)

The frontend Vue components described in Milestone 8 of the test plan
(`ProjectsView.vue`, `CreateProjectModal.vue`, `EditProjectModal.vue`,
`DeleteProjectModal.vue`, `InitProjectModal.vue`) have not been implemented
in `web/src/` yet.

The data factory helper `tests/web/helpers/seed_projects.ts` has been written
and exports `makeProjectSummary`, `makeProjectList`, `makeInitialisedProject`,
and `makeUninitialisedProject` for use once the components exist.

The component test files listed in Milestone 8 of the test plan cannot be
authored until the components are built by the frontend-developer. Once
the components land, the test-developer should implement:

- `tests/web/ProjectsView.test.ts`
- `tests/web/CreateProjectModal.test.ts`
- `tests/web/EditProjectModal.test.ts`
- `tests/web/DeleteProjectModal.test.ts`
- `tests/web/InitProjectModal.test.ts`

## Test files

- `internal/config/config_test.go` — Milestone 1 unit tests (appended to existing file)
- `tests/integration/projects_crud_test.go` — Milestones 2–7 integration tests
- `tests/web/helpers/seed_projects.ts` — `ProjectSummary` factory helper for frontend tests
