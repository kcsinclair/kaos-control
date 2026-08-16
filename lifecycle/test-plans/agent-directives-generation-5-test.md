---
title: "Test Plan — Stack-Tuned Directive Files & Agent Prompts at Init"
type: plan-test
status: done
lineage: agent-directives-generation
parent: lifecycle/requirements/agent-directives-generation-2.md
labels:
    - test
    - architecture
    - onboarding
    - directives
release: KC-Release5
---

# Test Plan — Stack-Tuned Directive Files & Agent Prompts at Init

Verifies the acceptance criteria of [[agent-directives-generation-2]] against
[[agent-directives-generation-3-be]] and [[agent-directives-generation-4-fe]]: `stack_profile`
parsing, AGENTS.md-primary rendering, config patching for the six standard agents, idempotency and
managed-region preservation, driver-based selectivity, and the automated first-run migration +
`migrate-directives` CLI + `init --refresh-directives`.

Integration tests live in `tests/` per convention; package-internal unit tests are owned by the
backend plan and referenced here only where they satisfy a criterion. Recall (project memory):
`testEnv` auto-logins as admin; devops-style URL helpers return full URLs for `http.Get`; write
endpoints re-index synchronously before responding.

Cross-references:
- [[agent-directives-generation-3-be]] — generator, config patch, migration primitives + endpoints under test.
- [[agent-directives-generation-4-fe]] — migration banner / refresh panel behaviours under test.
- [[architecture-templates]] — supplies the tech-stack `stack_profile` fixtures.

---

## Milestone 1 — `stack_profile` parsing (FR-2, FR-5, OQ-5)

### Description

Prove the parser reads the embedded profile from real catalog stacks and models `required: false`.

### Files to change

- **New** `tests/directives_stackprofile_test.go` (or unit in `internal/architecture`):
  - Parse the shipped `go-vue.md` → `run`, three roles, backend `write_paths`/`build`/`lint`/`test`
    match the block.
  - Parse `static-html-js.md` → `backend-developer` not required; frontend build empty.
  - A tech-stack with no `stack_profile:` block → typed "no profile" error (generation must fail
    loudly, not emit Go+Vue defaults — NFR-5).

### Acceptance criteria

- Every shipped tech-stack's `stack_profile` parses without error (table-driven over all 14 files).
- `required: false` roles are modelled as disabled.

---

## Milestone 2 — Directive rendering & idempotency (FR-1, FR-3, FR-4, FR-5, NFR-2)

### Description

Verify AGENTS.md-primary output: canonical AGENTS.md with the standing content, `@AGENTS.md`
pointers, stack-aware layout/commands, and byte-identical re-runs.

### Files to change

- **New** `tests/directives_generate_test.go`:
  - Seed a project with a promoted Go+Vue stack; run generation (via `POST …/directives/refresh`).
  - Assert `AGENTS.md` exists, wrapped in the managed markers, and contains: repo layout
    (`internal/`, `cmd/`, `web/src/`), lineage convention, frontmatter vocab, commit conventions,
    roles, and the required-reading pointer to `lifecycle/architecture/` (FR-4).
  - Assert `CLAUDE.md` and `GEMINI.md` are exactly the `@AGENTS.md` pointer (FR-3, OQ-7).
  - Assert a non-Go+Vue promoted stack yields layout/commands from *that* stack, not Go+Vue (FR-5).
  - Re-run generation → all files byte-identical, report shows zero changes (NFR-2, FR-10).

### Acceptance criteria

- The three files are emitted with the correct shape; only AGENTS.md carries a managed region.
- Generation is deterministic/idempotent; a second run is a no-op.

---

## Milestone 3 — Managed-region preservation & diff-before-overwrite (FR-11, OQ-6)

### Description

Prove refresh replaces only the generated region and never silently clobbers user prose.

### Files to change

- **New** `tests/directives_managed_region_test.go`:
  - After generation, append user prose **outside** the markers in `AGENTS.md`; re-run refresh →
    the generated region updates, the user prose survives verbatim.
  - Edit content **inside** the markers; refresh (no force) → the region is regenerated in place,
    no diff prompt — inside the markers is generated content by design (OQ-6).
  - A file with the markers deleted → refresh returns a whole-file diff and requires force.

### Acceptance criteria

- Prose outside the region is always preserved; content inside the markers is refreshed in place;
  the diff gate fires only when the markers are absent; a create (absent file) never prompts (FR-11).

---

## Milestone 4 — Config patching for the six standard agents (FR-6, FR-7, FR-8, FR-9, OQ-4)

### Description

Verify each standard agent is tuned to the stack, unrelated config is untouched, and the result
loads cleanly.

### Files to change

- **New** `tests/directives_config_patch_test.go`:
  - After generation on a Go+Vue project, load `lifecycle/config.yaml` via `config.LoadProject`:
    `backend-developer` write paths include `internal`, `cmd`, and the constant lifecycle paths;
    build/test tokens are `go build ./...` / `go test ./... -short`; `frontend-developer` maps to
    `web/src` with the pnpm commands.
  - Every analyst/planner/developer prompt contains the architecture-awareness clauses (flag
    architecture-breaking requirements; conform to `standards/`; propose an ADR — FR-8).
  - A `static-html-js` promotion disables `backend-developer` and reports it (OQ-4).
  - Assert `users:`, `kanban:`, `stages:`, and a hand-added custom agent are **byte-unchanged**
    (FR-9); the patched file re-loads via `config.LoadProject` without error.
  - Re-patch with the same selection → no net change (NFR-2).

### Acceptance criteria

- The six agents are correct for the stack; unrelated config is preserved; result parses; idempotent.

---

## Milestone 5 — Driver selectivity & no orphans (FR-12, NFR-3)

### Description

Verify GEMINI.md is skipped when no gemini driver is configured and skips are reported.

### Files to change

- **New** `tests/directives_selectivity_test.go`:
  - Project with no `gemini`/`gemini-cli` driver → generation emits AGENTS.md + CLAUDE.md, **skips**
    GEMINI.md, and names it in the `skipped` report; no orphan file is left.
  - Adding a gemini driver and re-running → GEMINI.md now emitted.

### Acceptance criteria

- Selectivity is driver-driven and defaults to the standard set; skips are reported, never silent
  (NFR-3).

---

## Milestone 6 — First-run migration + CLI + refresh flag (FR-16, FR-14)

### Description

Verify legacy detection, automated migration, idempotency, the `migrate-directives` CLI, the
`init --refresh-directives` flag, and the summary flag that drives the frontend banner.

### Files to change

- **New** `tests/directives_migration_test.go`:
  - Legacy fixture (root `CLAUDE.md`, no `AGENTS.md`): project summary reports
    `directivesMigrationAvailable == true`.
  - `POST /api/projects/{project}/migrate-directives` → `AGENTS.md` (legacy body inside markers),
    `CLAUDE.md` == `@AGENTS.md`, `GEMINI.md` added (driver-gated); summary flag now false.
  - Re-post → idempotent no-op (empty file report).
  - A user-edited `AGENTS.md` present → migrate without force returns a `diff`, writes nothing; with
    force → applied.
  - All written files are retrievable via the artifacts index (re-index fired — FR-15).
- **New** `tests/directives_cli_test.go` (or extend an initcmd test):
  - `kaos-control migrate-directives <path>` on a legacy project produces the same file set as the
    endpoint; a pending diff without `-force` exits non-zero with a hint.
  - `kaos-control init --refresh-directives <path>` on a promoted project regenerates AGENTS.md +
    patches config, identical to the endpoint output for the same selection (FR-14).

### Acceptance criteria

- Migration is automatic-on-offer, idempotent, diff-gated, and reachable from both CLI and API;
  `init --refresh-directives` matches the wizard/endpoint output.

---

## Milestone 7 — Frontend behaviours (FR-16, FR-14 — UI)

### Description

Reference the vitest coverage owned by [[agent-directives-generation-4-fe]] and add the smoke path.

### Files to change

- Referenced (not duplicated): banner shows only when `directivesMigrationAvailable`; migrate flow
  clears the banner; diff response requires explicit overwrite; refresh panel reports the file set
  and `disabledAgents`; the shared `useDirectiveApply` composable drives modal + panel.

### Acceptance criteria

- The frontend vitest suite for the migration banner and refresh panel passes.

---

## Test data / fixtures

- `stack_profile` fixtures: the shipped tech-stack files under
  `lifecycle/architecture/tech-stacks/` (Go+Vue for the happy path, `static-html-js` for the
  `required: false` case, one more for the non-Go+Vue layout assertion), plus a minimal
  no-profile stack for the error case.
- A legacy-layout project fixture (root `CLAUDE.md`, no `AGENTS.md`) for the migration cases.
- A read-only user fixture for the endpoint role-gate assertions (reuse existing auth helpers).

## Verification (end-to-end)

1. `make test-unit` clean (parser/render/patch/migrate units from the backend plan).
2. `make test-integration` clean (all `tests/directives_*_test.go`).
3. `pnpm test` clean (frontend vitest from [[agent-directives-generation-4-fe]]).
4. Full acceptance sweep: every checkbox in [[agent-directives-generation-2]] §Acceptance Criteria
   maps to at least one milestone above and passes.
