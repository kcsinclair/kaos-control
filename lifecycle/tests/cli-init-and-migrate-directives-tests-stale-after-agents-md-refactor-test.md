---
title: tests/cli_init_test.go and tests/cli_directives_test.go — updated for the AGENTS.md-primary layout
type: test
status: draft
lineage: agent-directives-generation
parent: lifecycle/defects/cli-init-and-migrate-directives-tests-stale-after-agents-md-refactor.md
created: "2026-08-20T13:10:00+10:00"
assignees:
    - role: product-owner
      who: agent
---

# tests/cli_init_test.go and tests/cli_directives_test.go — updated for the AGENTS.md-primary layout

Fixes the six failing integration tests reported in
[lifecycle/defects/cli-init-and-migrate-directives-tests-stale-after-agents-md-refactor.md](lifecycle/defects/cli-init-and-migrate-directives-tests-stale-after-agents-md-refactor.md).
Both files still assumed the pre-refactor layout (top-level `devops/`, a
static `CLAUDE.md` seed file); `internal/initcmd`/`internal/directives` now
scaffold `lifecycle/devops/`, `lifecycle/architecture/`, and generate the
AGENTS.md-primary directive set (`AGENTS.md` canonical, `CLAUDE.md`/
`GEMINI.md` as `@AGENTS.md` pointers) via `directives.Generate`.

## `tests/cli_init_test.go`

- `lifecycleDirs` now matches `internal/initcmd/scaffold.go:lifecycleDirs`
  exactly: dropped the stray top-level `devops` entry, added the
  `lifecycle/architecture` entry that scaffold.go already creates (this was
  stale beyond what the defect described — the entry was silently untested).
- `seedFiles` now matches `internal/initcmd/seedfiles.go:seedFileSpecs`:
  dropped `CLAUDE.md` (no longer a static seed) and fixed
  `devops/sample.yaml` → `lifecycle/devops/sample.yaml`.
- Added a `directiveFiles` list (`AGENTS.md`, `CLAUDE.md`, `GEMINI.md`) for
  the files `directives.Generate` writes, checked separately from
  `seedFiles` since they report and skip differently (no
  `skipped: ... (already exists; use --force to overwrite)` stderr line —
  that message is specific to `writeSeedFiles`).
- `TestInit_FullFlow_EmptyDir` — asserts `AGENTS.md` (not `CLAUDE.md`)
  contains the five required sections, and separately asserts `CLAUDE.md`
  is byte-identical to `@AGENTS.md\n`; asserts all three `directiveFiles`
  exist and are reported as `created`.
- `TestInit_Idempotency` — snapshots and re-verifies byte-identical content
  for `directiveFiles` in addition to `seedFiles` across a second,
  no-op run (the directive files refresh surgically via managed-region
  markers, which must be a no-op with nothing changed).
- `TestInit_ForceFlags/force-all` — plants and verifies overwrite of the
  `directiveFiles` marker content alongside `seedFiles`, since `--force`
  implies `ForceFlags.ClaudeMd`, which now covers `AGENTS.md`/`CLAUDE.md`/
  `GEMINI.md` instead of a plain seed file.

## `tests/cli_directives_test.go`

- Added `legacyClaudeMdFixture`: a genuine pre-refactor `CLAUDE.md` body
  (the last rendered form of the removed `CLAUDE.md.tmpl`, before commit
  `0a6956c2`), written directly to disk instead of derived from `runInit`'s
  output. `runInit` now always produces the already-migrated pointer form,
  so deriving the "legacy" fixture from it meant `migrate-directives` saw
  nothing to migrate and correctly no-opped — the defect's root cause.
- `TestMigrateDirectivesCmd_LegacyLayout_ProducesAgentsClaudeGemini` — after
  `runInit`, removes the `AGENTS.md` and `GEMINI.md` that init now
  establishes and overwrites `CLAUDE.md` with `legacyClaudeMdFixture`,
  rewinding the project to the genuine legacy state before invoking
  `migrate-directives`. Assertions unchanged: `AGENTS.md` wraps the legacy
  body in managed-region markers, `CLAUDE.md` becomes a bare pointer,
  `GEMINI.md` is skipped and reported (no gemini driver configured).
- `TestMigrateDirectivesCmd_PendingDiff_ExitsNonZeroWithForceHint` — same
  legacy rewind, then a hand-written `AGENTS.md` (no managed markers,
  differing from the migrated content) triggers the pending-diff path:
  `migrate-directives` exits non-zero with a `-force` hint and leaves
  `AGENTS.md` untouched; `-force` then replaces it.
- `TestInitRefreshDirectivesCmd_PromotedProject_RegeneratesAgentsAndPatchesConfig`
  was already passing against the current behaviour and is unchanged.

## Verification

```
go test ./tests/... -tags=integration -run \
  'TestInit_FullFlow_EmptyDir|TestInit_Idempotency|TestInit_ForceFlags|TestInit_NonExistentPath|TestMigrateDirectivesCmd_LegacyLayout_ProducesAgentsClaudeGemini|TestMigrateDirectivesCmd_PendingDiff_ExitsNonZeroWithForceHint|TestInitRefreshDirectivesCmd_PromotedProject_RegeneratesAgentsAndPatchesConfig' \
  -v -count=1
```

All seven tests pass. Full `go test ./... -tags=integration -count=1` run
pending in this session to confirm no other regressions.
