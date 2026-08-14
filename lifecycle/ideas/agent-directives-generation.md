---
title: Agent Directives Generation — CLAUDE.md, AGENTS.md and Stack-Tuned Prompts at Init
type: idea
status: draft
lineage: agent-directives-generation
created: "2026-08-14T12:30:00+10:00"
priority: normal
labels:
    - architecture
    - onboarding
    - backend
    - feature
release: KC-Release5
parent: lifecycle/ideas/architecture-templates.md
---

# Agent Directives Generation — CLAUDE.md, AGENTS.md and Stack-Tuned Prompts at Init

kaos-control initialisation must create **all the prompts correctly for all
the agents**, along with `CLAUDE.md` and the equivalent directive files for
the other agent CLIs — Codex (`AGENTS.md`), Gemini (`GEMINI.md`), Antigravity,
and future drivers. Today `kaos-control init` emits a generic `CLAUDE.md`
([[cli-init-scaffold]]); this idea extends that to a full, stack-aware
directive set generated from the Architecture Wizard's selection.

## What gets generated

When the wizard's scaffolding step runs ([[architecture-templates]] §4,
[[onboarding-architecture-selection]]):

- **Per-CLI directive files** at the project root: `CLAUDE.md`, `AGENTS.md`,
  `GEMINI.md`, and Antigravity's directive location — all rendered from **one
  shared source of truth** (single template + per-CLI wrapper) so they never
  drift from each other. Content covers repo layout for the chosen stack,
  artifact/lineage conventions, frontmatter requirements, commit conventions,
  roles, and — new — a pointer to `lifecycle/architecture/` (the promoted
  architecture and stack, `architecture-summary.md`, `decisions/` ADRs, and
  `standards/`) as required reading before designing or building.
- **Agent prompt templates in `lifecycle/config.yaml`**, correct for the
  chosen stack: right `allowed_write_paths` (e.g. Go+Vue → backend writes
  `internal/`+`cmd/`, frontend writes `web/src/`), right build/test commands
  in each developer prompt (`go build`/`go vet` vs `pytest`/`ruff` vs
  `pnpm build`…), and the architecture-awareness clauses: analysts flag
  **architecture-breaking requirements** against the chosen
  architecture/stack; planners and developers conform to the recorded
  architecture and standards and **propose an ADR** rather than silently
  deviating.

## Behaviour

- Generation is idempotent and re-runnable: re-running the wizard (or a
  future `kaos-control init --refresh-directives`) regenerates from the
  current selection, showing a diff before overwriting user-edited files.
- Directive files a project doesn't need (no Gemini driver configured) can be
  skipped; the default is to emit the standard set.
- Naming/location choices follow the wizard convention: ask the user, offer
  **"decide for me"**.

This closes the loop begun by [[cli-init-scaffold]] and
[[kaos-control-init-bootstrap]]: init no longer just scaffolds directories —
it wires the whole agent workforce to the chosen architecture.
