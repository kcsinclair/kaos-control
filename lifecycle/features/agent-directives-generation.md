---
title: Agent directives generation (AGENTS.md-primary)
type: feature
status: approved
lineage: feature-agent-directives-generation
created: "2026-08-21T12:12:00+10:00"
summary: Generates one canonical AGENTS.md with CLAUDE.md/GEMINI.md as pointers, so every agent tool shares the same directives.
function: Agents
labels:
    - feature
    - agent
    - directives
related_to:
    - lifecycle/requirements/agent-directives-generation-2.md
    - lifecycle/architecture/decisions/adr-0005-agents-md-primary-directives.md
---

# Agent directives generation (AGENTS.md-primary)

One authoritative directives file for every coding agent, generated from the
project's stack and roles.

## What it does

- **AGENTS.md is canonical**, written with a managed region
  (`<!-- kaos-control:generated:start/end -->`) and an optional `--language`
  hint. `CLAUDE.md` and `GEMINI.md` are generated as thin `@AGENTS.md` pointers,
  so per-tool files can't drift from the source.
- **Stack-aware content**, with a generic fallback when no tech-stack is
  promoted.
- **Emitted at init** and by the wizard scaffold step; regeneration preserves
  human-authored additions outside the managed markers (a file that lost its
  markers is not silently overwritten — a diff is withheld pending force).
- **Migrate existing projects** via `POST /projects/{project}/migrate-directives`
  and refresh via `/directives/refresh`.

See [[adr-0005-agents-md-primary-directives]].
