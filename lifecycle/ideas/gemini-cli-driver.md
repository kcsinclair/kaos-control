---
title: Gemini CLI Agent Driver
type: idea
status: done
lineage: gemini-cli-driver
created: "2026-05-20T00:00:00+10:00"
priority: normal
labels:
    - agent
    - driver
    - feature
release: KC-Release3
---

# Gemini CLI Agent Driver

> Backfilled idea — this feature shipped before a formal lifecycle artifact was
> raised. Recorded here for traceability and release accounting.

## Idea

Add Google Gemini as an agent option, in two complementary forms: a first-class
`gemini` driver that talks to Gemini API models directly, and a `gemini-cli`
driver that drives the `agy` Gemini CLI as an autonomous agent.

## What shipped

- `gemini` driver (Gemini API models) —
  [internal/agent/gemini.go](../../internal/agent/gemini.go).
- `gemini-cli` driver (runs the `agy` CLI) —
  [internal/agent/gemini_cli.go](../../internal/agent/gemini_cli.go): passes
  `--add-dir <project-root>` so `agy` sees the workspace and `--print-timeout`
  so it isn't cut off mid-reply, and runs `cmd.Wait()` asynchronously to
  unwedge the supervisor when `agy` detaches a daemon child.
- Both exposed as radio options in
  [web/src/components/agent/AgentConfigForm.vue](../../web/src/components/agent/AgentConfigForm.vue);
  a `GEMINI_API_KEY` hint surfaces for the `gemini` driver.

## References

- PROJECT_PLAN rolling log — 2026-05-22 (×2: `--add-dir`, `--print-timeout`),
  2026-05-22 (detached-child deadlock fix), 2026-05-30 (UI radios), 2026-05-27
  (driver-options defect). See [plans/PROJECT_PLAN.md](../../plans/PROJECT_PLAN.md).
- Release notes: [RELEASE_NOTES-0.1.3.md](../../RELEASE_NOTES-0.1.3.md) — "New
  agent drivers".
- Related drivers: [[codex-cli-driver]], [[claude-hooks-driver]].
