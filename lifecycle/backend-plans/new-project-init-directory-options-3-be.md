---
created: "2026-08-14T16:27:23+10:00"
title: "New Project Init: Existing or New Directory — Backend Plan"
type: plan-backend
status: done
lineage: new-project-init-directory-options
parent: lifecycle/requirements/new-project-init-directory-options-2.md
---

# New Project Init: Existing or New Directory — Backend Plan

Implements the two-mode New Project onboarding (`Use existing directory` / `Create
new directory`) on the server. Both modes converge on a fully-scaffolded, registered
project via `initcmd.ScaffoldProject` — indistinguishable at rest from a CLI-`init`
project ([[cli-init-scaffold]], [[kaos-control-init-bootstrap]]). See sibling plans
[[new-project-init-directory-options]] (frontend) and (test).

Current surface being extended:
- `internal/config/config.go` — `ValidatePath`, `ValidatePathFormat`, `IsInitialised`,
  `SaveProjectEntry`, `DeleteProjectEntry`, `ProjectEntry`, `kaosControlConfigDir`.
- `internal/http/projects.go` — `handleCreateProject`, `handleCheckDirectory`,
  `isWritable`, `projectSummary`.
- `internal/initcmd/initcmd.go` — `ScaffoldProject` (already idempotent / non-destructive;
  satisfies FR5 for free — a skipped file is `Result{Created:false}`).
- `internal/sandbox/sandbox.go` — `Resolve` (relative-path traversal guard).
- Routes: `internal/http/server.go` lines ~168–176.

Design decision driving the milestones: **dir creation, scaffold, and registration for
new-directory mode must happen inside a single backend operation** so that FR8's
"no partial scaffold left behind" cleanup can remove exactly the directory the tool
created. The mode-aware logic therefore lands in the create/onboard path, not split
across `create` then `init`.

## Milestone 1: Path normalisation & target-resolution helpers

**Description:** Add pure, unit-testable helpers in `internal/config` for FR9 (trim +
`~` expansion) and FR3/FR4 target resolution. `NormalizePath(raw string) string` trims
leading/trailing whitespace and expands a leading `~` / `~/` to the server user's home
(`os.UserHomeDir`). `ValidateDirName(name string) error` rejects empty names and any
name containing `/`, `\`, or a `..` traversal segment (FR4). `ResolveNewTarget(parent,
name string) (string, error)` normalises the parent, validates the name, and joins them
via `sandbox.Resolve(parent, name)` so a crafted name cannot escape the parent (NFR1).

**Files to change:**
- `internal/config/config.go` — add `NormalizePath`, `ValidateDirName`, `ResolveNewTarget`.
- `internal/config/config_test.go` — unit tests for each (covered in the test plan too).

**Acceptance criteria:**
- `NormalizePath("  ~/foo  ")` returns `<home>/foo`; a path with no `~` is only trimmed.
- `ValidateDirName` returns a distinct error for empty, `a/b`, `a\b`, `..`, and `a/../b`.
- `ResolveNewTarget` returns an absolute joined path for a valid name and a
  traversal error (wrapping `sandbox.ErrPathTraversal`) for `../escape`.
- All new functions have no filesystem side effects except `NormalizePath`'s
  `os.UserHomeDir` lookup.

## Milestone 2: Mode-aware validation endpoint

**Description:** Extend `handleCheckDirectory` to serve both modes so the UI can give
pre-submit feedback (NFR4, ~500 ms). Accept `{mode: "existing"|"new", path?, parent?,
name?}`. For `existing`: normalise `path`, then report `exists`, `isDir`, `writable`,
`initialised`. For `new`: normalise `parent`, validate `name`, resolve the target, and
report `parentExists`, `parentWritable`, `nameValid` (+ reason), `targetExists`,
`resolvedPath`. Always return the normalised `resolvedPath` so the UI can display exactly
what will be written (FR9). Keep the existing route ordering (registered before
`/{project}`).

**Files to change:**
- `internal/http/projects.go` — rework `handleCheckDirectory` request/response structs;
  reuse `config.NormalizePath`, `ValidateDirName`, `ResolveNewTarget`, `IsInitialised`,
  `isWritable`.
- `internal/http/projects.go` — add a `checkDirectoryResult` response type with the
  per-rule booleans + `resolvedPath` + optional `reason`.

**Acceptance criteria:**
- `existing` mode returns `exists/isDir/writable/initialised` and the resolved path for a
  real directory, and `exists=false` for a missing path — HTTP 200 either way (validation,
  not error).
- `new` mode returns `parentExists`, `parentWritable`, `nameValid`, `targetExists`, and
  `resolvedPath`; an invalid `name` yields `nameValid=false` with a specific `reason`.
- A path/parent inside the kaos-control config dir is rejected (reuses the
  `ValidatePathFormat` config-dir guard).
- Response is returned within the NFR4 budget for a local path (no scaffolding performed).

## Milestone 3: Mode-aware onboarding (create → validate → scaffold → register)

**Description:** Make the New Project create path mode-aware and end-to-end. Extend the
`handleCreateProject` body with `mode`, `parent`, and a directory `name` (distinct from the
project `name`). Flow:
1. Validate project name (`config.ValidateProjectName`) and uniqueness (existing check).
2. Resolve the target: `existing` → `NormalizePath(path)` + `ValidatePathFormat`;
   `new` → `ResolveNewTarget(parent, name)`.
3. Pre-write validation (FR4), each failure returning a **distinct** error code/message
   (`path_missing`, `not_a_directory`, `not_writable`, `target_exists`, `invalid_name`,
   `already_initialised`, `parent_missing`, `parent_not_writable`).
4. Reject an already-initialised target (`config.IsInitialised`) rather than re-scaffolding
   (FR4/NFR2) — return an `already_initialised` result, write nothing.
5. `new` mode only: `os.Mkdir(target, 0o755)` (the target only, not missing parents — FR6);
   record `createdDir=true` for rollback.
6. Scaffold via `initcmd.ScaffoldProject{ProjectRoot: target, ProjectName: name,
   OwnerEmail: session user}` — idempotent/non-destructive guarantees FR5.
7. Register (`config.SaveProjectEntry` + `s.RegisterProject`) using the resolved path (FR7).
8. On any failure after step 5 in `new` mode, best-effort `os.RemoveAll(target)` **only if**
   `createdDir` (FR8); roll back the registry entry as `handleCreateProject` already does.

**Files to change:**
- `internal/http/projects.go` — extend `handleCreateProject` body + logic; factor the
  scaffold-and-register block so both modes share it.
- `internal/http/projects.go` — extend `projectSummary`/create response with
  `resolvedPath`, `created []string`, `alreadyInitialised bool`, and a
  `partialCompletion bool` (true when scaffolding an existing dir added some but not all
  lifecycle subdirs — drives the frontend info notification per the resolved question).

**Acceptance criteria:**
- `existing` mode into a populated directory leaves every pre-existing file byte-for-byte
  unchanged and only adds missing scaffold files (`created` lists only new paths).
- `new` mode creates the target dir (and only it) and scaffolds a clean project; a missing
  parent or unwritable parent is rejected before any write.
- An already-initialised target returns `alreadyInitialised=true` and modifies nothing.
- A forced scaffold failure in `new` mode leaves no directory behind (`os.RemoveAll` ran);
  in `existing` mode the pre-existing directory is never removed.
- On success the project is registered at `resolvedPath` and immediately available for
  indexing (same registration path as the current flow, FR7).

## Milestone 4: Path-safety hardening (NFR1)

**Description:** Ensure no user string reaches the filesystem un-sanitised. Route the
new-directory `name` through `sandbox.Resolve` (done in M1) and re-assert the config-dir
guard on the *resolved* target for both modes before any write. Add a regression assertion
that a `name` of `../../etc` or an absolute `name` is rejected with a traversal/absolute
error, never joined.

**Files to change:**
- `internal/http/projects.go` — call the config-dir recheck on the resolved target in the
  onboarding flow; map `sandbox.ErrPathTraversal`/`ErrAbsolutePath` to the `invalid_name`
  error code.

**Acceptance criteria:**
- A crafted traversal `name` cannot escape the chosen parent (verified against
  `sandbox.Resolve` semantics).
- A resolved target equal to or inside the kaos-control config dir is rejected.
- Every FR4 rejection is emitted as a specific, human-readable message (FR8) — no generic
  "invalid" catch-all.

## Milestone 5: Routes, response types & wiring

**Description:** Confirm route registration still orders `check-directory` before
`/{project}` and that the create route carries the extended body. Update the JSON response
contract consumed by the frontend ([[new-project-init-directory-options]] frontend plan).

**Files to change:**
- `internal/http/server.go` — no new routes needed; verify ordering at lines ~168–176.
- `internal/http/projects.go` — finalise response JSON shapes.

**Acceptance criteria:**
- `POST /projects` accepts and dispatches both modes; `POST /projects/check-directory`
  serves both modes.
- Response JSON includes `resolvedPath`, `created`, `alreadyInitialised`,
  `partialCompletion` and is stable for the frontend contract.
- `make lint` and `make test-unit` pass.
