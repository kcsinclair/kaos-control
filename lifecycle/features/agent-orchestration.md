---
title: Agent orchestration
type: feature
status: approved
lineage: feature-agent-orchestration
created: "2026-08-21T15:04:00+10:00"
summary: Per-role agent configuration, run history, live progress, and lineage locking that runs coding agents against the lifecycle.
function: Agents
labels:
    - feature
    - agents
related_to:
    - lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md
---

# Agent orchestration

Agents are configured, launched, and observed as first-class lifecycle
citizens.

## What it does

- **Configured per role.** Each agent is bound to one or more roles with a
  focused prompt template, a sandboxed `allowed_write_paths` allowlist,
  optional model override, optional timeout.
- **Drivers.** `claude-code-cli` (default) shells out to
  `claude --dangerously-skip-permissions -p`, parsing stream-json events for
  live progress; other drivers cover mediated Claude, Codex, Gemini, and
  Ollama.
- **Active-status lifecycle.** When an agent starts, the target's status
  moves to a configured `active_status` (e.g. `in-development`) and back to
  `done` on success — bundled into the agent's own commit.
- **Run history.** Every run persists to SQLite with status, exit code,
  stderr tail, artifacts produced, target, role, started/finished
  timestamps. Survives schema rebuilds (cache-resilient).
- **Live progress.** Stdout / stderr streams to the **Agents** page via
  WebSocket; expandable per-run detail with the last ~4 KB of stderr.
- **Lineage locks.** Every run acquires an exclusive lock on the target's
  lineage; concurrent runs on the same lineage are rejected. Stale locks
  reaped after 5 min of no heartbeat.
- **Crash recovery.** Runs left in `running` after a server restart are
  automatically marked `failed`.
- **Kill button.** SIGTERM the running agent process from the UI.
- **Agent launcher panels.** Per-agent cards on the Agents page show the
  current ready-work for that role, with one-click run.

Reachable at **Agents**; API under `/agents/*`. See also
[[agent-permission-mediation]] for write/command enforcement.
