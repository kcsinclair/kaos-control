---
title: CLI Init Scaffold — Backend Plan
type: plan-backend
status: approved
lineage: cli-init-scaffold
parent: lifecycle/requirements/cli-init-scaffold-2.md
---

# CLI Init Scaffold — Backend Plan

This plan implements the `kaos-control init` subcommand as a pure-Go CLI feature. The command scaffolds lifecycle directories, emits seed files (config, CLAUDE.md, .claude/settings.json, .gitignore), and reports what it created or skipped. All templates are embedded in the binary. No HTTP/API surface is required.

Key design decisions driven by resolved questions in the requirement:
- CLAUDE.md is **parameterised** — the command accepts `--project-name` and `--language` flags and interpolates them into the template.
- `--force` is **split** into granular flags (`--force-config`, `--force-claude-md`, `--force-settings`, `--force-gitignore`) alongside a blanket `--force` that sets all of them.
- A `.gitignore` is emitted containing the SQLite index file pattern.

## Milestone 1: Subcommand Routing in main.go

**Description:** Introduce first-class subcommand dispatch in the binary entry point. Today `main.go` calls `flag.Parse()` with no subcommand awareness. Add a top-level switch on `os.Args[1]` that routes to `init` or falls through to the existing server path (default when no subcommand or `serve` is given).

**Files to change:**
- `cmd/kaos-control/main.go` — Before `flag.Parse()`, inspect `os.Args[1]`. If `"init"`, delegate to `initcmd.Run(os.Args[2:])` and exit. If `"serve"` or no subcommand, continue with existing server startup. Print usage on unrecognised subcommand.

**Acceptance criteria:**
- [ ] `kaos-control` (no args) starts the server as before — zero regression.
- [ ] `kaos-control serve` also starts the server.
- [ ] `kaos-control init` invokes the init path (may error until Milestone 2 is done).
- [ ] `kaos-control unknown` prints usage to stderr and exits 1.
- [ ] No third-party CLI framework is introduced (stdlib `flag` + manual dispatch).

## Milestone 2: Init Command Core — Package and Flag Parsing

**Description:** Create `internal/initcmd/` package with the command's flag parsing, validation, and orchestration entrypoint.

**Files to change:**
- `internal/initcmd/initcmd.go` (new) — Export `Run(args []string) error`. Parse flags from a dedicated `flag.FlagSet`:
  - `<path>` positional (first non-flag arg, defaults to `"."`).
  - `--force` (bool) — sets all granular force flags.
  - `--force-config` (bool) — allow overwriting `lifecycle/config.yaml`.
  - `--force-claude-md` (bool) — allow overwriting `CLAUDE.md`.
  - `--force-settings` (bool) — allow overwriting `.claude/settings.json`.
  - `--force-gitignore` (bool) — allow overwriting `.gitignore`.
  - `--project-name` (string) — interpolated into CLAUDE.md; defaults to the directory name of the resolved path.
  - `--language` (string) — primary language hint for CLAUDE.md; defaults to empty (omitted from template when blank).
- Resolve `<path>` to absolute via `filepath.Abs`. Create it with `os.MkdirAll` if absent.
- Return exit code 0 on success, 1 on hard error.

**Acceptance criteria:**
- [ ] `kaos-control init /tmp/new-project` creates `/tmp/new-project` if it doesn't exist.
- [ ] `kaos-control init` resolves to the current working directory.
- [ ] Relative paths are resolved to absolute.
- [ ] `--force` implies all `--force-*` flags.
- [ ] `--project-name` defaults to the basename of the target directory.
- [ ] Unknown flags produce a usage message and exit 1.

## Milestone 3: Directory Scaffold

**Description:** Create all lifecycle subdirectories and the root `tests/` directory, each containing a `.gitkeep` file.

**Files to change:**
- `internal/initcmd/scaffold.go` (new) — `scaffoldDirs(root string) ([]Result, error)`. Iterates a static list of directory paths, calls `os.MkdirAll`, and writes `.gitkeep` (empty file) into each. Returns a `[]Result` where each entry records the relative path and whether it was `created` or `skipped`.

Directories to create (from FR-2):
```
lifecycle/ideas/
lifecycle/requirements/
lifecycle/backend-plans/
lifecycle/frontend-plans/
lifecycle/test-plans/
lifecycle/tests/
lifecycle/prototypes/
lifecycle/releases/
lifecycle/sprints/
lifecycle/defects/
tests/
```

**Acceptance criteria:**
- [ ] All eleven directories exist after `init`, each with a `.gitkeep`.
- [ ] If a directory already exists, it is skipped silently (no error).
- [ ] If `.gitkeep` already exists inside an existing directory, it is not overwritten.
- [ ] Uses `filepath.Join` for all path construction (cross-platform).
- [ ] Result list accurately reports `created` vs `skipped` for each directory.

## Milestone 4: Embedded Seed Templates

**Description:** Embed the seed file templates into the binary using `embed.FS`. Templates use `text/template` syntax for parameterisation.

**Files to change:**
- `internal/initcmd/templates/` (new directory) — Contains:
  - `config.yaml.tmpl` — Full default `lifecycle/config.yaml` with the seven roles, ten stages, six agent definitions, `required_plans`, and `git.default_branch: main`. Uses `defaultProject()` from `internal/config` as the reference for correctness. Template variables: `{{.ProjectName}}`.
  - `CLAUDE.md.tmpl` — Sections: repo layout, lineage convention, frontmatter requirements, commit conventions, agent roles. Template variables: `{{.ProjectName}}`, `{{.Language}}` (conditionally rendered).
  - `settings.json.tmpl` — Minimal `.claude/settings.json` stub with sane defaults.
  - `gitignore.tmpl` — Contains `lifecycle/.kaos-control.db` and common patterns.
- `internal/initcmd/embed.go` (new) — `//go:embed templates/*` var `templateFS embed.FS`. Export a `renderTemplate(name string, data TemplateData) ([]byte, error)` function.
- `internal/initcmd/initcmd.go` — Define `TemplateData` struct: `ProjectName string`, `Language string`.

**Acceptance criteria:**
- [ ] `config.yaml.tmpl` renders YAML that passes `config.LoadProject()` without error.
- [ ] `CLAUDE.md.tmpl` includes all five required sections (layout, lineage, frontmatter, commits, roles).
- [ ] `CLAUDE.md.tmpl` interpolates `ProjectName` in the heading and omits the language section when `Language` is empty.
- [ ] `settings.json.tmpl` renders valid JSON.
- [ ] `gitignore.tmpl` includes `lifecycle/.kaos-control.db`.
- [ ] All templates are embedded — no filesystem reads at runtime (NFR-1, NFR-3).
- [ ] `make build` succeeds with templates included.

## Milestone 5: Seed File Writer with Idempotency

**Description:** Write rendered seed files to disk, respecting the skip-or-overwrite logic per FR-6.

**Files to change:**
- `internal/initcmd/seedfiles.go` (new) — `writeSeedFiles(root string, data TemplateData, force ForceFlags) ([]Result, error)`. For each seed file:
  1. Render template.
  2. Check if target file exists.
  3. If exists and corresponding force flag is false → skip, print to stderr: `skipped: <rel-path> (already exists; use --force to overwrite)`.
  4. If exists and force flag is true → overwrite.
  5. If not exists → create (including parent dirs via `os.MkdirAll`).
  6. Record result.
- `internal/initcmd/initcmd.go` — `ForceFlags` struct with per-file booleans.

Seed files written:
| Template | Target path | Force flag |
|---|---|---|
| `config.yaml.tmpl` | `lifecycle/config.yaml` | `--force-config` |
| `CLAUDE.md.tmpl` | `CLAUDE.md` | `--force-claude-md` |
| `settings.json.tmpl` | `.claude/settings.json` | `--force-settings` |
| `gitignore.tmpl` | `.gitignore` | `--force-gitignore` |

**Acceptance criteria:**
- [ ] Existing files are never overwritten without the matching force flag.
- [ ] Skipped files produce a stderr message with the relative path.
- [ ] `--force` overwrites seed files but never deletes directories or non-seed files.
- [ ] Parent directories (e.g., `.claude/`) are created if absent.
- [ ] File permissions are 0644.

## Milestone 6: Output Summary and Exit Codes

**Description:** Print a human-readable summary to stdout and set the correct exit code per FR-7.

**Files to change:**
- `internal/initcmd/initcmd.go` — After `scaffoldDirs` and `writeSeedFiles`, print the summary header (`Initialized kaos-control project at <abs-path>`) followed by each result line (`  created  <path>` or `  skipped  <path> (already exists)`). Exit 0 on success, 1 on any hard I/O error.

**Acceptance criteria:**
- [ ] Output matches the format in FR-7 (header line, then indented `created`/`skipped` lines).
- [ ] All created and skipped items appear in the summary.
- [ ] Exit code 0 on success (including partial skips).
- [ ] Exit code 1 on permission denied, invalid path, or I/O failure.
- [ ] Running in an existing project with all files present produces all `skipped` lines and exits 0.

## Milestone 7: Config Template Validation

**Description:** Ensure the emitted `lifecycle/config.yaml` is loadable by the existing config system. This is a correctness gate — if the template drifts from what `LoadProject` expects, init is broken.

**Files to change:**
- `internal/initcmd/initcmd_test.go` (new) — Unit test: render `config.yaml.tmpl` → write to temp dir → call `config.LoadProject(tempDir)` → assert no error. Also assert key fields: six agents exist, seven roles listed, `required_plans.ticket` contains all three plan types.

**Acceptance criteria:**
- [ ] Test renders the config template and loads it with `config.LoadProject` successfully.
- [ ] Test asserts agent count, role list, and required_plans content.
- [ ] Test runs as part of `make test-unit`.
- [ ] If `config.Project` struct changes in future, this test catches template drift.

## Cross-references

- [[cli-init-scaffold-4-fe]] — Frontend plan: no UI changes required for this CLI-only feature, but the frontend plan documents a "project not initialised" guard that depends on this command having run.
- [[cli-init-scaffold-5-test]] — Test plan: integration tests exercise the full init flow, idempotency, force flags, and template validity.
