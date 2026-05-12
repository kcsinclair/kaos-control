---
title: "Test Plan: Auto-Create Projects Directory on First Run"
type: plan-test
status: draft
lineage: auto-create-projects-dir
parent: lifecycle/ideas/auto-create-projects-dir.md
---

# Test Plan: Auto-Create Projects Directory on First Run

Integration tests to verify that the projects directory (and data directory) are automatically created during `config.LoadApp()`, covering fresh-install, pre-existing, and error scenarios.

## Milestone 1 — Test: directory created when missing

### Description

Write an integration test that calls `config.LoadApp()` with a config path whose parent exists but whose `projects/` subdirectory does not. Assert the directory is created with the correct permissions.

### Files to change

- `tests/integration/auto_create_projects_dir_test.go` — new test file.

### Acceptance criteria

- [ ] Test calls `config.LoadApp(tmpDir + "/config.yaml")` where `tmpDir/projects/` does not exist.
- [ ] After `LoadApp` returns successfully, `tmpDir/projects/` exists.
- [ ] The directory permission is `0o700` (on platforms that support Unix permissions).
- [ ] `tmpDir/data/` is also created with `0o700`.

## Milestone 2 — Test: idempotent when directory already exists

### Description

Write a test that pre-creates `projects/` and `data/` directories before calling `LoadApp`. Verify `LoadApp` succeeds without error and the directories remain intact with their original permissions.

### Files to change

- `tests/integration/auto_create_projects_dir_test.go` — add test case.

### Acceptance criteria

- [ ] Pre-create `tmpDir/projects/` and `tmpDir/data/` with `0o755`.
- [ ] `config.LoadApp()` succeeds.
- [ ] Directories still exist and their permissions are unchanged (MkdirAll is a no-op on existing dirs).

## Milestone 3 — Test: error on unwritable parent

### Description

Write a test that makes the parent directory read-only so that `os.MkdirAll` fails. Verify that `LoadApp` returns a meaningful error mentioning the projects directory.

### Files to change

- `tests/integration/auto_create_projects_dir_test.go` — add test case.

### Acceptance criteria

- [ ] Set parent directory to `0o444` (read-only).
- [ ] `config.LoadApp()` returns a non-nil error.
- [ ] The error message contains `"creating projects dir"` or `"creating data dir"`.
- [ ] Skip this test on platforms where permission enforcement is unreliable (e.g., `t.Skip` on Windows/CI if needed).

## Cross-links

- [[auto-create-projects-dir]] — originating idea.
- Backend plan (`auto-create-projects-dir-2-be`) implements the `os.MkdirAll` calls that these tests verify.
- Frontend plan (`auto-create-projects-dir-3-fe`) — no frontend interaction; no frontend tests needed.
