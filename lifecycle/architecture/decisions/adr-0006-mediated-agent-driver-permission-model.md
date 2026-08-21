---
title: Mediated agent driver with tool-call permission enforcement
type: adr
status: approved
lineage: adr-mediated-agent-driver-permission-model
created: "2026-08-21T11:48:00+10:00"
labels:
    - adr
    - architecture
    - agent
    - security
related_to:
    - adr-0001-no-header-based-client-ip-trust
    - filesystem-sandboxing
---

# ADR-0006: Mediated agent driver with tool-call permission enforcement

## Context

kaos-control orchestrates agent runs that write artifacts and code
(`internal/agent/`). Agents execute tools — file writes, shell commands — that
can touch anything the process can reach. Two risks follow: an agent writing
outside its remit (e.g. a frontend agent editing `internal/`), and an agent
running commands it should not. Each configured agent already declares scoped
`allowed_write_paths` and, for shell, allow/deny lists; those declarations need
*enforcement*, not just documentation.

## Decision

Run agents through a **mediated driver** that intercepts tool calls before they
execute and enforces per-agent policy:

- **Write scoping** — file-mutating tools (Write/Edit/MultiEdit/NotebookWrite)
  are checked against the agent's `allowed_write_paths`; out-of-scope writes are
  denied.
- **Command policy** — shell commands are checked against the agent's
  `bash_allowlist` / `bash_denylist` with a configurable `on_denial` behaviour.
- **Denials are recorded** on the run (`denied_tool_calls`) and can **pause the
  queue** (`PauseQueue`) so a human reviews rather than the run silently
  proceeding or failing opaquely.
- Path checks resolve through the traversal-safe filesystem resolver
  (`internal/sandbox/`, see [[filesystem-sandboxing]]) so `..` and symlink
  escapes cannot defeat the scope.

## Consequences

- Agent authority is bounded by configuration and enforced at the tool-call
  boundary, not left to the agent's good behaviour.
- Denied calls are observable (surfaced on the run detail) and actionable
  (queue pause), turning a would-be silent overreach into a review gate.
- Adds a mediation layer between the driver and the underlying agent CLI; each
  supported driver must route tool calls through it to get the guarantees.
- Complements the local-auth/trust decisions ([[adr-0001-no-header-based-client-ip-trust]]):
  the system is defensive about both who calls it and what its agents may do.
