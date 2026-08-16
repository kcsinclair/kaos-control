---
title: Agent Directives Generation — Integration Tests
type: test
status: done
lineage: agent-directives-generation
parent: lifecycle/test-plans/agent-directives-generation-5-test.md
assignees:
    - role: product-owner
      who: agent
---

# Agent Directives Generation — Integration Tests

Integration test suite for [[agent-directives-generation]], covering the six HTTP/CLI-facing
milestones of [lifecycle/test-plans/agent-directives-generation-5-test.md](lifecycle/test-plans/agent-directives-generation-5-test.md).
Package-internal unit coverage (parser field-level assertions, `writeFile`/`mergeManaged`,
`PatchAgentConfig`, `Migrate`) already exists under `internal/architecture/` and
`internal/directives/` per the backend plan and is not duplicated here.

Run with: `go test ./tests/integration/... ./tests/... -tags integration -run TestDirectives`
(plus `TestStackProfile` for Milestone 1, which lives alongside the other `directives_*` files).

## Milestone 1 — `stack_profile` parsing

**File:** `tests/integration/directives_stackprofile_test.go`

- `TestStackProfile_AllShippedTechStacks_ParseWithoutError` — table-driven over all 14 shipped
  `lifecycle/architecture/tech-stacks/*.md` catalog files (via the embedded `catalogfs`); each
  parses without error and has at least one role.
- `TestStackProfile_StaticHTMLJS_BackendDeveloperDisabled` — `required: false` is modelled via
  `RoleProfile.IsRequired() == false`.

## Milestone 2 — Directive rendering & idempotency

**File:** `tests/integration/directives_generate_test.go`

- `TestDirectivesGenerate_GoVue_RendersFullSetWithStandingContent` — `POST
  /api/p/{project}/directives/refresh` on a project with a promoted Go+Vue stack emits AGENTS.md
  (managed markers, repo layout, lineage convention, frontmatter requirements, commit conventions,
  roles, and the `lifecycle/architecture/` pointer), and CLAUDE.md/GEMINI.md as bare `@AGENTS.md`
  pointers.
- `TestDirectivesGenerate_NonGoVueStack_UsesThatStacksLayout` — a promoted `python-fastapi` stack
  yields `app/`/`uvicorn`/`pytest` in AGENTS.md, not the Go+Vue layout.
- `TestDirectivesGenerate_Idempotent_SecondRunIsNoOp` — a second refresh with the same selection
  reports every file `skipped` and AGENTS.md is byte-identical.

## Milestone 3 — Managed-region preservation & diff-before-overwrite

**File:** `tests/integration/directives_managed_region_test.go`

- `TestDirectivesManagedRegion_ProseOutsideMarkersSurvivesRefresh` — user prose written outside the
  `<!-- kaos-control:generated:start/end -->` markers survives a refresh that changes the generated
  region.
- `TestDirectivesManagedRegion_MarkersDeleted_RequiresForce` — a whole-file replacement (markers
  removed) is withheld behind a `diff` until `force: true`.
- `TestDirectivesManagedRegion_EditsInsideMarkers_AlwaysRefreshedNoForceNeeded` — see Open Questions
  below; documents the verified actual behaviour (no diff-gate for edits *inside* intact markers).

## Milestone 4 — Config patching for the six standard agents

**File:** `tests/integration/directives_config_patch_test.go`

- `TestDirectivesConfigPatch_GoVue_TunesStandardAgents` — after a refresh, `backend-developer`'s
  `allowed_write_paths`/build token and every standard agent's architecture-awareness clauses
  (reuses `hasReadArchitectureDirective`/`hasProposeADRDirective` from `architecture_directives_test.go`)
  are correct via `config.LoadProject`.
- `TestDirectivesConfigPatch_UnrelatedBlocksUntouched` — `users:`, the custom agent, and its prompt
  are byte-unchanged.
- `TestDirectivesConfigPatch_Idempotent_SecondPatchIsNoOp` — re-patching the same selection leaves
  `lifecycle/config.yaml` byte-identical.
- `TestDirectivesConfigPatch_StaticSite_DisablesBackendDeveloper` — a `static-html-js` promotion
  disables `backend-developer` (`enabled: false`), reports it in `disabledAgents`, and the file
  still reloads via `config.LoadProject`.

## Milestone 5 — Driver selectivity & no orphans

**File:** `tests/integration/directives_selectivity_test.go`

- `TestDirectivesSelectivity_NoGeminiDriver_SkipsGeminiMd` — no gemini/gemini-cli driver configured
  → GEMINI.md is skipped, reported in `skipped`, and never written.
- `TestDirectivesSelectivity_AddingGeminiDriver_EmitsGeminiMdOnRerun` — adding a gemini-cli driver
  and re-running now emits GEMINI.md.

## Milestone 6 — First-run migration + CLI + refresh flag

**Files:** `tests/integration/directives_migration_test.go` (HTTP), `tests/cli_directives_test.go` (CLI)

- `TestDirectivesMigrate_LegacyLayout_SummaryReportsAvailable` /
  `TestDirectivesMigrate_NoLegacyClaudeMd_SummaryReportsUnavailable` — `GET /api/projects/{project}`
  surfaces `directivesMigrationAvailable`.
- `TestDirectivesMigrate_LegacyLayout_ProducesAgentsClaudeGemini` — `POST
  /api/projects/{project}/migrate-directives` wraps the legacy CLAUDE.md body into AGENTS.md,
  rewrites CLAUDE.md as the pointer, adds GEMINI.md, and clears the summary flag.
- `TestDirectivesMigrate_RepostAfterMigration_IsIdempotentNoOp` — a second migrate call is a true
  no-op (empty file report).
- `TestDirectivesMigrate_UserEditedAgentsMd_RequiresForce` — a pre-existing, differing AGENTS.md is
  diff-gated; `force: true` applies it.
- `TestDirectivesMigrate_ReadOnlyUser_Returns403` — the project-admin role gate.
- `TestDirectivesMigrate_WrittenFilesNotInArtifactsIndex` — see Open Questions below.
- `TestMigrateDirectivesCmd_LegacyLayout_ProducesAgentsClaudeGemini` /
  `TestMigrateDirectivesCmd_PendingDiff_ExitsNonZeroWithForceHint` — `kaos-control
  migrate-directives [-force]` matches the endpoint's file set and behaviour, and a pending diff
  exits non-zero with a `-force` hint.
- `TestInitRefreshDirectivesCmd_PromotedProject_RegeneratesAgentsAndPatchesConfig` — `kaos-control
  init --refresh-directives [-force]` on a promoted project matches the refresh endpoint's output
  (layout-correct AGENTS.md, patched `lifecycle/config.yaml`), and a second run is a no-op.

## Milestone 7 — Frontend behaviours

Not duplicated here — owned by [[agent-directives-generation-4-fe]]'s vitest suite (migration
banner, refresh panel, `useDirectiveApply` composable).

## Resolved Questions

Both confirmed against the code; shipped behaviour is correct — the requirement/test-plan wording
was revised to match (as FR-1/FR-3 were for the AGENTS.md-primary decision). No code changes.

1. **M3 — edits inside intact markers.** `writeFile` regenerates the managed region in place on
   every refresh (no diff-gate) — correct per OQ-6. Diff-gating applies only when the markers are
   absent. FR-11 and the test-plan M3 wording updated accordingly.

2. **M6 — FR-15 indexing.** The root directive files (`AGENTS.md`/`CLAUDE.md`/`GEMINI.md`) are not
   lifecycle artefacts and are intentionally outside the artefact index/watcher (`lifecycle/**/*.md`
   only) — exactly as `CLAUDE.md` has always been. FR-15 revised to say so; only the
   `lifecycle/config.yaml` patch is picked up (via `ReloadConfig`).
   `TestDirectivesMigrate_WrittenFilesNotInArtifactsIndex` guards this as intended behaviour.
