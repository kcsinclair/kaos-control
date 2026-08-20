---
title: "tests/cli_init_test.go and tests/cli_directives_test.go are stale after the AGENTS.md-primary directives refactor"
type: defect
status: approved
lineage: agent-directives-generation
parent: lifecycle/tests/agent-directives-generation-6-test.md
created: "2026-08-20T12:23:00+10:00"
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

# tests/cli_init_test.go and tests/cli_directives_test.go are stale after the AGENTS.md-primary directives refactor

## Reproduction Steps

1. `make test-integration` (runs `go test ./... -count=1 -tags=integration`, which includes `-run` over the top-level `tests` package).
2. Observe failures in `github.com/kaos-control/kaos-control/tests`:
   - `TestInit_FullFlow_EmptyDir`
   - `TestInit_Idempotency`
   - `TestInit_ForceFlags/force-all`
   - `TestInit_NonExistentPath`
   - `TestMigrateDirectivesCmd_LegacyLayout_ProducesAgentsClaudeGemini`
   - `TestMigrateDirectivesCmd_PendingDiff_ExitsNonZeroWithForceHint`

## Expected Behaviour

`tests/cli_init_test.go` and `tests/cli_directives_test.go` assert against the current `init`/`migrate-directives` behaviour: `internal/initcmd/seedfiles.go` writes the devops seed under `lifecycle/devops/sample.yaml` (not a top-level `devops/`), and `CLAUDE.md` is generated as a one-line pointer (`@AGENTS.md`) by `directives.Generate`, with the real content — including the `## Repository Layout`, `## Lineage Filename Convention`, `## Frontmatter Requirements`, `## Commit Conventions`, `## Agent Roles` sections — living in `AGENTS.md` (see `internal/directives/templates/AGENTS.md.tmpl`).

## Actual Behaviour

Both test files still encode the pre-refactor layout:

1. `tests/cli_init_test.go`:
   - `lifecycleDirs` (line 74-88) includes a top-level `"devops"` entry and expects a `devops/.gitkeep`; `seedFiles` (line 93-99) expects `"devops/sample.yaml"`. The actual `init` output only creates `lifecycle/devops/.gitkeep` and writes `lifecycle/devops/sample.yaml` (per `internal/initcmd/seedfiles.go:43` and the scaffold dir list) — there is no top-level `devops/` any more.
   - `claudeMdSections` (line 101-108) asserts those five section headings are literal substrings of **`CLAUDE.md`**. Since `internal/initcmd/seedfiles.go:28-30` — "CLAUDE.md is no longer a static seed... AGENTS.md canonical + CLAUDE.md/GEMINI.md pointers" — `CLAUDE.md` now only contains `@AGENTS.md\n`; the sections exist in `AGENTS.md`, not `CLAUDE.md`.

2. `tests/cli_directives_test.go`, `TestMigrateDirectivesCmd_LegacyLayout_ProducesAgentsClaudeGemini` and `TestMigrateDirectivesCmd_PendingDiff_ExitsNonZeroWithForceHint`:
   - Both call `runInit` to produce a "legacy" `CLAUDE.md` fixture (line 47: `legacyClaudeMd, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))`), then run `migrate-directives` expecting it to convert that legacy file into `AGENTS.md` + pointers. Because `runInit` now always produces the *already-migrated* pointer form (`@AGENTS.md`), `migrate-directives` correctly sees nothing to do and no-ops (`"nothing to do (already migrated, or no CLAUDE.md found)"`), so the tests' migration assertions fail.

## Logs / Output

```
--- FAIL: TestInit_FullFlow_EmptyDir (0.02s)
    cli_init_test.go:133: missing .gitkeep in devops: stat .../devops/.gitkeep: no such file or directory
    cli_init_test.go:181: CLAUDE.md missing section "## Repository Layout"
    cli_init_test.go:181: CLAUDE.md missing section "## Lineage Filename Convention"
    cli_init_test.go:181: CLAUDE.md missing section "## Frontmatter Requirements"
    cli_init_test.go:181: CLAUDE.md missing section "## Commit Conventions"
    cli_init_test.go:181: CLAUDE.md missing section "## Agent Roles"
--- FAIL: TestInit_Idempotency (0.02s)
    cli_init_test.go:212: snapshot read devops/sample.yaml: open .../devops/sample.yaml: no such file or directory
--- FAIL: TestInit_ForceFlags/force-all (0.02s)
    cli_init_test.go:328: planting marker in devops/sample.yaml: open .../devops/sample.yaml: no such file or directory
--- FAIL: TestInit_NonExistentPath (0.02s)
    cli_init_test.go:398: scaffold missing devops/.gitkeep in created directory: stat .../nested/project/devops/.gitkeep: no such file or directory
--- FAIL: TestMigrateDirectivesCmd_LegacyLayout_ProducesAgentsClaudeGemini (0.03s)
    cli_directives_test.go:65: AGENTS.md missing the legacy CLAUDE.md body: ... legacy CLAUDE.md:
        @AGENTS.md
    cli_directives_test.go:82: GEMINI.md should not have been written with no gemini driver configured
    cli_directives_test.go:85: stdout does not report GEMINI.md as skipped
--- FAIL: TestMigrateDirectivesCmd_PendingDiff_ExitsNonZeroWithForceHint (0.03s)
    cli_directives_test.go:102: expected non-zero exit for a pending diff, got 0
```

## Fix guidance

- `cli_init_test.go`: change `lifecycleDirs`/`seedFiles` to `lifecycle/devops` / `lifecycle/devops/sample.yaml` (drop the top-level `devops` entries), and change `claudeMdSections` to read from `AGENTS.md` instead of `CLAUDE.md` (and separately assert `CLAUDE.md` contents equal `@AGENTS.md\n`).
- `cli_directives_test.go`: stop deriving the "legacy" fixture from `runInit`'s output. Write a genuine pre-refactor `CLAUDE.md` (the old full-body content) directly to disk before invoking `migrate-directives`, so the migration path under test is actually exercised.
