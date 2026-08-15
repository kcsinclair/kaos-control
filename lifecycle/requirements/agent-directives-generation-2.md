---
title: Agent Directives Generation — Stack-Tuned Directive Files & Agent Prompts at Init
type: requirement
status: blocked
lineage: agent-directives-generation
parent: lifecycle/ideas/agent-directives-generation.md
labels:
    - architecture
    - onboarding
    - backend
    - feature
release: KC-Release5
assignees:
    - role: product-owner
      who: agent
---

# Agent Directives Generation — Stack-Tuned Directive Files & Agent Prompts at Init

## Problem

Today `kaos-control init` emits a single, generic `CLAUDE.md`
([[cli-init-scaffold]]) and a `lifecycle/config.yaml` whose agent prompt
templates and `allowed_write_paths` are hard-coded for this repo's Go+Vue
layout. Two gaps follow:

1. **Only one CLI is served.** Projects that drive agents with Codex
   (`AGENTS.md`), Gemini (`GEMINI.md`), Antigravity, or a future driver get no
   directive file, so those agents run without the repo layout, artifact/lineage
   conventions, frontmatter rules, commit conventions, roles, or the pointer to
   `lifecycle/architecture/` that Claude Code receives.
2. **The workforce is not wired to the chosen architecture.** After the
   Architecture Wizard ([[onboarding-architecture-selection]]) promotes an
   architecture + stack, the seeded agent prompts still assume Go+Vue: wrong
   `allowed_write_paths`, wrong build/test commands, and no architecture-awareness
   clauses (flag architecture-breaking requirements; conform to recorded
   architecture/standards; propose an ADR rather than deviate silently).

When directive files are hand-maintained per CLI they also **drift** from each
other and from the config. This requirement defines generation of the full,
stack-aware directive set — all per-CLI files plus the agent prompt templates —
from **one shared source of truth**, driven by the wizard's selection and
re-runnable on demand.

## Goals / Non-goals

### Goals

- Generate a **set of per-CLI directive files** at the project root from one
  shared content model: `CLAUDE.md`, `AGENTS.md`, `GEMINI.md`, and Antigravity's
  directive file — so they never drift from each other.
- Make directive content **stack-aware**: repo layout, and any embedded
  build/test commands, reflect the chosen architecture + tech stack, not a fixed
  Go+Vue assumption.
- Include in every directive file the standing content: repo layout,
  artifact/lineage conventions, frontmatter requirements, commit conventions,
  roles, and a **pointer to `lifecycle/architecture/`** (promoted architecture and
  stack, `architecture-summary.md`, `decisions/` ADRs, `standards/`) as required
  reading before designing or building.
- **Generate/patch the agent prompt templates** in `lifecycle/config.yaml` so each
  agent is correct for the chosen stack: right `allowed_write_paths`, right
  build/test commands per developer role, and the architecture-awareness clauses.
- Make generation **idempotent and re-runnable** (re-running the wizard, or a
  `kaos-control init --refresh-directives`), regenerating from the current
  selection and **showing a diff before overwriting** user-edited files.
- Allow a project to **skip** directive files it does not need (no Gemini driver →
  no `GEMINI.md`), defaulting to the standard set.
- Follow the wizard convention for any naming/location choice: ask, and offer a
  **"decide for me"** default.

### Non-goals

- The Architecture Wizard UX and the opt-in point that invokes this step — owned
  by [[onboarding-architecture-selection]] (FR-17); this requirement is the
  *callee*.
- The catalog contents and the **per-stack profile data** (repo layout, build/test
  commands, write-path conventions per architecture+stack) this generator consumes
  — owned by [[architecture-templates]].
- The on-disk architecture artefact model (promotion, ADR numbering, summary,
  standards seeding) — owned by [[architectural-artefacts]].
- The base directory scaffold and seed `config.yaml`/`.claude/settings.json` — the
  v1 behaviour owned by [[cli-init-scaffold]]; this requirement extends the
  directive/prompt portion of it, it does not re-specify the scaffold.
- Actually **running** agents or validating that a generated directive produces
  good agent behaviour.
- Authoring the substantive prose of any single directive template (that is
  content, embedded once); this requirement specifies the generation mechanism and
  what each file must contain.

## Detailed Requirements

### Functional — single source of truth

- **FR-1** All per-CLI directive files are rendered from **one shared content
  model** (a single template plus per-CLI wrappers), not authored independently.
  Regenerating produces the same body in every file, adapted only by each CLI's
  required filename/location and any per-CLI framing.
- **FR-2** The shared content model is populated from the project's **selection
  inputs**: the promoted architecture + tech stack (per [[architectural-artefacts]])
  and the per-stack profile (per [[architecture-templates]]). Given the same
  inputs, generation is deterministic (NFR-2).

### Functional — directive files

- **FR-3** Generation emits the following directive files at the project root,
  each at the location its CLI expects:

  | File | CLI / driver | Location |
  |---|---|---|
  | `CLAUDE.md` | Claude Code | project root |
  | `AGENTS.md` | Codex | project root |
  | `GEMINI.md` | Gemini | project root |
  | *(Antigravity directive file)* | Antigravity | its documented location *(see OQ-1)* |

  The set is **extensible**: adding a new driver adds a wrapper + target location
  without changing the shared body.
- **FR-4** Each directive file contains, at minimum: (a) the repository layout for
  the chosen stack; (b) the artifact/lineage filename convention (slug, monotonic
  per-lineage index, suffix rules, architecture exception); (c) frontmatter
  requirements (required fields, type vocabulary, status vocabulary); (d) commit
  conventions; (e) the roles and their scope of writes; and (f) a **required-reading
  pointer to `lifecycle/architecture/`** — the summary, promoted architecture and
  stack, `decisions/` ADRs, and `standards/` — to be read before any design,
  planning, or implementation work.
- **FR-5** Any stack-specific detail in a directive file (repo layout, and any
  build/test commands mentioned) is taken from the chosen stack, never hard-coded
  to Go+Vue.

### Functional — agent prompt templates

- **FR-6** Generation writes/updates the **agent prompt templates and
  `allowed_write_paths`** in `lifecycle/config.yaml` so each of the standard
  developer agents is correct for the chosen stack — e.g. Go+Vue → backend writes
  `internal/` + `cmd/`, frontend writes `web/src/`; a Python+React stack maps to
  that stack's source roots instead.
- **FR-7** Each developer agent's prompt template carries the **correct build/test
  commands** for the chosen stack (e.g. `go build ./...` / `go vet ./...` vs
  `pytest` / `ruff` vs `pnpm build` / `pnpm exec vue-tsc --noEmit`), consistent
  with that agent's driver.
- **FR-8** Every generated analyst/planner/developer prompt includes the
  **architecture-awareness clauses**: analysts flag **architecture-breaking
  requirements** against the chosen architecture/stack; planners and developers
  **conform** to the recorded architecture and `standards/` and **propose an ADR**
  in `lifecycle/architecture/decisions/` rather than deviating silently.
- **FR-9** The emitted `lifecycle/config.yaml` parses without error via
  kaos-control's existing `config.LoadProject()`, and updates to it are **scoped**:
  generation touches the standard agents' prompt templates and write paths and
  leaves unrelated config (roles, stages, users, kanban, dashboard, user-added
  agents) unchanged *(see OQ-4)*.

### Functional — idempotency, diff, and selectivity

- **FR-10** Generation is **idempotent and re-runnable**: re-running the wizard or
  `kaos-control init --refresh-directives` regenerates every managed file/section
  from the current selection; a second run with the same selection produces no
  net change.
- **FR-11** Before overwriting any file that has been **edited since it was last
  generated**, the process shows a **diff** and requires confirmation; it never
  silently overwrites user edits. Files that do not yet exist are created without a
  diff prompt. *(The mechanism for distinguishing generated vs user-edited content
  — full-file diff vs managed-region markers — is OQ-3.)*
- **FR-12** A project may **skip** directive files for CLIs it does not use;
  selection defaults to the standard set and is driven by which drivers the project
  has configured *(see OQ-2)*. Skipping a file is not an error and is reported.
- **FR-13** Where generation needs a naming or location choice (e.g. an ambiguous
  Antigravity path), it **prompts the user and offers a "decide for me" default**,
  consistent with the wizard convention.

### Functional — integration

- **FR-14** Generation is invokable **both** as the opt-in scaffolding step handed
  off from the Architecture Wizard ([[onboarding-architecture-selection]] FR-17)
  **and** standalone via `kaos-control init --refresh-directives` on an existing
  project, producing identical output for the same selection.
- **FR-15** Files written/updated by generation are plain markdown / YAML picked up
  by the existing indexing paths (startup scan, fsnotify watch, API writes) with no
  special-casing.

### Non-functional

- **NFR-1** Generation works **entirely offline**; all templates and per-stack
  profiles are embedded in the binary (or read from the shipped catalog), with no
  network access at runtime.
- **NFR-2** Generation is **deterministic**: the same selection (architecture +
  stack + configured drivers) yields byte-identical directive files and the same
  config edits on every run.
- **NFR-3** A completed run leaves **no orphaned or duplicate** files: a directive
  file no longer applicable (its driver was removed) is not left stale without the
  user being told.
- **NFR-4** Generation of the full set completes in **under 1 second** on a local
  filesystem.
- **NFR-5** All generated content is accurate for a **freshly promoted** project:
  the layout, commands, and write paths must match what an agent would actually
  find on disk for the chosen stack.

## Acceptance Criteria

- [ ] Running generation on a promoted project emits `CLAUDE.md`, `AGENTS.md`,
      `GEMINI.md`, and the Antigravity directive file, all with the same body
      content differing only by per-CLI framing/location. *(FR-1, FR-3)*
- [ ] Each emitted directive file contains repo layout, lineage convention,
      frontmatter requirements, commit conventions, roles, and a required-reading
      pointer to `lifecycle/architecture/`. *(FR-4)* — see [[architectural-artefacts]]
- [ ] For a non-Go+Vue selection, the repo layout and any build/test commands in
      the directive files reflect the chosen stack, not Go+Vue. *(FR-5, FR-2)* — see [[architecture-templates]]
- [ ] After generation, each developer agent in `lifecycle/config.yaml` has
      `allowed_write_paths` and build/test commands correct for the chosen stack.
      *(FR-6, FR-7)*
- [ ] Every generated analyst/planner/developer prompt contains the
      architecture-awareness clauses (flag architecture-breaking requirements;
      conform to standards; propose an ADR rather than deviate). *(FR-8)*
- [ ] The regenerated `lifecycle/config.yaml` passes `config.LoadProject()` and
      leaves roles, stages, users, and non-standard agents unchanged. *(FR-9)*
- [ ] Re-running generation with the same selection produces no net change
      (idempotent). *(FR-10, NFR-2)*
- [ ] Generation shows a diff and requires confirmation before overwriting a
      directive file that was edited after it was last generated; it never silently
      overwrites user edits. *(FR-11)*
- [ ] A project with no Gemini driver configured skips `GEMINI.md`, defaults to the
      standard set otherwise, and reports what was skipped. *(FR-12)*
- [ ] A naming/location choice (e.g. the Antigravity path) prompts the user and
      offers a "decide for me" default. *(FR-13)*
- [ ] Generation runs both as the wizard's opt-in scaffolding step and standalone
      via `kaos-control init --refresh-directives`, with identical output for the
      same selection. *(FR-14)* — see [[onboarding-architecture-selection]], [[cli-init-scaffold]]
- [ ] All generated files are re-indexed by the existing paths with no
      special-casing. *(FR-15)*
- [ ] Generation runs offline in under 1 second and leaves no orphaned/duplicate
      files. *(NFR-1, NFR-3, NFR-4)*

## Open Questions

- **OQ-1** What is the exact **filename and location of the Antigravity directive
  file**? If it is not yet stable, does v1 emit a best-effort default (prompting
  per FR-13) or defer Antigravity to a follow-up while keeping the wrapper
  extension point?

> Anti

- **OQ-2** How does generation know **which CLIs a project uses** — from the
  distinct `driver:` values across configured agents in `lifecycle/config.yaml`, an
  explicit `directives:` list, or a wizard prompt? This drives the default set in
  FR-12.

- **OQ-3** What mechanism distinguishes **generated content from user edits** for
  the diff-before-overwrite guarantee (FR-11): whole-file diff against the last
  generated version, or **managed-region markers** that let a regeneration update
  only the generated block while preserving surrounding user prose?

- **OQ-4** For the config edits (FR-6–FR-9): does generation only ever touch the
  **six standard agents**, or should it also update user-added developer agents
  that target the same roles? What happens to an agent whose role no longer maps to
  a source root in the chosen stack?

- **OQ-5** Where does the **stack → (repo layout, build/test commands, write paths)**
  profile data live so this generator can consume it — embedded per-stack profiles
  shipped with the catalog ([[architecture-templates]]), or fields on the promoted
  `tech-stack` artefact ([[architectural-artefacts]])? This is a hard dependency for
  FR-2/FR-5/FR-6/FR-7.

- **OQ-6** Should the shared directive body be **rendered fresh from the current
  spec/architecture on every run**, or snapshot at promotion time? A live render
  keeps directives current as the architecture evolves; a snapshot keeps them
  stable until an explicit refresh.
