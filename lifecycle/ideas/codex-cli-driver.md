---
title: Codex CLI Agent Driver
type: idea
status: done
lineage: codex-cli-driver
created: "2026-05-27T00:00:00+10:00"
priority: normal
labels:
    - agent
    - driver
    - feature
release: KC-Release3
---

# Codex CLI Agent Driver

> Backfilled idea — this feature shipped before a formal lifecycle artifact was
> raised. Recorded here for traceability and release accounting.

## Idea

Add an agent driver that runs OpenAI's Codex CLI (`codex exec`) as an autonomous
agent, so a project can run agents on Codex models alongside the existing Claude
and Ollama drivers.

## What shipped

- New `codex-cli` driver in
  [internal/agent/codex_cli.go](../../internal/agent/codex_cli.go): runs
  `codex exec --json --dangerously-bypass-approvals-and-sandbox`, passes
  `--cd <project-root>` for workspace correctness, and optionally maps
  `timeout_minutes` to `--timeout <seconds>` only when the installed Codex
  binary advertises the flag. JSONL/raw stdout streams into the existing
  progress/log plumbing.
- Exposed as a `codex-cli` radio option in
  [web/src/components/agent/AgentConfigForm.vue](../../web/src/components/agent/AgentConfigForm.vue);
  the model field is optional (Codex uses its own default), and run summary
  cards hide Claude token metrics for Codex runs.
- Shared CLI process handling refactored (`claudeProcess` → `cliProcess`) with
  an async `cmd.Wait()` so detached child processes can't hold stdout/stderr
  pipes open.

## References

- PROJECT_PLAN rolling log — 2026-05-27 (backend driver), 2026-05-27 (UI
  exposure), 2026-05-30 (merge of PR #9), 2026-05-30 (codex probe-timeout
  widening). See [plans/PROJECT_PLAN.md](../../plans/PROJECT_PLAN.md).
- Release notes: [RELEASE_NOTES-0.1.3.md](../../RELEASE_NOTES-0.1.3.md) — "New
  agent drivers".
- Related drivers: [[gemini-cli-driver]], [[claude-hooks-driver]].
