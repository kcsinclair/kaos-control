---
title: AGENTS.md-primary agent directives (CLAUDE.md / GEMINI.md as pointers)
type: adr
status: approved
lineage: adr-agents-md-primary-directives
created: "2026-08-21T11:47:00+10:00"
labels:
    - adr
    - architecture
    - directives
    - agent
---

# ADR-0005: AGENTS.md-primary agent directives

## Context

kaos-control generates per-project agent directives so that coding agents
(Claude Code, Gemini, others) share one authoritative set of instructions.
Historically each tool read its own file (`CLAUDE.md`, `GEMINI.md`), which
drifts: the same guidance had to be duplicated and kept in sync per tool.

The project is also deliberately tool-agnostic and open — no single agent
vendor is privileged.

## Decision

Make **`AGENTS.md` the canonical directives file**, with a managed region
delimited by `<!-- kaos-control:generated:start -->` / `:end` markers.
`CLAUDE.md` and `GEMINI.md` are generated as thin **`@AGENTS.md` pointers** that
delegate to the canonical file rather than holding their own copy.

`internal/directives.Generate` owns this: it writes AGENTS.md (with the managed
region and optional `--language` hint), plus the per-tool pointer files. When no
tech stack is promoted it falls back to a generic AGENTS.md (`GenericAgents`).
Project `init` and the wizard's scaffold step both call it.

## Consequences

- One source of truth for agent guidance; per-tool files can never drift from
  it because they only point at it.
- Adding support for a new agent tool is a new pointer file, not a new copy of
  the directives.
- The managed-region markers let regeneration update kaos-control's content
  while preserving human-authored additions outside the markers; a file that
  lost its markers is not silently overwritten (a diff is withheld pending
  force).
- Signals the project's open, multi-agent stance: no vendor's file is the
  primary.
