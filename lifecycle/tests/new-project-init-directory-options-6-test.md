---
title: "New Project Init: Existing or New Directory — Test Coverage"
type: test
status: in-qa
lineage: new-project-init-directory-options
parent: lifecycle/test-plans/new-project-init-directory-options-5-test.md
---

# New Project Init: Existing or New Directory — Test Coverage

Test suite for the mode-aware New Project onboarding flow (`POST /api/projects`,
`POST /api/projects/check-directory`). Covers every FR/NFR in
[lifecycle/requirements/new-project-init-directory-options-2.md](../requirements/new-project-init-directory-options-2.md).

## Milestone 1 — Unit tests (pure helpers)

**File:** `internal/config/config_test.go`

Already implemented alongside the backend plan; exercised by `go test ./internal/config/...`.

| Test | Requirement |
|---|---|
| `TestNormalizePath` | FR9 — whitespace trim + `~`/`~/` expansion |
| `TestValidateDirName` | FR4 — distinct errors for empty, `a/b`, `a\b`, `..`, `a/../b` |
| `TestResolveNewTarget` | FR3, FR4, NFR1 — join via `sandbox.Resolve`, reject traversal/absolute names |

## Milestones 2–6 — Integration tests

**File:** `tests/integration/project_init_directory_options_test.go`

Run with: `go test ./tests/integration/ -tags integration -run TestOnboard\|TestCheckDirectory_ExistingMode_Normalises`

Drives the real router via the `crudTestEnv` harness (`newCRUDTestEnv`, shared
with `projects_crud_test.go`) so `POST /api/projects` and
`POST /api/projects/check-directory` run against the actual `handleCreateProject`
/ `handleCheckDirectory` handlers, with `t.TempDir()` for all filesystem work.

| Test function | Requirement(s) | Scenario |
|---|---|---|
| `TestOnboard_ExistingMode_NonDestructive` | FR5, FR7, NFR3 | Pre-populated dir (custom `CLAUDE.md`, nested `lifecycle/ideas/keep.md`, unrelated `data.txt`) onboards with every pre-existing file byte-identical, only missing scaffold files created, project registered, and file set is a superset of a CLI-`init` reference scaffold |
| `TestOnboard_ExistingMode_PathMissing` | FR4, FR8 | Non-existent path → `path_missing`, nothing written |
| `TestOnboard_ExistingMode_NotADirectory` | FR4, FR8 | Path is a file → `not_a_directory` |
| `TestOnboard_ExistingMode_NotWritable` | FR4, FR8 | `chmod 0500` dir → `not_writable`, no `lifecycle/` written (skipped as root) |
| `TestOnboard_ExistingMode_AlreadyInitialised` | FR4, NFR2 | Pre-existing `lifecycle/config.yaml` → `alreadyInitialised=true`, file untouched |
| `TestOnboard_NewMode_CreatesCleanProject` | FR6, FR7, NFR3 | Fresh parent + name → exactly one new entry under parent, clean scaffold matches CLI-`init` reference file-for-file, registered at resolved path |
| `TestOnboard_NewMode_TargetExists` | FR4, FR8 | Target pre-exists → `target_exists`, existing content untouched |
| `TestOnboard_NewMode_ParentMissing` | FR4, FR8 | Missing parent → `parent_missing` |
| `TestOnboard_NewMode_ParentNotWritable` | FR4, FR8 | `chmod 0500` parent → `parent_not_writable`, nothing written (skipped as root) |
| `TestOnboard_NewMode_InvalidDirName` | FR4, FR8 | Empty, `a/b`, `a\b`, `..` names → `invalid_name` before any write; parent entry count unchanged |
| `TestOnboard_NewMode_ScaffoldFailureRollsBackCreatedDir` | FR8 | Process umask dropped so `os.Mkdir` yields a write-protected (but still owner-removable) target; `initcmd.ScaffoldProject`'s first nested `MkdirAll` fails deterministically; asserts the created directory is fully removed |
| `TestOnboard_ExistingMode_ScaffoldFailureNeverRemovesDir` | FR8 | Same class of induced scaffold failure (a plain file blocking `lifecycle/`) in existing mode; asserts the pre-existing directory and its content are never removed |
| `TestOnboard_NewMode_TraversalNameCannotEscapeParent` | NFR1 | `dirName="../escape"` → `invalid_name`, grandparent directory entry count unchanged, no sibling created |
| `TestOnboard_ExistingMode_TargetInsideConfigDirRejected` | NFR1 | `path` inside `$XDG_CONFIG_HOME/kaos-control` → `invalid_path` |
| `TestOnboard_NewMode_TargetInsideConfigDirRejected` | NFR1 | `parent` inside `$XDG_CONFIG_HOME/kaos-control` → `invalid_path`, nothing written |
| `TestCheckDirectory_ExistingMode_NormalisesWhitespaceAndTilde` | FR9 | `"  ~/sub-project  "` → `resolvedPath` matches the expanded, trimmed absolute path |
| `TestOnboard_NewMode_ResolvedPathMatchesWrittenPath` | FR9 | `"~"`-prefixed, whitespace-padded `parent` → `resolvedPath` in the create response matches exactly where the project was written on disk |

Pre-existing coverage for the single-mode (`mode` omitted → `"existing"`)
`check-directory`/create/list/update/delete/init paths lives in
`tests/integration/projects_crud_test.go` and is unaffected by this change
(re-run alongside as a regression check — see command below).

```
go test ./tests/integration/ -tags integration \
  -run 'TestOnboard|TestCheckDirectory|TestCreateProject|TestListProjects|TestGetProject|TestUpdateProject|TestDeleteProject|TestInitProject'
```

## Requirement → test traceability

| Requirement | Covered by |
|---|---|
| FR1 (mode selection) | Exercised implicitly by every `mode: "existing"` / `mode: "new"` test above (server dispatch in `handleCreateProject`/`handleCheckDirectory`) |
| FR2 (existing-directory input) | `TestOnboard_ExistingMode_*` |
| FR3 (new-directory input: parent + name) | `TestOnboard_NewMode_*`, `TestResolveNewTarget` |
| FR4 (pre-write validation) | `TestOnboard_ExistingMode_PathMissing/NotADirectory/NotWritable/AlreadyInitialised`, `TestOnboard_NewMode_TargetExists/ParentMissing/ParentNotWritable/InvalidDirName`, `TestValidateDirName` |
| FR5 (non-destructive existing scaffold) | `TestOnboard_ExistingMode_NonDestructive` |
| FR6 (new-directory creation, target only) | `TestOnboard_NewMode_CreatesCleanProject` |
| FR7 (registration) | `TestOnboard_ExistingMode_NonDestructive`, `TestOnboard_NewMode_CreatesCleanProject` (GET after create) |
| FR8 (distinct error reporting + rollback) | All `*_PathMissing/NotADirectory/NotWritable/TargetExists/ParentMissing/ParentNotWritable/InvalidDirName` tests (distinct codes) + `*_ScaffoldFailure*` tests (rollback) |
| FR9 (path normalisation) | `TestNormalizePath`, `TestCheckDirectory_ExistingMode_NormalisesWhitespaceAndTilde`, `TestOnboard_NewMode_ResolvedPathMatchesWrittenPath` |
| NFR1 (path safety / sandbox) | `TestResolveNewTarget` traversal/absolute cases, `TestOnboard_NewMode_TraversalNameCannotEscapeParent`, `TestOnboard_*Mode_TargetInsideConfigDirRejected` |
| NFR2 (idempotency/safety) | `TestOnboard_ExistingMode_AlreadyInitialised` |
| NFR3 (consistency with CLI `init`) | `TestOnboard_ExistingMode_NonDestructive`, `TestOnboard_NewMode_CreatesCleanProject` (both compare against `initcmd.Run` reference output) |
| NFR4 (feedback latency) | Not independently timed — `check-directory` requests in this suite complete in single-digit milliseconds against a local filesystem with no scaffolding performed, consistent with the ~500 ms budget; no dedicated latency assertion was added since local-disk `os.Stat` calls are not a realistic source of regression |

`make test-unit` and the integration suite (`make test-integration`, or the
`-run` filters above) pass locally.
