---
created: "2026-08-15T17:42:32+10:00"
title: AGENTS.md-Primary Directive Files + Migration Command
type: requirement
status: blocked
lineage: agents-md-primary-directives
priority: medium
parent: lifecycle/ideas/agents-md-primary-directives.md
labels:
    - onboarding
    - agents
    - directives
    - migration
    - backend
release: KC-Release5
assignees:
    - role: product-owner
      who: agent
---

# AGENTS.md-Primary Directive Files + Migration Command

## Problem

`kaos-control init` today emits a single **`CLAUDE.md`** at the project root
([internal/initcmd/seedfiles.go](../../internal/initcmd/seedfiles.go),
`CLAUDE.md.tmpl`) and no directive file for any other CLI. Projects that drive
agents with **Codex** (`AGENTS.md`) or **Gemini / Antigravity** (`GEMINI.md`)
get no directives at all — those agents run without repo layout, artifact and
lineage conventions, frontmatter rules, commit conventions, roles, or the
required-reading pointer to `lifecycle/architecture/`. This is gap #1 of
[[agent-directives-generation]].

The emerging best practice is **`AGENTS.md` as the single canonical directive
file**, with the other CLIs' files reduced to imports rather than N full copies
that drift. This requirement is the **file-structure + migration half** of that
work; it is shippable **independently of the Architecture Wizard** and
immediately closes the multi-CLI gap for both new and existing projects. The
**stack-aware content** half (directive body and `config.yaml` agent prompts
tuned to a chosen stack) stays with [[agent-directives-generation]].

## Goals / Non-goals

### Goals

- **New projects**: `kaos-control init` emits the canonical directive set —
  `AGENTS.md` (real content) plus `CLAUDE.md` and `GEMINI.md` as `@AGENTS.md`
  imports — instead of a lone `CLAUDE.md`.
- **Existing projects**: a one-shot, idempotent **`kaos-control
  migrate-directives`** command (and a matching `init --refresh-directives`)
  converts the legacy single-`CLAUDE.md` layout to the canonical set.
- **Single source of truth**: only `AGENTS.md` holds real directive content;
  `CLAUDE.md` and `GEMINI.md` are thin imports that cannot drift.
- **Managed-region markers** delimit the generated block inside `AGENTS.md`, so
  a later stack-aware regeneration ([[agent-directives-generation]]) can update
  only the generated region while preserving user prose.
- **Safe by default**: never silently clobber user-edited content; stop and ask,
  or show a diff, before overwriting.
- **Dogfood**: apply the migration to kaos-control's own repo.

### Non-goals

- The **stack-aware directive body** (repo layout, build/test commands derived
  from the chosen architecture + stack) — owned by [[agent-directives-generation]]
  (FR-2/FR-5).
- The **`lifecycle/config.yaml` agent prompt templates** (write paths, build/test
  commands, architecture-awareness clauses) — owned by
  [[agent-directives-generation]] (FR-6/FR-7/FR-8).
- The **per-stack profile data** those consume — owned by [[architecture-templates]].
- The **Architecture Wizard** UX and its opt-in hand-off — owned by
  [[onboarding-architecture-selection]].
- The base scaffold and seed `config.yaml` / `.claude/settings.json` — owned by
  [[cli-init-scaffold]]; this requirement extends only the directive-file portion.

## Detailed Requirements

### Functional — the canonical directive set

- **FR-1** The canonical directive set at the project root is:

  | File | Content |
  |---|---|
  | `AGENTS.md` | The single **canonical** directive file; the only file holding real content. Codex reads it natively; Antigravity auto-discovers and loads it. |
  | `CLAUDE.md` | A one-line **`@AGENTS.md`** import (Claude Code loads the included content into context). |
  | `GEMINI.md` | A one-line **`@AGENTS.md`** import, covering both plain `gemini-cli` (which follows the directive) and Antigravity. |

- **FR-2** `AGENTS.md` carries the same standing content the current
  `CLAUDE.md.tmpl` provides (repo layout, artifact/lineage filename convention,
  frontmatter requirements, commit conventions, roles, required-reading pointer
  to `lifecycle/architecture/`). This requirement moves that content into
  `AGENTS.md` **as-is**; tuning its body to a chosen stack is out of scope
  (see [[agent-directives-generation]]).
- **FR-3** The generated content inside `AGENTS.md` is wrapped in
  **managed-region markers** (a clearly-labelled BEGIN/END comment pair) so that
  a future stack-aware regeneration updates only the region between the markers
  and leaves any user prose outside them untouched. Text outside the markers is
  never modified by generation or migration.
- **FR-4** `CLAUDE.md` and `GEMINI.md` each contain exactly the import directive
  `@AGENTS.md` (plus, at most, a short generated header comment); they hold no
  duplicated body content.

### Functional — `init` for new projects

- **FR-5** `kaos-control init` on a project with **no** existing root directive
  file emits `AGENTS.md`, `CLAUDE.md`, and `GEMINI.md` per FR-1, replacing the
  previous behaviour of emitting only `CLAUDE.md`.
- **FR-6** The existing `init` skip/force semantics still apply: if a target
  directive file already exists and its force flag is not set, that file is
  skipped and reported (matching current `writeSeedFiles` behaviour), rather than
  overwritten.

### Functional — the migration command

- **FR-7** `kaos-control migrate-directives` (and the equivalent
  `init --refresh-directives`) converts an existing project from the legacy
  layout to the canonical set:
  1. If **no `AGENTS.md`** exists and a root `CLAUDE.md` exists: **rename**
     `CLAUDE.md` → `AGENTS.md` (preserving its full content, wrapping the
     managed portion in the FR-3 markers), then write `CLAUDE.md` as the
     `@AGENTS.md` import and write `GEMINI.md` as the `@AGENTS.md` import.
  2. If neither `AGENTS.md` nor `CLAUDE.md` exists: emit the full canonical set
     (equivalent to FR-5).
- **FR-8** **Pre-existing `AGENTS.md` guard**: if the project already has an
  `AGENTS.md`, migration **stops and asks the user** how to proceed (keep as-is /
  overwrite with the canonical content / abort). It never merges or overwrites a
  pre-existing `AGENTS.md` without explicit user choice.
- **FR-9** **Idempotency**: running `migrate-directives` on a project already in
  the canonical layout makes **no net change** — `AGENTS.md` is left untouched,
  and `CLAUDE.md` / `GEMINI.md` are confirmed to be the `@AGENTS.md` imports
  (rewritten only if they diverge from the import form). A second consecutive run
  produces no diff.
- **FR-10** **Diff-before-overwrite**: before rewriting any directive file that
  differs from what generation would produce, the command shows a **diff** and
  requires confirmation (interactive) or an explicit `--force`-style flag
  (non-interactive). Files that do not yet exist are created without a diff
  prompt. Within `AGENTS.md`, the diff/overwrite is scoped to the managed region
  (FR-3).
- **FR-11** The migration is **reported**: the command prints, per file, whether
  it was created, rewritten, renamed, skipped, or left unchanged, and (on the
  pre-existing-`AGENTS.md` path) what choice was taken.

### Functional — integration & dogfood

- **FR-12** Directive files live at the **project root**, outside `lifecycle/`,
  and are therefore **not** indexed — consistent with today's `CLAUDE.md`. This
  requirement introduces no new indexing path or special-casing.
- **FR-13** File writes/renames go through kaos-control's existing
  path-safe filesystem handling; migration never writes outside the project
  root.
- **FR-14** kaos-control's own repository is migrated as part of shipping this:
  `CLAUDE.md` → `AGENTS.md` (managed region marked) + `CLAUDE.md` and `GEMINI.md`
  `@AGENTS.md` imports.

### Non-functional

- **NFR-1** Generation and migration work **entirely offline**; all templates
  are embedded in the binary — no network access at runtime.
- **NFR-2** Generation is **deterministic**: the same input layout yields
  byte-identical `AGENTS.md` managed region and identical import files on every
  run.
- **NFR-3** The command **never destroys content**: the rename preserves the full
  original `CLAUDE.md` body inside `AGENTS.md`; no user prose outside the managed
  region is lost on any path.
- **NFR-4** A full new-project emit or single-project migration completes in
  **under 1 second** on a local filesystem.
- **NFR-5** Behaviour is covered by tests in the existing `internal/initcmd`
  (and, where a CLI surface is exercised, `tests/`) suites: fresh init, legacy
  migration, idempotent re-run, and the pre-existing-`AGENTS.md` guard.

### Architecture-Breaking Requirements

This project has **not yet run the Architecture Wizard**, so there is no
`lifecycle/architecture/architecture-summary.md`, `decisions/`, or `standards/`
to check against; the recorded architecture is the one documented in
[CLAUDE.md](../../CLAUDE.md) and the catalog entries
[modular-monolith](../architecture/architectures/modular-monolith.md) +
[go-vue](../architecture/tech-stacks/go-vue.md): a **single Go binary** with an
embedded Vue SPA and embedded templates, operating **offline**.

No requirement here is architecture-breaking. Assessed dimensions:

- **Offline operation** — *satisfied*. All directive templates are embedded in
  the binary (`internal/initcmd/templates`); generation and migration require no
  network (NFR-1), consistent with the single-binary/offline model.
- **Filesystem / path safety** — *satisfied within the architecture*. Root
  directive files (`AGENTS.md`, `CLAUDE.md`, `GEMINI.md`) are written where the
  current `CLAUDE.md` already is; writes stay within the project root (FR-13),
  reusing the existing path-safe handling.
- **Indexing / realtime** — *no impact*. Directive files sit outside
  `lifecycle/` and are not indexed (FR-12); no new SQLite, watcher, or WS path is
  introduced.
- **Collaboration / scale / security / cost** — *no impact*. This is a
  local, single-user CLI file operation with no server, network, multi-tenant, or
  compute-scaling surface.

There is **no conflict** to flag against the (not-yet-existing) architecture
summary. If the wizard later promotes a non-Go+Vue stack, the **body** of
`AGENTS.md` becomes stack-dependent — but that is owned by
[[agent-directives-generation]], and the managed-region markers (FR-3) are the
mechanism that keeps this file-structure work compatible with that later change.

## Acceptance Criteria

- [ ] `kaos-control init` on a project with no root directive file emits
      `AGENTS.md` (real content), `CLAUDE.md` (`@AGENTS.md`), and `GEMINI.md`
      (`@AGENTS.md`). *(FR-1, FR-5)* — see [[cli-init-scaffold]]
- [ ] `AGENTS.md` contains repo layout, lineage convention, frontmatter
      requirements, commit conventions, roles, and the required-reading pointer
      to `lifecycle/architecture/`, wrapped in managed-region BEGIN/END markers.
      *(FR-2, FR-3)*
- [ ] `CLAUDE.md` and `GEMINI.md` each contain the `@AGENTS.md` import and no
      duplicated body content. *(FR-4)*
- [ ] `kaos-control migrate-directives` on a legacy single-`CLAUDE.md` project
      renames `CLAUDE.md` → `AGENTS.md` (content preserved, managed region
      marked), and writes `CLAUDE.md` + `GEMINI.md` as `@AGENTS.md` imports.
      *(FR-7)*
- [ ] Running `migrate-directives` where an `AGENTS.md` already exists **stops
      and asks the user** and never overwrites it without an explicit choice.
      *(FR-8)*
- [ ] Re-running `migrate-directives` on an already-canonical project produces no
      net change; a second consecutive run yields no diff. *(FR-9, NFR-2)*
- [ ] Before rewriting a directive file that differs from generated output, the
      command shows a diff and requires confirmation (or an explicit force flag);
      non-existent files are created without a prompt. *(FR-10)*
- [ ] The command reports, per file, whether it was created, rewritten, renamed,
      skipped, or unchanged. *(FR-11)*
- [ ] No directive file is indexed; no new indexing/watcher/WS path is added.
      *(FR-12)*
- [ ] No content is written outside the project root, and no user prose outside
      the managed region is ever lost on any path. *(FR-13, NFR-3)*
- [ ] Generation/migration runs offline in under 1 second. *(NFR-1, NFR-4)*
- [ ] kaos-control's own repo is migrated: `CLAUDE.md` → `AGENTS.md` +
      `@AGENTS.md` imports. *(FR-14)*
- [ ] Tests cover fresh init, legacy migration, idempotent re-run, and the
      pre-existing-`AGENTS.md` guard. *(NFR-5)*
- [ ] The stack-aware body and `config.yaml` prompt tuning remain out of scope
      and stay tracked in [[agent-directives-generation]].

## Open Questions

- **OQ-1** Managed-region marker syntax: what exact comment form delimits the
  generated block in `AGENTS.md`? It must be an HTML/markdown comment
  (`<!-- ... -->`) so it renders invisibly, and stable enough for a future
  regeneration to locate reliably. (Proposal: a labelled
  `<!-- kaos-control:directives BEGIN -->` / `<!-- kaos-control:directives END -->`
  pair.)
- **OQ-2** Non-interactive migration: for CI/headless runs where FR-8's "stop and
  ask" cannot prompt, what is the default — abort with a non-zero exit, or require
  an explicit `--force` / `--keep-existing` flag to choose up front?
- **OQ-3** Does `GEMINI.md` need any Gemini/Antigravity-specific framing around
  the `@AGENTS.md` import, or is the bare import directive sufficient for both
  `gemini-cli` and Antigravity? (Idea resolved that both follow the directive;
  confirm no wrapper is required.)
- **OQ-4** Should `migrate-directives` be a standalone subcommand, `init
  --refresh-directives`, or both aliased to one implementation? (FR-7 assumes one
  shared implementation reachable both ways.)
