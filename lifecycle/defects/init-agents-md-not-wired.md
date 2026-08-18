---
title: Project Initialisation Does Not Set Up AGENTS.md with Claude and Gemini Directives
type: defect
status: approved
lineage: init-agents-md-not-wired
created: "2026-08-19T09:25:51+10:00"
priority: normal
labels:
    - defect
    - onboarding
    - directives
    - agent
    - config
---

# Project Initialisation Does Not Set Up AGENTS.md with Claude and Gemini Directives

## Reproduction Steps

1. Run project initialisation for a new or existing project.
2. Observe the files created or updated by the initialisation process.
3. Check whether `AGENTS.md` is created and populated with Gemini and Claude directives.
4. Check whether `CLAUDE.md` is updated to reference `AGENTS.md` (i.e. contains `@AGENTS.md`) rather than holding the full content itself.

## Expected Behaviour

Project initialisation should:
- Create `AGENTS.md` containing the agent directives for both Claude and Gemini.
- Set `CLAUDE.md` content to delegate to `AGENTS.md` via `@AGENTS.md` (or equivalent include mechanism).
- Ensure the new `AGENTS.md`-based structure is fully wired up so both Claude Code and Gemini agents pick up the correct directives from the shared file.

## Actual Behaviour

Project initialisation does not create or populate `AGENTS.md`, and does not update `CLAUDE.md` to use the new `@AGENTS.md` delegation pattern. The current `CLAUDE.md` continues to hold the full content directly, leaving the Gemini directives integration unimplemented and the new dual-agent structure not established during onboarding.
