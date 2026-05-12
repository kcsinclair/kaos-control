---
title: 'Tests: Auto-Create Projects Directory on First Run'
type: test
status: draft
lineage: auto-create-projects-dir
parent: lifecycle/test-plans/auto-create-projects-dir-4-test.md
---

# Tests: Auto-Create Projects Directory on First Run

Integration tests covering the `config.LoadApp` directory auto-creation behaviour implemented in the `auto-create-projects-dir` feature.

## Test file

`tests/integration/auto_create_projects_dir_test.go`

Build tag: `integration`

## Scenarios covered

### Milestone 1 — Directory created when missing (`TestLoadApp_CreatesMissingProjectsDir`)

Calls `config.LoadApp` against a config path whose parent exists but whose `projects/` and `data/` subdirectories do not. Asserts:

- `LoadApp` returns no error.
- Both `projects/` and `data/` are created under the temp directory.
- Each directory has Unix permissions `0o700`.

Skipped on Windows where Unix permission enforcement is not applicable.

### Milestone 2 — Idempotent when directories already exist (`TestLoadApp_IdempotentWhenDirsExist`)

Pre-creates `projects/` and `data/` with `0o755` before calling `LoadApp`. Asserts:

- `LoadApp` returns no error.
- Both directories still exist after the call.
- Permissions remain `0o755` (MkdirAll is a no-op on existing directories) — checked on non-Windows platforms.

### Milestone 3 — Error on unwritable target directory (`TestLoadApp_ErrorOnUnwritableParent`)

Writes a config file that points `projects_dir` and `data_dir` at paths inside a read-only directory (`0o444`). Asserts:

- `LoadApp` returns a non-nil error.
- The error message contains `"creating projects dir"` or `"creating data dir"`.

Skipped on Windows and when the process runs as root (both cases make permission enforcement unreliable).
