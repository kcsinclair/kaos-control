---
title: CLI Init Scaffold Command
type: requirement
status: planning
lineage: cli-init-scaffold
priority: high
parent: ideas/cli-init-scaffold.md
labels:
    - feature
    - onboarding
    - backend
    - go
    - v1
release: KC-Release0
assignees:
    - role: product-owner
      who: agent
---

# CLI Init Scaffold Command

## Problem

Setting up a new kaos-control project today requires manual directory creation, copy-pasting a `lifecycle/config.yaml` from an existing project, and writing a `CLAUDE.md` by hand. This multi-step process is error-prone (missed directories, stale config templates) and discourages adoption — a developer who has never used the tool must read the full spec before they can make a first commit.

A single `kaos-control init` command should eliminate this friction entirely.

## Goals / Non-goals

### Goals

- Provide a `kaos-control init [<path>]` subcommand that creates all standard lifecycle directories and seed files in one invocation.
- Emit a valid `lifecycle/config.yaml` pre-populated with the default roles, stages, agent definitions, and workflow settings from the spec (§5, §6, §7).
- Emit a project-root `CLAUDE.md` containing the standard guidance Claude Code needs (repo layout, artifact conventions, frontmatter requirements, commit conventions, agent roles).
- Emit `.claude/settings.json` stub with sane defaults for Claude Code integration (e.g. allowed tools, permission prompts).
- Make the command safe to run in an existing repository — never overwrite files that already exist; report what was skipped.
- Make the command work in both empty directories and existing repos with code already present.

### Non-goals

- Interactive wizard or prompts (v1 is non-interactive; flags only).
- Git repository initialisation (`git init`) — the user is expected to handle this themselves or already have a repo.
- Registering the project with the kaos-control app config (`~/.kaos-control/projects/*.yaml`) — that is a separate concern.
- Generating language-specific code scaffolds (e.g. Go `go.mod`, Node `package.json`).
- Remote template fetching or template registries.

## Detailed Requirements

### Functional

#### FR-1: Subcommand interface

The binary must accept `init` as a subcommand:

```
kaos-control init [<path>] [--force]
```

- `<path>` defaults to the current working directory (`.`) when omitted.
- `<path>` may be absolute or relative; the command must resolve it to an absolute path.
- If `<path>` does not exist, create it (including intermediate directories).
- `--force` allows overwriting existing seed files (`lifecycle/config.yaml`, `CLAUDE.md`, `.claude/settings.json`). Without this flag, existing files are never overwritten.

#### FR-2: Directory scaffold

Create the following directories under `<path>/lifecycle/`, each containing a `.gitkeep` file so they are tracked by git even when empty:

| Directory | Purpose |
|---|---|
| `ideas/` | Originating idea artifacts |
| `requirements/` | Detailed requirement artifacts |
| `backend-plans/` | Backend implementation plans |
| `frontend-plans/` | Frontend implementation plans |
| `test-plans/` | Test plans |
| `tests/` | Test artifacts documenting suites |
| `prototypes/` | Prototype artifacts |
| `releases/` | Release artifacts |
| `sprints/` | Sprint artifacts |
| `defects/` | Defect artifacts raised by QA |

Also create `<path>/tests/` (repo-root integration test directory) with a `.gitkeep`.

#### FR-3: Seed `lifecycle/config.yaml`

Write a valid `lifecycle/config.yaml` with:

- `roles`: the seven standard roles (`product-owner`, `analyst`, `backend-developer`, `frontend-developer`, `test-developer`, `qa`, `reviewer`, `approver`).
- `stages`: the ten default stage-to-directory mappings matching FR-2.
- `agents`: the six standard agent definitions (`requirements-analyst`, `planning-analyst`, `backend-developer`, `frontend-developer`, `test-developer`, `qa`) with their default prompt templates, `allowed_write_paths`, and `driver: claude-code-cli`.
- `required_plans.ticket`: `[plan-backend, plan-frontend, plan-test]`.
- `git.default_branch`: `main`.
- `users`: an empty list (user fills in later).

The emitted YAML must parse without error by kaos-control's existing `config.LoadProject()`.

#### FR-4: Seed `CLAUDE.md`

Write a `CLAUDE.md` at the project root covering:

1. Repository layout (matching the scaffolded structure).
2. Lineage filename convention (slug, monotonic index, suffix rules).
3. Frontmatter requirements (required fields, type vocabulary, status vocabulary).
4. Commit conventions (small, focused commits; reference artifact slugs).
5. Agent roles and scope of writes.

The content must be derived from the authoritative spec sections (§3.3, §4, §5, §7) and be accurate for a freshly scaffolded project.

#### FR-5: Seed `.claude/settings.json`

Write a `.claude/settings.json` containing a minimal valid configuration stub. At minimum this should include permission settings that allow Claude Code to operate within the lifecycle scope without excessive prompting.

#### FR-6: Idempotency and safety

- If a directory already exists, skip it silently.
- If a file already exists and `--force` is not set, skip it and print a message to stderr: `skipped: <relative-path> (already exists; use --force to overwrite)`.
- If `--force` is set, overwrite seed files but never delete directories or non-seed files.
- Exit code `0` on success (including when files were skipped).
- Exit code `1` on hard errors (permission denied, invalid path, I/O failure).

#### FR-7: Output

Print a summary to stdout listing every directory and file created or skipped. Example:

```
Initialized kaos-control project at /home/user/my-project

  created  lifecycle/ideas/
  created  lifecycle/requirements/
  ...
  created  lifecycle/config.yaml
  skipped  CLAUDE.md (already exists)
  created  .claude/settings.json
```

### Non-functional

#### NFR-1: No network access

The command must work entirely offline. All templates are compiled into the binary.

#### NFR-2: Performance

Initialisation must complete in under 1 second for a local filesystem.

#### NFR-3: Embed templates

Seed file content must be embedded in the Go binary via `embed.FS` (or generated in code), not read from external template files at runtime.

#### NFR-4: Cross-platform paths

Use `filepath.Join` and `os.MkdirAll` for all path construction. The command must work on macOS, Linux, and Windows.

## Acceptance Criteria

- [ ] `kaos-control init` in an empty directory creates all ten `lifecycle/` subdirectories, each with a `.gitkeep`.
- [ ] `kaos-control init` creates `lifecycle/config.yaml` that passes `config.LoadProject()` validation.
- [ ] `kaos-control init` creates a `CLAUDE.md` at the project root with sections covering layout, lineage, frontmatter, commits, and roles.
- [ ] `kaos-control init` creates `.claude/settings.json` with a valid JSON structure.
- [ ] `kaos-control init` creates `tests/.gitkeep` at the project root.
- [ ] Running `kaos-control init` twice without `--force` does not overwrite any files; skipped files are reported on stderr.
- [ ] Running `kaos-control init --force` overwrites seed files but preserves existing directories and non-seed files.
- [ ] `kaos-control init /tmp/new-project` creates the target directory if it does not exist.
- [ ] `kaos-control init` in a directory with existing code (e.g. a Go module) does not modify any existing files outside the scaffold set.
- [ ] The emitted `lifecycle/config.yaml` includes all six standard agent definitions with valid prompt templates.
- [ ] The command exits `0` on success and `1` on hard errors.
- [ ] All seed content is embedded in the binary — no network access or external file reads at runtime.
- [ ] `make build` succeeds with the new subcommand included.
- [ ] `make test-unit` passes with tests covering the init logic.

## Resolved Questions

- Should the seed `CLAUDE.md` be parameterised (e.g. project name, primary language) or entirely static in v1?

> parameterised, the init process should ask for any information it needs to complete the setup.

- Should `kaos-control init` also create a `.gitignore` entry for the SQLite index file (`lifecycle/.kaos-control.db` or similar)?

> Yes.

- Should the `--force` flag be split into per-file flags (e.g. `--force-config`, `--force-claude-md`) or is a single flag sufficient?

> Separate force is a good idea.
