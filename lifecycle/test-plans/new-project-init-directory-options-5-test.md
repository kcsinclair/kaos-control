---
title: "New Project Init: Existing or New Directory — Test Plan"
type: plan-test
status: done
lineage: new-project-init-directory-options
parent: lifecycle/requirements/new-project-init-directory-options-2.md
---

# New Project Init: Existing or New Directory — Test Plan

Covers the two-mode onboarding across unit (config helpers) and integration (HTTP)
layers, verifying every acceptance criterion in the requirement. Integration tests live in
`tests/integration/` and drive the real router via the `testEnv` harness (auto-logs in as
admin — see project memory); all filesystem work uses `t.TempDir()` so nothing pollutes
the tree. Backend contract per [[new-project-init-directory-options]] (backend plan).

Files added/changed:
- `internal/config/config_test.go` — unit tests for the new helpers.
- `tests/integration/project_init_directory_options_test.go` — new integration file.
- `lifecycle/tests/new-project-init-directory-options-*.md` — artifact describing coverage.

## Milestone 1: Unit tests — normalisation & resolution helpers

**Description:** Test `NormalizePath`, `ValidateDirName`, `ResolveNewTarget` (FR9, FR4,
NFR1) as pure functions.

**Files to change:**
- `internal/config/config_test.go`.

**Acceptance criteria:**
- `NormalizePath` trims whitespace and expands a leading `~`/`~/`; leaves other paths only
  trimmed.
- `ValidateDirName` returns distinct errors for empty, `a/b`, `a\b`, `..`, `a/../b`.
- `ResolveNewTarget` joins a valid name onto the parent and rejects `../escape` and an
  absolute name with a traversal/absolute error.

## Milestone 2: Existing mode — happy path & non-destructive scaffold (FR5)

**Description:** Create a temp directory pre-populated with sentinel files (a `CLAUDE.md`
with known bytes, a nested `lifecycle/ideas/keep.md`, an unrelated `data.txt`). Onboard in
existing mode and assert non-destruction.

**Files to change:**
- `tests/integration/project_init_directory_options_test.go`.

**Acceptance criteria:**
- Every pre-existing file is byte-for-byte identical after onboarding (hash before/after).
- Only missing scaffold files/dirs are created; `created` lists exactly those.
- The project is registered at the resolved path and `lifecycle/config.yaml` exists (project
  is initialised, available for indexing — FR7).
- Result at rest matches a CLI-`init` project ([[cli-init-scaffold]]): same key files present.

## Milestone 3: Existing mode — validation failures (FR4/FR8)

**Description:** Exercise each rejection with a distinct assertion.

**Files to change:**
- `tests/integration/project_init_directory_options_test.go`.

**Acceptance criteria:**
- Non-existent path → rejected with a path-missing message; nothing written.
- Path that is a file (not a directory) → rejected with a not-a-directory message.
- Non-writable directory (chmod `0o500`, skip if running as root) → rejected with a
  not-writable message; nothing written.
- Already-initialised directory → `alreadyInitialised` result, directory unmodified (NFR2).

## Milestone 4: New mode — happy path & creation semantics (FR6)

**Description:** Provide an existing parent temp dir + a new directory name; onboard and
assert clean creation.

**Files to change:**
- `tests/integration/project_init_directory_options_test.go`.

**Acceptance criteria:**
- The target directory is created (and only the target — a missing intermediate parent is
  never auto-created).
- A clean project is scaffolded into it (`lifecycle/config.yaml` present) and registered at
  the resolved path (FR7).
- The result matches a CLI-`init` project at rest (NFR3).

## Milestone 5: New mode — validation failures & invalid names (FR4/FR8)

**Description:** Cover every new-mode rejection.

**Files to change:**
- `tests/integration/project_init_directory_options_test.go`.

**Acceptance criteria:**
- Target already exists → rejected with a target-exists message; existing content untouched.
- Parent does not exist → rejected with a parent-missing message.
- Parent not writable (chmod `0o500`, skip as root) → rejected; nothing written.
- Directory name containing `/`, `\`, `..`, or empty → rejected before any filesystem write
  (assert the target was never created).

## Milestone 6: Cleanup, path safety & normalisation (FR8, NFR1, FR9)

**Description:** Verify rollback and safety guarantees.

**Files to change:**
- `tests/integration/project_init_directory_options_test.go`.

**Acceptance criteria:**
- Induced scaffold failure in new mode (e.g. a pre-created read-only collision inside the
  freshly-made target) leaves **no** directory behind — the tool removed the dir it created.
- The same induced failure in existing mode never removes the pre-existing directory.
- A crafted traversal directory name cannot escape the parent (target stays within it);
  a resolved target inside the kaos-control config dir is rejected (NFR1).
- A whitespace-padded and/or `~`-prefixed input resolves to the expected absolute path, and
  the resolved path reported to the client matches what was written (FR9).

## Milestone 7: Coverage artifact

**Description:** Record what the test code covers in a `lifecycle/tests/` artifact per the
project convention.

**Files to change:**
- `lifecycle/tests/new-project-init-directory-options-*.md` (type `test`, correct lineage
  index continuing after this plan).

**Acceptance criteria:**
- Artifact lists each requirement (FR1–FR9, NFR1–NFR4) and the test(s) covering it.
- `make test-unit` and the integration suite pass locally.
