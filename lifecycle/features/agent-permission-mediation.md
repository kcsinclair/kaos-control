---
title: Agent permission mediation
type: feature
status: approved
lineage: feature-agent-permission-mediation
created: "2026-08-21T12:13:00+10:00"
summary: A mediated driver enforces per-agent write scope and command allow/deny at the tool-call boundary, records denials, and can pause the queue.
function: Agents
labels:
    - feature
    - agent
    - security
related_to:
    - lifecycle/architecture/decisions/adr-0006-mediated-agent-driver-permission-model.md
---

# Agent permission mediation

Agent authority is bounded by configuration and enforced, not left to the
agent's good behaviour.

## What it does

- **Write scoping.** File-mutating tool calls (Write/Edit/MultiEdit/NotebookWrite)
  are checked against each agent's `allowed_write_paths` through the
  traversal-safe resolver; out-of-scope writes are denied.
- **Command policy.** Shell commands are checked against per-agent
  `bash_allowlist` / `bash_denylist` with a configurable `on_denial` behaviour.
- **Denials are observable.** Recorded on the run (`denied_tool_calls`) and
  surfaced in the run detail.
- **Queue pause.** A denial can pause the agent queue so a human reviews rather
  than the run proceeding or failing opaquely.
- **Multiple drivers.** `claude-code-cli`, `claude-mediated`, `claude-env`,
  `codex-cli`, `gemini`, `gemini-cli`, and `ollama` — the mediated path applies
  the guarantees above.

See [[adr-0006-mediated-agent-driver-permission-model]] and
[[filesystem-sandboxing]].
