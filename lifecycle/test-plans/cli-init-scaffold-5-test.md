---
title: "CLI Init Scaffold — Test Plan"
type: plan-test
status: in-development
lineage: cli-init-scaffold
parent: lifecycle/requirements/cli-init-scaffold-2.md
---

# CLI Init Scaffold — Test Plan

This plan covers unit tests for the init command internals and integration tests that exercise the full CLI flow end-to-end. All tests use temporary directories — no fixtures pollute the working tree.

## Milestone 1: Unit Tests — Directory Scaffold

**Description:** Test `scaffoldDirs()` in isolation, verifying it creates the correct directory tree with `.gitkeep` files and handles pre-existing directories.

**Files to change:**
- `internal/initcmd/scaffold_test.go` (new) — Unit tests using `t.TempDir()`.

**Acceptance criteria:**
- [ ] Test: all eleven directories are created in an empty temp dir, each containing `.gitkeep`.
- [ ] Test: calling `scaffoldDirs` twice is idempotent — no errors, `.gitkeep` files not recreated.
- [ ] Test: pre-existing directories with content are not modified (content preserved).
- [ ] Test: result list correctly reports `created` vs `skipped` for each directory.
- [ ] Test: paths use `filepath.Join` (verified by running on the OS's native separator).

## Milestone 2: Unit Tests — Template Rendering

**Description:** Test that each embedded template renders valid output with various input combinations.

**Files to change:**
- `internal/initcmd/embed_test.go` (new) — Unit tests for `renderTemplate()`.

**Acceptance criteria:**
- [ ] Test: `config.yaml.tmpl` renders valid YAML (parse with `gopkg.in/yaml.v3` and assert no error).
- [ ] Test: rendered config passes `config.LoadProject()` validation — six agents, seven roles, three required plan types.
- [ ] Test: `CLAUDE.md.tmpl` with `ProjectName: "My Project"` includes "My Project" in the heading.
- [ ] Test: `CLAUDE.md.tmpl` with empty `Language` omits the language section entirely.
- [ ] Test: `CLAUDE.md.tmpl` with `Language: "Go"` includes "Go" in the appropriate section.
- [ ] Test: `CLAUDE.md.tmpl` contains all five required sections (layout, lineage, frontmatter, commits, roles).
- [ ] Test: `settings.json.tmpl` renders valid JSON (parse with `encoding/json` and assert no error).
- [ ] Test: `gitignore.tmpl` contains the line `lifecycle/.kaos-control.db`.

## Milestone 3: Unit Tests — Seed File Writer Idempotency

**Description:** Test the skip/overwrite logic in `writeSeedFiles()` with various force flag combinations.

**Files to change:**
- `internal/initcmd/seedfiles_test.go` (new) — Unit tests.

**Acceptance criteria:**
- [ ] Test: in an empty dir, all four seed files are created.
- [ ] Test: with all files pre-existing and no force flags, all four are skipped; stderr contains the skip messages.
- [ ] Test: `--force-config` overwrites only `lifecycle/config.yaml`; other existing files are skipped.
- [ ] Test: `--force-claude-md` overwrites only `CLAUDE.md`.
- [ ] Test: `--force-settings` overwrites only `.claude/settings.json`.
- [ ] Test: `--force-gitignore` overwrites only `.gitignore`.
- [ ] Test: `--force` (blanket) overwrites all four seed files.
- [ ] Test: overwriting preserves file permissions at 0644.
- [ ] Test: non-seed files in the directory are never touched regardless of force flags.

## Milestone 4: Integration Test — Full Init Flow (Empty Directory)

**Description:** Run the `kaos-control init` command against an empty temporary directory and verify the complete output.

**Files to change:**
- `tests/cli_init_test.go` (new) — Integration test that invokes the built binary.

**Acceptance criteria:**
- [ ] Test: exit code is 0.
- [ ] Test: stdout contains `Initialized kaos-control project at <abs-path>`.
- [ ] Test: stdout lists all eleven directories as `created`.
- [ ] Test: stdout lists all four seed files as `created`.
- [ ] Test: `lifecycle/config.yaml` exists and is valid YAML.
- [ ] Test: `CLAUDE.md` exists and contains the expected sections.
- [ ] Test: `.claude/settings.json` exists and is valid JSON.
- [ ] Test: `.gitignore` exists and contains `lifecycle/.kaos-control.db`.
- [ ] Test: all `lifecycle/` subdirectories contain `.gitkeep`.
- [ ] Test: `tests/.gitkeep` exists at the project root.

## Milestone 5: Integration Test — Idempotency (Double Run)

**Description:** Run `kaos-control init` twice on the same directory without `--force` and verify no files are overwritten.

**Files to change:**
- `tests/cli_init_test.go` — Add subtest.

**Acceptance criteria:**
- [ ] Test: second run exits 0.
- [ ] Test: second run's stdout shows all items as `skipped`.
- [ ] Test: stderr contains skip messages with `(already exists; use --force to overwrite)` for each seed file.
- [ ] Test: file contents from the first run are byte-identical after the second run.

## Milestone 6: Integration Test — Force Flags

**Description:** Verify that granular force flags overwrite only their target files.

**Files to change:**
- `tests/cli_init_test.go` — Add subtests for each force flag and the blanket `--force`.

**Acceptance criteria:**
- [ ] Test: `--force-config` — modify `lifecycle/config.yaml` content, re-run with flag, verify it was overwritten; verify `CLAUDE.md` was NOT overwritten.
- [ ] Test: `--force` — all four seed files are overwritten; directories remain intact.
- [ ] Test: force flags never delete directories or non-seed files.

## Milestone 7: Integration Test — Non-Existent Target Path

**Description:** Verify `kaos-control init /tmp/<random>/nested/path` creates intermediate directories.

**Files to change:**
- `tests/cli_init_test.go` — Add subtest.

**Acceptance criteria:**
- [ ] Test: target directory is created (including intermediates).
- [ ] Test: full scaffold is present inside the newly-created directory.
- [ ] Test: exit code is 0.

## Milestone 8: Integration Test — Existing Repo with Code

**Description:** Verify that running `init` in a directory with existing files (e.g., a `go.mod`, `main.go`, `README.md`) does not modify any of those files.

**Files to change:**
- `tests/cli_init_test.go` — Add subtest. Seed the temp dir with a few files before running init.

**Acceptance criteria:**
- [ ] Test: pre-existing `go.mod`, `main.go`, and `README.md` are byte-identical before and after init.
- [ ] Test: lifecycle scaffold is created alongside existing files.
- [ ] Test: exit code is 0.

## Milestone 9: Integration Test — Error Cases

**Description:** Verify correct exit codes and error messages for failure modes.

**Files to change:**
- `tests/cli_init_test.go` — Add subtests.

**Acceptance criteria:**
- [ ] Test: `kaos-control init /root/no-permission` (or equivalent unwritable path) exits 1 with a meaningful error message.
- [ ] Test: `kaos-control init --unknown-flag` exits 1 with usage output.

## Milestone 10: Frontend Test — Init Required Banner

**Description:** Test the "project not initialised" banner from [[cli-init-scaffold-4-fe]] renders when the backend reports a missing config.

**Files to change:**
- `tests/cli_init_banner_test.go` (new) or equivalent frontend test file — Verify the banner appears when the project API returns the config-missing error.

**Acceptance criteria:**
- [ ] Test: when the project load API returns a config-missing error, the `InitRequiredBanner` component renders.
- [ ] Test: the banner includes the project path in the suggested init command.
- [ ] Test: normal project views render when the project is properly initialised (no false positives).

## Cross-references

- [[cli-init-scaffold-3-be]] — Backend plan: Milestones 1–3 unit-test the functions defined in that plan's Milestones 2–5. Milestone 7 (config validation test) in the backend plan overlaps with Milestone 2 here — both verify `LoadProject()` compatibility, but the backend plan's test lives in the `initcmd` package while this plan's version is an integration test against the built binary.
- [[cli-init-scaffold-4-fe]] — Frontend plan: Milestone 10 here tests the `InitRequiredBanner` component from that plan's Milestone 1.
