---
title: 'New Project Init: Existing or New Directory — Requirements'
type: requirement
status: blocked
lineage: new-project-init-directory-options
priority: normal
parent: ideas/new-project-init-directory-options.md
labels:
    - feature
    - frontend
    - onboarding
    - ux
    - ui
    - v1
release: KC-Release5
assignees:
    - role: product-owner
      who: agent
---

# New Project Init: Existing or New Directory — Requirements

## Problem

The "New Project" flow assumes the user is creating a fresh location for their project. Users who already maintain a project directory on disk have no first-class way to bring it under lifecycle management — they must either create a throwaway folder or edit registration YAML by hand. Conversely, users starting from a blank slate must pre-create a directory outside the tool before registering it. A single entry point that supports both an **existing directory** and a **new directory** removes this friction and makes onboarding predictable for both audiences.

## Goals / Non-goals

### Goals

- Offer two explicit initialisation modes when the user starts a New Project: **Use existing directory** and **Create new directory**.
- For the existing-directory mode, scaffold the `lifecycle/` structure and config into a user-chosen pre-existing folder **without deleting, overwriting, or reordering any pre-existing files**.
- For the new-directory mode, create a new folder from a user-supplied parent location plus directory name, then initialise it from scratch.
- Both modes converge on the same end state: a fully initialised, registered kaos-control project ready to index.
- Validate the target path up front and surface clear, actionable errors before any files are written.

### Non-goals

- No change to the underlying scaffold *content* (directory set, `config.yaml`, `CLAUDE.md`) beyond where it is placed — that is covered by [[cli-init-scaffold]] and [[kaos-control-init-bootstrap]].
- No migration or reformatting of a user's existing files (e.g. importing pre-existing markdown into `lifecycle/`).
- No graphical filesystem browser is required by this requirement; a typed/pasted path is sufficient (a picker may be delivered separately — see Open Questions).
- No multi-directory or monorepo-aware initialisation.

## Detailed Requirements

### Functional

**FR1 — Mode selection.** The New Project entry point presents exactly two mutually exclusive modes: "Use existing directory" and "Create new directory". One mode is selected at a time; the form fields shown adapt to the selected mode.

**FR2 — Existing-directory input.** In existing-directory mode the user supplies a single absolute path to a directory that already exists on disk.

**FR3 — New-directory input.** In new-directory mode the user supplies (a) a parent directory path that already exists, and (b) a new directory name. The target path is the parent joined with the name.

**FR4 — Pre-write validation.** Before any files are written, the backend validates the resolved target:
- Existing mode: the path exists, is a directory, and is readable/writable by the server process.
- New mode: the parent exists and is writable, and the target (parent + name) does **not** already exist.
- The directory name in new mode is non-empty and contains no path separators or traversal segments (`/`, `\`, `..`).
- A target that already contains an initialised kaos-control project (a `lifecycle/` directory with a `config.yaml`) is reported as already-initialised and rejected rather than re-scaffolded.

**FR5 — Non-destructive existing-directory scaffold.** Scaffolding into an existing directory must only create missing files/directories. Any pre-existing file or directory (including a pre-existing `lifecycle/` subtree, `CLAUDE.md`, or unrelated content) must be left byte-for-byte unchanged. If a required scaffold file already exists, it is skipped, not overwritten.

**FR6 — New-directory creation.** In new-directory mode the backend creates the target directory (and only the target; it does not create missing parents) and then scaffolds into it as for a clean init.

**FR7 — Registration.** On success, the project is registered (`~/.kaos-control/projects/*.yaml`) pointing at the resolved target path, using the same registration path as the existing New Project flow, and becomes immediately available for indexing.

**FR8 — Error reporting.** Every validation failure (FR4) returns a distinct, human-readable message identifying the cause (path missing, not a directory, not writable, already exists, invalid name, already initialised). No partial scaffold is left behind if scaffolding fails mid-way in new-directory mode (best-effort cleanup of the directory the tool created).

**FR9 — Path normalisation.** Leading/trailing whitespace is trimmed and `~` is expanded to the server user's home directory before validation, so the displayed resolved path matches what is written.

### Non-functional

**NFR1 — Path safety.** All path handling routes through the existing sandbox/path-traversal-safe resolver ([internal/sandbox](internal/sandbox)); no user-supplied string reaches the filesystem un-sanitised.

**NFR2 — Idempotency & safety.** Re-running existing-directory init against the same already-initialised directory is a no-op with an already-initialised message, never a destructive re-scaffold (see FR4/FR5).

**NFR3 — Consistency.** Both modes produce a project indistinguishable at rest from one created by the CLI `init` command ([[cli-init-scaffold]]).

**NFR4 — Feedback latency.** Validation feedback for a typed path is returned promptly (target: within ~500 ms for a local path) so the user learns of problems before submitting.

## Acceptance Criteria

- [ ] New Project entry point shows two selectable modes: "Use existing directory" and "Create new directory" (FR1).
- [ ] Existing mode accepts an absolute directory path; new mode accepts a parent path + directory name (FR2, FR3).
- [ ] Submitting existing mode with a path that does not exist, is a file, or is not writable is rejected with a mode-specific message and writes nothing (FR4, FR8).
- [ ] Submitting new mode where the target already exists is rejected; where the parent does not exist or is not writable is rejected (FR4, FR8).
- [ ] A directory name containing `/`, `\`, `..`, or that is empty is rejected before any filesystem write (FR4).
- [ ] Initialising into an existing directory that already contains files leaves every pre-existing file byte-for-byte unchanged and only adds missing scaffold files (FR5).
- [ ] Initialising a target that is already an initialised kaos-control project returns an "already initialised" result and does not modify it (FR4, NFR2).
- [ ] New mode creates the target directory and scaffolds a clean project into it (FR6).
- [ ] After either mode succeeds, a project registration exists pointing at the resolved path and the project is available for indexing (FR7), matching a CLI-`init` project at rest (NFR3, see [[cli-init-scaffold]]).
- [ ] A scaffold failure in new mode leaves no partially created project directory behind (FR8).
- [ ] All target paths are resolved through the sandbox resolver; a crafted traversal path cannot escape the intended target (NFR1).
- [ ] `~` expansion and whitespace trimming are applied before validation and reflected in the resolved path shown to the user (FR9).

## Open Questions

- Does this requirement need a graphical directory picker, or is a typed/pasted path acceptable for v1? (Non-goals assume typed path; confirm with product-owner.)

> Typed path for v1

- When scaffolding into an existing directory that already has a partial `lifecycle/` tree (some but not all expected subdirectories), should the tool complete the missing pieces silently, or warn the user first?

- Should the New Project form pre-fill or suggest a default parent location (e.g. last-used path or the server user's home directory)?

- Is there a maximum-depth or allowlist policy on where projects may be created (e.g. must live under a configured projects root), or is any writable path permitted?
