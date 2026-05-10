---
title: "CLI Init Scaffold — Integration Test Suite"
type: test
status: draft
lineage: cli-init-scaffold
parent: lifecycle/test-plans/cli-init-scaffold-5-test.md
---

# CLI Init Scaffold — Integration Test Suite

Integration tests for the `kaos-control init` subcommand, implemented against the compiled binary. All tests use `t.TempDir()` for isolation; no fixture files pollute the working tree.

## Test File

`tests/cli_init_test.go` — package `cli_test`, build tag `//go:build integration`.

## Setup

`TestMain` compiles the binary once into a `os.MkdirTemp` directory before any test runs and stores the path in the package-level `binPath` variable. All tests invoke `runInit(t, args...)` which calls `exec.Command(binPath, "init", args...)` and captures stdout, stderr, and exit code.

## Scenarios Covered

### Milestone 4 — Full Init Flow (Empty Directory) · `TestInit_FullFlow_EmptyDir`

- Verifies exit code 0.
- Verifies stdout contains `Initialized kaos-control project at <abs-path>`.
- Verifies all 11 `lifecycle/` subdirectories and `tests/` contain `.gitkeep` and are listed as `created` in stdout.
- Verifies all 4 seed files (`lifecycle/config.yaml`, `CLAUDE.md`, `.claude/settings.json`, `.gitignore`) exist and are listed as `created` in stdout.
- Parses `lifecycle/config.yaml` with `gopkg.in/yaml.v3` — must be valid YAML.
- Parses `.claude/settings.json` with `encoding/json` — must be valid JSON.
- Checks `CLAUDE.md` contains the five required sections: Repository Layout, Lineage Filename Convention, Frontmatter Requirements, Commit Conventions, Agent Roles.
- Checks `.gitignore` contains the line `lifecycle/.kaos-control.db`.

### Milestone 5 — Idempotency (Double Run) · `TestInit_Idempotency`

- Runs init twice on the same directory without any force flags.
- Second run must exit 0.
- Second run stdout must contain no `created` items; all 11 dirs and 4 files must appear as `skipped … (already exists)`.
- Second run stderr must contain `skipped: <file> (already exists; use --force to overwrite)` for each seed file.
- Seed file contents are snapshotted after the first run and byte-compared after the second; they must be identical.

### Milestone 6 — Force Flags · `TestInit_ForceFlags`

**Subtest `force-config`:**
- Runs init, then corrupts `lifecycle/config.yaml` with invalid content.
- Re-runs with `--force-config`; verifies `lifecycle/config.yaml` is now valid YAML (overwritten).
- Verifies `CLAUDE.md` is byte-identical to its value before the second run (was not overwritten).

**Subtest `force-all`:**
- Runs init, then plants the string `"MARKER"` in all four seed files and an unrelated `custom.txt`.
- Re-runs with `--force`; verifies all seed files no longer contain `"MARKER"` and appear as `created` in stdout.
- Verifies all 11 directory `.gitkeep` files still exist.
- Verifies `custom.txt` is byte-identical to its planted content (non-seed files are never touched).

### Milestone 7 — Non-Existent Target Path · `TestInit_NonExistentPath`

- Calls `init <base>/nested/project` where neither `nested` nor `project` exist yet.
- Verifies exit code 0, the target directory was created, and the full scaffold is present inside it.

### Milestone 8 — Existing Repo With Code · `TestInit_ExistingCodeRepo`

- Seeds a temp directory with `go.mod`, `main.go`, and `README.md`.
- Runs init; verifies exit code 0.
- Verifies each pre-existing file is byte-identical after init.
- Verifies the lifecycle scaffold was created alongside the existing files.

### Milestone 9 — Error Cases · `TestInit_ErrorCases`

**Subtest `unwritable-path`:**
- Creates a `0o444` (read-only) parent directory.
- Runs init targeting a new subdirectory inside it.
- Verifies exit code 1 and non-empty stderr. Skipped when running as root.

**Subtest `unknown-flag`:**
- Runs `init --this-flag-does-not-exist-xyz`.
- Verifies exit code 1 and non-empty stderr.

## Out of Scope (Milestone 10 — Frontend Banner)

The test plan's Milestone 10 requires testing the `InitRequiredBanner` Vue component rendering when the backend returns a config-missing error. This scenario requires a frontend component test framework (e.g., Vitest with `@vue/test-utils`) that is not yet configured in this repository. No implementation was attempted; a follow-up ticket should set up frontend unit testing infrastructure before revisiting this milestone.
