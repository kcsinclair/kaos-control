---
title: AGENTS.md-primary directive files + a simple migration option
type: idea
status: blocked
lineage: agents-md-primary-directives
priority: medium
labels:
    - onboarding
    - agents
    - directives
    - migration
    - backend
assignees:
    - role: product-owner
      who: agent
---

# AGENTS.md-primary directive files + a simple migration option

## Context

`kaos-control init` today emits **only `CLAUDE.md`**
([internal/initcmd/seedfiles.go](../../internal/initcmd/seedfiles.go),
`CLAUDE.md.tmpl`). Projects that drive agents with **Codex** (`AGENTS.md`) or
**Gemini/Antigravity** (`GEMINI.md`) get no directive file at all — the gap #1
of [agent-directives-generation-2](../requirements/agent-directives-generation-2.md).

The emerging best practice is **`AGENTS.md` as the single canonical directive
file**, with the other CLIs' files reduced to references — rather than
maintaining N full copies that drift. This idea is the **file-structure +
migration half** of that requirement; it is shippable **independently of the
Architecture Wizard** and immediately closes the multi-CLI gap for existing
projects.

## The convention

- **`AGENTS.md`** — the canonical directive file at the project root; the single
  source of truth.
  - **Codex** reads `AGENTS.md` natively.
  - **Antigravity (Gemini)** auto-discovers and loads **both `AGENTS.md` and
    `GEMINI.md`** — so `AGENTS.md` alone is sufficient; no `GEMINI.md` and no
    `@AGENTS.md` directive are required for Antigravity (confirmed 2026-08-15).
- **`CLAUDE.md`** — a thin file that imports the canonical content via Claude
  Code's include syntax, `@AGENTS.md`, so the content is actually loaded into
  context, not merely referenced.
- **`GEMINI.md`** — **not required** when the Gemini target is Antigravity
  (auto-load). Only needed if a plain `gemini-cli` target is used that does not
  auto-load `AGENTS.md` (see OQ below) — then a pointer or generated copy.

Net directive set: **`AGENTS.md` (real) + `CLAUDE.md` (`@AGENTS.md` import)**,
with `GEMINI.md` optional.

## What this dissolves from agent-directives-generation-2

The AGENTS.md-primary model is a cleaner realization of the requirement's own
goal (one source of truth) and removes much of its mechanical weight:

- **FR-1 (single source / no drift)** — one real file; the rest are imports that
  cannot drift. The "render identical bodies into N files and keep in sync"
  machinery goes away.
- **FR-3 / NFR-3 (emit + no orphans)** — two of the files become one-liners (or
  vanish); far less to manage.
- **FR-11 / OQ-3 (diff-before-overwrite; generated vs user edit)** — shrinks to a
  single file (`AGENTS.md`); users edit the one canonical file directly.
- **OQ-1 (Antigravity file location)** — dissolved; Antigravity auto-loads
  `AGENTS.md`, so there is no separate Antigravity file or ambiguous path.
- **OQ-2 (which CLIs a project uses)** — simplifies to "always write `AGENTS.md`
  + the `CLAUDE.md` import; add `GEMINI.md` only for a non-auto-loading target."

## The migration option

A one-shot command (and matching `init` behaviour for new projects):

- **`kaos-control migrate-directives`** (or `init --refresh-directives`): if
  `AGENTS.md` does not exist, rename the existing `CLAUDE.md` → `AGENTS.md`;
  write `CLAUDE.md` as the single line `@AGENTS.md`; leave `GEMINI.md` out
  (Antigravity auto-loads) unless a plain-gemini target needs it.
- **New projects:** `init` emits `AGENTS.md` + the `CLAUDE.md` import instead of
  a lone `CLAUDE.md`.
- Idempotent; if `AGENTS.md` already exists and was user-edited, show a diff
  before touching it rather than overwriting.
- **Dogfood:** apply it to kaos-control's own repo (`CLAUDE.md` → `AGENTS.md` +
  `@AGENTS.md` import), so the tool eats its own migration.

## Out of scope (stays in agent-directives-generation-2)

The **stack-aware content** — the substantive half — is *not* addressed here:

- **FR-2 / FR-5** — the *body* of `AGENTS.md` reflecting the chosen stack (repo
  layout, build/test commands) rather than hard-coded Go+Vue.
- **FR-6 / FR-7 / FR-8** — the `lifecycle/config.yaml` agent prompt templates
  (stack-correct write paths, build/test commands, architecture-awareness
  clauses).
- **OQ-5** — the per-stack profile data those consume.

Those stay coupled to the Architecture Wizard + per-stack profiles. This idea
gives the file *structure and migration* now; the *content tuning* remains the
requirement's job.

## Open Questions

- **Plain `gemini-cli` (non-Antigravity):** does it auto-load `AGENTS.md`, or
  only `GEMINI.md`? Decides whether `GEMINI.md` can be dropped entirely or must
  be a pointer/copy for that target.

> Yes, gemini-cli will follow the directive.  So lets include GEMINI.md with the @AGENTS.md directive.  Handles gemini and antigravity.

- **`AGENTS.md` regeneration granularity:** whole-file diff, or managed-region
  markers so stack-aware regeneration (part B) updates only the generated block
  while preserving user prose? (Ties to OQ-3.)

> Lets go with "managed-region markers so stack-aware regeneration"

- **Existing hand-maintained `AGENTS.md`:** if a project already has one, does
  migration merge, defer, or diff-and-confirm?

> If there is already an AGENTS.md, the migration should stop and ask the user what they want to do.

## Related

- [agent-directives-generation-2](../requirements/agent-directives-generation-2.md)
  — the requirement this narrows; AGENTS.md-primary supersedes its "N full
  files" model, leaving the stack-aware content generation as the remaining work.
- [cli-init-scaffold](../ideas/cli-init-scaffold.md) — the v1 init scaffold this
  extends (currently emits only `CLAUDE.md`).
- [onboarding-architecture-selection](onboarding-architecture-selection.md) — the
  wizard that drives the stack-aware content half.
