---
created: "2026-08-16T19:05:37+10:00"
title: "Backend Plan — Stack-Tuned Directive Files & Agent Prompts at Init"
type: plan-backend
status: done
lineage: agent-directives-generation
parent: lifecycle/requirements/agent-directives-generation-2.md
labels:
    - backend
    - architecture
    - onboarding
    - directives
release: KC-Release5
---

# Backend Plan — Stack-Tuned Directive Files & Agent Prompts at Init

This plan implements the server-side generator for [[agent-directives-generation-2]]: parse the
promoted stack's `stack_profile`, render the **AGENTS.md-primary** directive set (AGENTS.md
canonical + CLAUDE.md/GEMINI.md pointers), patch the six standard agent prompt templates in
`lifecycle/config.yaml` to the chosen stack, and add the automated **first-run migration** with a
`kaos-control migrate-directives` CLI and an `init --refresh-directives` refresh path. Generation
is snapshot-at-promotion, idempotent, and preserves user prose via managed-region markers.

**Scope boundary.** This lineage owns the *directive/prompt generation mechanism* and the
*migration*. It does **not** own: the Architecture Wizard UX that invokes it as its scaffolding
step ([[onboarding-architecture-selection]] FR-17 — the *caller*); the catalog/promotion/ADR
artefact model ([[architectural-artefacts-2]], whose `internal/architecture` primitives this plan
consumes); the catalog seed content ([[architecture-templates]]); or the migration-offer UI and
refresh affordance ([[agent-directives-generation-4-fe]]). Concrete directive *prose* is a
one-time embedded template — this plan specifies the generation and what each file must contain.

Today's baseline (verified): `internal/initcmd` renders a **static** `CLAUDE.md.tmpl` +
`config.yaml.tmpl` via `text/template` (`embed.go:17`, `seedfiles.go:22`); there is **no**
AGENTS.md/GEMINI.md writer, **no** `stack_profile` parser, and **no** first-run/migration
detection anywhere in Go. All greenfield.

Cross-references:
- [[agent-directives-generation-4-fe]] — Frontend plan (migration-offer banner, refresh action, wizard opt-in panel).
- [[agent-directives-generation-5-test]] — Test plan.
- [[onboarding-architecture-selection]] — wizard; calls the generator as its FR-17 scaffolding step.
- [[architectural-artefacts-2]] — promotion/ADR primitives + `internal/architecture` package this plan reads from.
- [[architecture-templates]] — ships the `stack_profile:` blocks in the tech-stack catalog.
- [[agents-md-primary-directives]] — the AGENTS.md-primary convention this plan realises.

---

## Milestone 1 — Parse `stack_profile` from the promoted tech-stack

### Description

Add a parser for the `stack_profile:` YAML block embedded at the end of each `tech-stack`
markdown (present on all 14 shipped stacks, e.g.
[lifecycle/architecture/tech-stacks/go-vue.md](lifecycle/architecture/tech-stacks/go-vue.md)).
It is the single input for both directive content (FR-2/FR-5) and config patching
(FR-6/FR-7). Lives in `internal/architecture` beside `catalog.go`, which already loads
tech-stack frontmatter but ignores this block (confirmed: no `stack_profile` reader exists).

### Files to change

- **New** `internal/architecture/stackprofile.go`:
  - Types mirroring the embedded schema:
    ```go
    type RoleProfile struct {
        Required   *bool    `yaml:"required"`   // nil == true; false disables the agent
        WritePaths []string `yaml:"write_paths"`
        Build      string   `yaml:"build"`
        Lint       string   `yaml:"lint"`
        Test       string   `yaml:"test"`
        Note       string   `yaml:"note,omitempty"`
    }
    type RepoLayoutEntry struct { Path string `yaml:"path"`; Note string `yaml:"note"` }
    type StackProfile struct {
        Run        string                  `yaml:"run"`
        RepoLayout []RepoLayoutEntry       `yaml:"repo_layout"`
        Roles      map[string]RoleProfile  `yaml:"roles"`
    }
    ```
  - `func ParseStackProfile(mdBytes []byte) (StackProfile, error)` — extract the fenced
    ```` ```yaml ```` block whose top-level key is `stack_profile:`, unmarshal its
    `stack_profile` value. Tolerate inline `# comments` (yaml.v3 handles them). Return a typed
    error if no block is found (a stack without a profile is a hard error for generation — NFR-5).
  - `func LoadPromotedStackProfile(projectRoot string) (StackProfile, string, error)` — locate the
    **promoted** tech-stack root copy (root-level `type: tech-stack` file, via the same scan
    `internal/architecture` already uses for promoted copies), read + `ParseStackProfile` it, and
    return the profile plus the stack's title. Errors clearly if nothing is promoted yet
    (generation is gated on promotion — FR-2).
  - `func (r RoleProfile) IsRequired() bool { return r.Required == nil || *r.Required }`.

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Unit test `internal/architecture/stackprofile_test.go`:
  - Parsing the real `go-vue.md` bytes yields `Run == "go run ./cmd/<app>"`, three roles, and
    `backend-developer.WritePaths == ["internal","cmd"]`, `.Build == "go build ./..."`.
  - Parsing `static-html-js.md` yields `backend-developer.IsRequired() == false` and a
    frontend-developer with empty `Build`.
  - A markdown with no `stack_profile:` block returns the typed "no profile" error.

---

## Milestone 2 — Shared content model + directive rendering (AGENTS.md-primary)

### Description

Render the directive set from **one shared content model** (FR-1) into AGENTS.md (canonical),
CLAUDE.md (`@AGENTS.md` import), and GEMINI.md (`@AGENTS.md` import — OQ-7: gemini-cli follows the
`@AGENTS.md` directive, so a pointer, not a full copy). The generated body carries a
**managed-region marker** so a later refresh replaces only that region and preserves user prose
(OQ-6). New package `internal/directives`.

### Files to change

- **New** `internal/directives/model.go`:
  - `type DirectiveModel struct { ProjectName, Language string; Stack architecture.StackProfile; StackTitle string; RepoLayout []architecture.RepoLayoutEntry; ArchitecturePointer bool }`.
  - `func BuildModel(projectRoot string) (DirectiveModel, error)` — pulls project name from config,
    calls `architecture.LoadPromotedStackProfile` (M1). If no stack is promoted, returns a sentinel
    error so callers can fall back to the generic (pre-wizard) directive.
- **New** `internal/directives/render.go`:
  - `//go:embed templates/AGENTS.md.tmpl` — the canonical body template. It contains the standing
    content (FR-4): repo layout (from `Stack.RepoLayout`), artifact/lineage convention, frontmatter
    vocab, commit conventions, roles, and the **required-reading pointer to
    `lifecycle/architecture/`** (summary, promoted architecture+stack, `decisions/`, `standards/`).
    Any stack-specific line (layout, build/test) is interpolated from the model, never hard-coded
    to Go+Vue (FR-5).
  - Managed-region constants: `const genStart = "<!-- kaos-control:generated:start -->"`,
    `genEnd = "<!-- kaos-control:generated:end -->"`. The rendered body is wrapped in these.
  - `func RenderAgents(m DirectiveModel) ([]byte, error)` — execute the template, wrap in markers.
  - `func RenderPointer(canonical string) []byte` — returns a `@AGENTS.md`\n pointer file body
    (used for both CLAUDE.md and GEMINI.md). Keep it a literal one-liner so the pointer files carry
    no managed region (they never drift).
- **New** `internal/directives/write.go`:
  - `type FileWrite struct { Path string; Created, Changed, Skipped bool; Diff string }`.
  - `func mergeManaged(existing, freshBody []byte) ([]byte, bool, error)` — if `existing` contains
    the marker pair, replace only the region between markers (preserving user prose above/below);
    else treat the whole file as generated. Returns merged bytes + changed flag.
  - `func writeFile(path string, fresh []byte, force bool) (FileWrite, error)` — reads existing,
    `mergeManaged`, and: creates if absent (no diff prompt, FR-11); if the region changed and the
    file was user-edited, returns a `Diff` and **does not write** unless `force` (the UI/CLI
    confirms — FR-11); atomic temp-file + `os.Rename`.

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Unit test `internal/directives/render_test.go`:
  - `RenderAgents` for a Go+Vue model contains `internal/`, `cmd/`, `web/src/` and the
    `lifecycle/architecture/` required-reading pointer, wrapped in the marker pair.
  - Same model rendered twice → **byte-identical** output (NFR-2).
  - `mergeManaged` preserves prose written outside the markers and replaces only the inner region.
  - `RenderPointer` output is exactly the `@AGENTS.md` import line.

---

## Milestone 3 — Patch the six standard agent prompt templates in `config.yaml`

### Description

Write/update the standard agents in `lifecycle/config.yaml` for the chosen stack (FR-6/FR-7/FR-8):
each developer role's `allowed_write_paths` and build/lint/test commands come from the stack
profile; every analyst/planner/developer prompt carries the architecture-awareness clauses; a role
marked `required: false` for the stack is disabled. Edits are **scoped** — only the six standard
agents (`requirements-analyst`, `planning-analyst`, `backend-developer`, `frontend-developer`,
`test-developer`, `qa`); roles, stages, users, kanban, dashboard, and user-added agents are left
untouched (FR-9, OQ-4). `config.ValidateAndRepair` already programmatically self-repairs
generation agents (`config.go:781`), establishing precedent for config mutation.

### Files to change

- **New** `internal/directives/config_patch.go`:
  - `func PatchAgentConfig(projectRoot string, m DirectiveModel) (changed bool, err error)`:
    - Read `lifecycle/config.yaml` into a `yaml.Node` (document node) to **preserve comments and
      unrelated keys** — do not round-trip through the typed struct (that would drop comments and
      re-emit defaults).
    - Locate the `agents:` sequence; for each of the six standard agents (match by `name`), map its
      role → `m.Stack.Roles[role]`:
      - Set `allowed_write_paths` = stack `write_paths` **plus** the constant lifecycle paths
        (`lifecycle/<stage>-plans`, `lifecycle/architecture/decisions`) merged per role (dedup).
      - Set the role's `prompt_templates` build/test tokens from `Build`/`Lint`/`Test` (replace only
        the managed sub-block delimited by markers inside the block scalar, mirroring M2 — so
        user-added prose in a template survives).
      - Ensure the architecture-awareness clauses are present (analysts flag architecture-breaking
        requirements; planners/developers conform to `standards/` and propose an ADR rather than
        deviate — FR-8).
      - If `RoleProfile.IsRequired() == false`, mark the agent disabled (set an `enabled: false`
        field / drop from active set — verify the existing disable mechanism used by
        `ValidateAndRepair`) and record it for the caller's skip report (OQ-4).
    - Write back only if changed; atomic write; re-load via `config.LoadProject` to assert the
      result parses (FR-9) before returning success (on parse failure, do not leave a broken file —
      write to temp, validate, rename).
  - Keep the marker convention identical to M2 so refreshes are surgical.

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Unit test `internal/directives/config_patch_test.go`:
  - Patching a Go+Vue fixture sets `backend-developer` write paths to include `internal`, `cmd`,
    and the constant lifecycle paths, and its build token to `go build ./...`.
  - A `static-html-js` profile disables `backend-developer` and reports it in the skip list; a
    hand-added custom agent and the `users:`/`kanban:` blocks are byte-unchanged.
  - The patched file re-loads via `config.LoadProject` without error (FR-9).
  - Re-patching with the same model is a no-op (`changed == false`, NFR-2).

---

## Milestone 4 — Generation orchestrator + selectivity

### Description

Tie M1–M3 into one deterministic, idempotent operation (FR-10/FR-14) with driver-based selectivity
(FR-12): skip `GEMINI.md` when no `gemini`/`gemini-cli` driver is configured; always emit AGENTS.md
+ CLAUDE.md. Report created/changed/skipped files and disabled agents (NFR-3).

### Files to change

- **New** `internal/directives/generate.go`:
  - `type GenerateResult struct { Files []FileWrite; DisabledAgents []string; Skipped []string }`.
  - `func Generate(projectRoot string, opts GenerateOptions) (GenerateResult, error)`:
    - `GenerateOptions{ Force bool; Drivers []string }` — `Drivers` defaults to those found in
      config (`config.LoadProject().Agents[].Driver`).
    - `BuildModel` → `RenderAgents` → write AGENTS.md; `RenderPointer` → CLAUDE.md; write GEMINI.md
      only if a gemini driver is present (else record in `Skipped` — FR-12); `PatchAgentConfig`.
    - Deterministic ordering; no clock/network (NFR-1/NFR-2). After writing, synchronously
      re-index the written markdown so the graph reflects them (FR-15) — reuse the project's
      re-index entry point, or leave to the caller if `Generate` has no project handle (document
      which).
  - `func GenericAgents(projectName, language string) ([]byte, error)` — the pre-wizard fallback:
    AGENTS.md with standing content but a generic/placeholder repo-layout section, for projects
    that run migration before promoting a stack.

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Unit test: `Generate` on a promoted Go+Vue fixture writes AGENTS.md (with markers), CLAUDE.md
  (`@AGENTS.md`), GEMINI.md (gemini driver present), and patches config — all reported in
  `GenerateResult`; a second run reports zero changes (idempotent).
- With no gemini driver, GEMINI.md is skipped and named in `Skipped`.

---

## Milestone 5 — First-run migration + `migrate-directives` CLI + `init --refresh-directives`

### Description

Detect the legacy single-`CLAUDE.md` layout and offer automated migration (FR-16); expose it as a
CLI subcommand and expose refresh via an init flag (FR-14). Migration renames `CLAUDE.md` →
`AGENTS.md`, rewrites `CLAUDE.md` as `@AGENTS.md`, and adds `GEMINI.md`; idempotent on an
already-migrated project; diff-before-overwrite on a user-edited `AGENTS.md`.

### Files to change

- **New** `internal/directives/migrate.go`:
  - `func NeedsMigration(projectRoot string) (bool, error)` — true when a root `CLAUDE.md` exists,
    no `AGENTS.md` exists, and `CLAUDE.md` is not already a bare `@AGENTS.md` pointer.
  - `func Migrate(projectRoot string, opts MigrateOptions) (GenerateResult, error)` —
    `MigrateOptions{ Force bool }`. Rename legacy `CLAUDE.md` → `AGENTS.md` (wrapping its body in
    managed markers so future refreshes are surgical), write the `@AGENTS.md` CLAUDE.md pointer,
    add GEMINI.md (driver-gated as in M4). Idempotent: on an already-migrated layout, a no-op
    returning empty `Files`. If `AGENTS.md` already exists and differs, return a `Diff` and require
    `Force` (FR-16).
- **New** `internal/migratecmd/migratecmd.go`:
  - `func Run(args []string) error` — `kaos-control migrate-directives [-force] [path]`; prints the
    file report; on a pending diff without `-force`, prints the diff and exits non-zero with a hint.
- **Edit** `cmd/kaos-control` dispatch (where `init`, `backfill-created` subcommands are routed):
  register `migrate-directives`.
- **Edit** `internal/initcmd/initcmd.go`:
  - Add a `-refresh-directives` flag (`initcmd.go:128` flag block). When set, skip the full
    scaffold and instead call `directives.Generate(root, {Force: force})`, printing the report
    (FR-14). Same output as the wizard's scaffolding step for the same selection.
- **Edit** `internal/http/projects.go`:
  - Extend the project summary with `directivesMigrationAvailable bool` (compute via
    `directives.NeedsMigration(e.Path)` alongside the existing `Initialised` at `projects.go:36`)
    so the frontend banner ([[agent-directives-generation-4-fe]]) can surface the offer.
  - Add `POST /api/projects/{project}/migrate-directives` → `directives.Migrate` (force flag in
    body) and `POST /api/p/{project}/directives/refresh` → `directives.Generate`, both returning the
    `GenerateResult`; role-gate as project-admin. Mount in `internal/http/server.go` near the init
    route (`server.go:173`).
  - **Git tracking (FR-17):** the root directive files are outside the index/watcher (FR-15) and
    generation never touches git, so both handlers populate `res.GitCommands` via a
    `directiveGitCommands(projectPath, res.Files)` helper — when the project is a git repo, the
    `git add …`/`git commit …` for every Created/Changed (non-`Diff`, non-`Skipped`) file, mirroring
    `handleInitProject`'s already-initialised-repo branch. Add a `GitCommands []string` field to
    `GenerateResult`. Auto-commit is deliberately not done (existing repo may hold staged user work).

### Acceptance criteria

- `go build ./... && go vet ./...` clean; `kaos-control migrate-directives --help` works.
- Unit test `internal/directives/migrate_test.go`:
  - Legacy fixture (root `CLAUDE.md`, no `AGENTS.md`) → `NeedsMigration == true`; `Migrate`
    produces `AGENTS.md` (legacy body inside markers) + `CLAUDE.md` == `@AGENTS.md` + `GEMINI.md`.
  - Re-running `Migrate` on the result → `NeedsMigration == false`, no-op.
  - A user-edited `AGENTS.md` present → `Migrate` returns a diff and writes nothing without
    `Force`.
- Integration (in [[agent-directives-generation-5-test]]): the project summary reports
  `directivesMigrationAvailable`, and the migrate/refresh endpoints produce the expected files and
  re-index them.

---

## Risk notes

- **YAML comment preservation** — `PatchAgentConfig` uses a `yaml.Node` round-trip, not the typed
  struct, specifically to keep comments and unrelated keys; the M3 "unrelated blocks byte-unchanged"
  test guards regressions. Block-scalar prompt edits are confined to marker-delimited sub-regions.
- **Managed-region tampering** — if a user deletes the markers, refresh falls back to whole-file
  replacement behind a diff prompt (never a silent clobber). Documented in the generated file's
  header comment.
- **No promoted stack** — generation/migration must not hard-fail a pre-wizard project; the
  `GenericAgents` fallback (M4) covers migration before promotion, and `BuildModel` returns a
  sentinel the orchestrator handles.
- **Migration atomicity** — rename + two writes are not transactional; on a mid-operation failure
  the idempotent `NeedsMigration`/`Migrate` re-run converges. Single-user semantics accepted for v1.

## Verification (end-to-end)

1. `make lint` clean.
2. `make test-unit` clean (new `internal/architecture/stackprofile` + `internal/directives` units).
3. `make test-integration` clean ([[agent-directives-generation-5-test]]).
4. Manual smoke via `make run`: on a legacy project, the summary reports
   `directivesMigrationAvailable`; `POST …/migrate-directives` yields AGENTS.md + `@AGENTS.md`
   CLAUDE.md + GEMINI.md. After promoting a stack, `kaos-control init --refresh-directives`
   regenerates AGENTS.md with the stack's layout/commands and patches the six agents; re-running is
   a no-op; all files appear in the index without special-casing.
